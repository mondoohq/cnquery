// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"sync"

	"go.mondoo.com/mql/llx"
	"go.mondoo.com/mql/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/providers/zoom/connection"
	"go.mondoo.com/mql/types"
)

// mqlZoomAccountInternal caches the account settings behind lazy fetches. The
// Zoom settings endpoint splits what this resource reads across two views: the
// un-optioned response carries the recording and sign-in security sections,
// while the meeting-security defaults are only returned for
// `?option=meeting_security`. Each view is fetched at most once and guarded by
// double-check locking, so querying any subset of the fields backed by a view
// costs exactly one call for it, and querying none costs nothing.
type mqlZoomAccountInternal struct {
	settingsFetched bool
	settings        *connection.AccountSettings
	settingsLock    sync.Mutex

	meetingSecurityFetched bool
	meetingSecurity        *connection.MeetingSecuritySettings
	meetingSecurityLock    sync.Mutex

	meetingAuthFetched bool
	meetingAuth        *connection.MeetingAuthenticationSettings
	meetingAuthLock    sync.Mutex
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

// fetchSettings performs the single un-optioned account-settings GET this
// resource's recording and sign-in fields read from, caching the result behind
// double-check locking so concurrent field access only fetches once.
func (a *mqlZoomAccount) fetchSettings() (*connection.AccountSettings, error) {
	if a.settingsFetched {
		return a.settings, nil
	}
	a.settingsLock.Lock()
	defer a.settingsLock.Unlock()
	if a.settingsFetched {
		return a.settings, nil
	}

	conn := a.MqlRuntime.Connection.(*connection.ZoomConnection)
	settings, err := conn.Client().GetAccountSettings(context.Background(), conn.AccountID())
	if err != nil {
		return nil, err
	}
	a.settings = settings
	a.settingsFetched = true
	return a.settings, nil
}

// fetchMeetingSecurity performs the `?option=meeting_security` account-settings
// GET the meeting* fields read from. The meeting_security object is absent from
// the un-optioned response, so it needs its own call rather than a second read
// of fetchSettings.
func (a *mqlZoomAccount) fetchMeetingSecurity() (*connection.MeetingSecuritySettings, error) {
	if a.meetingSecurityFetched {
		return a.meetingSecurity, nil
	}
	a.meetingSecurityLock.Lock()
	defer a.meetingSecurityLock.Unlock()
	if a.meetingSecurityFetched {
		return a.meetingSecurity, nil
	}

	conn := a.MqlRuntime.Connection.(*connection.ZoomConnection)
	ms, err := conn.Client().GetAccountMeetingSecurity(context.Background(), conn.AccountID())
	if err != nil {
		return nil, err
	}
	a.meetingSecurity = ms
	a.meetingSecurityFetched = true
	return a.meetingSecurity, nil
}

// fetchMeetingAuthentication performs the `?option=meeting_authentication`
// account-settings GET. Zoom reports the meeting-authentication requirement as
// a top-level boolean of that view, not as a member of meeting_security, so it
// needs its own call.
func (a *mqlZoomAccount) fetchMeetingAuthentication() (*connection.MeetingAuthenticationSettings, error) {
	if a.meetingAuthFetched {
		return a.meetingAuth, nil
	}
	a.meetingAuthLock.Lock()
	defer a.meetingAuthLock.Unlock()
	if a.meetingAuthFetched {
		return a.meetingAuth, nil
	}

	conn := a.MqlRuntime.Connection.(*connection.ZoomConnection)
	auth, err := conn.Client().GetAccountMeetingAuthentication(context.Background(), conn.AccountID())
	if err != nil {
		return nil, err
	}
	a.meetingAuth = auth
	a.meetingAuthFetched = true
	return a.meetingAuth, nil
}

func (a *mqlZoomAccount) meetingWaitingRoomEnabled() (bool, error) {
	s, err := a.fetchMeetingSecurity()
	if err != nil {
		return false, err
	}
	return s.WaitingRoom, nil
}

func (a *mqlZoomAccount) meetingPasscodeRequired() (bool, error) {
	s, err := a.fetchMeetingSecurity()
	if err != nil {
		return false, err
	}
	return s.MeetingPasswordRequirement, nil
}

func (a *mqlZoomAccount) meetingPmiPasscodeRequired() (bool, error) {
	s, err := a.fetchMeetingSecurity()
	if err != nil {
		return false, err
	}
	return s.PmiPasswordRequirement, nil
}

func (a *mqlZoomAccount) meetingEncryptionType() (string, error) {
	s, err := a.fetchMeetingSecurity()
	if err != nil {
		return "", err
	}
	return s.EncryptionType, nil
}

func (a *mqlZoomAccount) meetingE2eeAvailable() (bool, error) {
	s, err := a.fetchMeetingSecurity()
	if err != nil {
		return false, err
	}
	return s.E2eeAvailable, nil
}

func (a *mqlZoomAccount) meetingAuthenticationRequired() (bool, error) {
	s, err := a.fetchMeetingAuthentication()
	if err != nil {
		return false, err
	}
	return s.MeetingAuthentication, nil
}

// meetingOnlyAccountUsersCanJoin is retained for backwards compatibility and
// reports the same value as meetingSignedInUsersOnly.
func (a *mqlZoomAccount) meetingOnlyAccountUsersCanJoin() (bool, error) {
	return a.meetingSignedInUsersOnly()
}

func (a *mqlZoomAccount) meetingSignedInUsersOnly() (bool, error) {
	s, err := a.fetchSettings()
	if err != nil {
		return false, err
	}
	return s.ScheduleMeeting.EnforceLogin, nil
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
