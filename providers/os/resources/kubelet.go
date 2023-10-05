// copyright: 2019, Dominik Richter and Christoph Hartmann
// author: Dominik Richter
// author: Christoph Hartmann

package resources

import (
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/afero"
	kubeletconfigv1beta1 "k8s.io/kubelet/config/v1beta1"

	"sigs.k8s.io/yaml"

	"go.mondoo.com/cnquery/v9/llx"
	"go.mondoo.com/cnquery/v9/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v9/providers-sdk/v1/util/convert"
	"go.mondoo.com/cnquery/v9/providers/os/connection/shared"
	"go.mondoo.com/cnquery/v9/types"
)

const defaultKubeletConfig = "/var/lib/kubelet/config.yaml"

func initKubelet(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 0 {
		return args, nil, nil
	}

	p, err := getKubeletProcess(runtime)
	if err != nil {
		return nil, nil, err
	}
	args["process"] = llx.ResourceData(p, "process")

	kubeletFlagsData := p.GetFlags()
	if kubeletFlagsData.Error != nil {
		return nil, nil, err
	}
	kubeletFlags := kubeletFlagsData.Data

	// Check kubelet for "--config" flag and set path to config file accordingly
	configFilePath := defaultKubeletConfig
	if kubeletConfigFilePath, ok := kubeletFlags["config"]; ok {
		path, ok := kubeletConfigFilePath.(string)
		if !ok {
			return nil, nil, errors.New("wrong type for value of '--config' flag, it must be a string")
		}
		configFilePath = path
	}

	provider, ok := runtime.Connection.(shared.Connection)
	if !ok {
		return nil, nil, fmt.Errorf("error getting operating system provider")
	}
	// AKS has no kubelet config file
	configFileExists, err := afero.Exists(provider.FileSystem(), configFilePath)
	if err != nil {
		return nil, nil, fmt.Errorf("error when checking whether config file exists: %v", err)
	}

	if configFileExists {
		f, err := CreateResource(runtime, "file", map[string]*llx.RawData{
			"path": llx.StringData(configFilePath),
		})
		if err != nil {
			return nil, nil, err
		}
		mqlFile, ok := f.(*mqlFile)
		if !ok {
			return nil, nil, err
		}
		args["configFile"] = llx.ResourceData(mqlFile, "file")
	} else {
		args["configFile"] = llx.NilData
	}

	// I cannot re-use "mqlFile" here, as it is not read at this point in time
	configuration, err := createConfiguration(kubeletFlags, configFilePath, provider, configFileExists)
	if err != nil {
		return nil, nil, err
	}
	args["configuration"] = llx.MapData(configuration, types.String)

	return args, nil, nil
}

// createConfiguration applies the kubelet defaults to the config and then
// merges the kubelet flags and the kubelet config file into a single map
// This map is representing the running state of the kubelet config
func createConfiguration(kubeletFlags map[string]interface{}, configFilePath string, provider shared.Connection, configFileExists bool) (map[string]interface{}, error) {
	kubeletConfig := kubeletconfigv1beta1.KubeletConfiguration{}
	SetDefaults_KubeletConfiguration(&kubeletConfig)

	// AKS has no kubelet config file
	if configFileExists {
		configFileContent, err := afero.ReadFile(provider.FileSystem(), configFilePath)
		if err != nil {
			return nil, fmt.Errorf("error when getting file content: %v", err)
		}
		err = yaml.Unmarshal([]byte(configFileContent), &kubeletConfig)
		if err != nil {
			return nil, fmt.Errorf("error when converting file content into KubeletConfiguration: %v", err)
		}
	}

	options, err := convert.JsonToDict(kubeletConfig)
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

func getKubeletProcess(runtime *plugin.Runtime) (*mqlProcess, error) {
	obj, err := CreateResource(runtime, "processes", nil)
	if err != nil {
		return nil, err
	}
	processes := obj.(*mqlProcesses)

	data := processes.GetList()
	if data.Error != nil {
		return nil, data.Error
	}
	for _, process := range data.Data {
		mqlProcess := process.(*mqlProcess)
		exec := mqlProcess.Executable
		if exec.Error != nil {
			continue
		}
		if strings.HasSuffix(exec.Data, "kubelet") {
			return mqlProcess, nil
		}
	}
	return nil, errors.New("no kubelet process found")
}
