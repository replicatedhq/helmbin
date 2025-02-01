package helm

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Masterminds/semver/v3"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/chartutil"
	"helm.sh/helm/v3/pkg/downloader"
	"helm.sh/helm/v3/pkg/getter"
	"helm.sh/helm/v3/pkg/pusher"
	"helm.sh/helm/v3/pkg/registry"
	"helm.sh/helm/v3/pkg/release"
	"helm.sh/helm/v3/pkg/releaseutil"
	"helm.sh/helm/v3/pkg/repo"
	"helm.sh/helm/v3/pkg/storage/driver"
	"helm.sh/helm/v3/pkg/uploader"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

var (
	// getters is a list of known getters for both http and
	// oci schemes.
	getters = getter.Providers{
		getter.Provider{
			Schemes: []string{"http", "https"},
			New:     getter.NewHTTPGetter,
		},
		getter.Provider{
			Schemes: []string{"oci"},
			New:     getter.NewOCIGetter,
		},
	}

	// pushers holds all supported pushers (uploaders).
	pushers = pusher.Providers{
		pusher.Provider{
			Schemes: []string{"oci"},
			New:     pusher.NewOCIPusher,
		},
	}
)

var (
	_ Client = (*Helm)(nil)
)

type Client interface {
	Close() error
	AddRepo(repo *repo.Entry) error
	Latest(reponame, chart string) (string, error)
	PullOCI(url string, version string) (string, error)
	Pull(repo string, chart string, version string) (string, error)
	RegistryAuth(server, user, pass string) error
	Push(path, dst string) error
	GetChartMetadata(chartPath string) (*chart.Metadata, error)
	Install(ctx context.Context, opts InstallOptions) (*release.Release, error)
	Upgrade(ctx context.Context, opts UpgradeOptions) (*release.Release, error)
	Uninstall(ctx context.Context, opts UninstallOptions) error
	Render(releaseName string, chartPath string, values map[string]interface{}, namespace string) ([][]byte, error)
}

type RESTClientGetterFactory func(namespace string) genericclioptions.RESTClientGetter

func NewHelm(opts HelmOptions) (*Helm, error) {
	tmpdir, err := os.MkdirTemp(os.TempDir(), "helm-cache-*")
	if err != nil {
		return nil, err
	}
	registryOpts := []registry.ClientOption{}
	if opts.Writer != nil {
		registryOpts = append(registryOpts, registry.ClientOptWriter(opts.Writer))
	}
	var kversion *semver.Version
	if opts.K0sVersion != "" {
		sv, err := semver.NewVersion(opts.K0sVersion)
		if err != nil {
			return nil, fmt.Errorf("parse k0s version: %w", err)
		}
		kversion = sv
	}
	regcli, err := registry.NewClient(registryOpts...)
	if err != nil {
		return nil, fmt.Errorf("create registry client: %w", err)
	}
	return &Helm{
		tmpdir:        tmpdir,
		kubeconfig:    opts.KubeConfig,
		kversion:      kversion,
		regcli:        regcli,
		logFn:         opts.LogFn,
		getterFactory: opts.RESTClientGetterFactory,
		airgapPath:    opts.AirgapPath,
	}, nil
}

type HelmOptions struct {
	KubeConfig              string
	K0sVersion              string
	AirgapPath              string
	Writer                  io.Writer
	LogFn                   action.DebugLog
	RESTClientGetterFactory RESTClientGetterFactory
}

type InstallOptions struct {
	ReleaseName  string
	ChartPath    string
	ChartVersion string
	Values       map[string]interface{}
	Namespace    string
	Timeout      time.Duration
}

type UpgradeOptions struct {
	ReleaseName  string
	ChartPath    string
	ChartVersion string
	Values       map[string]interface{}
	Namespace    string
	Timeout      time.Duration
	Force        bool
}

type UninstallOptions struct {
	ReleaseName    string
	Namespace      string
	Wait           bool
	IgnoreNotFound bool
}

type Helm struct {
	tmpdir        string
	kversion      *semver.Version
	kubeconfig    string
	regcli        *registry.Client
	repocfg       string
	repos         []*repo.Entry
	logFn         action.DebugLog
	getterFactory RESTClientGetterFactory
	airgapPath    string
}

