package os

import (
	"go.mondoo.com/cnquery/resources"
	"go.mondoo.com/cnquery/resources/packs/core/plist"
)

func (m *mqlMacosAlf) id() (string, error) {
	return "macos.alf", nil
}

func (s *mqlMacosAlf) init(args *resources.Args) (*resources.Args, MacosAlf, error) {
	// TODO: use s.Runtime.CreateResource("parse.plist", "path", "/Library/Preferences/com.apple.alf.plist") in future
	osProvider, err := osProvider(s.MotorRuntime.Motor)
	if err != nil {
		return nil, nil, err
	}

	f, err := osProvider.FS().Open("/Library/Preferences/com.apple.alf.plist")
	if err != nil {
		return nil, nil, err
	}
	defer f.Close()

	alfConfig, err := plist.Decode(f)
	if err != nil {
		return nil, nil, err
	}

	explicitAuthsRaw := alfConfig["explicitauths"].([]interface{})
	explicitAuths := []interface{}{}
	for i := range explicitAuthsRaw {
		entry := explicitAuthsRaw[i].(map[string]interface{})
		explicitAuths = append(explicitAuths, entry["id"])
	}

	(*args)["allowDownloadSignedEnabled"] = int64(alfConfig["allowdownloadsignedenabled"].(float64))
	(*args)["allowSignedEnabled"] = int64(alfConfig["allowsignedenabled"].(float64))
	(*args)["firewallUnload"] = int64(alfConfig["firewallunload"].(float64))
	(*args)["globalState"] = int64(alfConfig["globalstate"].(float64))
	(*args)["loggingEnabled"] = int64(alfConfig["loggingenabled"].(float64))
	(*args)["loggingOption"] = int64(alfConfig["loggingoption"].(float64))
	(*args)["stealthEnabled"] = int64(alfConfig["stealthenabled"].(float64))
	(*args)["version"] = alfConfig["version"].(string)
	(*args)["exceptions"] = alfConfig["exceptions"].([]interface{})
	(*args)["explicitAuths"] = explicitAuths
	(*args)["applications"] = alfConfig["applications"].([]interface{})

	return args, nil, nil
}
