package k8s

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"go.mondoo.com/cnquery/motor/asset"
	"go.mondoo.com/cnquery/motor/motorid/gce"
	"go.mondoo.com/cnquery/motor/platform"
	"go.mondoo.com/cnquery/motor/providers"
	"go.mondoo.com/cnquery/motor/providers/k8s"
	v1 "k8s.io/api/core/v1"
)

// ListNodes lits all nodes in the cluster.
func ListNodes(p k8s.KubernetesProvider, connection *providers.Config, clusterIdentifier string) ([]*asset.Asset, []nodeRelationshipInfo, error) {
	nodes, err := p.Nodes()
	if err != nil {
		return nil, nil, err
	}

	assets := []*asset.Asset{}
	nodeRelationshipInfos := []nodeRelationshipInfo{}
	for i := range nodes {
		node := nodes[i]
		asset, err := createAssetFromObject(&node, p.Runtime(), connection, clusterIdentifier)
		if err != nil {
			return nil, nil, errors.Wrap(err, "failed to create asset from node")
		}

		assets = append(assets, asset)
		nInfo, _ := detectNodeRelationshipInfo(node)
		if nInfo.hostInstanceAsset != nil {
			asset.RelatedAssets = append(asset.RelatedAssets, nInfo.hostInstanceAsset)
		}
		nodeRelationshipInfos = append(nodeRelationshipInfos, nInfo)
	}

	return assets, nodeRelationshipInfos, nil
}

type nodeRelationshipInfo struct {
	cloudAccountAsset *asset.Asset
	hostInstanceAsset *asset.Asset
}

var (
	gkeProviderIDInfoRegexp     = regexp.MustCompile("^gce://([\\-0-9a-zA-Z]+)/([\\-0-9a-zA-Z]+)/.*")
	aksProviderIDInstanceRegexp = regexp.MustCompile("^azure:///(.+)$")
)

func detectNodeRelationshipInfo(node v1.Node) (nodeRelationshipInfo, bool) {
	for k := range node.Labels {
		if strings.HasPrefix(k, "eks.amazonaws.com") {
			// The node info doesn't seem to have the AWS Account id
			return nodeRelationshipInfo{}, false
		} else if strings.HasPrefix(k, "cloud.google.com/gke") {
			return gkeRelationshipInfo(node)
		} else if strings.HasPrefix(k, "kubernetes.azure.com") {
			return aksRelationshipInfo(node)
		}
	}
	hostname := node.Labels["kubernetes.io/hostname"]
	if hostname == "" {
		return nodeRelationshipInfo{}, false
	}
	return nodeRelationshipInfo{
		hostInstanceAsset: &asset.Asset{
			Name:        hostname,
			PlatformIds: []string{"//platformid.api.mondoo.app/hostname/" + hostname},
		},
	}, true
}

func gkeRelationshipInfo(node v1.Node) (nodeRelationshipInfo, bool) {
	matches := gkeProviderIDInfoRegexp.FindStringSubmatch(node.Spec.ProviderID)
	if len(matches) != 3 {
		return nodeRelationshipInfo{}, false
	}
	project := matches[1]
	zone := matches[2]
	instanceID := node.Annotations["container.googleapis.com/instance_id"]
	instanceIDInt, err := strconv.ParseUint(instanceID, 10, 64)
	if err != nil {
		return nodeRelationshipInfo{}, false
	}
	if project != "" && zone != "" && instanceID != "" {
		cloudAccountAsset := &asset.Asset{
			Name: "GCP project " + project,
			Platform: &platform.Platform{
				Kind:    providers.Kind_KIND_API,
				Runtime: providers.RUNTIME_GCP,
				Title:   "Google Cloud Platform",
			},
			PlatformIds: []string{"//platformid.api.mondoo.app/runtime/gcp/projects/" + project},
		}
		hostInstanceAsset := &asset.Asset{
			Name: node.Labels["kubernetes.io/hostname"],
			Platform: &platform.Platform{
				Kind:    providers.Kind_KIND_VIRTUAL_MACHINE,
				Runtime: providers.RUNTIME_GCP_COMPUTE,
				Arch:    node.Labels["kubernetes.io/arch"],
			},
			PlatformIds:   []string{gce.MondooGcpInstanceID(project, zone, instanceIDInt)},
			RelatedAssets: []*asset.Asset{cloudAccountAsset},
		}
		return nodeRelationshipInfo{
			cloudAccountAsset: cloudAccountAsset,
			hostInstanceAsset: hostInstanceAsset,
		}, true
	}
	return nodeRelationshipInfo{}, false
}

func aksRelationshipInfo(node v1.Node) (nodeRelationshipInfo, bool) {
	matches := aksProviderIDInstanceRegexp.FindStringSubmatch(node.Spec.ProviderID)
	if len(matches) != 2 {
		return nodeRelationshipInfo{}, false
	}
	parts := strings.Split(matches[1], "/")
	if len(parts) < 2 || parts[0] != "subscriptions" {
		return nodeRelationshipInfo{}, false
	}
	sub := parts[1]
	cloudAccountAsset := &asset.Asset{
		Name: "Azure subscription " + sub,
		Platform: &platform.Platform{
			Kind:    providers.Kind_KIND_API,
			Runtime: providers.RUNTIME_AZ,
		},
		PlatformIds: []string{"//platformid.api.mondoo.app/runtime/azure/subscriptions/" + sub},
	}
	hostInstanceAsset := &asset.Asset{
		Name: node.Labels["kubernetes.io/hostname"],
		Platform: &platform.Platform{
			Kind:    providers.Kind_KIND_VIRTUAL_MACHINE,
			Runtime: providers.RUNTIME_AZ_COMPUTE,
			Arch:    node.Labels["kubernetes.io/arch"],
		},
		PlatformIds:   []string{"//platformid.api.mondoo.app/runtime/azure/" + matches[1]},
		RelatedAssets: []*asset.Asset{cloudAccountAsset},
	}
	return nodeRelationshipInfo{
		cloudAccountAsset: cloudAccountAsset,
		hostInstanceAsset: hostInstanceAsset,
	}, true
}
