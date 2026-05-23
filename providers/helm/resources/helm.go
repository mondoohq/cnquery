// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"sync"
	"sync/atomic"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
	"go.mondoo.com/mql/v13/providers/helm/connection"
	"go.mondoo.com/mql/v13/types"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chartutil"
	"helm.sh/helm/v3/pkg/engine"
)

func (r *mqlHelm) id() (string, error) {
	return "helm", nil
}

func (r *mqlHelm) charts() ([]any, error) {
	conn := r.MqlRuntime.Connection.(*connection.HelmConnection)
	charts := conn.Charts()

	var mqlCharts []any
	for _, c := range charts {
		mqlChart, err := newMqlHelmChart(r.MqlRuntime, c)
		if err != nil {
			return nil, err
		}
		mqlCharts = append(mqlCharts, mqlChart)
	}
	return mqlCharts, nil
}

type mqlHelmChartInternal struct {
	chartObj    *chart.Chart
	rendered    map[string]string
	renderedErr error
	lock        sync.Mutex
	fetched     atomic.Bool
}

func newMqlHelmChart(runtime *plugin.Runtime, c *chart.Chart) (*mqlHelmChart, error) {
	meta := c.Metadata

	keywords := convert.SliceAnyToInterface(meta.Keywords)
	sources := convert.SliceAnyToInterface(meta.Sources)

	args := map[string]*llx.RawData{
		"__id":        llx.StringData("helm.chart:" + meta.Name + ":" + meta.Version),
		"name":        llx.StringData(meta.Name),
		"version":     llx.StringData(meta.Version),
		"apiVersion":  llx.StringData(meta.APIVersion),
		"type":        llx.StringData(string(meta.Type)),
		"appVersion":  llx.StringData(meta.AppVersion),
		"description": llx.StringData(meta.Description),
		"keywords":    llx.ArrayData(keywords, types.String),
		"home":        llx.StringData(meta.Home),
		"sources":     llx.ArrayData(sources, types.String),
		"icon":        llx.StringData(meta.Icon),
		"deprecated":  llx.BoolData(meta.Deprecated),
		"kubeVersion": llx.StringData(meta.KubeVersion),
	}
	if meta.Annotations == nil {
		args["annotations"] = llx.NilData
	} else {
		annotations := make(map[string]any, len(meta.Annotations))
		for k, v := range meta.Annotations {
			annotations[k] = v
		}
		args["annotations"] = llx.MapData(annotations, types.String)
	}

	res, err := CreateResource(runtime, "helm.chart", args)
	if err != nil {
		return nil, err
	}
	mqlChart := res.(*mqlHelmChart)
	mqlChart.chartObj = c
	return mqlChart, nil
}

func (c *mqlHelmChart) id() (string, error) {
	return "helm.chart:" + c.Name.Data + ":" + c.Version.Data, nil
}

func (c *mqlHelmChart) fetchRendered() (map[string]string, error) {
	if c.fetched.Load() {
		return c.rendered, c.renderedErr
	}
	c.lock.Lock()
	defer c.lock.Unlock()
	if c.fetched.Load() {
		return c.rendered, c.renderedErr
	}

	e := engine.Engine{Strict: false}
	options := chartutil.ReleaseOptions{
		Name:      c.chartObj.Name(),
		Namespace: "default",
		IsInstall: true,
	}
	vals, err := chartutil.ToRenderValues(c.chartObj, c.chartObj.Values, options, nil)
	if err != nil {
		c.renderedErr = err
		c.fetched.Store(true)
		return nil, err
	}
	c.rendered, c.renderedErr = e.Render(c.chartObj, vals)
	c.fetched.Store(true)
	return c.rendered, c.renderedErr
}

func (c *mqlHelmChart) dependencies() ([]any, error) {
	deps := c.chartObj.Metadata.Dependencies
	var mqlDeps []any
	for _, dep := range deps {
		mqlDep, err := newMqlHelmDependency(c.MqlRuntime, c.chartObj.Name(), dep)
		if err != nil {
			return nil, err
		}
		mqlDeps = append(mqlDeps, mqlDep)
	}
	return mqlDeps, nil
}

