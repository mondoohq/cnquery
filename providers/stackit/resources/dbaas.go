// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/stackitcloud/stackit-sdk-go/services/mongodbflex"
	"github.com/stackitcloud/stackit-sdk-go/services/observability"
	postgresflex "github.com/stackitcloud/stackit-sdk-go/services/postgresflex/v2api"
	sqlserverflex "github.com/stackitcloud/stackit-sdk-go/services/sqlserverflex/v2api"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
)

// ------------------------- Postgres Flex -------------------------
//
// The ListInstances endpoint returns id/name/status only. Everything else
// (version, flavor, ACL, replicas, storage, backup schedule, options) lives
// behind GetInstance(projectId, region, instanceId). We model that detail as
// computed methods that share a single cached fetch per instance.

type mqlStackitPostgresFlexInstanceInternal struct {
	fetched atomic.Bool
	detail  *postgresflex.Instance
	lock    sync.Mutex
}

func (r *mqlStackitPostgresFlex) instances() ([]any, error) {
	c := conn(r.MqlRuntime)
	client, err := c.PostgresFlex()
	if err != nil {
		return nil, err
	}
	resp, err := client.DefaultAPI.ListInstances(bgctx(), c.ProjectID(), c.Region()).Execute()
	if err != nil {
		if isAccessDenied(err) {
			return []any{}, nil
		}
		return nil, err
	}
	items, _ := resp.GetItemsOk()
	out := make([]any, 0, len(items))
	for i := range items {
		inst := items[i]
		args := map[string]*llx.RawData{
			"id":     llx.StringData(inst.GetId()),
			"name":   llx.StringData(inst.GetName()),
			"status": llx.StringData(inst.GetStatus()),
			"region": llx.StringData(c.Region()),
		}
		res, err := CreateResource(r.MqlRuntime, "stackit.postgresFlex.instance", args)
		if err != nil {
			return nil, err
		}
		out = append(out, res)
	}
	return out, nil
}

func (r *mqlStackitPostgresFlexInstance) id() (string, error) {
	return "stackit.postgresFlex.instance/" + r.Id.Data, nil
}

// fetchDetail pulls the full instance object via GetInstance and caches it
// for the lifetime of this resource. Double-check locked so concurrent field
// accesses share one API call.
func (r *mqlStackitPostgresFlexInstance) fetchDetail() (*postgresflex.Instance, error) {
	if r.fetched.Load() {
		return r.detail, nil
	}
	r.lock.Lock()
	defer r.lock.Unlock()
	if r.fetched.Load() {
		return r.detail, nil
	}
	c := conn(r.MqlRuntime)
	client, err := c.PostgresFlex()
	if err != nil {
		return nil, err
	}
	resp, err := client.DefaultAPI.GetInstance(bgctx(), c.ProjectID(), c.Region(), r.Id.Data).Execute()
	if err != nil {
		if isAccessDenied(err) {
			r.fetched.Store(true)
			return nil, nil
		}
		return nil, err
	}
	if item, ok := resp.GetItemOk(); ok {
		r.detail = item
	}
	r.fetched.Store(true)
	return r.detail, nil
}

func (r *mqlStackitPostgresFlexInstance) version() (string, error) {
	d, err := r.fetchDetail()
	if err != nil || d == nil {
		return "", err
	}
	return d.GetVersion(), nil
}

func (r *mqlStackitPostgresFlexInstance) flavor() (any, error) {
	d, err := r.fetchDetail()
	if err != nil || d == nil {
		return nil, err
	}
	return toDict(d.GetFlavor()), nil
}

func (r *mqlStackitPostgresFlexInstance) acl() ([]any, error) {
	d, err := r.fetchDetail()
	if err != nil || d == nil {
		return []any{}, err
	}
	if a, ok := d.GetAclOk(); ok {
		return strSlice(a.GetItems()), nil
	}
	return []any{}, nil
}

func (r *mqlStackitPostgresFlexInstance) replicas() (int64, error) {
	d, err := r.fetchDetail()
	if err != nil || d == nil {
		return 0, err
	}
	return int64(d.GetReplicas()), nil
}

func (r *mqlStackitPostgresFlexInstance) storage() (any, error) {
	d, err := r.fetchDetail()
	if err != nil || d == nil {
		return nil, err
	}
	return toDict(d.GetStorage()), nil
}

func (r *mqlStackitPostgresFlexInstance) backupSchedule() (string, error) {
	d, err := r.fetchDetail()
	if err != nil || d == nil {
		return "", err
	}
	return d.GetBackupSchedule(), nil
}

func (r *mqlStackitPostgresFlexInstance) options() (map[string]any, error) {
	d, err := r.fetchDetail()
	if err != nil || d == nil {
		return map[string]any{}, err
	}
	return stringMap(d.GetOptions()), nil
}

// ------------------------- MongoDB Flex -------------------------
//
// Mirrors the Postgres Flex pattern: list returns id/name/status only,
// detail (version/flavor/replicas/storage/backupSchedule/acl/options) is
// lazy-loaded once per instance.

type mqlStackitMongoDbFlexInstanceInternal struct {
	fetched atomic.Bool
	detail  *mongodbflex.Instance
	lock    sync.Mutex
}

