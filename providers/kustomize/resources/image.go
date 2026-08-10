// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"fmt"
	"strconv"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/kustomize/connection"
	kustomizeTypes "sigs.k8s.io/kustomize/api/types"
)

// initKustomizeImage resolves the selector the schema documents,
// `kustomize.image(name: "nginx")`, against the loaded kustomizations.
//
// Without it the runtime built the resource straight from the `name` arg: every
// other field was left UNSET (surfacing client-side as "primitive with no type
// information"), and because no `__id` was ever assigned, every bare lookup in
// a session shared the cache key `kustomize.image\x00` — so the second lookup
// returned the first one's data.
//
// A miss returns an error rather than falling through to `args, nil, nil`,
// which would rebuild the same husk. The first matching entry wins; an
// `images:` list may legally repeat a name (one entry overriding the tag,
// another the digest), and the full set stays available through
// `kustomize.kustomization.images`.
func initKustomizeImage(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) == 0 {
		return args, nil, nil
	}
	nameArg, ok := args["name"]
	if !ok || nameArg == nil {
		return args, nil, nil
	}
	name, _ := nameArg.Value.(string)
	if name == "" {
		return args, nil, nil
	}

	conn, ok := runtime.Connection.(*connection.KustomizeConnection)
	if !ok {
		return args, nil, nil
	}
	for _, entry := range conn.Kustomizations() {
		if entry == nil || entry.Kustomization == nil {
			continue
		}
		for i := range entry.Kustomization.Images {
			if entry.Kustomization.Images[i].Name != name {
				continue
			}
			mqlImg, err := newMqlKustomizeImage(runtime, entry.Path, i, entry.Kustomization.Images[i])
			if err != nil {
				return nil, nil, err
			}
			return args, mqlImg, nil
		}
	}
	return nil, nil, fmt.Errorf("kustomize: no image override named %q", name)
}

func newMqlKustomizeImage(runtime *plugin.Runtime, kustPath string, idx int, img kustomizeTypes.Image) (*mqlKustomizeImage, error) {
	// Include the list index in the __id: an images: list can legally contain
	// two entries with the same name (e.g. one overriding the tag, one the
	// digest); without the index the second collides with the first in the
	// resource cache and is lost.
	id := "kustomize.image:" + kustPath + ":" + strconv.Itoa(idx) + ":" + img.Name

	res, err := CreateResource(runtime, "kustomize.image", map[string]*llx.RawData{
		"__id":    llx.StringData(id),
		"name":    llx.StringData(img.Name),
		"newName": llx.StringData(img.NewName),
		"newTag":  llx.StringData(img.NewTag),
		"digest":  llx.StringData(img.Digest),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlKustomizeImage), nil
}

var _ plugin.Resource = (*mqlKustomizeImage)(nil)
