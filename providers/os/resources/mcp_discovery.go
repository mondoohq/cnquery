// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"strings"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/inventory"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
)

// mcpClientResources are the AI tool resources that declare MCP servers. Each
// exposes a `mcpServers()` accessor; discovery reuses those accessors (rather
// than re-parsing configs) so the discovered assets stay in lockstep with the
// `running` field on each server resource.
var mcpClientResources = []string{
	"claude.code",
	"openai.codex",
	"cursor",
	"github.copilot",
	"gemini",
	"windsurf",
}

// mcpClientLike is satisfied by every AI tool resource that lists MCP servers.
type mcpClientLike interface {
	GetMcpServers() *plugin.TValue[[]any]
}

// mcpServerLike is satisfied by every *.mcpServer resource. The field names are
// identical across tools, so one interface covers all six concrete types.
type mcpServerLike interface {
	plugin.Resource
	GetName() *plugin.TValue[string]
	GetType() *plugin.TValue[string]
	GetCommand() *plugin.TValue[string]
	GetArgs() *plugin.TValue[[]any]
	GetUrl() *plugin.TValue[string]
}

// DiscoverMCPServerAssets enumerates the MCP servers configured on the host and
// returns one asset per server. Each asset carries:
//   - a `mcp` connection Config (protocol + target) with everything the AI
//     provider's MCP connector needs to connect. The os provider never connects
//     to the MCP server itself.
//   - a resource-anchored relationship back to the host: the anchor is the
//     mcpServer resource's own (type, id), matching the `running` field. See ADR 030.
func DiscoverMCPServerAssets(runtime *plugin.Runtime, host *inventory.Asset) ([]*inventory.Asset, error) {
	hostRef := &inventory.Asset{
		Id:          host.GetId(),
		Mrn:         host.GetMrn(),
		PlatformIds: host.GetPlatformIds(),
	}

	var assets []*inventory.Asset
	for _, clientName := range mcpClientResources {
		res, err := NewResource(runtime, clientName, map[string]*llx.RawData{})
		if err != nil {
			log.Debug().Err(err).Str("client", clientName).Msg("mcp discovery: skipping client")
			continue
		}
		client, ok := res.(mcpClientLike)
		if !ok {
			continue
		}
		servers := client.GetMcpServers()
		if servers.Error != nil {
			log.Debug().Err(servers.Error).Str("client", clientName).Msg("mcp discovery: failed to list servers")
			continue
		}

		for _, raw := range servers.Data {
			srv, ok := raw.(mcpServerLike)
			if !ok {
				continue
			}
			conf := mcpConnectionConfig(srv)
			if conf == nil {
				// no usable connection info (neither command nor url)
				continue
			}
			// No PlatformIds here: identity is assigned by the AI provider's
			// Connect() at connect time. The relationship back to the host is
			// resource-anchored on (type, id), not platform-ID-based. See ADR 030.
			assets = append(assets, &inventory.Asset{
				Name:        srv.GetName().Data,
				State:       inventory.State_STATE_ONLINE,
				Connections: []*inventory.Config{conf},
				Relationships: []*inventory.AssetRelationship{
					{
						Asset:        hostRef,
						ResourceType: srv.MqlName(),
						ResourceId:   srv.MqlID(),
					},
				},
			})
		}
	}
	return assets, nil
}

// mcpConnectionConfig builds the `mcp` connection Config the AI provider needs.
// It reads `Options["protocol"]` ("stdio"|"http"|"https") and `Options["target"]`
// (a shell command for stdio, a URL for http/https). Returns nil when the server
// exposes neither a command nor a url.
func mcpConnectionConfig(srv mcpServerLike) *inventory.Config {
	command := srv.GetCommand().Data
	url := srv.GetUrl().Data

	// stdio is the only local transport. Every other transport (http, https,
	// sse, streamable-http, ...) is a remote, URL-based one, so we classify by
	// whether a URL is present rather than enumerating names, so a config with a
	// transport we don't recognize by name is not silently dropped.
	if deriveMcpTransport(srv.GetType().Data, command, url) == mcpTransportStdio {
		if command == "" {
			return nil
		}
		args := srv.GetArgs()
		if args.Error != nil {
			// Args failed to resolve: emit the bare command rather than silently
			// dropping arguments, and leave a trace so a broken target is explained.
			log.Warn().Err(args.Error).Str("server", srv.GetName().Data).
				Msg("mcp discovery: failed to read server args, emitting command without arguments")
		}
		parts := make([]string, 0, 1+len(args.Data))
		parts = append(parts, command)
		parts = append(parts, anySliceToStrings(args.Data)...)
		// We deliberately do NOT carry the server's env (secrets). Discovery
		// emits only how to start the server; secrets are supplied explicitly at
		// connect time via the AI provider's `--env` flag, so they never enter
		// the inventory. See ADR 030.
		return &inventory.Config{
			Type: "mcp",
			Options: map[string]string{
				"protocol": "stdio",
				"target":   shellQuoteJoin(parts),
			},
		}
	}

	// Remote transport: needs a URL to reach the server.
	if url == "" {
		return nil
	}
	protocol := "http"
	if strings.HasPrefix(strings.ToLower(url), "https://") {
		protocol = "https"
	}
	return &inventory.Config{
		Type: "mcp",
		Options: map[string]string{
			"protocol": protocol,
			"target":   url,
		},
	}
}

func anySliceToStrings(in []any) []string {
	out := make([]string, 0, len(in))
	for _, v := range in {
		if s, ok := v.(string); ok {
			out = append(out, s)
		}
	}
	return out
}

// shellQuoteJoin joins a command and its args into a single string that a POSIX
// shell-word splitter (the AI provider uses shellquote.Split) reparses into the
// original tokens. Tokens with shell-special characters are single-quoted.
func shellQuoteJoin(parts []string) string {
	quoted := make([]string, len(parts))
	for i, p := range parts {
		quoted[i] = shellQuote(p)
	}
	return strings.Join(quoted, " ")
}

func shellQuote(s string) string {
	if s == "" {
		return "''"
	}
	if !strings.ContainsAny(s, " \t\n\r'\"\\$`&|;<>()*?[]#~=%!{}") {
		return s
	}
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}