func (r *mqlStackitMongoDbFlex) instances() ([]any, error) {
	c := conn(r.MqlRuntime)
	client, err := c.MongoDbFlex()
	if err != nil {
		return nil, err
	}
	// Unlike the other Flex engines, the MongoDB Flex ListInstances endpoint
	// requires a `tag` query parameter. An empty tag returns all instances in
	// the project, so use the request builder rather than the convenience
	// ListInstancesExecute (which cannot set a tag).
	resp, err := client.ListInstances(bgctx(), c.ProjectID(), c.Region()).Tag("").Execute()
	if err != nil {
		if isAccessDenied(err) {
			return []any{}, nil
		}
		return nil, err
	}
	items, _ := resp.GetItemsOk()
	out := make([]any, 0, len(items))
	for i := range items {
		inst := items[i]
		args := map[string]*llx.RawData{
			"id":     llx.StringData(inst.GetId()),
			"name":   llx.StringData(inst.GetName()),
			"status": llx.StringData(string(inst.GetStatus())),
			"region": llx.StringData(c.Region()),
		}
		res, err := CreateResource(r.MqlRuntime, "stackit.mongoDbFlex.instance", args)
		if err != nil {
			return nil, err
		}
		out = append(out, res)
	}
	return out, nil
}

func (r *mqlStackitMongoDbFlexInstance) id() (string, error) {
	return "stackit.mongoDbFlex.instance/" + r.Id.Data, nil
}

func (r *mqlStackitMongoDbFlexInstance) fetchDetail() (*mongodbflex.Instance, error) {
	if r.fetched.Load() {
		return r.detail, nil
	}
	r.lock.Lock()
	defer r.lock.Unlock()
	if r.fetched.Load() {
		return r.detail, nil
	}
	c := conn(r.MqlRuntime)
	client, err := c.MongoDbFlex()
	if err != nil {
		return nil, err
	}
	resp, err := client.GetInstanceExecute(bgctx(), c.ProjectID(), r.Id.Data, c.Region())
	if err != nil {
		if isAccessDenied(err) {
			r.fetched.Store(true)
			return nil, nil
		}
		return nil, err
	}
	if item, ok := resp.GetItemOk(); ok {
		r.detail = &item
	}
	r.fetched.Store(true)
	return r.detail, nil
}

func (r *mqlStackitMongoDbFlexInstance) version() (string, error) {
	d, err := r.fetchDetail()
	if err != nil || d == nil {
		return "", err
	}
	return d.GetVersion(), nil
}

func (r *mqlStackitMongoDbFlexInstance) flavor() (any, error) {
	d, err := r.fetchDetail()
	if err != nil || d == nil {
		return nil, err
	}
	return toDict(d.GetFlavor()), nil
}

func (r *mqlStackitMongoDbFlexInstance) replicas() (int64, error) {
	d, err := r.fetchDetail()
	if err != nil || d == nil {
		return 0, err
	}
	return int64(d.GetReplicas()), nil
}

func (r *mqlStackitMongoDbFlexInstance) storage() (any, error) {
	d, err := r.fetchDetail()
	if err != nil || d == nil {
		return nil, err
	}
	return toDict(d.GetStorage()), nil
}

func (r *mqlStackitMongoDbFlexInstance) backupSchedule() (string, error) {
	d, err := r.fetchDetail()
	if err != nil || d == nil {
		return "", err
	}
	return d.GetBackupSchedule(), nil
}

func (r *mqlStackitMongoDbFlexInstance) acl() ([]any, error) {
	d, err := r.fetchDetail()
	if err != nil || d == nil {
		return []any{}, err
	}
	if a, ok := d.GetAclOk(); ok {
		return strSlice(a.GetItems()), nil
	}
	return []any{}, nil
}

func (r *mqlStackitMongoDbFlexInstance) options() (map[string]any, error) {
	d, err := r.fetchDetail()
	if err != nil || d == nil {
		return map[string]any{}, err
	}
	return stringMap(d.GetOptions()), nil
}

// ------------------------- OpenSearch -------------------------

func (r *mqlStackitOpenSearch) instances() ([]any, error) {
	c := conn(r.MqlRuntime)
	client, err := c.OpenSearch()
	if err != nil {
		return nil, err
	}
	resp, err := client.ListInstancesExecute(bgctx(), c.ProjectID(), c.Region())
	if err != nil {
		if isAccessDenied(err) {
			return []any{}, nil
		}
		return nil, err
	}
	items, _ := resp.GetInstancesOk()
	out := make([]any, 0, len(items))
	for i := range items {
		inst := items[i]
		lop := inst.GetLastOperation()
		args := map[string]*llx.RawData{
			"id":                 llx.StringData(inst.GetInstanceId()),
			"name":               llx.StringData(inst.GetName()),
			"status":             llx.StringData(string(lop.GetState())),
			"planName":           llx.StringData(inst.GetPlanName()),
			"planId":             llx.StringData(inst.GetPlanId()),
			"offeringName":       llx.StringData(inst.GetOfferingName()),
			"cfOrganizationGuid": llx.StringData(inst.GetCfOrganizationGuid()),
			"cfSpaceGuid":        llx.StringData(inst.GetCfSpaceGuid()),
			"dashboardUrl":       llx.StringData(inst.GetDashboardUrl()),
			"imageUrl":           llx.StringData(inst.GetImageUrl()),
			"parameters":         llx.DictData(toDict(inst.GetParameters())),
		}
		res, err := CreateResource(r.MqlRuntime, "stackit.openSearch.instance", args)
		if err != nil {
			return nil, err
		}
		out = append(out, res)
	}
	return out, nil
}

