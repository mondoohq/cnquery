package k8s

import (
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnquery/motor/providers"
	"go.mondoo.com/cnquery/motor/providers/k8s"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestListNodesAKS(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	p := k8s.NewMockKubernetesProvider(mockCtrl)

	nodes := []corev1.Node{
		{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Node",
				APIVersion: "v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: "aks-default-36939070-vmss000000",
				UID:  "acc8d118-f62a-4743-a55c-71dd19201c6c",
				Annotations: map[string]string{
					"csi.volume.kubernetes.io/nodeid":                        `{"disk.csi.azure.com":"aks-default-36939070-vmss000000","file.csi.azure.com":"aks-default-36939070-vmss000000"}`,
					"node.alpha.kubernetes.io/ttl":                           "0",
					"volumes.kubernetes.io/controller-managed-attach-detach": "true",
				},
				Labels: map[string]string{
					"agentpool":                                       "default",
					"beta.kubernetes.io/arch":                         "amd64",
					"beta.kubernetes.io/instance-type":                "standard_d2_v2",
					"beta.kubernetes.io/os":                           "linux",
					"failure-domain.beta.kubernetes.io/region":        "eastus",
					"failure-domain.beta.kubernetes.io/zone":          "0",
					"kubernetes.azure.com/agentpool":                  "default",
					"kubernetes.azure.com/cluster":                    "MC_mondoo-operator-tests-wcou_mondoo-operator-tests-wcou_eastus",
					"kubernetes.azure.com/kubelet-identity-client-id": "c032ffd9-e9c3-4c4b-bece-1cee42d3da09",
					"kubernetes.azure.com/mode":                       "system",
					"kubernetes.azure.com/node-image-version":         "AKSUbuntu-1804containerd-2022.08.15",
					"kubernetes.azure.com/os-sku":                     "Ubuntu",
					"kubernetes.azure.com/role":                       "agent",
					"kubernetes.azure.com/storageprofile":             "managed",
					"kubernetes.azure.com/storagetier":                "Standard_LRS",
					"kubernetes.io/arch":                              "amd64",
					"kubernetes.io/hostname":                          "aks-default-36939070-vmss000000",
					"kubernetes.io/os":                                "linux",
					"kubernetes.io/role":                              "agent",
					"node-role.kubernetes.io/agent":                   "",
					"node.kubernetes.io/instance-type":                "standard_d2_v2",
					"storageprofile":                                  "managed",
					"storagetier":                                     "Standard_LRS",
					"topology.disk.csi.azure.com/zone":                "",
					"topology.kubernetes.io/region":                   "eastus",
					"topology.kubernetes.io/zone":                     "0",
				},
			},
			Spec: corev1.NodeSpec{
				ProviderID: "azure:///subscriptions/f1a2873a-6b27-4097-aa7c-3df51f103e96/resourceGroups/mc_mondoo-operator-tests-wcou_mondoo-operator-tests-wcou_eastus/providers/Microsoft.Compute/virtualMachineScaleSets/aks-default-36939070-vmss/virtualMachines/0",
				PodCIDR:    "10.244.0.0/24",
				PodCIDRs:   []string{"10.244.0.0/24"},
			},
		},
	}

	p.EXPECT().Runtime().Return("k8s-cluster")
	p.EXPECT().Nodes().Return(nodes, nil)

	clusterIdentifier := "//platformid.api.mondoo.app/runtime/k8s/uid/e26043bb-8669-48a2-b684-b1e132198cdc"

	pCfg := &providers.Config{}
	assets, relInfo, err := ListNodes(p, pCfg, clusterIdentifier)
	require.NoError(t, err)
	require.Len(t, assets, 1)
	require.Equal(t, "Kubernetes Node", assets[0].Platform.Title)
	require.Equal(t, "k8s-node", assets[0].Platform.Name)
	require.Equal(t, providers.Kind_KIND_K8S_OBJECT, assets[0].Platform.Kind)
	require.ElementsMatch(t, []string{"k8s"}, assets[0].Platform.Family)
	require.Equal(t, []string{"//platformid.api.mondoo.app/runtime/k8s/uid/e26043bb-8669-48a2-b684-b1e132198cdc/nodes/name/aks-default-36939070-vmss000000"}, assets[0].PlatformIds)

	// Adds relatonship to host
	require.Len(t, assets[0].RelatedAssets, 1)
	require.Equal(t, "aks-default-36939070-vmss000000", assets[0].RelatedAssets[0].Name)
	require.Equal(t, providers.Kind_KIND_VIRTUAL_MACHINE, assets[0].RelatedAssets[0].Platform.Kind)
	require.Equal(t, providers.RUNTIME_AZ_COMPUTE, assets[0].RelatedAssets[0].Platform.Runtime)
	require.Equal(t, "amd64", assets[0].RelatedAssets[0].Platform.Arch)
	require.Equal(t, []string{"//platformid.api.mondoo.app/runtime/azure/subscriptions/f1a2873a-6b27-4097-aa7c-3df51f103e96/resourceGroups/mc_mondoo-operator-tests-wcou_mondoo-operator-tests-wcou_eastus/providers/Microsoft.Compute/virtualMachineScaleSets/aks-default-36939070-vmss/virtualMachines/0"}, assets[0].RelatedAssets[0].PlatformIds)

	require.NotNil(t, relInfo[0].hostInstanceAsset)
	require.Equal(t, assets[0].RelatedAssets[0], relInfo[0].hostInstanceAsset)
	require.NotNil(t, relInfo[0].cloudAccountAsset)
	require.Equal(t, []string{"//platformid.api.mondoo.app/runtime/azure/subscriptions/f1a2873a-6b27-4097-aa7c-3df51f103e96"}, relInfo[0].cloudAccountAsset.PlatformIds)
	require.Equal(t, providers.Kind_KIND_API, relInfo[0].cloudAccountAsset.Platform.Kind)
	require.Equal(t, providers.RUNTIME_AZ, relInfo[0].cloudAccountAsset.Platform.Runtime)
}

