// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package providers

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewMondooProviderRegistry(t *testing.T) {
	tests := []struct {
		name     string
		opts     []MondooProviderRegistryOption
		expected string
	}{
		{
			name:     "default registry",
			opts:     nil,
			expected: "https://releases.mondoo.com/providers",
		},
		{
			name:     "custom base URL",
			opts:     []MondooProviderRegistryOption{WithBaseURL("https://my-registry.com/providers")},
			expected: "https://my-registry.com/providers",
		},
		{
			name:     "custom base URL with trailing slash",
			opts:     []MondooProviderRegistryOption{WithBaseURL("https://my-registry.com/providers/")},
			expected: "https://my-registry.com/providers/",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registry := NewMondooProviderRegistry(tt.opts...)
			assert.Equal(t, tt.expected, registry.BaseURL)
		})
	}
}

func TestMondooProviderRegistry_GetLatestVersion(t *testing.T) {
	// Create a mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/latest.json" {
			versions := ProviderVersions{
				Providers: []ProviderVersion{
					{Name: "aws", Version: "1.2.3"},
					{Name: "azure", Version: "2.4.6"},
					{Name: "gcp", Version: "3.1.4"},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			if err := json.NewEncoder(w).Encode(versions); err != nil {
				http.Error(w, "Failed to encode response", http.StatusInternalServerError)
			}
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	registry := NewMondooProviderRegistry(WithBaseURL(server.URL))

	tests := []struct {
		name     string
		provider string
		want     string
		wantErr  bool
	}{
		{
			name:     "existing provider",
			provider: "aws",
			want:     "1.2.3",
			wantErr:  false,
		},
		{
			name:     "another existing provider",
			provider: "azure",
			want:     "2.4.6",
			wantErr:  false,
		},
		{
			name:     "non-existing provider",
			provider: "nonexistent",
			want:     "",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			got, err := registry.GetLatestVersion(ctx, tt.provider)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "cannot determine latest version")
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestMondooProviderRegistry_GetLatestVersion_ServerError(t *testing.T) {
	// Test server returning an error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}))
	defer server.Close()

	registry := NewMondooProviderRegistry(WithBaseURL(server.URL))
	ctx := context.Background()

	_, err := registry.GetLatestVersion(ctx, "aws")
	assert.Error(t, err)
}

func TestMondooProviderRegistry_GetLatestVersion_InvalidJSON(t *testing.T) {
	// Test server returning invalid JSON
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/latest.json" {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte("invalid json")) // nolint:errcheck
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	registry := NewMondooProviderRegistry(WithBaseURL(server.URL))
	ctx := context.Background()

	_, err := registry.GetLatestVersion(ctx, "aws")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse response")
}

func TestMondooProviderRegistry_DownloadProvider(t *testing.T) {
	expectedContent := "fake-provider-content"

	// Create a mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check the expected path format: /aws/1.2.3/aws_1.2.3_linux_amd64.tar.xz
		expectedPath := "/aws/1.2.3/aws_1.2.3_linux_amd64.tar.xz"
		if r.URL.Path == expectedPath {
			w.Header().Set("Content-Type", "application/octet-stream")
			w.Write([]byte(expectedContent)) // nolint:errcheck
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	registry := NewMondooProviderRegistry(WithBaseURL(server.URL))
	ctx := context.Background()

	t.Run("successful download", func(t *testing.T) {
		reader, err := registry.DownloadProvider(ctx, "aws", "1.2.3", "linux", "amd64")
		require.NoError(t, err)
		defer reader.Close()

		content, err := io.ReadAll(reader)
		require.NoError(t, err)
		assert.Equal(t, expectedContent, string(content))
	})

	t.Run("provider not found", func(t *testing.T) {
		_, err := registry.DownloadProvider(ctx, "nonexistent", "1.0.0", "linux", "amd64")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot find provider")
	})
}
