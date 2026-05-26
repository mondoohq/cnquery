// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"time"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/huggingface/connection"
	hfmodels "go.mondoo.com/mql/v13/providers/huggingface/internal/huggingface-hub-go/models"
	"go.mondoo.com/mql/v13/types"
)

func hfConn(r *plugin.Runtime) *connection.HuggingfaceConnection {
	return r.Connection.(*connection.HuggingfaceConnection)
}

type mqlHuggingfaceInternal struct {
	whoami     *hfmodels.User
	whoamiOnce sync.Once
	whoamiErr  error
}

func (r *mqlHuggingface) fetchWhoAmI() (*hfmodels.User, error) {
	r.whoamiOnce.Do(func() {
		client := hfConn(r.MqlRuntime).Client()
		r.whoami, r.whoamiErr = client.WhoAmI(context.Background())
	})
	return r.whoami, r.whoamiErr
}

func (r *mqlHuggingface) id() (string, error) {
	return "huggingface", nil
}

func (r *mqlHuggingface) user() (*mqlHuggingfaceUser, error) {
	user, err := r.fetchWhoAmI()
	if err != nil {
		return nil, err
	}

	res, err := CreateResource(r.MqlRuntime, "huggingface.user", map[string]*llx.RawData{
		"__id":      llx.StringData("huggingface.user/" + user.ID),
		"id":        llx.StringData(user.ID),
		"name":      llx.StringData(user.Name),
		"fullname":  llx.StringData(user.Fullname),
		"type":      llx.StringData(user.Type),
		"isPro":     llx.BoolData(user.IsPro),
		"avatarUrl": llx.StringData(user.AvatarURL),
	})
	if err != nil {
		return nil, err
	}

	return res.(*mqlHuggingfaceUser), nil
}

func (r *mqlHuggingface) organizations() ([]any, error) {
	user, err := r.fetchWhoAmI()
	if err != nil {
		return nil, err
	}

	orgs := make([]any, 0, len(user.Orgs))
	for _, org := range user.Orgs {
		res, err := CreateResource(r.MqlRuntime, "huggingface.organization", map[string]*llx.RawData{
			"__id":         llx.StringData("huggingface.organization/" + org.ID),
			"id":           llx.StringData(org.ID),
			"name":         llx.StringData(org.Name),
			"fullname":     llx.StringData(org.Fullname),
			"email":        llx.StringData(org.Email),
			"canPay":       llx.BoolData(org.CanPay),
			"avatarUrl":    llx.StringData(org.AvatarURL),
			"roleInOrg":    llx.StringData(org.RoleInOrg),
			"isEnterprise": llx.BoolData(org.IsEnterprise),
		})
		if err != nil {
			return nil, err
		}
		orgs = append(orgs, res)
	}

	return orgs, nil
}

// namespace returns the single namespace this connection is scoped to.
func (r *mqlHuggingface) resolveNamespace() (string, error) {
	conn := hfConn(r.MqlRuntime)
	if ns := conn.Namespace(); ns != "" {
		return ns, nil
	}
	user, err := r.fetchWhoAmI()
	if err != nil {
		return "", err
	}
	return user.Name, nil
}

func (r *mqlHuggingface) models() ([]any, error) {
	ns, err := r.resolveNamespace()
	if err != nil {
		return nil, err
	}

	client := hfConn(r.MqlRuntime).Client()

	opts := hfmodels.NewModelListOptions()
	opts.Author = ns
	opts.Limit = 1000

	modelList, err := client.ListModels(context.Background(), opts)
	if err != nil {
		return nil, err
	}

	models := make([]any, 0, len(modelList.Models))
	for _, m := range modelList.Models {
		res, err := createModelResource(r.MqlRuntime, m)
		if err != nil {
			return nil, err
		}
		models = append(models, res)
	}

	return models, nil
}

