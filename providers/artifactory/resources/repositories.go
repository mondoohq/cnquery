// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
)

// repositoryRecord is what the repository list reports. It carries the fields
// that identify a repository and, for a remote one, the upstream it proxies.
type repositoryRecord struct {
	Key         string `json:"key"`
	Type        string `json:"type"`
	PackageType string `json:"packageType"`
	Description string `json:"description"`
	URL         string `json:"url"`
}

// repositoryDetailRecord is the configuration of a single repository. The list
// endpoint reports none of it, so it is read per repository and only when a
// query asks for one of these fields.
type repositoryDetailRecord struct {
	Key                         string   `json:"key"`
	RClass                      string   `json:"rclass"`
	PackageType                 string   `json:"packageType"`
	URL                         string   `json:"url"`
	Repositories                []string `json:"repositories"`
	XrayIndex                   *bool    `json:"xrayIndex"`
	BlackedOut                  *bool    `json:"blackedOut"`
	Offline                     *bool    `json:"offline"`
	IncludesPattern             string   `json:"includesPattern"`
	ExcludesPattern             string   `json:"excludesPattern"`
	RepoLayoutRef               string   `json:"repoLayoutRef"`
	AllowAnyHostAuth            *bool    `json:"allowAnyHostAuth"`
	ExternalDependenciesEnabled *bool    `json:"externalDependenciesEnabled"`
}

type mqlArtifactoryRepositoryInternal struct {
	// lock guards the single configuration read that backs every detail-only
	// field, so asking for several of them costs one call. Only a successful
	// read is kept, so a transient failure is retried rather than failing every
	// later field for the rest of the scan.
	lock   sync.Mutex
	detail *repositoryDetailRecord
	// detailLoaded is read on the fast path without the lock, so it is atomic.
	// A plain bool there would be an unsynchronized read against the write the
	// lock holder makes, which is a data race whatever the value happens to be.
	detailLoaded atomic.Bool
}

func (a *mqlArtifactory) repositories() ([]any, error) {
	conn := artifactoryConn(a.MqlRuntime)

	var records []repositoryRecord
	if err := conn.GetJSON(context.Background(), conn.ArtifactoryURL("/api/repositories"), &records); err != nil {
		return nil, err
	}

	res := make([]any, 0, len(records))
	for i := range records {
		repo, err := newArtifactoryRepository(a.MqlRuntime, &records[i])
		if err != nil {
			return nil, err
		}
		res = append(res, repo)
	}
	return res, nil
}

