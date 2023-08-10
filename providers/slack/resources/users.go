// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"
	"time"

	"github.com/slack-go/slack"
	"go.mondoo.com/cnquery/llx"
	"go.mondoo.com/cnquery/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/providers-sdk/v1/util/convert"
	"go.mondoo.com/cnquery/providers/slack/connection"
)

func (o *mqlSlackUsers) id() (string, error) {
	return "slack.users", nil
}

func (s *mqlSlackUsers) list() ([]interface{}, error) {
	conn := s.MqlRuntime.Connection.(*connection.SlackConnection)
	client := conn.Client()
	ctx := context.Background()

	// requires users:read scope
	users, err := client.GetUsersContext(ctx)
	if err != nil {
		return nil, err
	}
	var list []interface{}
	for i := range users {
		mqlUser, err := newMqlSlackUser(s.MqlRuntime, users[i])
		if err != nil {
			return nil, err
		}
		list = append(list, mqlUser)
	}

	return list, nil
}

func (s *mqlSlackUsers) bots() ([]interface{}, error) {
	all, err := s.list()
	if err != nil {
		return all, err
	}

	res := []interface{}{}
	for i := range all {
		cur := all[i]
		usr := cur.(*mqlSlackUser)
		isBot := usr.IsBot.Data
		if isBot == true {
			res = append(res, cur)
		}
	}
	return res, nil
}

func (s *mqlSlackUsers) members() ([]interface{}, error) {
	all, err := s.list()
	if err != nil {
		return all, err
	}

	res := []interface{}{}
	for i := range all {
		cur := all[i]
		usr := cur.(*mqlSlackUser)
		isBot := usr.IsBot.Data
		if isBot != true {
			res = append(res, cur)
		}
	}
	return res, nil
}

func (s *mqlSlackUsers) admins() ([]interface{}, error) {
	all, err := s.list()
	if err != nil {
		return all, err
	}

	res := []interface{}{}
	for i := range all {
		cur := all[i]
		usr := cur.(*mqlSlackUser)
		isAdmin := usr.IsAdmin.Data
		if err != nil {
			return nil, err
		}
		if isAdmin == true {
			res = append(res, cur)
		}
	}
	return res, nil
}

func (s *mqlSlackUsers) owners() ([]interface{}, error) {
	all, err := s.list()
	if err != nil {
		return all, err
	}

	res := []interface{}{}
	for i := range all {
		cur := all[i]
		usr := cur.(*mqlSlackUser)
		isOwner := usr.IsOwner.Data
		if isOwner == true {
			res = append(res, cur)
		}
	}
	return res, nil
}

type userProfile struct {
	FirstName             string     `json:"firstName"`
	LastName              string     `json:"lastName"`
	RealName              string     `json:"realName"`
	RealNameNormalized    string     `json:"realNameNormalized"`
	DisplayName           string     `json:"displayName"`
	DisplayNameNormalized string     `json:"displayNameNormalized"`
	Email                 string     `json:"email"`
	Skype                 string     `json:"skype"`
	Phone                 string     `json:"phone"`
	Title                 string     `json:"title"`
	BotID                 string     `json:"botId,omitempty"`
	ApiAppID              string     `json:"apiAppId,omitempty"`
	StatusText            string     `json:"statusText,omitempty"`
	StatusEmoji           string     `json:"statusEmoji,omitempty"`
	StatusExpiration      *time.Time `json:"statusExpiration"`
	Team                  string     `json:"team"`
}

func newUserProfile(u slack.UserProfile) userProfile {
	statusExpiration := time.Unix(int64(u.StatusExpiration), 0)

	return userProfile{
		FirstName:             u.FirstName,
		LastName:              u.LastName,
		RealName:              u.RealName,
		RealNameNormalized:    u.RealNameNormalized,
		DisplayName:           u.DisplayName,
		DisplayNameNormalized: u.DisplayNameNormalized,
		Email:                 u.Email,
		Skype:                 u.Skype,
		Phone:                 u.Phone,
		Title:                 u.Title,
		BotID:                 u.BotID,
		ApiAppID:              u.ApiAppID,
		StatusText:            u.StatusText,
		StatusEmoji:           u.StatusEmoji,
		StatusExpiration:      &statusExpiration,
		Team:                  u.Team,
	}
}

