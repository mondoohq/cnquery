// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"github.com/packethost/packngo"
	"go.mondoo.com/cnquery/v10/llx"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/plugin"
)

func (r *mqlEquinixMetalUser) id() (string, error) {
	return r.Url.Data, r.Url.Error
}

func newMqlUser(runtime *plugin.Runtime, user *packngo.User) (*mqlEquinixMetalUser, error) {
	created, _ := parseEquinixTime(user.Created)
	updated, _ := parseEquinixTime(user.Updated)

	var twitter, facebook, linkedin string
	if user.SocialAccounts != nil {
		twitter = user.SocialAccounts.Twitter
		linkedin = user.SocialAccounts.LinkedIn
		// TODO: let's update the used fields here, I'm not sure which ones are needed (dom)
	}

	mqlEquinixUser, err := CreateResource(runtime, "equinix.metal.user", map[string]*llx.RawData{
		"url":           llx.StringData(user.URL),
		"id":            llx.StringData(user.ID),
		"firstName":     llx.StringData(user.FirstName),
		"lastName":      llx.StringData(user.LastName),
		"fullName":      llx.StringData(user.FullName),
		"email":         llx.StringData(user.Email),
		"phoneNumber":   llx.StringData(user.PhoneNumber),
		"twitter":       llx.StringData(twitter),
		"facebook":      llx.StringData(facebook),
		"linkedin":      llx.StringData(linkedin),
		"timezone":      llx.StringData(user.TimeZone),
		"twoFactorAuth": llx.StringData(user.TwoFactorAuth),
		"avatarUrl":     llx.StringData(user.AvatarURL),
		"createdAt":     llx.TimeData(created),
		"updatedAt":     llx.TimeData(updated),
	})
	if err != nil {
		return nil, err
	}

	return mqlEquinixUser.(*mqlEquinixMetalUser), nil
}
