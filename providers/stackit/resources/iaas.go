// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"github.com/stackitcloud/stackit-sdk-go/services/iaas"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/types"
)

// ------------------------- servers -------------------------

func (r *mqlStackit) servers() ([]any, error) {
	c := conn(r.MqlRuntime)
	client, err := c.IaaS()
	if err != nil {
		return nil, err
	}
	// Details(true) so the response includes nics, security groups, and volumes;
	// without it the API returns servers with those fields empty.
	resp, err := client.ListServers(bgctx(), c.ProjectID(), c.Region()).Details(true).Execute()
	if err != nil {
		if isAccessDenied(err) {
			return []any{}, nil
		}
		return nil, err
	}
	items, _ := resp.GetItemsOk()
	out := make([]any, 0, len(items))
	for i := range items {
		res, err := buildServer(r.MqlRuntime, &items[i])
		if err != nil {
			return nil, err
		}
		out = append(out, res)
	}
	return out, nil
}

func buildServer(runtime *plugin.Runtime, s *iaas.Server) (plugin.Resource, error) {
	nics := []any{}
	if v, ok := s.GetNicsOk(); ok {
		for _, n := range v {
			allowed := []any{}
			for _, a := range n.GetAllowedAddresses() {
				if a.String != nil {
					allowed = append(allowed, *a.String)
				}
			}
			nics = append(nics, map[string]any{
				"nicId":            n.GetNicId(),
				"networkId":        n.GetNetworkId(),
				"networkName":      n.GetNetworkName(),
				"ipv4":             n.GetIpv4(),
				"ipv6":             n.GetIpv6(),
				"mac":              n.GetMac(),
				"securityGroups":   strSlice(n.GetSecurityGroups()),
				"allowedAddresses": allowed,
				"publicIp":         n.GetPublicIp(),
				"nicSecurity":      n.GetNicSecurity(),
			})
		}
	}

	createdAt, ok1 := s.GetCreatedAtOk()
	launchedAt, ok2 := s.GetLaunchedAtOk()
	updatedAt, ok3 := s.GetUpdatedAtOk()

	vtpmEnabled := false
	if v, ok := s.GetVtpmOk(); ok {
		vtpmEnabled = v.GetEnabled()
	}

	args := map[string]*llx.RawData{
		"id":                  llx.StringData(s.GetId()),
		"name":                llx.StringData(s.GetName()),
		"status":              llx.StringData(s.GetStatus()),
		"powerStatus":         llx.StringData(s.GetPowerStatus()),
		"machineType":         llx.StringData(s.GetMachineType()),
		"availabilityZone":    llx.StringData(s.GetAvailabilityZone()),
		"createdAt":           llx.TimeDataPtr(timeOrNil(createdAt, ok1)),
		"launchedAt":          llx.TimeDataPtr(timeOrNil(launchedAt, ok2)),
		"updatedAt":           llx.TimeDataPtr(timeOrNil(updatedAt, ok3)),
		"errorMessage":        llx.StringData(s.GetErrorMessage()),
		"configDrive":         llx.BoolData(s.GetConfigDrive()),
		"vtpmEnabled":         llx.BoolData(vtpmEnabled),
		"keypairName":         llx.StringData(s.GetKeypairName()),
		"imageId":             llx.StringData(s.GetImageId()),
		"volumeIds":           strSliceData(s.GetVolumes()),
		"securityGroupIds":    strSliceData(s.GetSecurityGroups()),
		"serviceAccountMails": strSliceData(s.GetServiceAccountMails()),
		"nics":                llx.ArrayData(nics, types.Dict),
		"userData":            llx.StringData(string(s.GetUserData())),
		"labels":              labelData(s.GetLabels()),
		"metadata":            metadataData(s.GetMetadata()),
	}
	return CreateResource(runtime, "stackit.server", args)
}

func (r *mqlStackitServer) id() (string, error) {
	return "stackit.server/" + r.Id.Data, nil
}

func initStackitServer(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	id, ok := idArg(args, "id")
	if !ok {
		id, ok = conn(runtime).AssetObjectID("compute")
	}
	if !ok {
		return args, nil, nil
	}
	c := conn(runtime)
	client, err := c.IaaS()
	if err != nil {
		return nil, nil, err
	}
	s, err := client.GetServer(bgctx(), c.ProjectID(), c.Region(), id).Details(true).Execute()
	if err != nil {
		return nil, nil, err
	}
	res, err := buildServer(runtime, s)
	if err != nil {
		return nil, nil, err
	}
	return nil, res, nil
}

