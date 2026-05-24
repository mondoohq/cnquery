// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/ollama/ollama/api"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/ollama/connection"
	"go.mondoo.com/mql/v13/types"
)

func ollamaConn(runtime *plugin.Runtime) *connection.OllamaConnection {
	return runtime.Connection.(*connection.OllamaConnection)
}

func (r *mqlOllama) id() (string, error) {
	return "ollama", nil
}

func (r *mqlOllama) host() (string, error) {
	return ollamaConn(r.MqlRuntime).Host(), nil
}

func (r *mqlOllama) models() ([]interface{}, error) {
	conn := ollamaConn(r.MqlRuntime)
	client := conn.Client()

	resp, err := client.List(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to list models: %w", err)
	}

	res := make([]interface{}, 0, len(resp.Models))
	for _, m := range resp.Models {
		families := make([]interface{}, len(m.Details.Families))
		for i, f := range m.Details.Families {
			families[i] = f
		}

		mqlModel, err := CreateResource(r.MqlRuntime, "ollama.model", map[string]*llx.RawData{
			"__id":              llx.StringData(m.Digest),
			"name":              llx.StringData(m.Name),
			"model":             llx.StringData(m.Model),
			"modifiedAt":        llx.TimeData(m.ModifiedAt),
			"size":              llx.IntData(m.Size),
			"digest":            llx.StringData(m.Digest),
			"format":            llx.StringData(m.Details.Format),
			"family":            llx.StringData(m.Details.Family),
			"families":          llx.ArrayData(families, types.String),
			"parameterSize":     llx.StringData(m.Details.ParameterSize),
			"quantizationLevel": llx.StringData(m.Details.QuantizationLevel),
			"parentModel":       llx.StringData(m.Details.ParentModel),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlModel)
	}

	return res, nil
}

func (r *mqlOllama) runningModels() ([]interface{}, error) {
	conn := ollamaConn(r.MqlRuntime)
	client := conn.Client()

	resp, err := client.ListRunning(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to list running models: %w", err)
	}

	res := make([]interface{}, 0, len(resp.Models))
	for _, m := range resp.Models {
		mqlRunning, err := CreateResource(r.MqlRuntime, "ollama.runningModel", map[string]*llx.RawData{
			"__id":          llx.StringData("running/" + m.Digest),
			"name":          llx.StringData(m.Name),
			"expiresAt":     llx.TimeData(m.ExpiresAt),
			"sizeVram":      llx.IntData(m.SizeVRAM),
			"contextLength": llx.IntData(int64(m.ContextLength)),
		})
		if err != nil {
			return nil, err
		}

		rm := mqlRunning.(*mqlOllamaRunningModel)
		rm.cacheDigest = m.Digest
		rm.cacheDetails = m.Details

		res = append(res, mqlRunning)
	}

	return res, nil
}

type mqlOllamaModelInternal struct {
	fetched bool
	show    *api.ShowResponse
	lock    sync.Mutex
}

func (r *mqlOllamaModel) fetchShow() (*api.ShowResponse, error) {
	if r.fetched {
		return r.show, nil
	}
	r.lock.Lock()
	defer r.lock.Unlock()
	if r.fetched {
		return r.show, nil
	}
	conn := ollamaConn(r.MqlRuntime)
	show, err := conn.Client().Show(context.Background(), &api.ShowRequest{
		Model: r.GetName().Data,
	})
	if err != nil {
		return nil, err
	}
	r.show = show
	r.fetched = true
	return r.show, nil
}

func (r *mqlOllamaModel) id() (string, error) {
	return r.Digest.Data, nil
}

func (r *mqlOllamaModel) license() (string, error) {
	show, err := r.fetchShow()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(show.License), nil
}

func (r *mqlOllamaModel) modelfile() (string, error) {
	show, err := r.fetchShow()
	if err != nil {
		return "", err
	}
	return show.Modelfile, nil
}

func (r *mqlOllamaModel) system() (string, error) {
	show, err := r.fetchShow()
	if err != nil {
		return "", err
	}
	return show.System, nil
}

func (r *mqlOllamaModel) template() (string, error) {
	show, err := r.fetchShow()
	if err != nil {
		return "", err
	}
	return show.Template, nil
}

func (r *mqlOllamaModel) capabilities() ([]interface{}, error) {
	show, err := r.fetchShow()
	if err != nil {
		return nil, err
	}
	caps := make([]interface{}, len(show.Capabilities))
	for i, c := range show.Capabilities {
		caps[i] = string(c)
	}
	return caps, nil
}

func (r *mqlOllamaModel) info() (*mqlOllamaModelInfo, error) {
	show, err := r.fetchShow()
	if err != nil {
		return nil, err
	}

	mi := show.ModelInfo
	arch := getString(mi, "general.architecture")

	languages := getStringSlice(mi, "general.languages")
	tags := getStringSlice(mi, "general.tags")
	datasets := getStringSlice(mi, "general.datasets")

	res, err := CreateResource(r.MqlRuntime, "ollama.model.info", map[string]*llx.RawData{
		"__id":              llx.StringData(r.Digest.Data + "/info"),
		"architecture":      llx.StringData(arch),
		"basename":          llx.StringData(getString(mi, "general.basename")),
		"finetune":          llx.StringData(getString(mi, "general.finetune")),
		"sizeLabel":         llx.StringData(getString(mi, "general.size_label")),
		"license":           llx.StringData(getString(mi, "general.license")),
		"author":            llx.StringData(getString(mi, "general.author")),
		"description":       llx.StringData(getString(mi, "general.description")),
		"parameterCount":    llx.IntData(getInt(mi, "general.parameter_count")),
		"languages":         llx.ArrayData(languages, types.String),
		"tags":              llx.ArrayData(tags, types.String),
		"datasets":          llx.ArrayData(datasets, types.String),
		"contextLength":     llx.IntData(getArchInt(mi, arch, "context_length")),
		"embeddingLength":   llx.IntData(getArchInt(mi, arch, "embedding_length")),
		"blockCount":        llx.IntData(getArchInt(mi, arch, "block_count")),
		"feedForwardLength": llx.IntData(getArchInt(mi, arch, "feed_forward_length")),
		"headCount":         llx.IntData(getArchInt(mi, arch, "attention.head_count")),
		"headCountKv":       llx.IntData(getArchInt(mi, arch, "attention.head_count_kv")),
		"vocabSize":         llx.IntData(getArchInt(mi, arch, "vocab_size")),
		"expertCount":       llx.IntData(getArchInt(mi, arch, "expert_count")),
		"expertUsedCount":   llx.IntData(getArchInt(mi, arch, "expert_used_count")),
		"tokenizerModel":    llx.StringData(getString(mi, "tokenizer.ggml.model")),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlOllamaModelInfo), nil
}

func getString(m map[string]any, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func getInt(m map[string]any, key string) int64 {
	if v, ok := m[key]; ok {
		switch n := v.(type) {
		case float64:
			return int64(n)
		case int64:
			return n
		}
	}
	return 0
}

func getArchInt(m map[string]any, arch, key string) int64 {
	return getInt(m, arch+"."+key)
}

func getStringSlice(m map[string]any, key string) []interface{} {
	v, ok := m[key]
	if !ok || v == nil {
		return []interface{}{}
	}
	switch s := v.(type) {
	case []any:
		res := make([]interface{}, 0, len(s))
		for _, item := range s {
			if str, ok := item.(string); ok {
				res = append(res, str)
			}
		}
		return res
	}
	return []interface{}{}
}

func (r *mqlOllamaModelInfo) id() (string, error) {
	return r.Architecture.Data + "/" + r.Basename.Data + "/" + r.SizeLabel.Data, nil
}

type mqlOllamaRunningModelInternal struct {
	cacheDigest  string
	cacheDetails api.ModelDetails
}

func (r *mqlOllamaRunningModel) id() (string, error) {
	return "running/" + r.cacheDigest, nil
}

func (r *mqlOllamaRunningModel) model() (*mqlOllamaModel, error) {
	families := make([]interface{}, len(r.cacheDetails.Families))
	for i, f := range r.cacheDetails.Families {
		families[i] = f
	}

	res, err := NewResource(r.MqlRuntime, "ollama.model", map[string]*llx.RawData{
		"__id":              llx.StringData(r.cacheDigest),
		"name":              llx.StringData(r.GetName().Data),
		"model":             llx.StringData(r.GetName().Data),
		"modifiedAt":        llx.TimeData(time.Time{}),
		"size":              llx.IntData(0),
		"digest":            llx.StringData(r.cacheDigest),
		"format":            llx.StringData(r.cacheDetails.Format),
		"family":            llx.StringData(r.cacheDetails.Family),
		"families":          llx.ArrayData(families, types.String),
		"parameterSize":     llx.StringData(r.cacheDetails.ParameterSize),
		"quantizationLevel": llx.StringData(r.cacheDetails.QuantizationLevel),
		"parentModel":       llx.StringData(r.cacheDetails.ParentModel),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlOllamaModel), nil
}
