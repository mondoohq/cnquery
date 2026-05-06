// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/vmware/govmomi/vim25/mo"
	vimtypes "github.com/vmware/govmomi/vim25/types"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
	"go.mondoo.com/mql/v13/providers/vsphere/connection"
	"go.mondoo.com/mql/v13/providers/vsphere/resources/resourceclient"
	"go.mondoo.com/mql/v13/types"
)

type mqlVsphereHostInternal struct {
	host *mo.HostSystem
}

func (v *mqlVsphereHost) id() (string, error) {
	return v.Moid.Data, nil
}

// hostHardeningArgs extracts a few audit-relevant scalar fields from mo.HostSystem.
// firewallIncomingBlocked / firewallOutgoingBlocked reflect the host firewall's
// default policy; the firewall service itself is always running on ESXi.
// secureBootEnabled is reported by HostCapability.UefiSecureBoot, which requires
// vSphere 8.0.3+; on older hosts it's reported as false.
func hostHardeningArgs(hostInfo *mo.HostSystem) (lockdownMode string, firewallIncomingBlocked, firewallOutgoingBlocked, secureBootEnabled bool) {
	if hostInfo == nil {
		return
	}
	if hostInfo.Config != nil {
		lockdownMode = string(hostInfo.Config.LockdownMode)
		if fw := hostInfo.Config.Firewall; fw != nil {
			if fw.DefaultPolicy.IncomingBlocked != nil {
				firewallIncomingBlocked = *fw.DefaultPolicy.IncomingBlocked
			}
			if fw.DefaultPolicy.OutgoingBlocked != nil {
				firewallOutgoingBlocked = *fw.DefaultPolicy.OutgoingBlocked
			}
		}
	}
	if hostInfo.Capability != nil && hostInfo.Capability.UefiSecureBoot != nil {
		secureBootEnabled = *hostInfo.Capability.UefiSecureBoot
	}
	return
}

func initVsphereHost(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 0 {
		return args, nil, nil
	}

	conn := runtime.Connection.(*connection.VsphereConnection)
	h, hostInfo, err := esxiHostProperties(conn)
	if err != nil {
		return nil, nil, err
	}

	props, err := resourceclient.HostProperties(hostInfo)
	if err != nil {
		return nil, nil, err
	}

	var name string
	if hostInfo != nil {
		name = hostInfo.Name
	}

	lockdownMode, firewallIncomingBlocked, firewallOutgoingBlocked, secureBootEnabled := hostHardeningArgs(hostInfo)

	args["moid"] = llx.StringData(h.Reference().Encode())
	args["name"] = llx.StringData(name)
	args["properties"] = llx.DictData(props)
	args["inventoryPath"] = llx.StringData(h.InventoryPath)
	args["lockdownMode"] = llx.StringData(lockdownMode)
	args["firewallIncomingBlocked"] = llx.BoolData(firewallIncomingBlocked)
	args["firewallOutgoingBlocked"] = llx.BoolData(firewallOutgoingBlocked)
	args["secureBootEnabled"] = llx.BoolData(secureBootEnabled)

	return args, nil, nil
}

func (v *mqlVsphereHost) esxiClient() (*resourceclient.Esxi, error) {
	conn := v.MqlRuntime.Connection.(*connection.VsphereConnection)

	if v.InventoryPath.Error != nil {
		return nil, v.InventoryPath.Error
	}
	path := v.InventoryPath.Data

	return esxiClient(conn, path)
}

