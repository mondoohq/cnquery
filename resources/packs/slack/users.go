package slack

import (
	"context"
	"errors"

	"github.com/slack-go/slack"
	"go.mondoo.com/cnquery/resources"
)

func (o *mqlSlackUsers) id() (string, error) {
	return "slack.users", nil
}

func (s *mqlSlackUsers) GetList() ([]interface{}, error) {
	op, err := slackProvider(s.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}
	ctx := context.Background()
	client := op.Client()

	// requires users:read scope
	users, err := client.GetUsersContext(ctx)
	if err != nil {
		return nil, err
	}
	var list []interface{}
	for i := range users {
		mqlUser, err := newMqlSlackUser(s.MotorRuntime, users[i])
		if err != nil {
			return nil, err
		}
		list = append(list, mqlUser)
	}

	return list, nil
}

func (s *mqlSlackUsers) GetBots() ([]interface{}, error) {
	all, err := s.GetList()
	if err != nil {
		return all, err
	}

	res := []interface{}{}
	for i := range all {
		cur := all[i]
		usr := cur.(SlackUser)
		isBot, err := usr.IsBot()
		if err != nil {
			return nil, err
		}
		if isBot == true {
			res = append(res, cur)
		}
	}
	return res, nil
}

func (s *mqlSlackUsers) GetMembers() ([]interface{}, error) {
	all, err := s.GetList()
	if err != nil {
		return all, err
	}

	res := []interface{}{}
	for i := range all {
		cur := all[i]
		usr := cur.(SlackUser)
		isBot, err := usr.IsBot()
		if err != nil {
			return nil, err
		}
		if isBot != true {
			res = append(res, cur)
		}
	}
	return res, nil
}

func (s *mqlSlackUsers) GetAdmins() ([]interface{}, error) {
	all, err := s.GetList()
	if err != nil {
		return all, err
	}

	res := []interface{}{}
	for i := range all {
		cur := all[i]
		usr := cur.(SlackUser)
		isAdmin, err := usr.IsAdmin()
		if err != nil {
			return nil, err
		}
		if isAdmin == true {
			res = append(res, cur)
		}
	}
	return res, nil
}

func (s *mqlSlackUsers) GetOwner() ([]interface{}, error) {
	all, err := s.GetList()
	if err != nil {
		return all, err
	}

	res := []interface{}{}
	for i := range all {
		cur := all[i]
		usr := cur.(SlackUser)
		isOwner, err := usr.IsOwner()
		if err != nil {
			return nil, err
		}
		if isOwner == true {
			res = append(res, cur)
		}
	}
	return res, nil
}

func newMqlSlackUser(runtime *resources.Runtime, user slack.User) (interface{}, error) {
	var enterpriseUser interface{}

	if user.Enterprise.ID != "" {
		var err error
		enterpriseUser, err = newMqlSlackEnterpriseUser(runtime, user.Enterprise)
		if err != nil {
			return nil, err
		}
	}

	return runtime.CreateResource("slack.user",
		"id", user.ID,
		"teamId", user.TeamID,
		"name", user.Name,
		"deleted", user.Deleted,
		"color", user.Color,
		"realName", user.RealName,
		"timeZone", user.TZ,
		"timeZoneLabel", user.TZLabel,
		"timeZoneOffset", int64(user.TZOffset),
		"isBot", user.IsBot,
		"isAdmin", user.IsAdmin,
		"isOwner", user.IsOwner,
		"isPrimaryOwner", user.IsPrimaryOwner,
		"isRestricted", user.IsRestricted,
		"isUltraRestricted", user.IsUltraRestricted,
		"isStranger", user.IsStranger,
		"isAppUser", user.IsAppUser,
		"isInvitedUser", user.IsInvitedUser,
		"has2FA", user.Has2FA,
		"hasFiles", user.HasFiles,
		"presence", user.Presence,
		"locale", user.Locale,
		"enterpriseUser", enterpriseUser,
	)
}

func (o *mqlSlackUser) id() (string, error) {
	teamID, err := o.TeamId()
	if err != nil {
		return "", err
	}
	id, err := o.Id()
	if err != nil {
		return "", err
	}
	return "slack.user/" + teamID + "/" + id, nil
}

func newMqlSlackEnterpriseUser(runtime *resources.Runtime, user slack.EnterpriseUser) (interface{}, error) {
	return runtime.CreateResource("slack.enterpriseUser",
		"id", user.ID,
		"enterpriseId", user.EnterpriseID,
		"enterpriseName", user.EnterpriseName,
		"isAdmin", user.IsAdmin,
		"isOwner", user.IsOwner,
	)
}

func (o *mqlSlackEnterpriseUser) id() (string, error) {
	enterpriseID, err := o.EnterpriseId()
	if err != nil {
		return "", err
	}
	id, err := o.Id()
	if err != nil {
		return "", err
	}
	return "slack.enterpriseUser/" + enterpriseID + "/" + id, nil
}

// init method for user
func (s *mqlSlackUser) init(args *resources.Args) (*resources.Args, SlackUser, error) {
	if len(*args) > 2 {
		return args, nil, nil
	}

	idRaw := (*args)["id"]
	if idRaw == nil {
		return args, nil, nil
	}

	id, ok := idRaw.(string)
	if !ok {
		return args, nil, nil
	}

	op, err := slackProvider(s.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, nil, err
	}

	client := op.Client()
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

	mqlSlackUser, err := newMqlSlackUser(s.MotorRuntime, userList[0])
	if err != nil {
		return nil, nil, err
	}

	usr := mqlSlackUser.(SlackUser)
	return nil, usr, nil
}
