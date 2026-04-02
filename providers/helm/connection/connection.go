// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package connection

import (
	"errors"
	"os"
	"path/filepath"

	"go.mondoo.com/mql/v13/providers-sdk/v1/inventory"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
)

var _ plugin.Connection = (*HelmConnection)(nil)

type HelmConnection struct {
	plugin.Connection
	Conf   *inventory.Config
	asset  *inventory.Asset
	path   string
	charts []*chart.Chart
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

func (c *HelmConnection) Charts() []*chart.Chart {
	return c.charts
}

func (c *HelmConnection) Path() string {
	return c.path
}

// loadCharts loads Helm charts from a path. The path can be:
// - A chart directory (contains Chart.yaml)
// - A .tgz archive
// - A directory containing multiple chart directories
func loadCharts(chartPath string) ([]*chart.Chart, error) {
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
		return []*chart.Chart{c}, nil
	}

	// If the directory itself is a chart (has Chart.yaml), load it
	if _, err := os.Stat(filepath.Join(chartPath, "Chart.yaml")); err == nil {
		c, err := loader.LoadDir(chartPath)
		if err != nil {
			return nil, err
		}
		return []*chart.Chart{c}, nil
	}

	// Otherwise, scan for chart subdirectories
	var charts []*chart.Chart
	entries, err := os.ReadDir(chartPath)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		subPath := filepath.Join(chartPath, entry.Name())
		if _, err := os.Stat(filepath.Join(subPath, "Chart.yaml")); err == nil {
			c, err := loader.LoadDir(subPath)
			if err != nil {
				continue
			}
			charts = append(charts, c)
		}
	}

	return charts, nil
}