func (v *mqlVsphereHost) standardSwitch() ([]any, error) {
	esxiClient, err := v.esxiClient()
	if err != nil {
		return nil, err
	}

	vswitches, err := esxiClient.VswitchStandard()
	if err != nil {
		return nil, err
	}

	mqlVswitches := make([]any, len(vswitches))
	for i, s := range vswitches {
		name := s["Name"].(string)
		mqlVswitch, err := CreateResource(v.MqlRuntime, "vsphere.vswitch.standard", map[string]*llx.RawData{
			"__id":       llx.StringData(esxiClient.InventoryPath + "/" + name),
			"name":       llx.StringData(name),
			"properties": llx.DictData(s),
		})
		if err != nil {
			return nil, err
		}

		// store host inventory path, so that sub resources can use that to quickly query more
		r := mqlVswitch.(*mqlVsphereVswitchStandard)
		r.hostInventoryPath = esxiClient.InventoryPath
		r.parentResource = v

		mqlVswitches[i] = mqlVswitch
	}

	return mqlVswitches, nil
}

func (v *mqlVsphereHost) distributedSwitch() ([]any, error) {
	esxiClient, err := v.esxiClient()
	if err != nil {
		return nil, err
	}

	vswitches, err := esxiClient.VswitchDvs()
	if err != nil {
		return nil, err
	}

	mqlVswitches := make([]any, len(vswitches))
	for i, s := range vswitches {
		name := s["Name"].(string)
		mqlVswitch, err := CreateResource(v.MqlRuntime, "vsphere.vswitch.dvs", map[string]*llx.RawData{
			"__id":       llx.StringData(esxiClient.InventoryPath + "/" + name),
			"name":       llx.StringData(name),
			"properties": llx.DictData(s),
		})
		if err != nil {
			return nil, err
		}

		// store host inventory path, so that sub resources can use that to quickly query more
		r := mqlVswitch.(*mqlVsphereVswitchDvs)
		r.hostInventoryPath = esxiClient.InventoryPath
		r.parentResource = v

		mqlVswitches[i] = mqlVswitch
	}

	return mqlVswitches, nil
}

func (v *mqlVsphereHost) adapters() ([]any, error) {
	esxiClient, err := v.esxiClient()
	if err != nil {
		return nil, err
	}
	adapters, err := esxiClient.Adapters()
	if err != nil {
		return nil, err
	}

	pParams, err := esxiClient.ListNicPauseParams()
	if err != nil {
		return nil, err
	}

	pauseParams := map[string]map[string]any{}
	// sort pause params by nic
	for i, p := range pParams {
		nicName := pParams[i]["NIC"].(string)
		pauseParams[nicName] = p
	}

	mqlAdapters := make([]any, len(adapters))
	for i, a := range adapters {
		nicName := a["Name"].(string)
		pParams := pauseParams[nicName]

		mqlAdapter, err := CreateResource(v.MqlRuntime, "vsphere.vmnic", map[string]*llx.RawData{
			"__id":        llx.StringData(esxiClient.InventoryPath + "/" + nicName),
			"name":        llx.StringData(nicName),
			"properties":  llx.DictData(a),
			"pauseParams": llx.DictData(pParams),
		})
		if err != nil {
			return nil, err
		}

		// set inventory path
		r := mqlAdapter.(*mqlVsphereVmnic)
		r.hostInventoryPath = esxiClient.InventoryPath

		mqlAdapters[i] = mqlAdapter
	}

	return mqlAdapters, nil
}

func (v *mqlVsphereVmnic) id() (string, error) {
	return v.Name.Data, nil
}

type mqlVsphereVmnicInternal struct {
	hostInventoryPath string
}

func (v *mqlVsphereVmnic) esxiClient() (*resourceclient.Esxi, error) {
	conn := v.MqlRuntime.Connection.(*connection.VsphereConnection)
	return esxiClient(conn, v.hostInventoryPath)
}

func (v *mqlVsphereVmnic) details() (map[string]any, error) {
	name := v.Name.Data

	esxiClient, err := v.esxiClient()
	if err != nil {
		return nil, err
	}

	return esxiClient.ListNicDetails(name)
}