func (h *Helm) prepare() error {
	if h.repocfg != "" {
		return nil
	}

	data, err := yaml.Marshal(repo.File{Repositories: h.repos})
	if err != nil {
		return fmt.Errorf("marshal repositories: %w", err)
	}

	repocfg := filepath.Join(h.tmpdir, "config.yaml")
	if err := os.WriteFile(repocfg, data, 0644); err != nil {
		return fmt.Errorf("write repositories: %w", err)
	}

	for _, repository := range h.repos {
		chrepo, err := repo.NewChartRepository(
			repository, getters,
		)
		if err != nil {
			return fmt.Errorf("create chart repo: %w", err)
		}
		chrepo.CachePath = h.tmpdir
		_, err = chrepo.DownloadIndexFile()
		if err != nil {
			return fmt.Errorf("download index file: %w", err)
		}
	}
	h.repocfg = repocfg
	return nil
}

func (h *Helm) Close() error {
	return os.RemoveAll(h.tmpdir)
}

func (h *Helm) AddRepo(repo *repo.Entry) error {
	h.repos = append(h.repos, repo)
	return nil
}

func (h *Helm) Latest(reponame, chart string) (string, error) {
	for _, repository := range h.repos {
		if repository.Name != reponame {
			continue
		}
		chrepo, err := repo.NewChartRepository(repository, getters)
		if err != nil {
			return "", fmt.Errorf("create chart repo: %w", err)
		}
		chrepo.CachePath = h.tmpdir
		idx, err := chrepo.DownloadIndexFile()
		if err != nil {
			return "", fmt.Errorf("download index file: %w", err)
		}

		repoidx, err := repo.LoadIndexFile(idx)
		if err != nil {
			return "", fmt.Errorf("load index file: %w", err)
		}

		versions, ok := repoidx.Entries[chart]
		if !ok {
			return "", fmt.Errorf("chart %s not found", chart)
		} else if len(versions) == 0 {
			return "", fmt.Errorf("chart %s has no versions", chart)
		}

		if len(versions) == 0 {
			return "", fmt.Errorf("chart %s has no versions", chart)
		}
		return versions[0].Version, nil
	}
	return "", fmt.Errorf("repository %s not found", reponame)
}

func (h *Helm) PullOCI(url string, version string) (string, error) {
	if err := h.prepare(); err != nil {
		return "", fmt.Errorf("prepare: %w", err)
	}

	dl := downloader.ChartDownloader{
		Out:              io.Discard,
		Options:          []getter.Option{},
		RepositoryConfig: h.repocfg,
		RepositoryCache:  h.tmpdir,
		Getters:          getters,
	}

	path, _, err := dl.DownloadTo(url, version, h.tmpdir)
	if err != nil {
		return "", fmt.Errorf("download chart %s: %w", url, err)
	}

	return path, nil
}

func (h *Helm) Pull(repo string, chart string, version string) (string, error) {
	if err := h.prepare(); err != nil {
		return "", fmt.Errorf("prepare: %w", err)
	}

	dl := downloader.ChartDownloader{
		Out:              io.Discard,
		Options:          []getter.Option{},
		RepositoryConfig: h.repocfg,
		RepositoryCache:  h.tmpdir,
		Getters:          getters,
	}

	ref := fmt.Sprintf("%s/%s", repo, chart)
	dst, _, err := dl.DownloadTo(ref, version, os.TempDir())
	if err != nil {
		return "", fmt.Errorf("download chart %s: %w", ref, err)
	}

	return dst, nil
}

func (h *Helm) RegistryAuth(server, user, pass string) error {
	return h.regcli.Login(server, registry.LoginOptBasicAuth(user, pass))
}

func (h *Helm) Push(path, dst string) error {
	up := uploader.ChartUploader{
		Out:     os.Stdout,
		Pushers: pushers,
		Options: []pusher.Option{pusher.WithRegistryClient(h.regcli)},
	}

	return up.UploadTo(path, dst)
}

func (h *Helm) GetChartMetadata(chartPath string) (*chart.Metadata, error) {
	chartRequested, err := loader.Load(chartPath)
	if err != nil {
		return nil, fmt.Errorf("load chart: %w", err)
	}

	return chartRequested.Metadata, nil
}

// reference: https://github.com/helm/helm/blob/0d66425d9a745d8a289b1a5ebb6ccc744436da95/cmd/helm/upgrade.go#L122-L125
func (h *Helm) ReleaseExists(ctx context.Context, namespace string, releaseName string) (bool, error) {
	cfg, err := h.getActionCfg(namespace)
	if err != nil {
		return false, fmt.Errorf("get action configuration: %w", err)
	}

	client := action.NewHistory(cfg)
	client.Max = 1

	versions, err := client.Run(releaseName)
	if errors.Is(err, driver.ErrReleaseNotFound) || isReleaseUninstalled(versions) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("get release history: %w", err)
	}

	return true, nil
}