func newArtifactoryRepository(runtime *plugin.Runtime, rec *repositoryRecord) (*mqlArtifactoryRepository, error) {
	res, err := CreateResource(runtime, "artifactory.repository", map[string]*llx.RawData{
		"key":         llx.StringData(rec.Key),
		"type":        llx.StringData(strings.ToLower(rec.Type)),
		"packageType": llx.StringData(strings.ToLower(rec.PackageType)),
		"description": llx.StringData(rec.Description),
		"url":         llx.StringData(rec.URL),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlArtifactoryRepository), nil
}

// initArtifactoryRepository resolves a repository by its key, so that
// `artifactory.repository(key: "example-docker")` works without listing every
// repository first.
func initArtifactoryRepository(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 1 {
		return args, nil, nil
	}

	key := ""
	if data, ok := args["key"]; ok {
		if s, ok := data.Value.(string); ok {
			key = s
		}
	}
	if key == "" {
		return nil, nil, errors.New("artifactory.repository requires a key")
	}

	conn := artifactoryConn(runtime)

	var detail repositoryDetailRecord
	if err := conn.GetJSON(context.Background(), repositoryDetailURL(runtime, key), &detail); err != nil {
		return nil, nil, err
	}

	repo, err := newArtifactoryRepository(runtime, &repositoryRecord{
		Key:         key,
		Type:        detail.RClass,
		PackageType: detail.PackageType,
		URL:         detail.URL,
	})
	if err != nil {
		return nil, nil, err
	}

	// The configuration was just read, so record it rather than fetching it
	// again the first time a detail field is queried. The resource is already
	// registered with the runtime at this point, so the write takes the same
	// lock every reader takes.
	repo.lock.Lock()
	if !repo.detailLoaded.Load() {
		repo.detail = &detail
		repo.detailLoaded.Store(true)
	}
	repo.lock.Unlock()

	return args, repo, nil
}

func repositoryDetailURL(runtime *plugin.Runtime, key string) string {
	conn := artifactoryConn(runtime)
	return conn.ArtifactoryURL("/api/repositories/" + url.PathEscape(key))
}

func (r *mqlArtifactoryRepository) id() (string, error) {
	return "artifactory.repository/" + r.Key.Data, r.Key.Error
}

// config reads the repository's configuration once and shares it with every
// field that needs it.
func (r *mqlArtifactoryRepository) config() (*repositoryDetailRecord, error) {
	if r.detailLoaded.Load() {
		return r.detail, nil
	}
	r.lock.Lock()
	defer r.lock.Unlock()
	if r.detailLoaded.Load() {
		return r.detail, nil
	}

	conn := artifactoryConn(r.MqlRuntime)
	var detail repositoryDetailRecord
	if err := conn.GetJSON(context.Background(), repositoryDetailURL(r.MqlRuntime, r.Key.Data), &detail); err != nil {
		return nil, err
	}

	r.detail = &detail
	r.detailLoaded.Store(true)
	return r.detail, nil
}

func (r *mqlArtifactoryRepository) memberRepositories() ([]any, error) {
	detail, err := r.config()
	if err != nil {
		return nil, err
	}
	return strSliceToAny(detail.Repositories), nil
}

// memberRepositoryRefs resolves the members of a virtual repository against
// the instance's repository list, so the whole set costs one list call rather
// than one call per member.
func (r *mqlArtifactoryRepository) memberRepositoryRefs() ([]any, error) {
	members, err := r.memberRepositories()
	if err != nil {
		return nil, err
	}
	keys := make([]string, 0, len(members))
	for _, m := range members {
		if key, ok := m.(string); ok {
			keys = append(keys, key)
		}
	}
	return resolveRepositories(r.MqlRuntime, keys)
}

func (r *mqlArtifactoryRepository) xrayIndex() (bool, error) {
	return r.optionalBool(func(d *repositoryDetailRecord) *bool { return d.XrayIndex }, &r.XrayIndex)
}

func (r *mqlArtifactoryRepository) blackedOut() (bool, error) {
	return r.optionalBool(func(d *repositoryDetailRecord) *bool { return d.BlackedOut }, &r.BlackedOut)
}

func (r *mqlArtifactoryRepository) offline() (bool, error) {
	return r.optionalBool(func(d *repositoryDetailRecord) *bool { return d.Offline }, &r.Offline)
}

func (r *mqlArtifactoryRepository) allowAnyHostAuth() (bool, error) {
	return r.optionalBool(func(d *repositoryDetailRecord) *bool { return d.AllowAnyHostAuth }, &r.AllowAnyHostAuth)
}

func (r *mqlArtifactoryRepository) externalDependenciesEnabled() (bool, error) {
	return r.optionalBool(func(d *repositoryDetailRecord) *bool { return d.ExternalDependenciesEnabled }, &r.ExternalDependenciesEnabled)
}

// optionalBool reports a setting the instance omits for repository types it
// does not apply to. Such a setting stays null, so a repository that cannot
// carry it is distinguishable from one that has it turned off.
func (r *mqlArtifactoryRepository) optionalBool(pick func(*repositoryDetailRecord) *bool, field *plugin.TValue[bool]) (bool, error) {
	detail, err := r.config()
	if err != nil {
		return false, err
	}
	value := pick(detail)
	if value == nil {
		field.State = plugin.StateIsNull | plugin.StateIsSet
		return false, nil
	}
	return *value, nil
}

func (r *mqlArtifactoryRepository) includesPattern() (string, error) {
	detail, err := r.config()
	if err != nil {
		return "", err
	}
	return detail.IncludesPattern, nil
}

func (r *mqlArtifactoryRepository) excludesPattern() (string, error) {
	detail, err := r.config()
	if err != nil {
		return "", err
	}
	return detail.ExcludesPattern, nil
}

func (r *mqlArtifactoryRepository) repoLayoutRef() (string, error) {
	detail, err := r.config()
	if err != nil {
		return "", err
	}
	return detail.RepoLayoutRef, nil
}

// permissionTargets reports every target whose repository scope covers this
// repository, including the ones that reach it through a wildcard key.
func (r *mqlArtifactoryRepository) permissionTargets() ([]any, error) {
	targets, err := allPermissionTargets(r.MqlRuntime)
	if err != nil {
		return nil, err
	}

	res := []any{}
	for _, target := range targets {
		covers, err := target.coversRepository(r.Key.Data, r.Type.Data)
		if err != nil {
			return nil, err
		}
		if covers {
			res = append(res, target)
		}
	}
	return res, nil
}

// anonymousActions unions what every permission target gives the anonymous
// user over this repository. It answers the question a policy asks about a
// publishing repository: what can somebody with no credential do here.
func (r *mqlArtifactoryRepository) anonymousActions() ([]any, error) {
	targets, err := allPermissionTargets(r.MqlRuntime)
	if err != nil {
		return nil, err
	}

	seen := map[string]bool{}
	actions := []string{}
	for _, target := range targets {
		covers, err := target.coversRepository(r.Key.Data, r.Type.Data)
		if err != nil {
			return nil, err
		}
		if !covers {
			continue
		}
		scope := target.recordScope(scopeRepo)
		if scope == nil {
			continue
		}
		for _, action := range scope.Actions.Users[AnonymousUser] {
			if seen[action] {
				continue
			}
			seen[action] = true
			actions = append(actions, action)
		}
	}
	return strSliceToAny(actions), nil
}

// resolveRepositories turns repository keys into repository resources by
// scanning the instance's repository list, which is fetched once and cached on
// the root resource. A key that names no repository on the instance, such as
// one of the wildcard keys a permission target may use, is skipped.
func resolveRepositories(runtime *plugin.Runtime, keys []string) ([]any, error) {
	if len(keys) == 0 {
		return []any{}, nil
	}

	root, err := getArtifactory(runtime)
	if err != nil {
		return nil, err
	}
	repositories := root.GetRepositories()
	if repositories.Error != nil {
		return nil, repositories.Error
	}

	byKey := make(map[string]*mqlArtifactoryRepository, len(repositories.Data))
	for _, it := range repositories.Data {
		if repo, ok := it.(*mqlArtifactoryRepository); ok {
			byKey[repo.Key.Data] = repo
		}
	}

	res := []any{}
	for _, key := range keys {
		if repo, ok := byKey[key]; ok {
			res = append(res, repo)
		}
	}
	return res, nil
}
