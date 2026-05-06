// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/vmware/govmomi/crypto"
	"github.com/vmware/govmomi/find"
	"github.com/vmware/govmomi/object"
	ssoadmintypes "github.com/vmware/govmomi/ssoadmin/types"
	"github.com/vmware/govmomi/view"
	"github.com/vmware/govmomi/vim25/mo"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
	"go.mondoo.com/mql/v13/providers/vsphere/connection"
	"go.mondoo.com/mql/v13/providers/vsphere/resources/resourceclient"
	"go.mondoo.com/mql/v13/types"
)

func getClientInstance(conn *connection.VsphereConnection) *resourceclient.Client {
	return resourceclient.New(conn.Client())
}

func esxiClient(conn *connection.VsphereConnection, path string) (*resourceclient.Esxi, error) {
	vClient := getClientInstance(conn)

	host, err := vClient.HostByInventoryPath(path)
	if err != nil {
		return nil, err
	}

	esxi := resourceclient.NewEsxiClient(vClient.Client, path, host)
	return esxi, nil
}

func (v *mqlVsphere) id() (string, error) {
	return "vsphere", nil
}

func (v *mqlVsphere) about() (map[string]any, error) {
	conn := v.MqlRuntime.Connection.(*connection.VsphereConnection)
	client := getClientInstance(conn)

	return client.AboutInfo()
}

func (v *mqlVsphere) datacenters() ([]any, error) {
	conn := v.MqlRuntime.Connection.(*connection.VsphereConnection)
	client := getClientInstance(conn)

	// fetch datacenters
	dcs, err := client.ListDatacenters()
	if err != nil {
		return nil, err
	}

	// convert datacenter to MQL
	datacenters := make([]any, len(dcs))
	for i, dc := range dcs {
		mqlDc, err := CreateResource(v.MqlRuntime, "vsphere.datacenter", map[string]*llx.RawData{
			"moid":          llx.StringData(dc.Reference().Encode()),
			"name":          llx.StringData(dc.Name()),
			"inventoryPath": llx.StringData(dc.InventoryPath),
		})
		if err != nil {
			return nil, err
		}

		datacenters[i] = mqlDc
	}

	return datacenters, nil
}

func (v *mqlVsphere) licenses() ([]any, error) {
	conn := v.MqlRuntime.Connection.(*connection.VsphereConnection)
	client := getClientInstance(conn)

	// fetch license
	lcs, err := client.ListLicenses()
	if err != nil {
		return nil, err
	}

	// convert licenses to MQL
	licenses := make([]any, len(lcs))
	for i, l := range lcs {
		mqlLicense, err := CreateResource(v.MqlRuntime, "vsphere.license", map[string]*llx.RawData{
			"name":  llx.StringData(l.Name),
			"total": llx.IntData(int64(l.Total)),
			"used":  llx.IntData(int64(l.Used)),
		})
		if err != nil {
			return nil, err
		}

		licenses[i] = mqlLicense
	}

	return licenses, nil
}

func (v *mqlVsphere) roles() ([]any, error) {
	conn := v.MqlRuntime.Connection.(*connection.VsphereConnection)
	authMgr := object.NewAuthorizationManager(conn.Client().Client)

	rolesList, err := authMgr.RoleList(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to list authorization roles: %w", err)
	}

	mqlRoles := make([]any, len(rolesList))
	for i, r := range rolesList {
		var label, summary string
		if d := r.Info.GetDescription(); d != nil {
			label = d.Label
			summary = d.Summary
		}
		mqlRole, err := CreateResource(v.MqlRuntime, "vsphere.role", map[string]*llx.RawData{
			"roleId":     llx.IntData(int64(r.RoleId)),
			"name":       llx.StringData(r.Name),
			"label":      llx.StringData(label),
			"summary":    llx.StringData(summary),
			"system":     llx.BoolData(r.System),
			"privileges": llx.ArrayData(convert.SliceAnyToInterface(r.Privilege), types.String),
		})
		if err != nil {
			return nil, err
		}
		mqlRoles[i] = mqlRole
	}
	return mqlRoles, nil
}