func (r *mqlStackitOpenSearchInstance) id() (string, error) {
	return "stackit.openSearch.instance/" + r.Id.Data, nil
}

// ------------------------- MariaDB -------------------------

func (r *mqlStackitMariaDb) instances() ([]any, error) {
	c := conn(r.MqlRuntime)
	client, err := c.MariaDb()
	if err != nil {
		return nil, err
	}
	resp, err := client.ListInstancesExecute(bgctx(), c.ProjectID(), c.Region())
	if err != nil {
		if isAccessDenied(err) {
			return []any{}, nil
		}
		return nil, err
	}
	items, _ := resp.GetInstancesOk()
	out := make([]any, 0, len(items))
	for i := range items {
		inst := items[i]
		lop := inst.GetLastOperation()
		args := map[string]*llx.RawData{
			"id":                 llx.StringData(inst.GetInstanceId()),
			"name":               llx.StringData(inst.GetName()),
			"status":             llx.StringData(string(lop.GetState())),
			"planName":           llx.StringData(inst.GetPlanName()),
			"planId":             llx.StringData(inst.GetPlanId()),
			"offeringName":       llx.StringData(inst.GetOfferingName()),
			"cfOrganizationGuid": llx.StringData(inst.GetCfOrganizationGuid()),
			"cfSpaceGuid":        llx.StringData(inst.GetCfSpaceGuid()),
			"dashboardUrl":       llx.StringData(inst.GetDashboardUrl()),
			"imageUrl":           llx.StringData(inst.GetImageUrl()),
			"parameters":         llx.DictData(toDict(inst.GetParameters())),
		}
		res, err := CreateResource(r.MqlRuntime, "stackit.mariaDb.instance", args)
		if err != nil {
			return nil, err
		}
		out = append(out, res)
	}
	return out, nil
}

func (r *mqlStackitMariaDbInstance) id() (string, error) {
	return "stackit.mariaDb.instance/" + r.Id.Data, nil
}

// ------------------------- Redis -------------------------

func (r *mqlStackitRedis) instances() ([]any, error) {
	c := conn(r.MqlRuntime)
	client, err := c.Redis()
	if err != nil {
		return nil, err
	}
	resp, err := client.ListInstancesExecute(bgctx(), c.ProjectID(), c.Region())
	if err != nil {
		if isAccessDenied(err) {
			return []any{}, nil
		}
		return nil, err
	}
	items, _ := resp.GetInstancesOk()
	out := make([]any, 0, len(items))
	for i := range items {
		inst := items[i]
		lop := inst.GetLastOperation()
		args := map[string]*llx.RawData{
			"id":                 llx.StringData(inst.GetInstanceId()),
			"name":               llx.StringData(inst.GetName()),
			"status":             llx.StringData(string(lop.GetState())),
			"planName":           llx.StringData(inst.GetPlanName()),
			"planId":             llx.StringData(inst.GetPlanId()),
			"offeringName":       llx.StringData(inst.GetOfferingName()),
			"cfOrganizationGuid": llx.StringData(inst.GetCfOrganizationGuid()),
			"cfSpaceGuid":        llx.StringData(inst.GetCfSpaceGuid()),
			"dashboardUrl":       llx.StringData(inst.GetDashboardUrl()),
			"imageUrl":           llx.StringData(inst.GetImageUrl()),
			"parameters":         llx.DictData(toDict(inst.GetParameters())),
		}
		res, err := CreateResource(r.MqlRuntime, "stackit.redis.instance", args)
		if err != nil {
			return nil, err
		}
		out = append(out, res)
	}
	return out, nil
}

func (r *mqlStackitRedisInstance) id() (string, error) {
	return "stackit.redis.instance/" + r.Id.Data, nil
}

// ------------------------- RabbitMQ -------------------------

func (r *mqlStackitRabbitMq) instances() ([]any, error) {
	c := conn(r.MqlRuntime)
	client, err := c.RabbitMq()
	if err != nil {
		return nil, err
	}
	resp, err := client.ListInstancesExecute(bgctx(), c.ProjectID(), c.Region())
	if err != nil {
		if isAccessDenied(err) {
			return []any{}, nil
		}
		return nil, err
	}
	items, _ := resp.GetInstancesOk()
	out := make([]any, 0, len(items))
	for i := range items {
		inst := items[i]
		lop := inst.GetLastOperation()
		args := map[string]*llx.RawData{
			"id":                 llx.StringData(inst.GetInstanceId()),
			"name":               llx.StringData(inst.GetName()),
			"status":             llx.StringData(string(lop.GetState())),
			"planName":           llx.StringData(inst.GetPlanName()),
			"planId":             llx.StringData(inst.GetPlanId()),
			"offeringName":       llx.StringData(inst.GetOfferingName()),
			"cfOrganizationGuid": llx.StringData(inst.GetCfOrganizationGuid()),
			"cfSpaceGuid":        llx.StringData(inst.GetCfSpaceGuid()),
			"dashboardUrl":       llx.StringData(inst.GetDashboardUrl()),
			"imageUrl":           llx.StringData(inst.GetImageUrl()),
			"parameters":         llx.DictData(toDict(inst.GetParameters())),
		}
		res, err := CreateResource(r.MqlRuntime, "stackit.rabbitMq.instance", args)
		if err != nil {
			return nil, err
		}
		out = append(out, res)
	}
	return out, nil
}

