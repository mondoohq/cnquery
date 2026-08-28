// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"go.mondoo.com/mql/llx"
	"go.mondoo.com/mql/providers/os/connection/shared"
	"go.mondoo.com/mql/providers/os/resources/powershell"
)

// Win32_PnPSignedDriver is the broadest driver source that needs no admin rights
// and carries signing status (IsSigned/Signer) directly. @(...) forces an array
// even for a single row.
const windowsDriversScript = `@(Get-CimInstance -ClassName Win32_PnPSignedDriver -ErrorAction SilentlyContinue |
  Select-Object DeviceName,FriendlyName,Description,DeviceClass,DriverProviderName,DriverVersion,
    DriverDate,IsSigned,Signer,InfName,DeviceID) | ConvertTo-Json -Compress`

type windowsDriverJSON struct {
	DeviceName         string          `json:"DeviceName"`
	FriendlyName       string          `json:"FriendlyName"`
	Description        string          `json:"Description"`
	DeviceClass        string          `json:"DeviceClass"`
	DriverProviderName string          `json:"DriverProviderName"`
	DriverVersion      string          `json:"DriverVersion"`
	DriverDate         json.RawMessage `json:"DriverDate"`
	IsSigned           bool            `json:"IsSigned"`
	Signer             string          `json:"Signer"`
	InfName            string          `json:"InfName"`
	DeviceID           string          `json:"DeviceID"`
}

// parseDriverDate tolerates the several shapes a WMI DriverDate can take across
// PowerShell versions: an ISO-8601 string (PS 7), a "/Date(ms)/" string
// (PS 5.1 / .NET), or a null/object which yields no timestamp.
func parseDriverDate(raw json.RawMessage) *time.Time {
	if len(raw) == 0 {
		return nil
	}
	var s string
	if err := json.Unmarshal(raw, &s); err != nil {
		return nil // null, object, or anything non-string -> no date
	}
	if strings.HasPrefix(s, "/Date(") {
		ms := strings.TrimSuffix(strings.TrimPrefix(s, "/Date("), ")/")
		if off := strings.IndexAny(ms, "+-"); off > 0 {
			ms = ms[:off]
		}
		if v, err := strconv.ParseInt(ms, 10, 64); err == nil {
			t := time.UnixMilli(v).UTC()
			return &t
		}
		return nil
	}
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339, "2006-01-02T15:04:05"} {
		if t, err := time.Parse(layout, s); err == nil {
			return &t
		}
	}
	return nil
}

func (w *mqlWindows) drivers() ([]any, error) {
	res := []any{}
	conn, ok := w.MqlRuntime.Connection.(shared.Connection)
	if !ok {
		return res, nil
	}

	cmd, err := conn.RunCommand(powershell.Encode(windowsDriversScript))
	if err != nil {
		return nil, err
	}
	out := strings.TrimSpace(readAll(cmd.Stdout))
	if out == "" {
		return res, nil
	}

	var drivers []windowsDriverJSON
	if out[0] == '[' {
		if err := json.Unmarshal([]byte(out), &drivers); err != nil {
			return nil, err
		}
	} else {
		var single windowsDriverJSON
		if err := json.Unmarshal([]byte(out), &single); err != nil {
			return nil, err
		}
		drivers = append(drivers, single)
	}

	for i := range drivers {
		d := drivers[i]
		ts := parseDriverDate(d.DriverDate)
		entry, err := CreateResource(w.MqlRuntime, "windows.driver", map[string]*llx.RawData{
			"name":        llx.StringData(d.DeviceName),
			"displayName": llx.StringData(d.FriendlyName),
			"description": llx.StringData(d.Description),
			"class":       llx.StringData(d.DeviceClass),
			"provider":    llx.StringData(d.DriverProviderName),
			"version":     llx.StringData(d.DriverVersion),
			"date":        llx.TimeDataPtr(ts),
			"signed":      llx.BoolData(d.IsSigned),
			"signer":      llx.StringData(d.Signer),
			"inf":         llx.StringData(d.InfName),
			"deviceId":    llx.StringData(d.DeviceID),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, entry)
	}
	return res, nil
}

func (d *mqlWindowsDriver) id() (string, error) {
	id := d.DeviceId.Data
	if id == "" {
		id = d.Name.Data + "/" + d.Version.Data + "/" + d.Inf.Data
	}
	return "windows.driver/" + id, nil
}