func (v *mqlVsphere) permissions() ([]any, error) {
	conn := v.MqlRuntime.Connection.(*connection.VsphereConnection)
	authMgr := object.NewAuthorizationManager(conn.Client().Client)

	perms, err := authMgr.RetrieveAllPermissions(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve permissions: %w", err)
	}

	mqlPerms := make([]any, len(perms))
	for i, p := range perms {
		var entityMoid, entityType string
		if p.Entity != nil {
			entityMoid = p.Entity.Encode()
			entityType = p.Entity.Type
		}
		// Principal kind disambiguates a user named "alice" from a group named "alice"
		// granted on the same entity — vSphere stores those as two distinct permissions.
		principalKind := "user"
		if p.Group {
			principalKind = "group"
		}
		id := entityMoid + ":" + principalKind + ":" + p.Principal
		mqlPerm, err := CreateResource(v.MqlRuntime, "vsphere.permission", map[string]*llx.RawData{
			"__id":       llx.StringData(id),
			"id":         llx.StringData(id),
			"entityMoid": llx.StringData(entityMoid),
			"entityType": llx.StringData(entityType),
			"principal":  llx.StringData(p.Principal),
			"group":      llx.BoolData(p.Group),
			"propagate":  llx.BoolData(p.Propagate),
		})
		if err != nil {
			return nil, err
		}
		mqlPerm.(*mqlVspherePermission).roleId = p.RoleId
		mqlPerms[i] = mqlPerm
	}
	return mqlPerms, nil
}

// mqlVspherePermissionInternal carries the raw roleId so role() can resolve
// the typed vsphere.role reference against the cached role list.
type mqlVspherePermissionInternal struct {
	roleId int32
}

func (v *mqlVsphere) folders() ([]any, error) {
	conn := v.MqlRuntime.Connection.(*connection.VsphereConnection)
	c := conn.Client().Client
	ctx := context.Background()

	mgr := view.NewManager(c)
	cv, err := mgr.CreateContainerView(ctx, c.ServiceContent.RootFolder, []string{"Folder"}, true)
	if err != nil {
		return nil, fmt.Errorf("failed to create folder container view: %w", err)
	}
	defer func() { _ = cv.Destroy(ctx) }()

	var folders []mo.Folder
	if err := cv.Retrieve(ctx, []string{"Folder"}, []string{"name", "childType", "childEntity"}, &folders); err != nil {
		return nil, fmt.Errorf("failed to retrieve folders: %w", err)
	}

	mqlFolders := make([]any, 0, len(folders))
	for _, f := range folders {
		ref := f.Reference()
		// mo.Folder doesn't carry inventoryPath; resolve it by walking the
		// ManagedEntity ancestor chain. find.InventoryPath does that via
		// PropertyCollector. One round trip per folder; acceptable on
		// typical vCenter sizes (tens of folders).
		path, err := find.InventoryPath(ctx, c, ref)
		if err != nil {
			// Resolving the path can fail if the folder is concurrently
			// removed; treat as "unknown" rather than aborting the whole
			// folder list.
			path = ""
		}
		mqlFolder, err := CreateResource(v.MqlRuntime, "vsphere.folder", map[string]*llx.RawData{
			"moid":          llx.StringData(ref.Encode()),
			"name":          llx.StringData(f.Name),
			"inventoryPath": llx.StringData(path),
			"childTypes":    llx.ArrayData(convert.SliceAnyToInterface(f.ChildType), types.String),
			"childCount":    llx.IntData(int64(len(f.ChildEntity))),
		})
		if err != nil {
			return nil, err
		}
		mqlFolders = append(mqlFolders, mqlFolder)
	}
	return mqlFolders, nil
}

