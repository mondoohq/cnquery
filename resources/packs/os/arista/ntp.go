package arista

type showNtpStatus struct {
	Status string `json:"status"`
}

func (s *showNtpStatus) GetCmd() string {
	return "show ntp status"
}

func (eos *Eos) NtpStatus() (*showNtpStatus, error) {
	shRsp := &showNtpStatus{}

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
