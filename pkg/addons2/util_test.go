package addons2

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_operatorImages(t *testing.T) {
	tests := []struct {
		name           string
		images         []string
		wantRepo       string
		wantTag        string
		wantUtilsImage string
		wantErr        string
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
			wantRepo:       "docker.io/replicated/embedded-cluster-operator-image",
			wantTag:        "latest-amd64",
			wantUtilsImage: "docker.io/replicated/ec-utils:latest-amd64",
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
			wantRepo:       "proxy.replicated.com/anonymous/replicated/embedded-cluster-operator-image",
			wantTag:        "v1.14.2-k8s-1.29-amd64@sha256:45a45e2ec6b73d2db029354cccfe7eb150dd7ef9dffe806db36de9b9ba0a66c6",
			wantUtilsImage: "proxy.replicated.com/anonymous/replicated/ec-utils:latest-amd64@sha256:2f3c5d81565eae3aea22f408af9a8ee91cd4ba010612c50c6be564869390639f",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := require.New(t)
			gotRepo, gotTag, gotUtilsImage, err := operatorImages(tt.images)
			if tt.wantErr != "" {
				req.Error(err)
				req.EqualError(err, tt.wantErr)
				return
			}
			req.Equal(tt.wantRepo, gotRepo)
			req.Equal(tt.wantTag, gotTag)
			req.Equal(tt.wantUtilsImage, gotUtilsImage)
		})
	}
}