func (v *mqlVsphere) identitySources() ([]any, error) {
	conn := v.MqlRuntime.Connection.(*connection.VsphereConnection)
	ctx := context.Background()

	admin, err := conn.SsoAdminClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to SSO admin: %w", err)
	}

	sources, err := admin.IdentitySources(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list identity sources: %w", err)
	}

	mqlSources := []any{}

	domainAliases := func(domains []ssoadmintypes.Domain) []string {
		aliases := make([]string, 0, len(domains)*2)
		for _, d := range domains {
			if d.Name != "" {
				aliases = append(aliases, d.Name)
			}
			if d.Alias != "" {
				aliases = append(aliases, d.Alias)
			}
		}
		return aliases
	}

	addSource := func(src ssoadmintypes.IdentitySource, kind string) error {
		mqlSrc, err := CreateResource(v.MqlRuntime, "vsphere.identitysource", map[string]*llx.RawData{
			"__id":                   llx.StringData(kind + ":" + src.Name),
			"name":                   llx.StringData(src.Name),
			"type":                   llx.StringData(kind),
			"authenticationType":     llx.StringData(""),
			"primaryUrl":             llx.StringData(""),
			"failoverUrl":            llx.StringData(""),
			"userBaseDn":             llx.StringData(""),
			"groupBaseDn":            llx.StringData(""),
			"authenticationUsername": llx.StringData(""),
			"alternativeNames":       llx.ArrayData(convert.SliceAnyToInterface(domainAliases(src.Domains)), types.String),
		})
		if err != nil {
			return err
		}
		mqlSources = append(mqlSources, mqlSrc)
		return nil
	}

	addLdapSource := func(src ssoadmintypes.LdapIdentitySource, kind string) error {
		mqlSrc, err := CreateResource(v.MqlRuntime, "vsphere.identitysource", map[string]*llx.RawData{
			"__id":                   llx.StringData(kind + ":" + src.Name),
			"name":                   llx.StringData(src.Name),
			"type":                   llx.StringData(kind),
			"authenticationType":     llx.StringData(src.AuthenticationDetails.AuthenticationType),
			"primaryUrl":             llx.StringData(src.Details.PrimaryURL),
			"failoverUrl":            llx.StringData(src.Details.FailoverURL),
			"userBaseDn":             llx.StringData(src.Details.UserBaseDn),
			"groupBaseDn":            llx.StringData(src.Details.GroupBaseDn),
			"authenticationUsername": llx.StringData(src.AuthenticationDetails.Username),
			"alternativeNames":       llx.ArrayData(convert.SliceAnyToInterface(domainAliases(src.Domains)), types.String),
		})
		if err != nil {
			return err
		}
		mqlSources = append(mqlSources, mqlSrc)
		return nil
	}

	if err := addSource(sources.System, "system"); err != nil {
		return nil, err
	}
	if sources.LocalOS != nil {
		if err := addSource(*sources.LocalOS, "localos"); err != nil {
			return nil, err
		}
	}
	if sources.NativeAD != nil {
		if err := addSource(*sources.NativeAD, "nativead"); err != nil {
			return nil, err
		}
	}
	for _, src := range sources.LDAPS {
		if err := addLdapSource(src, "ldap"); err != nil {
			return nil, err
		}
	}
	return mqlSources, nil
}

// role resolves the typed vsphere.role for this permission. First call after
// startup triggers a single AuthorizationManager.RoleList via vsphere.roles
// and caches the result; subsequent calls hit the cache.
func (p *mqlVspherePermission) role() (*mqlVsphereRole, error) {
	res, err := CreateResource(p.MqlRuntime, "vsphere", map[string]*llx.RawData{})
	if err != nil {
		return nil, err
	}
	roles := res.(*mqlVsphere).GetRoles()
	if roles.Error != nil {
		return nil, roles.Error
	}
	for _, r := range roles.Data {
		role := r.(*mqlVsphereRole)
		if role.RoleId.Data == int64(p.roleId) {
			return role, nil
		}
	}
	// Role lookup miss — most likely the role was deleted between the
	// permission scan and the role-list fetch. Mark the field resolved
	// and null so the runtime doesn't panic or re-fetch.
	p.Role.State = plugin.StateIsNull | plugin.StateIsSet
	return nil, nil
}