func (c *mqlHelmChart) maintainers() ([]any, error) {
	maintainers := c.chartObj.Metadata.Maintainers
	var mqlMaintainers []any
	for _, m := range maintainers {
		mqlM, err := newMqlHelmMaintainer(c.MqlRuntime, c.chartObj.Name(), m)
		if err != nil {
			return nil, err
		}
		mqlMaintainers = append(mqlMaintainers, mqlM)
	}
	return mqlMaintainers, nil
}

func (c *mqlHelmChart) templates() ([]any, error) {
	rendered, err := c.fetchRendered()
	if err != nil {
		log.Warn().Err(err).Str("chart", c.chartObj.Name()).Msg("failed to render helm chart templates")
	}

	var mqlTemplates []any
	for _, t := range c.chartObj.Templates {
		renderedContent := ""
		if rendered != nil {
			// Helm uses "chartName/templateName" as the key
			renderedContent = rendered[c.chartObj.Name()+"/"+t.Name]
		}
		mqlT, err := newMqlHelmTemplate(c.MqlRuntime, c.chartObj.Name(), t, renderedContent)
		if err != nil {
			return nil, err
		}
		mqlTemplates = append(mqlTemplates, mqlT)
	}
	return mqlTemplates, nil
}

func (c *mqlHelmChart) values() (any, error) {
	dict, err := convert.JsonToDict(c.chartObj.Values)
	if err != nil {
		return nil, err
	}
	return dict, nil
}

func (c *mqlHelmChart) resources() ([]any, error) {
	rendered, err := c.fetchRendered()
	if err != nil {
		log.Warn().Err(err).Str("chart", c.chartObj.Name()).Msg("failed to render helm chart templates, returning empty resources")
		return []any{}, nil
	}

	var allResources []any
	for templateKey, content := range rendered {
		resources, err := parseK8sResources(c.MqlRuntime, templateKey, content)
		if err != nil {
			continue
		}
		allResources = append(allResources, resources...)
	}
	return allResources, nil
}

func (c *mqlHelmChart) files() ([]any, error) {
	var mqlFiles []any
	for _, f := range c.chartObj.Files {
		mqlF, err := newMqlHelmFile(c.MqlRuntime, c.chartObj.Name(), f)
		if err != nil {
			return nil, err
		}
		mqlFiles = append(mqlFiles, mqlF)
	}
	return mqlFiles, nil
}

func newMqlHelmDependency(runtime *plugin.Runtime, chartName string, dep *chart.Dependency) (*mqlHelmDependency, error) {
	tags := convert.SliceAnyToInterface(dep.Tags)

	res, err := CreateResource(runtime, "helm.dependency", map[string]*llx.RawData{
		"__id":       llx.StringData("helm.dependency:" + chartName + ":" + dep.Name),
		"name":       llx.StringData(dep.Name),
		"version":    llx.StringData(dep.Version),
		"repository": llx.StringData(dep.Repository),
		"condition":  llx.StringData(dep.Condition),
		"tags":       llx.ArrayData(tags, types.String),
		"enabled":    llx.BoolData(dep.Enabled),
		"alias":      llx.StringData(dep.Alias),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlHelmDependency), nil
}

func newMqlHelmMaintainer(runtime *plugin.Runtime, chartName string, m *chart.Maintainer) (*mqlHelmMaintainer, error) {
	res, err := CreateResource(runtime, "helm.maintainer", map[string]*llx.RawData{
		"__id":  llx.StringData("helm.maintainer:" + chartName + ":" + m.Name),
		"name":  llx.StringData(m.Name),
		"email": llx.StringData(m.Email),
		"url":   llx.StringData(m.URL),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlHelmMaintainer), nil
}

func newMqlHelmFile(runtime *plugin.Runtime, chartName string, f *chart.File) (*mqlHelmFile, error) {
	res, err := CreateResource(runtime, "helm.file", map[string]*llx.RawData{
		"__id": llx.StringData("helm.file:" + chartName + ":" + f.Name),
		"path": llx.StringData(f.Name),
	})
	if err != nil {
		return nil, err
	}
	mqlFile := res.(*mqlHelmFile)
	mqlFile.cacheData = f.Data
	return mqlFile, nil
}

type mqlHelmFileInternal struct {
	cacheData []byte
}

func (f *mqlHelmFile) content() (string, error) {
	return string(f.cacheData), nil
}