func isReleaseUninstalled(versions []*release.Release) bool {
	return len(versions) > 0 && versions[len(versions)-1].Info.Status == release.StatusUninstalled
}

func (h *Helm) Install(ctx context.Context, opts InstallOptions) (*release.Release, error) {
	cfg, err := h.getActionCfg(opts.Namespace)
	if err != nil {
		return nil, fmt.Errorf("get action configuration: %w", err)
	}

	client := action.NewInstall(cfg)
	client.ReleaseName = opts.ReleaseName
	client.Namespace = opts.Namespace
	client.Replace = true
	client.CreateNamespace = true
	client.WaitForJobs = true
	client.Wait = true
	client.Atomic = true

	if opts.Timeout != 0 {
		client.Timeout = opts.Timeout
	} else {
		client.Timeout = 5 * time.Minute
	}

	var localPath string
	if h.airgapPath == "" {
		// online, pull chart from remote
		localPath, err = h.PullOCI(opts.ChartPath, opts.ChartVersion)
		if err != nil {
			return nil, fmt.Errorf("pull oci: %w", err)
		}
		defer os.RemoveAll(localPath)
	} else {
		// airgapped, use chart from airgap path
		localPath = filepath.Join(h.airgapPath, fmt.Sprintf("%s-%s.tgz", opts.ReleaseName, opts.ChartVersion))
	}

	chartRequested, err := loader.Load(localPath)
	if err != nil {
		return nil, fmt.Errorf("load chart: %w", err)
	}

	if req := chartRequested.Metadata.Dependencies; req != nil {
		if err := action.CheckDependencies(chartRequested, req); err != nil {
			return nil, fmt.Errorf("check chart dependencies: %w", err)
		}
	}

	cleanVals := cleanUpGenericMap(opts.Values)

	release, err := client.RunWithContext(ctx, chartRequested, cleanVals)
	if err != nil {
		return nil, fmt.Errorf("helm install: %w", err)
	}

	return release, nil
}

func (h *Helm) Upgrade(ctx context.Context, opts UpgradeOptions) (*release.Release, error) {
	cfg, err := h.getActionCfg(opts.Namespace)
	if err != nil {
		return nil, fmt.Errorf("get action configuration: %w", err)
	}

	client := action.NewUpgrade(cfg)
	client.Namespace = opts.Namespace
	client.WaitForJobs = true
	client.Wait = true
	client.Atomic = true
	client.Force = opts.Force

	if opts.Timeout != 0 {
		client.Timeout = opts.Timeout
	} else {
		client.Timeout = 5 * time.Minute
	}

	var localPath string
	if h.airgapPath == "" {
		// online, pull chart from remote
		localPath, err = h.PullOCI(opts.ChartPath, opts.ChartVersion)
		if err != nil {
			return nil, fmt.Errorf("pull oci: %w", err)
		}
		defer os.RemoveAll(localPath)
	} else {
		// airgapped, use chart from airgap path
		localPath = filepath.Join(h.airgapPath, fmt.Sprintf("%s-%s.tgz", opts.ReleaseName, opts.ChartVersion))
	}

	chartRequested, err := loader.Load(localPath)
	if err != nil {
		return nil, fmt.Errorf("load chart: %w", err)
	}

	if req := chartRequested.Metadata.Dependencies; req != nil {
		if err := action.CheckDependencies(chartRequested, req); err != nil {
			return nil, fmt.Errorf("check chart dependencies: %w", err)
		}
	}

	cleanVals := cleanUpGenericMap(opts.Values)

	release, err := client.RunWithContext(ctx, opts.ReleaseName, chartRequested, cleanVals)
	if err != nil {
		return nil, fmt.Errorf("helm upgrade: %w", err)
	}

	return release, nil
}

func (h *Helm) Uninstall(ctx context.Context, opts UninstallOptions) error {
	cfg, err := h.getActionCfg(opts.Namespace)
	if err != nil {
		return fmt.Errorf("get action configuration: %w", err)
	}

	client := action.NewUninstall(cfg)
	client.Wait = opts.Wait
	client.IgnoreNotFound = opts.IgnoreNotFound

	if deadline, ok := ctx.Deadline(); ok {
		client.Timeout = time.Until(deadline)
	}

	if _, err := client.Run(opts.ReleaseName); err != nil {
		return fmt.Errorf("uninstall release: %w", err)
	}

	return nil
}

