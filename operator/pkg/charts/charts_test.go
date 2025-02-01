package charts

import (
	"context"
	"testing"

	k0sv1beta1 "github.com/k0sproject/k0s/pkg/apis/k0s/v1beta1"
	"github.com/replicatedhq/embedded-cluster/kinds/apis/v1beta1"
	"github.com/replicatedhq/embedded-cluster/operator/pkg/registry"
	"github.com/replicatedhq/embedded-cluster/pkg/addons/adminconsole"
	"github.com/replicatedhq/embedded-cluster/pkg/addons/embeddedclusteroperator"
	"github.com/replicatedhq/embedded-cluster/pkg/addons/openebs"
	registryAddon "github.com/replicatedhq/embedded-cluster/pkg/addons/registry"
	"github.com/replicatedhq/embedded-cluster/pkg/addons/seaweedfs"
	"github.com/replicatedhq/embedded-cluster/pkg/addons/velero"
	"github.com/replicatedhq/embedded-cluster/pkg/release"
	"github.com/replicatedhq/embedded-cluster/pkg/versions"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
)

const test_openebsValues = `engines:
  local:
    lvm:
      enabled: false
    zfs:
      enabled: false
  replicated:
    mayastor:
      enabled: false
localpv-provisioner:
  analytics:
    enabled: false
  helperPod:
    image:
      registry: proxy.replicated.com/anonymous/
      repository: ""
      tag: ""
  hostpathClass:
    enabled: true
    isDefaultClass: true
  localpv:
    basePath: /var/lib/embedded-cluster/openebs-local
    image:
      registry: proxy.replicated.com/anonymous/
      repository: ""
      tag: ""
lvm-localpv:
  enabled: false
mayastor:
  enabled: false
preUpgradeHook:
  image:
    registry: proxy.replicated.com/anonymous
    repo: ""
    tag: ""
zfs-localpv:
  enabled: false
`

const test_operatorValues = `embeddedBinaryName: test-binary-name
embeddedClusterID: e79f0701-67f3-4abf-a672-42a1f3ed231b
embeddedClusterK0sVersion: 0.0.0
embeddedClusterVersion: v1.2.3-operator
global:
  labels:
    replicated.com/disaster-recovery: infra
    replicated.com/disaster-recovery-chart: embedded-cluster-operator
image:
  repository: docker.io/replicated/embedded-cluster-operator-image
  tag: latest-amd64@sha256:eeed01216b5d2192afbd90e2e1f70419a8758551d8708f9d4b4f50f41d106ce8
utilsImage: abc-repo/ec-utils:latest-amd64@sha256:92dec6e167ff57b35953da389c2f62c8ed9e529fe8dac3c3621269c3a66291f0
`

const test_airgapOperatorValues = `embeddedBinaryName: test-binary-name
embeddedClusterID: e79f0701-67f3-4abf-a672-42a1f3ed231b
embeddedClusterK0sVersion: 0.0.0
embeddedClusterVersion: v1.2.3-operator
global:
  labels:
    replicated.com/disaster-recovery: infra
    replicated.com/disaster-recovery-chart: embedded-cluster-operator
image:
  repository: docker.io/replicated/embedded-cluster-operator-image
  tag: latest-amd64@sha256:eeed01216b5d2192afbd90e2e1f70419a8758551d8708f9d4b4f50f41d106ce8
isAirgap: "true"
utilsImage: abc-repo/ec-utils:latest-amd64@sha256:92dec6e167ff57b35953da389c2f62c8ed9e529fe8dac3c3621269c3a66291f0
`

const test_proxyOperatorValues = `embeddedBinaryName: test-binary-name
embeddedClusterID: e79f0701-67f3-4abf-a672-42a1f3ed231b
embeddedClusterK0sVersion: 0.0.0
embeddedClusterVersion: v1.2.3-operator
extraEnv:
- name: HTTP_PROXY
  value: http://proxy
- name: HTTPS_PROXY
  value: https://proxy
- name: NO_PROXY
  value: test.proxy,1.2.3.4
global:
  labels:
    replicated.com/disaster-recovery: infra
    replicated.com/disaster-recovery-chart: embedded-cluster-operator
image:
  repository: docker.io/replicated/embedded-cluster-operator-image
  tag: latest-amd64@sha256:eeed01216b5d2192afbd90e2e1f70419a8758551d8708f9d4b4f50f41d106ce8
utilsImage: abc-repo/ec-utils:latest-amd64@sha256:92dec6e167ff57b35953da389c2f62c8ed9e529fe8dac3c3621269c3a66291f0
`

const test_overriddenOperatorValues = `embeddedBinaryName: test-binary-name
embeddedClusterID: e79f0701-67f3-4abf-a672-42a1f3ed231b
embeddedClusterK0sVersion: 0.0.0
embeddedClusterVersion: abctest
global:
  labels:
    replicated.com/disaster-recovery: infra
    replicated.com/disaster-recovery-chart: embedded-cluster-operator
image:
  repository: docker.io/replicated/embedded-cluster-operator-image
  tag: latest-amd64@sha256:eeed01216b5d2192afbd90e2e1f70419a8758551d8708f9d4b4f50f41d106ce8
utilsImage: abc-repo/ec-utils:latest-amd64@sha256:92dec6e167ff57b35953da389c2f62c8ed9e529fe8dac3c3621269c3a66291f0
`

const test_onlineAdminConsoleValues = `embeddedClusterID: e79f0701-67f3-4abf-a672-42a1f3ed231b
embeddedClusterVersion: v1.2.3-operator
extraEnv:
- name: ENABLE_IMPROVED_DR
  value: "true"
images:
  kotsadm: ':'
  kurlProxy: ':'
  migrations: ':'
  rqlite: ':'
isAirgap: "false"
isHA: false
isHelmManaged: false
kurlProxy:
  enabled: true
  nodePort: 30000
labels:
  replicated.com/disaster-recovery: infra
  replicated.com/disaster-recovery-chart: admin-console
minimalRBAC: false
passwordSecretRef:
  key: passwordBcrypt
  name: kotsadm-password
privateCAs:
  configmapName: kotsadm-private-cas
  enabled: true
service:
  enabled: false
`