func (r *mqlStackitRabbitMqInstance) id() (string, error) {
	return "stackit.rabbitMq.instance/" + r.Id.Data, nil
}

// ------------------------- Secrets Manager -------------------------

func (r *mqlStackitSecretsManager) instances() ([]any, error) {
	c := conn(r.MqlRuntime)
	client, err := c.SecretsManager()
	if err != nil {
		return nil, err
	}
	resp, err := client.ListInstancesExecute(bgctx(), c.ProjectID())
	if err != nil {
		if isAccessDenied(err) {
			return []any{}, nil
		}
		return nil, err
	}
	items, _ := resp.GetInstancesOk()
	out := make([]any, 0, len(items))
	for i := range items {
		inst := items[i]
		args := map[string]*llx.RawData{
			"id":                 llx.StringData(inst.GetId()),
			"name":               llx.StringData(inst.GetName()),
			"state":              llx.StringData(inst.GetState()),
			"apiUrl":             llx.StringData(inst.GetApiUrl()),
			"secretsEngine":      llx.StringData(inst.GetSecretsEngine()),
			"secretCount":        llx.IntData(int64(inst.GetSecretCount())),
			"creationStartedAt":  llx.TimeDataPtr(parseDnsTime(inst.GetCreationStartDate())),
			"creationFinishedAt": llx.TimeDataPtr(parseDnsTime(inst.GetCreationFinishedDate())),
		}
		res, err := CreateResource(r.MqlRuntime, "stackit.secretsManager.instance", args)
		if err != nil {
			return nil, err
		}
		out = append(out, res)
	}
	return out, nil
}

func (r *mqlStackitSecretsManagerInstance) id() (string, error) {
	return "stackit.secretsManager.instance/" + r.Id.Data, nil
}

func (r *mqlStackitSecretsManagerInstance) acls() ([]any, error) {
	c := conn(r.MqlRuntime)
	client, err := c.SecretsManager()
	if err != nil {
		return nil, err
	}
	resp, err := client.ListACLsExecute(bgctx(), c.ProjectID(), r.Id.Data)
	if err != nil {
		if isAccessDenied(err) {
			return []any{}, nil
		}
		return nil, err
	}
	acls, _ := resp.GetAclsOk()
	out := make([]any, 0, len(acls))
	for i := range acls {
		out = append(out, acls[i].GetCidr())
	}
	return out, nil
}

// ------------------------- Observability -------------------------
//
// The list endpoint returns id/name/status/planName/serviceName only.
// planId/dashboardUrl/parameters/isUpdatable live behind GetInstance and
// are lazy-loaded once per instance.

type mqlStackitObservabilityInstanceInternal struct {
	fetched atomic.Bool
	detail  *observability.GetInstanceResponse
	lock    sync.Mutex
}

func (r *mqlStackitObservability) instances() ([]any, error) {
	c := conn(r.MqlRuntime)
	client, err := c.Observability()
	if err != nil {
		return nil, err
	}
	resp, err := client.ListInstancesExecute(bgctx(), c.ProjectID())
	if err != nil {
		if isAccessDenied(err) {
			return []any{}, nil
		}
		return nil, err
	}
	items, _ := resp.GetInstancesOk()
	out := make([]any, 0, len(items))
	for i := range items {
		inst := items[i]
		args := map[string]*llx.RawData{
			"id":           llx.StringData(inst.GetId()),
			"name":         llx.StringData(inst.GetName()),
			"status":       llx.StringData(string(inst.GetStatus())),
			"planName":     llx.StringData(inst.GetPlanName()),
			"offeringName": llx.StringData(inst.GetServiceName()),
		}
		res, err := CreateResource(r.MqlRuntime, "stackit.observability.instance", args)
		if err != nil {
			return nil, err
		}
		out = append(out, res)
	}
	return out, nil
}

func (r *mqlStackitObservabilityInstance) id() (string, error) {
	return "stackit.observability.instance/" + r.Id.Data, nil
}

func (r *mqlStackitObservabilityInstance) fetchDetail() (*observability.GetInstanceResponse, error) {
	if r.fetched.Load() {
		return r.detail, nil
	}
	r.lock.Lock()
	defer r.lock.Unlock()
	if r.fetched.Load() {
		return r.detail, nil
	}
	c := conn(r.MqlRuntime)
	client, err := c.Observability()
	if err != nil {
		return nil, err
	}
	resp, err := client.GetInstanceExecute(bgctx(), r.Id.Data, c.ProjectID())
	if err != nil {
		if isAccessDenied(err) {
			r.fetched.Store(true)
			return nil, nil
		}
		return nil, err
	}
	r.detail = resp
	r.fetched.Store(true)
	return r.detail, nil
}

func (r *mqlStackitObservabilityInstance) planId() (string, error) {
	d, err := r.fetchDetail()
	if err != nil || d == nil {
		return "", err
	}
	return d.GetPlanId(), nil
}

func (r *mqlStackitObservabilityInstance) dashboardUrl() (string, error) {
	d, err := r.fetchDetail()
	if err != nil || d == nil {
		return "", err
	}
	return d.GetDashboardUrl(), nil
}

