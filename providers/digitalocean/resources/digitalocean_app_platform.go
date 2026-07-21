// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"time"

	"github.com/digitalocean/godo"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/digitalocean/connection"
)

// timePtr returns a pointer to t, or nil when t is the zero value, so an
// omitted API timestamp surfaces as MQL null rather than year 0001.
func timePtr(t time.Time) *time.Time {
	if t.IsZero() {
		return nil
	}
	return &t
}

// componentNames collects the Name field from a deployment's component
// slice into an []interface{} suitable for an llx string array.
func componentNames[T any](items []T, name func(T) string) []interface{} {
	out := make([]interface{}, 0, len(items))
	for _, item := range items {
		out = append(out, name(item))
	}
	return out
}

// newAppDeployment builds a digitalocean.app.deployment resource from a
// godo deployment belonging to appID.
func newAppDeployment(runtime *plugin.Runtime, appID string, d *godo.Deployment) (*mqlDigitaloceanAppDeployment, error) {
	var successSteps, errorSteps, totalSteps int64
	if d.Progress != nil {
		successSteps = int64(d.Progress.SuccessSteps)
		errorSteps = int64(d.Progress.ErrorSteps)
		totalSteps = int64(d.Progress.TotalSteps)
	}

	res, err := CreateResource(runtime, "digitalocean.app.deployment", map[string]*llx.RawData{
		"__id":                 llx.StringData("digitalocean.app.deployment/" + appID + "/" + d.ID),
		"id":                   llx.StringData(d.ID),
		"appId":                llx.StringData(appID),
		"cause":                llx.StringData(d.Cause),
		"phase":                llx.StringData(string(d.Phase)),
		"tierSlug":             llx.StringData(d.TierSlug),
		"previousDeploymentId": llx.StringData(d.PreviousDeploymentID),
		"loadBalancerId":       llx.StringData(d.LoadBalancerID),
		"services":             llx.ArrayData(componentNames(d.Services, func(s *godo.DeploymentService) string { return s.Name }), "\x02"),
		"workers":              llx.ArrayData(componentNames(d.Workers, func(w *godo.DeploymentWorker) string { return w.Name }), "\x02"),
		"jobs":                 llx.ArrayData(componentNames(d.Jobs, func(j *godo.DeploymentJob) string { return j.Name }), "\x02"),
		"staticSites":          llx.ArrayData(componentNames(d.StaticSites, func(s *godo.DeploymentStaticSite) string { return s.Name }), "\x02"),
		"functions":            llx.ArrayData(componentNames(d.Functions, func(f *godo.DeploymentFunctions) string { return f.Name }), "\x02"),
		"progressSuccessSteps": llx.IntData(successSteps),
		"progressErrorSteps":   llx.IntData(errorSteps),
		"progressTotalSteps":   llx.IntData(totalSteps),
		"createdAt":            llx.TimeDataPtr(timePtr(d.CreatedAt)),
		"updatedAt":            llx.TimeDataPtr(timePtr(d.UpdatedAt)),
		"phaseLastUpdatedAt":   llx.TimeDataPtr(timePtr(d.PhaseLastUpdatedAt)),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlDigitaloceanAppDeployment), nil
}

func (r *mqlDigitaloceanAppDeployment) loadBalancer() (*mqlDigitaloceanLoadBalancer, error) {
	if r.LoadBalancerId.Data == "" {
		r.LoadBalancer.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	parent, err := parentDigitalocean(r.MqlRuntime)
	if err != nil {
		return nil, err
	}
	lbs, err := parent.loadBalancerByUIDs([]any{r.LoadBalancerId.Data})
	if err != nil {
		return nil, err
	}
	if len(lbs) == 0 {
		r.LoadBalancer.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	return lbs[0].(*mqlDigitaloceanLoadBalancer), nil
}

func (r *mqlDigitaloceanAppDeployment) previousDeployment() (*mqlDigitaloceanAppDeployment, error) {
	if r.PreviousDeploymentId.Data == "" {
		r.PreviousDeployment.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	conn := r.MqlRuntime.Connection.(*connection.DigitaloceanConnection)
	client := conn.Client()

	deployment, _, err := client.Apps.GetDeployment(context.Background(), r.AppId.Data, r.PreviousDeploymentId.Data)
	if err != nil {
		if isDoNotFound(err) {
			r.PreviousDeployment.State = plugin.StateIsSet | plugin.StateIsNull
			return nil, nil
		}
		return nil, err
	}
	return newAppDeployment(r.MqlRuntime, r.AppId.Data, deployment)
}

func (r *mqlDigitaloceanApp) deployments() ([]interface{}, error) {
	conn := r.MqlRuntime.Connection.(*connection.DigitaloceanConnection)
	client := conn.Client()

	var all []interface{}
	opt := &godo.ListOptions{PerPage: 200}
	for {
		deployments, resp, err := client.Apps.ListDeployments(context.Background(), r.Id.Data, opt)
		if err != nil {
			return nil, err
		}
		for _, d := range deployments {
			if d == nil {
				continue
			}
			res, err := newAppDeployment(r.MqlRuntime, r.Id.Data, d)
			if err != nil {
				return nil, err
			}
			all = append(all, res)
		}
		if resp == nil || resp.Links == nil || resp.Links.IsLastPage() {
			break
		}
		page, err := resp.Links.CurrentPage()
		if err != nil {
			return nil, err
		}
		opt.Page = page + 1
	}
	return all, nil
}

func (r *mqlDigitaloceanApp) activeDeployment() (*mqlDigitaloceanAppDeployment, error) {
	if r.ActiveDeploymentId.Data == "" {
		r.ActiveDeployment.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	conn := r.MqlRuntime.Connection.(*connection.DigitaloceanConnection)
	client := conn.Client()

	deployment, _, err := client.Apps.GetDeployment(context.Background(), r.Id.Data, r.ActiveDeploymentId.Data)
	if err != nil {
		if isDoNotFound(err) {
			r.ActiveDeployment.State = plugin.StateIsSet | plugin.StateIsNull
			return nil, nil
		}
		return nil, err
	}
	return newAppDeployment(r.MqlRuntime, r.Id.Data, deployment)
}

func (r *mqlDigitaloceanApp) alerts() ([]interface{}, error) {
	conn := r.MqlRuntime.Connection.(*connection.DigitaloceanConnection)
	client := conn.Client()

	alerts, _, err := client.Apps.ListAlerts(context.Background(), r.Id.Data)
	if err != nil {
		return nil, err
	}

	var all []interface{}
	for _, a := range alerts {
		if a == nil {
			continue
		}
		var (
			rule, operator, window string
			value                  float64
			disabled               bool
		)
		if a.Spec != nil {
			rule = string(a.Spec.Rule)
			operator = string(a.Spec.Operator)
			window = string(a.Spec.Window)
			value = float64(a.Spec.Value)
			disabled = a.Spec.Disabled
		}
		emails := make([]interface{}, len(a.Emails))
		for i, e := range a.Emails {
			emails[i] = e
		}
		res, err := CreateResource(r.MqlRuntime, "digitalocean.app.alert", map[string]*llx.RawData{
			"__id":              llx.StringData("digitalocean.app.alert/" + r.Id.Data + "/" + a.ID),
			"id":                llx.StringData(a.ID),
			"appId":             llx.StringData(r.Id.Data),
			"componentName":     llx.StringData(a.ComponentName),
			"rule":              llx.StringData(rule),
			"disabled":          llx.BoolData(disabled),
			"operator":          llx.StringData(operator),
			"value":             llx.FloatData(value),
			"window":            llx.StringData(window),
			"phase":             llx.StringData(string(a.Phase)),
			"emails":            llx.ArrayData(emails, "\x02"),
			"slackWebhookCount": llx.IntData(int64(len(a.SlackWebhooks))),
		})
		if err != nil {
			return nil, err
		}
		all = append(all, res)
	}
	return all, nil
}

func (r *mqlDigitaloceanApp) instances() ([]interface{}, error) {
	conn := r.MqlRuntime.Connection.(*connection.DigitaloceanConnection)
	client := conn.Client()

	instances, _, err := client.Apps.GetAppInstances(context.Background(), r.Id.Data, nil)
	if err != nil {
		// An app with no active deployment has no running instances.
		if isDoNotFound(err) {
			return []interface{}{}, nil
		}
		return nil, err
	}

	all := make([]interface{}, 0, len(instances))
	for _, inst := range instances {
		if inst == nil {
			continue
		}
		all = append(all, map[string]interface{}{
			"componentName": inst.ComponentName,
			"componentType": string(inst.ComponentType),
			"instanceName":  inst.InstanceName,
			"instanceAlias": inst.InstanceAlias,
		})
	}
	return all, nil
}
