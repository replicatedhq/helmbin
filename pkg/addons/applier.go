// Package addons manages the default addons installations in the cluster. Addons are
// mostly Helm Charts, but can also be other resources as the project evolves. All of
// the AddOns must implement the AddOn interface.
package addons

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/k0sproject/k0s/pkg/apis/k0s/v1beta1"
	k0sconfig "github.com/k0sproject/k0s/pkg/apis/k0s/v1beta1"
	embeddedclusterv1beta1 "github.com/replicatedhq/embedded-cluster-kinds/apis/v1beta1"
	"github.com/replicatedhq/embedded-cluster-kinds/types"
	"github.com/replicatedhq/troubleshoot/pkg/apis/troubleshoot/v1beta2"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/replicatedhq/embedded-cluster/pkg/addons/adminconsole"
	"github.com/replicatedhq/embedded-cluster/pkg/addons/embeddedclusteroperator"
	"github.com/replicatedhq/embedded-cluster/pkg/addons/openebs"
	"github.com/replicatedhq/embedded-cluster/pkg/addons/registry"
	"github.com/replicatedhq/embedded-cluster/pkg/addons/velero"
	"github.com/replicatedhq/embedded-cluster/pkg/defaults"
	"github.com/replicatedhq/embedded-cluster/pkg/helpers"
	"github.com/replicatedhq/embedded-cluster/pkg/kubeutils"
	"github.com/replicatedhq/embedded-cluster/pkg/spinner"
)

// AddOn is the interface that all addons must implement.
type AddOn interface {
	Version() (map[string]string, error)
	Name() string
	HostPreflights() (*v1beta2.HostPreflightSpec, error)
	GenerateHelmConfig(onlyDefaults bool) ([]v1beta1.Chart, []v1beta1.Repository, error)
	Outro(context.Context, client.Client) error
	GetProtectedFields() map[string][]string
	GetAdditionalImages() []string
}

// Applier is an entity that applies (installs and updates) addons in the cluster.
type Applier struct {
	prompt          bool
	verbose         bool
	config          v1beta1.ClusterConfig
	licenseFile     string
	onlyDefaults    bool
	endUserConfig   *embeddedclusterv1beta1.Config
	airgapBundle    string
	releaseMetadata *types.ReleaseMetadata
}

// Outro runs the outro in all enabled add-ons.
func (a *Applier) Outro(ctx context.Context) error {
	kcli, err := kubeutils.KubeClient()
	if err != nil {
		return fmt.Errorf("unable to create kube client: %w", err)
	}
	addons, err := a.load()
	if err != nil {
		return fmt.Errorf("unable to load addons: %w", err)
	}
	for _, addon := range addons {
		if err := addon.Outro(ctx, kcli); err != nil {
			return err
		}
	}

	err = spinForInstallation(ctx, kcli)
	if err != nil {
		return err
	}

	// this is a link to the admin console, and that is only installed if there is a license provided
	if a.licenseFile != "" {
		err = printKotsadmLinkMessage(a.licenseFile)
		if err != nil {
			return fmt.Errorf("unable to print success message: %w", err)
		}
	}
	return nil
}

// GenerateHelmConfigs generates the helm config for all the embedded charts.
func (a *Applier) GenerateHelmConfigs(additionalCharts []v1beta1.Chart, additionalRepositories []v1beta1.Repository) ([]v1beta1.Chart, []v1beta1.Repository, error) {
	charts := []v1beta1.Chart{}
	repositories := []v1beta1.Repository{}
	addons, err := a.load()
	if err != nil {
		return nil, nil, fmt.Errorf("unable to load addons: %w", err)
	}

	// charts required by embedded-cluster
	for _, addon := range addons {
		addonChartConfig, addonRepositoryConfig, err := addon.GenerateHelmConfig(a.onlyDefaults)
		if err != nil {
			return nil, nil, fmt.Errorf("unable to generate helm config for %s: %w", addon, err)
		}
		charts = append(charts, addonChartConfig...)
		repositories = append(repositories, addonRepositoryConfig...)
	}

	// charts required by the application
	charts = append(charts, additionalCharts...)
	if a.endUserConfig != nil {
		for i, chart := range charts {
			values, err := a.endUserConfig.Spec.ApplyEndUserAddOnOverrides(chart.Name, chart.Values)
			if err != nil {
				return nil, nil, fmt.Errorf("unable to apply end user overrides for %s: %w", chart.Name, err)
			}
			charts[i].Values = values
		}
	}
	repositories = append(repositories, additionalRepositories...)

	return charts, repositories, nil
}

