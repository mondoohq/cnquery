package k8s

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/gosimple/slug"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/spf13/afero"
	"go.mondoo.io/mondoo/motor/platform"
	"go.mondoo.io/mondoo/motor/transports"
	"go.mondoo.io/mondoo/motor/transports/fsutil"
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

func newApiTransport(namespace string, selectedResourceID string) (Transport, error) {
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

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, errors.Wrap(err, "could not create kubernetes clientset")
	}

	return &apiTransport{
		namespace:          namespace,
		config:             config,
		d:                  d,
		clientset:          clientset,
		selectedResourceID: selectedResourceID,
	}, nil
}

type apiTransport struct {
	d                  *resources.Discovery
	config             *rest.Config
	namespace          string
	clientset          *kubernetes.Clientset
	selectedResourceID string
}

func (t *apiTransport) RunCommand(command string) (*transports.Command, error) {
	return nil, errors.New("k8s does not implement RunCommand")
}

func (t *apiTransport) FileInfo(path string) (transports.FileInfoDetails, error) {
	return transports.FileInfoDetails{}, errors.New("k8s does not implement FileInfo")
}

func (t *apiTransport) FS() afero.Fs {
	return &fsutil.NoFs{}
}

func (t *apiTransport) Close() {}

func (t *apiTransport) Capabilities() transports.Capabilities {
	return transports.Capabilities{}
}

func (t *apiTransport) Kind() transports.Kind {
	return transports.Kind_KIND_API
}

func (t *apiTransport) Runtime() string {
	return transports.RUNTIME_KUBERNETES_CLUSTER
}

func (t *apiTransport) PlatformIdDetectors() []transports.PlatformIdDetector {
	return []transports.PlatformIdDetector{
		transports.TransportPlatformIdentifierDetector,
	}
}

func (t *apiTransport) ID() (string, error) {
	// we use "kube-system" namespace uid as identifier for the cluster
	result, err := t.Resources("namespaces", "kube-system")
	if err != nil {
		return "", err
	}

	if len(result.Resources) != 1 {
		return "", errors.New("could not identify the k8s cluster")
	}

	resource := result.Resources[0]
	obj, err := meta.Accessor(resource)
	if err != nil {
		return "", err
	}

	uid := string(obj.GetUID())
	return uid, nil
}

func (t *apiTransport) PlatformIdentifier() (string, error) {
	if t.selectedResourceID != "" {
		return t.selectedResourceID, nil
	}

	uid, err := t.ID()
	if err != nil {
		return "", err
	}

	id := NewPlatformID(uid)
	if t.namespace != "" {
		id += "/namespace/" + slug.Make(t.namespace)
	}

	return id, nil
}

func (t *apiTransport) Identifier() (string, error) {
	return t.PlatformIdentifier()
}

func (t *apiTransport) Name() (string, error) {
	ci, err := t.ClusterInfo()
	if err != nil {
		return "", err
	}
	return ci.Name, nil
}

func (t *apiTransport) ClusterInfo() (ClusterInfo, error) {
	ctx := context.Background()
	res := ClusterInfo{}

	// right now we use the name of the first node to identify the cluster
	result, err := t.clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return res, err
	}

	if len(result.Items) > 0 {
		node := result.Items[0]
		res.Name = node.GetName()
	}

	return res, nil
}

func (t *apiTransport) ServerVersion() *version.Info {
	return t.d.ServerVersion
}

func (t *apiTransport) SupportedResourceTypes() (*resources.ApiResourceIndex, error) {
	return t.d.SupportedResourceTypes()
}

func (t *apiTransport) Resources(kind string, name string) (*ResourceResult, error) {
	ctx := context.Background()
	ns := t.namespace
	allNs := false
	if len(ns) == 0 {
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
	objs, err := t.d.GetKindResources(ctx, *resType, ns, allNs)
	if err != nil {
		return nil, err
	}
	log.Debug().Msgf("found %d resource objects", len(objs))

	objs, err = resources.FilterResource(resType, objs, name)
	if err != nil {
		return nil, err
	}

	return &ResourceResult{
		Name:         name,
		Kind:         kind,
		ResourceType: resType,
		Resources:    objs,
		Namespace:    ns,
		AllNs:        allNs,
	}, err
}

func (t *apiTransport) PlatformInfo() *platform.Platform {
	release := ""
	build := ""
	arch := ""

	// check if it is a pod which shares the same connection
	// log.Info().Str("selected resource", t.selectedResourceID).Msg("check for platform info")
	platformData := &platform.Platform{
		Release: "",
		Build:   "",
		Arch:    "",
		Family:  []string{"k8s", "k8s-workload"},
		Kind:    transports.Kind_KIND_K8S_OBJECT,
		Runtime: transports.RUNTIME_KUBERNETES_CLUSTER,
	}
	switch selected := t.selectedResourceID; {
	case strings.Contains(selected, "/pods/"):
		platformData.Name = "k8s-pod"
		platformData.Title = "Kubernetes Pod"
		return platformData
	}

	// cluster
	sv := t.ServerVersion()
	if sv != nil {
		release = sv.GitVersion
		build = sv.BuildDate
		arch = sv.Platform
	}

	return &platform.Platform{
		Name:    "kubernetes",
		Title:   "Kubernetes",
		Release: release,
		Version: release,
		Build:   build,
		Arch:    arch,
		Family:  []string{"kubernetes"},
		Kind:    transports.Kind_KIND_API,
		Runtime: transports.RUNTIME_KUBERNETES_CLUSTER,
	}
}

func (t *apiTransport) Namespaces() ([]v1.Namespace, error) {
	ctx := context.Background()
	list, err := t.clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	return list.Items, err
}

func (t *apiTransport) Pods(namespace v1.Namespace) ([]v1.Pod, error) {
	ctx := context.Background()
	list, err := t.clientset.CoreV1().Pods(namespace.Name).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	return list.Items, err
}

func (t *apiTransport) Pod(namespace string, name string) (*v1.Pod, error) {
	ctx := context.Background()
	pod, err := t.clientset.CoreV1().Pods(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return pod, err
}