func (v *mqlVsphereHost) vmknics() ([]any, error) {
	esxiClient, err := v.esxiClient()
	if err != nil {
		return nil, err
	}
	vmknics, err := esxiClient.Vmknics()
	if err != nil {
		return nil, err
	}

	mqlVmknics := make([]any, len(vmknics))
	for i := range vmknics {
		entry := vmknics[i]
		nicName := entry.Properties["Name"].(string)
		mqlVswitch, err := CreateResource(v.MqlRuntime, "vsphere.vmknic", map[string]*llx.RawData{
			"__id":       llx.StringData(esxiClient.InventoryPath + "/" + nicName),
			"name":       llx.StringData(nicName),
			"properties": llx.DictData(entry.Properties),
			"ipv4":       llx.ArrayData(entry.Ipv4, types.Dict),
			"ipv6":       llx.ArrayData(entry.Ipv6, types.Dict),
			"tags":       llx.ArrayData(convert.SliceAnyToInterface(entry.Tags), types.String),
		})
		if err != nil {
			return nil, err
		}
		mqlVmknics[i] = mqlVswitch
	}

	return mqlVmknics, nil
}

func (v *mqlVsphereHost) packages() ([]any, error) {
	esxiClient, err := v.esxiClient()
	if err != nil {
		return nil, err
	}
	vibs, err := esxiClient.Vibs()
	if err != nil {
		return nil, err
	}

	mqlPackages := make([]any, len(vibs))
	for i := range vibs {
		vib := vibs[i]

		// parse timestamps in format "2020-07-16"
		format := "2006-01-02"
		var creationDate *time.Time
		parsedCreation, err := time.Parse(format, vib.CreationDate)
		if err != nil {
			return nil, errors.New("cannot parse vib creationDate: " + vib.CreationDate)
		}
		creationDate = &parsedCreation

		var installDate *time.Time
		parsedInstall, err := time.Parse(format, vib.InstallDate)
		if err != nil {
			return nil, errors.New("cannot parse vib installDate: " + vib.InstallDate)
		}
		installDate = &parsedInstall

		mqlVib, err := CreateResource(v.MqlRuntime, "esxi.vib", map[string]*llx.RawData{
			"id":              llx.StringData(vib.ID),
			"name":            llx.StringData(vib.Name),
			"acceptanceLevel": llx.StringData(vib.AcceptanceLevel),
			"creationDate":    llx.TimeDataPtr(creationDate),
			"installDate":     llx.TimeDataPtr(installDate),
			"status":          llx.StringData(vib.Status),
			"vendor":          llx.StringData(vib.Vendor),
			"version":         llx.StringData(vib.Version),
		})
		if err != nil {
			return nil, err
		}
		mqlPackages[i] = mqlVib
	}

	return mqlPackages, nil
}

func (v *mqlVsphereHost) acceptanceLevel() (string, error) {
	esxiClient, err := v.esxiClient()
	if err != nil {
		return "", err
	}
	return esxiClient.SoftwareAcceptance()
}

func (v *mqlVsphereHost) kernelModules() ([]any, error) {
	esxiClient, err := v.esxiClient()
	if err != nil {
		return nil, err
	}
	modules, err := esxiClient.KernelModules()
	if err != nil {
		return nil, err
	}

	mqlModules := make([]any, len(modules))
	for i, m := range modules {
		mqlModule, err := CreateResource(v.MqlRuntime, "esxi.kernelmodule", map[string]*llx.RawData{
			"__id":                 llx.StringData(esxiClient.InventoryPath + "/" + m.Module),
			"name":                 llx.StringData(m.Module),
			"modulefile":           llx.StringData(m.ModuleFile),
			"version":              llx.StringData(m.Version),
			"loaded":               llx.BoolData(m.Loaded),
			"license":              llx.StringData(m.License),
			"enabled":              llx.BoolData(m.Enabled),
			"signedStatus":         llx.StringData(m.SignedStatus),
			"signatureDigest":      llx.StringData(m.SignatureDigest),
			"signatureFingerprint": llx.StringData(m.SignatureFingerPrint),
			"vibAcceptanceLevel":   llx.StringData(m.VIBAcceptanceLevel),
		})
		if err != nil {
			return nil, err
		}
		mqlModules[i] = mqlModule
	}

	return mqlModules, nil
}

