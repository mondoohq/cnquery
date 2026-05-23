// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package connection

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/mql/v13/providers-sdk/v1/inventory"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
)

var _ plugin.Connection = (*HelmConnection)(nil)

// LoadedChart pairs a parsed chart with the filesystem path it was
// loaded from so the resource layer can distinguish two charts that
// happen to share name + version (e.g., feature-branch forks of the
// same chart in a multi-chart directory).
type LoadedChart struct {
	Path  string
	Chart *chart.Chart
}

type HelmConnection struct {
	plugin.Connection
	Conf   *inventory.Config
	asset  *inventory.Asset
	path   string
	charts []LoadedChart
}

func NewHelmConnection(id uint32, asset *inventory.Asset, conf *inventory.Config) (*HelmConnection, error) {
	conn := &HelmConnection{
		Connection: plugin.NewConnection(id, asset),
		Conf:       conf,
		asset:      asset,
	}

	if len(asset.Connections) == 0 {
		return nil, errors.New("no connection configuration provided")
	}
	cc := asset.Connections[0]
	chartPath := filepath.Clean(cc.Options["path"])
	conn.path = chartPath

	charts, err := loadCharts(chartPath)
	if err != nil {
		return nil, err
	}
	if len(charts) == 0 {
		return nil, errors.New("no Helm charts found at " + chartPath)
	}
	conn.charts = charts

	return conn, nil
}

func (c *HelmConnection) Name() string {
	return "helm"
}

func (c *HelmConnection) Asset() *inventory.Asset {
	return c.asset
}

// Charts returns the loaded charts paired with the filesystem path
// each was loaded from. Path uniqueness is what makes the resource
// layer's __id collision-free across same-name charts.
func (c *HelmConnection) Charts() []LoadedChart {
	return c.charts
}

func (c *HelmConnection) Path() string {
	return c.path
}

// loadCharts loads Helm charts from a path. The path can be:
// - A chart directory (contains Chart.yaml)
// - A .tgz archive
// - A directory containing multiple chart directories
func loadCharts(chartPath string) ([]LoadedChart, error) {
	fi, err := os.Stat(chartPath)
	if err != nil {
		return nil, err
	}

	// If it's a file (.tgz), load it directly
	if !fi.IsDir() {
		c, err := loader.LoadFile(chartPath)
		if err != nil {
			return nil, err
		}
		return []LoadedChart{{Path: chartPath, Chart: c}}, nil
	}

	// If the directory itself is a chart (has Chart.yaml), load it
	if _, err := os.Stat(filepath.Join(chartPath, "Chart.yaml")); err == nil {
		c, err := loader.LoadDir(chartPath)
		if err != nil {
			return nil, err
		}
		return []LoadedChart{{Path: chartPath, Chart: c}}, nil
	}

	// Otherwise, scan for chart subdirectories
	var charts []LoadedChart
	entries, err := os.ReadDir(chartPath)
	if err != nil {
		return nil, err
	}

	// Track subdirectories that have a Chart.yaml but fail to load —
	// a static-analysis tool should surface malformed charts rather
	// than silently pretend they don't exist.
	var skipped []string
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		subPath := filepath.Join(chartPath, entry.Name())
		if _, err := os.Stat(filepath.Join(subPath, "Chart.yaml")); err != nil {
			continue
		}
		c, err := loader.LoadDir(subPath)
		if err != nil {
			skipped = append(skipped, subPath)
			log.Warn().Err(err).Str("path", subPath).Msg("failed to load Helm chart; skipping")
			continue
		}
		charts = append(charts, LoadedChart{Path: subPath, Chart: c})
	}

	// If no chart loaded successfully but we found at least one
	// malformed candidate, propagate that as the connection error so
	// the caller sees "your chart didn't parse" instead of "no charts
	// found."
	if len(charts) == 0 && len(skipped) > 0 {
		return nil, errors.New("found chart subdirectories but none loaded successfully: " + skipped[0])
	}

	return charts, nil
}
