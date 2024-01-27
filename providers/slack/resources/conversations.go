// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"errors"
	"time"

	"github.com/slack-go/slack"
	"go.mondoo.com/cnquery/v10/llx"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/util/convert"
	"go.mondoo.com/cnquery/v10/providers/slack/connection"
)

func (o *mqlSlackConversations) id() (string, error) {
	return "slack.conversations", nil
}

func (s *mqlSlackConversations) listChannels(excludeArchived bool, types ...string) ([]interface{}, error) {
	conn := s.MqlRuntime.Connection.(*connection.SlackConnection)
	client := conn.Client()
	if client == nil {
		return nil, errors.New("cannot retrieve new data while using a mock connection")
	}

	list := []interface{}{}

	// https://api.slack.com/methods/conversations.list
	// scopes: channels:read, groups:read, im:read, mpim:read
	opts := &slack.GetConversationsParameters{
		Limit:           999, // use maximum, must be lower than 1000
		Types:           types,
		ExcludeArchived: excludeArchived,
	}

	for {
		conversations, cursor, err := client.GetConversations(opts)
		if err != nil {
			return nil, err
		}
		for i := range conversations {
			mqlUser, err := newMqlSlackConversation(s.MqlRuntime, conversations[i])
			if err != nil {
				return nil, err
			}
			list = append(list, mqlUser)
		}
		// check if we are at the end of pagination
		if cursor == "" {
			break
		}
		opts.Cursor = cursor
	}

	return list, nil
}

func (s *mqlSlackConversations) list() ([]interface{}, error) {
	return s.listChannels(false, "public_channel", "private_channel", "mpim", "im")
}

func (s *mqlSlackConversations) privateChannels() ([]interface{}, error) {
	return s.listChannels(true, "private_channel")
}

func (s *mqlSlackConversations) publicChannels() ([]interface{}, error) {
	return s.listChannels(true, "public_channel")
}

func (s *mqlSlackConversations) directMessages() ([]interface{}, error) {
	return s.listChannels(true, "mpim", "im")
}

type topic struct {
	Value   string     `json:"value"`
	Creator string     `json:"creator"`
	LastSet *time.Time `json:"lastSet"`
}

func newTopic(t slack.Topic) topic {
	lastSet := t.LastSet.Time()
	return topic{
		Value:   t.Value,
		Creator: t.Creator,
		LastSet: &lastSet,
	}
}

// custom object to make sure the json values match and the time is properly parsed

type purpose struct {
	Value   string     `json:"value"`
	Creator string     `json:"creator"`
	LastSet *time.Time `json:"lastSet"`
}

func newPurpose(p slack.Purpose) purpose {
	lastSet := p.LastSet.Time()
	return purpose{
		Value:   p.Value,
		Creator: p.Creator,
		LastSet: &lastSet,
	}
}

func newMqlSlackConversation(runtime *plugin.Runtime, conversation slack.Channel) (interface{}, error) {
	topic, err := convert.JsonToDict(newTopic(conversation.Topic))
	if err != nil {
		return nil, err
	}

	purpose, err := convert.JsonToDict(newPurpose(conversation.Purpose))
	if err != nil {
		return nil, err
	}

	created := conversation.Created.Time()

	r, err := CreateResource(runtime, "slack.conversation", map[string]*llx.RawData{
		"id":                 llx.StringData(conversation.ID),
		"name":               llx.StringData(conversation.Name),
		"created":            llx.TimeData(created),
		"locale":             llx.StringData(conversation.Locale),
		"topic":              llx.DictData(topic),
		"purpose":            llx.DictData(purpose),
		"isArchived":         llx.BoolData(conversation.IsArchived),
		"isOpen":             llx.BoolData(conversation.IsOpen),
		"isPrivate":          llx.BoolData(conversation.IsPrivate),
		"isIM":               llx.BoolData(conversation.IsIM),
		"isMpim":             llx.BoolData(conversation.IsMpIM),
		"isGroup":            llx.BoolData(conversation.IsGroup),
		"isChannel":          llx.BoolData(conversation.IsChannel),
		"isShared":           llx.BoolData(conversation.IsShared),
		"isExtShared":        llx.BoolData(conversation.IsExtShared),
		"isPendingExtShared": llx.BoolData(conversation.IsPendingExtShared),
		"isOrgShared":        llx.BoolData(conversation.IsOrgShared),
		"priority":           llx.FloatData(conversation.Priority),
	})
	if err != nil {
		return nil, err
	}
	mqlConversation := r.(*mqlSlackConversation)
	mqlConversation.creatorId = conversation.Creator
	return mqlConversation, nil
}

type mqlSlackConversationInternal struct {
	creatorId string
}

func (x *mqlSlackConversation) id() (string, error) {
	return "slack.conversation/" + x.Id.Data, nil
}

func (s *mqlSlackConversation) creator() (*mqlSlackUser, error) {
	if s.creatorId == "" {
		s.Creator = plugin.TValue[*mqlSlackUser]{State: plugin.StateIsSet | plugin.StateIsNull}
		return nil, nil
	}

	r, err := NewResource(s.MqlRuntime, "slack.user", map[string]*llx.RawData{
		"id": llx.StringData(s.creatorId),
	})
	if err != nil {
		return nil, err
	}
	return r.(*mqlSlackUser), nil
}

func (s *mqlSlackConversation) members() ([]interface{}, error) {
	conn := s.MqlRuntime.Connection.(*connection.SlackConnection)
	client := conn.Client()
	if client == nil {
		return nil, errors.New("cannot retrieve new data while using a mock connection")
	}

	var list []interface{}
	isChannel := s.IsChannel.Data
	if !isChannel {
		return list, nil
	}

	opts := &slack.GetUsersInConversationParameters{
		ChannelID: s.Id.Data,
		Limit:     999, // must be less than 1000
	}

	for {
		members, cursor, err := client.GetUsersInConversation(opts)
		if err != nil {
			return nil, err
		}

		for i := range members {
			user, err := NewResource(s.MqlRuntime, "slack.user", map[string]*llx.RawData{
				"id": llx.StringData(members[i]),
			})
			if err != nil {
				return nil, err
			}
			list = append(list, user)
		}

		// check if we are at the end of pagination
		if cursor == "" {
			break
		}
		opts.Cursor = cursor
	}

	return list, nil
}