func TestListNodesGKE(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	p := k8s.NewMockKubernetesProvider(mockCtrl)

	nodes := []corev1.Node{
		{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Node",
				APIVersion: "v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: "gke-gke-cluster-generic-pool-4dfcd37f-s3d6",
				UID:  "f2cd325c-23eb-465d-8843-9e53665779f0",
				Annotations: map[string]string{
					"container.googleapis.com/instance_id": "8976889368772093420",
				},
				Labels: map[string]string{
					"beta.kubernetes.io/arch":                  "amd64",
					"beta.kubernetes.io/instance-type":         "n1-standard-2",
					"beta.kubernetes.io/os":                    "linux",
					"cloud.google.com/gke-boot-disk":           "pd-standard",
					"cloud.google.com/gke-container-runtime":   "docker",
					"cloud.google.com/gke-netd-ready":          "true",
					"cloud.google.com/gke-nodepool":            "generic-pool",
					"cloud.google.com/gke-os-distribution":     "cos",
					"cloud.google.com/machine-family":          "n1",
					"cluster_name":                             "gke-cluster",
					"failure-domain.beta.kubernetes.io/region": "us-central1",
					"failure-domain.beta.kubernetes.io/zone":   "us-central1-b",
					"iam.gke.io/gke-metadata-server-enabled":   "true",
					"kubernetes.io/arch":                       "amd64",
					"kubernetes.io/hostname":                   "gke-gke-cluster-generic-pool-4dfcd37f-s3d6",
					"kubernetes.io/os":                         "linux",
					"node.kubernetes.io/instance-type":         "n1-standard-2",
					"node.kubernetes.io/masq-agent-ds-ready":   "true",
					"node_pool":                                "generic-pool",
					"topology.gke.io/zone":                     "us-central1-b",
					"topology.kubernetes.io/region":            "us-central1",
					"topology.kubernetes.io/zone":              "us-central1-b",
				},
			},
			Spec: corev1.NodeSpec{
				ProviderID: "gce://mondoo-test/us-central1-b/gke-gke-cluster-generic-pool-4dfcd37f-s3d6",
				PodCIDR:    "192.168.1.0/24",
				PodCIDRs:   []string{"192.168.1.0/24"},
			},
		},
	}

	p.EXPECT().Runtime().Return("k8s-cluster")
	p.EXPECT().Nodes().Return(nodes, nil)

	clusterIdentifier := "//platformid.api.mondoo.app/runtime/k8s/uid/e26043bb-8669-48a2-b684-b1e132198cdc"

	pCfg := &providers.Config{}
	assets, relInfo, err := ListNodes(p, pCfg, clusterIdentifier)
	require.NoError(t, err)
	require.Len(t, assets, 1)
	require.Equal(t, "Kubernetes Node", assets[0].Platform.Title)
	require.Equal(t, "k8s-node", assets[0].Platform.Name)
	require.Equal(t, providers.Kind_KIND_K8S_OBJECT, assets[0].Platform.Kind)
	require.ElementsMatch(t, []string{"k8s"}, assets[0].Platform.Family)
	require.Equal(t, []string{"//platformid.api.mondoo.app/runtime/k8s/uid/e26043bb-8669-48a2-b684-b1e132198cdc/nodes/name/gke-gke-cluster-generic-pool-4dfcd37f-s3d6"}, assets[0].PlatformIds)

	// Adds relatonship to host
	require.Len(t, assets[0].RelatedAssets, 1)
	require.Equal(t, "gke-gke-cluster-generic-pool-4dfcd37f-s3d6", assets[0].RelatedAssets[0].Name)
	require.Equal(t, providers.Kind_KIND_VIRTUAL_MACHINE, assets[0].RelatedAssets[0].Platform.Kind)
	require.Equal(t, providers.RUNTIME_GCP_COMPUTE, assets[0].RelatedAssets[0].Platform.Runtime)
	require.Equal(t, "amd64", assets[0].RelatedAssets[0].Platform.Arch)
	require.Equal(t, []string{"//platformid.api.mondoo.app/runtime/gcp/compute/v1/projects/mondoo-test/zones/us-central1-b/instances/8976889368772093420"}, assets[0].RelatedAssets[0].PlatformIds)

	require.NotNil(t, relInfo[0].hostInstanceAsset)
	require.Equal(t, assets[0].RelatedAssets[0], relInfo[0].hostInstanceAsset)
	require.NotNil(t, relInfo[0].cloudAccountAsset)
	require.Equal(t, []string{"//platformid.api.mondoo.app/runtime/gcp/projects/mondoo-test"}, relInfo[0].cloudAccountAsset.PlatformIds)
	require.Equal(t, providers.Kind_KIND_API, relInfo[0].cloudAccountAsset.Platform.Kind)
	require.Equal(t, providers.RUNTIME_GCP, relInfo[0].cloudAccountAsset.Platform.Runtime)
}

