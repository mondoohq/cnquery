package k8s

import (
	"github.com/rs/zerolog/log"
	"go.mondoo.io/mondoo/lumi/resources/kubectl"
	"go.mondoo.io/mondoo/motor"
	"go.mondoo.io/mondoo/motor/asset"
	"go.mondoo.io/mondoo/motor/discovery/credentials"
	"go.mondoo.io/mondoo/motor/platform/detector"
	"go.mondoo.io/mondoo/motor/providers"
	k8s_transport "go.mondoo.io/mondoo/motor/providers/k8s"
	"go.mondoo.io/mondoo/motor/providers/local"
)

const (
	DiscoveryAll             = "all"
	DiscoveryPods            = "pods"
	DiscoveryCronJobs        = "cronjobs"
	DiscoveryStatefulSets    = "statefulsets"
	DiscoveryContainerImages = "container-images"
)

type Resolver struct{}

func (r *Resolver) Name() string {
	return "Kubernetes Resolver"
}

func (r *Resolver) AvailableDiscoveryTargets() []string {
	return []string{
		DiscoveryAll,
		DiscoveryPods,
		DiscoveryCronJobs,
		DiscoveryStatefulSets,
		DiscoveryContainerImages,
	}
}

func (r *Resolver) Resolve(tc *providers.TransportConfig, cfn credentials.CredentialFn, sfn credentials.QuerySecretFn, userIdDetectors ...providers.PlatformIdDetector) ([]*asset.Asset, error) {
	resolved := []*asset.Asset{}
	namespacesFilter := []string{}

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

	allNamespaces := tc.Options["all-namespaces"]
	if allNamespaces != "true" {
		namespace := tc.Options["namespace"]
		if len(namespace) > 0 {
			namespacesFilter = append(namespacesFilter, namespace)
		} else {
			// try parse the current kubectl namespace
			if k8sctlConfig != nil && len(k8sctlConfig.CurrentNamespace()) > 0 {
				namespacesFilter = append(namespacesFilter, k8sctlConfig.CurrentNamespace())
			}
		}
	}

	log.Debug().Strs("namespaceFilter", namespacesFilter).Msg("resolve k8s assets")

	trans, err := k8s_transport.New(tc)
	if err != nil {
		return nil, err
	}

	clusterIdentifier, err := trans.Identifier()
	if err != nil {
		return nil, err
	}

	// detect platform info for the asset
	detector := detector.New(trans)
	pf, err := detector.Platform()
	if err != nil {
		return nil, err
	}

	// the name is still a bit unreliable
	// see https://github.com/kubernetes/kubernetes/issues/44954
	clusterName := ""

	if tc.Options["path"] != "" {
		clusterName, _ = trans.Name()
	} else {
		// try to parse context from kubectl config
		if clusterName == "" && k8sctlConfig != nil && len(k8sctlConfig.CurrentContext) > 0 {
			clusterName = k8sctlConfig.CurrentClusterName()
			log.Info().Str("cluster-name", clusterName).Msg("use cluster name from kube config")
		}

		// fallback to first node name if we could not gather the name from kubeconfig
		if clusterName == "" {
			name, err := trans.Name()
			if err == nil {
				clusterName = name
				log.Info().Str("cluster-name", clusterName).Msg("use cluster name from node name")
			}
		}

		clusterName = "K8S Cluster " + clusterName
		ns, ok := tc.Options[k8s_transport.OPTION_NAMESPACE]
		if ok && ns != "" {
			clusterName += " (Namespace: " + ns + ")"
		}
	}

	resolved = append(resolved, &asset.Asset{
		PlatformIds: []string{clusterIdentifier},
		Name:        clusterName,
		Platform:    pf,
		Connections: []*providers.TransportConfig{tc}, // pass-in the current config
		State:       asset.State_STATE_RUNNING,
	})

	additioanlAssets, err := addSeparateAssets(tc, trans, namespacesFilter, clusterIdentifier)
	if err != nil {
		return nil, err
	}
	resolved = append(resolved, additioanlAssets...)

	return resolved, nil
}

// addSeparateAssets Depending on config options it will search for additional assets which should be listed separately.
func addSeparateAssets(tc *providers.TransportConfig, transport k8s_transport.Transport, namespacesFilter []string, clusterIdentifier string) ([]*asset.Asset, error) {
	var resolved []*asset.Asset

	// discover k8s pods
	if tc.IncludesDiscoveryTarget(DiscoveryAll) || tc.IncludesDiscoveryTarget(DiscoveryPods) {
		// fetch pod information
		log.Debug().Strs("namespace", namespacesFilter).Msg("search for pods")
		connection := tc.Clone()
		assetList, err := ListPods(transport, connection, clusterIdentifier, namespacesFilter)
		if err != nil {
			log.Error().Err(err).Msg("could not fetch k8s pods")
			return nil, err
		}
		resolved = append(resolved, assetList...)
	}

	// discover k8s pod images
	if tc.IncludesDiscoveryTarget(DiscoveryAll) || tc.IncludesDiscoveryTarget(DiscoveryContainerImages) {
		// fetch pod information
		log.Debug().Strs("namespace", namespacesFilter).Msg("search for pods images")
		assetList, err := ListPodImages(transport, namespacesFilter)
		if err != nil {
			log.Error().Err(err).Msg("could not fetch k8s pods images")
			return nil, err
		}
		resolved = append(resolved, assetList...)
	}

	// discover cronjobs
	if tc.IncludesDiscoveryTarget(DiscoveryAll) || tc.IncludesDiscoveryTarget(DiscoveryCronJobs) {
		log.Debug().Strs("namespace", namespacesFilter).Msg("search for cronjobs")
		connection := tc.Clone()
		assetList, err := ListCronJobs(transport, connection, clusterIdentifier, namespacesFilter)
		if err != nil {
			log.Error().Err(err).Msg("could not fetch k8s cronjobs")
			return nil, err
		}
		resolved = append(resolved, assetList...)
	}

	// discover statefulsets
	if tc.IncludesDiscoveryTarget(DiscoveryAll) || tc.IncludesDiscoveryTarget(DiscoveryStatefulSets) {
		log.Debug().Strs("namespace", namespacesFilter).Msg("search for statefulsets")
		connection := tc.Clone()
		assetList, err := ListStatefulSets(transport, connection, clusterIdentifier, namespacesFilter)
		if err != nil {
			log.Error().Err(err).Msg("could not fetch k8s statefulsets")
			return nil, err
		}
		resolved = append(resolved, assetList...)
	}

	return resolved, nil
}