// createModelResource builds a model from list API data. Fields not returned
// by the list API (createdAt, license, cardData, config, siblings) are computed
// methods that call fetchDetail() on demand.
func createModelResource(runtime *plugin.Runtime, m hfmodels.Model) (*mqlHuggingfaceModel, error) {
	lastMod := parseHFTime(m.LastModified)

	res, err := CreateResource(runtime, "huggingface.model", map[string]*llx.RawData{
		"__id":         llx.StringData("huggingface.model/" + m.ID),
		"id":           llx.StringData(m.ID),
		"modelId":      llx.StringData(m.ModelID),
		"private":      llx.BoolData(m.Private),
		"pipelineTag":  llx.StringData(m.PipelineTag),
		"libraryName":  llx.StringData(m.LibraryName),
		"tags":         llx.ArrayData(stringsToInterface(m.Tags), types.String),
		"downloads":    llx.IntData(int64(m.Downloads)),
		"likes":        llx.IntData(int64(m.Likes)),
		"lastModified": llx.TimeDataPtr(lastMod),
		"gated":        llx.BoolData(m.Gated.IsGated),
		"disabled":     llx.BoolData(m.Disabled),
	})
	if err != nil {
		return nil, err
	}
	model := res.(*mqlHuggingfaceModel)
	model.listAuthor = m.Author
	model.listSha = m.Sha
	return model, nil
}

func (r *mqlHuggingface) datasets() ([]any, error) {
	ns, err := r.resolveNamespace()
	if err != nil {
		return nil, err
	}

	client := hfConn(r.MqlRuntime).Client()

	opts := hfmodels.NewDatasetListOptions()
	opts.Author = ns
	opts.Limit = 1000

	datasetList, err := client.ListDatasets(context.Background(), opts)
	if err != nil {
		return nil, err
	}

	datasets := make([]any, 0, len(datasetList.Datasets))
	for _, d := range datasetList.Datasets {
		res, err := CreateResource(r.MqlRuntime, "huggingface.dataset", map[string]*llx.RawData{
			"__id":        llx.StringData("huggingface.dataset/" + d.ID),
			"id":          llx.StringData(d.ID),
			"name":        llx.StringData(d.Name),
			"description": llx.StringData(d.Description),
			"tags":        llx.ArrayData(stringsToInterface(d.Tags), types.String),
			"author":      llx.StringData(d.Author),
			"downloads":   llx.IntData(int64(d.Downloads)),
			"likes":       llx.IntData(int64(d.Likes)),
			"private":     llx.BoolData(d.Private),
		})
		if err != nil {
			return nil, err
		}
		datasets = append(datasets, res)
	}

	return datasets, nil
}

func (r *mqlHuggingface) spaces() ([]any, error) {
	ns, err := r.resolveNamespace()
	if err != nil {
		return nil, err
	}

	client := hfConn(r.MqlRuntime).Client()

	opts := hfmodels.NewSpaceListOptions()
	opts.Author = ns
	opts.Limit = 1000

	spaceList, err := client.ListSpaces(context.Background(), opts)
	if err != nil {
		return nil, err
	}

	spaces := make([]any, 0, len(spaceList.Spaces))
	for _, sp := range spaceList.Spaces {
		res, err := CreateResource(r.MqlRuntime, "huggingface.space", map[string]*llx.RawData{
			"__id":        llx.StringData("huggingface.space/" + sp.ID),
			"id":          llx.StringData(sp.ID),
			"name":        llx.StringData(sp.Name),
			"description": llx.StringData(sp.Description),
			"tags":        llx.ArrayData(stringsToInterface(sp.Tags), types.String),
			"author":      llx.StringData(sp.Author),
			"likes":       llx.IntData(int64(sp.Likes)),
			"private":     llx.BoolData(sp.Private),
		})
		if err != nil {
			return nil, err
		}
		spaces = append(spaces, res)
	}

	return spaces, nil
}

func (r *mqlHuggingface) webhooks() ([]any, error) {
	client := hfConn(r.MqlRuntime).Client()

	webhookList, err := client.ListWebhooks(context.Background(), nil)
	if isAccessDenied(err) {
		return []any{}, nil
	}
	if err != nil {
		return nil, err
	}

	webhooks := make([]any, 0, len(webhookList))
	for _, w := range webhookList {
		watched := make([]any, 0, len(w.Watched))
		for _, wd := range w.Watched {
			watched = append(watched, map[string]any{
				"type": wd.Type,
				"name": wd.Name,
			})
		}

		res, err := CreateResource(r.MqlRuntime, "huggingface.webhook", map[string]*llx.RawData{
			"__id":       llx.StringData("huggingface.webhook/" + w.ID),
			"id":         llx.StringData(w.ID),
			"webhookUrl": llx.StringData(w.URL),
			"domains":    llx.ArrayData(stringsToInterface(w.Domains), types.String),
			"disabled":   llx.BoolData(w.Disabled),
			"watched":    llx.ArrayData(watched, types.Dict),
		})
		if err != nil {
			return nil, err
		}
		webhooks = append(webhooks, res)
	}

	return webhooks, nil
}