const test_overriddenOnlineAdminConsoleValues = `embeddedClusterID: e79f0701-67f3-4abf-a672-42a1f3ed231b
embeddedClusterVersion: abctest
extraEnv:
- name: ENABLE_IMPROVED_DR
  value: "true"
images:
  kotsadm: ':'
  kurlProxy: ':'
  migrations: ':'
  rqlite: ':'
isAirgap: "false"
isHA: false
isHelmManaged: false
kurlProxy:
  enabled: true
  nodePort: 30000
labels:
  replicated.com/disaster-recovery: infra
  replicated.com/disaster-recovery-chart: admin-console
minimalRBAC: false
passwordSecretRef:
  key: passwordBcrypt
  name: kotsadm-password
privateCAs:
  configmapName: kotsadm-private-cas
  enabled: true
service:
  enabled: false
`

const test_airgapAdminConsoleValues = `embeddedClusterID: e79f0701-67f3-4abf-a672-42a1f3ed231b
embeddedClusterVersion: v1.2.3-operator
extraEnv:
- name: ENABLE_IMPROVED_DR
  value: "true"
images:
  kotsadm: ':'
  kurlProxy: ':'
  migrations: ':'
  rqlite: ':'
isAirgap: "true"
isHA: false
isHelmManaged: false
kurlProxy:
  enabled: true
  nodePort: 30000
labels:
  replicated.com/disaster-recovery: infra
  replicated.com/disaster-recovery-chart: admin-console
minimalRBAC: false
passwordSecretRef:
  key: passwordBcrypt
  name: kotsadm-password
privateCAs:
  configmapName: kotsadm-private-cas
  enabled: true
service:
  enabled: false
`

const test_airgapHAAdminConsoleValues = `embeddedClusterID: e79f0701-67f3-4abf-a672-42a1f3ed231b
embeddedClusterVersion: v1.2.3-operator
extraEnv:
- name: ENABLE_IMPROVED_DR
  value: "true"
images:
  kotsadm: ':'
  kurlProxy: ':'
  migrations: ':'
  rqlite: ':'
isAirgap: "true"
isHA: true
isHelmManaged: false
kurlProxy:
  enabled: true
  nodePort: 30000
labels:
  replicated.com/disaster-recovery: infra
  replicated.com/disaster-recovery-chart: admin-console
minimalRBAC: false
passwordSecretRef:
  key: passwordBcrypt
  name: kotsadm-password
privateCAs:
  configmapName: kotsadm-private-cas
  enabled: true
service:
  enabled: false
`

const test_proxyAdminConsoleValues = `embeddedClusterID: e79f0701-67f3-4abf-a672-42a1f3ed231b
embeddedClusterVersion: v1.2.3-operator
extraEnv:
- name: ENABLE_IMPROVED_DR
  value: "true"
- name: HTTP_PROXY
  value: http://proxy
- name: HTTPS_PROXY
  value: https://proxy
- name: NO_PROXY
  value: test.proxy,1.2.3.4
images:
  kotsadm: ':'
  kurlProxy: ':'
  migrations: ':'
  rqlite: ':'
isAirgap: "false"
isHA: false
isHelmManaged: false
kurlProxy:
  enabled: true
  nodePort: 30000
labels:
  replicated.com/disaster-recovery: infra
  replicated.com/disaster-recovery-chart: admin-console
minimalRBAC: false
passwordSecretRef:
  key: passwordBcrypt
  name: kotsadm-password
privateCAs:
  configmapName: kotsadm-private-cas
  enabled: true
service:
  enabled: false
`

const test_veleroValues = `backupsEnabled: false
configMaps:
  fs-restore-action-config:
    data:
      image: ':'
    labels:
      velero.io/plugin-config: ""
      velero.io/pod-volume-restore: RestoreItemAction
credentials:
  existingSecret: cloud-credentials
deployNodeAgent: true
image:
  repository: ""
  tag: ""
initContainers:
- image: ':'
  imagePullPolicy: IfNotPresent
  name: velero-plugin-for-aws
  volumeMounts:
  - mountPath: /target
    name: plugins
kubectl:
  image:
    repository: ""
    tag: ""
nodeAgent:
  podVolumePath: /var/lib/embedded-cluster/k0s/kubelet/pods
snapshotsEnabled: false
`

const test_registryValues = `configData:
  auth:
    htpasswd:
      path: /auth/htpasswd
      realm: Registry
extraVolumeMounts:
- mountPath: /auth
  name: auth
extraVolumes:
- name: auth
  secret:
    secretName: registry-auth
fullnameOverride: registry
image:
  repository: ""
  tag: ""
persistence:
  accessMode: ReadWriteOnce
  enabled: true
  size: 10Gi
  storageClass: openebs-hostpath
podAnnotations:
  backup.velero.io/backup-volumes: data
replicaCount: 1
service:
  clusterIP: 10.96.0.11
storage: filesystem
tlsSecretName: registry-tls
`

const test_haRegistryValues = `affinity:
  podAntiAffinity:
    requiredDuringSchedulingIgnoredDuringExecution:
    - labelSelector:
        matchExpressions:
        - key: app
          operator: In
          values:
          - docker-registry
      topologyKey: kubernetes.io/hostname
configData:
  auth:
    htpasswd:
      path: /auth/htpasswd
      realm: Registry
  storage:
    s3:
      secure: false
extraVolumeMounts:
- mountPath: /auth
  name: auth
extraVolumes:
- name: auth
  secret:
    secretName: registry-auth
fullnameOverride: registry
image:
  repository: ""
  tag: ""
replicaCount: 2
s3:
  bucket: registry
  encrypt: false
  region: us-east-1
  regionEndpoint: 10.96.0.12:8333
  rootdirectory: /registry
  secure: false
secrets:
  s3:
    secretRef: seaweedfs-s3-rw
service:
  clusterIP: 10.96.0.11
storage: s3
tlsSecretName: registry-tls
`

