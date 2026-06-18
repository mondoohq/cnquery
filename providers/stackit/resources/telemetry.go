// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"time"

	tlink "github.com/stackitcloud/stackit-sdk-go/services/telemetrylink/v1betaapi"
	trouter "github.com/stackitcloud/stackit-sdk-go/services/telemetryrouter/v1betaapi"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
)

// telemetryTime normalizes the telemetry SDK's optional timestamps (returned
// as *time.Time, bool) into a *time.Time, mapping absent/zero values to nil.
func telemetryTime(p *time.Time, ok bool) *time.Time {
	if !ok || p == nil || p.IsZero() {
		return nil
	}
	return p
}

// ------------------------- Telemetry router -------------------------

func telemetryRouterArgs(region string, r *trouter.TelemetryRouterResponse) map[string]*llx.RawData {
	return map[string]*llx.RawData{
		"id":          llx.StringData(r.GetId()),
		"displayName": llx.StringData(r.GetDisplayName()),
		"description": llx.StringData(r.GetDescription()),
		"status":      llx.StringData(string(r.GetStatus())),
		"uri":         llx.StringData(r.GetUri()),
		"region":      llx.StringData(region),
		"filter":      llx.DictData(toDict(r.GetFilter())),
		"createdAt":   llx.TimeDataPtr(telemetryTime(r.GetCreationTimeOk())),
	}
}

func (r *mqlStackitTelemetry) routers() ([]any, error) {
	c := conn(r.MqlRuntime)
	client, err := c.TelemetryRouter()
	if err != nil {
		return nil, err
	}
	out := []any{}
	pageToken := ""
	for {
		req := client.DefaultAPI.ListTelemetryRouters(bgctx(), c.ProjectID(), c.Region())
		if pageToken != "" {
			req = req.PageToken(pageToken)
		}
		resp, err := req.Execute()
		if err != nil {
			if isAccessDenied(err) {
				return []any{}, nil
			}
			return nil, err
		}
		routers := resp.GetTelemetryRouters()
		for i := range routers {
			res, err := CreateResource(r.MqlRuntime, "stackit.telemetry.router", telemetryRouterArgs(c.Region(), &routers[i]))
			if err != nil {
				return nil, err
			}
			out = append(out, res)
		}
		pageToken = resp.GetNextPageToken()
		if pageToken == "" {
			break
		}
	}
	return out, nil
}

func (r *mqlStackitTelemetryRouter) id() (string, error) {
	return "stackit.telemetry.router/" + r.Id.Data, nil
}

func (r *mqlStackitTelemetryRouter) destinations() ([]any, error) {
	c := conn(r.MqlRuntime)
	client, err := c.TelemetryRouter()
	if err != nil {
		return nil, err
	}
	out := []any{}
	pageToken := ""
	for {
		req := client.DefaultAPI.ListDestinations(bgctx(), c.ProjectID(), r.Region.Data, r.Id.Data)
		if pageToken != "" {
			req = req.PageToken(pageToken)
		}
		resp, err := req.Execute()
		if err != nil {
			if isAccessDenied(err) {
				return []any{}, nil
			}
			return nil, err
		}
		dests := resp.GetDestinations()
		for i := range dests {
			d := dests[i]
			res, err := CreateResource(r.MqlRuntime, "stackit.telemetry.router.destination", map[string]*llx.RawData{
				"id":             llx.StringData(d.GetId()),
				"displayName":    llx.StringData(d.GetDisplayName()),
				"description":    llx.StringData(d.GetDescription()),
				"status":         llx.StringData(string(d.GetStatus())),
				"credentialType": llx.StringData(string(d.GetCredentialType())),
				"config":         llx.DictData(toDict(d.GetConfig())),
				"createdAt":      llx.TimeDataPtr(telemetryTime(d.GetCreationTimeOk())),
			})
			if err != nil {
				return nil, err
			}
			out = append(out, res)
		}
		pageToken = resp.GetNextPageToken()
		if pageToken == "" {
			break
		}
	}
	return out, nil
}

func (r *mqlStackitTelemetryRouterDestination) id() (string, error) {
	return "stackit.telemetry.router.destination/" + r.Id.Data, nil
}