func TestListNodesEKS(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	p := k8s.NewMockKubernetesProvider(mockCtrl)

	nodes := []corev1.Node{
		{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Node",
				APIVersion: "v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: "ip-10-0-5-36.eu-central-1.compute.internal",
				UID:  "c9a5bb24-e77b-46fd-be55-8a247faee098",
				Annotations: map[string]string{
					"alpha.kubernetes.io/provided-node-ip": "10.0.5.36",
				},
				Labels: map[string]string{
					"beta.kubernetes.io/arch":                       "amd64",
					"beta.kubernetes.io/instance-type":              "m5zn.large",
					"beta.kubernetes.io/os":                         "linux",
					"eks.amazonaws.com/capacityType":                "SPOT",
					"eks.amazonaws.com/nodegroup":                   "eks-managed-nodes-l3il-20220901164719853800000006",
					"eks.amazonaws.com/nodegroup-image":             "ami-01c52a64630ff492f",
					"eks.amazonaws.com/sourceLaunchTemplateId":      "lt-0b3c2c84c209ec814",
					"eks.amazonaws.com/sourceLaunchTemplateVersion": "1",
					"failure-domain.beta.kubernetes.io/region":      "eu-central-1",
					"failure-domain.beta.kubernetes.io/zone":        "eu-central-1b",
					"k8s.io/cloud-provider-aws":                     "10f49535c88faa0a8024328860a01464",
					"kubernetes.io/arch":                            "amd64",
					"kubernetes.io/hostname":                        "ip-10-0-5-36.eu-central-1.compute.internal",
					"kubernetes.io/os":                              "linux",
					"node.kubernetes.io/instance-type":              "m5zn.large",
					"topology.kubernetes.io/region":                 "eu-central-1",
					"topology.kubernetes.io/zone":                   "eu-central-1b",
				},
			},
			Spec: corev1.NodeSpec{
				ProviderID: "aws:///eu-central-1b/i-0178150be4c94393d",
			},
		},
	}

	p.EXPECT().Runtime().Return("k8s-cluster")
	p.EXPECT().Nodes().Return(nodes, nil)

	clusterIdentifier := "//platformid.api.mondoo.app/runtime/k8s/uid/e26043bb-8669-48a2-b684-b1e132198cdc"

	pCfg := &providers.Config{}
	assets, relInfo, err := ListNodes(p, pCfg, clusterIdentifier)
	require.NoError(t, err)
	require.Len(t, assets, 1)
	require.Equal(t, "Kubernetes Node", assets[0].Platform.Title)
	require.Equal(t, "k8s-node", assets[0].Platform.Name)
	require.Equal(t, providers.Kind_KIND_K8S_OBJECT, assets[0].Platform.Kind)
	require.ElementsMatch(t, []string{"k8s"}, assets[0].Platform.Family)
	require.Equal(t, []string{"//platformid.api.mondoo.app/runtime/k8s/uid/e26043bb-8669-48a2-b684-b1e132198cdc/nodes/name/ip-10-0-5-36.eu-central-1.compute.internal"}, assets[0].PlatformIds)

	require.Len(t, assets[0].RelatedAssets, 0)

	require.Nil(t, relInfo[0].hostInstanceAsset)
	require.Nil(t, relInfo[0].cloudAccountAsset)
}

