// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"fmt"
	"sync"
	"time"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers/proxmox/connection"
)

type mqlProxmoxNodeInternal struct {
	nodeName      string
	statusFetched bool
	nodeStatus    *connection.NodeStatus
	statusErr     error
	lock          sync.Mutex
}

func nodeConn(r *mqlProxmoxNode) *connection.PveConnection {
	return r.MqlRuntime.Connection.(*connection.PveConnection)
}

func (r *mqlProxmoxNode) id() (string, error) {
	return "proxmox.node/" + r.Name.Data, nil
}

func (r *mqlProxmoxNode) ensureStatus() {
	if r.statusFetched {
		return
	}
	r.lock.Lock()
	defer r.lock.Unlock()
	if r.statusFetched {
		return
	}
	r.nodeStatus, r.statusErr = nodeConn(r).GetNodeStatus(r.Name.Data)
	r.statusFetched = true
}

func (r *mqlProxmoxNode) cpuModel() (string, error) {
	r.ensureStatus()
	if r.statusErr != nil { return "", r.statusErr }
	return r.nodeStatus.CPUInfo.Model, nil
}

func (r *mqlProxmoxNode) cpuSockets() (int64, error) {
	r.ensureStatus()
	if r.statusErr != nil { return 0, r.statusErr }
	return int64(r.nodeStatus.CPUInfo.Sockets), nil
}

func (r *mqlProxmoxNode) cpuCores() (int64, error) {
	r.ensureStatus()
	if r.statusErr != nil { return 0, r.statusErr }
	return int64(r.nodeStatus.CPUInfo.Cores), nil
}

func (r *mqlProxmoxNode) cpuUsage() (float64, error) {
	r.ensureStatus()
	if r.statusErr != nil { return 0, r.statusErr }
	return r.nodeStatus.CPU, nil
}

func (r *mqlProxmoxNode) memTotal() (int64, error) {
	r.ensureStatus()
	if r.statusErr != nil { return 0, r.statusErr }
	return r.nodeStatus.Memory.Total, nil
}

func (r *mqlProxmoxNode) memUsed() (int64, error) {
	r.ensureStatus()
	if r.statusErr != nil { return 0, r.statusErr }
	return r.nodeStatus.Memory.Used, nil
}

func (r *mqlProxmoxNode) memFree() (int64, error) {
	r.ensureStatus()
	if r.statusErr != nil { return 0, r.statusErr }
	return r.nodeStatus.Memory.Free, nil
}

func (r *mqlProxmoxNode) swapTotal() (int64, error) {
	r.ensureStatus()
	if r.statusErr != nil { return 0, r.statusErr }
	return r.nodeStatus.Swap.Total, nil
}

func (r *mqlProxmoxNode) swapUsed() (int64, error) {
	r.ensureStatus()
	if r.statusErr != nil { return 0, r.statusErr }
	return r.nodeStatus.Swap.Used, nil
}

func (r *mqlProxmoxNode) kernelVersion() (string, error) {
	r.ensureStatus()
	if r.statusErr != nil { return "", r.statusErr }
	return r.nodeStatus.KVersion, nil
}

func (r *mqlProxmoxNode) pveVersion() (string, error) {
	r.ensureStatus()
	if r.statusErr != nil { return "", r.statusErr }
	return r.nodeStatus.PVEVer, nil
}

func (r *mqlProxmoxNode) uptime() (int64, error) {
	r.ensureStatus()
	if r.statusErr != nil { return 0, r.statusErr }
	return r.nodeStatus.Uptime, nil
}

