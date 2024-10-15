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
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
)

func Test_generateHelmConfigs(t *testing.T) {
	var addonMetadata = map[string]release.AddonMetadata{}

	// this function is used to replace the values of the addons so that we can test without having to update tests constantly
	replaceAddonMeta := func() {
		addonMetadata["admin-console"] = adminconsole.Metadata
		adminconsole.Metadata = release.AddonMetadata{
			Version:  "1.2.3-admin-console",
			Location: "oci://proxy.replicated.com/anonymous/registry.replicated.com/library/admin-console",
		}

		addonMetadata["embedded-cluster-operator"] = embeddedclusteroperator.Metadata
		embeddedclusteroperator.Metadata = release.AddonMetadata{
			Version:  "1.2.3-operator",
			Location: "oci://proxy.replicated.com/anonymous/registry.replicated.com/library/embedded-cluster-operator",
		}

		addonMetadata["openebs"] = openebs.Metadata
		openebs.Metadata = release.AddonMetadata{
			Version:  "1.2.3-openebs",
			Location: "oci://proxy.replicated.com/anonymous/registry.replicated.com/library/openebs",
		}

		addonMetadata["registry"] = registryAddon.Metadata
		registryAddon.Metadata = release.AddonMetadata{
			Version:  "1.2.3-registry",
			Location: "oci://proxy.replicated.com/anonymous/registry.replicated.com/library/docker-registry",
		}

		addonMetadata["seaweedfs"] = seaweedfs.Metadata
		seaweedfs.Metadata = release.AddonMetadata{
			Version:  "1.2.3-seaweedfs",
			Location: "oci://proxy.replicated.com/anonymous/registry.replicated.com/library/seaweedfs",
		}

		addonMetadata["velero"] = velero.Metadata
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

	restoreAddonMeta := func() {
		adminconsole.Metadata = addonMetadata["admin-console"]
		embeddedclusteroperator.Metadata = addonMetadata["embedded-cluster-operator"]
		openebs.Metadata = addonMetadata["openebs"]
		registryAddon.Metadata = addonMetadata["registry"]
		seaweedfs.Metadata = addonMetadata["seaweedfs"]
		velero.Metadata = addonMetadata["velero"]

		adminconsole.Render()
		embeddedclusteroperator.Render()
		openebs.Render()
		registryAddon.Render()
		seaweedfs.Render()
		velero.Render()
	}

	replaceAddonMeta()
	defer restoreAddonMeta()

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
						Name:      "openebs",
						ChartName: "oci://proxy.replicated.com/anonymous/registry.replicated.com/library/openebs",
						Version:   "1.2.3-openebs",
						Values: `engines:
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
`,
						TargetNS:     "openebs",
						ForceUpgrade: ptr.To(false),
						Order:        101,
					},
					{
						Name:      "embedded-cluster-operator",
						ChartName: "oci://proxy.replicated.com/anonymous/registry.replicated.com/library/embedded-cluster-operator",
						Version:   "1.2.3-operator",
						Values: `embeddedBinaryName: test-binary-name
embeddedClusterID: e79f0701-67f3-4abf-a672-42a1f3ed231b
embeddedClusterK0sVersion: 0.0.0
embeddedClusterVersion: v0.0.0
global:
  labels:
    replicated.com/disaster-recovery: infra
    replicated.com/disaster-recovery-chart: embedded-cluster-operator
image:
  repository: ""
  tag: ""
kotsVersion: 1.2.3-admin-console
utilsImage: ':'
`,
						TargetNS:     "embedded-cluster",
						ForceUpgrade: ptr.To(false),
						Order:        103,
					},
					{
						Name:      "admin-console",
						ChartName: "oci://proxy.replicated.com/anonymous/registry.replicated.com/library/admin-console",
						Version:   "1.2.3-admin-console",
						Values: `embeddedClusterID: e79f0701-67f3-4abf-a672-42a1f3ed231b
embeddedClusterVersion: v0.0.0
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
`,
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
						Name:      "openebs",
						ChartName: "oci://proxy.replicated.com/anonymous/registry.replicated.com/library/openebs",
						Version:   "1.2.3-openebs",
						Values: `engines:
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
`,
						TargetNS:     "openebs",
						ForceUpgrade: ptr.To(false),
						Order:        101,
					},
					{
						Name:      "docker-registry",
						ChartName: "oci://proxy.replicated.com/anonymous/registry.replicated.com/library/docker-registry",
						Version:   "1.2.3-registry",
						Values: `configData:
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
`,
						TargetNS:     "registry",
						ForceUpgrade: ptr.To(false),
						Order:        103,
					},
					{
						Name:      "embedded-cluster-operator",
						ChartName: "oci://proxy.replicated.com/anonymous/registry.replicated.com/library/embedded-cluster-operator",
						Version:   "1.2.3-operator",
						Values: `embeddedBinaryName: test-binary-name
embeddedClusterID: e79f0701-67f3-4abf-a672-42a1f3ed231b
embeddedClusterK0sVersion: 0.0.0
embeddedClusterVersion: v0.0.0
global:
  labels:
    replicated.com/disaster-recovery: infra
    replicated.com/disaster-recovery-chart: embedded-cluster-operator
image:
  repository: ""
  tag: ""
kotsVersion: 1.2.3-admin-console
utilsImage: ':'
`,
						TargetNS:     "embedded-cluster",
						ForceUpgrade: ptr.To(false),
						Order:        103,
					},
					{
						Name:      "admin-console",
						ChartName: "oci://proxy.replicated.com/anonymous/registry.replicated.com/library/admin-console",
						Version:   "1.2.3-admin-console",
						Values: `embeddedClusterID: e79f0701-67f3-4abf-a672-42a1f3ed231b
embeddedClusterVersion: v0.0.0
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
`,
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
						Name:      "openebs",
						ChartName: "oci://proxy.replicated.com/anonymous/registry.replicated.com/library/openebs",
						Version:   "1.2.3-openebs",
						Values: `engines:
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
`,
						TargetNS:     "openebs",
						ForceUpgrade: ptr.To(false),
						Order:        101,
					},
					{
						Name:      "docker-registry",
						ChartName: "oci://proxy.replicated.com/anonymous/registry.replicated.com/library/docker-registry",
						Version:   "1.2.3-registry",
						Values: `configData:
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
`,
						TargetNS:     "registry",
						ForceUpgrade: ptr.To(false),
						Order:        103,
					},
					{
						Name:      "seaweedfs",
						ChartName: "oci://proxy.replicated.com/anonymous/registry.replicated.com/library/seaweedfs",
						Version:   "1.2.3-seaweedfs",
						Values: `filer:
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
  data:
    hostPathPrefix: /var/lib/embedded-cluster/seaweedfs/ssd
  enableReplication: true
  logs:
    hostPathPrefix: /var/lib/embedded-cluster/seaweedfs/storage
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
`,
						TargetNS:     "seaweedfs",
						ForceUpgrade: ptr.To(false),
						Order:        102,
					},
					{
						Name:      "embedded-cluster-operator",
						ChartName: "oci://proxy.replicated.com/anonymous/registry.replicated.com/library/embedded-cluster-operator",
						Version:   "1.2.3-operator",
						Values: `embeddedBinaryName: test-binary-name
embeddedClusterID: e79f0701-67f3-4abf-a672-42a1f3ed231b
embeddedClusterK0sVersion: 0.0.0
embeddedClusterVersion: v0.0.0
global:
  labels:
    replicated.com/disaster-recovery: infra
    replicated.com/disaster-recovery-chart: embedded-cluster-operator
image:
  repository: ""
  tag: ""
kotsVersion: 1.2.3-admin-console
utilsImage: ':'
`,
						TargetNS:     "embedded-cluster",
						ForceUpgrade: ptr.To(false),
						Order:        103,
					},
					{
						Name:      "admin-console",
						ChartName: "oci://proxy.replicated.com/anonymous/registry.replicated.com/library/admin-console",
						Version:   "1.2.3-admin-console",
						Values: `embeddedClusterID: e79f0701-67f3-4abf-a672-42a1f3ed231b
embeddedClusterVersion: v0.0.0
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
`,
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
						Name:      "openebs",
						ChartName: "oci://proxy.replicated.com/anonymous/registry.replicated.com/library/openebs",
						Version:   "1.2.3-openebs",
						Values: `engines:
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
`,
						TargetNS:     "openebs",
						ForceUpgrade: ptr.To(false),
						Order:        101,
					},
					{
						Name:      "docker-registry",
						ChartName: "oci://proxy.replicated.com/anonymous/registry.replicated.com/library/docker-registry",
						Version:   "1.2.3-registry",
						Values: `affinity:
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
  regionEndpoint: DYNAMIC
  rootdirectory: /registry
  secure: false
secrets:
  s3:
    secretRef: seaweedfs-s3-rw
service:
  clusterIP: 10.96.0.11
storage: s3
tlsSecretName: registry-tls
`,
						TargetNS:     "registry",
						ForceUpgrade: ptr.To(false),
						Order:        103,
					},
					{
						Name:      "seaweedfs",
						ChartName: "oci://proxy.replicated.com/anonymous/registry.replicated.com/library/seaweedfs",
						Version:   "1.2.3-seaweedfs",
						Values: `filer:
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
  data:
    hostPathPrefix: /var/lib/embedded-cluster/seaweedfs/ssd
  enableReplication: true
  logs:
    hostPathPrefix: /var/lib/embedded-cluster/seaweedfs/storage
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
`,
						TargetNS:     "seaweedfs",
						ForceUpgrade: ptr.To(false),
						Order:        102,
					},
					{
						Name:      "embedded-cluster-operator",
						ChartName: "oci://proxy.replicated.com/anonymous/registry.replicated.com/library/embedded-cluster-operator",
						Version:   "1.2.3-operator",
						Values: `embeddedBinaryName: test-binary-name
embeddedClusterID: e79f0701-67f3-4abf-a672-42a1f3ed231b
embeddedClusterK0sVersion: 0.0.0
embeddedClusterVersion: v0.0.0
global:
  labels:
    replicated.com/disaster-recovery: infra
    replicated.com/disaster-recovery-chart: embedded-cluster-operator
image:
  repository: ""
  tag: ""
kotsVersion: 1.2.3-admin-console
utilsImage: ':'
`,
						TargetNS:     "embedded-cluster",
						ForceUpgrade: ptr.To(false),
						Order:        103,
					},
					{
						Name:      "admin-console",
						ChartName: "oci://proxy.replicated.com/anonymous/registry.replicated.com/library/admin-console",
						Version:   "1.2.3-admin-console",
						Values: `embeddedClusterID: e79f0701-67f3-4abf-a672-42a1f3ed231b
embeddedClusterVersion: v0.0.0
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
`,
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
				},
				Status: v1beta1.InstallationStatus{
					Conditions: tt.args.conditions,
				},
			}

			req := require.New(t)
			got, err := generateHelmConfigs(context.TODO(), &installation, &tt.args.clusterConfig)
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
				Repositories: []v1beta1.Repository{
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
				Repositories: []v1beta1.Repository{
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

func Test_updateInfraChartsFromInstall(t *testing.T) {
	type args struct {
		in            *v1beta1.Installation
		clusterConfig k0sv1beta1.ClusterConfig
		charts        []v1beta1.Chart
	}
	tests := []struct {
		name string
		args args
		want []v1beta1.Chart
	}{
		{
			name: "other chart",
			args: args{
				in: &v1beta1.Installation{
					Spec: v1beta1.InstallationSpec{
						ClusterID: "abc",
					},
				},
				charts: []v1beta1.Chart{
					{
						Name:   "test",
						Values: "abc: xyz",
					},
				},
			},
			want: []v1beta1.Chart{
				{
					Name:   "test",
					Values: "abc: xyz",
				},
			},
		},
		{
			name: "admin console and operator",
			args: args{
				in: &v1beta1.Installation{
					Spec: v1beta1.InstallationSpec{
						ClusterID:        "testid",
						BinaryName:       "testbin",
						AirGap:           true,
						HighAvailability: true,
					},
				},
				charts: []v1beta1.Chart{
					{
						Name:   "test",
						Values: "abc: xyz",
					},
					{
						Name:   "admin-console",
						Values: "abc: xyz",
					},
					{
						Name:   "embedded-cluster-operator",
						Values: "this: that",
					},
				},
			},
			want: []v1beta1.Chart{
				{
					Name:   "test",
					Values: "abc: xyz",
				},
				{
					Name:         "admin-console",
					Values:       "abc: xyz\nembeddedClusterID: testid\nisAirgap: \"true\"\nisHA: true\n",
					ForceUpgrade: ptr.To(false),
				},
				{
					Name:         "embedded-cluster-operator",
					Values:       "embeddedBinaryName: testbin\nembeddedClusterID: testid\nthis: that\n",
					ForceUpgrade: ptr.To(false),
				},
			},
		},
		{
			name: "admin console and operator with proxy",
			args: args{
				in: &v1beta1.Installation{
					Spec: v1beta1.InstallationSpec{
						ClusterID:        "testid",
						BinaryName:       "testbin",
						AirGap:           false,
						HighAvailability: false,
						Proxy: &v1beta1.ProxySpec{
							HTTPProxy:  "http://proxy",
							HTTPSProxy: "https://proxy",
							NoProxy:    "noproxy",
						},
					},
				},
				charts: []v1beta1.Chart{
					{
						Name:   "test",
						Values: "abc: xyz",
					},
					{
						Name:   "admin-console",
						Values: "abc: xyz",
					},
					{
						Name:   "embedded-cluster-operator",
						Values: "this: that",
					},
				},
			},
			want: []v1beta1.Chart{
				{
					Name:   "test",
					Values: "abc: xyz",
				},
				{
					Name:         "admin-console",
					Values:       "abc: xyz\nembeddedClusterID: testid\nextraEnv:\n- name: HTTP_PROXY\n  value: http://proxy\n- name: HTTPS_PROXY\n  value: https://proxy\n- name: NO_PROXY\n  value: noproxy\nisAirgap: \"false\"\nisHA: false\n",
					ForceUpgrade: ptr.To(false),
				},
				{
					Name:         "embedded-cluster-operator",
					Values:       "embeddedBinaryName: testbin\nembeddedClusterID: testid\nextraEnv:\n- name: HTTP_PROXY\n  value: http://proxy\n- name: HTTPS_PROXY\n  value: https://proxy\n- name: NO_PROXY\n  value: noproxy\nthis: that\n",
					ForceUpgrade: ptr.To(false),
				},
			},
		},
		{
			name: "velero with proxy",
			args: args{
				in: &v1beta1.Installation{
					Spec: v1beta1.InstallationSpec{
						ClusterID:        "testid",
						BinaryName:       "testbin",
						AirGap:           false,
						HighAvailability: false,
						Proxy: &v1beta1.ProxySpec{
							HTTPProxy:  "http://proxy",
							HTTPSProxy: "https://proxy",
							NoProxy:    "noproxy",
						},
					},
				},
				charts: []v1beta1.Chart{
					{
						Name:   "velero",
						Values: "abc: xyz\nconfiguration:\n  extraEnvVars: {}\n",
					},
				},
			},
			want: []v1beta1.Chart{
				{
					Name:         "velero",
					Values:       "abc: xyz\nconfiguration:\n  extraEnvVars:\n    HTTP_PROXY: http://proxy\n    HTTPS_PROXY: https://proxy\n    NO_PROXY: noproxy\n",
					ForceUpgrade: ptr.To(false),
				},
			},
		}, {
			name: "docker-registry",
			args: args{
				in: &v1beta1.Installation{
					Spec: v1beta1.InstallationSpec{
						ClusterID:  "testid",
						BinaryName: "testbin",
						AirGap:     true,
						Network:    &v1beta1.NetworkSpec{ServiceCIDR: "1.2.0.0/16"},
					},
				},
				clusterConfig: k0sv1beta1.ClusterConfig{},
				charts: []v1beta1.Chart{
					{
						Name:   "docker-registry",
						Values: "this: that\nand: another\nservice:\n  clusterIP: \"abc\"\n",
					},
				},
			},
			want: []v1beta1.Chart{
				{
					Name:         "docker-registry",
					Values:       "and: another\nservice:\n  clusterIP: 1.2.0.11\nthis: that\n",
					ForceUpgrade: ptr.To(false),
				},
			},
		},
		{
			name: "docker-registry ha",
			args: args{
				in: &v1beta1.Installation{
					Spec: v1beta1.InstallationSpec{
						ClusterID:        "testid",
						BinaryName:       "testbin",
						AirGap:           true,
						HighAvailability: true,
					},
				},
				clusterConfig: k0sv1beta1.ClusterConfig{},
				charts: []v1beta1.Chart{
					{
						Name: "docker-registry",
						Values: `image:
  tag: 2.8.3
replicaCount: 2
s3:
  bucket: registry
  encrypt: false
  region: us-east-1
  regionEndpoint: DYNAMIC
  rootdirectory: /registry
  secure: false
secrets:
  s3:
    secretRef: seaweedfs-s3-rw`,
					},
				},
			},
			want: []v1beta1.Chart{
				{
					Name: "docker-registry",
					Values: `image:
  tag: 2.8.3
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
`,
					ForceUpgrade: ptr.To(false),
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := require.New(t)
			got, err := updateInfraChartsFromInstall(tt.args.in, &tt.args.clusterConfig, tt.args.charts)
			req.NoError(err)
			req.ElementsMatch(tt.want, got)
		})
	}
}
