package slack

import (
	"time"

	"github.com/slack-go/slack"
	"go.mondoo.com/cnquery/resources"
	"go.mondoo.com/cnquery/resources/packs/core"
)

func (s *mqlSlack) GetConversations() ([]interface{}, error) {
	op, err := slackProvider(s.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}
	client := op.Client()

	list := []interface{}{}

	// https://api.slack.com/methods/conversations.list
	// scopes: channels:read, groups:read, im:read, mpim:read
	opts := &slack.GetConversationsParameters{
		Limit: 1000, // use maximum
		Types: []string{"public_channel", "private_channel", "mpim", "im"},
	}

	for {
		conversations, cursor, err := client.GetConversations(opts)
		if err != nil {
			return nil, err
		}
		for i := range conversations {
			mqlUser, err := newMqlSlackConversation(s.MotorRuntime, conversations[i])
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

// custom object to make sure the json values match and the time is properly parsed
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

func newMqlSlackConversation(runtime *resources.Runtime, conversation slack.Channel) (interface{}, error) {
	topic, err := core.JsonToDict(newTopic(conversation.Topic))
	if err != nil {
		return nil, err
	}

	purpose, err := core.JsonToDict(newPurpose(conversation.Purpose))
	if err != nil {
		return nil, err
	}

	created := conversation.Created.Time()

	var creator interface{}

	if conversation.Creator != "" {
		creator, err = runtime.CreateResource("slack.user",
			"id", conversation.Creator,
		)
		if err != nil {
			return nil, err
		}
	}

	return runtime.CreateResource("slack.conversation",
		"id", conversation.ID,
		"name", conversation.Name,
		"creator", creator,
		"created", &created,
		"locale", conversation.Locale,
		"topic", topic,
		"purpose", purpose,
		"isArchived", conversation.IsArchived,
		"isOpen", conversation.IsOpen,
		"isPrivate", conversation.IsPrivate,
		"isIM", conversation.IsIM,
		"isMpim", conversation.IsMpIM,
		"isGroup", conversation.IsGroup,
		"isChannel", conversation.IsChannel,
		"isShared", conversation.IsShared,
		"isExtShared", conversation.IsExtShared,
		"isPendingExtShared", conversation.IsPendingExtShared,
		"isOrgShared", conversation.IsOrgShared,
		"priority", conversation.Priority,
	)
}

func (o *mqlSlackConversation) id() (string, error) {
	id, err := o.Id()
	if err != nil {
		return "", err
	}
	return "slack.conversation/" + id, nil
}

func (s *mqlSlackConversation) GetMembers() ([]interface{}, error) {
	op, err := slackProvider(s.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	var list []interface{}

	isChannel, err := s.IsChannel()
	if err != nil {
		return nil, err
	}
	if !isChannel {
		return list, nil
	}

	id, err := s.Id()
	if err != nil {
		return nil, err
	}

	client := op.Client()

	opts := &slack.GetUsersInConversationParameters{
		ChannelID: id,
		Limit:     1000,
	}

	for {
		members, cursor, err := client.GetUsersInConversation(opts)
		if err != nil {
			return nil, err
		}

		for i := range members {
			user, err := s.MotorRuntime.CreateResource("slack.user",
				"id", members[i],
			)
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