func (r *mqlProxmoxNode) networks() ([]any, error) {
	conn := nodeConn(r)
	ifaces, err := conn.GetNodeNetworks(r.Name.Data)
	if err != nil { return nil, err }
	list := make([]any, len(ifaces))
	for i, ifc := range ifaces {
		res, err := CreateResource(r.MqlRuntime, "proxmox.network", map[string]*llx.RawData{
			"iface":       llx.StringData(ifc.Iface),
			"type":        llx.StringData(ifc.Type),
			"active":      llx.BoolData(ifc.Active == 1),
			"method":      llx.StringData(ifc.Method),
			"address":     llx.StringData(ifc.Address),
			"netmask":     llx.StringData(ifc.Netmask),
			"gateway":     llx.StringData(ifc.Gateway),
			"bridgePorts": llx.StringData(ifc.BridgePorts),
			"cidr":        llx.StringData(ifc.CIDR),
			"autostart":   llx.BoolData(ifc.Autostart == 1),
			"comments":    llx.StringData(ifc.Comments),
		})
		if err != nil { return nil, err }
		res.(*mqlProxmoxNetwork).parentNode = r.Name.Data
		list[i] = res
	}
	return list, nil
}

func (r *mqlProxmoxNode) dns() (*mqlProxmoxDns, error) {
	conn := nodeConn(r)
	d, err := conn.GetNodeDNS(r.Name.Data)
	if err != nil { return nil, err }
	res, err := CreateResource(r.MqlRuntime, "proxmox.dns", map[string]*llx.RawData{
		"search": llx.StringData(d.Search),
		"dns1":   llx.StringData(d.DNS1),
		"dns2":   llx.StringData(d.DNS2),
		"dns3":   llx.StringData(d.DNS3),
	})
	if err != nil { return nil, err }
	mqlDns := res.(*mqlProxmoxDns)
	mqlDns.parentNode = r.Name.Data
	return mqlDns, nil
}

func (r *mqlProxmoxNode) services() ([]any, error) {
	conn := nodeConn(r)
	svcs, err := conn.GetNodeServices(r.Name.Data)
	if err != nil { return nil, err }
	list := make([]any, len(svcs))
	for i, s := range svcs {
		res, err := CreateResource(r.MqlRuntime, "proxmox.service", map[string]*llx.RawData{
			"name":          llx.StringData(s.Name),
			"state":         llx.StringData(s.State),
			"description":   llx.StringData(s.Description),
			"unitFileState": llx.StringData(s.UnitFileState),
		})
		if err != nil { return nil, err }
		res.(*mqlProxmoxService).parentNode = r.Name.Data
		list[i] = res
	}
	return list, nil
}

func (r *mqlProxmoxNode) timezone() (string, error) {
	conn := nodeConn(r)
	t, err := conn.GetNodeTime(r.Name.Data)
	if err != nil { return "", err }
	return t.Timezone, nil
}

func (r *mqlProxmoxNode) storages() ([]any, error) {
	conn := nodeConn(r)
	storages, err := conn.GetNodeStorage(r.Name.Data)
	if err != nil { return nil, err }
	return storageInfoToResources(r.MqlRuntime, storages)
}

func (r *mqlProxmoxNode) certificates() ([]any, error) {
	conn := nodeConn(r)
	certs, err := conn.GetNodeCertificates(r.Name.Data)
	if err != nil { return nil, err }
	list := make([]any, len(certs))
	for i, c := range certs {
		var san []any
		for _, s := range c.San { san = append(san, s) }
		res, err := CreateResource(r.MqlRuntime, "proxmox.certificate", map[string]*llx.RawData{
			"filename":      llx.StringData(c.Filename),
			"fingerprint":   llx.StringData(c.Fingerprint),
			"issuer":        llx.StringData(c.Issuer),
			"notAfter":      llx.TimeData(time.Unix(c.NotAfter, 0).UTC()),
			"notBefore":     llx.TimeData(time.Unix(c.NotBefore, 0).UTC()),
			"publicKeyBits": llx.IntData(int64(c.PublicKeyBits)),
			"publicKeyType": llx.StringData(c.PublicKeyType),
			"san":           llx.ArrayData(san, "\x02"),
			"subject":       llx.StringData(c.Subject),
		})
		if err != nil { return nil, err }
		res.(*mqlProxmoxCertificate).parentNode = r.Name.Data
		list[i] = res
	}
	return list, nil
}

