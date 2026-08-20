// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/digitalocean/godo"
	"go.mondoo.com/mql/llx"
	"go.mondoo.com/mql/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/providers/digitalocean/connection"
)

func (r *mqlDigitalocean) microDroplets() ([]interface{}, error) {
	conn := r.MqlRuntime.Connection.(*connection.DigitaloceanConnection)
	client := conn.Client()

	instances, err := paginate(context.Background(), client.MicroDroplets.List)
	if err != nil {
		return nil, err
	}

	all := make([]interface{}, 0, len(instances))
	for i := range instances {
		args, err := microDropletArgs(&instances[i])
		if err != nil {
			return nil, err
		}
		res, err := CreateResource(r.MqlRuntime, "digitalocean.microDroplet", args)
		if err != nil {
			return nil, err
		}
		all = append(all, res)
	}
	return all, nil
}

// microDropletArgs maps a MicroDroplet to its MQL fields.
//
// The optional blocks are decoded to the reading that holds when they are
// absent rather than to a null: a MicroDroplet with no auto-pause block does
// not pause itself, and saying so as false keeps an assertion over these
// fields from passing on an instance nothing was read from.
func microDropletArgs(md *godo.MicroDroplet) (map[string]*llx.RawData, error) {
	id, err := resourceID("digitalocean.microDroplet", md.ID)
	if err != nil {
		return nil, err
	}

	autoPauseEnabled := false
	autoPauseIdleTimeout := ""
	if md.AutoPause != nil {
		autoPauseEnabled = md.AutoPause.Enabled != nil && *md.AutoPause.Enabled
		autoPauseIdleTimeout = md.AutoPause.IdleTimeout
	}

	return map[string]*llx.RawData{
		"__id":                 llx.StringData(id),
		"id":                   llx.StringData(md.ID),
		"name":                 llx.StringData(md.Name),
		"region":               llx.StringData(md.Region),
		"state":                llx.StringData(string(md.State)),
		"size":                 llx.StringData(md.Size),
		"networking":           llx.StringData(string(md.Networking)),
		"image":                llx.StringData(md.Image),
		"endpoint":             llx.StringData(md.Endpoint),
		"autoPauseEnabled":     llx.BoolData(autoPauseEnabled),
		"autoPauseIdleTimeout": llx.StringData(autoPauseIdleTimeout),
		"autoResumeEnabled":    llx.BoolData(md.AutoResume != nil && *md.AutoResume),
		"createdAt":            llx.TimeDataPtr(parseDoTime(md.Created)),
	}, nil
}

func initDigitaloceanMicroDroplet(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 1 {
		return args, nil, nil
	}
	id := stringArg(args, "id")
	if id == "" {
		return nil, nil, errors.New("digitalocean.microDroplet requires an id")
	}
	conn := runtime.Connection.(*connection.DigitaloceanConnection)
	md, _, err := conn.Client().MicroDroplets.Get(context.Background(), id)
	if err != nil {
		return nil, nil, err
	}
	// Returning no resource and no error here would have the runtime build a
	// blank MicroDroplet from the id alone, leaving every other field unset.
	if md == nil {
		return nil, nil, fmt.Errorf("digitalocean.microDroplet with id %q not found", id)
	}
	mdArgs, err := microDropletArgs(md)
	if err != nil {
		return nil, nil, err
	}
	return mdArgs, nil, nil
}

// microDropletIsPublic reports whether an instance answers on an address
// reachable from the internet.
//
// Only a VPC placement makes an instance private. Every other value,
// including a mode the API does not name, counts as reachable, so an
// instance whose placement cannot be established is reported as exposed
// rather than quietly passing an exposure check.
func microDropletIsPublic(networking godo.MicroDropletNetworking) bool {
	return !strings.EqualFold(string(networking), string(godo.MicroDropletNetworkingVPC))
}

func (r *mqlDigitaloceanMicroDroplet) isPublic() (bool, error) {
	return microDropletIsPublic(godo.MicroDropletNetworking(r.Networking.Data)), nil
}
