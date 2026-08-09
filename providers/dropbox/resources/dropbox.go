// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	dropboxsdk "github.com/dropbox/dropbox-sdk-go-unofficial/v6/dropbox"
	"github.com/dropbox/dropbox-sdk-go-unofficial/v6/dropbox/team"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
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

// initDropboxTeam reads the team-wide identity and sharing configuration
// singleton. There is exactly one team per connection, so it resolves entirely
// from the connection and uses the team ID as its cache key.
func initDropboxTeam(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 1 {
		return args, nil, nil
	}

	conn := runtime.Connection.(*connection.DropboxConnection)
	client := conn.Client()

	info, err := conn.TeamInfo()
	if err != nil {
		return nil, nil, err
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

	args["__id"] = llx.StringData(info.TeamId)
	args["id"] = llx.StringData(info.TeamId)
	args["name"] = llx.StringData(info.Name)
	args["numLicensedUsers"] = llx.IntData(int64(info.NumLicensedUsers))
	args["numProvisionedUsers"] = llx.IntData(int64(info.NumProvisionedUsers))
	args["numUsedLicenses"] = llx.IntData(int64(info.NumUsedLicenses))
	args["externalSharingAllowed"] = llx.BoolData(externalSharingAllowed)
	args["publicSharingAllowed"] = llx.BoolData(publicSharingAllowed)
	args["uploadApiRateLimit"] = llx.IntData(uploadRateLimit)

	return args, nil, nil
}
