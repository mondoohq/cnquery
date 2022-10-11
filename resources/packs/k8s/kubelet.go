// copyright: 2019, Dominik Richter and Christoph Hartmann
// author: Dominik Richter
// author: Christoph Hartmann

package k8s

import (
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/afero"
	kubeletconfigv1beta1 "k8s.io/kubelet/config/v1beta1"

	"sigs.k8s.io/yaml"

	"go.mondoo.com/cnquery/motor/providers/os"
	"go.mondoo.com/cnquery/resources"
	"go.mondoo.com/cnquery/resources/packs/core"
)

const defaultKubeletConfig = "/var/lib/kubelet/config.yaml"

func (k *mqlK8sKubelet) init(args *resources.Args) (*resources.Args, K8sKubelet, error) {
	if len(*args) > 0 {
		return args, nil, nil
	}

	p, err := k.getKubeletProcess()
	if err != nil {
		return nil, nil, err
	}
	(*args)["process"] = p

	kubeletFlags, err := p.Flags()
	if err != nil {
		return nil, nil, err
	}

	// Check kubelet for "--config" flag and set path to config file accordingly
	configFilePath := defaultKubeletConfig
	if kubeletConfigFilePath, ok := kubeletFlags["config"]; ok {
		path, ok := kubeletConfigFilePath.(string)
		if !ok {
			return nil, nil, errors.New("wrong type for value of '--config' flag, it must be a string")
		}
		configFilePath = path
	}

	provider, ok := k.MotorRuntime.Motor.Provider.(os.OperatingSystemProvider)
	if !ok {
		return nil, nil, fmt.Errorf("error getting operating system provider")
	}
	// AKS has no kubelet config file
	configFileExists, err := afero.Exists(provider.FS(), configFilePath)
	if err != nil {
		return nil, nil, fmt.Errorf("error when checking whether config file exists: %v", err)
	}

	if configFileExists {
		f, err := k.MotorRuntime.CreateResource("file", "path", configFilePath)
		if err != nil {
			return nil, nil, err
		}
		mqlFile, ok := f.(core.File)
		if !ok {
			return nil, nil, err
		}
		(*args)["configFile"] = mqlFile
	} else {
		(*args)["configFile"] = nil
	}

	// I cannot re-use "mqlFile" here, as it is not read at this point in time
	configuration, err := k.createConfiguration(kubeletFlags, configFilePath, provider, configFileExists)
	if err != nil {
		return nil, nil, err
	}
	(*args)["configuration"] = configuration

	return args, nil, nil
}

func (k *mqlK8sKubelet) id() (string, error) {
	return "k8s.kubelet", nil
}

// createConfiguration applies the kubelet defaults to the config and then
// merges the kubelet flags and the kubelet config file into a single map
// This map is representing the running state of the kubelet config
func (k *mqlK8sKubelet) createConfiguration(kubeletFlags map[string]interface{}, configFilePath string, provider os.OperatingSystemProvider, configFileExists bool) (map[string]interface{}, error) {
	kubeletConfig := kubeletconfigv1beta1.KubeletConfiguration{}
	SetDefaults_KubeletConfiguration(&kubeletConfig)

	// AKS has no kubelet config file
	if configFileExists {
		configFileContent, err := afero.ReadFile(provider.FS(), configFilePath)
		if err != nil {
			return nil, fmt.Errorf("error when getting file content: %v", err)
		}
		err = yaml.Unmarshal([]byte(configFileContent), &kubeletConfig)
		if err != nil {
			return nil, fmt.Errorf("error when converting file content into KubeletConfiguration: %v", err)
		}
	}

	options, err := core.JsonToDict(kubeletConfig)
	if err != nil {
		return nil, fmt.Errorf("error when converting KubeletConfig into dict: %v", err)
	}

	err = mergeFlagsIntoConfig(options, kubeletFlags)
	if err != nil {
		return nil, fmt.Errorf("error applying precedence to KubeletConfig: %v", err)
	}

	err = mergeDeprecatedFlagsIntoConfig(options, kubeletFlags)
	if err != nil {
		return nil, fmt.Errorf("error applying precedence for deprecated flags to KubeletConfig: %v", err)
	}

	return options, nil
}

func (k *mqlK8sKubelet) getKubeletProcess() (core.Process, error) {
	obj, err := k.MotorRuntime.CreateResource("processes")
	if err != nil {
		return nil, err
	}
	processes := obj.(core.Processes)

	processItems, err := processes.List()
	if err != nil {
		return nil, err
	}
	for _, process := range processItems {
		mqlProcess := process.(core.Process)
		exec, err := mqlProcess.Executable()
		if err != nil {
			continue
		}
		if strings.HasSuffix(exec, "kubelet") {
			return mqlProcess, nil
		}
	}
	return nil, errors.New("no kubelet process found")
}