func (r *mqlStackitObservabilityInstance) parameters() (any, error) {
	d, err := r.fetchDetail()
	if err != nil || d == nil {
		return nil, err
	}
	params, _ := d.GetParametersOk()
	return stringMap(params), nil
}

func (r *mqlStackitObservabilityInstance) isUpdatable() (bool, error) {
	d, err := r.fetchDetail()
	if err != nil || d == nil {
		return false, err
	}
	return d.GetIsUpdatable(), nil
}

// ------------------------- DBaaS init functions -------------------------
//
// Each instance type's schema declares `init(id? string)` so a user can
// query a single instance by id without listing first. Without these
// functions, `stackit.postgresFlex.instance(id: "uuid")` would hand back
// a zero-value resource and silently fail any downstream field reads.

func initStackitPostgresFlexInstance(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	id, ok := idArg(args, "id")
	if !ok {
		id, ok = conn(runtime).AssetObjectID("postgres-flex")
	}
	if !ok {
		return args, nil, nil
	}
	c := conn(runtime)
	client, err := c.PostgresFlex()
	if err != nil {
		return nil, nil, err
	}
	resp, err := client.DefaultAPI.GetInstance(bgctx(), c.ProjectID(), c.Region(), id).Execute()
	if err != nil {
		return nil, nil, err
	}
	inst, ok := resp.GetItemOk()
	if !ok {
		return nil, nil, fmt.Errorf("stackit.postgresFlex.instance with id %q not found", id)
	}
	res, err := CreateResource(runtime, "stackit.postgresFlex.instance", map[string]*llx.RawData{
		"id":     llx.StringData(inst.GetId()),
		"name":   llx.StringData(inst.GetName()),
		"status": llx.StringData(inst.GetStatus()),
		"region": llx.StringData(c.Region()),
	})
	if err != nil {
		return nil, nil, err
	}
	r := res.(*mqlStackitPostgresFlexInstance)
	r.detail = inst
	r.fetched.Store(true)
	return nil, res, nil
}

func initStackitMongoDbFlexInstance(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	id, ok := idArg(args, "id")
	if !ok {
		id, ok = conn(runtime).AssetObjectID("mongodb-flex")
	}
	if !ok {
		return args, nil, nil
	}
	c := conn(runtime)
	client, err := c.MongoDbFlex()
	if err != nil {
		return nil, nil, err
	}
	resp, err := client.GetInstanceExecute(bgctx(), c.ProjectID(), id, c.Region())
	if err != nil {
		return nil, nil, err
	}
	inst, ok := resp.GetItemOk()
	if !ok {
		return nil, nil, fmt.Errorf("stackit.mongoDbFlex.instance with id %q not found", id)
	}
	res, err := CreateResource(runtime, "stackit.mongoDbFlex.instance", map[string]*llx.RawData{
		"id":     llx.StringData(inst.GetId()),
		"name":   llx.StringData(inst.GetName()),
		"status": llx.StringData(string(inst.GetStatus())),
		"region": llx.StringData(c.Region()),
	})
	if err != nil {
		return nil, nil, err
	}
	r := res.(*mqlStackitMongoDbFlexInstance)
	r.detail = &inst
	r.fetched.Store(true)
	return nil, res, nil
}

func initStackitOpenSearchInstance(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	id, ok := idArg(args, "id")
	if !ok {
		id, ok = conn(runtime).AssetObjectID("opensearch")
	}
	if !ok {
		return args, nil, nil
	}
	c := conn(runtime)
	client, err := c.OpenSearch()
	if err != nil {
		return nil, nil, err
	}
	inst, err := client.GetInstanceExecute(bgctx(), c.ProjectID(), c.Region(), id)
	if err != nil {
		return nil, nil, err
	}
	if inst == nil {
		return nil, nil, fmt.Errorf("stackit.openSearch.instance with id %q not found", id)
	}
	lop := inst.GetLastOperation()
	res, err := CreateResource(runtime, "stackit.openSearch.instance", map[string]*llx.RawData{
		"id":                 llx.StringData(inst.GetInstanceId()),
		"name":               llx.StringData(inst.GetName()),
		"status":             llx.StringData(string(lop.GetState())),
		"planName":           llx.StringData(inst.GetPlanName()),
		"planId":             llx.StringData(inst.GetPlanId()),
		"offeringName":       llx.StringData(inst.GetOfferingName()),
		"cfOrganizationGuid": llx.StringData(inst.GetCfOrganizationGuid()),
		"cfSpaceGuid":        llx.StringData(inst.GetCfSpaceGuid()),
		"dashboardUrl":       llx.StringData(inst.GetDashboardUrl()),
		"imageUrl":           llx.StringData(inst.GetImageUrl()),
		"parameters":         llx.DictData(toDict(inst.GetParameters())),
	})
	if err != nil {
		return nil, nil, err
	}
	return nil, res, nil
}

