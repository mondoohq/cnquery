package kubectl

import (
	"fmt"
	"io"
	"io/ioutil"

	"go.mondoo.com/cnquery/motor/providers/os"

	"sigs.k8s.io/yaml"
)

// holds content of kubectl config view --minify
type KubectlConfig struct {
	Kind           string                          `json:"kind"`
	ApiVersion     string                          `json:"apiVersion"`
	CurrentContext string                          `json:"current-context"`
	Clusters       []*KubectlConfigClusterWithName `json:"clusters"`
	Contexts       []*KubectlConfigContextWithName `json:"contexts"`
	Users          []*KubectlConfigUserWithName    `json:"users"`
}

type KubectlConfigClusterWithName struct {
	Name    string               `json:"name"`
	Cluster KubectlConfigCluster `json:"cluster"`
}

type KubectlConfigCluster struct {
	Server                   string `json:"server,omitempty"`
	CertificateAuthorityData []byte `json:"certificate-authority-data,omitempty"`
}

type KubectlConfigContextWithName struct {
	Name    string               `json:"name"`
	Context KubectlConfigContext `json:"context"`
}

type KubectlConfigContext struct {
	Cluster   string `json:"cluster"`
	Namespace string `json:"namespace"`
	User      string `json:"user"`
}

type KubectlConfigUserWithName struct {
	Name string            `json:"name"`
	User KubectlConfigUser `json:"user"`
}

type KubectlConfigUser struct {
	ClientCertificateData []byte `json:"client-certificate-data,omitempty"`
	ClientKeyData         []byte `json:"client-key-data,omitempty"`
	Password              string `json:"password,omitempty"`
	Username              string `json:"username,omitempty"`
	Token                 string `json:"token,omitempty"`
}

func (kc *KubectlConfig) CurrentClusterName() string {
	name := ""
	for i := range kc.Contexts {
		if kc.Contexts[i].Name == kc.CurrentContext {
			if len(kc.Contexts[i].Context.Cluster) > 0 {
				return kc.Contexts[i].Context.Cluster
			} else {
				return name
			}
		}
	}

	return name
}

// Reads the namespace that is configured via `kubectl config set-context --current --namespace=default`
func (kc *KubectlConfig) CurrentNamespace() string {
	defaultNamespace := "default"
	for i := range kc.Contexts {
		if kc.Contexts[i].Name == kc.CurrentContext {
			if len(kc.Contexts[i].Context.Namespace) > 0 {
				return kc.Contexts[i].Context.Namespace
			} else {
				return defaultNamespace
			}
		}
	}

	return defaultNamespace
}

// ParseKubectlConfig parses the output of `kubectl config view --minify`
func ParseKubectlConfig(r io.Reader) (*KubectlConfig, error) {
	data, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}

	config := &KubectlConfig{}
	err = yaml.Unmarshal(data, config)
	if err != nil {
		return nil, fmt.Errorf("cannot parse current config from kubectl: %v", err)
	}
	return config, nil
}

func LoadKubeConfig(provider os.OperatingSystemProvider) (*KubectlConfig, error) {
	cmd, err := provider.RunCommand("kubectl config view --minify")
	if err != nil {
		return nil, err
	}

	return ParseKubectlConfig(cmd.Stdout)
}