func newMqlSlackUser(runtime *plugin.Runtime, user slack.User) (plugin.Resource, error) {
	var enterpriseUser interface{}

	userProfile, err := convert.JsonToDict(newUserProfile(user.Profile))
	if err != nil {
		return nil, err
	}

	if user.Enterprise.ID != "" {
		var err error
		enterpriseUser, err = newMqlSlackEnterpriseUser(runtime, user.Enterprise)
		if err != nil {
			return nil, err
		}
	}

	twoFactoryType := ""
	if user.TwoFactorType != nil {
		twoFactoryType = *user.TwoFactorType
	}

	return CreateResource(runtime, "slack.user", map[string]*llx.RawData{
		"id":                llx.StringData(user.ID),
		"teamId":            llx.StringData(user.TeamID),
		"name":              llx.StringData(user.Name),
		"deleted":           llx.BoolData(user.Deleted),
		"color":             llx.StringData(user.Color),
		"realName":          llx.StringData(user.RealName),
		"timeZone":          llx.StringData(user.TZ),
		"timeZoneLabel":     llx.StringData(user.TZLabel),
		"timeZoneOffset":    llx.IntData(int64(user.TZOffset)),
		"isBot":             llx.BoolData(user.IsBot),
		"isAdmin":           llx.BoolData(user.IsAdmin),
		"isOwner":           llx.BoolData(user.IsOwner),
		"isPrimaryOwner":    llx.BoolData(user.IsPrimaryOwner),
		"isRestricted":      llx.BoolData(user.IsRestricted),
		"isUltraRestricted": llx.BoolData(user.IsUltraRestricted),
		"isStranger":        llx.BoolData(user.IsStranger),
		"isAppUser":         llx.BoolData(user.IsAppUser),
		"isInvitedUser":     llx.BoolData(user.IsInvitedUser),
		"has2FA":            llx.BoolData(user.Has2FA),
		"twoFactorType":     llx.StringData(twoFactoryType),
		"hasFiles":          llx.BoolData(user.HasFiles),
		"presence":          llx.StringData(user.Presence),
		"locale":            llx.StringData(user.Locale),
		"profile":           llx.DictData(userProfile),
		"enterpriseUser":    llx.DictData(enterpriseUser),
	})
}

func (x *mqlSlackUser) id() (string, error) {
	return "slack.user/" + x.TeamId.Data + "/" + x.Id.Data, nil
}

func newMqlSlackEnterpriseUser(runtime *plugin.Runtime, user slack.EnterpriseUser) (interface{}, error) {
	return CreateResource(runtime, "slack.enterpriseUser", map[string]*llx.RawData{
		"id":             llx.StringData(user.ID),
		"enterpriseId":   llx.StringData(user.EnterpriseID),
		"enterpriseName": llx.StringData(user.EnterpriseName),
		"isAdmin":        llx.BoolData(user.IsAdmin),
		"isOwner":        llx.BoolData(user.IsOwner),
	})
}

func (x *mqlSlackEnterpriseUser) id() (string, error) {
	return "slack.enterpriseUser/" + x.EnterpriseId.Data + "/" + x.Id.Data, nil
}

// init method for user
func (s *mqlSlackUser) init(args map[string]interface{}) (map[string]interface{}, *mqlSlackUser, error) {
	// we only look up the user, if we have been supplied by its id and nothing else
	raw, ok := args["id"]
	if !ok || len(args) != 1 {
		return args, nil, nil
	}

	id, ok := raw.(string)

	conn := s.MqlRuntime.Connection.(*connection.SlackConnection)
	client := conn.Client()

	users, err := client.GetUsersInfo(id)
	if err != nil {
		return nil, nil, err
	}

	var userList []slack.User
	if users != nil {
		userList = *users
	}

	if len(userList) != 1 {
		return nil, nil, errors.New("user " + id + " not available")
	}

	usr, err := newMqlSlackUser(s.MqlRuntime, userList[0])
	if err != nil {
		return nil, nil, err
	}

	return nil, usr.(*mqlSlackUser), nil
}