func (a *Applier) GetAirgapCharts() ([]v1beta1.Chart, []v1beta1.Repository, error) {
	reg, err := registry.New(true)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to create registry addon: %w", err)
	}
	regChart, regRepo, err := reg.GenerateHelmConfig(true)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to generate helm config for registry: %w", err)
	}

	return regChart, regRepo, nil
}

func (a *Applier) GetBuiltinCharts() (map[string]k0sconfig.HelmExtensions, error) {
	builtinCharts := map[string]k0sconfig.HelmExtensions{}

	vel, err := velero.New(true)
	if err != nil {
		return nil, fmt.Errorf("unable to create velero addon: %w", err)
	}
	velChart, velRepo, err := vel.GenerateHelmConfig(true)
	if err != nil {
		return nil, fmt.Errorf("unable to generate helm config for velero: %w", err)
	}

	builtinCharts["velero"] = k0sconfig.HelmExtensions{
		Repositories: velRepo,
		Charts:       velChart,
	}

	return builtinCharts, nil
}

func (a *Applier) GetAdditionalImages() ([]string, error) {
	additionalImages := []string{}
	addons, err := a.load()
	if err != nil {
		return nil, fmt.Errorf("unable to load addons: %w", err)
	}
	for _, addon := range addons {
		additionalImages = append(additionalImages, addon.GetAdditionalImages()...)
	}

	return additionalImages, nil
}

// ProtectedFields returns the protected fields for all the embedded charts.
func (a *Applier) ProtectedFields() (map[string][]string, error) {
	protectedFields := map[string][]string{}
	addons, err := a.load()
	if err != nil {
		return protectedFields, fmt.Errorf("unable to load addons: %w", err)
	}
	for _, addon := range addons {
		for k, v := range addon.GetProtectedFields() {
			protectedFields[k] = v
		}
	}
	return protectedFields, nil
}

// HostPreflights reads all embedded host preflights from all add-ons and returns them
// merged in a single HostPreflightSpec.
func (a *Applier) HostPreflights() (*v1beta2.HostPreflightSpec, error) {
	addons, err := a.load()
	if err != nil {
		return nil, fmt.Errorf("unable to load addons: %w", err)
	}
	allpf := &v1beta2.HostPreflightSpec{}
	for _, addon := range addons {
		hpf, err := addon.HostPreflights()
		if err != nil {
			return nil, fmt.Errorf("unable to get preflights for %s: %w", addon.Name(), err)
		} else if hpf == nil {
			continue
		}
		allpf.Collectors = append(allpf.Collectors, hpf.Collectors...)
		allpf.Analyzers = append(allpf.Analyzers, hpf.Analyzers...)
	}
	return allpf, nil
}

// load instantiates and returns all addon appliers.
func (a *Applier) load() ([]AddOn, error) {
	addons := []AddOn{}
	obs, err := openebs.New()
	if err != nil {
		return nil, fmt.Errorf("unable to create openebs addon: %w", err)
	}
	addons = append(addons, obs)

	reg, err := registry.New(a.airgapBundle != "")
	if err != nil {
		return nil, fmt.Errorf("unable to create registry addon: %w", err)
	}
	addons = append(addons, reg)

	embedoperator, err := embeddedclusteroperator.New(a.endUserConfig, a.licenseFile, a.airgapBundle != "", a.releaseMetadata)
	if err != nil {
		return nil, fmt.Errorf("unable to create embedded cluster operator addon: %w", err)
	}
	addons = append(addons, embedoperator)

	snapshotsEnabled, err := helpers.SnapshotsEnabled(a.licenseFile)
	if err != nil {
		return nil, fmt.Errorf("unable to check if snapshots are enabled: %w", err)
	}
	vel, err := velero.New(snapshotsEnabled)
	if err != nil {
		return nil, fmt.Errorf("unable to create velero addon: %w", err)
	}
	addons = append(addons, vel)

	aconsole, err := adminconsole.New(defaults.KotsadmNamespace, a.prompt, a.config, a.licenseFile, a.airgapBundle)
	if err != nil {
		return nil, fmt.Errorf("unable to create admin console addon: %w", err)
	}
	addons = append(addons, aconsole)
	return addons, nil
}