func (r *mqlProxmoxNode) subscription() (*mqlProxmoxSubscription, error) {
	conn := nodeConn(r)
	sub, err := conn.GetNodeSubscription(r.Name.Data)
	if err != nil { return nil, err }
	res, err := CreateResource(r.MqlRuntime, "proxmox.subscription", map[string]*llx.RawData{
		"status":      llx.StringData(sub.Status),
		"serverId":    llx.StringData(sub.ServerID),
		"productName": llx.StringData(sub.ProductName),
		"regDate":     llx.StringData(sub.RegDate),
		"nextDueDate": llx.StringData(sub.NextDueDate),
		"level":       llx.StringData(sub.Level),
		"key":         llx.StringData(sub.Key),
	})
	if err != nil { return nil, err }
	mqlSub := res.(*mqlProxmoxSubscription)
	mqlSub.parentNode = r.Name.Data
	return mqlSub, nil
}

func (r *mqlProxmoxNode) repositories() ([]any, error) {
	conn := nodeConn(r)
	repoInfo, err := conn.GetNodeRepositories(r.Name.Data)
	if err != nil { return nil, err }
	var list []any
	idx := 0
	for _, file := range repoInfo.Files {
		for _, repo := range file.Repositories {
			repoID := fmt.Sprintf("%s:%d", file.Path, idx)
			name := repo.Comment
			if name == "" && len(repo.URIs) > 0 { name = repo.URIs[0] }
			var types, uris, suites, components []any
			for _, t := range repo.Types { types = append(types, t) }
			for _, u := range repo.URIs { uris = append(uris, u) }
			for _, s := range repo.Suites { suites = append(suites, s) }
			for _, c := range repo.Components { components = append(components, c) }
			res, err := CreateResource(r.MqlRuntime, "proxmox.repository", map[string]*llx.RawData{
				"id":         llx.StringData(repoID),
				"name":       llx.StringData(name),
				"enabled":    llx.BoolData(repo.Enabled),
				"types":      llx.ArrayData(types, "\x02"),
				"uris":       llx.ArrayData(uris, "\x02"),
				"suites":     llx.ArrayData(suites, "\x02"),
				"components": llx.ArrayData(components, "\x02"),
				"fileType":   llx.StringData(file.FileType),
			})
			if err != nil { return nil, err }
			res.(*mqlProxmoxRepository).parentNode = r.Name.Data
			list = append(list, res)
			idx++
		}
	}
	return list, nil
}

func (r *mqlProxmoxNode) updates() ([]any, error) {
	conn := nodeConn(r)
	updates, err := conn.GetNodeUpdates(r.Name.Data)
	if err != nil { return nil, err }
	list := make([]any, len(updates))
	for i, u := range updates {
		severity := "optional"
		if u.Priority == "important" || u.Priority == "required" { severity = "important" }
		if u.Priority == "standard" { severity = "recommended" }
		res, err := CreateResource(r.MqlRuntime, "proxmox.node.update", map[string]*llx.RawData{
			"package":          llx.StringData(u.Package),
			"installedVersion": llx.StringData(u.OldVersion),
			"newVersion":       llx.StringData(u.NewVersion),
			"severity":         llx.StringData(severity),
		})
		if err != nil { return nil, err }
		res.(*mqlProxmoxNodeUpdate).parentNode = r.Name.Data
		list[i] = res
	}
	return list, nil
}

func (r *mqlProxmoxNode) firewallRules() ([]any, error) {
	conn := nodeConn(r)
	rules, err := conn.GetNodeFirewallRules(r.Name.Data)
	if err != nil { return nil, err }
	return firewallRulesToResources(r.MqlRuntime, rules)
}

func (r *mqlProxmoxNode) vms() ([]any, error) {
	conn := nodeConn(r)
	vms, err := conn.GetNodeVMs(r.Name.Data)
	if err != nil { return nil, err }
	return vmInfoToResources(r.MqlRuntime, vms)
}
