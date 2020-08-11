package arista

import "github.com/aristanetworks/goeapi"

type ShowVersion struct {
	ModelName        string  `json:"modelName"`
	InternalVersion  string  `json:"internalVersion"`
	MfgName          string  `json:"mfgName"`
	ConfigMacAddress string  `json:"configMacAddress"`
	IsIntlVersion    bool    `json:"isIntlVersion"`
	SystemMacAddress string  `json:"systemMacAddress"`
	SerialNumber     string  `json:"serialNumber"`
	MemTotal         int     `json:"memTotal"`
	MemFree          int     `json:"memFree"`
	Uptime           float64 `json:"uptime"`
	BootupTimestamp  float64 `json:"bootupTimestamp"`

	Version          string `json:"version"`
	Architecture     string `json:"architecture"`
	InternalBuildID  string `json:"internalBuildId"`
	HardwareRevision string `json:"hardwareRevision"`
	HwMacAddress     string `json:"hwMacAddress"`
}

// GetCmd returns the command type this EapiCommand relates to
func (s ShowVersion) GetCmd() string {
	return "show version"
}

func GetVersion(node *goeapi.Node) (ShowVersion, error) {
	// NOTE: we do not use the built-in version sind the json conversion does not generate camelCase entry since the json tags are missing
	handle, err := node.GetHandle("json")
	if err != nil {
		return ShowVersion{}, err
	}

	var showversion ShowVersion
	err = handle.AddCommand(&showversion)
	if err != nil {
		return ShowVersion{}, err
	}

	if err := handle.Call(); err != nil {
		return ShowVersion{}, err
	}

	handle.Close()
	return showversion, nil
}
