// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package connection

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
	"gopkg.in/yaml.v3"
	"helm.sh/helm/v3/pkg/registry"
)

// resolveChartRef turns a possibly-remote chart reference into a local
// filesystem path that loadCharts can read:
//
//   - oci://registry/repo[:tag]   — pulled from the OCI registry
//   - http(s)://host/chart.tgz    — downloaded directly
//   - chartName with --repo URL   — resolved against the repo index, then pulled
//   - anything else               — treated as a local path (cleaned)
//
// For remote refs it downloads the chart archive into a temp directory and
// returns a cleanup func that removes it; loadCharts reads the archive fully
// into memory, so the caller can clean up immediately afterward. cleanup is
// nil for local paths.
//
// Fetching deliberately avoids helm's pkg/getter and pkg/repo, which pull in
// the kube/kubectl stack (incompatible with our pinned k8s.io/api and useless
// for static analysis). OCI uses pkg/registry directly; HTTP repositories and
// index resolution use net/http.
func resolveChartRef(rawPath string, opts map[string]string) (localPath string, cleanup func(), err error) {
	version := opts[OptionVersion]
	username := opts[OptionUsername]
	password := opts[OptionPassword]
	repoURL := opts[OptionRepo]

	isOCI := strings.HasPrefix(rawPath, registry.OCIScheme+"://")
	isHTTP := strings.HasPrefix(rawPath, "http://") || strings.HasPrefix(rawPath, "https://")

	// Purely local chart: no fetching.
	if !isOCI && !isHTTP && repoURL == "" {
		return filepath.Clean(rawPath), nil, nil
	}

	data, err := fetchChartArchive(rawPath, repoURL, version, username, password, isOCI)
	if err != nil {
		return "", nil, err
	}

	dest, err := os.MkdirTemp("", "mql-helm-chart-*")
	if err != nil {
		return "", nil, err
	}
	cleanup = func() {
		if rmErr := os.RemoveAll(dest); rmErr != nil {
			log.Warn().Err(rmErr).Str("dir", dest).Msg("failed to clean up helm chart download dir")
		}
	}

	archive := filepath.Join(dest, "chart.tgz")
	if err := os.WriteFile(archive, data, 0o600); err != nil {
		cleanup()
		return "", nil, err
	}
	return archive, cleanup, nil
}

// fetchChartArchive downloads the raw .tgz bytes of a remote chart.
func fetchChartArchive(rawPath, repoURL, version, username, password string, isOCI bool) ([]byte, error) {
	if isOCI {
		client, err := registry.NewClient()
		if err != nil {
			return nil, err
		}
		ref := strings.TrimPrefix(rawPath, registry.OCIScheme+"://")
		// Append the requested version as a tag when the ref isn't already
		// pinned by tag or digest.
		if version != "" && !strings.ContainsAny(lastPathSegment(ref), ":@") {
			ref += ":" + version
		}
		result, err := client.Pull(ref, registry.PullOptWithChart(true))
		if err != nil {
			return nil, err
		}
		return result.Chart.Data, nil
	}

	chartURL := rawPath
	if repoURL != "" {
		// Resolve "chartName" against the repository's index.yaml.
		resolved, err := resolveChartInRepo(repoURL, rawPath, version, username, password)
		if err != nil {
			return nil, err
		}
		chartURL = resolved
	}

	return httpGetBytes(chartURL, username, password)
}

// repoIndex is the subset of a Helm repository index.yaml we need to map a
// chart name + version to a downloadable archive URL.
type repoIndex struct {
	Entries map[string][]struct {
		Version string   `yaml:"version"`
		URLs    []string `yaml:"urls"`
	} `yaml:"entries"`
}

// resolveChartInRepo downloads <repoURL>/index.yaml and resolves a chart
// name (and optional version) to a concrete archive URL. With no version it
// picks the first entry, which Helm publishes newest-first.
func resolveChartInRepo(repoURL, chartName, version, username, password string) (string, error) {
	indexURL := strings.TrimSuffix(repoURL, "/") + "/index.yaml"
	data, err := httpGetBytes(indexURL, username, password)
	if err != nil {
		return "", fmt.Errorf("failed to fetch repository index %q: %w", indexURL, err)
	}

	var index repoIndex
	if err := yaml.Unmarshal(data, &index); err != nil {
		return "", fmt.Errorf("failed to parse repository index %q: %w", indexURL, err)
	}

	versions, ok := index.Entries[chartName]
	if !ok || len(versions) == 0 {
		return "", fmt.Errorf("chart %q not found in repository %q", chartName, repoURL)
	}

	chosen := versions[0]
	if version != "" {
		found := false
		for _, v := range versions {
			if v.Version == version {
				chosen = v
				found = true
				break
			}
		}
		if !found {
			return "", fmt.Errorf("chart %q version %q not found in repository %q", chartName, version, repoURL)
		}
	}
	if len(chosen.URLs) == 0 {
		return "", fmt.Errorf("chart %q has no download URL in repository %q", chartName, repoURL)
	}

	// Chart URLs may be relative to the repository URL.
	chartURL := chosen.URLs[0]
	if !strings.Contains(chartURL, "://") {
		base, err := url.Parse(strings.TrimSuffix(repoURL, "/") + "/")
		if err != nil {
			return "", err
		}
		ref, err := url.Parse(chartURL)
		if err != nil {
			return "", err
		}
		chartURL = base.ResolveReference(ref).String()
	}
	return chartURL, nil
}

// chartHTTPClient bounds remote chart/index fetches so a slow or stalled
// repository can't hang the connection indefinitely.
var chartHTTPClient = &http.Client{Timeout: 60 * time.Second}

// maxChartDownloadSize caps a downloaded chart archive or repository index
// (128 MiB) so a misbehaving repo can't exhaust memory.
const maxChartDownloadSize = 128 << 20

// httpGetBytes fetches a URL with optional basic auth, a request timeout,
// and a bounded read.
func httpGetBytes(rawURL, username, password string) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	if username != "" || password != "" {
		req.SetBasicAuth(username, password)
	}
	resp, err := chartHTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch %q: status %d", rawURL, resp.StatusCode)
	}
	return io.ReadAll(io.LimitReader(resp.Body, maxChartDownloadSize))
}

// lastPathSegment returns the final path segment of a registry reference,
// where a tag (":") or digest ("@") would appear.
func lastPathSegment(ref string) string {
	if i := strings.LastIndex(ref, "/"); i != -1 {
		return ref[i+1:]
	}
	return ref
}