const test_seaweedfsValues = `filer:
  data:
    size: 1Gi
    storageClass: openebs-hostpath
    type: persistentVolumeClaim
  imageOverride: ':'
  logs:
    size: 1Gi
    storageClass: openebs-hostpath
    type: persistentVolumeClaim
  podAnnotations:
    backup.velero.io/backup-volumes: data-filer,seaweedfs-filer-log-volume
  replicas: 3
  s3:
    createBuckets:
    - anonymousRead: false
      name: registry
    enableAuth: true
    enabled: true
    existingConfigSecret: secret-seaweedfs-s3
global:
  enableReplication: true
  registry: proxy.replicated.com/anonymous/
  replicationPlacment: "001"
master:
  config: |-
    [master.maintenance]
    # periodically run these scripts are the same as running them from 'weed shell'
    # note: running 'fs.meta.save' then 'fs.meta.load' will ensure metadata of all filers
    # are in sync in case of data loss from 1 or more filers
    scripts = """
      ec.encode -fullPercent=95 -quietFor=1h
      ec.rebuild -force
      ec.balance -force
      volume.balance -force
      volume.configure.replication -replication 001 -collectionPattern *
      volume.fix.replication
      fs.meta.save -o filer-backup.meta
      fs.meta.load filer-backup.meta
    """
    sleep_minutes = 17          # sleep minutes between each script execution
  data:
    hostPathPrefix: /var/lib/embedded-cluster/seaweedfs/ssd
  disableHttp: true
  imageOverride: ':'
  logs:
    hostPathPrefix: /var/lib/embedded-cluster/seaweedfs/storage
  replicas: 1
  volumeSizeLimitMB: 30000
volume:
  affinity: |
    # schedule on control-plane nodes
    nodeAffinity:
      requiredDuringSchedulingIgnoredDuringExecution:
        nodeSelectorTerms:
        - matchExpressions:
          - key: node-role.kubernetes.io/control-plane
            operator: Exists
    # schedule on different nodes when possible
    podAntiAffinity:
      requiredDuringSchedulingIgnoredDuringExecution:
        - labelSelector:
            matchExpressions:
            - key: app.kubernetes.io/name
              operator: In
              values:
              - seaweedfs
            - key: app.kubernetes.io/component
              operator: In
              values:
              - volume
          topologyKey: "kubernetes.io/hostname"
  dataDirs:
  - maxVolumes: 50
    name: data
    size: 10Gi
    storageClass: openebs-hostpath
    type: persistentVolumeClaim
  imageOverride: ':'
  podAnnotations:
    backup.velero.io/backup-volumes: data
  replicas: 3
`

var test_addonMetadata = map[string]release.AddonMetadata{}

// this function is used to replace the values of the addons so that we can run unit tests without having to update values constantly
func test_replaceAddonMeta() {
	test_addonMetadata["admin-console"] = adminconsole.Metadata
	adminconsole.Metadata = release.AddonMetadata{
		Version:  "1.2.3-admin-console",
		Location: "oci://proxy.replicated.com/anonymous/registry.replicated.com/library/admin-console",
	}

	test_addonMetadata["embedded-cluster-operator"] = embeddedclusteroperator.Metadata
	embeddedclusteroperator.Metadata = release.AddonMetadata{
		Location: "oci://proxy.replicated.com/anonymous/registry.replicated.com/library/embedded-cluster-operator",
	}
	versions.Version = "v1.2.3-operator" // This is not great, we use this to override the version of the operator chart
	// we can't use the version from the metadata because it won't be set in the operator binary
	// TODO fix this

	test_addonMetadata["openebs"] = openebs.Metadata
	openebs.Metadata = release.AddonMetadata{
		Version:  "1.2.3-openebs",
		Location: "oci://proxy.replicated.com/anonymous/registry.replicated.com/library/openebs",
	}

	test_addonMetadata["registry"] = registryAddon.Metadata
	registryAddon.Metadata = release.AddonMetadata{
		Version:  "1.2.3-registry",
		Location: "oci://proxy.replicated.com/anonymous/registry.replicated.com/library/docker-registry",
	}

	test_addonMetadata["seaweedfs"] = seaweedfs.Metadata
	seaweedfs.Metadata = release.AddonMetadata{
		Version:  "1.2.3-seaweedfs",
		Location: "oci://proxy.replicated.com/anonymous/registry.replicated.com/library/seaweedfs",
	}

	test_addonMetadata["velero"] = velero.Metadata
	velero.Metadata = release.AddonMetadata{
		Version:  "1.2.3-velero",
		Location: "oci://proxy.replicated.com/anonymous/registry.replicated.com/library/velero",
	}

	adminconsole.Render()
	embeddedclusteroperator.Render()
	openebs.Render()
	registryAddon.Render()
	seaweedfs.Render()
	velero.Render()
}

func test_restoreAddonMeta() {
	adminconsole.Metadata = test_addonMetadata["admin-console"]
	embeddedclusteroperator.Metadata = test_addonMetadata["embedded-cluster-operator"]
	openebs.Metadata = test_addonMetadata["openebs"]
	registryAddon.Metadata = test_addonMetadata["registry"]
	seaweedfs.Metadata = test_addonMetadata["seaweedfs"]
	velero.Metadata = test_addonMetadata["velero"]

	adminconsole.Render()
	embeddedclusteroperator.Render()
	openebs.Render()
	registryAddon.Render()
	seaweedfs.Render()
	velero.Render()
}

