// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	dropboxsdk "github.com/dropbox/dropbox-sdk-go-unofficial/v6/dropbox"
	"github.com/dropbox/dropbox-sdk-go-unofficial/v6/dropbox/team"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers/dropbox/connection"
)

func (r *mqlDropbox) id() (string, error) {
	return "dropbox", nil
}

// id returns the team ID as the resource's cache key. There is exactly one
// team per connection, so this is a stable, natural key.
func (r *mqlDropboxTeam) id() (string, error) {
	return "dropbox.team/" + r.Id.Data, nil
}

// conn returns the Dropbox connection backing this runtime.
func (r *mqlDropbox) conn() *connection.DropboxConnection {
	return r.MqlRuntime.Connection.(*connection.DropboxConnection)
}

// team reads the team-wide identity and sharing configuration singleton. Its
// cache key is the team ID, since there is exactly one team per connection.
func (r *mqlDropbox) team() (*mqlDropboxTeam, error) {
	conn := r.conn()
	client := conn.Client()

	info, err := conn.TeamInfo()
	if err != nil {
		return nil, err
	}

	var externalSharingAllowed, publicSharingAllowed bool
	var uploadRateLimit int64
	if info.Policies != nil && info.Policies.Sharing != nil {
		sharing := info.Policies.Sharing
		if sharing.SharedFolderMemberPolicy != nil {
			// "team" means only team members may join a shared folder; anything
			// else (anyone, team_and_approved) admits members from outside the team.
			externalSharingAllowed = sharing.SharedFolderMemberPolicy.Tag != "team"
		}
		if sharing.SharedLinkCreatePolicy != nil {
			// "team_only" and "default_no_one" keep new shared links scoped to the
			// team by default; the other tags (default_public, default_team_only)
			// permit a publicly accessible link.
			tag := sharing.SharedLinkCreatePolicy.Tag
			publicSharingAllowed = tag != "team_only" && tag != "default_no_one"
		}
	}

	features, err := client.FeaturesGetValues(&team.FeaturesGetValuesBatchArg{
		Features: []*team.Feature{{Tagged: dropboxsdk.Tagged{Tag: team.FeatureUploadApiRateLimit}}},
	})
	if err == nil {
		for _, v := range features.Values {
			if v.UploadApiRateLimit != nil && v.UploadApiRateLimit.Tag == team.UploadApiRateLimitValueLimit {
				uploadRateLimit = int64(v.UploadApiRateLimit.Limit)
			}
		}
	}

	res, err := CreateResource(r.MqlRuntime, "dropbox.team", map[string]*llx.RawData{
		"__id":                   llx.StringData(info.TeamId),
		"id":                     llx.StringData(info.TeamId),
		"name":                   llx.StringData(info.Name),
		"numLicensedUsers":       llx.IntData(int64(info.NumLicensedUsers)),
		"numProvisionedUsers":    llx.IntData(int64(info.NumProvisionedUsers)),
		"numUsedLicenses":        llx.IntData(int64(info.NumUsedLicenses)),
		"externalSharingAllowed": llx.BoolData(externalSharingAllowed),
		"publicSharingAllowed":   llx.BoolData(publicSharingAllowed),
		"uploadApiRateLimit":     llx.IntData(uploadRateLimit),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlDropboxTeam), nil
}