func isAccessDenied(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "(status: 401)") || strings.Contains(msg, "(status: 403)")
}

func isNotFound(err error) bool {
	return err != nil && strings.Contains(err.Error(), "(status: 404)")
}

func (r *mqlHuggingface) inferenceEndpoints() ([]any, error) {
	ns, err := r.resolveNamespace()
	if err != nil {
		return nil, err
	}

	client := hfConn(r.MqlRuntime).Client()

	endpointList, err := client.ListInferenceEndpoints(context.Background(), ns)
	if isAccessDenied(err) {
		return []any{}, nil
	}
	if err != nil {
		return nil, err
	}

	endpoints := make([]any, 0, len(endpointList.Endpoints))
	for _, ep := range endpointList.Endpoints {
		res, err := CreateResource(r.MqlRuntime, "huggingface.inferenceEndpoint", map[string]*llx.RawData{
			"__id":        llx.StringData("huggingface.inferenceEndpoint/" + ep.ID),
			"id":          llx.StringData(ep.ID),
			"name":        llx.StringData(ep.Name),
			"model":       llx.StringData(ep.Model),
			"framework":   llx.StringData(ep.Framework),
			"status":      llx.StringData(ep.Status),
			"endpointUrl": llx.StringData(ep.EndpointURL),
			"createdAt":   llx.TimeDataPtr(parseHFTime(ep.CreatedAt)),
			"updatedAt":   llx.TimeDataPtr(parseHFTime(ep.UpdatedAt)),
		})
		if err != nil {
			return nil, err
		}
		endpoints = append(endpoints, res)
	}

	return endpoints, nil
}

// --- huggingface.user ---

func initHuggingfaceUser(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 1 {
		return args, nil, nil
	}

	hf, err := CreateResource(runtime, "huggingface", map[string]*llx.RawData{})
	if err != nil {
		return nil, nil, err
	}
	root := hf.(*mqlHuggingface)

	user, err := root.fetchWhoAmI()
	if err != nil {
		return nil, nil, err
	}

	args["__id"] = llx.StringData("huggingface.user/" + user.ID)
	args["id"] = llx.StringData(user.ID)
	args["name"] = llx.StringData(user.Name)
	args["fullname"] = llx.StringData(user.Fullname)
	args["type"] = llx.StringData(user.Type)
	args["isPro"] = llx.BoolData(user.IsPro)
	args["avatarUrl"] = llx.StringData(user.AvatarURL)

	return args, nil, nil
}

func (r *mqlHuggingfaceUser) id() (string, error) {
	return "huggingface.user/" + r.Id.Data, nil
}

func (r *mqlHuggingfaceUser) accessToken() (*mqlHuggingfaceAccessToken, error) {
	hf, err := CreateResource(r.MqlRuntime, "huggingface", map[string]*llx.RawData{})
	if err != nil {
		return nil, err
	}
	root := hf.(*mqlHuggingface)

	user, err := root.fetchWhoAmI()
	if err != nil {
		return nil, err
	}

	auth := user.Auth.AccessToken

	globalPerms := make([]any, 0, len(auth.FineGrained.Global))
	for _, p := range auth.FineGrained.Global {
		globalPerms = append(globalPerms, p)
	}

	scopedPerms := make([]any, 0, len(auth.FineGrained.Scoped))
	for _, s := range auth.FineGrained.Scoped {
		perms := make([]any, 0, len(s.Permissions))
		for _, p := range s.Permissions {
			perms = append(perms, p)
		}

		res, err := CreateResource(r.MqlRuntime, "huggingface.accessToken.scope", map[string]*llx.RawData{
			"__id":        llx.StringData("huggingface.accessToken.scope/" + s.Entity.ID),
			"entityId":    llx.StringData(s.Entity.ID),
			"entityType":  llx.StringData(s.Entity.Type),
			"entityName":  llx.StringData(s.Entity.Name),
			"permissions": llx.ArrayData(perms, types.String),
		})
		if err != nil {
			return nil, err
		}
		scopedPerms = append(scopedPerms, res)
	}

	res, err := CreateResource(r.MqlRuntime, "huggingface.accessToken", map[string]*llx.RawData{
		"__id":              llx.StringData("huggingface.accessToken/" + auth.DisplayName),
		"displayName":       llx.StringData(auth.DisplayName),
		"role":              llx.StringData(auth.Role),
		"createdAt":         llx.TimeDataPtr(parseHFTime(auth.CreatedAt)),
		"canReadGatedRepos": llx.BoolData(auth.FineGrained.CanReadGatedRepos),
		"globalPermissions": llx.ArrayData(globalPerms, types.String),
		"scopedPermissions": llx.ArrayData(scopedPerms, types.Resource("huggingface.accessToken.scope")),
	})
	if err != nil {
		return nil, err
	}

	return res.(*mqlHuggingfaceAccessToken), nil
}

