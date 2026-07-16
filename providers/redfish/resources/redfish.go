// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"encoding/json"
	"sync"

	"github.com/rs/zerolog/log"
	"github.com/stmcginnis/gofish/oem/smc"
	"github.com/stmcginnis/gofish/schemas"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/redfish/connection"
	"go.mondoo.com/mql/v13/types"
)

func redfishConn(runtime *plugin.Runtime) *connection.RedfishConnection {
	return runtime.Connection.(*connection.RedfishConnection)
}

// uintPtrToAny converts an optional *uint into a dict-friendly value.
func uintPtrToAny(p *uint) any {
	if p == nil {
		return nil
	}
	return int64(*p)
}

func (r *mqlRedfish) id() (string, error) {
	return "redfish", nil
}

func (r *mqlRedfish) systems() ([]any, error) {
	systems, err := redfishConn(r.MqlRuntime).Systems()
	if err != nil {
		return nil, err
	}

	res := make([]any, 0, len(systems))
	for _, s := range systems {
		secureBoot := false
		if sb, err := s.SecureBoot(); err == nil && sb != nil {
			secureBoot = sb.SecureBootEnable
		}

		o, err := CreateResource(r.MqlRuntime, "redfish.system", map[string]*llx.RawData{
			"__id":              llx.StringData(s.ODataID),
			"uuid":              llx.StringData(s.UUID),
			"name":              llx.StringData(s.Name),
			"manufacturer":      llx.StringData(s.Manufacturer),
			"model":             llx.StringData(s.Model),
			"serialNumber":      llx.StringData(s.SerialNumber),
			"sku":               llx.StringData(s.SKU),
			"biosVersion":       llx.StringData(s.BiosVersion),
			"hostName":          llx.StringData(s.HostName),
			"powerState":        llx.StringData(string(s.PowerState)),
			"systemType":        llx.StringData(string(s.SystemType)),
			"secureBootEnabled": llx.BoolData(secureBoot),
		})
		if err != nil {
			return nil, err
		}
		mqlSystem := o.(*mqlRedfishSystem)
		mqlSystem.sys = s
		res = append(res, mqlSystem)
	}
	return res, nil
}

func (r *mqlRedfish) managers() ([]any, error) {
	managers, err := redfishConn(r.MqlRuntime).Managers()
	if err != nil {
		return nil, err
	}

	res := make([]any, 0, len(managers))
	for _, m := range managers {
		o, err := CreateResource(r.MqlRuntime, "redfish.manager", map[string]*llx.RawData{
			"__id":            llx.StringData(m.ODataID),
			"uuid":            llx.StringData(m.UUID),
			"manufacturer":    llx.StringData(m.Manufacturer),
			"model":           llx.StringData(m.Model),
			"firmwareVersion": llx.StringData(m.FirmwareVersion),
			"managerType":     llx.StringData(string(m.ManagerType)),
			"powerState":      llx.StringData(string(m.PowerState)),
			"dateTime":        llx.StringData(m.DateTime),
		})
		if err != nil {
			return nil, err
		}
		mqlManager := o.(*mqlRedfishManager)
		mqlManager.mgr = m
		res = append(res, mqlManager)
	}
	return res, nil
}

