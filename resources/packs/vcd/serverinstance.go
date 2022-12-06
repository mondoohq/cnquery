package vcd

import (
	"github.com/vmware/go-vcloud-director/v2/govcd"
	"github.com/vmware/go-vcloud-director/v2/types/v56"
	"go.mondoo.com/cnquery/resources"
)

// https://developer.vmware.com/apis/1260/vmware-cloud-director/doc/doc/types/VimServerType.html
func (s *mqlVcd) GetServerInstances() ([]interface{}, error) {
	op, err := vcdProvider(s.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}
	client := op.Client()

	vdcList, err := govcd.QueryVirtualCenters(client, "")
	if err != nil {
		return nil, err
	}

	list := []interface{}{}
	for i := range vdcList {
		entry, err := newMqlVcdServerInstance(s.MotorRuntime, vdcList[i])
		if err != nil {
			return nil, err
		}
		list = append(list, entry)
	}

	return list, nil
}

func newMqlVcdServerInstance(runtime *resources.Runtime, vcdProvider *types.QueryResultVirtualCenterRecordType) (interface{}, error) {
	return runtime.CreateResource("vcd.serverInstance",
		"name", vcdProvider.Name,
		"isBusy", vcdProvider.IsBusy,
		"isEnabled", vcdProvider.IsEnabled,
		"isSupported", vcdProvider.IsEnabled,
		"listenerState", vcdProvider.ListenerState,
		"status", vcdProvider.Status,
		"userName", vcdProvider.UserName,
		"vcVersion", vcdProvider.VcVersion,
		"uuid", vcdProvider.UUID,
		"vsmIP", vcdProvider.VsmIP,
	)
}

func (o *mqlVcdServerInstance) id() (string, error) {
	id, err := o.Name()
	if err != nil {
		return "", err
	}
	return "vcd.serverInstance/" + id, nil
}