func initStackitMariaDbInstance(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	id, ok := idArg(args, "id")
	if !ok {
		id, ok = conn(runtime).AssetObjectID("mariadb")
	}
	if !ok {
		return args, nil, nil
	}
	c := conn(runtime)
	client, err := c.MariaDb()
	if err != nil {
		return nil, nil, err
	}
	inst, err := client.GetInstanceExecute(bgctx(), c.ProjectID(), c.Region(), id)
	if err != nil {
		return nil, nil, err
	}
	if inst == nil {
		return nil, nil, fmt.Errorf("stackit.mariaDb.instance with id %q not found", id)
	}
	lop := inst.GetLastOperation()
	res, err := CreateResource(runtime, "stackit.mariaDb.instance", map[string]*llx.RawData{
		"id":                 llx.StringData(inst.GetInstanceId()),
		"name":               llx.StringData(inst.GetName()),
		"status":             llx.StringData(string(lop.GetState())),
		"planName":           llx.StringData(inst.GetPlanName()),
		"planId":             llx.StringData(inst.GetPlanId()),
		"offeringName":       llx.StringData(inst.GetOfferingName()),
		"cfOrganizationGuid": llx.StringData(inst.GetCfOrganizationGuid()),
		"cfSpaceGuid":        llx.StringData(inst.GetCfSpaceGuid()),
		"dashboardUrl":       llx.StringData(inst.GetDashboardUrl()),
		"imageUrl":           llx.StringData(inst.GetImageUrl()),
		"parameters":         llx.DictData(toDict(inst.GetParameters())),
	})
	if err != nil {
		return nil, nil, err
	}
	return nil, res, nil
}

func initStackitRedisInstance(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	id, ok := idArg(args, "id")
	if !ok {
		id, ok = conn(runtime).AssetObjectID("redis")
	}
	if !ok {
		return args, nil, nil
	}
	c := conn(runtime)
	client, err := c.Redis()
	if err != nil {
		return nil, nil, err
	}
	inst, err := client.GetInstanceExecute(bgctx(), c.ProjectID(), c.Region(), id)
	if err != nil {
		return nil, nil, err
	}
	if inst == nil {
		return nil, nil, fmt.Errorf("stackit.redis.instance with id %q not found", id)
	}
	lop := inst.GetLastOperation()
	res, err := CreateResource(runtime, "stackit.redis.instance", map[string]*llx.RawData{
		"id":                 llx.StringData(inst.GetInstanceId()),
		"name":               llx.StringData(inst.GetName()),
		"status":             llx.StringData(string(lop.GetState())),
		"planName":           llx.StringData(inst.GetPlanName()),
		"planId":             llx.StringData(inst.GetPlanId()),
		"offeringName":       llx.StringData(inst.GetOfferingName()),
		"cfOrganizationGuid": llx.StringData(inst.GetCfOrganizationGuid()),
		"cfSpaceGuid":        llx.StringData(inst.GetCfSpaceGuid()),
		"dashboardUrl":       llx.StringData(inst.GetDashboardUrl()),
		"imageUrl":           llx.StringData(inst.GetImageUrl()),
		"parameters":         llx.DictData(toDict(inst.GetParameters())),
	})
	if err != nil {
		return nil, nil, err
	}
	return nil, res, nil
}

func initStackitRabbitMqInstance(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	id, ok := idArg(args, "id")
	if !ok {
		id, ok = conn(runtime).AssetObjectID("rabbitmq")
	}
	if !ok {
		return args, nil, nil
	}
	c := conn(runtime)
	client, err := c.RabbitMq()
	if err != nil {
		return nil, nil, err
	}
	inst, err := client.GetInstanceExecute(bgctx(), c.ProjectID(), c.Region(), id)
	if err != nil {
		return nil, nil, err
	}
	if inst == nil {
		return nil, nil, fmt.Errorf("stackit.rabbitMq.instance with id %q not found", id)
	}
	lop := inst.GetLastOperation()
	res, err := CreateResource(runtime, "stackit.rabbitMq.instance", map[string]*llx.RawData{
		"id":                 llx.StringData(inst.GetInstanceId()),
		"name":               llx.StringData(inst.GetName()),
		"status":             llx.StringData(string(lop.GetState())),
		"planName":           llx.StringData(inst.GetPlanName()),
		"planId":             llx.StringData(inst.GetPlanId()),
		"offeringName":       llx.StringData(inst.GetOfferingName()),
		"cfOrganizationGuid": llx.StringData(inst.GetCfOrganizationGuid()),
		"cfSpaceGuid":        llx.StringData(inst.GetCfSpaceGuid()),
		"dashboardUrl":       llx.StringData(inst.GetDashboardUrl()),
		"imageUrl":           llx.StringData(inst.GetImageUrl()),
		"parameters":         llx.DictData(toDict(inst.GetParameters())),
	})
	if err != nil {
		return nil, nil, err
	}
	return nil, res, nil
}

func initStackitSecretsManagerInstance(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	id, ok := idArg(args, "id")
	if !ok {
		id, ok = conn(runtime).AssetObjectID("secrets-manager")
	}
	if !ok {
		return args, nil, nil
	}
	c := conn(runtime)
	client, err := c.SecretsManager()
	if err != nil {
		return nil, nil, err
	}
	inst, err := client.GetInstanceExecute(bgctx(), c.ProjectID(), id)
	if err != nil {
		return nil, nil, err
	}
	if inst == nil {
		return nil, nil, fmt.Errorf("stackit.secretsManager.instance with id %q not found", id)
	}
	res, err := CreateResource(runtime, "stackit.secretsManager.instance", map[string]*llx.RawData{
		"id":                 llx.StringData(inst.GetId()),
		"name":               llx.StringData(inst.GetName()),
		"state":              llx.StringData(inst.GetState()),
		"apiUrl":             llx.StringData(inst.GetApiUrl()),
		"secretsEngine":      llx.StringData(inst.GetSecretsEngine()),
		"secretCount":        llx.IntData(int64(inst.GetSecretCount())),
		"creationStartedAt":  llx.TimeDataPtr(parseDnsTime(inst.GetCreationStartDate())),
		"creationFinishedAt": llx.TimeDataPtr(parseDnsTime(inst.GetCreationFinishedDate())),
	})
	if err != nil {
		return nil, nil, err
	}
	return nil, res, nil
}

