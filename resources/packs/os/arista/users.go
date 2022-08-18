package arista

import "github.com/aristanetworks/goeapi/module"

func (eos *Eos) Users() []module.UserConfig {
	sys := module.User(eos.node)
	userConfigs := sys.GetAll()

	res := []module.UserConfig{}
	for i := range userConfigs {
		res = append(res, userConfigs[i])
	}
	return res
}

type showRoles struct {
	DefaultRole string              `json:"defaultRole"`
	Roles       map[string]UserRole `json:"roles"`
}

func (s *showRoles) GetCmd() string {
	return "show users roles"
}

type UserRole struct {
	Rules []struct {
		CmdPermission  string `json:"cmdPermission"`
		CmdRegex       string `json:"cmdRegex"`
		SequenceNumber int64  `json:"sequenceNumber"`
		Mode           string `json:"mode"`
	} `json:"rules"`
}

type EosRole struct {
	UserRole
	Name    string `json:"name"`
	Default bool   `json:"default"`
}

// show users roles
func (eos *Eos) Roles() ([]EosRole, error) {
	shRsp := &showRoles{}

	handle, err := eos.node.GetHandle("json")
	if err != nil {
		return nil, err
	}
	err = handle.AddCommand(shRsp)
	if err != nil {
		return nil, err
	}

	if err := handle.Call(); err != nil {
		return nil, err
	}

	handle.Close()

	res := []EosRole{}
	for k := range shRsp.Roles {
		res = append(res, EosRole{
			UserRole: shRsp.Roles[k],
			Default:  shRsp.DefaultRole == k,
			Name:     k,
		})
	}

	return res, nil
}
