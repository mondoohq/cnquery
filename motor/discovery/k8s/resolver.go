package k8s

import (
	"context"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/motor/asset"
	"go.mondoo.com/cnquery/motor/discovery/common"
	"go.mondoo.com/cnquery/motor/platform/detector"
	"go.mondoo.com/cnquery/motor/providers"
	"go.mondoo.com/cnquery/motor/providers/k8s"
	"go.mondoo.com/cnquery/motor/providers/k8s/resources"
	"go.mondoo.com/cnquery/motor/providers/local"
	"go.mondoo.com/cnquery/resources/packs/os/kubectl"
)

var _ common.ContextInitializer = (*Resolver)(nil)

const (
	DiscoveryAll             = "all"
	DiscoveryPods            = "pods"
	DiscoveryJobs            = "jobs"
	DiscoveryCronJobs        = "cronjobs"
	DiscoveryStatefulSets    = "statefulsets"
	DiscoveryDeployments     = "deployments"
	DiscoveryReplicaSets     = "replicasets"
	DiscoveryDaemonSets      = "daemonsets"
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
		DiscoveryJobs,
		DiscoveryCronJobs,
		DiscoveryStatefulSets,
		DiscoveryDeployments,
		DiscoveryReplicaSets,
		DiscoveryDaemonSets,
		DiscoveryContainerImages,
	}
}

