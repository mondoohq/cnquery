// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package inventory

import (
	"encoding/json"
	"errors"
	fmt "fmt"
	"regexp"
	"strings"

	"github.com/rs/zerolog/log"
)

// Printable Keys and Values are used by the cli/components package.
var assetPrintableKeys = []string{"name", "platform-id"}

func (a *Asset) PrintableKeys() []string {
	return assetPrintableKeys
}
func (a *Asset) PrintableValue(index int) string {
	switch assetPrintableKeys[index] {
	case "name":
		return a.Display()
	case "platform-id":
		return strings.Join(a.PlatformIds, " ")
	default:
		return a.String()
	}
}

// Display implements SelectableItem from the cli/components package.
func (a *Asset) Display() string {
	return a.HumanName()
}

func (a *Asset) HumanName() string {
	if a == nil {
		return ""
	}

	if a.Platform != nil {
		return fmt.Sprintf(a.Name + " (" + a.Platform.Title + ")")
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

func (a *Asset) AddAnnotations(annotations map[string]string) {
	if len(annotations) == 0 {
		return
	}

	if a.Annotations == nil {
		a.Annotations = map[string]string{}
	}

	for k := range annotations {
		a.Annotations[k] = annotations[k]
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

var AssetCategory_schemevalue = map[string]AssetCategory{
	// FIXME: DEPRECATED, remove in v11.0 vv
	"fleet": AssetCategory_CATEGORY_INVENTORY,
	// ^^
	"inventory": AssetCategory_CATEGORY_INVENTORY,
	"cicd":      AssetCategory_CATEGORY_CICD,
}

// UnmarshalJSON parses either an int or a string representation of
// CredentialType into the struct
func (s *AssetCategory) UnmarshalJSON(data []byte) error {
	// check if we have a number
	var code int32
	err := json.Unmarshal(data, &code)
	if err == nil {
		*s = AssetCategory(code)
	} else {
		var name string
		// we can ignore the error here because we just look it up and otherwise
		// tell users that we can't find the backend value
		_ = json.Unmarshal(data, &name)
		code, ok := AssetCategory_schemevalue[strings.TrimSpace(name)]
		if !ok {
			return errors.New("unknown backend value: " + string(data))
		}
		*s = code
	}
	return nil
}

var mondooLabelRegex = regexp.MustCompile(`^[a-z0-9]*\.?(mondoo.com\/)[a-z0-9\-]*`)

// Merges the mondoo-specific labels from the provided root into the provided asset
func (a *Asset) AddMondooLabels(root *Asset) {
	if a.Labels == nil {
		a.Labels = map[string]string{}
	}
	for k, v := range root.Labels {
		if mondooLabelRegex.MatchString(k) {
			a.Labels[k] = v
		}
	}
}
