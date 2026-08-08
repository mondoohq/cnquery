// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"sync"

	"go.mondoo.com/mql/v13/llx"
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

// account reads the account's identity eagerly and defers its
// meeting-security, recording, and sign-in settings to lazy field accessors.
func (r *mqlZoom) account() (*mqlZoomAccount, error) {
	conn := r.conn()
	client := conn.Client()

	info, err := client.GetAccount(context.Background(), conn.AccountID())
	if err != nil {
		return nil, err
	}

	res, err := CreateResource(r.MqlRuntime, "zoom.account", map[string]*llx.RawData{
		"__id":        llx.StringData(conn.AccountID()),
		"id":          llx.StringData(conn.AccountID()),
		"accountName": llx.StringDataPtr(&info.AccountName),
		"ownerEmail":  llx.StringDataPtr(&info.OwnerEmail),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlZoomAccount), nil
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

// sso reads the account's single sign-on configuration. Its cache key is
// the account ID, since there is exactly one SSO configuration per account.
func (a *mqlZoomAccount) sso() (*mqlZoomAccountSso, error) {
	conn := a.MqlRuntime.Connection.(*connection.ZoomConnection)
	s, err := conn.Client().GetSsoSettings(context.Background(), conn.AccountID())
	if err != nil {
		return nil, err
	}

	res, err := CreateResource(a.MqlRuntime, "zoom.account.sso", map[string]*llx.RawData{
		"__id":                llx.StringData(conn.AccountID() + "/sso"),
		"enabled":             llx.BoolData(s.Enabled),
		"domains":             llx.ArrayData(strToAnyList(s.Domains), types.String),
		"groupMappingEnabled": llx.BoolData(s.GroupMappingEnabled),
		"idpIssuer":           llx.StringData(s.IdpIssuer),
		"idpSsoUrl":           llx.StringData(s.IdpSsoUrl),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlZoomAccountSso), nil
}
