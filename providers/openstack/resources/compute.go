// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"sync"

	"github.com/gophercloud/gophercloud/v2/openstack/compute/v2/flavors"
	"github.com/gophercloud/gophercloud/v2/openstack/compute/v2/keypairs"
	"github.com/gophercloud/gophercloud/v2/openstack/compute/v2/servergroups"
	"github.com/gophercloud/gophercloud/v2/openstack/compute/v2/servers"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
)

// ---- openstack.compute.server ----

type mqlOpenstackComputeServerInternal struct {
	cacheFlavorID         string
	cacheImageID          string
	cacheKeyName          string
	cacheSecurityGroupSGs []string
	cacheVolumeIDs        []string
}

func (r *mqlOpenstackComputeServer) id() (string, error) {
	return "openstack.compute.server/" + r.Id.Data, nil
}

func initOpenstackComputeServer(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	return args, nil, nil
}

func (o *mqlOpenstack) servers() ([]any, error) {
	c := conn(o.MqlRuntime)
	client, err := c.ComputeClient()
	if err != nil {
		return nil, err
	}
	pages, err := servers.List(client, servers.ListOpts{}).AllPages(ctx())
	if err != nil {
		if translateOpenstackError(err) == nil {
			return []any{}, nil
		}
		return nil, err
	}
	items, err := servers.ExtractServers(pages)
	if err != nil {
		return nil, err
	}

	out := make([]any, 0, len(items))
	for i := range items {
		res, err := newMqlOpenstackComputeServer(o.MqlRuntime, &items[i])
		if err != nil {
			return nil, err
		}
		out = append(out, res)
	}
	return out, nil
}

func newMqlOpenstackComputeServer(runtime *plugin.Runtime, s *servers.Server) (*mqlOpenstackComputeServer, error) {
	imageID := serverImage(s.Image)
	flavorID := serverFlavorID(s.Flavor)
	sgNames := serverSecurityGroupNames(s.SecurityGroups)
	volumeIDs := serverVolumeIDs(s.AttachedVolumes)

	res, err := CreateResource(runtime, "openstack.compute.server", map[string]*llx.RawData{
		"__id":             llx.StringData("openstack.compute.server/" + s.ID),
		"id":               llx.StringData(s.ID),
		"name":             llx.StringData(s.Name),
		"status":           llx.StringData(s.Status),
		"powerState":       llx.IntData(int64(s.PowerState)),
		"vmState":          llx.StringData(s.VmState),
		"taskState":        llx.StringData(s.TaskState),
		"locked":           llx.BoolData(s.Locked != nil && *s.Locked),
		"hostId":           llx.StringData(s.HostID),
		"accessIPv4":       llx.StringData(s.AccessIPv4),
		"accessIPv6":       llx.StringData(s.AccessIPv6),
		"projectId":        llx.StringData(s.TenantID),
		"userId":           llx.StringData(s.UserID),
		"keyName":          llx.StringData(s.KeyName),
		"availabilityZone": llx.StringData(s.AvailabilityZone),
		"diskConfig":       llx.StringData(string(s.DiskConfig)),
		"addresses":        llx.DictData(toDict(s.Addresses)),
		"metadata":         stringMapData(s.Metadata),
		"tags":             stringSliceData(derefStrings(s.Tags)),
		"created":          llx.TimeDataPtr(timePtr(s.Created)),
		"updated":          llx.TimeDataPtr(timePtr(s.Updated)),
		"launchedAt":       llx.TimeDataPtr(timePtr(s.LaunchedAt)),
		"terminatedAt":     llx.TimeDataPtr(timePtr(s.TerminatedAt)),
	})
	if err != nil {
		return nil, err
	}
	mqlServer := res.(*mqlOpenstackComputeServer)
	mqlServer.cacheFlavorID = flavorID
	mqlServer.cacheImageID = imageID
	mqlServer.cacheKeyName = s.KeyName
	mqlServer.cacheSecurityGroupSGs = sgNames
	mqlServer.cacheVolumeIDs = volumeIDs
	return mqlServer, nil
}