// --- huggingface.model ---

func initHuggingfaceModel(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 1 {
		return args, nil, nil
	}

	idRaw, ok := args["id"]
	if !ok || idRaw == nil || idRaw.Value == nil {
		return args, nil, nil
	}
	modelID, ok := idRaw.Value.(string)
	if !ok || modelID == "" {
		return args, nil, nil
	}

	modelID, err := normalizeModelID(modelID)
	if err != nil {
		return nil, nil, err
	}
	args["id"] = llx.StringData(modelID)

	client := hfConn(runtime).Client()
	detail, err := client.GetModelDetail(context.Background(), modelID)
	if err != nil {
		return nil, nil, err
	}

	createdAt := parseHFTime(detail.CreatedAt)
	lastMod := parseHFTime(detail.LastModified)
	args["__id"] = llx.StringData("huggingface.model/" + detail.ID)
	args["modelId"] = llx.StringData(detail.ModelID)
	args["author"] = llx.StringData(detail.Author)
	args["private"] = llx.BoolData(detail.Private)
	args["pipelineTag"] = llx.StringData(detail.PipelineTag)
	args["libraryName"] = llx.StringData(detail.LibraryName)
	args["tags"] = llx.ArrayData(stringsToInterface(detail.Tags), types.String)
	args["downloads"] = llx.IntData(int64(detail.Downloads))
	args["likes"] = llx.IntData(int64(detail.Likes))
	args["sha"] = llx.StringData(detail.Sha)
	args["lastModified"] = llx.TimeDataPtr(lastMod)
	args["gated"] = llx.BoolData(detail.Gated.IsGated)
	args["disabled"] = llx.BoolData(detail.Disabled)
	args["createdAt"] = llx.TimeDataPtr(createdAt)

	res, err := CreateResource(runtime, "huggingface.model", args)
	if err != nil {
		return nil, nil, err
	}
	model := res.(*mqlHuggingfaceModel)
	model.detailOnce.Do(func() {
		model.detail = detail
	})

	return args, model, nil
}

type mqlHuggingfaceModelInternal struct {
	listAuthor string
	listSha    string
	detail     *hfmodels.ModelDetail
	detailOnce sync.Once
	detailErr  error
}

func (r *mqlHuggingfaceModel) fetchDetail() (*hfmodels.ModelDetail, error) {
	r.detailOnce.Do(func() {
		client := hfConn(r.MqlRuntime).Client()
		r.detail, r.detailErr = client.GetModelDetail(context.Background(), r.Id.Data)
	})
	return r.detail, r.detailErr
}

func (r *mqlHuggingfaceModel) id() (string, error) {
	return "huggingface.model/" + r.Id.Data, nil
}

func (r *mqlHuggingfaceModel) author() (string, error) {
	if r.listAuthor != "" {
		return r.listAuthor, nil
	}
	detail, err := r.fetchDetail()
	if err != nil {
		return "", err
	}
	if detail.Author != "" {
		return detail.Author, nil
	}
	if parts := strings.SplitN(detail.ID, "/", 2); len(parts) == 2 {
		return parts[0], nil
	}
	return "", nil
}

