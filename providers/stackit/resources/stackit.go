// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/stackit/connection"
)

func (r *mqlStackit) id() (string, error) {
	return "stackit/" + conn(r.MqlRuntime).ProjectID(), nil
}

func (r *mqlStackit) region() (string, error) {
	return conn(r.MqlRuntime).Region(), nil
}

func conn(runtime *plugin.Runtime) *connection.StackitConnection {
	return runtime.Connection.(*connection.StackitConnection)
}

// project fetches the project metadata from the resource-manager API.
func (r *mqlStackit) project() (*mqlStackitProject, error) {
	c := conn(r.MqlRuntime)
	client, err := c.ResourceManager()
	if err != nil {
		return nil, err
	}
	resp, err := client.GetProjectExecute(bgctx(), c.ProjectID())
	if err != nil {
		if isAccessDenied(err) {
			return markNull[mqlStackitProject](&r.Project)
		}
		return nil, err
	}

	lifecycle, _ := resp.GetLifecycleStateOk()
	labels, _ := resp.GetLabelsOk()
	createdAt, ok := resp.GetCreationTimeOk()

	parentID := ""
	if p, hasParent := resp.GetParentOk(); hasParent {
		parentID = p.GetContainerId()
		if parentID == "" {
			parentID = p.GetId()
		}
	}

	args := map[string]*llx.RawData{
		"id":             llx.StringData(resp.GetProjectId()),
		"name":           llx.StringData(resp.GetName()),
		"parent":         llx.StringData(parentID),
		"lifecycleState": llx.StringData(string(lifecycle)),
		"creationTime":   llx.TimeDataPtr(timeOrNil(createdAt, ok)),
		"labels":         labelData(labels),
	}
	res, err := CreateResource(r.MqlRuntime, "stackit.project", args)
	if err != nil {
		return nil, err
	}
	return res.(*mqlStackitProject), nil
}

func (r *mqlStackitProject) id() (string, error) {
	return "stackit.project/" + r.Id.Data, nil
}

// Each namespace resource has a stable id.

func (r *mqlStackitSke) id() (string, error) {
	return "stackit.ske/" + conn(r.MqlRuntime).ProjectID(), nil
}

func (r *mqlStackitObjectStorage) id() (string, error) {
	return "stackit.objectStorage/" + conn(r.MqlRuntime).ProjectID(), nil
}

func (r *mqlStackitDns) id() (string, error) {
	return "stackit.dns/" + conn(r.MqlRuntime).ProjectID(), nil
}

func (r *mqlStackitPostgresFlex) id() (string, error) {
	return "stackit.postgresFlex/" + conn(r.MqlRuntime).ProjectID(), nil
}

func (r *mqlStackitMongoDbFlex) id() (string, error) {
	return "stackit.mongoDbFlex/" + conn(r.MqlRuntime).ProjectID(), nil
}

func (r *mqlStackitOpenSearch) id() (string, error) {
	return "stackit.openSearch/" + conn(r.MqlRuntime).ProjectID(), nil
}

func (r *mqlStackitMariaDb) id() (string, error) {
	return "stackit.mariaDb/" + conn(r.MqlRuntime).ProjectID(), nil
}

func (r *mqlStackitRedis) id() (string, error) {
	return "stackit.redis/" + conn(r.MqlRuntime).ProjectID(), nil
}

func (r *mqlStackitRabbitMq) id() (string, error) {
	return "stackit.rabbitMq/" + conn(r.MqlRuntime).ProjectID(), nil
}

func (r *mqlStackitSecretsManager) id() (string, error) {
	return "stackit.secretsManager/" + conn(r.MqlRuntime).ProjectID(), nil
}

func (r *mqlStackitObservability) id() (string, error) {
	return "stackit.observability/" + conn(r.MqlRuntime).ProjectID(), nil
}

// Namespace getters on `stackit` just return the singleton namespace resource.

func (r *mqlStackit) ske() (*mqlStackitSke, error) {
	res, err := makeNamespace(r.MqlRuntime, "stackit.ske")
	if err != nil {
		return nil, err
	}
	return res.(*mqlStackitSke), nil
}

