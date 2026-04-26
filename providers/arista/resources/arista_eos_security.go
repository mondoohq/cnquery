// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"strconv"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/arista/resources/eos"
	"go.mondoo.com/mql/v13/types"
)

// fetchRunningConfig returns the EOS running-config, preferring the cached
// content on the existing arista.eos.runningConfig resource (which already
// handles double-checked locking) so we don't issue redundant
// `show running-config` calls when multiple security resources are queried.
func fetchRunningConfig(runtime *plugin.Runtime) (string, error) {
	rc, err := CreateResource(runtime, "arista.eos.runningConfig", map[string]*llx.RawData{})
	if err != nil {
		return "", err
	}
	return rc.(*mqlAristaEosRunningConfig).fetchContent(), nil
}

// stringSliceToAny converts []string to []any for llx.ArrayData.
func stringSliceToAny(s []string) []any {
	res := make([]any, len(s))
	for i, v := range s {
		res[i] = v
	}
	return res
}

// methodMapToDict converts map[string][]string to a map[string]any whose
// values are []any of strings, suitable for llx.MapData with the value type
// types.Array(types.String).
func methodMapToDict(m map[string][]string) map[string]any {
	out := make(map[string]any, len(m))
	for k, v := range m {
		out[k] = stringSliceToAny(v)
	}
	return out
}

// =====================================================================
// arista.eos.aaa
// =====================================================================

func (a *mqlAristaEosAaa) id() (string, error) {
	return "arista.eos.aaa", nil
}

