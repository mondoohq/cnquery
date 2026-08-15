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
	"time"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
)

// realmInternal is the realm of an account or a group that lives on the
// instance rather than in an identity provider.
const realmInternal = "internal"

type userListResponse struct {
	Users []userRecord `json:"users"`
}

// userRecord is an account as the Access API reports it. The list reports the
// identity fields only, so everything else is read per account.
type userRecord struct {
	Username                 string   `json:"username"`
	Email                    string   `json:"email"`
	Admin                    *bool    `json:"admin"`
	Realm                    string   `json:"realm"`
	Status                   string   `json:"status"`
	ProfileUpdatable         *bool    `json:"profile_updatable"`
	InternalPasswordDisabled *bool    `json:"internal_password_disabled"`
	DisableUIAccess          *bool    `json:"disable_ui_access"`
	Groups                   []string `json:"groups"`
	LastLoggedIn             isoTime  `json:"last_logged_in"`
}

type mqlArtifactoryUserInternal struct {
	// lock guards the single account read that backs every detail field. Only a
	// successful read is kept, so a transient failure is retried rather than
	// failing every later field for the rest of the scan.
	lock   sync.Mutex
	detail *userRecord
	// detailLoaded is read on the fast path without the lock, so it is atomic.
	// A plain bool there would be an unsynchronized read against the write the
	// lock holder makes, which is a data race whatever the value happens to be.
	detailLoaded atomic.Bool
}

func (a *mqlArtifactory) users() ([]any, error) {
	conn := artifactoryConn(a.MqlRuntime)

	var response userListResponse
	if err := conn.GetJSON(context.Background(), conn.AccessURL("/api/v2/users"), &response); err != nil {
		return nil, err
	}

	res := make([]any, 0, len(response.Users))
	for i := range response.Users {
		user, err := newArtifactoryUser(a.MqlRuntime, &response.Users[i])
		if err != nil {
			return nil, err
		}
		res = append(res, user)
	}
	return res, nil
}

// newArtifactoryUser builds the resource from a list entry. The entry carries
// the account name; the remaining fields are read on demand, because an
// instance backed by a directory can hold far more accounts than a query
// touches.
func newArtifactoryUser(runtime *plugin.Runtime, rec *userRecord) (*mqlArtifactoryUser, error) {
	res, err := CreateResource(runtime, "artifactory.user", map[string]*llx.RawData{
		"name": llx.StringData(rec.Username),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlArtifactoryUser), nil
}

// initArtifactoryUser resolves an account by name.
//
// The resource is built with CreateResource, which does not run this init a
// second time. Only NewResource does, so there is no recursion here.
func initArtifactoryUser(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
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
		return nil, nil, errors.New("artifactory.user requires a name")
	}

	user, err := newArtifactoryUser(runtime, &userRecord{Username: name})
	if err != nil {
		return nil, nil, err
	}
	return args, user, nil
}

func (u *mqlArtifactoryUser) id() (string, error) {
	return "artifactory.user/" + u.Name.Data, u.Name.Error
}

func userDetailURL(runtime *plugin.Runtime, name string) string {
	conn := artifactoryConn(runtime)
	return conn.AccessURL("/api/v2/users/" + url.PathEscape(name))
}

// account reads the account once and shares it with every field that needs it.
func (u *mqlArtifactoryUser) account() (*userRecord, error) {
	if u.detailLoaded.Load() {
		return u.detail, nil
	}
	u.lock.Lock()
	defer u.lock.Unlock()
	if u.detailLoaded.Load() {
		return u.detail, nil
	}

	conn := artifactoryConn(u.MqlRuntime)
	var detail userRecord
	if err := conn.GetJSON(context.Background(), userDetailURL(u.MqlRuntime, u.Name.Data), &detail); err != nil {
		return nil, err
	}

	u.detail = &detail
	u.detailLoaded.Store(true)
	return u.detail, nil
}

func (u *mqlArtifactoryUser) email() (string, error) {
	detail, err := u.account()
	if err != nil {
		return "", err
	}
	return detail.Email, nil
}

func (u *mqlArtifactoryUser) admin() (bool, error) {
	detail, err := u.account()
	if err != nil {
		return false, err
	}
	return boolValue(detail.Admin), nil
}

func (u *mqlArtifactoryUser) realm() (string, error) {
	detail, err := u.account()
	if err != nil {
		return "", err
	}
	return detail.Realm, nil
}

func (u *mqlArtifactoryUser) internal() (bool, error) {
	realm, err := u.realm()
	if err != nil {
		return false, err
	}
	return strings.EqualFold(realm, realmInternal), nil
}

func (u *mqlArtifactoryUser) internalPasswordDisabled() (bool, error) {
	detail, err := u.account()
	if err != nil {
		return false, err
	}
	return boolValue(detail.InternalPasswordDisabled), nil
}

func (u *mqlArtifactoryUser) profileUpdatable() (bool, error) {
	detail, err := u.account()
	if err != nil {
		return false, err
	}
	return boolValue(detail.ProfileUpdatable), nil
}

func (u *mqlArtifactoryUser) disableUiAccess() (bool, error) {
	detail, err := u.account()
	if err != nil {
		return false, err
	}
	return boolValue(detail.DisableUIAccess), nil
}

func (u *mqlArtifactoryUser) status() (string, error) {
	detail, err := u.account()
	if err != nil {
		return "", err
	}
	return detail.Status, nil
}

func (u *mqlArtifactoryUser) groups() ([]any, error) {
	detail, err := u.account()
	if err != nil {
		return nil, err
	}
	return strSliceToAny(detail.Groups), nil
}

func (u *mqlArtifactoryUser) groupRefs() ([]any, error) {
	detail, err := u.account()
	if err != nil {
		return nil, err
	}

	res := []any{}
	for _, name := range detail.Groups {
		group, err := findGroup(u.MqlRuntime, name)
		if err != nil {
			return nil, err
		}
		if group != nil {
			res = append(res, group)
		}
	}
	return res, nil
}

func (u *mqlArtifactoryUser) lastLoggedIn() (*time.Time, error) {
	detail, err := u.account()
	if err != nil {
		return nil, err
	}
	return detail.LastLoggedIn.Time(), nil
}

func (u *mqlArtifactoryUser) permissionTargets() ([]any, error) {
	return permissionTargetsFor(u.MqlRuntime, principalUser, u.Name.Data)
}

// findUser looks up an account in the instance's user list, which the root
// resource fetches once. A name the list does not hold reports nil, which
// callers turn into a null reference rather than an error.
func findUser(runtime *plugin.Runtime, name string) (*mqlArtifactoryUser, error) {
	if name == "" {
		return nil, nil
	}

	root, err := getArtifactory(runtime)
	if err != nil {
		return nil, err
	}
	users := root.GetUsers()
	if users.Error != nil {
		return nil, users.Error
	}

	for _, it := range users.Data {
		user, ok := it.(*mqlArtifactoryUser)
		if ok && user.Name.Data == name {
			return user, nil
		}
	}
	return nil, nil
}

// boolValue reads a flag the API omits on an account it does not apply to. An
// omitted flag is the safe reading, false, rather than an error.
func boolValue(v *bool) bool {
	return v != nil && *v
}