func initStackitObservabilityInstance(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	id, ok := idArg(args, "id")
	if !ok {
		return args, nil, nil
	}
	c := conn(runtime)
	client, err := c.Observability()
	if err != nil {
		return nil, nil, err
	}
	resp, err := client.GetInstanceExecute(bgctx(), id, c.ProjectID())
	if err != nil {
		return nil, nil, err
	}
	res, err := CreateResource(runtime, "stackit.observability.instance", map[string]*llx.RawData{
		"id":           llx.StringData(resp.GetId()),
		"name":         llx.StringData(resp.GetName()),
		"status":       llx.StringData(string(resp.GetStatus())),
		"planName":     llx.StringData(resp.GetPlanName()),
		"offeringName": llx.StringData(resp.GetServiceName()),
	})
	if err != nil {
		return nil, nil, err
	}
	r := res.(*mqlStackitObservabilityInstance)
	r.detail = resp
	r.fetched.Store(true)
	return nil, res, nil
}

// ------------------------- LogMe -------------------------
//
// CF service-broker shape, identical to MariaDB/Redis/RabbitMQ: the list
// returns the full instance surface (plan, offering, parameters), so there
// is no separate detail fetch.

func (r *mqlStackitLogMe) instances() ([]any, error) {
	c := conn(r.MqlRuntime)
	client, err := c.LogMe()
	if err != nil {
		return nil, err
	}
	resp, err := client.ListInstancesExecute(bgctx(), c.ProjectID(), c.Region())
	if err != nil {
		if isAccessDenied(err) {
			return []any{}, nil
		}
		return nil, err
	}
	items, _ := resp.GetInstancesOk()
	out := make([]any, 0, len(items))
	for i := range items {
		inst := items[i]
		lop := inst.GetLastOperation()
		args := map[string]*llx.RawData{
			"id":                 llx.StringData(inst.GetInstanceId()),
			"name":               llx.StringData(inst.GetName()),
			"status":             llx.StringData(string(lop.GetState())),
			"planName":           llx.StringData(inst.GetPlanName()),
			"planId":             llx.StringData(inst.GetPlanId()),
			"offeringName":       llx.StringData(inst.GetOfferingName()),
			"cfOrganizationGuid": llx.StringData(inst.GetCfOrganizationGuid()),
			"cfSpaceGuid":        llx.StringData(inst.GetCfSpaceGuid()),
			"dashboardUrl":       llx.StringData(inst.GetDashboardUrl()),
			"imageUrl":           llx.StringData(inst.GetImageUrl()),
			"parameters":         llx.DictData(toDict(inst.GetParameters())),
		}
		res, err := CreateResource(r.MqlRuntime, "stackit.logMe.instance", args)
		if err != nil {
			return nil, err
		}
		out = append(out, res)
	}
	return out, nil
}

func (r *mqlStackitLogMeInstance) id() (string, error) {
	return "stackit.logMe.instance/" + r.Id.Data, nil
}

func initStackitLogMeInstance(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	id, ok := idArg(args, "id")
	if !ok {
		id, ok = conn(runtime).AssetObjectID("logme")
	}
	if !ok {
		return args, nil, nil
	}
	c := conn(runtime)
	client, err := c.LogMe()
	if err != nil {
		return nil, nil, err
	}
	inst, err := client.GetInstanceExecute(bgctx(), c.ProjectID(), c.Region(), id)
	if err != nil {
		return nil, nil, err
	}
	if inst == nil {
		return nil, nil, fmt.Errorf("stackit.logMe.instance with id %q not found", id)
	}
	lop := inst.GetLastOperation()
	res, err := CreateResource(runtime, "stackit.logMe.instance", map[string]*llx.RawData{
		"id":                 llx.StringData(inst.GetInstanceId()),
		"name":               llx.StringData(inst.GetName()),
		"status":             llx.StringData(string(lop.GetState())),
		"planName":           llx.StringData(inst.GetPlanName()),
		"planId":             llx.StringData(inst.GetPlanId()),
		"offeringName":       llx.StringData(inst.GetOfferingName()),
		"cfOrganizationGuid": llx.StringData(inst.GetCfOrganizationGuid()),
		"cfSpaceGuid":        llx.StringData(inst.GetCfSpaceGuid()),
		"dashboardUrl":       llx.StringData(inst.GetDashboardUrl()),
		"imageUrl":           llx.StringData(inst.GetImageUrl()),
		"parameters":         llx.DictData(toDict(inst.GetParameters())),
	})
	if err != nil {
		return nil, nil, err
	}
	return nil, res, nil
}

