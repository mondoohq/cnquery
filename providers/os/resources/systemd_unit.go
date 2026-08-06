// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"errors"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
	"go.mondoo.com/mql/v13/providers/os/connection/shared"
	"go.mondoo.com/mql/v13/providers/os/resources/services"
	"go.mondoo.com/mql/v13/types"
)

func (u *mqlSystemdUnit) id() (string, error) {
	return "systemd.unit:" + u.Name.Data, nil
}

func initSystemdUnit(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) != 1 {
		return args, nil, nil
	}

	x, ok := args["name"]
	if !ok {
		return nil, nil, errors.New("cannot initialize systemd.unit, need at least a name")
	}

	name, ok := x.Value.(string)
	if !ok || name == "" {
		return nil, nil, errors.New("cannot look for a systemd unit with an empty name")
	}

	conn, ok := runtime.Connection.(shared.Connection)
	if !ok {
		return nil, nil, errors.New("systemd.unit is not supported on this connection")
	}

	mgr := services.ResolveSystemdUnitManager(conn)
	unit, err := mgr.Get(name)
	if err != nil {
		if errors.Is(err, services.ErrServiceNotFound) {
			res, err := createSystemdUnitResource(runtime, &services.SystemdUnit{Name: name})
			if err != nil {
				return nil, nil, err
			}
			return nil, res, nil
		}
		return nil, nil, err
	}

	res, err := createSystemdUnitResource(runtime, unit)
	if err != nil {
		return nil, nil, err
	}

	return nil, res, nil
}

// createSystemdUnitResource builds the resource from a unit. A unit that was not
// found is passed in zero-valued apart from its name, which reports every
// setting as explicitly off rather than as null, so an assertion over several of
// them fails on a miss instead of passing vacuously.
func createSystemdUnitResource(runtime *plugin.Runtime, unit *services.SystemdUnit) (plugin.Resource, error) {
	return CreateResource(runtime, "systemd.unit", map[string]*llx.RawData{
		"__id":          llx.StringData("systemd.unit:" + unit.Name),
		"name":          llx.StringData(unit.Name),
		"description":   llx.StringData(unit.Description),
		"installed":     llx.BoolData(unit.Installed),
		"fragmentPath":  llx.StringData(unit.FragmentPath),
		"loadState":     llx.StringData(unit.LoadState),
		"activeState":   llx.StringData(unit.ActiveState),
		"unitFileState": llx.StringData(unit.UnitFileState),
		"type":          llx.StringData(unit.Type),
		"execStart":     llx.StringData(unit.ExecStart),

		"user":        llx.StringData(unit.User),
		"group":       llx.StringData(unit.Group),
		"dynamicUser": llx.BoolData(unit.DynamicUser),
		"umask":       llx.StringData(unit.UMask),

		"noNewPrivileges":         llx.BoolData(unit.NoNewPrivileges),
		"protectSystem":           llx.StringData(unit.ProtectSystem),
		"protectHome":             llx.StringData(unit.ProtectHome),
		"privateTmp":              llx.BoolData(unit.PrivateTmp),
		"privateDevices":          llx.BoolData(unit.PrivateDevices),
		"privateNetwork":          llx.BoolData(unit.PrivateNetwork),
		"privateUsers":            llx.BoolData(unit.PrivateUsers),
		"protectKernelTunables":   llx.BoolData(unit.ProtectKernelTunables),
		"protectKernelModules":    llx.BoolData(unit.ProtectKernelModules),
		"protectKernelLogs":       llx.BoolData(unit.ProtectKernelLogs),
		"protectControlGroups":    llx.StringData(unit.ProtectControlGroups),
		"protectClock":            llx.BoolData(unit.ProtectClock),
		"protectHostname":         llx.BoolData(unit.ProtectHostname),
		"protectProc":             llx.StringData(unit.ProtectProc),
		"procSubset":              llx.StringData(unit.ProcSubset),
		"restrictSUIDSGID":        llx.BoolData(unit.RestrictSUIDSGID),
		"restrictRealtime":        llx.BoolData(unit.RestrictRealtime),
		"restrictNamespaces":      llx.StringData(unit.RestrictNamespaces),
		"restrictAddressFamilies": llx.StringData(unit.RestrictAddressFamilies),
		"lockPersonality":         llx.BoolData(unit.LockPersonality),
		"memoryDenyWriteExecute":  llx.BoolData(unit.MemoryDenyWriteExecute),
		"removeIPC":               llx.BoolData(unit.RemoveIPC),
		"keyringMode":             llx.StringData(unit.KeyringMode),

		"capabilityBoundingSet":   llx.ArrayData(convert.SliceAnyToInterface(unit.CapabilityBoundingSet), types.String),
		"ambientCapabilities":     llx.ArrayData(convert.SliceAnyToInterface(unit.AmbientCapabilities), types.String),
		"systemCallFilter":        llx.ArrayData(convert.SliceAnyToInterface(unit.SystemCallFilter), types.String),
		"systemCallArchitectures": llx.StringData(unit.SystemCallArchitectures),
		"readWritePaths":          llx.ArrayData(convert.SliceAnyToInterface(unit.ReadWritePaths), types.String),
		"readOnlyPaths":           llx.ArrayData(convert.SliceAnyToInterface(unit.ReadOnlyPaths), types.String),
		"inaccessiblePaths":       llx.ArrayData(convert.SliceAnyToInterface(unit.InaccessiblePaths), types.String),
	})
}

func (u *mqlSystemdUnits) id() (string, error) {
	return "systemd.units", nil
}

func (u *mqlSystemdUnits) list() ([]any, error) {
	conn, ok := u.MqlRuntime.Connection.(shared.Connection)
	if !ok {
		return nil, errors.New("systemd.units is not supported on this connection")
	}

	mgr := services.ResolveSystemdUnitManager(conn)
	units, err := mgr.List()
	if err != nil {
		return nil, err
	}

	res := make([]any, 0, len(units))
	for _, unit := range units {
		resource, err := createSystemdUnitResource(u.MqlRuntime, unit)
		if err != nil {
			return nil, err
		}
		res = append(res, resource)
	}

	return res, nil
}