func TestListNodesK3S(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	p := k8s.NewMockKubernetesProvider(mockCtrl)

	nodes := []corev1.Node{
		{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Node",
				APIVersion: "v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: "x1",
				UID:  "08677417-062a-4521-af10-901913b575cf",
				Annotations: map[string]string{
					"k3s.io/hostname":              "x1",
					"k3s.io/internal-ip":           "192.168.1.87",
					"k3s.io/node-args":             `'["server","--write-kubeconfig-mode","0644"]'`,
					"k3s.io/node-config-hash":      "LUZJBAJBVUEWLANIK5CQFBP3IKZUSSX643EDQVRRLL4O4D6AVNLQ====",
					"k3s.io/node-env":              `'{"K3S_DATA_DIR":"/var/lib/rancher/k3s/data/577968fa3d58539cc4265245941b7be688833e6bf5ad7869fa2afe02f15f1cd2"}'`,
					"node.alpha.kubernetes.io/ttl": "0",
					"volumes.kubernetes.io/controller-managed-attach-detach": "true",
				},
				Labels: map[string]string{
					"beta.kubernetes.io/arch":               "amd64",
					"beta.kubernetes.io/instance-type":      "k3s",
					"beta.kubernetes.io/os":                 "linux",
					"egress.k3s.io/cluster":                 "true",
					"kubernetes.io/arch":                    "amd64",
					"kubernetes.io/hostname":                "x1",
					"kubernetes.io/os":                      "linux",
					"node-role.kubernetes.io/control-plane": "true",
					"node-role.kubernetes.io/master":        "true",
					"node.kubernetes.io/instance-type":      "k3s",
				},
			},
			Spec: corev1.NodeSpec{
				ProviderID: "k3s://x1",
				PodCIDR:    "10.42.0.0/24",
				PodCIDRs:   []string{"10.42.0.0/24"},
			},
		},
	}

	p.EXPECT().Runtime().Return("k8s-cluster")
	p.EXPECT().Nodes().Return(nodes, nil)

	clusterIdentifier := "//platformid.api.mondoo.app/runtime/k8s/uid/e26043bb-8669-48a2-b684-b1e132198cdc"

	pCfg := &providers.Config{}
	assets, relInfo, err := ListNodes(p, pCfg, clusterIdentifier)
	require.NoError(t, err)
	require.Len(t, assets, 1)
	require.Equal(t, "Kubernetes Node", assets[0].Platform.Title)
	require.Equal(t, "k8s-node", assets[0].Platform.Name)
	require.Equal(t, providers.Kind_KIND_K8S_OBJECT, assets[0].Platform.Kind)
	require.ElementsMatch(t, []string{"k8s"}, assets[0].Platform.Family)
	require.Equal(t, []string{"//platformid.api.mondoo.app/runtime/k8s/uid/e26043bb-8669-48a2-b684-b1e132198cdc/nodes/name/x1"}, assets[0].PlatformIds)

	// Adds relatonship to host
	require.Len(t, assets[0].RelatedAssets, 1)
	require.Equal(t, "x1", assets[0].RelatedAssets[0].Name)
	require.Equal(t, providers.Kind_KIND_UNKNOWN, assets[0].RelatedAssets[0].GetPlatform().GetKind())
	require.Equal(t, "", assets[0].RelatedAssets[0].GetPlatform().GetRuntime())
	require.Equal(t, []string{"//platformid.api.mondoo.app/hostname/x1"}, assets[0].RelatedAssets[0].PlatformIds)

	require.NotNil(t, relInfo[0].hostInstanceAsset)
	require.Equal(t, assets[0].RelatedAssets[0], relInfo[0].hostInstanceAsset)
	require.Nil(t, relInfo[0].cloudAccountAsset)
}