// ------------------------- SQLServer Flex -------------------------
//
// Flex shape, identical to Postgres/MongoDB Flex: the list returns
// id/name/status only; version/flavor/acl/replicas/storage/backupSchedule/
// options are lazy-loaded once per instance via GetInstance.

type mqlStackitSqlServerFlexInstanceInternal struct {
	fetched atomic.Bool
	detail  *sqlserverflex.Instance
	lock    sync.Mutex
}

func (r *mqlStackitSqlServerFlex) instances() ([]any, error) {
	c := conn(r.MqlRuntime)
	client, err := c.SqlServerFlex()
	if err != nil {
		return nil, err
	}
	resp, err := client.DefaultAPI.ListInstances(bgctx(), c.ProjectID(), c.Region()).Execute()
	if err != nil {
		if isAccessDenied(err) {
			return []any{}, nil
		}
		return nil, err
	}
	items, _ := resp.GetItemsOk()
	out := make([]any, 0, len(items))
	for i := range items {
		inst := items[i]
		args := map[string]*llx.RawData{
			"id":     llx.StringData(inst.GetId()),
			"name":   llx.StringData(inst.GetName()),
			"status": llx.StringData(inst.GetStatus()),
			"region": llx.StringData(c.Region()),
		}
		res, err := CreateResource(r.MqlRuntime, "stackit.sqlServerFlex.instance", args)
		if err != nil {
			return nil, err
		}
		out = append(out, res)
	}
	return out, nil
}

func (r *mqlStackitSqlServerFlexInstance) id() (string, error) {
	return "stackit.sqlServerFlex.instance/" + r.Id.Data, nil
}

func (r *mqlStackitSqlServerFlexInstance) fetchDetail() (*sqlserverflex.Instance, error) {
	if r.fetched.Load() {
		return r.detail, nil
	}
	r.lock.Lock()
	defer r.lock.Unlock()
	if r.fetched.Load() {
		return r.detail, nil
	}
	c := conn(r.MqlRuntime)
	client, err := c.SqlServerFlex()
	if err != nil {
		return nil, err
	}
	resp, err := client.DefaultAPI.GetInstance(bgctx(), c.ProjectID(), r.Id.Data, c.Region()).Execute()
	if err != nil {
		if isAccessDenied(err) {
			r.fetched.Store(true)
			return nil, nil
		}
		return nil, err
	}
	if item, ok := resp.GetItemOk(); ok {
		r.detail = item
	}
	r.fetched.Store(true)
	return r.detail, nil
}

func (r *mqlStackitSqlServerFlexInstance) version() (string, error) {
	d, err := r.fetchDetail()
	if err != nil || d == nil {
		return "", err
	}
	return d.GetVersion(), nil
}

func (r *mqlStackitSqlServerFlexInstance) flavor() (any, error) {
	d, err := r.fetchDetail()
	if err != nil || d == nil {
		return nil, err
	}
	return toDict(d.GetFlavor()), nil
}

func (r *mqlStackitSqlServerFlexInstance) acl() ([]any, error) {
	d, err := r.fetchDetail()
	if err != nil || d == nil {
		return []any{}, err
	}
	if a, ok := d.GetAclOk(); ok {
		return strSlice(a.GetItems()), nil
	}
	return []any{}, nil
}

func (r *mqlStackitSqlServerFlexInstance) replicas() (int64, error) {
	d, err := r.fetchDetail()
	if err != nil || d == nil {
		return 0, err
	}
	return int64(d.GetReplicas()), nil
}

func (r *mqlStackitSqlServerFlexInstance) storage() (any, error) {
	d, err := r.fetchDetail()
	if err != nil || d == nil {
		return nil, err
	}
	return toDict(d.GetStorage()), nil
}

func (r *mqlStackitSqlServerFlexInstance) backupSchedule() (string, error) {
	d, err := r.fetchDetail()
	if err != nil || d == nil {
		return "", err
	}
	return d.GetBackupSchedule(), nil
}

func (r *mqlStackitSqlServerFlexInstance) options() (map[string]any, error) {
	d, err := r.fetchDetail()
	if err != nil || d == nil {
		return map[string]any{}, err
	}
	return stringMap(d.GetOptions()), nil
}

func initStackitSqlServerFlexInstance(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	id, ok := idArg(args, "id")
	if !ok {
		id, ok = conn(runtime).AssetObjectID("sqlserver-flex")
	}
	if !ok {
		return args, nil, nil
	}
	c := conn(runtime)
	client, err := c.SqlServerFlex()
	if err != nil {
		return nil, nil, err
	}
	resp, err := client.DefaultAPI.GetInstance(bgctx(), c.ProjectID(), id, c.Region()).Execute()
	if err != nil {
		return nil, nil, err
	}
	inst, ok := resp.GetItemOk()
	if !ok {
		return nil, nil, fmt.Errorf("stackit.sqlServerFlex.instance with id %q not found", id)
	}
	res, err := CreateResource(runtime, "stackit.sqlServerFlex.instance", map[string]*llx.RawData{
		"id":     llx.StringData(inst.GetId()),
		"name":   llx.StringData(inst.GetName()),
		"status": llx.StringData(inst.GetStatus()),
		"region": llx.StringData(c.Region()),
	})
	if err != nil {
		return nil, nil, err
	}
	r := res.(*mqlStackitSqlServerFlexInstance)
	r.detail = inst
	r.fetched.Store(true)
	return nil, res, nil
}