func (r *Resolver) Resolve(ctx context.Context, root *asset.Asset, tc *providers.Config, cfn common.CredentialFn, sfn common.QuerySecretFn, userIdDetectors ...providers.PlatformIdDetector) ([]*asset.Asset, error) {
	resolved := []*asset.Asset{}
	namespacesFilter := []string{}

	var k8sctlConfig *kubectl.KubectlConfig
	localProvider, err := local.New()
	if err == nil {
		k8sctlConfig, err = kubectl.LoadKubeConfig(localProvider)
		if err != nil {
			return nil, err
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

	p, err := k8s.New(ctx, tc)
	if err != nil {
		return nil, err
	}

	clusterIdentifier, err := p.Identifier()
	if err != nil {
		return nil, err
	}

	// detect platform info for the asset
	detector := detector.New(p)
	pf, err := detector.Platform()
	if err != nil {
		return nil, err
	}

	// the name is still a bit unreliable
	// see https://github.com/kubernetes/kubernetes/issues/44954
	clusterName := ""

	if tc.Options["path"] != "" {
		clusterName, _ = p.Name()
	} else {
		// try to parse context from kubectl config
		if clusterName == "" && k8sctlConfig != nil && len(k8sctlConfig.CurrentContext) > 0 {
			clusterName = k8sctlConfig.CurrentClusterName()
			log.Info().Str("cluster-name", clusterName).Msg("use cluster name from kube config")
		}

		// fallback to first node name if we could not gather the name from kubeconfig
		if clusterName == "" {
			name, err := p.Name()
			if err == nil {
				clusterName = name
				log.Info().Str("cluster-name", clusterName).Msg("use cluster name from node name")
			}
		}

		clusterName = "K8S Cluster " + clusterName
		ns, ok := tc.Options[k8s.OPTION_NAMESPACE]
		if ok && ns != "" {
			clusterName += " (Namespace: " + ns + ")"
		}
	}

	resolved = append(resolved, &asset.Asset{
		PlatformIds: []string{clusterIdentifier},
		Name:        clusterName,
		Platform:    pf,
		Connections: []*providers.Config{tc}, // pass-in the current config
		State:       asset.State_STATE_RUNNING,
	})

	additioanlAssets, err := addSeparateAssets(tc, p, namespacesFilter, clusterIdentifier)
	if err != nil {
		return nil, err
	}
	resolved = append(resolved, additioanlAssets...)

	return resolved, nil
}

func (r *Resolver) InitCtx(ctx context.Context) context.Context {
	return resources.SetDiscoveryCache(ctx, resources.NewDiscoveryCache())
}

// addSeparateAssets Depending on config options it will search for additional assets which should be listed separately.
func addSeparateAssets(tc *providers.Config, p k8s.KubernetesProvider, namespacesFilter []string, clusterIdentifier string) ([]*asset.Asset, error) {
	var resolved []*asset.Asset

	// discover deployments
	if tc.IncludesDiscoveryTarget(DiscoveryAll) || tc.IncludesDiscoveryTarget(DiscoveryDeployments) {
		// fetch deployment information
		log.Debug().Strs("namespace", namespacesFilter).Msg("search for deployments")
		connection := tc.Clone()
		assetList, err := ListDeployments(p, connection, clusterIdentifier, namespacesFilter)
		if err != nil {
			log.Error().Err(err).Msg("could not fetch k8s deployments")
			return nil, err
		}
		resolved = append(resolved, assetList...)
	}

	// discover k8s pods
	if tc.IncludesDiscoveryTarget(DiscoveryAll) || tc.IncludesDiscoveryTarget(DiscoveryPods) {
		// fetch pod information
		log.Debug().Strs("namespace", namespacesFilter).Msg("search for pods")
		connection := tc.Clone()
		assetList, err := ListPods(p, connection, clusterIdentifier, namespacesFilter)
		if err != nil {
			log.Error().Err(err).Msg("could not fetch k8s pods")
			return nil, err
		}
		resolved = append(resolved, assetList...)
	}

	// discovery k8s daemonsets
	if tc.IncludesDiscoveryTarget(DiscoveryAll) || tc.IncludesDiscoveryTarget(DiscoveryDaemonSets) {
		log.Debug().Strs("namespace", namespacesFilter).Msg("search for daemonsets")
		connection := tc.Clone()
		assetList, err := ListDaemonSets(p, connection, clusterIdentifier, namespacesFilter)
		if err != nil {
			log.Error().Err(err).Msg("could not fetch k8s daemonsets")
			return nil, err
		}
		resolved = append(resolved, assetList...)
	}

	// discover k8s pod images
	if tc.IncludesDiscoveryTarget(DiscoveryAll) || tc.IncludesDiscoveryTarget(DiscoveryContainerImages) {
		// fetch pod information
		log.Debug().Strs("namespace", namespacesFilter).Msg("search for pods images")
		assetList, err := ListPodImages(p, namespacesFilter)
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
		assetList, err := ListCronJobs(p, connection, clusterIdentifier, namespacesFilter)
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
		assetList, err := ListStatefulSets(p, connection, clusterIdentifier, namespacesFilter)
		if err != nil {
			log.Error().Err(err).Msg("could not fetch k8s statefulsets")
			return nil, err
		}
		resolved = append(resolved, assetList...)
	}

	// discover jobs
	if tc.IncludesDiscoveryTarget(DiscoveryAll) || tc.IncludesDiscoveryTarget(DiscoveryJobs) {
		log.Debug().Strs("namespace", namespacesFilter).Msg("search for jobs")
		connection := tc.Clone()
		assetList, err := ListJobs(p, connection, clusterIdentifier, namespacesFilter)
		if err != nil {
			log.Error().Err(err).Msg("could not fetch k8s jobs")
			return nil, err
		}
		resolved = append(resolved, assetList...)
	}

	// discover replicasets
	if tc.IncludesDiscoveryTarget(DiscoveryAll) || tc.IncludesDiscoveryTarget(DiscoveryReplicaSets) {
		log.Debug().Strs("namespace", namespacesFilter).Msg("search for replicasets")
		connection := tc.Clone()
		assetList, err := ListReplicaSets(p, connection, clusterIdentifier, namespacesFilter)
		if err != nil {
			log.Error().Err(err).Msg("could not fetch k8s replicasets")
			return nil, err
		}
		resolved = append(resolved, assetList...)
	}

	return resolved, nil
}
