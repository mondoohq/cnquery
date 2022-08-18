package arista

type showSnmp struct {
	Enabled bool `json:"enabled"`
}

func (s *showSnmp) GetCmd() string {
	return "show snmp"
}

func (eos *Eos) Snmp() (*showSnmp, error) {
	shRsp := &showSnmp{}

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

	return shRsp, nil
}

type showSnmpNotifications struct {
	Notifications []Notification `json:"notifications"`
}

func (s *showSnmpNotifications) GetCmd() string {
	return "show snmp notification"
}

type Notification struct {
	Reason    string `json:"reason"`
	Name      string `json:"name"`
	Enabled   bool   `json:"enabled"`
	Component string `json:"component"`
}

func (eos *Eos) SnmpNotifications() ([]Notification, error) {
	shRsp := &showSnmpNotifications{}

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

	return shRsp.Notifications, nil
}
