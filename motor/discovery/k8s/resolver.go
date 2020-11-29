package k8s

import (
	"strings"

	"github.com/rs/zerolog/log"
	"go.mondoo.io/mondoo/lumi/resources/kubectl"
	"go.mondoo.io/mondoo/motor"
	"go.mondoo.io/mondoo/motor/asset"
	"go.mondoo.io/mondoo/motor/transports"
	"go.mondoo.io/mondoo/motor/transports/local"
)

type K8sConfig struct {
	Context   string
	Namespace string
	Pod       string
}

func ParseK8SContext(k8sUrl string) K8sConfig {
	var config K8sConfig

	k8sUrl = strings.TrimPrefix(k8sUrl, "k8s://")

	keyValues := strings.Split(k8sUrl, "/")
	for i := 0; i < len(keyValues); {
		if keyValues[i] == "namespace" {
			if i+1 < len(keyValues) {
				config.Namespace = keyValues[i+1]
			}
		}
		if keyValues[i] == "context" {
			if i+1 < len(keyValues) {
				config.Context = keyValues[i+1]
			}
		}
		if keyValues[i] == "pod" {
			if i+1 < len(keyValues) {
				config.Pod = keyValues[i+1]
			}
		}
		i = i + 2
	}

	return config
}

type Resolver struct{}

func (r *Resolver) Name() string {
	return "Kubernetes Resolver"
}

func (r *Resolver) ParseConnectionURL(url string, opts ...transports.TransportConfigOption) (*transports.TransportConfig, error) {
	// parse context from url
	config := ParseK8SContext(url)

	return &transports.TransportConfig{
		// TODO: we need to set the backend here
		Options: map[string]string{
			"context":   config.Context,
			"namespace": config.Namespace,
			"pod":       config.Pod,
		},
	}, nil
}

func (r *Resolver) Resolve(t *transports.TransportConfig) ([]*asset.Asset, error) {
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

	k8sContext := t.Options["context"]
	if len(k8sContext) == 0 {
		// try to parse context from kubectl
		if k8sctlConfig != nil && len(k8sctlConfig.CurrentContext) > 0 {
			k8sContext = k8sctlConfig.CurrentContext
		}
	}

	namespace := t.Options["namespace"]
	if len(namespace) > 0 {
		namespacesFilter = append(namespacesFilter, namespace)
	} else {
		// try parse the current kubectl namespace
		if k8sctlConfig != nil && len(k8sctlConfig.CurrentNamespace()) > 0 {
			namespacesFilter = append(namespacesFilter, k8sctlConfig.CurrentNamespace())
		}
	}

	pod := t.Options["pod"]
	if len(pod) > 0 {
		podFilter = append(podFilter, pod)
	}

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

	return resolved, nil
}