func (r *mqlRedfish) chassis() ([]any, error) {
	svc := redfishConn(r.MqlRuntime).Client().Service
	chassis, err := svc.Chassis()
	if err != nil {
		return nil, err
	}

	res := make([]any, 0, len(chassis))
	for _, c := range chassis {
		o, err := CreateResource(r.MqlRuntime, "redfish.chassisEnclosure", map[string]*llx.RawData{
			"__id":         llx.StringData(c.ODataID),
			"name":         llx.StringData(c.Name),
			"chassisType":  llx.StringData(string(c.ChassisType)),
			"manufacturer": llx.StringData(c.Manufacturer),
			"model":        llx.StringData(c.Model),
			"serialNumber": llx.StringData(c.SerialNumber),
			"sku":          llx.StringData(c.SKU),
			"powerState":   llx.StringData(string(c.PowerState)),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, o)
	}
	return res, nil
}

func (r *mqlRedfish) accounts() ([]any, error) {
	svc := redfishConn(r.MqlRuntime).Client().Service
	accountService, err := svc.AccountService()
	if err != nil {
		return nil, err
	}
	accounts, err := accountService.Accounts()
	if err != nil {
		return nil, err
	}

	res := make([]any, 0, len(accounts))
	for _, a := range accounts {
		accountTypes := make([]any, 0, len(a.AccountTypes))
		for _, t := range a.AccountTypes {
			accountTypes = append(accountTypes, string(t))
		}

		o, err := CreateResource(r.MqlRuntime, "redfish.account", map[string]*llx.RawData{
			"__id":         llx.StringData(a.ODataID),
			"userName":     llx.StringData(a.UserName),
			"roleId":       llx.StringData(a.RoleID),
			"enabled":      llx.BoolData(a.Enabled),
			"locked":       llx.BoolData(a.Locked),
			"accountTypes": llx.ArrayData(accountTypes, types.String),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, o)
	}
	return res, nil
}

func (r *mqlRedfish) firmware() ([]any, error) {
	svc := redfishConn(r.MqlRuntime).Client().Service
	updateService, err := svc.UpdateService()
	if err != nil {
		return nil, err
	}
	inventory, err := updateService.FirmwareInventory()
	if err != nil {
		return nil, err
	}

	res := make([]any, 0, len(inventory))
	for _, sw := range inventory {
		o, err := CreateResource(r.MqlRuntime, "redfish.softwareInventory", map[string]*llx.RawData{
			"__id":                   llx.StringData(sw.ODataID),
			"name":                   llx.StringData(sw.Name),
			"version":                llx.StringData(sw.Version),
			"manufacturer":           llx.StringData(sw.Manufacturer),
			"softwareId":             llx.StringData(sw.SoftwareID),
			"updateable":             llx.BoolData(sw.Updateable),
			"releaseDate":            llx.StringData(sw.ReleaseDate),
			"lowestSupportedVersion": llx.StringData(sw.LowestSupportedVersion),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, o)
	}
	return res, nil
}

// mqlRedfishSystemInternal caches the source system so sub-collections can be
// resolved without re-fetching the parent.
type mqlRedfishSystemInternal struct {
	sys *schemas.ComputerSystem
}

func (r *mqlRedfishSystem) processors() ([]any, error) {
	if r.sys == nil {
		return nil, nil
	}
	processors, err := r.sys.Processors()
	if err != nil {
		return nil, err
	}

	res := make([]any, 0, len(processors))
	for _, p := range processors {
		o, err := CreateResource(r.MqlRuntime, "redfish.processor", map[string]*llx.RawData{
			"__id":           llx.StringData(p.ODataID),
			"manufacturer":   llx.StringData(p.Manufacturer),
			"model":          llx.StringData(p.Model),
			"processorType":  llx.StringData(string(p.ProcessorType)),
			"instructionSet": llx.StringData(string(p.InstructionSet)),
			"socket":         llx.StringData(p.Socket),
			"totalCores":     llx.IntDataPtr(p.TotalCores),
			"totalThreads":   llx.IntDataPtr(p.TotalThreads),
			"maxSpeedMHz":    llx.IntDataPtr(p.MaxSpeedMHz),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, o)
	}
	return res, nil
}

func (r *mqlRedfishSystem) memory() ([]any, error) {
	if r.sys == nil {
		return nil, nil
	}
	memory, err := r.sys.Memory()
	if err != nil {
		return nil, err
	}

	res := make([]any, 0, len(memory))
	for _, m := range memory {
		o, err := CreateResource(r.MqlRuntime, "redfish.memory", map[string]*llx.RawData{
			"__id":              llx.StringData(m.ODataID),
			"capacityMiB":       llx.IntDataPtr(m.CapacityMiB),
			"memoryDeviceType":  llx.StringData(string(m.MemoryDeviceType)),
			"operatingSpeedMhz": llx.IntDataPtr(m.OperatingSpeedMhz),
			"manufacturer":      llx.StringData(m.Manufacturer),
			"partNumber":        llx.StringData(m.PartNumber),
			"serialNumber":      llx.StringData(m.SerialNumber),
			"rankCount":         llx.IntDataPtr(m.RankCount),
			"dataWidthBits":     llx.IntDataPtr(m.DataWidthBits),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, o)
	}
	return res, nil
}

func (r *mqlRedfishSystem) ethernetInterfaces() ([]any, error) {
	if r.sys == nil {
		return nil, nil
	}
	interfaces, err := r.sys.EthernetInterfaces()
	if err != nil {
		return nil, err
	}

	res := make([]any, 0, len(interfaces))
	for _, eth := range interfaces {
		o, err := CreateResource(r.MqlRuntime, "redfish.ethernetInterface", map[string]*llx.RawData{
			"__id":                llx.StringData(eth.ODataID),
			"macAddress":          llx.StringData(eth.MACAddress),
			"permanentMACAddress": llx.StringData(eth.PermanentMACAddress),
			"speedMbps":           llx.IntDataPtr(eth.SpeedMbps),
			"fullDuplex":          llx.BoolData(eth.FullDuplex),
			"linkStatus":          llx.StringData(string(eth.LinkStatus)),
			"interfaceEnabled":    llx.BoolData(eth.InterfaceEnabled),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, o)
	}
	return res, nil
}

// mqlRedfishManagerInternal caches the source manager for computed fields.
type mqlRedfishManagerInternal struct {
	mgr *schemas.Manager
}

func (r *mqlRedfishManager) networkProtocol() (any, error) {
	if r.mgr == nil {
		r.NetworkProtocol.State = plugin.StateIsNull | plugin.StateIsSet
		return nil, nil
	}
	np, err := r.mgr.NetworkProtocol()
	if err != nil {
		return nil, err
	}
	if np == nil {
		r.NetworkProtocol.State = plugin.StateIsNull | plugin.StateIsSet
		return nil, nil
	}

	result := map[string]any{
		"hostName": np.HostName,
		"fqdn":     np.FQDN,
		"http":     map[string]any{"enabled": np.HTTP.ProtocolEnabled, "port": uintPtrToAny(np.HTTP.Port)},
		"https":    map[string]any{"enabled": np.HTTPS.ProtocolEnabled, "port": uintPtrToAny(np.HTTPS.Port)},
		"ssh":      map[string]any{"enabled": np.SSH.ProtocolEnabled, "port": uintPtrToAny(np.SSH.Port)},
		"ipmi":     map[string]any{"enabled": np.IPMI.ProtocolEnabled, "port": uintPtrToAny(np.IPMI.Port)},
		"snmp":     map[string]any{"enabled": np.SNMP.ProtocolEnabled, "port": uintPtrToAny(np.SNMP.Port)},
		"kvmip":    map[string]any{"enabled": np.KVMIP.ProtocolEnabled, "port": uintPtrToAny(np.KVMIP.Port)},
		"virtualMedia": map[string]any{
			"enabled": np.VirtualMedia.ProtocolEnabled,
			"port":    uintPtrToAny(np.VirtualMedia.Port),
		},
	}
	return result, nil
}

// hpeManagerOem is the subset of the HPE manager OEM block we surface. Older
// iLO firmware nests data under "Hp" rather than "Hpe".
type hpeManagerOem struct {
	Hpe *hpeLicenseWrap `json:"Hpe"`
	Hp  *hpeLicenseWrap `json:"Hp"`
}

type hpeLicenseWrap struct {
	License struct {
		LicenseType   string `json:"LicenseType"`
		LicenseString string `json:"LicenseString"`
	} `json:"License"`
}

// mqlRedfishHpeInternal caches the parsed HPE OEM license data.
type mqlRedfishHpeInternal struct {
	once               sync.Once
	cachedLicenseType  string
	cachedLicenseLabel string
}

func (r *mqlRedfishHpe) id() (string, error) {
	return "redfish.hpe", nil
}

func (r *mqlRedfishHpe) load() {
	r.once.Do(func() {
		managers, err := redfishConn(r.MqlRuntime).Managers()
		if err != nil {
			log.Warn().Err(err).Msg("redfish: could not list managers for HPE OEM detection")
			return
		}
		for _, m := range managers {
			if len(m.OEM) == 0 {
				continue
			}
			var oem hpeManagerOem
			if err := json.Unmarshal(m.OEM, &oem); err != nil {
				log.Debug().Err(err).Msg("redfish: could not parse HPE manager OEM block")
				continue
			}
			wrap := oem.Hpe
			if wrap == nil {
				wrap = oem.Hp
			}
			if wrap == nil {
				continue
			}
			r.cachedLicenseType = wrap.License.LicenseType
			r.cachedLicenseLabel = wrap.License.LicenseString
			return
		}
	})
}

func (r *mqlRedfishHpe) licenseType() (string, error) {
	r.load()
	return r.cachedLicenseType, nil
}

func (r *mqlRedfishHpe) licenseLabel() (string, error) {
	r.load()
	return r.cachedLicenseLabel, nil
}

// dellSystemOem is the subset of the Dell system OEM block we surface.
type dellSystemOem struct {
	Dell struct {
		DellSystem struct {
			SystemGeneration string `json:"SystemGeneration"`
			SystemID         int64  `json:"SystemID"`
			BIOSReleaseDate  string `json:"BIOSReleaseDate"`
		} `json:"DellSystem"`
	} `json:"Dell"`
}

// mqlRedfishDellInternal caches the parsed Dell OEM system data.
type mqlRedfishDellInternal struct {
	once                  sync.Once
	cachedGeneration      string
	cachedSystemID        int64
	cachedBiosReleaseDate string
}

func (r *mqlRedfishDell) id() (string, error) {
	return "redfish.dell", nil
}

func (r *mqlRedfishDell) load() {
	r.once.Do(func() {
		systems, err := redfishConn(r.MqlRuntime).Systems()
		if err != nil {
			log.Warn().Err(err).Msg("redfish: could not list systems for Dell OEM detection")
			return
		}
		for _, s := range systems {
			if len(s.OEM) == 0 {
				continue
			}
			var oem dellSystemOem
			if err := json.Unmarshal(s.OEM, &oem); err != nil {
				log.Debug().Err(err).Msg("redfish: could not parse Dell system OEM block")
				continue
			}
			if oem.Dell.DellSystem.SystemGeneration == "" && oem.Dell.DellSystem.SystemID == 0 {
				continue
			}
			r.cachedGeneration = oem.Dell.DellSystem.SystemGeneration
			r.cachedSystemID = oem.Dell.DellSystem.SystemID
			r.cachedBiosReleaseDate = oem.Dell.DellSystem.BIOSReleaseDate
			return
		}
	})
}

func (r *mqlRedfishDell) generation() (string, error) {
	r.load()
	return r.cachedGeneration, nil
}

func (r *mqlRedfishDell) systemID() (int64, error) {
	r.load()
	return r.cachedSystemID, nil
}

func (r *mqlRedfishDell) biosReleaseDate() (string, error) {
	r.load()
	return r.cachedBiosReleaseDate, nil
}

// mqlRedfishSupermicroInternal caches the Supermicro OEM data, which lives in
// linked sub-resources (license manager, system lockdown) rather than inline
// in the manager's OEM block.
type mqlRedfishSupermicroInternal struct {
	once           sync.Once
	cachedLicenses []any
	cachedLockdown bool
}

func (r *mqlRedfishSupermicro) id() (string, error) {
	return "redfish.supermicro", nil
}

func (r *mqlRedfishSupermicro) load() {
	r.once.Do(func() {
		managers, err := redfishConn(r.MqlRuntime).Managers()
		if err != nil {
			log.Warn().Err(err).Msg("redfish: could not list managers for Supermicro OEM detection")
			return
		}
		for _, m := range managers {
			smcManager, err := smc.FromManager(m)
			if err != nil {
				continue
			}

			found := false
			if lm, err := smcManager.LicenseManager(); err == nil && lm != nil {
				if ql, err := lm.QueryLicense(); err == nil && ql != nil {
					licenses := make([]any, 0, len(ql.Licenses))
					for _, license := range ql.Licenses {
						licenses = append(licenses, license)
					}
					r.cachedLicenses = licenses
					found = true
				}
			}
			if sl, err := smcManager.SysLockdown(); err == nil && sl != nil {
				r.cachedLockdown = sl.Enabled
				found = true
			}
			if found {
				return
			}
		}
	})
}

func (r *mqlRedfishSupermicro) licenses() ([]any, error) {
	r.load()
	return r.cachedLicenses, nil
}

func (r *mqlRedfishSupermicro) systemLockdownEnabled() (bool, error) {
	r.load()
	return r.cachedLockdown, nil
}
