// copyright: 2019, Dominik Richter and Christoph Hartmann
// author: Dominik Richter
// author: Christoph Hartmann

package k8s

import (
	"strings"

	"go.mondoo.com/cnquery/resources/packs/core"
)

// parseFlagsIntoConfig adds flags to the kubelet config
// It does not take care of deprecated flags
// The flags are not just kubelet specific, but can also be global flags
// That also means, that some flags do not have a matching parameter in the kubelet config file and are added as is
// The list of flags is taken from
// /var/lib/minikube/binaries/v1.24.3/kubelet --help | grep -v DEPRECATED | grep -v -E "(vmodule|version|help|v level)"
func parseFlagsIntoConfig(kubeletConfig map[string]interface{}, flags map[string]interface{}) error {
	if _, ok := flags["azure-container-registry-config"]; ok {
		kubeletConfig["azure-container-registry-config"] = flags["azure-container-registry-config"]
	}
	if _, ok := flags["bootstrap-kubeconfig"]; ok {
		kubeletConfig["bootstrap-kubeconfig"] = flags["bootstrap-kubeconfig"]
	}
	if _, ok := flags["cert-dir"]; ok {
		kubeletConfig["cert-dir"] = flags["cert-dir"]
	}
	if _, ok := flags["config"]; ok {
		kubeletConfig["config"] = flags["config"]
	}
	if _, ok := flags["container-runtime-endpoint"]; ok {
		kubeletConfig["container-runtime-endpoint"] = flags["container-runtime-endpoint"]
	}
	if _, ok := flags["exit-on-lock-contention"]; ok {
		kubeletConfig["exit-on-lock-contention"] = flags["exit-on-lock-contention"]
	}
	if _, ok := flags["feature-gates"]; ok {
		featureFlags := map[string]string{}
		for _, feature := range strings.Split(flags["feature-gates"].(string), ",") {
			featureSplit := strings.Split(feature, "=")
			featureFlags[featureSplit[0]] = featureSplit[1]
		}
		data, err := core.JsonToDict(featureFlags)
		if err != nil {
			return err
		}
		kubeletConfig["feature-gates"] = data
	}
	if _, ok := flags["hostname-override"]; ok {
		kubeletConfig["hostname-override"] = flags["hostname-override"]
	}
	if _, ok := flags["housekeeping-interval"]; ok {
		kubeletConfig["housekeeping-interval"] = flags["housekeeping-interval"]
	} else {
		kubeletConfig["housekeeping-interval"] = "10s"
	}
	if _, ok := flags["image-credential-provider-bin-dir"]; ok {
		kubeletConfig["image-credential-provider-bin-dir"] = flags["image-credential-provider-bin-dir"]
	}
	if _, ok := flags["image-credential-provider-config"]; ok {
		kubeletConfig["image-credential-provider-config"] = flags["image-credential-provider-config"]
	}
	if _, ok := flags["image-service-endpoint"]; ok {
		kubeletConfig["image-service-endpoint"] = flags["image-service-endpoint"]
	}
	if _, ok := flags["kubeconfig"]; ok {
		kubeletConfig["kubeconfig"] = flags["kubeconfig"]
	}
	if _, ok := flags["lock-file"]; ok {
		kubeletConfig["lock-file"] = flags["lock-file"]
	}
	if _, ok := flags["log-flush-frequency"]; ok {
		kubeletConfig["log-flush-frequency"] = flags["log-flush-frequency"]
	}
	if _, ok := flags["logging-format"]; ok {
		kubeletConfig["logging-format"] = flags["logging-format"]
	}
	if _, ok := flags["node-ip"]; ok {
		kubeletConfig["node-ip"] = flags["node-ip"]
	}
	if _, ok := flags["node-labels"]; ok {
		nodeLabels := map[string]string{}
		for _, label := range strings.Split(flags["node-labels"].(string), ",") {
			labelSplit := strings.Split(label, "=")
			nodeLabels[labelSplit[0]] = labelSplit[1]
		}
		data, err := core.JsonToDict(nodeLabels)
		if err != nil {
			return err
		}
		kubeletConfig["node-labels"] = data
	}
	if _, ok := flags["root-dir"]; ok {
		kubeletConfig["root-dir"] = flags["root-dir"]
	}
	if _, ok := flags["runtime-cgroups"]; ok {
		kubeletConfig["runtime-cgroups"] = flags["runtime-cgroups"]
	}
	if _, ok := flags["seccomp-default"]; ok {
		kubeletConfig["seccomp-default"] = flags["seccomp-default"]
	}
	if _, ok := flags["tls-cipher-suites"]; ok {
		ciphers := strings.Split(flags["tls-cipher-suites"].(string), ",")
		data, err := core.JsonToDictSlice(ciphers)
		if err != nil {
			return err
		}
		kubeletConfig["tls-cipher-suites"] = data
	}

	return nil
}
