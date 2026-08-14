// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
)

type groupListResponse struct {
	Groups []groupRecord `json:"groups"`
}

// groupRecord is a group as the Access API reports it. The list reports the
// name and description only, so the rest is read per group.
type groupRecord struct {
	Name            string   `json:"name"`
	Description     string   `json:"description"`
	AutoJoin        *bool    `json:"auto_join"`
	AdminPrivileges *bool    `json:"admin_privileges"`
	Realm           string   `json:"realm"`
	RealmAttributes string   `json:"realm_attributes"`
	ExternalID      string   `json:"external_id"`
	Members         []string `json:"members"`
}

type mqlArtifactoryGroupInternal struct {
	// lock guards the single group read that backs every detail field. Only a
	// successful read is kept, so a transient failure is retried rather than
	// failing every later field for the rest of the scan.
	lock   sync.Mutex
	detail *groupRecord
	// detailLoaded is read on the fast path without the lock, so it is atomic.
	// A plain bool there would be an unsynchronized read against the write the
	// lock holder makes, which is a data race whatever the value happens to be.
	detailLoaded atomic.Bool
}

func (a *mqlArtifactory) groups() ([]any, error) {
	conn := artifactoryConn(a.MqlRuntime)

	var response groupListResponse
	if err := conn.GetJSON(context.Background(), conn.AccessURL("/api/v2/groups"), &response); err != nil {
		return nil, err
	}

	res := make([]any, 0, len(response.Groups))
	for i := range response.Groups {
		group, err := newArtifactoryGroup(a.MqlRuntime, &response.Groups[i])
		if err != nil {
			return nil, err
		}
		res = append(res, group)
	}
	return res, nil
}

func newArtifactoryGroup(runtime *plugin.Runtime, rec *groupRecord) (*mqlArtifactoryGroup, error) {
	res, err := CreateResource(runtime, "artifactory.group", map[string]*llx.RawData{
		"name": llx.StringData(rec.Name),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlArtifactoryGroup), nil
}

// initArtifactoryGroup resolves a group by name.
func initArtifactoryGroup(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 1 {
		return args, nil, nil
	}

	name := ""
	if data, ok := args["name"]; ok {
		if s, ok := data.Value.(string); ok {
			name = s
		}
	}
	if name == "" {
		return nil, nil, errors.New("artifactory.group requires a name")
	}

	group, err := newArtifactoryGroup(runtime, &groupRecord{Name: name})
	if err != nil {
		return nil, nil, err
	}
	return args, group, nil
}

func (g *mqlArtifactoryGroup) id() (string, error) {
	return "artifactory.group/" + g.Name.Data, g.Name.Error
}

func groupDetailURL(runtime *plugin.Runtime, name string) string {
	conn := artifactoryConn(runtime)
	return conn.AccessURL("/api/v2/groups/" + url.PathEscape(name))
}

// definition reads the group once and shares it with every field that needs it.
func (g *mqlArtifactoryGroup) definition() (*groupRecord, error) {
	if g.detailLoaded.Load() {
		return g.detail, nil
	}
	g.lock.Lock()
	defer g.lock.Unlock()
	if g.detailLoaded.Load() {
		return g.detail, nil
	}

	conn := artifactoryConn(g.MqlRuntime)
	var detail groupRecord
	if err := conn.GetJSON(context.Background(), groupDetailURL(g.MqlRuntime, g.Name.Data), &detail); err != nil {
		return nil, err
	}

	g.detail = &detail
	g.detailLoaded.Store(true)
	return g.detail, nil
}

func (g *mqlArtifactoryGroup) description() (string, error) {
	detail, err := g.definition()
	if err != nil {
		return "", err
	}
	return detail.Description, nil
}

func (g *mqlArtifactoryGroup) adminPrivileges() (bool, error) {
	detail, err := g.definition()
	if err != nil {
		return false, err
	}
	return boolValue(detail.AdminPrivileges), nil
}

func (g *mqlArtifactoryGroup) autoJoin() (bool, error) {
	detail, err := g.definition()
	if err != nil {
		return false, err
	}
	return boolValue(detail.AutoJoin), nil
}

func (g *mqlArtifactoryGroup) realm() (string, error) {
	detail, err := g.definition()
	if err != nil {
		return "", err
	}
	return detail.Realm, nil
}

func (g *mqlArtifactoryGroup) realmAttributes() (string, error) {
	detail, err := g.definition()
	if err != nil {
		return "", err
	}
	return detail.RealmAttributes, nil
}

func (g *mqlArtifactoryGroup) internal() (bool, error) {
	realm, err := g.realm()
	if err != nil {
		return false, err
	}
	return strings.EqualFold(realm, realmInternal), nil
}

// users resolves the group's members against the instance's user list. A
// member the list does not hold, such as an account of a directory the
// instance has not imported, is skipped.
func (g *mqlArtifactoryGroup) users() ([]any, error) {
	detail, err := g.definition()
	if err != nil {
		return nil, err
	}

	res := []any{}
	for _, name := range detail.Members {
		user, err := findUser(g.MqlRuntime, name)
		if err != nil {
			return nil, err
		}
		if user != nil {
			res = append(res, user)
		}
	}
	return res, nil
}

func (g *mqlArtifactoryGroup) permissionTargets() ([]any, error) {
	return permissionTargetsFor(g.MqlRuntime, principalGroup, g.Name.Data)
}

// findGroup looks up a group in the instance's group list, which the root
// resource fetches once. A name the list does not hold reports nil.
func findGroup(runtime *plugin.Runtime, name string) (*mqlArtifactoryGroup, error) {
	if name == "" {
		return nil, nil
	}

	root, err := getArtifactory(runtime)
	if err != nil {
		return nil, err
	}
	groups := root.GetGroups()
	if groups.Error != nil {
		return nil, groups.Error
	}

	for _, it := range groups.Data {
		group, ok := it.(*mqlArtifactoryGroup)
		if ok && group.Name.Data == name {
			return group, nil
		}
	}
	return nil, nil
}