func (v *mqlVsphere) kmsClusters() ([]any, error) {
	conn := v.MqlRuntime.Connection.(*connection.VsphereConnection)
	mgr, err := crypto.GetManagerKmip(conn.Client().Client)
	if err != nil {
		// CryptoManager isn't available on direct ESXi connections; return
		// an empty list rather than an error so policies don't break.
		if errors.Is(err, object.ErrNotSupported) {
			return []any{}, nil
		}
		return nil, fmt.Errorf("failed to get KMIP manager: %w", err)
	}

	clusters, err := mgr.ListKmipServers(context.Background(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list KMIP servers: %w", err)
	}

	mqlClusters := make([]any, len(clusters))
	for i, c := range clusters {
		servers := make([]any, 0, len(c.Servers))
		for _, s := range c.Servers {
			servers = append(servers, map[string]any{
				"name":    s.Name,
				"address": s.Address,
				"port":    int64(s.Port),
			})
		}
		mqlCluster, err := CreateResource(v.MqlRuntime, "vsphere.kmsCluster", map[string]*llx.RawData{
			"clusterId":      llx.StringData(c.ClusterId.Id),
			"useAsDefault":   llx.BoolData(c.UseAsDefault),
			"managementType": llx.StringData(c.ManagementType),
			"serverCount":    llx.IntData(int64(len(c.Servers))),
			"servers":        llx.ArrayData(servers, types.Dict),
		})
		if err != nil {
			return nil, err
		}
		mqlClusters[i] = mqlCluster
	}
	return mqlClusters, nil
}

func (v *mqlEsxi) id() (string, error) {
	return "esxi", nil
}

func esxiHostProperties(conn *connection.VsphereConnection) (*object.HostSystem, *mo.HostSystem, error) {
	var h *object.HostSystem
	vClient := conn.Client()
	cl := resourceclient.New(vClient)
	if !vClient.IsVC() {
		// ESXi connections only have one host
		dcs, err := cl.ListDatacenters()
		if err != nil {
			return nil, nil, err
		}

		if len(dcs) != 1 {
			return nil, nil, errors.New("could not find single esxi datacenter")
		}

		dc := dcs[0]

		hosts, err := cl.ListHosts(dc, nil)
		if err != nil {
			return nil, nil, err
		}

		if len(hosts) != 1 {
			return nil, nil, errors.New("could not find single esxi host")
		}

		h = hosts[0]
	} else {
		// check if the connection was initialized with a specific host
		identifier, err := conn.Identifier()
		if err != nil || !connection.IsVsphereResourceID(identifier) {
			return nil, nil, errors.New("singular host resource is ambiguous on a vCenter connection; use vsphere.datacenter.hosts or vsphere.cluster.hosts to enumerate, or connect directly to an ESXi host")
		}

		// extract type and inventory
		moid, err := connection.ParseVsphereResourceID(identifier)
		if err != nil {
			return nil, nil, err
		}

		if moid.Type != "HostSystem" {
			return nil, nil, errors.New("singular host resource is not supported for vsphere type " + moid.Type)
		}

		h, err = cl.HostByMoid(moid)
		if err != nil {
			return nil, nil, fmt.Errorf("could not find the esxi host via platform id: %s: %w", identifier, err)
		}
	}

	// todo sync with GetHosts
	hostInfo, err := resourceclient.HostInfo(context.Background(), h)
	if err != nil {
		return nil, nil, err
	}

	return h, hostInfo, nil
}

func (v *mqlEsxiCommand) id() (string, error) {
	return v.Command.Data, nil
}

func initEsxiCommand(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	conn := runtime.Connection.(*connection.VsphereConnection)

	if len(args) > 2 {
		return args, nil, nil
	}

	// check if the command arg is provided
	commandRaw := args["command"]
	if commandRaw == nil {
		return args, nil, nil
	}

	// check if the connection was initialized with a specific host
	identifier, err := conn.Identifier()
	if err != nil || !connection.IsVsphereResourceID(identifier) {
		return nil, nil, errors.New("could not determine inventoryPath from provider connection")
	}

	h, err := hostSystem(conn, identifier)
	if err != nil {
		return nil, nil, err
	}

	args["inventoryPath"] = llx.StringData(h.InventoryPath)
	return args, nil, nil
}

func hostSystem(conn *connection.VsphereConnection, identifier string) (*object.HostSystem, error) {
	var h *object.HostSystem
	vClient := conn.Client()
	cl := resourceclient.New(vClient)

	// extract type and inventory
	moid, err := connection.ParseVsphereResourceID(identifier)
	if err != nil {
		return nil, err
	}

	if moid.Type != "HostSystem" {
		return nil, errors.New("ESXi resource is not supported for vsphere type " + moid.Type)
	}

	h, err = cl.HostByMoid(moid)
	if err != nil {
		return nil, errors.New("could not find the esxi host via platform id: " + identifier)
	}

	return h, nil
}

func (v *mqlEsxiCommand) result() ([]any, error) {
	conn := v.MqlRuntime.Connection.(*connection.VsphereConnection)

	if v.InventoryPath.Error != nil {
		return nil, v.InventoryPath.Error
	}
	path := v.InventoryPath.Data

	esxiClient, err := esxiClient(conn, path)
	if err != nil {
		return nil, err
	}

	if v.Command.Error != nil {
		return nil, v.Command.Error
	}
	cmd := v.Command.Data

	res := []any{}

	resp, err := esxiClient.Command(cmd)
	if err != nil {
		return nil, err
	}

	for i := range resp {
		res = append(res, resp[i])
	}

	return res, nil
}

func (v *mqlVsphereLicense) id() (string, error) {
	return v.Name.Data, nil
}

func (v *mqlVsphereVmknic) id() (string, error) {
	return v.Name.Data, nil
}

func (v *mqlEsxiVib) id() (string, error) {
	return v.Id.Data, nil
}

func (v *mqlEsxiKernelmodule) id() (string, error) {
	return v.Name.Data, nil
}

func (v *mqlEsxiService) id() (string, error) {
	return v.Key.Data, nil
}

func (v *mqlEsxiTimezone) id() (string, error) {
	return v.Key.Data, nil
}

func (v *mqlEsxiNtpconfig) id() (string, error) {
	return v.Id.Data, nil
}

func (v *mqlVsphereRole) id() (string, error) {
	return strconv.FormatInt(v.RoleId.Data, 10), nil
}

func (v *mqlVsphereDatastore) id() (string, error) {
	return v.Moid.Data, nil
}

func (v *mqlEsxiCertificate) id() (string, error) {
	return v.Id.Data, nil
}

func (v *mqlVsphereFolder) id() (string, error) {
	return v.Moid.Data, nil
}

func (v *mqlVsphereResourcepool) id() (string, error) {
	return v.Moid.Data, nil
}

func (v *mqlEsxiFirewallRuleset) id() (string, error) {
	return v.Id.Data, nil
}

func (v *mqlEsxiFirewallRule) id() (string, error) {
	return v.Id.Data, nil
}

func (v *mqlEsxiIscsiAdapter) id() (string, error) {
	return v.Id.Data, nil
}

func (v *mqlVsphereKmsCluster) id() (string, error) {
	return v.ClusterId.Data, nil
}