func Test_generateHelmConfigs(t *testing.T) {
	test_replaceAddonMeta()
	defer test_restoreAddonMeta()

	type args struct {
		in            v1beta1.Extensions
		conditions    []metav1.Condition
		clusterConfig k0sv1beta1.ClusterConfig
	}
	tests := []struct {
		name             string
		args             args
		airgap           bool
		highAvailability bool
		disasterRecovery bool
		operatorLocation string
		proxy            *v1beta1.ProxySpec
		want             *v1beta1.Helm
	}{
		{
			name:             "online non-ha no-velero",
			airgap:           false,
			highAvailability: false,
			disasterRecovery: false,
			args: args{
				in: v1beta1.Extensions{
					Helm: &v1beta1.Helm{
						ConcurrencyLevel: 2,
						Repositories:     nil,
						Charts: []v1beta1.Chart{
							{
								Name:    "test",
								Version: "1.0.0",
								Order:   20,
							},
						},
					},
				},
			},
			want: &v1beta1.Helm{
				ConcurrencyLevel: 1,
				Repositories:     nil,
				Charts: []v1beta1.Chart{
					{
						Name:    "test",
						Version: "1.0.0",
						Order:   120,
					},
					{
						Name:         "openebs",
						ChartName:    "oci://proxy.replicated.com/anonymous/registry.replicated.com/library/openebs",
						Version:      "1.2.3-openebs",
						Values:       test_openebsValues,
						TargetNS:     "openebs",
						ForceUpgrade: ptr.To(false),
						Order:        101,
					},
					{
						Name:         "embedded-cluster-operator",
						ChartName:    "oci://proxy.replicated.com/anonymous/registry.replicated.com/library/embedded-cluster-operator",
						Version:      "1.2.3-operator",
						Values:       test_operatorValues,
						TargetNS:     "embedded-cluster",
						ForceUpgrade: ptr.To(false),
						Order:        103,
					},
					{
						Name:         "admin-console",
						ChartName:    "oci://proxy.replicated.com/anonymous/registry.replicated.com/library/admin-console",
						Version:      "1.2.3-admin-console",
						Values:       test_onlineAdminConsoleValues,
						TargetNS:     "kotsadm",
						ForceUpgrade: ptr.To(false),
						Order:        105,
					},
				},
			},
		},
		{
			name:             "online non-ha no-velero different operator location",
			operatorLocation: "test-operator-location",
			airgap:           false,
			highAvailability: false,
			disasterRecovery: false,
			args: args{
				in: v1beta1.Extensions{
					Helm: &v1beta1.Helm{
						ConcurrencyLevel: 2,
						Repositories:     nil,
						Charts: []v1beta1.Chart{
							{
								Name:    "test",
								Version: "1.0.0",
								Order:   20,
							},
						},
					},
				},
			},
			want: &v1beta1.Helm{
				ConcurrencyLevel: 1,
				Repositories:     nil,
				Charts: []v1beta1.Chart{
					{
						Name:    "test",
						Version: "1.0.0",
						Order:   120,
					},
					{
						Name:         "openebs",
						ChartName:    "oci://proxy.replicated.com/anonymous/registry.replicated.com/library/openebs",
						Version:      "1.2.3-openebs",
						Values:       test_openebsValues,
						TargetNS:     "openebs",
						ForceUpgrade: ptr.To(false),
						Order:        101,
					},
					{
						Name:         "embedded-cluster-operator",
						ChartName:    "test-operator-location",
						Version:      "1.2.3-operator",
						Values:       test_operatorValues,
						TargetNS:     "embedded-cluster",
						ForceUpgrade: ptr.To(false),
						Order:        103,
					},
					{
						Name:         "admin-console",
						ChartName:    "oci://proxy.replicated.com/anonymous/registry.replicated.com/library/admin-console",
						Version:      "1.2.3-admin-console",
						Values:       test_onlineAdminConsoleValues,
						TargetNS:     "kotsadm",
						ForceUpgrade: ptr.To(false),
						Order:        105,
					},
				},
			},
		},
		{
			name:             "online non-ha velero",
			airgap:           false,
			highAvailability: false,
			disasterRecovery: true,
			args: args{
				in: v1beta1.Extensions{
					Helm: &v1beta1.Helm{
						ConcurrencyLevel: 2,
						Repositories:     nil,
						Charts: []v1beta1.Chart{
							{
								Name:    "test",
								Version: "1.0.0",
								Order:   20,
							},
						},
					},
				},
			},
			want: &v1beta1.Helm{
				ConcurrencyLevel: 1,
				Repositories:     nil,
				Charts: []v1beta1.Chart{
					{
						Name:    "test",
						Version: "1.0.0",
						Order:   120,
					},
					{
						Name:         "openebs",
						ChartName:    "oci://proxy.replicated.com/anonymous/registry.replicated.com/library/openebs",
						Version:      "1.2.3-openebs",
						Values:       test_openebsValues,
						TargetNS:     "openebs",
						ForceUpgrade: ptr.To(false),
						Order:        101,
					},
					{
						Name:         "embedded-cluster-operator",
						ChartName:    "oci://proxy.replicated.com/anonymous/registry.replicated.com/library/embedded-cluster-operator",
						Version:      "1.2.3-operator",
						Values:       test_operatorValues,
						TargetNS:     "embedded-cluster",
						ForceUpgrade: ptr.To(false),
						Order:        103,
					},
					{
						Name:         "velero",
						ChartName:    "oci://proxy.replicated.com/anonymous/registry.replicated.com/library/velero",
						Version:      "1.2.3-velero",
						Values:       test_veleroValues,
						TargetNS:     "velero",
						ForceUpgrade: ptr.To(false),
						Order:        103,
					},
					{
						Name:         "admin-console",
						ChartName:    "oci://proxy.replicated.com/anonymous/registry.replicated.com/library/admin-console",
						Version:      "1.2.3-admin-console",
						Values:       test_onlineAdminConsoleValues,
						TargetNS:     "kotsadm",
						ForceUpgrade: ptr.To(false),
						Order:        105,
					},
				},
			},
		},
		{
			name:             "airgap, non-ha, no-velero",
			airgap:           true,
			highAvailability: false,
			disasterRecovery: false,
			args: args{
				in: v1beta1.Extensions{},
			},
			want: &v1beta1.Helm{
				ConcurrencyLevel: 1,
				Repositories:     nil,
				Charts: []v1beta1.Chart{
					{
						Name:         "openebs",
						ChartName:    "oci://proxy.replicated.com/anonymous/registry.replicated.com/library/openebs",
						Version:      "1.2.3-openebs",
						Values:       test_openebsValues,
						TargetNS:     "openebs",
						ForceUpgrade: ptr.To(false),
						Order:        101,
					},
					{
						Name:         "docker-registry",
						ChartName:    "oci://proxy.replicated.com/anonymous/registry.replicated.com/library/docker-registry",
						Version:      "1.2.3-registry",
						Values:       test_registryValues,
						TargetNS:     "registry",
						ForceUpgrade: ptr.To(false),
						Order:        103,
					},
					{
						Name:         "embedded-cluster-operator",
						ChartName:    "oci://proxy.replicated.com/anonymous/registry.replicated.com/library/embedded-cluster-operator",
						Version:      "1.2.3-operator",
						Values:       test_airgapOperatorValues,
						TargetNS:     "embedded-cluster",
						ForceUpgrade: ptr.To(false),
						Order:        103,
					},
					{
						Name:         "admin-console",
						ChartName:    "oci://proxy.replicated.com/anonymous/registry.replicated.com/library/admin-console",
						Version:      "1.2.3-admin-console",
						Values:       test_airgapAdminConsoleValues,
						TargetNS:     "kotsadm",
						ForceUpgrade: ptr.To(false),
						Order:        105,
					},
				},
			},
		},
		{
			name:             "ha airgap enabled, migration incomplete",
			airgap:           true,
			highAvailability: true,
			args: args{
				in: v1beta1.Extensions{},
				conditions: []metav1.Condition{
					{
						Type:   registry.RegistryMigrationStatusConditionType,
						Status: metav1.ConditionFalse,
						Reason: "MigrationInProgress",
					},
				},
			},
			want: &v1beta1.Helm{
				ConcurrencyLevel: 1,
				Repositories:     nil,
				Charts: []v1beta1.Chart{
					{
						Name:         "openebs",
						ChartName:    "oci://proxy.replicated.com/anonymous/registry.replicated.com/library/openebs",
						Version:      "1.2.3-openebs",
						Values:       test_openebsValues,
						TargetNS:     "openebs",
						ForceUpgrade: ptr.To(false),
						Order:        101,
					},
					{
						Name:         "docker-registry",
						ChartName:    "oci://proxy.replicated.com/anonymous/registry.replicated.com/library/docker-registry",
						Version:      "1.2.3-registry",
						Values:       test_registryValues,
						TargetNS:     "registry",
						ForceUpgrade: ptr.To(false),
						Order:        103,
					},
					{
						Name:         "seaweedfs",
						ChartName:    "oci://proxy.replicated.com/anonymous/registry.replicated.com/library/seaweedfs",
						Version:      "1.2.3-seaweedfs",
						Values:       test_seaweedfsValues,
						TargetNS:     "seaweedfs",
						ForceUpgrade: ptr.To(false),
						Order:        102,
					},
					{
						Name:         "embedded-cluster-operator",
						ChartName:    "oci://proxy.replicated.com/anonymous/registry.replicated.com/library/embedded-cluster-operator",
						Version:      "1.2.3-operator",
						Values:       test_airgapOperatorValues,
						TargetNS:     "embedded-cluster",
						ForceUpgrade: ptr.To(false),
						Order:        103,
					},
					{
						Name:         "admin-console",
						ChartName:    "oci://proxy.replicated.com/anonymous/registry.replicated.com/library/admin-console",
						Version:      "1.2.3-admin-console",
						Values:       test_airgapHAAdminConsoleValues,
						TargetNS:     "kotsadm",
						ForceUpgrade: ptr.To(false),
						Order:        105,
					},
				},
			},
		},
		{
			name:             "ha airgap enabled, migration complete",
			airgap:           true,
			highAvailability: true,
			args: args{
				in: v1beta1.Extensions{},
				conditions: []metav1.Condition{
					{
						Type:   registry.RegistryMigrationStatusConditionType,
						Status: metav1.ConditionTrue,
						Reason: "MigrationComplete",
					},
				},
			},
			want: &v1beta1.Helm{
				ConcurrencyLevel: 1,
				Repositories:     nil,
				Charts: []v1beta1.Chart{
					{
						Name:         "openebs",
						ChartName:    "oci://proxy.replicated.com/anonymous/registry.replicated.com/library/openebs",
						Version:      "1.2.3-openebs",
						Values:       test_openebsValues,
						TargetNS:     "openebs",
						ForceUpgrade: ptr.To(false),
						Order:        101,
					},
					{
						Name:         "docker-registry",
						ChartName:    "oci://proxy.replicated.com/anonymous/registry.replicated.com/library/docker-registry",
						Version:      "1.2.3-registry",
						Values:       test_haRegistryValues,
						TargetNS:     "registry",
						ForceUpgrade: ptr.To(false),
						Order:        103,
					},
					{
						Name:         "seaweedfs",
						ChartName:    "oci://proxy.replicated.com/anonymous/registry.replicated.com/library/seaweedfs",
						Version:      "1.2.3-seaweedfs",
						Values:       test_seaweedfsValues,
						TargetNS:     "seaweedfs",
						ForceUpgrade: ptr.To(false),
						Order:        102,
					},
					{
						Name:         "embedded-cluster-operator",
						ChartName:    "oci://proxy.replicated.com/anonymous/registry.replicated.com/library/embedded-cluster-operator",
						Version:      "1.2.3-operator",
						Values:       test_airgapOperatorValues,
						TargetNS:     "embedded-cluster",
						ForceUpgrade: ptr.To(false),
						Order:        103,
					},
					{
						Name:         "admin-console",
						ChartName:    "oci://proxy.replicated.com/anonymous/registry.replicated.com/library/admin-console",
						Version:      "1.2.3-admin-console",
						Values:       test_airgapHAAdminConsoleValues,
						TargetNS:     "kotsadm",
						ForceUpgrade: ptr.To(false),
						Order:        105,
					},
				},
			},
		},
		{
			name:             "online non-ha no-velero with-proxy",
			airgap:           false,
			highAvailability: false,
			disasterRecovery: false,
			proxy: &v1beta1.ProxySpec{
				HTTPProxy:  "http://proxy",
				HTTPSProxy: "https://proxy",
				NoProxy:    "test.proxy,1.2.3.4",
			},
			args: args{
				in: v1beta1.Extensions{
					Helm: &v1beta1.Helm{
						ConcurrencyLevel: 2,
						Repositories:     nil,
						Charts: []v1beta1.Chart{
							{
								Name:    "test",
								Version: "1.0.0",
								Order:   20,
							},
						},
					},
				},
			},
			want: &v1beta1.Helm{
				ConcurrencyLevel: 1,
				Repositories:     nil,
				Charts: []v1beta1.Chart{
					{
						Name:    "test",
						Version: "1.0.0",
						Order:   120,
					},
					{
						Name:         "openebs",
						ChartName:    "oci://proxy.replicated.com/anonymous/registry.replicated.com/library/openebs",
						Version:      "1.2.3-openebs",
						Values:       test_openebsValues,
						TargetNS:     "openebs",
						ForceUpgrade: ptr.To(false),
						Order:        101,
					},
					{
						Name:         "embedded-cluster-operator",
						ChartName:    "oci://proxy.replicated.com/anonymous/registry.replicated.com/library/embedded-cluster-operator",
						Version:      "1.2.3-operator",
						Values:       test_proxyOperatorValues,
						TargetNS:     "embedded-cluster",
						ForceUpgrade: ptr.To(false),
						Order:        103,
					},
					{
						Name:         "admin-console",
						ChartName:    "oci://proxy.replicated.com/anonymous/registry.replicated.com/library/admin-console",
						Version:      "1.2.3-admin-console",
						Values:       test_proxyAdminConsoleValues,
						TargetNS:     "kotsadm",
						ForceUpgrade: ptr.To(false),
						Order:        105,
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			installation := v1beta1.Installation{
				Spec: v1beta1.InstallationSpec{
					Config: &v1beta1.ConfigSpec{
						Version:    "1.0.0",
						Extensions: tt.args.in,
					},
					AirGap:           tt.airgap,
					HighAvailability: tt.highAvailability,
					LicenseInfo: &v1beta1.LicenseInfo{
						IsDisasterRecoverySupported: tt.disasterRecovery,
					},
					ClusterID:  "e79f0701-67f3-4abf-a672-42a1f3ed231b",
					BinaryName: "test-binary-name",
					Proxy:      tt.proxy,
				},
				Status: v1beta1.InstallationStatus{
					Conditions: tt.args.conditions,
				},
			}

			images := []string{
				"abc-repo/ec-utils:latest-amd64@sha256:92dec6e167ff57b35953da389c2f62c8ed9e529fe8dac3c3621269c3a66291f0",                                    // the utils image is needed
				"docker.io/replicated/another-image:latest-arm64@sha256:a9ab9db181f9898283a87be0f79d85cb8f3d22a790b71f52c8a9d339e225dedd",                   // this image is here to ensure we're actually searching
				"docker.io/replicated/embedded-cluster-operator-image:latest-amd64@sha256:eeed01216b5d2192afbd90e2e1f70419a8758551d8708f9d4b4f50f41d106ce8", // this operator image is needed
			}

			ol := "oci://proxy.replicated.com/anonymous/registry.replicated.com/library/embedded-cluster-operator"
			if tt.operatorLocation != "" {
				ol = tt.operatorLocation
			}

			req := require.New(t)
			got, err := generateHelmConfigs(context.TODO(), &installation, &tt.args.clusterConfig, images, ol)
			req.NoError(err)
			req.Equal(tt.want, got)
		})
	}
}

func Test_applyUserProvidedAddonOverrides(t *testing.T) {
	tests := []struct {
		name         string
		installation *v1beta1.Installation
		config       *v1beta1.Helm
		want         *v1beta1.Helm
	}{
		{
			name:         "no config",
			installation: &v1beta1.Installation{},
			config: &v1beta1.Helm{
				Charts: []v1beta1.Chart{
					{
						Name:    "test",
						Version: "1.0.0",
						Values:  "abc: xyz",
					},
				},
			},
			want: &v1beta1.Helm{
				Charts: []v1beta1.Chart{
					{
						Name:    "test",
						Version: "1.0.0",
						Values:  "abc: xyz",
					},
				},
			},
		},
		{
			name: "no override",
			installation: &v1beta1.Installation{
				Spec: v1beta1.InstallationSpec{
					Config: &v1beta1.ConfigSpec{
						UnsupportedOverrides: v1beta1.UnsupportedOverrides{
							BuiltInExtensions: []v1beta1.BuiltInExtension{},
						},
					},
				},
			},
			config: &v1beta1.Helm{
				Charts: []v1beta1.Chart{
					{
						Name:    "test",
						Version: "1.0.0",
						Values:  "abc: xyz",
					},
				},
			},
			want: &v1beta1.Helm{
				Charts: []v1beta1.Chart{
					{
						Name:    "test",
						Version: "1.0.0",
						Values:  "abc: xyz",
					},
				},
			},
		},
		{
			name: "single addition",
			installation: &v1beta1.Installation{
				Spec: v1beta1.InstallationSpec{
					Config: &v1beta1.ConfigSpec{
						UnsupportedOverrides: v1beta1.UnsupportedOverrides{
							BuiltInExtensions: []v1beta1.BuiltInExtension{
								{
									Name:   "test",
									Values: "foo: bar",
								},
							},
						},
					},
				},
			},
			config: &v1beta1.Helm{
				Charts: []v1beta1.Chart{
					{
						Name:    "test",
						Version: "1.0.0",
						Values:  "abc: xyz",
					},
				},
			},
			want: &v1beta1.Helm{
				Charts: []v1beta1.Chart{
					{
						Name:    "test",
						Version: "1.0.0",
						Values:  "abc: xyz\nfoo: bar\n",
					},
				},
			},
		},
		{
			name: "single override",
			installation: &v1beta1.Installation{
				Spec: v1beta1.InstallationSpec{
					Config: &v1beta1.ConfigSpec{
						UnsupportedOverrides: v1beta1.UnsupportedOverrides{
							BuiltInExtensions: []v1beta1.BuiltInExtension{
								{
									Name:   "test",
									Values: "abc: newvalue",
								},
							},
						},
					},
				},
			},
			config: &v1beta1.Helm{
				Charts: []v1beta1.Chart{
					{
						Name:    "test",
						Version: "1.0.0",
						Values:  "abc: xyz",
					},
				},
			},
			want: &v1beta1.Helm{
				Charts: []v1beta1.Chart{
					{
						Name:    "test",
						Version: "1.0.0",
						Values:  "abc: newvalue\n",
					},
				},
			},
		},
		{
			name: "multiple additions and overrides",
			installation: &v1beta1.Installation{
				Spec: v1beta1.InstallationSpec{
					Config: &v1beta1.ConfigSpec{
						UnsupportedOverrides: v1beta1.UnsupportedOverrides{
							BuiltInExtensions: []v1beta1.BuiltInExtension{
								{
									Name:   "chart0",
									Values: "added: added\noverridden: overridden",
								},
								{
									Name:   "chart1",
									Values: "foo: replacement",
								},
							},
						},
					},
				},
			},
			config: &v1beta1.Helm{
				ConcurrencyLevel: 999,
				Repositories: []k0sv1beta1.Repository{
					{
						Name: "repo",
						URL:  "https://repo",
					},
				},
				Charts: []v1beta1.Chart{
					{
						Name:    "chart0",
						Version: "1.0.0",
						Values:  "abc: xyz",
					},
					{
						Name:    "chart1",
						Version: "1.0.0",
						Values:  "foo: bar",
					},
				},
			},
			want: &v1beta1.Helm{
				ConcurrencyLevel: 999,
				Repositories: []k0sv1beta1.Repository{
					{
						Name: "repo",
						URL:  "https://repo",
					},
				},
				Charts: []v1beta1.Chart{
					{
						Name:    "chart0",
						Version: "1.0.0",
						Values:  "abc: xyz\nadded: added\noverridden: overridden\n",
					},
					{
						Name:    "chart1",
						Version: "1.0.0",
						Values:  "foo: replacement\n",
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := require.New(t)
			got, err := applyUserProvidedAddonOverrides(tt.installation, tt.config)
			req.NoError(err)
			req.Equal(tt.want, got)
		})
	}
}

func Test_operatorImages(t *testing.T) {
	tests := []struct {
		name    string
		images  []string
		want    map[string]release.AddonImage
		wantErr string
	}{
		{
			name:    "no images",
			images:  []string{},
			wantErr: "no embedded-cluster-operator-image found in images",
		},
		{
			name: "images but no match",
			images: []string{
				"docker.io/replicated/another-image:latest-arm64@sha256:a9ab9db181f9898283a87be0f79d85cb8f3d22a790b71f52c8a9d339e225dedd",
			},
			wantErr: "no embedded-cluster-operator-image found in images",
		},
		{
			name: "operator but no utils",
			images: []string{
				"docker.io/replicated/another-image:latest-arm64@sha256:a9ab9db181f9898283a87be0f79d85cb8f3d22a790b71f52c8a9d339e225dedd",
				"docker.io/replicated/embedded-cluster-operator-image:latest-amd64@sha256:eeed01216b5d2192afbd90e2e1f70419a8758551d8708f9d4b4f50f41d106ce8",
			},
			wantErr: "no ec-utils found in images",
		},
		{
			name: "images but no sha256",
			images: []string{
				"docker.io/replicated/embedded-cluster-operator-image:latest-amd64",
				"docker.io/replicated/ec-utils:latest-amd64",
			},
			want: map[string]release.AddonImage{
				"embedded-cluster-operator": {
					Repo: "docker.io/replicated/embedded-cluster-operator-image",
					Tag: map[string]string{
						"amd64": "latest-amd64",
						"arm64": "latest-amd64",
					},
				},
				"utils": {
					Repo: "docker.io/replicated/ec-utils",
					Tag: map[string]string{
						"amd64": "latest-amd64",
						"arm64": "latest-amd64",
					},
				},
			},
		},
		{
			name: "normal input",
			images: []string{
				"proxy.replicated.com/anonymous/kotsadm/kotsadm-migrations:v1.117.3-amd64@sha256:56d2765497a57c06ef6e9f7705cf5218d21a978d197575a3c22fe7d84db07f0a",
				"proxy.replicated.com/anonymous/kotsadm/kotsadm:v1.117.3-amd64@sha256:d47ac4df627ac357452efffb717776adb452c3ab9b466ef3ccaf808df722b7a6",
				"proxy.replicated.com/anonymous/kotsadm/kurl-proxy:v1.117.3-amd64@sha256:816bcbc273ec51255d7b459e49831ce09fd361db4a295d31f61d7af02177860f",
				"proxy.replicated.com/anonymous/kotsadm/rqlite:8.30.4-r0-amd64@sha256:884ac56b236e059e420858c94d90a083fe48b666c8b3433da612b9380906ce41",
				"proxy.replicated.com/anonymous/registry.k8s.io/kube-proxy:v1.29.9-amd64@sha256:eb9e12af6de3613c05afcb9743a30589c16454bfa085c3091248a6f55b799304",
				"proxy.replicated.com/anonymous/registry.k8s.io/pause:3.9-amd64@sha256:8d4106c88ec0bd28001e34c975d65175d994072d65341f62a8ab0754b0fafe10",
				"proxy.replicated.com/anonymous/replicated/ec-calico-cni:3.28.2-r0-amd64@sha256:61de906f9ca1b2abdcca4e15769fa289b2949f0ece27a9247d21a960e70c31eb",
				"proxy.replicated.com/anonymous/replicated/ec-calico-kube-controllers:3.28.2-r0-amd64@sha256:10774c8200c36b8e7af3ad9c88bbf637eb553bbe4dc97810aee9d1a899a9da4a",
				"proxy.replicated.com/anonymous/replicated/ec-calico-node:3.28.2-r0-amd64@sha256:8946806cce8889d63feb26440e2cb1781b372083d41c882020faaebf834bfa3b",
				"proxy.replicated.com/anonymous/replicated/ec-coredns:1.11.3-r7-amd64@sha256:1258b039d78e85c17bec40e587f5cb963998c6039a7d727bef09a84d7eedddba",
				"proxy.replicated.com/anonymous/replicated/ec-kubectl:1.31.1-r0-amd64@sha256:92701c7575ffd5ddf025099451add26aa9484c68646d6fc865a7f8b95ccf1168",
				"proxy.replicated.com/anonymous/replicated/ec-metrics-server:0.7.2-r1-amd64@sha256:05e3db63e7ecce0a543fad1a3c7292ce14e49efbc2ef65524266990df52c95a5",
				"proxy.replicated.com/anonymous/replicated/ec-openebs-linux-utils:4.1.1-amd64@sha256:aecf4bc398935bc74d7b1c008b5394ba01fea8d25d79d758666de8e6dc9994fb",
				"proxy.replicated.com/anonymous/replicated/ec-openebs-provisioner-localpv:4.1.1-r0-amd64@sha256:de7f0330f19d50d9f1f804ae44d388998a2d1d1eb11e45965005404463f0d0bd",
				"proxy.replicated.com/anonymous/replicated/ec-registry:2.8.3-r0@sha256:5b76ebd0a362009e31a05ac487c690f5ece0e11f6c4d9261ca63a3f162b57660",
				"proxy.replicated.com/anonymous/replicated/ec-seaweedfs:3.71-r1-amd64@sha256:fe06f85b49d3cf35718a62851e4712617fbeca16fb9100fdd8bfd09c202b98dc",
				"proxy.replicated.com/anonymous/replicated/ec-utils:latest-amd64@sha256:2f3c5d81565eae3aea22f408af9a8ee91cd4ba010612c50c6be564869390639f",
				"proxy.replicated.com/anonymous/replicated/ec-velero-plugin-for-aws:1.10.1-r1-amd64@sha256:0766116b831d1028bfc8a47ed6c9c23a2890ae013592a5ef7eb613b9c70e5f97",
				"proxy.replicated.com/anonymous/replicated/ec-velero-restore-helper:1.14.1-r1-amd64@sha256:aef818ef819274578240a8dfaf70546c762db98090d292ab3e8e44a6658fae95",
				"proxy.replicated.com/anonymous/replicated/ec-velero:1.14.1-r1-amd64@sha256:9a3b8341b74cef8deadea4b3cbaa1d91a0cda06a57821a0dc376428ef44ddfe7",
				"proxy.replicated.com/anonymous/replicated/embedded-cluster-local-artifact-mirror:v1.14.2-k8s-1.29@sha256:54463ce6b6fba13a25138890aa1ac28ae4f93f53cdb78a99d15abfdc1b5eddf5",
				"proxy.replicated.com/anonymous/replicated/embedded-cluster-operator-image:v1.14.2-k8s-1.29-amd64@sha256:45a45e2ec6b73d2db029354cccfe7eb150dd7ef9dffe806db36de9b9ba0a66c6",
			},
			want: map[string]release.AddonImage{
				"embedded-cluster-operator": {
					Repo: "proxy.replicated.com/anonymous/replicated/embedded-cluster-operator-image",
					Tag: map[string]string{
						"amd64": "v1.14.2-k8s-1.29-amd64@sha256:45a45e2ec6b73d2db029354cccfe7eb150dd7ef9dffe806db36de9b9ba0a66c6",
						"arm64": "v1.14.2-k8s-1.29-amd64@sha256:45a45e2ec6b73d2db029354cccfe7eb150dd7ef9dffe806db36de9b9ba0a66c6",
					},
				},
				"utils": {
					Repo: "proxy.replicated.com/anonymous/replicated/ec-utils",
					Tag: map[string]string{
						"amd64": "latest-amd64@sha256:2f3c5d81565eae3aea22f408af9a8ee91cd4ba010612c50c6be564869390639f",
						"arm64": "latest-amd64@sha256:2f3c5d81565eae3aea22f408af9a8ee91cd4ba010612c50c6be564869390639f",
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := require.New(t)
			got, err := operatorImages(tt.images)
			if tt.wantErr != "" {
				req.Error(err)
				req.EqualError(err, tt.wantErr)
				return
			}
			req.Equal(tt.want, got)
		})
	}
}