func (v *mqlVsphereHost) advancedSettings() (map[string]any, error) {
	conn := v.MqlRuntime.Connection.(*connection.VsphereConnection)
	vClient := getClientInstance(conn)

	if v.InventoryPath.Error != nil {
		return nil, v.InventoryPath.Error
	}
	path := v.InventoryPath.Data

	host, err := vClient.HostByInventoryPath(path)
	if err != nil {
		return nil, err
	}

	return resourceclient.HostOptions(host)
}

func (v *mqlVsphereHost) services() ([]any, error) {
	conn := v.MqlRuntime.Connection.(*connection.VsphereConnection)
	vClient := getClientInstance(conn)

	if v.InventoryPath.Error != nil {
		return nil, v.InventoryPath.Error
	}
	path := v.InventoryPath.Data

	host, err := vClient.HostByInventoryPath(path)
	if err != nil {
		return nil, err
	}

	services, err := resourceclient.HostServices(host)
	if err != nil {
		return nil, err
	}
	mqlServices := make([]any, len(services))
	for i, s := range services {
		mqlService, err := CreateResource(v.MqlRuntime, "esxi.service", map[string]*llx.RawData{
			"__id":     llx.StringData(path + "/" + s.Key),
			"key":      llx.StringData(s.Key),
			"label":    llx.StringData(s.Label),
			"required": llx.BoolData(s.Required),
			"running":  llx.BoolData(s.Running),
			"ruleset":  llx.ArrayData(convert.SliceAnyToInterface(s.Ruleset), types.String),
			"policy":   llx.StringData(s.Policy), // on, off, automatic
		})
		if err != nil {
			return nil, err
		}
		mqlServices[i] = mqlService
	}
	return mqlServices, nil
}

func (v *mqlVsphereHost) timezone() (*mqlEsxiTimezone, error) {
	conn := v.MqlRuntime.Connection.(*connection.VsphereConnection)
	vClient := getClientInstance(conn)

	if v.InventoryPath.Error != nil {
		return nil, v.InventoryPath.Error
	}
	path := v.InventoryPath.Data

	host, err := vClient.HostByInventoryPath(path)
	if err != nil {
		return nil, err
	}

	datetimeinfo, err := resourceclient.HostDateTime(host)
	if err != nil {
		return nil, err
	}

	if datetimeinfo == nil {
		return nil, errors.New("vsphere does not return HostDateTimeSystem timezone information")
	}

	mqlTimezone, err := CreateResource(v.MqlRuntime, "esxi.timezone", map[string]*llx.RawData{
		"__id":        llx.StringData(path + "/" + datetimeinfo.TimeZone.Key),
		"key":         llx.StringData(datetimeinfo.TimeZone.Key),
		"name":        llx.StringData(datetimeinfo.TimeZone.Name),
		"offset":      llx.IntData(int64(datetimeinfo.TimeZone.GmtOffset)),
		"description": llx.StringData(datetimeinfo.TimeZone.Description),
	})
	if err != nil {
		return nil, err
	}
	return mqlTimezone.(*mqlEsxiTimezone), nil
}

func (v *mqlVsphereHost) ntp() (*mqlEsxiNtpconfig, error) {
	conn := v.MqlRuntime.Connection.(*connection.VsphereConnection)
	vClient := getClientInstance(conn)

	if v.InventoryPath.Error != nil {
		return nil, v.InventoryPath.Error
	}
	path := v.InventoryPath.Data

	host, err := vClient.HostByInventoryPath(path)
	if err != nil {
		return nil, err
	}

	datetimeinfo, err := resourceclient.HostDateTime(host)
	if err != nil {
		return nil, err
	}

	var server []any
	var config []any

	if datetimeinfo != nil && datetimeinfo.NtpConfig != nil {
		server = convert.SliceAnyToInterface(datetimeinfo.NtpConfig.Server)
		config = convert.SliceAnyToInterface(datetimeinfo.NtpConfig.ConfigFile)
	}

	mqlNtpConfig, err := CreateResource(v.MqlRuntime, "esxi.ntpconfig", map[string]*llx.RawData{
		"id":     llx.StringData("ntp/" + host.InventoryPath),
		"server": llx.ArrayData(server, types.String),
		"config": llx.ArrayData(config, types.String),
	})
	if err != nil {
		return nil, err
	}

	return mqlNtpConfig.(*mqlEsxiNtpconfig), nil
}

