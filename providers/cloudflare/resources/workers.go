// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1
package resources

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers/cloudflare/connection"
)

func (c *mqlCloudflareZone) workers() (*mqlCloudflareWorkers, error) {
	accountID, err := c.zoneAccountID()
	if err != nil {
		return nil, err
	}
	res, err := CreateResource(c.MqlRuntime, "cloudflare.workers", map[string]*llx.RawData{
		"__id": llx.StringData("cloudflare.workers@" + accountID),
	})
	if err != nil {
		return nil, err
	}

	workers := res.(*mqlCloudflareWorkers)
	workers.AccountID = accountID

	return workers, nil
}

// workerScript mirrors the account workers-scripts list entry. We decode it via
// the client's generic Get to preserve fields (size, deployment_id,
// pipeline_hash) that the typed cloudflare-go v6 script struct no longer
// exposes.
type workerScript struct {
	ID               string    `json:"id"`
	ETag             string    `json:"etag"`
	Size             int64     `json:"size"`
	DeploymentID     *string   `json:"deployment_id"`
	PipelineHash     *string   `json:"pipeline_hash"`
	PlacementMode    string    `json:"placement_mode"`
	LastDeployedFrom *string   `json:"last_deployed_from"`
	Logpush          *bool     `json:"logpush"`
	CreatedOn        time.Time `json:"created_on"`
	ModifiedOn       time.Time `json:"modified_on"`
}

type mqlCloudflareWorkersInternal struct {
	AccountID string

	workerListLock sync.Mutex
	workerListDone bool
	workerList     []workerScript
	workerListErr  error
}

// fetchWorkerList caches the per-account workers list so that workers() and
// secrets() share a single list API call.
func (c *mqlCloudflareWorkers) fetchWorkerList() ([]workerScript, error) {
	if c.workerListDone {
		return c.workerList, c.workerListErr
	}
	c.workerListLock.Lock()
	defer c.workerListLock.Unlock()
	if c.workerListDone {
		return c.workerList, c.workerListErr
	}

	conn := c.MqlRuntime.Connection.(*connection.CloudflareConnection)
	var env struct {
		Result []workerScript `json:"result"`
	}
	uri := fmt.Sprintf("accounts/%s/workers/scripts", c.mqlCloudflareWorkersInternal.AccountID)
	if err := conn.Cf.Get(context.TODO(), uri, nil, &env); err != nil {
		c.workerListErr = err
	} else {
		c.workerList = env.Result
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

		res, err := NewResource(c.MqlRuntime, "cloudflare.workers.worker", map[string]*llx.RawData{
			"id":               llx.StringData(w.ID),
			"etag":             llx.StringData(w.ETag),
			"size":             llx.IntData(w.Size),
			"deploymentId":     llx.StringDataPtr(w.DeploymentID),
			"pipelineHash":     llx.StringDataPtr(w.PipelineHash),
			"placementMode":    llx.StringData(w.PlacementMode),
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

	workerList, err := c.fetchWorkerList()
	if err != nil {
		return nil, err
	}

	var result []any
	for i := range workerList {
		w := workerList[i]

		var env struct {
			Result []struct {
				Name string `json:"name"`
				Type string `json:"type"`
			} `json:"result"`
		}
		uri := fmt.Sprintf("accounts/%s/workers/scripts/%s/secrets", c.mqlCloudflareWorkersInternal.AccountID, w.ID)
		if err := conn.Cf.Get(context.TODO(), uri, nil, &env); err != nil {
			// Skip scripts where we lack permission rather than failing the
			// whole enumeration.
			if isUnavailable(err) {
				continue
			}
			return nil, err
		}

		for j := range env.Result {
			s := env.Result[j]
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

type pagesDeployConfig struct {
	EnvVars map[string]*struct {
		Type string `json:"type"`
	} `json:"env_vars"`
}

type pagesProject struct {
	Name              string `json:"name"`
	DeploymentConfigs struct {
		Preview    pagesDeployConfig `json:"preview"`
		Production pagesDeployConfig `json:"production"`
	} `json:"deployment_configs"`
}

// pageEnvVars enumerates environment variable bindings across every Pages
// project. We expose only `{name, type, environment}` and never `value` so
// secret bindings cannot leak even if the API were to return one.
func (c *mqlCloudflareWorkers) pageEnvVars() ([]any, error) {
	conn := c.MqlRuntime.Connection.(*connection.CloudflareConnection)

	projects, err := cfGetPaged[pagesProject](conn, fmt.Sprintf("accounts/%s/pages/projects", c.mqlCloudflareWorkersInternal.AccountID))
	if err != nil {
		if isUnavailable(err) {
			return nil, nil
		}
		return nil, err
	}

	var result []any
	emit := func(env, projectName string, cfg pagesDeployConfig) error {
		for name, v := range cfg.EnvVars {
			varType := ""
			if v != nil {
				varType = v.Type
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

	for j := range projects {
		p := projects[j]
		if err := emit("preview", p.Name, p.DeploymentConfigs.Preview); err != nil {
			return nil, err
		}
		if err := emit("production", p.Name, p.DeploymentConfigs.Production); err != nil {
			return nil, err
		}
	}

	return result, nil
}
