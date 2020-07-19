package vsphere

import (
	"context"
	"fmt"
	"strconv"

	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/types"
)

func hostProperties(host *object.HostSystem) (*mo.HostSystem, error) {
	ctx, cancel := context.WithTimeout(context.Background(), DefaultAPITimeout)
	defer cancel()
	var props mo.HostSystem
	if err := host.Properties(ctx, host.Reference(), nil, &props); err != nil {
		return nil, err
	}
	return &props, nil
}

func (c *Client) HostProperties(host *object.HostSystem) (map[string]interface{}, error) {
	props, err := hostProperties(host)
	if err != nil {
		return nil, err
	}

	dataProps := map[string]interface{}{}
	dataProps["PowerState"] = string(props.Runtime.PowerState)
	dataProps["ConnectionState"] = string(props.Runtime.ConnectionState)
	dataProps["InMaintenanceMode"] = strconv.FormatBool(props.Runtime.InMaintenanceMode)
	dataProps["LockdownMode"] = hostLockdownString(props.Config.LockdownMode)
	return dataProps, nil
}

func HostOptions(host *object.HostSystem) (map[string]interface{}, error) {
	ctx := context.Background()
	m, err := host.ConfigManager().OptionManager(ctx)
	if err != nil {
		return nil, err
	}

	var om mo.OptionManager
	err = m.Properties(ctx, m.Reference(), []string{"setting"}, &om)

	advancedProps := map[string]interface{}{}
	for i := range om.Setting {
		prop := om.Setting[i]
		key := prop.GetOptionValue().Key
		value := fmt.Sprintf("%v", prop.GetOptionValue().Value)
		advancedProps[key] = value
	}
	return advancedProps, nil
}

func HostServices(host *object.HostSystem) ([]types.HostService, error) {
	ctx := context.Background()
	ss, err := host.ConfigManager().ServiceSystem(ctx)
	if err != nil {
		return nil, err
	}
	return ss.Service(ctx)
}

func HostDateTime(host *object.HostSystem) (*types.HostDateTimeInfo, error) {
	ctx := context.Background()
	s, err := host.ConfigManager().DateTimeSystem(ctx)
	if err != nil {
		return nil, err
	}

	var hs mo.HostDateTimeSystem
	if err = s.Properties(ctx, s.Reference(), nil, &hs); err != nil {
		return nil, err
	}
	return &hs.DateTimeInfo, nil
}
