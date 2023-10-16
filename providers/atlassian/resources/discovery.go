package resources

import (
	"context"

	"go.mondoo.com/cnquery/v9/llx"
	"go.mondoo.com/cnquery/v9/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v9/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v9/providers/atlassian/connection"
	"go.mondoo.com/cnquery/v9/providers/atlassian/connection/admin"
	"go.mondoo.com/cnquery/v9/utils/stringx"
)

func Discover(runtime *plugin.Runtime, opts map[string]string) (*inventory.Inventory, error) {
	conn := runtime.Connection.(*admin.AdminConnection)

	in := &inventory.Inventory{Spec: &inventory.InventorySpec{
		Assets: []*inventory.Asset{},
	}}

	targets := handleTargets(conn.Conf.Discover.Targets)
	list, err := discover(runtime, targets)
	if err != nil {
		return in, err
	}

	in.Spec.Assets = list
	return in, nil
}

func handleTargets(targets []string) []string {
	if stringx.Contains(targets, connection.DiscoveryAll) {
		return []string{connection.DiscoveryOrganization}
	}
	return targets
}

func discover(runtime *plugin.Runtime, targets []string) ([]*inventory.Asset, error) {
	conn := runtime.Connection.(*admin.AdminConnection)
	assetList := []*inventory.Asset{}
	orgAssets, err := org(runtime, "test", conn, targets)
	if err != nil {
		return nil, err
	}
	assetList = append(assetList, orgAssets...)
	return assetList, nil
}

func org(runtime *plugin.Runtime, orgName string, conn *admin.AdminConnection, targets []string) ([]*inventory.Asset, error) {
	assetList := []*inventory.Asset{}
	client := conn.Client()
	orgs, _, err := client.Organization.Gets(context.Background(), "")
	if err != nil {
		return nil, err
	}
	for _, org := range orgs.Data {
		mqlOrg, err := getMqlAtlassianOrg(runtime, org.ID, org.Attributes.Name)
		if err != nil {
			return nil, err
		}
		assetList = append(assetList, &inventory.Asset{
			PlatformIds: []string{conn.PlatformID()},
			Name:        mqlOrg.Id.Data,
			Platform:    conn.PlatformInfo(),
			Labels:      map[string]string{},
			Connections: []*inventory.Config{cloneInventoryConf(conn.Conf)},
		})
	}
	return assetList, nil
}

func getMqlAtlassianOrg(runtime *plugin.Runtime, orgId string, orgName string) (*mqlAtlassianAdminOrganization, error) {
	res, err := NewResource(runtime, "atlassian.admin.organization", map[string]*llx.RawData{
		"id":   llx.StringData(orgId),
		"name": llx.StringData(orgName),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlAtlassianAdminOrganization), nil
}

func cloneInventoryConf(invConf *inventory.Config) *inventory.Config {
	invConfClone := invConf.Clone()
	// We do not want to run discovery again for the already discovered assets
	invConfClone.Discover = &inventory.Discovery{}
	return invConfClone
}