func (r *mqlStackitTelemetryRouter) accessTokens() ([]any, error) {
	c := conn(r.MqlRuntime)
	client, err := c.TelemetryRouter()
	if err != nil {
		return nil, err
	}
	out := []any{}
	pageToken := ""
	for {
		req := client.DefaultAPI.ListAccessTokens(bgctx(), c.ProjectID(), r.Region.Data, r.Id.Data)
		if pageToken != "" {
			req = req.PageToken(pageToken)
		}
		resp, err := req.Execute()
		if err != nil {
			if isAccessDenied(err) {
				return []any{}, nil
			}
			return nil, err
		}
		tokens := resp.GetAccessTokens()
		for i := range tokens {
			t := tokens[i]
			res, err := CreateResource(r.MqlRuntime, "stackit.telemetry.router.accessToken", map[string]*llx.RawData{
				"id":          llx.StringData(t.GetId()),
				"displayName": llx.StringData(t.GetDisplayName()),
				"description": llx.StringData(t.GetDescription()),
				"status":      llx.StringData(string(t.GetStatus())),
				"creatorId":   llx.StringData(t.GetCreatorId()),
				"expiresAt":   llx.TimeDataPtr(telemetryTime(t.GetExpirationTimeOk())),
			})
			if err != nil {
				return nil, err
			}
			out = append(out, res)
		}
		pageToken = resp.GetNextPageToken()
		if pageToken == "" {
			break
		}
	}
	return out, nil
}

func (r *mqlStackitTelemetryRouterAccessToken) id() (string, error) {
	return "stackit.telemetry.router.accessToken/" + r.Id.Data, nil
}

func initStackitTelemetryRouter(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	id, ok := idArg(args, "id")
	if !ok {
		return args, nil, nil
	}
	c := conn(runtime)
	client, err := c.TelemetryRouter()
	if err != nil {
		return nil, nil, err
	}
	resp, err := client.DefaultAPI.GetTelemetryRouter(bgctx(), c.ProjectID(), c.Region(), id).Execute()
	if err != nil {
		return nil, nil, err
	}
	res, err := CreateResource(runtime, "stackit.telemetry.router", telemetryRouterArgs(c.Region(), resp))
	if err != nil {
		return nil, nil, err
	}
	return nil, res, nil
}

// ------------------------- Telemetry link -------------------------

type mqlStackitTelemetryLinkInternal struct {
	cacheRouterID string
}

func telemetryLinkArgs(l *tlink.TelemetryLinkResponse) map[string]*llx.RawData {
	return map[string]*llx.RawData{
		"id":          llx.StringData(l.GetId()),
		"displayName": llx.StringData(l.GetDisplayName()),
		"description": llx.StringData(l.GetDescription()),
		"enabled":     llx.BoolData(l.GetEnabled()),
		"status":      llx.StringData(string(l.GetStatus())),
		"region":      llx.StringData(l.GetRegionId()),
		"createdAt":   llx.TimeDataPtr(telemetryTime(l.GetCreateTimeOk())),
	}
}

func (r *mqlStackitTelemetry) link() (*mqlStackitTelemetryLink, error) {
	c := conn(r.MqlRuntime)
	client, err := c.TelemetryLink()
	if err != nil {
		return nil, err
	}
	resp, err := client.DefaultAPI.GetProjectTelemetryLink(bgctx(), c.ProjectID(), c.Region()).Execute()
	if err != nil {
		if isAccessDenied(err) || isNotFound(err) {
			r.Link.State = plugin.StateIsSet | plugin.StateIsNull
			return nil, nil
		}
		return nil, err
	}
	res, err := CreateResource(r.MqlRuntime, "stackit.telemetry.link", telemetryLinkArgs(resp))
	if err != nil {
		return nil, err
	}
	link := res.(*mqlStackitTelemetryLink)
	link.cacheRouterID = resp.GetTelemetryRouterId()
	return link, nil
}

func (r *mqlStackitTelemetryLink) id() (string, error) {
	return "stackit.telemetry.link/" + r.Id.Data, nil
}

func (r *mqlStackitTelemetryLink) router() (*mqlStackitTelemetryRouter, error) {
	if r.cacheRouterID == "" {
		r.Router.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	res, err := NewResource(r.MqlRuntime, "stackit.telemetry.router", map[string]*llx.RawData{
		"id": llx.StringData(r.cacheRouterID),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlStackitTelemetryRouter), nil
}
