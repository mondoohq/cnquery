// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers/digitalocean/connection"
)

// DigitalOcean Functions are backed by Apache OpenWhisk. godo lists
// namespaces and triggers but has no endpoint for the deployed functions
// (actions) themselves — those live behind each namespace's OpenWhisk
// API host. We reach them with the namespace's UUID/Key as HTTP Basic
// credentials, the same way `doctl serverless` does.

// owAction is the summary an OpenWhisk `list actions` call returns. The
// list omits the full exec block, so the runtime kind and web-export
// posture come from annotations rather than dedicated fields.
type owAction struct {
	Namespace   string       `json:"namespace"`
	Name        string       `json:"name"`
	Version     string       `json:"version"`
	Updated     int64        `json:"updated"`
	Annotations []owKeyValue `json:"annotations"`
	Limits      *owLimits    `json:"limits"`
}

type owKeyValue struct {
	Key   string      `json:"key"`
	Value interface{} `json:"value"`
}

type owLimits struct {
	Timeout     int `json:"timeout"`
	Memory      int `json:"memory"`
	Logs        int `json:"logs"`
	Concurrency int `json:"concurrency"`
}

func owAnnotation(anns []owKeyValue, key string) (interface{}, bool) {
	for _, a := range anns {
		if a.Key == key {
			return a.Value, true
		}
	}
	return nil, false
}

// owTruthy interprets an OpenWhisk annotation value as a boolean. The
// web-export and require-whisk-auth annotations may arrive as a bool, or
// as a string ("true", "raw", or an auth token), so any non-empty,
// non-false value counts as true.
func owTruthy(v interface{}) bool {
	switch t := v.(type) {
	case bool:
		return t
	case string:
		return t != "" && !strings.EqualFold(t, "false") && !strings.EqualFold(t, "no")
	case nil:
		return false
	default:
		return true
	}
}

func owString(v interface{}) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

// listOpenWhiskActions pages through a namespace's OpenWhisk actions.
func listOpenWhiskActions(ctx context.Context, apiHost, uuid, key string) ([]owAction, error) {
	base := strings.TrimRight(apiHost, "/") + "/api/v1/namespaces/_/actions"
	httpClient := &http.Client{Timeout: 30 * time.Second}

	const perPage = 200
	var all []owAction
	skip := 0
	for {
		url := fmt.Sprintf("%s?limit=%d&skip=%d", base, perPage, skip)
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return nil, err
		}
		req.SetBasicAuth(uuid, key)
		req.Header.Set("Accept", "application/json")

		resp, err := httpClient.Do(req)
		if err != nil {
			return nil, err
		}
		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, err
		}
		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("openwhisk list actions failed (%d): %s", resp.StatusCode, strings.TrimSpace(string(body)))
		}

		var page []owAction
		if err := json.Unmarshal(body, &page); err != nil {
			return nil, err
		}
		all = append(all, page...)
		if len(page) < perPage {
			break
		}
		skip += perPage
	}
	return all, nil
}

func (r *mqlDigitaloceanFunctionNamespace) functions() ([]interface{}, error) {
	conn := r.MqlRuntime.Connection.(*connection.DigitaloceanConnection)
	client := conn.Client()

	ctx := context.Background()

	// The namespace list does not include the API key needed to call the
	// OpenWhisk host, so fetch the namespace directly to obtain it.
	ns, _, err := client.Functions.GetNamespace(ctx, r.Uuid.Data)
	if err != nil {
		if isDoNotFound(err) {
			return []interface{}{}, nil
		}
		return nil, err
	}
	if ns.ApiHost == "" || ns.Key == "" {
		// Without an API host and key we can't reach the OpenWhisk host.
		return []interface{}{}, nil
	}

	actions, err := listOpenWhiskActions(ctx, ns.ApiHost, ns.UUID, ns.Key)
	if err != nil {
		return nil, err
	}

	all := make([]interface{}, 0, len(actions))
	for _, a := range actions {
		// The action's namespace field is "<nsUuid>" or "<nsUuid>/<package>".
		pkg := ""
		if idx := strings.IndexByte(a.Namespace, '/'); idx >= 0 {
			pkg = a.Namespace[idx+1:]
		}

		runtime := ""
		if v, ok := owAnnotation(a.Annotations, "exec"); ok {
			runtime = owString(v)
		}
		webExported := false
		if v, ok := owAnnotation(a.Annotations, "web-export"); ok {
			webExported = owTruthy(v)
		}
		requiresApiKey := false
		if v, ok := owAnnotation(a.Annotations, "require-whisk-auth"); ok {
			requiresApiKey = owTruthy(v)
		}

		var timeoutMs, memoryMb, logSizeMb, concurrency int64
		if a.Limits != nil {
			timeoutMs = int64(a.Limits.Timeout)
			memoryMb = int64(a.Limits.Memory)
			logSizeMb = int64(a.Limits.Logs)
			concurrency = int64(a.Limits.Concurrency)
		}

		var updatedAt *time.Time
		if a.Updated > 0 {
			t := time.UnixMilli(a.Updated)
			updatedAt = &t
		}

		res, err := CreateResource(r.MqlRuntime, "digitalocean.function.action", map[string]*llx.RawData{
			"__id":           llx.StringData("digitalocean.function.action/" + r.Uuid.Data + "/" + pkg + "/" + a.Name),
			"namespaceUuid":  llx.StringData(r.Uuid.Data),
			"name":           llx.StringData(a.Name),
			"package":        llx.StringData(pkg),
			"version":        llx.StringData(a.Version),
			"runtime":        llx.StringData(runtime),
			"webExported":    llx.BoolData(webExported),
			"requiresApiKey": llx.BoolData(requiresApiKey),
			"timeoutMs":      llx.IntData(timeoutMs),
			"memoryMb":       llx.IntData(memoryMb),
			"logSizeMb":      llx.IntData(logSizeMb),
			"concurrency":    llx.IntData(concurrency),
			"updatedAt":      llx.TimeDataPtr(updatedAt),
		})
		if err != nil {
			return nil, err
		}
		all = append(all, res)
	}
	return all, nil
}