func (r *mqlOpenstackComputeServer) image() (*mqlOpenstackImage, error) {
	if r.cacheImageID == "" {
		r.Image.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	res, err := NewResource(r.MqlRuntime, "openstack.image", map[string]*llx.RawData{
		"id": llx.StringData(r.cacheImageID),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlOpenstackImage), nil
}

func (r *mqlOpenstackComputeServer) volumes() ([]any, error) {
	if len(r.cacheVolumeIDs) == 0 {
		return []any{}, nil
	}
	out := make([]any, 0, len(r.cacheVolumeIDs))
	for _, id := range r.cacheVolumeIDs {
		if id == "" {
			continue
		}
		res, err := NewResource(r.MqlRuntime, "openstack.blockstorage.volume", map[string]*llx.RawData{
			"id": llx.StringData(id),
		})
		if err != nil {
			return nil, err
		}
		out = append(out, res)
	}
	return out, nil
}

func serverVolumeIDs(in []servers.AttachedVolume) []string {
	out := make([]string, 0, len(in))
	for _, v := range in {
		if v.ID == "" {
			continue
		}
		out = append(out, v.ID)
	}
	return out
}

// serverGroupMetadata converts the SDK's map[string]any (Nova returns
// metadata typed loosely) into map[string]string. Non-string values are
// dropped — Nova validates metadata as string-to-string, so this only fires
// on out-of-spec clouds.
func serverGroupMetadata(in map[string]any) map[string]string {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]string, len(in))
	for k, raw := range in {
		if s, ok := raw.(string); ok {
			out[k] = s
		}
	}
	return out
}