// Versions returns a map with the version of each addon that will be applied.
func (a *Applier) Versions(additionalCharts []v1beta1.Chart) (map[string]string, error) {
	addons, err := a.load()
	if err != nil {
		return nil, fmt.Errorf("unable to load addons: %w", err)
	}

	versions := map[string]string{}
	for _, addon := range addons {
		version, err := addon.Version()
		if err != nil {
			return nil, fmt.Errorf("unable to get version (%s): %w", addon.Name(), err)
		}
		for k, v := range version {
			versions[k] = v
		}
	}

	for _, chart := range additionalCharts {
		versions[chart.Name] = chart.Version
	}

	return versions, nil
}

// waitForKubernetes waits until we manage to make a successful connection to the
// Kubernetes API server.
func (a *Applier) waitForKubernetes(ctx context.Context) error {
	loading := spinner.Start()
	defer func() {
		loading.Closef("Kubernetes API server is ready")
	}()
	kcli, err := kubeutils.KubeClient()
	if err != nil {
		return fmt.Errorf("unable to create kubernetes client: %w", err)
	}
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()
	counter := 1
	loading.Infof("1/n Waiting for Kubernetes API server to be ready")
	for {
		select {
		case <-ticker.C:
		case <-ctx.Done():
			return ctx.Err()
		}
		counter++
		if err := kcli.List(ctx, &corev1.NamespaceList{}); err != nil {
			loading.Infof(
				"%d/n Waiting for Kubernetes API server to be ready.",
				counter,
			)
			continue
		}
		return nil
	}
}

func spinForInstallation(ctx context.Context, cli client.Client) error {
	installSpin := spinner.Start()
	installSpin.Infof("Waiting for additional components to complete installation")

	bgCtx, cancel := context.WithCancel(ctx)
	ch := make(chan embeddedclusterv1beta1.InstallationStatus)
	defer cancel()
	spinMut := sync.Mutex{}
	go func() {
		spinMut.Lock()
		defer spinMut.Unlock()
		for {
			select {
			case <-bgCtx.Done():
				return
			case meta, ok := <-ch:
				if !ok {
					return // channel closed
				}

				// figure out what to log
				if meta.State != embeddedclusterv1beta1.InstallationStatePendingChartCreation {
					installSpin.Infof("Waiting for additional components to complete installation: %s", meta.Reason)
				} else {
					chartNames := ""
					if len(meta.PendingCharts) == 0 {
						continue
					} else if len(meta.PendingCharts) == 1 {
						// A
						chartNames = meta.PendingCharts[0]
					} else if len(meta.PendingCharts) == 2 {
						// A and B
						chartNames = strings.Join(meta.PendingCharts, " and ")
					} else {
						// A, B, and C
						chartNames = strings.Join(meta.PendingCharts[:len(meta.PendingCharts)-1], ", ") + " and " + meta.PendingCharts[len(meta.PendingCharts)-1]
					}

					installSpin.Infof("Waiting for additional components %s to complete installation", chartNames)
				}
			}
		}
	}()

	err := kubeutils.WaitForInstallation(ctx, cli, ch)
	if err != nil {
		return fmt.Errorf("unable to wait for installation: %w", err)
	}
	cancel()
	spinMut.Lock() // prevent closing the spinner while we are still writing to it
	installSpin.Closef("Installation is complete!")
	return nil
}

// printKotsadmLinkMessage prints the success message when the admin console is online.
func printKotsadmLinkMessage(licenseFile string) error {
	license, err := helpers.ParseLicense(licenseFile)
	if err != nil {
		return fmt.Errorf("unable to parse license: %w", err)
	}

	successColor := "\033[32m"
	colorReset := "\033[0m"
	ipaddr := defaults.TryDiscoverPublicIP()
	if ipaddr == "" {
		var err error
		ipaddr, err = defaults.PreferredNodeIPAddress()
		if err != nil {
			logrus.Errorf("unable to determine node IP address: %v", err)
			ipaddr = "NODE-IP-ADDRESS"
		}
	}
	successMessage := fmt.Sprintf("Visit the admin console to configure and install %s: %shttp://%s:%v%s",
		license.Spec.AppSlug, successColor, ipaddr, adminconsole.DEFAULT_ADMIN_CONSOLE_NODE_PORT, colorReset,
	)
	logrus.Info(successMessage)

	return nil
}

// NewApplier creates a new Applier instance with all addons registered.
func NewApplier(opts ...Option) *Applier {
	applier := &Applier{
		prompt:       true,
		verbose:      true,
		config:       v1beta1.ClusterConfig{},
		licenseFile:  "",
		airgapBundle: "",
	}
	for _, fn := range opts {
		fn(applier)
	}
	return applier
}
