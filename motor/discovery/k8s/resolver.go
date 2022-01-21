package k8s

import (
	"github.com/rs/zerolog/log"
	"go.mondoo.io/mondoo/lumi/resources/kubectl"
	"go.mondoo.io/mondoo/motor"
	"go.mondoo.io/mondoo/motor/asset"
	"go.mondoo.io/mondoo/motor/discovery/common"
	"go.mondoo.io/mondoo/motor/platform"
	"go.mondoo.io/mondoo/motor/transports"
	k8s_transport "go.mondoo.io/mondoo/motor/transports/k8s"
	"go.mondoo.io/mondoo/motor/transports/local"
)

const (
	DiscoveryAll             = "all"
	DiscoveryContainerImages = "container-images"
)

type Resolver struct{}

func (r *Resolver) Name() string {
	return "Kubernetes Resolver"
}

func (r *Resolver) AvailableDiscoveryTargets() []string {
	return []string{DiscoveryAll, DiscoveryContainerImages}
}

func (r *Resolver) Resolve(tc *transports.TransportConfig, cfn common.CredentialFn, sfn common.QuerySecretFn, userIdDetectors ...transports.PlatformIdDetector) ([]*asset.Asset, error) {
	resolved := []*asset.Asset{}
	namespacesFilter := []string{}
	podFilter := []string{}

	var k8sctlConfig *kubectl.KubectlConfig
	localTransport, err := local.New()
	if err == nil {
		m, err := motor.New(localTransport)
		if err == nil {
			k8sctlConfig, err = kubectl.LoadKubeConfig(m)
			if err != nil {
				return nil, err
			}
		}
	}

	k8sContext := tc.Options["context"]
	if len(k8sContext) == 0 {
		// try to parse context from kubectl
		if k8sctlConfig != nil && len(k8sctlConfig.CurrentContext) > 0 {
			k8sContext = k8sctlConfig.CurrentContext
		}
	}

	namespace := tc.Options["namespace"]
	if len(namespace) > 0 {
		namespacesFilter = append(namespacesFilter, namespace)
	} else {
		// try parse the current kubectl namespace
		if k8sctlConfig != nil && len(k8sctlConfig.CurrentNamespace()) > 0 {
			namespacesFilter = append(namespacesFilter, k8sctlConfig.CurrentNamespace())
		}
	}

	pod := tc.Options["pod"]
	if len(pod) > 0 {
		podFilter = append(podFilter, pod)
	}

	log.Debug().Strs("podFilter", podFilter).Strs("namespaceFilter", namespacesFilter).Msg("resolve k8s assets")

	// add k8s api
	// add aws api as asset
	trans, err := k8s_transport.New(tc)
	// trans, err := aws_transport.New(t, transportOpts...)
	if err != nil {
		return nil, err
	}

	identifier, err := trans.Identifier()
	if err != nil {
		return nil, err
	}

	// detect platform info for the asset
	detector := platform.NewDetector(trans)
	pf, err := detector.Platform()
	if err != nil {
		return nil, err
	}

	// the name is still a bit unreliable
	// see https://github.com/kubernetes/kubernetes/issues/44954
	clusterName := ""

	if tc.Options["path"] != "" {
		// manifest parent directory name
		clusterName = common.ProjectNameFromPath(tc.Options["path"])
		clusterName = "K8S Manifest " + clusterName
	} else {
		// try to parse context from kubectl config
		if clusterName == "" && k8sctlConfig != nil && len(k8sctlConfig.CurrentContext) > 0 {
			clusterName = k8sctlConfig.CurrentClusterName()
			log.Info().Str("cluster-name", clusterName).Msg("use cluster name from kube config")
		}

		// fallback to first node name if we could not gather the name from kubeconfig
		if clusterName == "" {
			ci, err := trans.ClusterInfo()
			if err == nil {
				clusterName = ci.Name
				log.Info().Str("cluster-name", clusterName).Msg("use cluster name from node name")
			}
		}

		clusterName = "K8S Cluster " + clusterName
	}

	resolved = append(resolved, &asset.Asset{
		PlatformIds: []string{identifier},
		Name:        clusterName,
		Platform:    pf,
		Connections: []*transports.TransportConfig{tc}, // pass-in the current config
	})

	// discover ec2 instances
	if tc.IncludesDiscoveryTarget(DiscoveryAll) || tc.IncludesDiscoveryTarget(DiscoveryContainerImages) {
		// fetch pod informaton
		log.Debug().Str("context", k8sContext).Strs("namespace", namespacesFilter).Strs("namespace", podFilter).Msg("search for pods")
		assetList, err := ListPodImages(k8sContext, namespacesFilter, podFilter)
		if err != nil {
			log.Error().Err(err).Msg("could not fetch k8s images")
			return nil, err
		}

		for i := range assetList {
			log.Debug().Str("name", assetList[i].Name).Str("image", assetList[i].Connections[0].Host).Msg("resolved pod")
			resolved = append(resolved, assetList[i])
		}
	}
	return resolved, nil
}
