// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"errors"
	"strconv"
	"strings"
	"sync"

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
	for _, lc := range charts {
		mqlChart, err := newMqlHelmChart(r.MqlRuntime, lc.Chart, lc.Path)
		if err != nil {
			return nil, err
		}
		mqlCharts = append(mqlCharts, mqlChart)
	}
	return mqlCharts, nil
}

type mqlHelmChartInternal struct {
	chartObj        *chart.Chart
	chartPath       string
	renderedOnce    sync.Once
	rendered        map[string]string
	renderedErr     error
	resourcesOnce   sync.Once
	cachedResources []any
}

func newMqlHelmChart(runtime *plugin.Runtime, c *chart.Chart, chartPath string) (*mqlHelmChart, error) {
	// Guard against archives that load with a nil Metadata pointer.
	// loader.LoadDir always populates Metadata for a valid chart, but
	// loader.LoadFile (the .tgz path) can return a chart with nil
	// Metadata on a truncated/malformed archive.
	if c == nil || c.Metadata == nil {
		return nil, errors.New("helm chart has no metadata")
	}
	meta := c.Metadata

	keywords := convert.SliceAnyToInterface(meta.Keywords)
	sources := convert.SliceAnyToInterface(meta.Sources)

	// Include the chart's filesystem path in the cache key so two
	// distinct charts that happen to share name + version (common for
	// feature-branch forks in a multi-chart directory) don't collide
	// on CreateResource's cache.
	chartKey := meta.Name + ":" + meta.Version
	if chartPath != "" {
		chartKey += ":" + chartPath
	}

	args := map[string]*llx.RawData{
		"__id":        llx.StringData("helm.chart:" + chartKey),
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
	mqlChart.chartPath = chartPath
	return mqlChart, nil
}

func (c *mqlHelmChart) id() (string, error) {
	key := c.Name.Data + ":" + c.Version.Data
	if c.chartPath != "" {
		key += ":" + c.chartPath
	}
	return "helm.chart:" + key, nil
}

func (c *mqlHelmChart) fetchRendered() (map[string]string, error) {
	c.renderedOnce.Do(func() {
		e := engine.Engine{Strict: false}
		options := chartutil.ReleaseOptions{
			Name:      c.chartObj.Name(),
			Namespace: "default",
			IsInstall: true,
		}
		vals, err := chartutil.ToRenderValues(c.chartObj, c.chartObj.Values, options, nil)
		if err != nil {
			c.renderedErr = err
			return
		}
		c.rendered, c.renderedErr = e.Render(c.chartObj, vals)
	})
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
	for i, m := range maintainers {
		mqlM, err := newMqlHelmMaintainer(c.MqlRuntime, c.chartObj.Name(), i, m)
		if err != nil {
			return nil, err
		}
		mqlMaintainers = append(mqlMaintainers, mqlM)
	}
	return mqlMaintainers, nil
}

func (c *mqlHelmChart) templates() ([]any, error) {
	rendered, renderErr := c.fetchRendered()
	if renderErr != nil {
		log.Warn().Err(renderErr).Str("chart", c.chartObj.Name()).Msg("failed to render helm chart templates")
	}

	var mqlTemplates []any
	for _, t := range c.chartObj.Templates {
		renderedContent := ""
		if rendered != nil {
			// Helm uses "chartName/templateName" as the key
			renderedContent = rendered[c.chartObj.Name()+"/"+t.Name]
		}
		// Pass renderErr through so the template's rendered() accessor
		// can surface it instead of silently returning "" — that lets
		// policy authors distinguish "rendered to empty output" from
		// "rendering failed."
		mqlT, err := newMqlHelmTemplate(c.MqlRuntime, c.chartObj.Name(), t, renderedContent, renderErr)
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
	return c.fetchResources()
}

// fetchResources parses every rendered template into K8s resources
// and caches the result on the chart's Internal struct so repeated
// reads of helm.chart.resources don't reparse the same YAML on
// every access. helm.template.resources parses its own rendered
// string directly (scoped to a single template), so it doesn't go
// through this cache.
//
// Chart-wide render failures are intentionally swallowed at this
// level — see TestHelmRequiredValuesGraceful. A chart that uses
// `required` with no values returns no resources but does not error
// out the whole audit query. Per-template failures still surface
// through helm.template.rendered() and helm.template.resources().
func (c *mqlHelmChart) fetchResources() ([]any, error) {
	c.resourcesOnce.Do(func() {
		// Initialize to an empty (non-nil) slice so the success-with-
		// zero-resources path and the render-error path return the
		// same shape — callers can rely on a non-nil result either way.
		c.cachedResources = []any{}
		rendered, err := c.fetchRendered()
		if err != nil {
			log.Warn().Err(err).Str("chart", c.chartObj.Name()).Msg("failed to render helm chart templates, returning empty resources")
			return
		}
		for templateKey, content := range rendered {
			resources, err := parseK8sResources(c.MqlRuntime, templateKey, content)
			if err != nil {
				continue
			}
			c.cachedResources = append(c.cachedResources, resources...)
		}
	})
	return c.cachedResources, nil
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
		"sourceType": llx.StringData(classifyHelmSource(dep.Repository, dep.Alias)),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlHelmDependency), nil
}

// classifyHelmSource categorizes a dependency's source from its
// repository value (offline, no network). An empty repository with an
// alias is a sibling-chart reference; an empty repository with neither
// is unknown.
func classifyHelmSource(repository, alias string) string {
	repo := strings.TrimSpace(repository)
	switch {
	case repo == "":
		if alias != "" {
			return "alias"
		}
		return "unknown"
	case strings.HasPrefix(repo, "oci://"):
		return "oci"
	case strings.HasPrefix(repo, "https://"):
		return "https"
	case strings.HasPrefix(repo, "http://"):
		return "http"
	case strings.HasPrefix(repo, "file://"), strings.HasPrefix(repo, "./"), strings.HasPrefix(repo, "../"), strings.HasPrefix(repo, "/"):
		return "file"
	default:
		return "unknown"
	}
}

func (d *mqlHelmDependency) registryRef() (*mqlHelmOciRef, error) {
	if d.SourceType.Data != "oci" {
		d.RegistryRef.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}

	parsed := parseOciRef(d.Repository.Data, d.Version.Data)

	res, err := CreateResource(d.MqlRuntime, "helm.ociRef", map[string]*llx.RawData{
		"__id":       llx.StringData("helm.dependency:" + d.Name.Data + "/ociRef"),
		"reference":  llx.StringData(parsed.reference),
		"registry":   llx.StringData(parsed.registry),
		"repository": llx.StringData(parsed.repository),
		"tag":        llx.StringData(parsed.tag),
		"digest":     llx.StringData(parsed.digest),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlHelmOciRef), nil
}

type ociRef struct {
	reference  string
	registry   string
	repository string
	tag        string
	digest     string
}

// parseOciRef decomposes an oci:// reference (and the dependency's
// version constraint) into registry / repository / tag / digest. It is
// defensive: an unparseable reference yields whatever parts could be
// recovered with empty strings for the rest, and never panics.
//
//	oci://ghcr.io/acme/charts/redis        -> registry=ghcr.io repository=acme/charts/redis
//	oci://ghcr.io/acme/redis:1.2.3         -> ... tag=1.2.3
//	oci://ghcr.io/acme/redis@sha256:abc... -> ... digest=sha256:abc...
//
// The tag falls back to the dependency version when it's a concrete
// version (no range operators); the digest falls back to the version
// when the version itself pins a sha256 digest.
func parseOciRef(repository, version string) ociRef {
	ref := ociRef{reference: repository}

	rest := strings.TrimPrefix(strings.TrimSpace(repository), "oci://")

	// A digest pin (@sha256:...) takes precedence over a :tag suffix.
	if at := strings.LastIndex(rest, "@"); at != -1 {
		ref.digest = rest[at+1:]
		rest = rest[:at]
	}

	// Split host from path. Everything before the first "/" is the
	// registry host; the remainder is the repository path.
	if slash := strings.Index(rest, "/"); slash != -1 {
		ref.registry = rest[:slash]
		repoPath := rest[slash+1:]

		// A :tag suffix on the final path segment (only when no digest
		// was found, and only if the colon isn't part of a host:port —
		// which lives in the registry segment, already split off).
		if ref.digest == "" {
			if colon := strings.LastIndex(repoPath, ":"); colon != -1 {
				ref.tag = repoPath[colon+1:]
				repoPath = repoPath[:colon]
			}
		}
		ref.repository = repoPath
	} else {
		// No path component — treat the whole thing as the registry.
		ref.registry = rest
	}

	// Fall back to the version constraint for tag/digest when the
	// reference itself didn't carry one.
	v := strings.TrimSpace(version)
	if ref.digest == "" && strings.HasPrefix(v, "sha256:") {
		ref.digest = v
	} else if ref.tag == "" && v != "" && isConcreteVersion(v) {
		ref.tag = v
	}

	return ref
}

// isConcreteVersion reports whether a version string is a single
// concrete version rather than a SemVer range/constraint. OCI tags are
// concrete, so only a concrete version maps onto a tag.
func isConcreteVersion(v string) bool {
	return !strings.ContainsAny(v, "^~*><= |,x")
}

func newMqlHelmMaintainer(runtime *plugin.Runtime, chartName string, idx int, m *chart.Maintainer) (*mqlHelmMaintainer, error) {
	// Include the loop index so a chart that declares two maintainers
	// with the same name doesn't silently dedupe through the resource
	// cache. (The Helm spec permits duplicate maintainer names.)
	id := "helm.maintainer:" + chartName + ":" + strconv.Itoa(idx) + ":" + m.Name
	res, err := CreateResource(runtime, "helm.maintainer", map[string]*llx.RawData{
		"__id":  llx.StringData(id),
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