func (r *mqlStackitServer) image() (*mqlStackitImage, error) {
	if r.ImageId.Data == "" {
		return markNull[mqlStackitImage](&r.Image)
	}
	res, err := NewResource(r.MqlRuntime, "stackit.image", map[string]*llx.RawData{
		"id": llx.StringData(r.ImageId.Data),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlStackitImage), nil
}

func (r *mqlStackitServer) keyPair() (*mqlStackitKeyPair, error) {
	if r.KeypairName.Data == "" {
		return markNull[mqlStackitKeyPair](&r.KeyPair)
	}
	res, err := NewResource(r.MqlRuntime, "stackit.keyPair", map[string]*llx.RawData{
		"name": llx.StringData(r.KeypairName.Data),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlStackitKeyPair), nil
}

func (r *mqlStackitServer) volumes() ([]any, error) {
	out := make([]any, 0, len(r.VolumeIds.Data))
	for _, raw := range r.VolumeIds.Data {
		id, ok := raw.(string)
		if !ok || id == "" {
			continue
		}
		v, err := NewResource(r.MqlRuntime, "stackit.volume", map[string]*llx.RawData{
			"id": llx.StringData(id),
		})
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, nil
}

func (r *mqlStackitServer) securityGroups() ([]any, error) {
	// Collect security-group ids from the server-level list and, since STACKIT
	// attaches security groups per network interface (the server-level list is
	// usually empty), from each NIC as well. Deduplicate while preserving order.
	seen := map[string]struct{}{}
	ids := []string{}
	add := func(id string) {
		if id == "" {
			return
		}
		if _, dup := seen[id]; dup {
			return
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
	}
	for _, raw := range r.SecurityGroupIds.Data {
		if id, ok := raw.(string); ok {
			add(id)
		}
	}
	nics := r.GetNics()
	if nics.Error == nil {
		for _, raw := range nics.Data {
			nic, ok := raw.(map[string]any)
			if !ok {
				continue
			}
			sgs, ok := nic["securityGroups"].([]any)
			if !ok {
				continue
			}
			for _, s := range sgs {
				if id, ok := s.(string); ok {
					add(id)
				}
			}
		}
	}

	out := make([]any, 0, len(ids))
	for _, id := range ids {
		sg, err := NewResource(r.MqlRuntime, "stackit.securityGroup", map[string]*llx.RawData{
			"id": llx.StringData(id),
		})
		if err != nil {
			return nil, err
		}
		out = append(out, sg)
	}
	return out, nil
}

// ------------------------- volumes -------------------------

func (r *mqlStackit) volumes() ([]any, error) {
	c := conn(r.MqlRuntime)
	client, err := c.IaaS()
	if err != nil {
		return nil, err
	}
	resp, err := client.ListVolumesExecute(bgctx(), c.ProjectID(), c.Region())
	if err != nil {
		if isAccessDenied(err) {
			return []any{}, nil
		}
		return nil, err
	}
	items, _ := resp.GetItemsOk()
	out := make([]any, 0, len(items))
	for i := range items {
		res, err := buildVolume(r.MqlRuntime, &items[i])
		if err != nil {
			return nil, err
		}
		out = append(out, res)
	}
	return out, nil
}

func buildVolume(runtime *plugin.Runtime, v *iaas.Volume) (plugin.Resource, error) {
	var (
		imageID    string
		snapshotID string
	)
	if src, ok := v.GetSourceOk(); ok {
		switch src.GetType() {
		case "image":
			imageID = src.GetId()
		case "snapshot", "backup":
			snapshotID = src.GetId()
		}
	}
	serverID := v.GetServerId()

	var (
		kekKeyID      string
		kekKeyVersion int64
	)
	if ep, ok := v.GetEncryptionParametersOk(); ok {
		kekKeyID = ep.GetKekKeyId()
		kekKeyVersion = ep.GetKekKeyVersion()
	}

	createdAt, ok1 := v.GetCreatedAtOk()
	updatedAt, ok2 := v.GetUpdatedAtOk()

	args := map[string]*llx.RawData{
		"id":                   llx.StringData(v.GetId()),
		"name":                 llx.StringData(v.GetName()),
		"description":          llx.StringData(v.GetDescription()),
		"size":                 llx.IntData(int64(v.GetSize())),
		"status":               llx.StringData(v.GetStatus()),
		"availabilityZone":     llx.StringData(v.GetAvailabilityZone()),
		"performanceClass":     llx.StringData(v.GetPerformanceClass()),
		"bootable":             llx.BoolData(v.GetBootable()),
		"imageId":              llx.StringData(imageID),
		"sourceSnapshotId":     llx.StringData(snapshotID),
		"serverId":             llx.StringData(serverID),
		"encrypted":            llx.BoolData(v.GetEncrypted()),
		"encryptionKeyId":      llx.StringData(kekKeyID),
		"encryptionKeyVersion": llx.IntData(kekKeyVersion),
		"createdAt":            llx.TimeDataPtr(timeOrNil(createdAt, ok1)),
		"updatedAt":            llx.TimeDataPtr(timeOrNil(updatedAt, ok2)),
		"labels":               labelData(v.GetLabels()),
	}
	return CreateResource(runtime, "stackit.volume", args)
}

func (r *mqlStackitVolume) id() (string, error) {
	return "stackit.volume/" + r.Id.Data, nil
}

func initStackitVolume(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	id, ok := idArg(args, "id")
	if !ok {
		return args, nil, nil
	}
	c := conn(runtime)
	client, err := c.IaaS()
	if err != nil {
		return nil, nil, err
	}
	v, err := client.GetVolumeExecute(bgctx(), c.ProjectID(), c.Region(), id)
	if err != nil {
		return nil, nil, err
	}
	res, err := buildVolume(runtime, v)
	if err != nil {
		return nil, nil, err
	}
	return nil, res, nil
}

func (r *mqlStackitVolume) image() (*mqlStackitImage, error) {
	if r.ImageId.Data == "" {
		return markNull[mqlStackitImage](&r.Image)
	}
	res, err := NewResource(r.MqlRuntime, "stackit.image", map[string]*llx.RawData{
		"id": llx.StringData(r.ImageId.Data),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlStackitImage), nil
}

func (r *mqlStackitVolume) server() (*mqlStackitServer, error) {
	if r.ServerId.Data == "" {
		return markNull[mqlStackitServer](&r.Server)
	}
	res, err := NewResource(r.MqlRuntime, "stackit.server", map[string]*llx.RawData{
		"id": llx.StringData(r.ServerId.Data),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlStackitServer), nil
}

// ------------------------- snapshots -------------------------

func (r *mqlStackit) snapshots() ([]any, error) {
	c := conn(r.MqlRuntime)
	client, err := c.IaaS()
	if err != nil {
		return nil, err
	}
	resp, err := client.ListSnapshotsInProjectExecute(bgctx(), c.ProjectID(), c.Region())
	if err != nil {
		if isAccessDenied(err) {
			return []any{}, nil
		}
		return nil, err
	}
	items, _ := resp.GetItemsOk()
	out := make([]any, 0, len(items))
	for i := range items {
		res, err := buildSnapshot(r.MqlRuntime, &items[i])
		if err != nil {
			return nil, err
		}
		out = append(out, res)
	}
	return out, nil
}

func buildSnapshot(runtime *plugin.Runtime, s *iaas.Snapshot) (plugin.Resource, error) {
	createdAt, ok1 := s.GetCreatedAtOk()
	updatedAt, ok2 := s.GetUpdatedAtOk()
	args := map[string]*llx.RawData{
		"id":               llx.StringData(s.GetId()),
		"name":             llx.StringData(s.GetName()),
		"status":           llx.StringData(s.GetStatus()),
		"size":             llx.IntData(int64(s.GetSize())),
		"availabilityZone": llx.StringData(s.GetAvailabilityZone()),
		"volumeId":         llx.StringData(s.GetVolumeId()),
		"createdAt":        llx.TimeDataPtr(timeOrNil(createdAt, ok1)),
		"updatedAt":        llx.TimeDataPtr(timeOrNil(updatedAt, ok2)),
		"labels":           labelData(s.GetLabels()),
	}
	return CreateResource(runtime, "stackit.snapshot", args)
}

func (r *mqlStackitSnapshot) id() (string, error) {
	return "stackit.snapshot/" + r.Id.Data, nil
}

func initStackitSnapshot(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	id, ok := idArg(args, "id")
	if !ok {
		return args, nil, nil
	}
	c := conn(runtime)
	client, err := c.IaaS()
	if err != nil {
		return nil, nil, err
	}
	s, err := client.GetSnapshotExecute(bgctx(), c.ProjectID(), c.Region(), id)
	if err != nil {
		return nil, nil, err
	}
	res, err := buildSnapshot(runtime, s)
	if err != nil {
		return nil, nil, err
	}
	return nil, res, nil
}

func (r *mqlStackitSnapshot) volume() (*mqlStackitVolume, error) {
	if r.VolumeId.Data == "" {
		return markNull[mqlStackitVolume](&r.Volume)
	}
	res, err := NewResource(r.MqlRuntime, "stackit.volume", map[string]*llx.RawData{
		"id": llx.StringData(r.VolumeId.Data),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlStackitVolume), nil
}

// ------------------------- images -------------------------

func (r *mqlStackit) images() ([]any, error) {
	c := conn(r.MqlRuntime)
	client, err := c.IaaS()
	if err != nil {
		return nil, err
	}
	resp, err := client.ListImagesExecute(bgctx(), c.ProjectID(), c.Region())
	if err != nil {
		if isAccessDenied(err) {
			return []any{}, nil
		}
		return nil, err
	}
	items, _ := resp.GetItemsOk()
	out := make([]any, 0, len(items))
	for i := range items {
		res, err := buildImage(r.MqlRuntime, &items[i])
		if err != nil {
			return nil, err
		}
		out = append(out, res)
	}
	return out, nil
}

func buildImage(runtime *plugin.Runtime, img *iaas.Image) (plugin.Resource, error) {
	var checksum map[string]any
	if cs, ok := img.GetChecksumOk(); ok {
		checksum = map[string]any{
			"algorithm": cs.GetAlgorithm(),
			"digest":    cs.GetDigest(),
		}
	}
	var config map[string]any
	if cfg, ok := img.GetConfigOk(); ok {
		config = map[string]any{
			"bootMenu":               cfg.GetBootMenu(),
			"cdromBus":               ptrStr(cfg.GetCdromBus()),
			"diskBus":                ptrStr(cfg.GetDiskBus()),
			"nicModel":               ptrStr(cfg.GetNicModel()),
			"operatingSystem":        cfg.GetOperatingSystem(),
			"operatingSystemDistro":  ptrStr(cfg.GetOperatingSystemDistro()),
			"operatingSystemVersion": ptrStr(cfg.GetOperatingSystemVersion()),
			"rescueBus":              ptrStr(cfg.GetRescueBus()),
			"rescueDevice":           ptrStr(cfg.GetRescueDevice()),
			"secureBoot":             cfg.GetSecureBoot(),
			"uefi":                   cfg.GetUefi(),
			"videoModel":             ptrStr(cfg.GetVideoModel()),
			"virtioScsi":             cfg.GetVirtioScsi(),
		}
	}
	createdAt, ok1 := img.GetCreatedAtOk()
	updatedAt, ok2 := img.GetUpdatedAtOk()

	args := map[string]*llx.RawData{
		"id":          llx.StringData(img.GetId()),
		"name":        llx.StringData(img.GetName()),
		"status":      llx.StringData(img.GetStatus()),
		"diskFormat":  llx.StringData(img.GetDiskFormat()),
		"minDiskSize": llx.IntData(int64(img.GetMinDiskSize())),
		"minRam":      llx.IntData(int64(img.GetMinRam())),
		"protected":   llx.BoolData(img.GetProtected()),
		"owner":       llx.StringData(img.GetOwner()),
		"scope":       llx.StringData(img.GetScope()),
		"checksum":    llx.DictData(checksum),
		"config":      llx.DictData(config),
		"createdAt":   llx.TimeDataPtr(timeOrNil(createdAt, ok1)),
		"updatedAt":   llx.TimeDataPtr(timeOrNil(updatedAt, ok2)),
		"labels":      labelData(img.GetLabels()),
	}
	return CreateResource(runtime, "stackit.image", args)
}

func (r *mqlStackitImage) id() (string, error) {
	return "stackit.image/" + r.Id.Data, nil
}

func initStackitImage(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	id, ok := idArg(args, "id")
	if !ok {
		return args, nil, nil
	}
	c := conn(runtime)
	client, err := c.IaaS()
	if err != nil {
		return nil, nil, err
	}
	img, err := client.GetImageExecute(bgctx(), c.ProjectID(), c.Region(), id)
	if err != nil {
		return nil, nil, err
	}
	res, err := buildImage(runtime, img)
	if err != nil {
		return nil, nil, err
	}
	return nil, res, nil
}

// ------------------------- networks -------------------------

func (r *mqlStackit) networks() ([]any, error) {
	c := conn(r.MqlRuntime)
	client, err := c.IaaS()
	if err != nil {
		return nil, err
	}
	resp, err := client.ListNetworksExecute(bgctx(), c.ProjectID(), c.Region())
	if err != nil {
		if isAccessDenied(err) {
			return []any{}, nil
		}
		return nil, err
	}
	items, _ := resp.GetItemsOk()
	out := make([]any, 0, len(items))
	for i := range items {
		res, err := buildNetwork(r.MqlRuntime, &items[i])
		if err != nil {
			return nil, err
		}
		out = append(out, res)
	}
	return out, nil
}

func buildNetwork(runtime *plugin.Runtime, n *iaas.Network) (plugin.Resource, error) {
	var (
		ipv4Gateway    string
		ipv4Nameserv   []string
		ipv4Prefixes   []string
		ipv4PrefixSing string
		ipv6Gateway    string
		ipv6Nameserv   []string
		ipv6Prefixes   []string
		ipv6PrefixSing string
	)
	if ipv4 := n.GetIpv4(); !iaasNetworkIPv4Empty(ipv4) {
		ipv4Gateway = ptrStr(ipv4.GetGateway())
		ipv4Nameserv = ipv4.GetNameservers()
		ipv4Prefixes = ipv4.GetPrefixes()
		if len(ipv4Prefixes) > 0 {
			ipv4PrefixSing = ipv4Prefixes[0]
		}
	}
	if ipv6 := n.GetIpv6(); !iaasNetworkIPv6Empty(ipv6) {
		ipv6Gateway = ptrStr(ipv6.GetGateway())
		ipv6Nameserv = ipv6.GetNameservers()
		ipv6Prefixes = ipv6.GetPrefixes()
		if len(ipv6Prefixes) > 0 {
			ipv6PrefixSing = ipv6Prefixes[0]
		}
	}

	createdAt, okCreated := n.GetCreatedAtOk()

	args := map[string]*llx.RawData{
		"id":              llx.StringData(n.GetId()),
		"name":            llx.StringData(n.GetName()),
		"routed":          llx.BoolData(n.GetRouted()),
		"createdAt":       llx.TimeDataPtr(timeOrNil(createdAt, okCreated)),
		"ipv4Prefix":      llx.StringData(ipv4PrefixSing),
		"ipv4Gateway":     llx.StringData(ipv4Gateway),
		"ipv4Nameservers": strSliceData(ipv4Nameserv),
		"ipv4Prefixes":    strSliceData(ipv4Prefixes),
		"ipv6Prefix":      llx.StringData(ipv6PrefixSing),
		"ipv6Gateway":     llx.StringData(ipv6Gateway),
		"ipv6Nameservers": strSliceData(ipv6Nameserv),
		"ipv6Prefixes":    strSliceData(ipv6Prefixes),
		"state":           llx.StringData(n.GetStatus()),
		"labels":          labelData(n.GetLabels()),
	}
	return CreateResource(runtime, "stackit.network", args)
}

func (r *mqlStackitNetwork) id() (string, error) {
	return "stackit.network/" + r.Id.Data, nil
}

func initStackitNetwork(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	id, ok := idArg(args, "id")
	if !ok {
		return args, nil, nil
	}
	c := conn(runtime)
	client, err := c.IaaS()
	if err != nil {
		return nil, nil, err
	}
	n, err := client.GetNetworkExecute(bgctx(), c.ProjectID(), c.Region(), id)
	if err != nil {
		return nil, nil, err
	}
	res, err := buildNetwork(runtime, n)
	if err != nil {
		return nil, nil, err
	}
	return nil, res, nil
}

// iaasNetworkIPv4Empty / iaasNetworkIPv6Empty: the SDK's GetIpv4()/GetIpv6()
// always return a (possibly zero-value) struct, so check the inner pointers
// for "set"-ness instead.
func iaasNetworkIPv4Empty(v iaas.NetworkIPv4) bool {
	return v.Gateway == nil && v.Nameservers == nil && v.Prefixes == nil && v.PublicIp == nil
}

func iaasNetworkIPv6Empty(v iaas.NetworkIPv6) bool {
	return v.Gateway == nil && v.Nameservers == nil && v.Prefixes == nil
}

// ------------------------- public IPs -------------------------

func (r *mqlStackit) publicIps() ([]any, error) {
	c := conn(r.MqlRuntime)
	client, err := c.IaaS()
	if err != nil {
		return nil, err
	}
	resp, err := client.ListPublicIPsExecute(bgctx(), c.ProjectID(), c.Region())
	if err != nil {
		if isAccessDenied(err) {
			return []any{}, nil
		}
		return nil, err
	}
	items, _ := resp.GetItemsOk()
	out := make([]any, 0, len(items))
	for i := range items {
		res, err := buildPublicIp(r.MqlRuntime, &items[i])
		if err != nil {
			return nil, err
		}
		out = append(out, res)
	}
	return out, nil
}

func buildPublicIp(runtime *plugin.Runtime, ip *iaas.PublicIp) (plugin.Resource, error) {
	args := map[string]*llx.RawData{
		"id":                 llx.StringData(ip.GetId()),
		"ip":                 llx.StringData(ip.GetIp()),
		"networkInterfaceId": llx.StringData(ptrStr(ip.GetNetworkInterface())),
		"labels":             labelData(ip.GetLabels()),
	}
	return CreateResource(runtime, "stackit.publicIp", args)
}

func (r *mqlStackitPublicIp) id() (string, error) {
	return "stackit.publicIp/" + r.Id.Data, nil
}

func initStackitPublicIp(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	id, ok := idArg(args, "id")
	if !ok {
		return args, nil, nil
	}
	c := conn(runtime)
	client, err := c.IaaS()
	if err != nil {
		return nil, nil, err
	}
	ip, err := client.GetPublicIPExecute(bgctx(), c.ProjectID(), c.Region(), id)
	if err != nil {
		return nil, nil, err
	}
	res, err := buildPublicIp(runtime, ip)
	if err != nil {
		return nil, nil, err
	}
	return nil, res, nil
}

// ------------------------- security groups -------------------------

func (r *mqlStackit) securityGroups() ([]any, error) {
	c := conn(r.MqlRuntime)
	client, err := c.IaaS()
	if err != nil {
		return nil, err
	}
	resp, err := client.ListSecurityGroupsExecute(bgctx(), c.ProjectID(), c.Region())
	if err != nil {
		if isAccessDenied(err) {
			return []any{}, nil
		}
		return nil, err
	}
	items, _ := resp.GetItemsOk()
	out := make([]any, 0, len(items))
	for i := range items {
		res, err := buildSecurityGroup(r.MqlRuntime, &items[i])
		if err != nil {
			return nil, err
		}
		out = append(out, res)
	}
	return out, nil
}

func buildSecurityGroup(runtime *plugin.Runtime, sg *iaas.SecurityGroup) (plugin.Resource, error) {
	createdAt, ok1 := sg.GetCreatedAtOk()
	updatedAt, ok2 := sg.GetUpdatedAtOk()
	args := map[string]*llx.RawData{
		"id":          llx.StringData(sg.GetId()),
		"name":        llx.StringData(sg.GetName()),
		"description": llx.StringData(sg.GetDescription()),
		"stateful":    llx.BoolData(sg.GetStateful()),
		"createdAt":   llx.TimeDataPtr(timeOrNil(createdAt, ok1)),
		"updatedAt":   llx.TimeDataPtr(timeOrNil(updatedAt, ok2)),
		"labels":      labelData(sg.GetLabels()),
	}
	return CreateResource(runtime, "stackit.securityGroup", args)
}

func (r *mqlStackitSecurityGroup) id() (string, error) {
	return "stackit.securityGroup/" + r.Id.Data, nil
}

func initStackitSecurityGroup(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	id, ok := idArg(args, "id")
	if !ok {
		return args, nil, nil
	}
	c := conn(runtime)
	client, err := c.IaaS()
	if err != nil {
		return nil, nil, err
	}
	sg, err := client.GetSecurityGroupExecute(bgctx(), c.ProjectID(), c.Region(), id)
	if err != nil {
		return nil, nil, err
	}
	res, err := buildSecurityGroup(runtime, sg)
	if err != nil {
		return nil, nil, err
	}
	return nil, res, nil
}

func (r *mqlStackitSecurityGroup) rules() ([]any, error) {
	c := conn(r.MqlRuntime)
	client, err := c.IaaS()
	if err != nil {
		return nil, err
	}
	resp, err := client.ListSecurityGroupRulesExecute(bgctx(), c.ProjectID(), c.Region(), r.Id.Data)
	if err != nil {
		if isAccessDenied(err) {
			return []any{}, nil
		}
		return nil, err
	}
	items, _ := resp.GetItemsOk()
	out := make([]any, 0, len(items))
	for i := range items {
		rule := items[i]

		var icmpType, icmpCode int64
		if icmp, ok := rule.GetIcmpParametersOk(); ok {
			icmpType = int64(icmp.GetType())
			icmpCode = int64(icmp.GetCode())
		}
		var portMin, portMax int64
		if pr, ok := rule.GetPortRangeOk(); ok {
			portMin = int64(pr.GetMin())
			portMax = int64(pr.GetMax())
		}
		var protocol string
		if p, ok := rule.GetProtocolOk(); ok {
			protocol = p.GetName()
		}

		createdAt, okCreated := rule.GetCreatedAtOk()

		args := map[string]*llx.RawData{
			"id":                    llx.StringData(rule.GetId()),
			"securityGroupId":       llx.StringData(r.Id.Data),
			"direction":             llx.StringData(rule.GetDirection()),
			"ethertype":             llx.StringData(rule.GetEthertype()),
			"protocol":              llx.StringData(protocol),
			"description":           llx.StringData(rule.GetDescription()),
			"icmpType":              llx.IntData(icmpType),
			"icmpCode":              llx.IntData(icmpCode),
			"portRangeMin":          llx.IntData(portMin),
			"portRangeMax":          llx.IntData(portMax),
			"ipRange":               llx.StringData(rule.GetIpRange()),
			"remoteSecurityGroupId": llx.StringData(rule.GetRemoteSecurityGroupId()),
			"createdAt":             llx.TimeDataPtr(timeOrNil(createdAt, okCreated)),
		}
		res, err := CreateResource(r.MqlRuntime, "stackit.securityGroup.rule", args)
		if err != nil {
			return nil, err
		}
		out = append(out, res)
	}
	return out, nil
}

func (r *mqlStackitSecurityGroupRule) id() (string, error) {
	return "stackit.securityGroup.rule/" + r.SecurityGroupId.Data + "/" + r.Id.Data, nil
}

// ------------------------- key pairs -------------------------

func (r *mqlStackit) keyPairs() ([]any, error) {
	c := conn(r.MqlRuntime)
	client, err := c.IaaS()
	if err != nil {
		return nil, err
	}
	resp, err := client.ListKeyPairsExecute(bgctx())
	if err != nil {
		if isAccessDenied(err) {
			return []any{}, nil
		}
		return nil, err
	}
	items, _ := resp.GetItemsOk()
	out := make([]any, 0, len(items))
	for i := range items {
		res, err := buildKeyPair(r.MqlRuntime, &items[i])
		if err != nil {
			return nil, err
		}
		out = append(out, res)
	}
	return out, nil
}

func buildKeyPair(runtime *plugin.Runtime, kp *iaas.Keypair) (plugin.Resource, error) {
	createdAt, ok1 := kp.GetCreatedAtOk()
	updatedAt, ok2 := kp.GetUpdatedAtOk()
	args := map[string]*llx.RawData{
		"name":        llx.StringData(kp.GetName()),
		"fingerprint": llx.StringData(kp.GetFingerprint()),
		"publicKey":   llx.StringData(kp.GetPublicKey()),
		"createdAt":   llx.TimeDataPtr(timeOrNil(createdAt, ok1)),
		"updatedAt":   llx.TimeDataPtr(timeOrNil(updatedAt, ok2)),
		"labels":      labelData(kp.GetLabels()),
	}
	return CreateResource(runtime, "stackit.keyPair", args)
}

func (r *mqlStackitKeyPair) id() (string, error) {
	return "stackit.keyPair/" + r.Name.Data, nil
}

func initStackitKeyPair(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	v, ok := args["name"]
	if !ok || v == nil {
		return args, nil, nil
	}
	name, ok := v.Value.(string)
	if !ok || name == "" {
		return args, nil, nil
	}
	c := conn(runtime)
	client, err := c.IaaS()
	if err != nil {
		return nil, nil, err
	}
	kp, err := client.GetKeyPairExecute(bgctx(), name)
	if err != nil {
		return nil, nil, err
	}
	res, err := buildKeyPair(runtime, kp)
	if err != nil {
		return nil, nil, err
	}
	return nil, res, nil
}