func (h *Helm) Render(releaseName string, chartPath string, values map[string]interface{}, namespace string) ([][]byte, error) {
	cfg := &action.Configuration{}

	client := action.NewInstall(cfg)
	client.DryRun = true
	client.ReleaseName = releaseName
	client.Replace = true
	client.ClientOnly = true
	client.IncludeCRDs = true
	client.Namespace = namespace

	if h.kversion != nil {
		// since ClientOnly is true we need to initialize KubeVersion otherwise resorts defaults
		client.KubeVersion = &chartutil.KubeVersion{
			Version: fmt.Sprintf("v%d.%d.0", h.kversion.Major(), h.kversion.Minor()),
			Major:   fmt.Sprintf("%d", h.kversion.Major()),
			Minor:   fmt.Sprintf("%d", h.kversion.Minor()),
		}
	}

	chartRequested, err := loader.Load(chartPath)
	if err != nil {
		return nil, fmt.Errorf("load chart: %w", err)
	}

	if req := chartRequested.Metadata.Dependencies; req != nil {
		if err := action.CheckDependencies(chartRequested, req); err != nil {
			return nil, fmt.Errorf("failed dependency check: %w", err)
		}
	}

	cleanVals := cleanUpGenericMap(values)

	release, err := client.Run(chartRequested, cleanVals)
	if err != nil {
		return nil, fmt.Errorf("run render: %w", err)
	}

	var manifests bytes.Buffer
	fmt.Fprintln(&manifests, strings.TrimSpace(release.Manifest))
	for _, m := range release.Hooks {
		fmt.Fprintf(&manifests, "---\n# Source: %s\n%s\n", m.Path, m.Manifest)
	}

	resources := [][]byte{}
	splitManifests := releaseutil.SplitManifests(manifests.String())
	for _, manifest := range splitManifests {
		manifest = strings.TrimSpace(manifest)
		resources = append(resources, []byte(manifest))
	}

	return resources, nil
}

func (h *Helm) getActionCfg(namespace string) (*action.Configuration, error) {
	getter := h.getRESTClientGetter(namespace)

	cfg := &action.Configuration{}
	var logFn action.DebugLog
	if h.logFn != nil {
		logFn = h.logFn
	} else {
		logFn = _logFn
	}
	if err := cfg.Init(getter, namespace, "secret", logFn); err != nil {
		return nil, fmt.Errorf("init helm configuration: %w", err)
	}
	return cfg, nil
}

func (h *Helm) getRESTClientGetter(namespace string) genericclioptions.RESTClientGetter {
	if h.getterFactory != nil {
		return h.getterFactory(namespace)
	}

	cfgFlags := &genericclioptions.ConfigFlags{
		Namespace: &namespace,
	}
	if h.kubeconfig != "" {
		cfgFlags.KubeConfig = &h.kubeconfig
	}
	return cfgFlags
}

// cleanUpGenericMap is a helper to "cleanup" generic yaml parsing where nested maps
// are unmarshalled with type map[interface{}]interface{}
func cleanUpGenericMap(in map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	for k, v := range in {
		result[fmt.Sprintf("%v", k)] = cleanUpMapValue(v)
	}
	return result
}

// Cleans up the value in the map, recurses in case of arrays and maps
func cleanUpMapValue(v interface{}) interface{} {
	// Keep null values as nil to avoid type mismatches
	if v == nil {
		return nil
	}
	switch v := v.(type) {
	case []interface{}:
		return cleanUpInterfaceArray(v)
	case map[string]interface{}:
		return cleanUpGenericMap(v)
	case map[interface{}]interface{}:
		return cleanUpInterfaceMap(v)
	case string:
		return v
	case int:
		return v
	case bool:
		return v
	case float64:
		return v
	default:
		return fmt.Sprintf("%v", v)
	}
}

// Cleans up a slice of interfaces into slice of actual values
func cleanUpInterfaceArray(in []interface{}) []interface{} {
	result := make([]interface{}, len(in))
	for i, v := range in {
		result[i] = cleanUpMapValue(v)
	}
	return result
}

// Cleans up the map keys to be strings
func cleanUpInterfaceMap(in map[interface{}]interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	for k, v := range in {
		result[fmt.Sprintf("%v", k)] = cleanUpMapValue(v)
	}
	return result
}

func _logFn(format string, args ...interface{}) {
	log := logrus.WithField("component", "helm")
	log.Debugf(format, args...)
}
