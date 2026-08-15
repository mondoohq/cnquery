// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"sync"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/zoom/connection"
	"go.mondoo.com/mql/v13/types"
)

// mqlZoomAccountInternal caches the account settings behind a single lazy
// fetch. All the meeting-security, recording, and sign-in fields on
// zoom.account are computed accessors that share this fetch, so querying
// any subset of them still costs exactly one GET /accounts/{id}/settings
// call, guarded by double-check locking against concurrent field reads.
type mqlZoomAccountInternal struct {
	fetched  bool
	settings *connection.AccountSettings
	lock     sync.Mutex
}

// initZoomAccount populates the account singleton on construction. It is an
// init rather than a zoom accessor because zoom.account is a
// directly-addressable resource: a field of the same dotted name on the parent
// would collide with the resource and leave its fields unset when queried as
// `zoom.account`. It reads the account's identity eagerly and defers the
// meeting-security, recording, and sign-in settings to lazy field accessors.
func initZoomAccount(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	// fast path: the caller already provided a populated resource
	if len(args) > 1 {
		return args, nil, nil
	}

	conn := runtime.Connection.(*connection.ZoomConnection)
	info, err := conn.Client().GetAccount(context.Background(), conn.AccountID())
	if err != nil {
		return nil, nil, err
	}

	args["__id"] = llx.StringData(conn.AccountID())
	args["id"] = llx.StringData(conn.AccountID())
	args["accountName"] = llx.StringDataPtr(&info.AccountName)
	args["ownerEmail"] = llx.StringDataPtr(&info.OwnerEmail)
	return args, nil, nil
}

// fetchSettings performs the single account-settings GET this resource's
// meeting/recording/sign-in fields all read from, caching the result behind
// double-check locking so concurrent field access only fetches once.
func (a *mqlZoomAccount) fetchSettings() (*connection.AccountSettings, error) {
	if a.fetched {
		return a.settings, nil
	}
	a.lock.Lock()
	defer a.lock.Unlock()
	if a.fetched {
		return a.settings, nil
	}

	conn := a.MqlRuntime.Connection.(*connection.ZoomConnection)
	settings, err := conn.Client().GetAccountSettings(context.Background(), conn.AccountID())
	if err != nil {
		return nil, err
	}
	a.settings = settings
	a.fetched = true
	return a.settings, nil
}

func (a *mqlZoomAccount) meetingWaitingRoomEnabled() (bool, error) {
	s, err := a.fetchSettings()
	if err != nil {
		return false, err
	}
	return s.MeetingSecurity.WaitingRoom, nil
}

func (a *mqlZoomAccount) meetingPasscodeRequired() (bool, error) {
	s, err := a.fetchSettings()
	if err != nil {
		return false, err
	}
	return s.MeetingSecurity.MeetingPasswordRequirement, nil
}

func (a *mqlZoomAccount) meetingPmiPasscodeRequired() (bool, error) {
	s, err := a.fetchSettings()
	if err != nil {
		return false, err
	}
	return s.MeetingSecurity.PmiPasswordRequirement, nil
}

func (a *mqlZoomAccount) meetingEncryptionType() (string, error) {
	s, err := a.fetchSettings()
	if err != nil {
		return "", err
	}
	return s.MeetingSecurity.EncryptionType, nil
}

func (a *mqlZoomAccount) meetingE2eeAvailable() (bool, error) {
	s, err := a.fetchSettings()
	if err != nil {
		return false, err
	}
	return s.MeetingSecurity.E2eeAvailable, nil
}

func (a *mqlZoomAccount) meetingAuthenticationRequired() (bool, error) {
	s, err := a.fetchSettings()
	if err != nil {
		return false, err
	}
	return s.MeetingSecurity.MeetingAuthentication, nil
}

func (a *mqlZoomAccount) meetingOnlyAccountUsersCanJoin() (bool, error) {
	s, err := a.fetchSettings()
	if err != nil {
		return false, err
	}
	return s.MeetingSecurity.OnlyAuthenticatedCanJoin, nil
}

func (a *mqlZoomAccount) cloudRecordingEnabled() (bool, error) {
	s, err := a.fetchSettings()
	if err != nil {
		return false, err
	}
	return s.Recording.CloudRecording, nil
}

func (a *mqlZoomAccount) cloudRecordingEncryptionEnabled() (bool, error) {
	s, err := a.fetchSettings()
	if err != nil {
		return false, err
	}
	return s.Recording.CloudRecordingEncryption, nil
}

func (a *mqlZoomAccount) signInSessionTimeoutMinutes() (int64, error) {
	s, err := a.fetchSettings()
	if err != nil {
		return 0, err
	}
	return s.Security.SignInSessionTimeout, nil
}

// initZoomAccountSso populates the account's single sign-on singleton on
// construction. It is an init rather than a zoom.account accessor because
// zoom.account.sso is a directly-addressable resource whose dotted name would
// otherwise collide with an sso field on zoom.account and leave its fields
// unset when queried as `zoom.account.sso`. There is exactly one SSO
// configuration per connected account, so it resolves from the connection
// alone.
func initZoomAccountSso(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	// fast path: the caller already provided a populated resource
	if len(args) > 1 {
		return args, nil, nil
	}

	conn := runtime.Connection.(*connection.ZoomConnection)
	s, err := conn.Client().GetSsoSettings(context.Background(), conn.AccountID())
	if err != nil {
		return nil, nil, err
	}

	args["__id"] = llx.StringData(conn.AccountID() + "/sso")
	args["enabled"] = llx.BoolData(s.Enabled)
	args["domains"] = llx.ArrayData(strToAnyList(s.Domains), types.String)
	args["groupMappingEnabled"] = llx.BoolData(s.GroupMappingEnabled)
	args["idpIssuer"] = llx.StringData(s.IdpIssuer)
	args["idpSsoUrl"] = llx.StringData(s.IdpSsoUrl)
	return args, nil, nil
}
