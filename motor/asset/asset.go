package asset

import (
	fmt "fmt"

	platform "go.mondoo.io/mondoo/motor/platform"
)

//go:generate protoc --proto_path=$GOPATH/src:. --proto_path=$GOPATH/pkg/mod/github.com/gogo/protobuf@v1.3.1/gogoproto --falcon_out=. --iam-actions_out=. --gofast_out=Mgoogle/protobuf/any.proto=github.com/gogo/protobuf/types:$GOPATH/src asset.proto

func (a *Asset) HumanName() string {
	if a == nil {
		return ""
	}

	if a.Platform != nil {
		return fmt.Sprintf("%s (%s)", a.Name, a.Platform.Kind.Name())
	}

	return a.Name
}

func (a *Asset) EnsureReferenceID(ids ...string) {
	if a.ReferenceIDs == nil {
		a.ReferenceIDs = ids
		return
	}

	// check if the id is already included
	keys := map[string]bool{}
	for _, k := range a.ReferenceIDs {
		keys[k] = true
	}

	// append entry
	for _, id := range ids {
		_, ok := keys[id]
		if !ok {
			a.ReferenceIDs = append(a.ReferenceIDs, id)
		}
	}
}

func (a *Asset) UpdatePlatform(pf *platform.Platform) {
	if pf == nil {
		return
	}

	if a.Platform == nil {
		a.Platform = &platform.Platform{}
	}

	// merge existing information with the scan
	a.Platform.Name = pf.Name
	a.Platform.Release = pf.Release
	a.Platform.Arch = pf.Arch
}

func (a *Asset) AddReferenceID(ids ...string) {
	if a.ReferenceIDs == nil {
		a.ReferenceIDs = []string{}
	}

	a.ReferenceIDs = append(a.ReferenceIDs, ids...)
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