func (r *mqlOpenstackComputeServer) flavor() (*mqlOpenstackComputeFlavor, error) {
	if r.cacheFlavorID == "" {
		r.Flavor.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	res, err := NewResource(r.MqlRuntime, "openstack.compute.flavor", map[string]*llx.RawData{
		"id": llx.StringData(r.cacheFlavorID),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlOpenstackComputeFlavor), nil
}

func (r *mqlOpenstackComputeServer) keypair() (*mqlOpenstackComputeKeypair, error) {
	if r.cacheKeyName == "" {
		r.Keypair.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	res, err := NewResource(r.MqlRuntime, "openstack.compute.keypair", map[string]*llx.RawData{
		"name": llx.StringData(r.cacheKeyName),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlOpenstackComputeKeypair), nil
}

// securityGroups resolves the security groups Nova reports on the server.
// Nova returns groups by name, but Neutron security groups are keyed by ID, so
// we resolve names against the project's groups list. Names that don't match
// (e.g. groups in a different project the user can't see) are skipped silently.
func (r *mqlOpenstackComputeServer) securityGroups() ([]any, error) {
	if len(r.cacheSecurityGroupSGs) == 0 {
		return []any{}, nil
	}

	c := conn(r.MqlRuntime)
	client, err := c.NetworkClient()
	if err != nil {
		return nil, err
	}

	out := make([]any, 0, len(r.cacheSecurityGroupSGs))
	for _, name := range r.cacheSecurityGroupSGs {
		id, err := lookupSecurityGroupIDByName(client, name)
		if err != nil {
			return nil, err
		}
		if id == "" {
			continue
		}
		res, err := NewResource(r.MqlRuntime, "openstack.securityGroup", map[string]*llx.RawData{
			"id": llx.StringData(id),
		})
		if err != nil {
			return nil, err
		}
		out = append(out, res)
	}
	return out, nil
}

// serverImage extracts the image ID from the `image` field on a server. Nova
// returns "" (or a JSON string of "") when the server is booted from volume.
func serverImage(raw any) string {
	if raw == nil {
		return ""
	}
	if v, ok := raw.(map[string]any); ok {
		id, _ := v["id"].(string)
		return id
	}
	return ""
}

func serverFlavorID(raw map[string]any) string {
	if raw == nil {
		return ""
	}
	if id, ok := raw["id"].(string); ok {
		return id
	}
	if og, ok := raw["original_name"].(string); ok {
		return og
	}
	return ""
}

func serverSecurityGroupNames(in []map[string]any) []string {
	out := make([]string, 0, len(in))
	for _, sg := range in {
		if name, ok := sg["name"].(string); ok && name != "" {
			out = append(out, name)
		}
	}
	return out
}

// derefStrings turns *[]string into []string, returning empty when nil.
// gophercloud uses *[]string for tags so it can distinguish unset from empty.
func derefStrings(p *[]string) []string {
	if p == nil {
		return nil
	}
	return *p
}

// toDict converts a map[string]any (or nil) into a dict-friendly map[string]any.
func toDict(in map[string]any) map[string]any {
	if in == nil {
		return map[string]any{}
	}
	return in
}

// ---- openstack.compute.flavor ----

func (r *mqlOpenstackComputeFlavor) id() (string, error) {
	return "openstack.compute.flavor/" + r.Id.Data, nil
}

type mqlOpenstackComputeFlavorInternal struct {
	specsLock sync.Mutex
	specsDone bool
}

func initOpenstackComputeFlavor(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if _, ok := args["id"]; !ok {
		return args, nil, nil
	}
	if _, hasName := args["name"]; hasName {
		return args, nil, nil
	}

	id, ok := stringArg(args, "id")
	if !ok || id == "" {
		return args, nil, nil
	}

	c := conn(runtime)
	client, err := c.ComputeClient()
	if err != nil {
		return nil, nil, err
	}
	f, err := flavors.Get(ctx(), client, id).Extract()
	if err != nil {
		if translateOpenstackError(err) == nil {
			return args, nil, nil
		}
		return nil, nil, err
	}
	populateFlavorArgs(args, f)
	return args, nil, nil
}

func (o *mqlOpenstack) flavors() ([]any, error) {
	c := conn(o.MqlRuntime)
	client, err := c.ComputeClient()
	if err != nil {
		return nil, err
	}
	pages, err := flavors.ListDetail(client, flavors.ListOpts{}).AllPages(ctx())
	if err != nil {
		if translateOpenstackError(err) == nil {
			return []any{}, nil
		}
		return nil, err
	}
	items, err := flavors.ExtractFlavors(pages)
	if err != nil {
		return nil, err
	}

	out := make([]any, 0, len(items))
	for i := range items {
		args := map[string]*llx.RawData{}
		populateFlavorArgs(args, &items[i])
		res, err := CreateResource(o.MqlRuntime, "openstack.compute.flavor", args)
		if err != nil {
			return nil, err
		}
		out = append(out, res)
	}
	return out, nil
}

func populateFlavorArgs(args map[string]*llx.RawData, f *flavors.Flavor) {
	args["__id"] = llx.StringData("openstack.compute.flavor/" + f.ID)
	args["id"] = llx.StringData(f.ID)
	args["name"] = llx.StringData(f.Name)
	args["vcpus"] = llx.IntData(int64(f.VCPUs))
	args["ram"] = llx.IntData(int64(f.RAM))
	args["disk"] = llx.IntData(int64(f.Disk))
	args["swap"] = llx.IntData(int64(f.Swap))
	args["ephemeral"] = llx.IntData(int64(f.Ephemeral))
	args["rxtxFactor"] = llx.FloatData(f.RxTxFactor)
	args["isPublic"] = llx.BoolData(f.IsPublic)
	args["description"] = llx.StringData(f.Description)
}

func (r *mqlOpenstackComputeFlavor) extraSpecs() (map[string]any, error) {
	r.specsLock.Lock()
	defer r.specsLock.Unlock()
	if r.specsDone {
		return r.ExtraSpecs.Data, nil
	}

	c := conn(r.MqlRuntime)
	client, err := c.ComputeClient()
	if err != nil {
		return nil, err
	}
	specs, err := flavors.ListExtraSpecs(ctx(), client, r.Id.Data).Extract()
	if err != nil {
		if translateOpenstackError(err) == nil {
			r.specsDone = true
			return map[string]any{}, nil
		}
		return nil, err
	}
	r.specsDone = true
	return stringMap(specs), nil
}

// ---- openstack.compute.keypair ----

func (r *mqlOpenstackComputeKeypair) id() (string, error) {
	return "openstack.compute.keypair/" + r.Name.Data, nil
}

func initOpenstackComputeKeypair(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	return args, nil, nil
}

func (o *mqlOpenstack) keypairs() ([]any, error) {
	c := conn(o.MqlRuntime)
	client, err := c.ComputeClient()
	if err != nil {
		return nil, err
	}
	pages, err := keypairs.List(client, keypairs.ListOpts{}).AllPages(ctx())
	if err != nil {
		if translateOpenstackError(err) == nil {
			return []any{}, nil
		}
		return nil, err
	}
	items, err := keypairs.ExtractKeyPairs(pages)
	if err != nil {
		return nil, err
	}

	out := make([]any, 0, len(items))
	for _, k := range items {
		res, err := CreateResource(o.MqlRuntime, "openstack.compute.keypair", map[string]*llx.RawData{
			"__id":        llx.StringData("openstack.compute.keypair/" + k.Name),
			"name":        llx.StringData(k.Name),
			"type":        llx.StringData(k.Type),
			"fingerprint": llx.StringData(k.Fingerprint),
			"publicKey":   llx.StringData(k.PublicKey),
			"userId":      llx.StringData(k.UserID),
		})
		if err != nil {
			return nil, err
		}
		out = append(out, res)
	}
	return out, nil
}

// ---- openstack.compute.serverGroup ----

func (r *mqlOpenstackComputeServerGroup) id() (string, error) {
	return "openstack.compute.serverGroup/" + r.Id.Data, nil
}

func initOpenstackComputeServerGroup(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	return args, nil, nil
}

func (o *mqlOpenstack) serverGroups() ([]any, error) {
	c := conn(o.MqlRuntime)
	client, err := c.ComputeClient()
	if err != nil {
		return nil, err
	}
	pages, err := servergroups.List(client, servergroups.ListOpts{}).AllPages(ctx())
	if err != nil {
		if translateOpenstackError(err) == nil {
			return []any{}, nil
		}
		return nil, err
	}
	items, err := servergroups.ExtractServerGroups(pages)
	if err != nil {
		return nil, err
	}

	out := make([]any, 0, len(items))
	for _, g := range items {
		res, err := CreateResource(o.MqlRuntime, "openstack.compute.serverGroup", map[string]*llx.RawData{
			"__id":      llx.StringData("openstack.compute.serverGroup/" + g.ID),
			"id":        llx.StringData(g.ID),
			"name":      llx.StringData(g.Name),
			"policies":  stringSliceData(g.Policies),
			"members":   stringSliceData(g.Members),
			"metadata":  stringMapData(serverGroupMetadata(g.Metadata)),
			"projectId": llx.StringData(g.ProjectID),
			"userId":    llx.StringData(g.UserID),
		})
		if err != nil {
			return nil, err
		}
		out = append(out, res)
	}
	return out, nil
}

func (r *mqlOpenstackComputeServerGroup) memberServers() ([]any, error) {
	if len(r.Members.Data) == 0 {
		return []any{}, nil
	}
	out := make([]any, 0, len(r.Members.Data))
	for _, m := range r.Members.Data {
		id, ok := m.(string)
		if !ok || id == "" {
			continue
		}
		res, err := NewResource(r.MqlRuntime, "openstack.compute.server", map[string]*llx.RawData{
			"id": llx.StringData(id),
		})
		if err != nil {
			return nil, err
		}
		out = append(out, res)
	}
	return out, nil
}