func (r *mqlHuggingfaceModel) sha() (string, error) {
	if r.listSha != "" {
		return r.listSha, nil
	}
	detail, err := r.fetchDetail()
	if err != nil {
		return "", err
	}
	return detail.Sha, nil
}

func (r *mqlHuggingfaceModel) createdAt() (*time.Time, error) {
	detail, err := r.fetchDetail()
	if err != nil {
		return nil, err
	}
	return parseHFTime(detail.CreatedAt), nil
}

func (r *mqlHuggingfaceModel) license() (string, error) {
	detail, err := r.fetchDetail()
	if err != nil {
		return "", err
	}
	if cd := detail.CardData; cd != nil {
		if v, ok := cd["license"]; ok {
			return fmt.Sprintf("%v", v), nil
		}
	}
	return "", nil
}

func (r *mqlHuggingfaceModel) cardData() (any, error) {
	detail, err := r.fetchDetail()
	if err != nil {
		return nil, err
	}
	if detail.CardData == nil {
		return map[string]any{}, nil
	}
	return detail.CardData, nil
}

func (r *mqlHuggingfaceModel) modelCard() (string, error) {
	client := hfConn(r.MqlRuntime).Client()
	data, err := client.DownloadModelFile(context.Background(), r.Id.Data, "README.md")
	if isAccessDenied(err) || isNotFound(err) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func (r *mqlHuggingfaceModel) config() (any, error) {
	client := hfConn(r.MqlRuntime).Client()
	data, err := client.DownloadModelFile(context.Background(), r.Id.Data, "config.json")
	if isAccessDenied(err) || isNotFound(err) {
		return map[string]any{}, nil
	}
	if err != nil {
		return nil, err
	}

	var cfg map[string]any
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config.json: %w", err)
	}
	return cfg, nil
}

func (r *mqlHuggingfaceModel) siblings() ([]any, error) {
	detail, err := r.fetchDetail()
	if err != nil {
		return nil, err
	}

	result := make([]any, 0, len(detail.Siblings))
	for _, s := range detail.Siblings {
		result = append(result, map[string]any{
			"rfilename": s.Rfilename,
		})
	}
	return result, nil
}

// --- simple id() methods ---

func (r *mqlHuggingfaceOrganization) id() (string, error) {
	return "huggingface.organization/" + r.Id.Data, nil
}

func (r *mqlHuggingfaceAccessToken) id() (string, error) {
	return "huggingface.accessToken/" + r.DisplayName.Data, nil
}

func (r *mqlHuggingfaceAccessTokenScope) id() (string, error) {
	return "huggingface.accessToken.scope/" + r.EntityId.Data, nil
}

func (r *mqlHuggingfaceDataset) id() (string, error) {
	return "huggingface.dataset/" + r.Id.Data, nil
}

func (r *mqlHuggingfaceSpace) id() (string, error) {
	return "huggingface.space/" + r.Id.Data, nil
}

func (r *mqlHuggingfaceWebhook) id() (string, error) {
	return "huggingface.webhook/" + r.Id.Data, nil
}

func (r *mqlHuggingfaceInferenceEndpoint) id() (string, error) {
	return "huggingface.inferenceEndpoint/" + r.Id.Data, nil
}

// --- helpers ---

func stringsToInterface(s []string) []any {
	result := make([]any, len(s))
	for i, v := range s {
		result[i] = v
	}
	return result
}

// normalizeModelID accepts either "owner/model" or "https://huggingface.co/owner/model"
// and returns the canonical "owner/model" form.
func normalizeModelID(input string) (string, error) {
	if strings.HasPrefix(input, "https://") || strings.HasPrefix(input, "http://") {
		u, err := url.Parse(input)
		if err != nil {
			return "", fmt.Errorf("invalid model URL: %w", err)
		}
		parts := strings.Split(strings.Trim(u.Path, "/"), "/")
		if len(parts) < 2 {
			return "", fmt.Errorf("model URL must contain owner/model path: %s", input)
		}
		return parts[0] + "/" + parts[1], nil
	}
	return input, nil
}

func parseHFTime(s string) *time.Time {
	if s == "" {
		return nil
	}
	t, err := time.Parse(time.RFC3339Nano, s)
	if err != nil {
		t, err = time.Parse("2006-01-02T15:04:05.000Z", s)
		if err != nil {
			return nil
		}
	}
	return &t
}
