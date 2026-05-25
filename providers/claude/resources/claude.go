// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/claude/connection"
)

func parseTime(s string) (time.Time, error) {
	if s == "" {
		return time.Time{}, nil
	}
	return time.Parse(time.RFC3339, s)
}

func conn(runtime *plugin.Runtime) *connection.ClaudeConnection {
	return runtime.Connection.(*connection.ClaudeConnection)
}

func requireAdmin(runtime *plugin.Runtime) (*connection.AdminClient, error) {
	c := conn(runtime)
	if c.AdminToken() == "" {
		return nil, fmt.Errorf("admin API key required: set --admin-token or ANTHROPIC_ADMIN_API_KEY")
	}
	return connection.NewAdminClient(c.AdminToken(), c.Host()), nil
}

func parseFamily(id string) string {
	for _, f := range []string{"opus", "sonnet", "haiku"} {
		if strings.Contains(id, f) {
			return f
		}
	}
	return ""
}

// claude

func (r *mqlClaude) id() (string, error) {
	return "claude", nil
}

func (r *mqlClaude) host() (string, error) {
	return conn(r.MqlRuntime).Host(), nil
}

func (r *mqlClaude) models() ([]interface{}, error) {
	c := conn(r.MqlRuntime)
	client := c.Client()

	pager := client.Models.ListAutoPaging(context.Background(), anthropic.ModelListParams{})

	var res []interface{}
	for pager.Next() {
		m := pager.Current()

		mqlModel, err := CreateResource(r.MqlRuntime, "claude.model", map[string]*llx.RawData{
			"__id":                       llx.StringData(m.ID),
			"id":                         llx.StringData(m.ID),
			"displayName":                llx.StringData(m.DisplayName),
			"vendor":                     llx.StringData("Anthropic"),
			"family":                     llx.StringData(parseFamily(m.ID)),
			"type":                       llx.StringData("model"),
			"createdAt":                  llx.TimeData(m.CreatedAt),
			"maxInputTokens":             llx.IntData(m.MaxInputTokens),
			"maxTokens":                  llx.IntData(m.MaxTokens),
			"batchSupported":             llx.BoolData(m.Capabilities.Batch.Supported),
			"citationsSupported":         llx.BoolData(m.Capabilities.Citations.Supported),
			"codeExecutionSupported":     llx.BoolData(m.Capabilities.CodeExecution.Supported),
			"imageInputSupported":        llx.BoolData(m.Capabilities.ImageInput.Supported),
			"pdfInputSupported":          llx.BoolData(m.Capabilities.PDFInput.Supported),
			"structuredOutputsSupported": llx.BoolData(m.Capabilities.StructuredOutputs.Supported),
			"thinkingSupported":          llx.BoolData(m.Capabilities.Thinking.Supported),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlModel)
	}
	if err := pager.Err(); err != nil {
		return nil, fmt.Errorf("listing models: %w", err)
	}

	return res, nil
}

func initClaudeModel(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 2 {
		return args, nil, nil
	}

	rawID, ok := args["id"]
	if !ok {
		return args, nil, nil
	}

	id, ok := rawID.Value.(string)
	if !ok || id == "" {
		return args, nil, nil
	}

	c := conn(runtime)
	client := c.Client()

	m, err := client.Models.Get(context.Background(), id, anthropic.ModelGetParams{})
	if err != nil {
		return nil, nil, fmt.Errorf("getting model %q: %w", id, err)
	}

	args["__id"] = llx.StringData(m.ID)
	args["id"] = llx.StringData(m.ID)
	args["displayName"] = llx.StringData(m.DisplayName)
	args["vendor"] = llx.StringData("Anthropic")
	args["family"] = llx.StringData(parseFamily(m.ID))
	args["type"] = llx.StringData("model")
	args["createdAt"] = llx.TimeData(m.CreatedAt)
	args["maxInputTokens"] = llx.IntData(m.MaxInputTokens)
	args["maxTokens"] = llx.IntData(m.MaxTokens)
	args["batchSupported"] = llx.BoolData(m.Capabilities.Batch.Supported)
	args["citationsSupported"] = llx.BoolData(m.Capabilities.Citations.Supported)
	args["codeExecutionSupported"] = llx.BoolData(m.Capabilities.CodeExecution.Supported)
	args["imageInputSupported"] = llx.BoolData(m.Capabilities.ImageInput.Supported)
	args["pdfInputSupported"] = llx.BoolData(m.Capabilities.PDFInput.Supported)
	args["structuredOutputsSupported"] = llx.BoolData(m.Capabilities.StructuredOutputs.Supported)
	args["thinkingSupported"] = llx.BoolData(m.Capabilities.Thinking.Supported)

	return args, nil, nil
}

func initClaudeOrganization(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 2 {
		return args, nil, nil
	}

	admin, err := requireAdmin(runtime)
	if err != nil {
		return nil, nil, err
	}

	org, err := admin.GetOrganization(context.Background())
	if err != nil {
		return nil, nil, err
	}

	args["__id"] = llx.StringData(org.ID)
	args["id"] = llx.StringData(org.ID)
	args["name"] = llx.StringData(org.Name)

	return args, nil, nil
}
