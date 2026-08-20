// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/mql/providers-sdk/v1/plugin"
)

// TestMcpConnectionConfig verifies that discovery emits how to start the server
// (protocol + target) and, crucially, never the server's env/secrets — those are
// supplied explicitly at connect time via the AI provider's `--env`. See ADR 030.
func TestMcpConnectionConfig(t *testing.T) {
	t.Run("stdio emits protocol+target and no env", func(t *testing.T) {
		srv := &mqlClaudeCodeMcpServer{
			Type:    plugin.TValue[string]{Data: "stdio", State: plugin.StateIsSet},
			Command: plugin.TValue[string]{Data: "npx", State: plugin.StateIsSet},
			Args:    plugin.TValue[[]any]{Data: []any{"-y", "@scope/server", "run"}, State: plugin.StateIsSet},
		}

		conf := mcpConnectionConfig(srv)
		require.NotNil(t, conf)
		assert.Equal(t, "mcp", conf.Type)
		assert.Equal(t, "stdio", conf.Options["protocol"])
		assert.Equal(t, "npx -y @scope/server run", conf.Options["target"])
		_, hasEnv := conf.Options["env"]
		assert.False(t, hasEnv, "discovery must not carry env/secrets")
	})

	t.Run("http server uses url target", func(t *testing.T) {
		srv := &mqlClaudeCodeMcpServer{
			Type: plugin.TValue[string]{Data: "http", State: plugin.StateIsSet},
			Url:  plugin.TValue[string]{Data: "https://mcp.example.com/mcp", State: plugin.StateIsSet},
		}
		conf := mcpConnectionConfig(srv)
		require.NotNil(t, conf)
		assert.Equal(t, "https", conf.Options["protocol"])
		assert.Equal(t, "https://mcp.example.com/mcp", conf.Options["target"])
	})

	t.Run("sse server is a remote transport, not dropped", func(t *testing.T) {
		// Cursor remote servers declare `type: "sse"`; it is URL-based like http,
		// so discovery must emit it rather than silently drop it. See ADR 030.
		srv := &mqlClaudeCodeMcpServer{
			Type: plugin.TValue[string]{Data: "sse", State: plugin.StateIsSet},
			Url:  plugin.TValue[string]{Data: "https://mcp.example.com/sse", State: plugin.StateIsSet},
		}
		conf := mcpConnectionConfig(srv)
		require.NotNil(t, conf)
		assert.Equal(t, "https", conf.Options["protocol"])
		assert.Equal(t, "https://mcp.example.com/sse", conf.Options["target"])
	})

	t.Run("transport type is matched case-insensitively", func(t *testing.T) {
		// A config that spells the type in any case must still be recognized as
		// stdio rather than falling through to the URL branch and being dropped.
		srv := &mqlClaudeCodeMcpServer{
			Type:    plugin.TValue[string]{Data: "STDIO", State: plugin.StateIsSet},
			Command: plugin.TValue[string]{Data: "npx", State: plugin.StateIsSet},
		}
		conf := mcpConnectionConfig(srv)
		require.NotNil(t, conf)
		assert.Equal(t, "stdio", conf.Options["protocol"])
		assert.Equal(t, "npx", conf.Options["target"])
	})
}