func (v *mqlVsphereHost) snmp() (map[string]any, error) {
	esxiClient, err := v.esxiClient()
	if err != nil {
		return nil, err
	}
	return esxiClient.Snmp()
}

// firewallRulesets exposes the per-service ESXi firewall ruleset definitions
// from mo.HostSystem.Config.Firewall (already fetched via HostInfo). Each
// ruleset bundles a per-service rule list, an enabled flag, and the
// allowed-IP scope; the global default-deny posture lives on
// vsphere.host.firewallIncomingBlocked / firewallOutgoingBlocked.
func (v *mqlVsphereHost) firewallRulesets() ([]any, error) {
	if v.host == nil || v.host.Config == nil || v.host.Config.Firewall == nil {
		return []any{}, nil
	}
	hostPath := ""
	if v.InventoryPath.Error == nil {
		hostPath = v.InventoryPath.Data
	}

	mqlRulesets := make([]any, 0, len(v.host.Config.Firewall.Ruleset))
	for _, rs := range v.host.Config.Firewall.Ruleset {
		rsId := hostPath + "/firewall/" + rs.Key

		allIp := false
		allowedIPs := []any{}
		allowedNetworks := []any{}
		if rs.AllowedHosts != nil {
			allIp = rs.AllowedHosts.AllIp
			for _, ip := range rs.AllowedHosts.IpAddress {
				allowedIPs = append(allowedIPs, ip)
			}
			for _, n := range rs.AllowedHosts.IpNetwork {
				allowedNetworks = append(allowedNetworks, map[string]any{
					"network":      n.Network,
					"prefixLength": int64(n.PrefixLength),
				})
			}
		}

		mqlRules := make([]any, len(rs.Rule))
		for i, r := range rs.Rule {
			ruleId := rsId + "/" + strconv.Itoa(i)
			mqlRule, err := CreateResource(v.MqlRuntime, "esxi.firewallRule", map[string]*llx.RawData{
				"__id":      llx.StringData(ruleId),
				"id":        llx.StringData(ruleId),
				"port":      llx.IntData(int64(r.Port)),
				"endPort":   llx.IntData(int64(r.EndPort)),
				"direction": llx.StringData(string(r.Direction)),
				"portType":  llx.StringData(string(r.PortType)),
				"protocol":  llx.StringData(r.Protocol),
			})
			if err != nil {
				return nil, err
			}
			mqlRules[i] = mqlRule
		}

		mqlRs, err := CreateResource(v.MqlRuntime, "esxi.firewallRuleset", map[string]*llx.RawData{
			"__id":               llx.StringData(rsId),
			"id":                 llx.StringData(rsId),
			"key":                llx.StringData(rs.Key),
			"label":              llx.StringData(rs.Label),
			"enabled":            llx.BoolData(rs.Enabled),
			"required":           llx.BoolData(rs.Required),
			"service":            llx.StringData(rs.Service),
			"allIpsAllowed":      llx.BoolData(allIp),
			"allowedIpAddresses": llx.ArrayData(allowedIPs, types.String),
			"allowedIpNetworks":  llx.ArrayData(allowedNetworks, types.Dict),
			"rules":              llx.ArrayData(mqlRules, types.Resource("esxi.firewallRule")),
		})
		if err != nil {
			return nil, err
		}
		mqlRulesets = append(mqlRulesets, mqlRs)
	}
	return mqlRulesets, nil
}

