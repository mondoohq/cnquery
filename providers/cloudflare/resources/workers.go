// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1
package resources

import (
	"context"
	"errors"
	"sync"

	"github.com/cloudflare/cloudflare-go"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers/cloudflare/connection"
)

func (c *mqlCloudflareZone) workers() (*mqlCloudflareWorkers, error) {
	res, err := CreateResource(c.MqlRuntime, "cloudflare.workers", map[string]*llx.RawData{
		"__id": llx.StringData("cloudflare.workers@" + c.GetAccount().Data.GetId().Data),
	})
	if err != nil {
		return nil, err
	}

	workers := res.(*mqlCloudflareWorkers)
	workers.AccountID = c.GetAccount().Data.GetId().Data

	return workers, nil
}

type mqlCloudflareWorkersInternal struct {
	AccountID string

	workerListLock sync.Mutex
	workerListDone bool
	workerList     []cloudflare.WorkerMetaData
	workerListErr  error
}

// fetchWorkerList caches the per-account workers list so that workers() and
// secrets() share a single ListWorkers API call.
func (c *mqlCloudflareWorkers) fetchWorkerList() ([]cloudflare.WorkerMetaData, error) {
	if c.workerListDone {
		return c.workerList, c.workerListErr
	}
	c.workerListLock.Lock()
	defer c.workerListLock.Unlock()
	if c.workerListDone {
		return c.workerList, c.workerListErr
	}

	conn := c.MqlRuntime.Connection.(*connection.CloudflareConnection)
	resp, _, err := conn.Cf.ListWorkers(context.TODO(), &cloudflare.ResourceContainer{
		Identifier: c.mqlCloudflareWorkersInternal.AccountID,
		Level:      cloudflare.AccountRouteLevel,
	}, cloudflare.ListWorkersParams{})
	if err != nil {
		c.workerListErr = err
	} else {
		c.workerList = resp.WorkerList
	}
	c.workerListDone = true
	return c.workerList, c.workerListErr
}

func (c *mqlCloudflareWorkers) workers() ([]any, error) {
	workerList, err := c.fetchWorkerList()
	if err != nil {
		return nil, err
	}

	var result []any
	for i := range workerList {
		w := workerList[i]

		placementMode := ""
		if w.PlacementMode != nil {
			placementMode = string(*w.PlacementMode)
		}

		res, err := NewResource(c.MqlRuntime, "cloudflare.workers.worker", map[string]*llx.RawData{
			"id":               llx.StringData(w.ID),
			"etag":             llx.StringData(w.ETAG),
			"size":             llx.IntData(w.Size),
			"deploymentId":     llx.StringDataPtr(w.DeploymentId),
			"pipelineHash":     llx.StringDataPtr(w.PipelineHash),
			"placementMode":    llx.StringData(placementMode),
			"lastDeployedFrom": llx.StringDataPtr(w.LastDeployedFrom),
			"logPush":          llx.BoolDataPtr(w.Logpush),
			"createdOn":        llx.TimeData(w.CreatedOn),
			"modifiedOn":       llx.TimeData(w.ModifiedOn),
		})
		if err != nil {
			return nil, err
		}
		result = append(result, res)
	}

	return result, nil
}

func (c *mqlCloudflareWorkers) pages() ([]any, error) {
	return nil, nil
}

func (c *mqlCloudflareWorkersSecret) id() (string, error) {
	return c.GetScriptName().Data + "/" + c.GetName().Data, nil
}

func (c *mqlCloudflarePagesEnvVar) id() (string, error) {
	return c.GetProjectName().Data + "/" + c.GetEnvironment().Data + "/" + c.GetName().Data, nil
}

// secrets enumerates secret bindings across every worker script in the
// account. We only surface name + type — Cloudflare's API never returns the
// secret value, and we explicitly avoid fields that could expose plaintext.
func (c *mqlCloudflareWorkers) secrets() ([]any, error) {
	conn := c.MqlRuntime.Connection.(*connection.CloudflareConnection)

	rc := &cloudflare.ResourceContainer{
		Identifier: c.mqlCloudflareWorkersInternal.AccountID,
		Level:      cloudflare.AccountRouteLevel,
	}

	workerList, err := c.fetchWorkerList()
	if err != nil {
		return nil, err
	}

	var result []any
	for i := range workerList {
		w := workerList[i]
		secrets, err := conn.Cf.ListWorkersSecrets(context.TODO(), rc, cloudflare.ListWorkersSecretsParams{
			ScriptName: w.ID,
		})
		if err != nil {
			// Skip scripts where we lack permission rather than failing the
			// whole enumeration.
			var notFound *cloudflare.NotFoundError
			var authN *cloudflare.AuthenticationError
			var authZ *cloudflare.AuthorizationError
			if errors.As(err, &notFound) || errors.As(err, &authN) || errors.As(err, &authZ) {
				continue
			}
			return nil, err
		}

		for j := range secrets.Result {
			s := secrets.Result[j]
			res, err := CreateResource(c.MqlRuntime, "cloudflare.workers.secret", map[string]*llx.RawData{
				"__id":       llx.StringData("cloudflare.workers.secret@" + w.ID + "/" + s.Name),
				"scriptName": llx.StringData(w.ID),
				"name":       llx.StringData(s.Name),
				"secretType": llx.StringData(s.Type),
			})
			if err != nil {
				return nil, err
			}
			result = append(result, res)
		}
	}

	return result, nil
}

// pageEnvVars enumerates environment variable bindings across every Pages
// project. We expose only `{name, type, environment}` and never `value` so
// secret bindings cannot leak even if the API were to return one.
func (c *mqlCloudflareWorkers) pageEnvVars() ([]any, error) {
	conn := c.MqlRuntime.Connection.(*connection.CloudflareConnection)

	rc := &cloudflare.ResourceContainer{
		Identifier: c.mqlCloudflareWorkersInternal.AccountID,
	}

	var (
		result  []any
		page    = 1
		envvars = func(env, projectName string, m cloudflare.EnvironmentVariableMap) error {
			for name, v := range m {
				varType := ""
				if v != nil {
					varType = string(v.Type)
				}
				res, err := CreateResource(c.MqlRuntime, "cloudflare.pages.envVar", map[string]*llx.RawData{
					"__id":        llx.StringData("cloudflare.pages.envVar@" + projectName + "/" + env + "/" + name),
					"projectName": llx.StringData(projectName),
					"environment": llx.StringData(env),
					"name":        llx.StringData(name),
					"type":        llx.StringData(varType),
				})
				if err != nil {
					return err
				}
				result = append(result, res)
			}
			return nil
		}
	)

	for {
		params := cloudflare.ListPagesProjectsParams{
			PaginationOptions: cloudflare.PaginationOptions{Page: page, PerPage: 50},
		}
		projects, info, err := conn.Cf.ListPagesProjects(context.TODO(), rc, params)
		if err != nil {
			var notFound *cloudflare.NotFoundError
			var authN *cloudflare.AuthenticationError
			var authZ *cloudflare.AuthorizationError
			if errors.As(err, &notFound) || errors.As(err, &authN) || errors.As(err, &authZ) {
				return nil, nil
			}
			return nil, err
		}

		for j := range projects {
			p := projects[j]
			if err := envvars("preview", p.Name, p.DeploymentConfigs.Preview.EnvVars); err != nil {
				return nil, err
			}
			if err := envvars("production", p.Name, p.DeploymentConfigs.Production.EnvVars); err != nil {
				return nil, err
			}
		}

		if !info.HasMorePages() {
			break
		}
		page++
	}

	return result, nil
}
