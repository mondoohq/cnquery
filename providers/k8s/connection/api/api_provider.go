package api

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/providers/k8s/connection/shared"
	"go.mondoo.com/cnquery/providers/k8s/connection/shared/resources"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/version"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	"k8s.io/client-go/util/homedir"
)

const (
	Api shared.ConnectionType = "api"
)

type ApiConnection struct {
	runtime            string
	id                 uint32
	asset              *inventory.Asset
	d                  *resources.Discovery
	config             *rest.Config
	namespace          string
	clientset          *kubernetes.Clientset
	currentClusterName string
}

func NewConnection(id uint32, asset *inventory.Asset) (shared.Connection, error) {
	// check if the user .kube/config file exists
	// NOTE: BuildConfigFromFlags falls back to cluster loading when .kube/config string is empty
	// therefore we want to only change the kubeconfig string when the file really exists
	var kubeconfigPath string

	// use KUBECONFIG as default
	// https://kubernetes.io/docs/tasks/access-application-cluster/configure-access-multiple-clusters/#set-the-kubeconfig-environment-variable
	kubeconfigPath = os.Getenv("KUBECONFIG")

	// if no config is set, try to load the default kubeconfig path if nothing was provided
	if kubeconfigPath == "" {
		if home := homedir.HomeDir(); home != "" {
			kubeconfigPathHome := filepath.Join(home, ".kube", "config")
			if _, err := os.Stat(kubeconfigPathHome); err == nil {
				kubeconfigPath = kubeconfigPathHome
			}
		}
	}

	config, err := buildConfigFromFlags("", kubeconfigPath, "")
	if err != nil {
		return nil, err
	}

	kubeConfig, err := (&clientcmd.ClientConfigLoadingRules{ExplicitPath: kubeconfigPath}).Load()
	if err != nil {
		return nil, err
	}

	// enable-client side throttling
	// avoids the cli warning: Waited for 1.000907542s due to client-side throttling, not priority and fairness
	config.QPS = 1000
	config.Burst = 1000

	// initialize api
	d, err := resources.NewDiscoveryCache().Get(config)
	if err != nil {
		return nil, err
	}
	log.Debug().Msg("loaded kubeconfig successfully")

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, errors.Wrap(err, "could not create kubernetes clientset")
	}

	res := ApiConnection{
		id:                 id,
		asset:              asset,
		d:                  d,
		config:             config,
		clientset:          clientset,
		currentClusterName: kubeConfig.Contexts[kubeConfig.CurrentContext].Cluster,
	}

	return &res, nil
}

// buildConfigFromFlags we rebuild clientcmd.BuildConfigFromFlags to make sure we do not log warnings for every
// scan.
func buildConfigFromFlags(masterUrl, kubeconfigPath string, context string) (*restclient.Config, error) {
	if kubeconfigPath == "" && masterUrl == "" {
		kubeconfig, err := restclient.InClusterConfig()
		if err == nil {
			return kubeconfig, nil
		}
	}
	return clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		&clientcmd.ClientConfigLoadingRules{ExplicitPath: kubeconfigPath},
		&clientcmd.ConfigOverrides{ClusterInfo: clientcmdapi.Cluster{Server: masterUrl}, CurrentContext: context}).ClientConfig()
}

func (p *ApiConnection) ID() uint32 {
	return p.id
}

func (p *ApiConnection) ClusterName() (string, error) {
	ctx := context.Background()

	// right now we use the name of the first node to identify the cluster
	result, err := p.clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return "", err
	}

	if len(result.Items) > 0 {
		node := result.Items[0]
		return node.GetName(), nil
	}

	return "", fmt.Errorf("cannot determine cluster name")
}

func (p *ApiConnection) Name() string {
	opts := p.asset.Connections[0].Options

	var clusterName string
	// the name is still a bit unreliable
	// see https://github.com/kubernetes/kubernetes/issues/44954
	if len(opts["context"]) > 0 {
		clusterName = opts["context"]
		log.Info().Str("cluster-name", clusterName).Msg("use cluster name from --context")
	} else {
		clusterName = ""

		// try to parse context from kubectl config
		if clusterName == "" && len(p.currentClusterName) > 0 {
			clusterName = p.currentClusterName
		}

		// fallback to first node name if we could not gather the name from kubeconfig
		if clusterName == "" {
			name, err := p.ClusterName()
			if err == nil {
				clusterName = name
				log.Info().Str("cluster-name", clusterName).Msg("use cluster name from node name")
			}
		}

		clusterName = "K8s Cluster " + clusterName
	}
	return clusterName
}

func (p *ApiConnection) Type() shared.ConnectionType {
	return Api
}

func (p *ApiConnection) Asset() *inventory.Asset {
	return p.asset
}

func (t *ApiConnection) ServerVersion() *version.Info {
	return t.d.ServerVersion
}

func (t *ApiConnection) SupportedResourceTypes() (*resources.ApiResourceIndex, error) {
	return t.d.SupportedResourceTypes()
}

func (t *ApiConnection) Resources(kind string, name string, namespace string) (*shared.ResourceResult, error) {
	ctx := context.Background()
	allNs := false
	if len(namespace) == 0 {
		allNs = true
	}

	// discover api and resources that have a list method
	resTypes, err := t.d.SupportedResourceTypes()
	if err != nil {
		return nil, err
	}
	log.Debug().Msg("completed querying resource types")

	resType, err := resTypes.Lookup(kind)
	if err != nil {
		return nil, err
	}

	log.Debug().Msgf("fetch all %s resources", kind)
	objs, err := t.d.GetKindResources(ctx, *resType, namespace, allNs)
	if err != nil {
		return nil, err
	}
	log.Debug().Msgf("found %d resource objects", len(objs))

	objs, err = resources.FilterResource(resType, objs, name, namespace)
	if err != nil {
		return nil, err
	}

	return &shared.ResourceResult{
		Name:         name,
		Kind:         kind,
		ResourceType: resType,
		Resources:    objs,
		Namespace:    namespace,
		AllNs:        allNs,
	}, err
}

func (t *ApiConnection) Nodes() ([]v1.Node, error) {
	ctx := context.Background()
	list, err := t.clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	// needed because of https://github.com/kubernetes/client-go/issues/861
	for i := range list.Items {
		list.Items[i].SetGroupVersionKind(v1.SchemeGroupVersion.WithKind("Node"))
	}
	return list.Items, err
}
