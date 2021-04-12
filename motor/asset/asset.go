package asset

import (
	fmt "fmt"
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