func (r *mqlStackit) objectStorage() (*mqlStackitObjectStorage, error) {
	res, err := makeNamespace(r.MqlRuntime, "stackit.objectStorage")
	if err != nil {
		return nil, err
	}
	return res.(*mqlStackitObjectStorage), nil
}

func (r *mqlStackit) dns() (*mqlStackitDns, error) {
	res, err := makeNamespace(r.MqlRuntime, "stackit.dns")
	if err != nil {
		return nil, err
	}
	return res.(*mqlStackitDns), nil
}

func (r *mqlStackit) postgresFlex() (*mqlStackitPostgresFlex, error) {
	res, err := makeNamespace(r.MqlRuntime, "stackit.postgresFlex")
	if err != nil {
		return nil, err
	}
	return res.(*mqlStackitPostgresFlex), nil
}

func (r *mqlStackit) mongoDbFlex() (*mqlStackitMongoDbFlex, error) {
	res, err := makeNamespace(r.MqlRuntime, "stackit.mongoDbFlex")
	if err != nil {
		return nil, err
	}
	return res.(*mqlStackitMongoDbFlex), nil
}

func (r *mqlStackit) openSearch() (*mqlStackitOpenSearch, error) {
	res, err := makeNamespace(r.MqlRuntime, "stackit.openSearch")
	if err != nil {
		return nil, err
	}
	return res.(*mqlStackitOpenSearch), nil
}

func (r *mqlStackit) sqlServerFlex() (*mqlStackitSqlServerFlex, error) {
	res, err := makeNamespace(r.MqlRuntime, "stackit.sqlServerFlex")
	if err != nil {
		return nil, err
	}
	return res.(*mqlStackitSqlServerFlex), nil
}

func (r *mqlStackit) logMe() (*mqlStackitLogMe, error) {
	res, err := makeNamespace(r.MqlRuntime, "stackit.logMe")
	if err != nil {
		return nil, err
	}
	return res.(*mqlStackitLogMe), nil
}

func (r *mqlStackit) mariaDb() (*mqlStackitMariaDb, error) {
	res, err := makeNamespace(r.MqlRuntime, "stackit.mariaDb")
	if err != nil {
		return nil, err
	}
	return res.(*mqlStackitMariaDb), nil
}

func (r *mqlStackit) redis() (*mqlStackitRedis, error) {
	res, err := makeNamespace(r.MqlRuntime, "stackit.redis")
	if err != nil {
		return nil, err
	}
	return res.(*mqlStackitRedis), nil
}

func (r *mqlStackit) rabbitMq() (*mqlStackitRabbitMq, error) {
	res, err := makeNamespace(r.MqlRuntime, "stackit.rabbitMq")
	if err != nil {
		return nil, err
	}
	return res.(*mqlStackitRabbitMq), nil
}

func (r *mqlStackit) secretsManager() (*mqlStackitSecretsManager, error) {
	res, err := makeNamespace(r.MqlRuntime, "stackit.secretsManager")
	if err != nil {
		return nil, err
	}
	return res.(*mqlStackitSecretsManager), nil
}

func (r *mqlStackit) observability() (*mqlStackitObservability, error) {
	res, err := makeNamespace(r.MqlRuntime, "stackit.observability")
	if err != nil {
		return nil, err
	}
	return res.(*mqlStackitObservability), nil
}

func (r *mqlStackit) telemetry() (*mqlStackitTelemetry, error) {
	res, err := makeNamespace(r.MqlRuntime, "stackit.telemetry")
	if err != nil {
		return nil, err
	}
	return res.(*mqlStackitTelemetry), nil
}

func (r *mqlStackit) sfs() (*mqlStackitSfs, error) {
	res, err := makeNamespace(r.MqlRuntime, "stackit.sfs")
	if err != nil {
		return nil, err
	}
	return res.(*mqlStackitSfs), nil
}

func (r *mqlStackit) kms() (*mqlStackitKms, error) {
	res, err := makeNamespace(r.MqlRuntime, "stackit.kms")
	if err != nil {
		return nil, err
	}
	return res.(*mqlStackitKms), nil
}

func (r *mqlStackit) iam() (*mqlStackitIam, error) {
	res, err := makeNamespace(r.MqlRuntime, "stackit.iam")
	if err != nil {
		return nil, err
	}
	return res.(*mqlStackitIam), nil
}
