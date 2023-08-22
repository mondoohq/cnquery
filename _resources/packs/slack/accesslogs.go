// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package slack

import (
	"time"

	"github.com/slack-go/slack"
	"go.mondoo.com/cnquery/resources"
)

func (s *mqlSlack) GetAccessLogs() ([]interface{}, error) {
	op, err := slackProvider(s.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}
	client := op.Client()

	accessLogs, _, err := client.GetAccessLogs(slack.AccessLogParameters{
		Count: 1000,
	})
	if err != nil {
		return nil, err
	}
	list := []interface{}{}
	for i := range accessLogs {
		mqlUser, err := newMqlSlackLogin(s.MotorRuntime, accessLogs[i])
		if err != nil {
			return nil, err
		}
		list = append(list, mqlUser)
	}

	return list, nil
}

func newMqlSlackLogin(runtime *resources.Runtime, login slack.Login) (interface{}, error) {
	dateFirst := time.Unix(int64(login.DateFirst), 0)
	dateLast := time.Unix(int64(login.DateLast), 0)
	return runtime.CreateResource("slack.login",
		"userID", login.UserID,
		"username", login.Username,
		"count", int64(login.Count),
		"ip", login.IP,
		"userAgent", login.UserAgent,
		"isp", login.ISP,
		"country", login.Country,
		"region", login.Region,
		"dateFirst", &dateFirst,
		"dateLast", &dateLast,
	)
}

func (o *mqlSlackLogin) id() (string, error) {
	id, err := o.UserID()
	if err != nil {
		return "", err
	}

	ip, err := o.Ip()
	if err != nil {
		return "", err
	}

	ua, err := o.UserAgent()
	if err != nil {
		return "", err
	}

	return "slack.login/user/" + id + "/ip/" + ip + "/useragent/" + ua, nil
}
