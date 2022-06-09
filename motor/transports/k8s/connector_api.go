package k8s

import (
	"context"
	"os"
	"path/filepath"

	"github.com/cockroachdb/errors"
	"github.com/gosimple/slug"
	"github.com/rs/zerolog/log"
	"go.mondoo.io/mondoo/motor/platform"
	"go.mondoo.io/mondoo/motor/transports"
	"go.mondoo.io/mondoo/motor/transports/k8s/resources"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/version"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

// KubeConfig
// - $HOME/.kube/config
// Service Account
// - /var/run/secrets/kubernetes.io/serviceaccount/token
// - /var/run/secrets/kubernetes.io/serviceaccount/ca.crt
func NewApiConnector(namespace string) (*ApiConnector, error) {
	// check if the user .kube/config file exists
	// NOTE: BuildConfigFromFlags falls back to cluster loading when .kube/config string is empty
	// therefore we want to only change the kubeconfig string when the file really exists
	var kubeconfig string

	// use KUBECONFIG as default
	// https://kubernetes.io/docs/tasks/access-application-cluster/configure-access-multiple-clusters/#set-the-kubeconfig-environment-variable
	kubeconfig = os.Getenv("KUBECONFIG")

	// if no config is set, try to load the default kubeconfig path if nothing was provided
	if kubeconfig == "" {
		if home := homedir.HomeDir(); home != "" {
			kubeconfigpath := filepath.Join(home, ".kube", "config")
			if _, err := os.Stat(kubeconfigpath); err == nil {
				kubeconfig = kubeconfigpath
			}
		}
	}

	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return nil, err
	}

	// enable-client side throttling
	// avoids the cli warning: Waited for 1.000907542s due to client-side throttling, not priority and fairness
	config.QPS = 1000
	config.Burst = 1000

	// initialize api client
	d, err := resources.NewDiscovery(config)
	if err != nil {
		return nil, err
	}
	log.Debug().Msg("loaded kubeconfig successfully")

	return &ApiConnector{
		namespace: namespace,
		config:    config,
		d:         d,
	}, nil
}

type ApiConnector struct {
	d         *resources.Discovery
	config    *rest.Config
	namespace string
}

func (ac *ApiConnector) Identifier() (string, error) {
	// we use "kube-system" namespace uid as identifier for the cluster
	result, err := ac.Resources("namespaces", "kube-system")
	if err != nil {
		return "", err
	}

	if len(result.RootResources) != 1 {
		return "", errors.New("could not identify the k8s cluster")
	}

	resource := result.RootResources[0]
	obj, err := meta.Accessor(resource)
	if err != nil {
		return "", err
	}

	uid := string(obj.GetUID())
	id := "//platformid.api.mondoo.app/runtime/k8s/uid/" + uid

	if ac.namespace != "" {
		id += "/namespace/" + slug.Make(ac.namespace)
	}

	return id, nil
}

func (ac *ApiConnector) Name() (string, error) {
	ci, err := ac.ClusterInfo()
	if err != nil {
		return "", err
	}
	return ci.Name, nil
}

type ClusterInfo struct {
	Name string
}

func (ac *ApiConnector) ClusterInfo() (ClusterInfo, error) {
	res := ClusterInfo{}

	// right now we use the name of the first node to identify the cluster
	result, err := ac.Resources("nodes.v1.", "")
	if err != nil {
		return res, err
	}

	if len(result.RootResources) > 0 {
		node := result.RootResources[0]
		obj, err := meta.Accessor(node)
		if err != nil {
			log.Error().Err(err).Msg("could not access object attributes")
			return res, err
		}
		res.Name = obj.GetName()
	}

	return res, nil
}

func (ac *ApiConnector) ServerVersion() *version.Info {
	return ac.d.ServerVersion
}

func (ac *ApiConnector) SupportedResourceTypes() (*resources.ApiResourceIndex, error) {
	return ac.d.SupportedResourceTypes()
}

func (ac *ApiConnector) Resources(kind string, name string) (*ResourceResult, error) {
	ctx := context.Background()
	ns := ac.namespace
	allNs := false
	if len(ns) == 0 {
		allNs = true
	}

	// discover api and resources that have a list method
	resTypes, err := ac.d.SupportedResourceTypes()
	if err != nil {
		return nil, err
	}
	log.Debug().Msg("completed querying resource types")

	// return all resources for specified resource tpyes and namespace
	log.Debug().Msg("fetch all resource objects")
	resourceObjects, err := ac.d.GetAllResources(ctx, resTypes, ns, allNs)
	if err != nil {
		return nil, errors.Wrap(err, "could not query resource objects")
	}
	log.Debug().Msgf("found %d resource objects", len(resourceObjects))

	// find root nodes
	resType, rootResources, err := resources.FilterResource(resTypes, resourceObjects, kind, name)

	return &ResourceResult{
		Name:          name,
		Kind:          kind,
		ResourceType:  resType,
		AllResources:  resourceObjects,
		RootResources: rootResources,
		Namespace:     ns,
		AllNs:         allNs,
	}, err
}

func (ac *ApiConnector) PlatformInfo() *platform.Platform {
	release := ""
	build := ""
	arch := ""

	sv := ac.ServerVersion()
	if sv != nil {
		release = sv.GitVersion
		build = sv.BuildDate
		arch = sv.Platform
	}

	return &platform.Platform{
		Name:    "kubernetes",
		Title:   "Kubernetes",
		Release: release,
		Build:   build,
		Arch:    arch,
		Kind:    transports.Kind_KIND_API,
		Runtime: transports.RUNTIME_KUBERNETES,
	}
}

func (ac *ApiConnector) Namespaces() (*v1.NamespaceList, error) {
	ctx := context.Background()
	clientset, err := kubernetes.NewForConfig(ac.config)
	if err != nil {
		return nil, errors.Wrap(err, "could not create kubernetes clientset")
	}

	return clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
}

func (ac *ApiConnector) Pods(namespace v1.Namespace) (*v1.PodList, error) {
	ctx := context.Background()
	clientset, err := kubernetes.NewForConfig(ac.config)
	if err != nil {
		return nil, errors.Wrap(err, "could not create kubernetes clientset")
	}
	return clientset.CoreV1().Pods(namespace.Name).List(ctx, metav1.ListOptions{})
}