// iscsiAdapters lists iSCSI host bus adapters from
// mo.HostSystem.Config.StorageDevice.HostBusAdapter, exposing the IQN and
// CHAP authentication posture. The CHAP secret itself is intentionally NOT
// exposed — only the policy and the username.
func (v *mqlVsphereHost) iscsiAdapters() ([]any, error) {
	if v.host == nil || v.host.Config == nil || v.host.Config.StorageDevice == nil {
		return []any{}, nil
	}
	hostPath := ""
	if v.InventoryPath.Error == nil {
		hostPath = v.InventoryPath.Data
	}

	mqlAdapters := []any{}
	for _, ba := range v.host.Config.StorageDevice.HostBusAdapter {
		hba, ok := ba.(*vimtypes.HostInternetScsiHba)
		if !ok || hba == nil {
			continue
		}
		auth := hba.AuthenticationProperties
		id := hostPath + "/iscsi/" + hba.Device
		mqlAdapter, err := CreateResource(v.MqlRuntime, "esxi.iscsiAdapter", map[string]*llx.RawData{
			"__id":                         llx.StringData(id),
			"id":                           llx.StringData(id),
			"device":                       llx.StringData(hba.Device),
			"iScsiName":                    llx.StringData(hba.IScsiName),
			"iScsiAlias":                   llx.StringData(hba.IScsiAlias),
			"chapAuthEnabled":              llx.BoolData(auth.ChapAuthEnabled),
			"chapAuthenticationType":       llx.StringData(auth.ChapAuthenticationType),
			"chapName":                     llx.StringData(auth.ChapName),
			"mutualChapAuthenticationType": llx.StringData(auth.MutualChapAuthenticationType),
			"mutualChapName":               llx.StringData(auth.MutualChapName),
		})
		if err != nil {
			return nil, err
		}
		mqlAdapters = append(mqlAdapters, mqlAdapter)
	}
	return mqlAdapters, nil
}

func (v *mqlVsphereHost) certificate() (*mqlEsxiCertificate, error) {
	conn := v.MqlRuntime.Connection.(*connection.VsphereConnection)
	vClient := getClientInstance(conn)

	if v.InventoryPath.Error != nil {
		return nil, v.InventoryPath.Error
	}
	path := v.InventoryPath.Data

	host, err := vClient.HostByInventoryPath(path)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	certMgr, err := host.ConfigManager().CertificateManager(ctx)
	if err != nil {
		return nil, err
	}
	// Older ESXi versions (and some host states) don't expose a
	// HostCertificateManager; CertificateManager returns (nil, nil).
	// Mark the field resolved-and-null so the runtime doesn't panic
	// or re-fetch.
	if certMgr == nil {
		v.Certificate.State = plugin.StateIsNull | plugin.StateIsSet
		return nil, nil
	}
	info, err := certMgr.CertificateInfo(ctx)
	if err != nil {
		return nil, err
	}

	id := "cert/" + path
	if info.Kind != "" {
		id += "/" + info.Kind
	}
	args := map[string]*llx.RawData{
		"__id":    llx.StringData(id),
		"id":      llx.StringData(id),
		"kind":    llx.StringData(info.Kind),
		"subject": llx.StringData(info.Subject),
		"issuer":  llx.StringData(info.Issuer),
		"status":  llx.StringData(info.Status),
	}
	if info.NotBefore != nil {
		args["notBefore"] = llx.TimeData(*info.NotBefore)
	} else {
		args["notBefore"] = llx.TimeData(time.Time{})
	}
	if info.NotAfter != nil {
		args["notAfter"] = llx.TimeData(*info.NotAfter)
	} else {
		args["notAfter"] = llx.TimeData(time.Time{})
	}

	mqlCert, err := CreateResource(v.MqlRuntime, "esxi.certificate", args)
	if err != nil {
		return nil, err
	}
	return mqlCert.(*mqlEsxiCertificate), nil
}