func (a *mqlAristaEos) aaa() (*mqlAristaEosAaa, error) {
	rc, err := fetchRunningConfig(a.MqlRuntime)
	if err != nil {
		return nil, err
	}
	cfg := eos.ParseAaaConfig(rc)

	res, err := CreateResource(a.MqlRuntime, "arista.eos.aaa", map[string]*llx.RawData{
		"authenticationLogin":          llx.MapData(methodMapToDict(cfg.AuthenticationLogin), types.Array(types.String)),
		"authenticationEnable":         llx.MapData(methodMapToDict(cfg.AuthenticationEnable), types.Array(types.String)),
		"authorizationCommands":        llx.MapData(methodMapToDict(cfg.AuthorizationCommands), types.Array(types.String)),
		"authorizationExec":            llx.MapData(methodMapToDict(cfg.AuthorizationExec), types.Array(types.String)),
		"accountingCommands":           llx.MapData(methodMapToDict(cfg.AccountingCommands), types.Array(types.String)),
		"accountingExec":               llx.MapData(methodMapToDict(cfg.AccountingExec), types.Array(types.String)),
		"tacacsServers":                llx.ArrayData(stringSliceToAny(cfg.TacacsServers), types.String),
		"radiusServers":                llx.ArrayData(stringSliceToAny(cfg.RadiusServers), types.String),
		"defaultLoginPermitsLocalOnly": llx.BoolData(cfg.DefaultLoginPermitsLocalOnly),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlAristaEosAaa), nil
}

// =====================================================================
// arista.eos.sshSettings
// =====================================================================

func (a *mqlAristaEosSshSettings) id() (string, error) {
	return "arista.eos.sshSettings", nil
}

func (a *mqlAristaEos) sshSettings() (*mqlAristaEosSshSettings, error) {
	rc, err := fetchRunningConfig(a.MqlRuntime)
	if err != nil {
		return nil, err
	}
	s := eos.ParseSshSettings(rc)

	res, err := CreateResource(a.MqlRuntime, "arista.eos.sshSettings", map[string]*llx.RawData{
		"enabled":            llx.BoolData(s.Enabled),
		"protocolVersion":    llx.StringData(s.ProtocolVersion),
		"idleTimeout":        llx.IntData(int64(s.IdleTimeout)),
		"serverPort":         llx.IntData(int64(s.ServerPort)),
		"authenticationMode": llx.StringData(s.AuthenticationMode),
		"ciphers":            llx.ArrayData(stringSliceToAny(s.Ciphers), types.String),
		"keyExchange":        llx.ArrayData(stringSliceToAny(s.KeyExchange), types.String),
		"macs":               llx.ArrayData(stringSliceToAny(s.Macs), types.String),
		"hostkeyAlgorithms":  llx.ArrayData(stringSliceToAny(s.HostkeyAlgorithms), types.String),
		"fipsRestrictions":   llx.BoolData(s.FipsRestrictions),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlAristaEosSshSettings), nil
}

// =====================================================================
// arista.eos.snmpCommunity (list)
// =====================================================================

type mqlAristaEosSnmpCommunityInternal struct {
	cacheAcl string
}

func (a *mqlAristaEosSnmpCommunity) id() (string, error) {
	return "arista.eos.snmpCommunity/" + a.Name.Data, a.Name.Error
}

func (a *mqlAristaEos) snmpCommunities() ([]any, error) {
	rc, err := fetchRunningConfig(a.MqlRuntime)
	if err != nil {
		return nil, err
	}
	communities := eos.ParseSnmpCommunities(rc)
	res := make([]any, 0, len(communities))
	for _, c := range communities {
		mqlC, err := CreateResource(a.MqlRuntime, "arista.eos.snmpCommunity", map[string]*llx.RawData{
			"name":   llx.StringData(c.Name),
			"access": llx.StringData(c.Access),
			"acl":    llx.StringData(c.ACL),
			"ipv6":   llx.BoolData(c.IPv6),
		})
		if err != nil {
			return nil, err
		}
		mqlC.(*mqlAristaEosSnmpCommunity).cacheAcl = c.ACL
		res = append(res, mqlC)
	}
	return res, nil
}

func (a *mqlAristaEosSnmpCommunity) aclResource() (*mqlAristaEosAcl, error) {
	if a.cacheAcl == "" {
		a.AclResource.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	mqlAcl, err := NewResource(a.MqlRuntime, "arista.eos.acl", map[string]*llx.RawData{
		"name": llx.StringData(a.cacheAcl),
	})
	if err != nil {
		return nil, err
	}
	return mqlAcl.(*mqlAristaEosAcl), nil
}

// =====================================================================
// arista.eos.telnetService
// =====================================================================

func (a *mqlAristaEosTelnetService) id() (string, error) {
	return "arista.eos.telnetService", nil
}

func (a *mqlAristaEos) telnetService() (*mqlAristaEosTelnetService, error) {
	rc, err := fetchRunningConfig(a.MqlRuntime)
	if err != nil {
		return nil, err
	}
	t := eos.ParseTelnetService(rc)

	res, err := CreateResource(a.MqlRuntime, "arista.eos.telnetService", map[string]*llx.RawData{
		"configured":    llx.BoolData(t.Configured),
		"enabled":       llx.BoolData(t.Enabled),
		"idleTimeout":   llx.IntData(int64(t.IdleTimeout)),
		"sessionLimit":  llx.IntData(int64(t.SessionLimit)),
		"perHostLimit":  llx.IntData(int64(t.PerHostLimit)),
		"ipAccessGroup": llx.StringData(t.IPAccessGroup),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlAristaEosTelnetService), nil
}

// =====================================================================
// arista.eos.passwordPolicy
// =====================================================================

func (a *mqlAristaEosPasswordPolicy) id() (string, error) {
	return "arista.eos.passwordPolicy", nil
}

func (a *mqlAristaEos) passwordPolicy() (*mqlAristaEosPasswordPolicy, error) {
	rc, err := fetchRunningConfig(a.MqlRuntime)
	if err != nil {
		return nil, err
	}
	p := eos.ParsePasswordPolicy(rc)

	res, err := CreateResource(a.MqlRuntime, "arista.eos.passwordPolicy", map[string]*llx.RawData{
		"lockoutFailure":             llx.IntData(int64(p.LockoutFailure)),
		"lockoutWindowSeconds":       llx.IntData(int64(p.LockoutWindowSeconds)),
		"lockoutDurationSeconds":     llx.IntData(int64(p.LockoutDurationSeconds)),
		"allowNopasswordRemoteLogin": llx.BoolData(p.AllowNopasswordRemoteLogin),
		"logOnFailure":               llx.BoolData(p.LogOnFailure),
		"logOnSuccess":               llx.BoolData(p.LogOnSuccess),
		"policyName":                 llx.StringData(p.PolicyName),
		"minimumLength":              llx.IntData(int64(p.MinimumLength)),
		"minimumDigits":              llx.IntData(int64(p.MinimumDigits)),
		"minimumUppercase":           llx.IntData(int64(p.MinimumUppercase)),
		"minimumLowercase":           llx.IntData(int64(p.MinimumLowercase)),
		"minimumSpecial":             llx.IntData(int64(p.MinimumSpecial)),
		"maximumRepetitive":          llx.IntData(int64(p.MaximumRepetitive)),
		"maximumSequential":          llx.IntData(int64(p.MaximumSequential)),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlAristaEosPasswordPolicy), nil
}

// =====================================================================
// arista.eos.ntp.authKeys (list) + arista.eos.ntp.authenticationEnabled
// =====================================================================

func (a *mqlAristaEosNtpAuthKey) id() (string, error) {
	if a.Id.Error != nil {
		return "", a.Id.Error
	}
	return "arista.eos.ntpAuthKey/" + strconv.FormatInt(a.Id.Data, 10), nil
}

func (a *mqlAristaEosNtp) authKeys() ([]any, error) {
	rc, err := fetchRunningConfig(a.MqlRuntime)
	if err != nil {
		return nil, err
	}
	state := eos.ParseNtpAuth(rc)
	res := make([]any, 0, len(state.Keys))
	for _, k := range state.Keys {
		mqlK, err := CreateResource(a.MqlRuntime, "arista.eos.ntpAuthKey", map[string]*llx.RawData{
			"id":       llx.IntData(int64(k.ID)),
			"hashAlgo": llx.StringData(k.HashAlgo),
			"trusted":  llx.BoolData(k.Trusted),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlK)
	}
	return res, nil
}

func (a *mqlAristaEosNtp) authenticationEnabled() (bool, error) {
	rc, err := fetchRunningConfig(a.MqlRuntime)
	if err != nil {
		return false, err
	}
	state := eos.ParseNtpAuth(rc)
	return state.AuthenticationEnabled, nil
}

// =====================================================================
// arista.eos.controlPlanePolicer
// =====================================================================

type mqlAristaEosControlPlanePolicerInternal struct {
	cacheIpAccessGroup  string
	cacheIp6AccessGroup string
}

func (a *mqlAristaEosControlPlanePolicer) id() (string, error) {
	return "arista.eos.controlPlanePolicer", nil
}

func (a *mqlAristaEos) controlPlanePolicer() (*mqlAristaEosControlPlanePolicer, error) {
	rc, err := fetchRunningConfig(a.MqlRuntime)
	if err != nil {
		return nil, err
	}
	c := eos.ParseControlPlanePolicer(rc)
	res, err := CreateResource(a.MqlRuntime, "arista.eos.controlPlanePolicer", map[string]*llx.RawData{
		"configured":     llx.BoolData(c.Configured),
		"policyApplied":  llx.BoolData(c.PolicyApplied),
		"policyName":     llx.StringData(c.PolicyName),
		"ipAccessGroup":  llx.StringData(c.IPAccessGroup),
		"ip6AccessGroup": llx.StringData(c.IP6AccessGroup),
	})
	if err != nil {
		return nil, err
	}
	mqlPolicer := res.(*mqlAristaEosControlPlanePolicer)
	mqlPolicer.cacheIpAccessGroup = c.IPAccessGroup
	mqlPolicer.cacheIp6AccessGroup = c.IP6AccessGroup
	return mqlPolicer, nil
}

func (a *mqlAristaEosControlPlanePolicer) ipAccessGroupAcl() (*mqlAristaEosAcl, error) {
	if a.cacheIpAccessGroup == "" {
		a.IpAccessGroupAcl.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	mqlAcl, err := NewResource(a.MqlRuntime, "arista.eos.acl", map[string]*llx.RawData{
		"name": llx.StringData(a.cacheIpAccessGroup),
	})
	if err != nil {
		return nil, err
	}
	return mqlAcl.(*mqlAristaEosAcl), nil
}

func (a *mqlAristaEosControlPlanePolicer) ip6AccessGroupAcl() (*mqlAristaEosAcl, error) {
	if a.cacheIp6AccessGroup == "" {
		a.Ip6AccessGroupAcl.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	mqlAcl, err := NewResource(a.MqlRuntime, "arista.eos.acl", map[string]*llx.RawData{
		"name": llx.StringData(a.cacheIp6AccessGroup),
	})
	if err != nil {
		return nil, err
	}
	return mqlAcl.(*mqlAristaEosAcl), nil
}

// =====================================================================
// arista.eos.portSecurity (list)
// =====================================================================

func (a *mqlAristaEosPortSecurity) id() (string, error) {
	return "arista.eos.portSecurity/" + a.Interface.Data, a.Interface.Error
}

func (a *mqlAristaEos) portSecurity() ([]any, error) {
	rc, err := fetchRunningConfig(a.MqlRuntime)
	if err != nil {
		return nil, err
	}
	configs := eos.ParsePortSecurity(rc)
	res := make([]any, 0, len(configs))
	for _, c := range configs {
		mqlC, err := CreateResource(a.MqlRuntime, "arista.eos.portSecurity", map[string]*llx.RawData{
			"interface":           llx.StringData(c.Interface),
			"enabled":             llx.BoolData(c.Enabled),
			"maximumMacAddresses": llx.IntData(int64(c.MaximumMacAddresses)),
			"violationAction":     llx.StringData(c.ViolationAction),
			"stickyLearning":      llx.BoolData(c.StickyLearning),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlC)
	}
	return res, nil
}
