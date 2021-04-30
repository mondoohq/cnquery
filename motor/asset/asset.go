package asset

import (
	fmt "fmt"
	"github.com/rs/zerolog/log"
)

//go:generate protoc --proto_path=$PWD:. --go_out=. --go_opt=paths=source_relative --falcon_out=. --iam-actions_out=. asset.proto

func (a *Asset) HumanName() string {
	if a == nil {
		return ""
	}

	if a.Platform != nil {
		return fmt.Sprintf("%s (%s)", a.Name, a.Platform.Kind.Name())
	}

	return a.Name
}

func (a *Asset) EnsurePlatformID(ids ...string) {
	if a.PlatformIds == nil {
		a.PlatformIds = ids
		return
	}

	// check if the id is already included
	keys := map[string]bool{}
	for _, k := range a.PlatformIds {
		keys[k] = true
	}

	// append entry
	for _, id := range ids {
		_, ok := keys[id]
		if !ok {
			a.PlatformIds = append(a.PlatformIds, id)
		}
	}
}

func (a *Asset) AddPlatformID(ids ...string) {
	if a.PlatformIds == nil {
		a.PlatformIds = []string{}
	}

	a.PlatformIds = append(a.PlatformIds, ids...)
}

// AddLabels adds the provided labels
// existing labels with the same key will be overwritten
func (a *Asset) AddLabels(labels map[string]string) {
	if a.Labels == nil {
		a.Labels = map[string]string{}
	}

	// copy labels
	for k := range labels {
		a.Labels[k] = labels[k]
	}
}

func NewState(state string) State {
	switch state {
	case "STATE_UNKNOWN":
		return State_STATE_UNKNOWN
	case "STATE_ERROR":
		return State_STATE_ERROR
	case "STATE_PENDING":
		return State_STATE_PENDING
	case "STATE_RUNNING":
		return State_STATE_RUNNING
	case "STATE_STOPPING":
		return State_STATE_STOPPING
	case "STATE_STOPPED":
		return State_STATE_STOPPED
	case "STATE_SHUTDOWN":
		return State_STATE_SHUTDOWN
	case "STATE_TERMINATED":
		return State_STATE_TERMINATED
	case "STATE_REBOOT":
		return State_STATE_REBOOT
	case "STATE_ONLINE":
		return State_STATE_ONLINE
	case "STATE_OFFLINE":
		return State_STATE_OFFLINE
	case "STATE_DELETED":
		return State_STATE_DELETED
	default:
		log.Debug().Str("state", state).Msg("unknown asset state")
		return State_STATE_UNKNOWN
	}
}
