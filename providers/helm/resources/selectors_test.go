// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/inventory"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/helm/connection"
)

func newSelectorRuntime(t *testing.T, chartPath string) *plugin.Runtime {
	t.Helper()
	asset := &inventory.Asset{Connections: []*inventory.Config{{
		Options: map[string]string{"path": chartPath},
	}}}
	conn, err := connection.NewHelmConnection(1, asset, &inventory.Config{})
	require.NoError(t, err)
	return plugin.NewRuntime(conn, nil, false, CreateResource, NewResource, GetData, SetData, nil)
}

// The schema documents `helm.template(name: ...)`. Without an init the runtime
// built the resource from the `name` arg alone — `raw` was fabricated empty and
// no __id was assigned, so every bare lookup shared the key `helm.template\x00`.
func TestInitHelmTemplateResolvesDocumentedSelector(t *testing.T) {
	rt := newSelectorRuntime(t, "../testdata/mychart")

	res, err := NewResource(rt, "helm.template", map[string]*llx.RawData{
		"name": llx.StringData("templates/deployment.yaml"),
	})
	require.NoError(t, err)

	tpl := res.(*mqlHelmTemplate)
	assert.Equal(t, "templates/deployment.yaml", tpl.Name.Data)
	assert.NotEmpty(t, tpl.Raw.Data, "raw source must be populated, not fabricated empty")
	assert.NotEmpty(t, tpl.MqlID(), "a resolved template must carry a stable __id")
}

func TestInitHelmTemplateDoesNotAlias(t *testing.T) {
	rt := newSelectorRuntime(t, "../testdata/mychart")

	a, err := NewResource(rt, "helm.template", map[string]*llx.RawData{
		"name": llx.StringData("templates/deployment.yaml"),
	})
	require.NoError(t, err)
	b, err := NewResource(rt, "helm.template", map[string]*llx.RawData{
		"name": llx.StringData("templates/service.yaml"),
	})
	require.NoError(t, err)

	ta, tb := a.(*mqlHelmTemplate), b.(*mqlHelmTemplate)
	require.NotEqual(t, ta.MqlID(), tb.MqlID(), "distinct templates need distinct __ids")
	assert.Equal(t, "templates/deployment.yaml", ta.Name.Data)
	assert.Equal(t, "templates/service.yaml", tb.Name.Data)
	assert.NotEqual(t, ta.Raw.Data, tb.Raw.Data)
}

func TestInitHelmTemplateUnknownNameErrors(t *testing.T) {
	rt := newSelectorRuntime(t, "../testdata/mychart")

	_, err := NewResource(rt, "helm.template", map[string]*llx.RawData{
		"name": llx.StringData("templates/does-not-exist.yaml"),
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "does-not-exist")
}

// helm.file's `content` used to read "" because the Internal cache was never
// populated on a bare-constructed resource — a fabricated empty answer for a
// file that plainly has content.
func TestInitHelmFileResolvesDocumentedSelector(t *testing.T) {
	rt := newSelectorRuntime(t, "../testdata/featurechart")

	res, err := NewResource(rt, "helm.file", map[string]*llx.RawData{
		"path": llx.StringData("README.md"),
	})
	require.NoError(t, err)

	f := res.(*mqlHelmFile)
	assert.Equal(t, "README.md", f.Path.Data)
	assert.NotEmpty(t, f.MqlID(), "a resolved file must carry a stable __id")

	content, err := f.content()
	require.NoError(t, err)
	assert.NotEmpty(t, content, "content must come from the real file, not a fabricated empty string")

	assert.Positive(t, f.Size.Data, "size must be populated, not left unset")
}

// Two selectors must not alias on the empty __id they used to share.
func TestInitHelmFileDoesNotAlias(t *testing.T) {
	rt := newSelectorRuntime(t, "../testdata/featurechart")

	a, err := NewResource(rt, "helm.file", map[string]*llx.RawData{
		"path": llx.StringData("README.md"),
	})
	require.NoError(t, err)
	b, err := NewResource(rt, "helm.file", map[string]*llx.RawData{
		"path": llx.StringData("crds/crontab.yaml"),
	})
	require.NoError(t, err)

	fa, fb := a.(*mqlHelmFile), b.(*mqlHelmFile)
	require.NotEqual(t, fa.MqlID(), fb.MqlID(), "distinct files need distinct __ids")
	assert.Equal(t, "README.md", fa.Path.Data)
	assert.Equal(t, "crds/crontab.yaml", fb.Path.Data)
}

func TestInitHelmFileUnknownPathErrors(t *testing.T) {
	rt := newSelectorRuntime(t, "../testdata/mychart")

	_, err := NewResource(rt, "helm.file", map[string]*llx.RawData{
		"path": llx.StringData("no/such/file.txt"),
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no/such/file.txt")
}

// A bare resource with no args stays a valid empty state — the legitimate fast
// path, distinct from a lookup miss.
func TestHelmSelectorBareResourcesAllowed(t *testing.T) {
	rt := newSelectorRuntime(t, "../testdata/mychart")

	args, res, err := initHelmFile(rt, map[string]*llx.RawData{})
	require.NoError(t, err)
	assert.Nil(t, res)
	assert.NotNil(t, args)

	args, res, err = initHelmTemplate(rt, map[string]*llx.RawData{})
	require.NoError(t, err)
	assert.Nil(t, res)
	assert.NotNil(t, args)
}
