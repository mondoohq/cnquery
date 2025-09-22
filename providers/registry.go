// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/cockroachdb/errors"
	"github.com/rs/zerolog/log"
)

var DefaultProviderRegistryURL = "https://releases.mondoo.com/providers"

var registry ProviderRegistry = NewMondooProviderRegistry()

// SetProviderRegistry allows setting a custom provider registry implementation.
// It must be called before any provider installation occurs.
func SetProviderRegistry(r ProviderRegistry) {
	registry = r
}

// ProviderRegistry defines the interface for provider registries that can
// fetch provider versions and download provider packages.
type ProviderRegistry interface {
	// GetLatestVersion returns the latest version available for the given provider name
	GetLatestVersion(ctx context.Context, name string) (string, error)

	// DownloadProvider downloads a provider package and returns a ReadCloser for the content
	DownloadProvider(ctx context.Context, name, version, os, arch string) (io.ReadCloser, error)
}

// MondooProviderRegistry implements ProviderRegistry for Mondoo's provider registry
type MondooProviderRegistry struct {
	BaseURL string
}

// MondooProviderRegistryOption defines a function type for configuring MondooProviderRegistry
type MondooProviderRegistryOption func(*MondooProviderRegistry)

// WithBaseURL sets the base URL for the provider registry
func WithBaseURL(baseURL string) MondooProviderRegistryOption {
	return func(r *MondooProviderRegistry) {
		r.BaseURL = baseURL
	}
}

// NewMondooProviderRegistry creates a new MondooProviderRegistry with the given options.
// By default, it uses "https://releases.mondoo.com/providers" as the base URL.
func NewMondooProviderRegistry(opts ...MondooProviderRegistryOption) *MondooProviderRegistry {
	r := &MondooProviderRegistry{
		BaseURL: DefaultProviderRegistryURL,
	}

	for _, opt := range opts {
		opt(r)
	}

	return r
}

func LatestVersion(ctx context.Context, name string) (string, error) {
	return registry.GetLatestVersion(ctx, name)
}

// GetLatestVersion fetches the latest version for the given provider name
func (r *MondooProviderRegistry) GetLatestVersion(ctx context.Context, name string) (string, error) {
	client, err := httpClientWithRetry()
	if err != nil {
		return "", err
	}

	latestURL, err := url.JoinPath(r.BaseURL, "latest.json")
	if err != nil {
		return "", errors.Wrap(err, "failed to construct latest version URL")
	}

	res, err := client.Get(latestURL)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()

	data, err := io.ReadAll(res.Body)
	if err != nil {
		log.Debug().Err(err).Msg("reading latest.json failed")
		return "", errors.New("failed to read response from upstream provider versions")
	}

	var upstreamVersions ProviderVersions
	err = json.Unmarshal(data, &upstreamVersions)
	if err != nil {
		log.Debug().Err(err).Msg("parsing latest.json failed")
		return "", errors.New("failed to parse response from upstream provider versions")
	}

	var latestVersion string
	for i := range upstreamVersions.Providers {
		if upstreamVersions.Providers[i].Name == name {
			latestVersion = upstreamVersions.Providers[i].Version
			break
		}
	}

	if latestVersion == "" {
		return "", errors.New("cannot determine latest version of provider '" + name + "'")
	}
	return latestVersion, nil
}

// DownloadProvider downloads a provider package from the registry
func (r *MondooProviderRegistry) DownloadProvider(ctx context.Context, name, version, os, arch string) (io.ReadCloser, error) {
	// Build the filename using the same pattern as the original
	filename := fmt.Sprintf("%s_%s_%s_%s.tar.xz", name, version, os, arch)

	// Construct the download URL using url.JoinPath for robust path handling
	downloadURL, err := url.JoinPath(r.BaseURL, name, version, filename)
	if err != nil {
		return nil, errors.Wrap(err, "failed to construct download URL")
	}

	log.Debug().Str("url", downloadURL).Msg("downloading provider from URL")

	client, err := httpClientWithRetry()
	if err != nil {
		return nil, err
	}

	res, err := client.Get(downloadURL)
	if err != nil {
		return nil, errors.Wrap(err, "failed to download "+name+"-"+version)
	}

	if res.StatusCode == http.StatusNotFound {
		return nil, errors.New("cannot find provider " + name + "-" + version + " under url " + downloadURL)
	} else if res.StatusCode != http.StatusOK {
		log.Debug().Str("url", downloadURL).Int("status", res.StatusCode).Msg("failed to download from URL (status code)")
		res.Body.Close()
		return nil, errors.New("failed to download " + name + "-" + version + ", received status code: " + res.Status)
	}

	return res.Body, nil
}
