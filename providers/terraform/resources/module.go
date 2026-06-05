// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"path/filepath"
	"strings"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/terraform/connection"
)

// fileModuleEntry maps a directory holding a module's .tf files to the module
// call that introduced it.
type fileModuleEntry struct {
	dir  string
	call *mqlTerraformBlock
}

// moduleForFile returns the module call whose source directory contains the
// given file, or nil for root-module files. The longest matching directory
// wins so nested modules resolve to the innermost call.
func (t *mqlTerraform) moduleForFile(path string) *mqlTerraformBlock {
	return matchModuleEntry(t.fileModuleIndex(), path)
}

// matchModuleEntry finds the module call whose source directory contains the
// file (longest match wins), or nil for root files. It does not lock.
func matchModuleEntry(entries []fileModuleEntry, path string) *mqlTerraformBlock {
	if path == "" {
		return nil
	}
	dir := filepath.Dir(path)

	var best *mqlTerraformBlock
	bestLen := -1
	for i := range entries {
		e := entries[i]
		if dir == e.dir || strings.HasPrefix(dir, e.dir+string(filepath.Separator)) {
			if len(e.dir) > bestLen {
				bestLen = len(e.dir)
				best = e.call
			}
		}
	}
	return best
}

// blockFilePath returns the source file a block was parsed from, or "".
func blockFilePath(b *mqlTerraformBlock) string {
	if b == nil || b.Start.State != plugin.StateIsSet || b.Start.Data == nil {
		return ""
	}
	return b.Start.Data.Path.Data
}

// fileModuleIndex builds (once) the directory -> module-call mapping from the
// `module` blocks, resolving local sources relative to the call's file and
// initialized sources from the modules manifest.
func (t *mqlTerraform) fileModuleIndex() []fileModuleEntry {
	blocksRaw := t.GetBlocks()
	if blocksRaw.Error != nil {
		return nil
	}
	allBlocks := blocksRaw.Data

	var manifest *connection.ModuleManifest
	if conn, ok := t.MqlRuntime.Connection.(*connection.Connection); ok {
		manifest = conn.ModulesManifest()
	}

	t.lock.Lock()
	defer t.lock.Unlock()
	if t.fileModules != nil {
		return t.fileModules
	}

	entries := []fileModuleEntry{}
	for i := range allBlocks {
		b := allBlocks[i].(*mqlTerraformBlock)
		if b.Type.Data != "module" {
			continue
		}
		source := blockStringArgument(b, "source")
		if source == "" {
			continue
		}

		var dir string
		if strings.HasPrefix(source, "./") || strings.HasPrefix(source, "../") {
			if b.Start.State == plugin.StateIsSet && b.Start.Data != nil {
				callDir := filepath.Dir(b.Start.Data.Path.Data)
				dir = filepath.Clean(filepath.Join(callDir, source))
			}
		} else if manifest != nil {
			name := labelAt(b, 0)
			for j := range manifest.Records {
				if manifest.Records[j].Key == name && manifest.Records[j].Dir != "" {
					dir = filepath.Clean(manifest.Records[j].Dir)
					break
				}
			}
		}

		if dir != "" {
			entries = append(entries, fileModuleEntry{dir: dir, call: b})
		}
	}

	t.fileModules = entries
	return entries
}

type mqlTerraformModuleInternal struct {
	tf      *mqlTerraform
	tfBlock *mqlTerraformBlock
}

// moduleMetaArguments are the module-call arguments that are not module inputs.
var moduleMetaArguments = map[string]bool{
	"source":     true,
	"version":    true,
	"count":      true,
	"for_each":   true,
	"depends_on": true,
	"providers":  true,
}

// moduleCalls builds a terraform.module record from each `module` block in the
// source HCL, so module calls are queryable without `terraform init`.
func (t *mqlTerraform) moduleCalls() ([]any, error) {
	if err := t.refreshCache(nil); err != nil {
		return nil, err
	}
	blocksRaw := t.GetBlocks()
	if blocksRaw.Error != nil {
		return nil, blocksRaw.Error
	}

	res := []any{}
	for i := range blocksRaw.Data {
		b := blocksRaw.Data[i].(*mqlTerraformBlock)
		if b.Type.Data != "module" {
			continue
		}
		r, err := CreateResource(t.MqlRuntime, "terraform.module", map[string]*llx.RawData{
			"key":     llx.StringData(labelAt(b, 0)),
			"source":  llx.StringData(blockStringArgument(b, "source")),
			"version": llx.StringData(blockStringArgument(b, "version")),
			"dir":     llx.StringData(""),
		})
		if err != nil {
			return nil, err
		}
		m := r.(*mqlTerraformModule)
		m.tf = t
		m.tfBlock = b
		res = append(res, m)
	}
	return res, nil
}

// moduleBlock returns the HCL block backing this module — the one captured at
// construction for module calls, or the one located by the manifest path.
func (m *mqlTerraformModule) moduleBlock() *mqlTerraformBlock {
	if m.tfBlock != nil {
		return m.tfBlock
	}
	blk := m.GetBlock()
	if blk.Error != nil {
		return nil
	}
	return blk.Data
}

func (m *mqlTerraformModule) sourceType() (string, error) {
	return classifyModuleSource(m.Source.Data), nil
}

// classifyModuleSource maps a module source address to one of local, git,
// http, registry, or "" (unset), following Terraform's source-addressing.
func classifyModuleSource(s string) string {
	switch {
	case s == "":
		return ""
	case strings.HasPrefix(s, "./"), strings.HasPrefix(s, "../"), strings.HasPrefix(s, "/"):
		return "local"
	case strings.HasPrefix(s, "git::"), strings.HasPrefix(s, "git@"),
		strings.Contains(s, "github.com"), strings.Contains(s, "gitlab.com"),
		strings.Contains(s, "bitbucket.org"), strings.HasSuffix(s, ".git"):
		return "git"
	case strings.HasPrefix(s, "http://"), strings.HasPrefix(s, "https://"),
		strings.HasPrefix(s, "s3::"), strings.HasPrefix(s, "gcs::"):
		return "http"
	default:
		return "registry"
	}
}

func (m *mqlTerraformModule) inputs() (any, error) {
	b := m.moduleBlock()
	if b == nil {
		return map[string]any{}, nil
	}
	args, err := b.arguments()
	if err != nil {
		return nil, err
	}
	out := map[string]any{}
	for k := range args {
		if !moduleMetaArguments[k] {
			out[k] = args[k]
		}
	}
	return out, nil
}

func (m *mqlTerraformModule) references() ([]any, error) {
	b := m.moduleBlock()
	if b == nil {
		return []any{}, nil
	}
	if m.tf != nil {
		return terraformReferences(m.MqlRuntime, m.tf, b)
	}
	return b.references()
}

// --- navigating into a module's own source ---

// terraformRef resolves the parent terraform resource, constructing it when the
// module was created bare (e.g. a manifest module from terraform.modules).
func (m *mqlTerraformModule) terraformRef() (*mqlTerraform, error) {
	if m.tf != nil {
		return m.tf, nil
	}
	o, err := CreateResource(m.MqlRuntime, "terraform", map[string]*llx.RawData{})
	if err != nil {
		return nil, err
	}
	tf := o.(*mqlTerraform)
	if err := tf.refreshCache(nil); err != nil {
		return nil, err
	}
	m.tf = tf
	return tf, nil
}

// callBlock returns the module-call HCL block, located by key when it was not
// captured at construction.
func (m *mqlTerraformModule) callBlock(tf *mqlTerraform) *mqlTerraformBlock {
	if m.tfBlock != nil {
		return m.tfBlock
	}
	key := m.Key.Data
	blocksRaw := tf.GetBlocks()
	if blocksRaw.Error != nil {
		return nil
	}
	for i := range blocksRaw.Data {
		b := blocksRaw.Data[i].(*mqlTerraformBlock)
		if b.Type.Data == "module" && labelAt(b, 0) == key {
			return b
		}
	}
	return nil
}

// scopedBlocks returns the blocks of a given type declared inside this module's
// source (i.e. associated with this module call).
func (m *mqlTerraformModule) scopedBlocks(blockType string) ([]*mqlTerraformBlock, *mqlTerraform, error) {
	tf, err := m.terraformRef()
	if err != nil {
		return nil, nil, err
	}
	call := m.callBlock(tf)
	if call == nil {
		return nil, tf, nil
	}
	callID, _ := call.id()
	entries := tf.fileModuleIndex()

	blocksRaw := tf.GetBlocks()
	if blocksRaw.Error != nil {
		return nil, tf, blocksRaw.Error
	}

	var out []*mqlTerraformBlock
	for i := range blocksRaw.Data {
		b := blocksRaw.Data[i].(*mqlTerraformBlock)
		if b.Type.Data == blockType && blockModuleKey(b, entries) == callID {
			out = append(out, b)
		}
	}
	return out, tf, nil
}

func (m *mqlTerraformModule) resources() ([]any, error) {
	blocks, tf, err := m.scopedBlocks("resource")
	if err != nil {
		return nil, err
	}
	out := make([]any, 0, len(blocks))
	for _, b := range blocks {
		r, err := newTerraformResource(tf, b)
		if err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, nil
}

func (m *mqlTerraformModule) dataSources() ([]any, error) {
	blocks, tf, err := m.scopedBlocks("data")
	if err != nil {
		return nil, err
	}
	out := make([]any, 0, len(blocks))
	for _, b := range blocks {
		r, err := newTerraformDatasource(tf, b)
		if err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, nil
}

func (m *mqlTerraformModule) variables() ([]any, error) {
	blocks, tf, err := m.scopedBlocks("variable")
	if err != nil {
		return nil, err
	}
	out := make([]any, 0, len(blocks))
	for _, b := range blocks {
		r, err := newTerraformVariable(tf, b)
		if err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, nil
}

func (m *mqlTerraformModule) outputs() ([]any, error) {
	blocks, tf, err := m.scopedBlocks("output")
	if err != nil {
		return nil, err
	}
	out := make([]any, 0, len(blocks))
	for _, b := range blocks {
		r, err := newTerraformOutput(tf, b)
		if err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, nil
}

func (m *mqlTerraformModule) childModules() ([]any, error) {
	blocks, tf, err := m.scopedBlocks("module")
	if err != nil {
		return nil, err
	}
	out := make([]any, 0, len(blocks))
	for _, b := range blocks {
		r, err := newModuleFromBlock(tf, b)
		if err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, nil
}

func (m *mqlTerraformModule) files() ([]any, error) {
	tf, err := m.terraformRef()
	if err != nil {
		return nil, err
	}
	dir := m.moduleDir(tf)
	if dir == "" {
		return []any{}, nil
	}
	conn, ok := tf.MqlRuntime.Connection.(*connection.Connection)
	if !ok || conn.Parser() == nil {
		return []any{}, nil
	}

	out := []any{}
	for path := range conn.Parser().Files() {
		fileDir := filepath.Dir(path)
		if fileDir == dir || strings.HasPrefix(fileDir, dir+string(filepath.Separator)) {
			f, err := CreateResource(tf.MqlRuntime, "terraform.file", map[string]*llx.RawData{
				"path": llx.StringData(path),
			})
			if err != nil {
				return nil, err
			}
			out = append(out, f)
		}
	}
	return out, nil
}

// moduleDir returns the on-disk source directory of this module.
func (m *mqlTerraformModule) moduleDir(tf *mqlTerraform) string {
	call := m.callBlock(tf)
	if call != nil {
		entries := tf.fileModuleIndex()
		for i := range entries {
			if entries[i].call == call {
				return entries[i].dir
			}
		}
	}
	return m.Dir.Data
}

// newModuleFromBlock builds a terraform.module from a `module` block.
func newModuleFromBlock(t *mqlTerraform, b *mqlTerraformBlock) (*mqlTerraformModule, error) {
	r, err := CreateResource(t.MqlRuntime, "terraform.module", map[string]*llx.RawData{
		"key":     llx.StringData(labelAt(b, 0)),
		"source":  llx.StringData(blockStringArgument(b, "source")),
		"version": llx.StringData(blockStringArgument(b, "version")),
		"dir":     llx.StringData(""),
	})
	if err != nil {
		return nil, err
	}
	mod := r.(*mqlTerraformModule)
	mod.tf = t
	mod.tfBlock = b
	return mod, nil
}

// blockModuleCall returns the module call a block belongs to, by file path.
func blockModuleCall(t *mqlTerraform, b *mqlTerraformBlock) *mqlTerraformBlock {
	if b == nil || b.Start.State != plugin.StateIsSet || b.Start.Data == nil {
		return nil
	}
	return t.moduleForFile(b.Start.Data.Path.Data)
}

// qualifiedAddress prefixes an address with its module path, e.g.
// "module.vpc.aws_subnet.private", or returns it unchanged for root resources.
func qualifiedAddress(moduleCall *mqlTerraformBlock, base string) string {
	if moduleCall == nil {
		return base
	}
	return "module." + labelAt(moduleCall, 0) + "." + base
}

// --- typed selection by type / name ---

func initTerraformResource(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	return initTypedSelection(runtime, args, "resource")
}

func initTerraformDatasource(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	return initTypedSelection(runtime, args, "data")
}

func initTypedSelection(runtime *plugin.Runtime, args map[string]*llx.RawData, kind string) (map[string]*llx.RawData, plugin.Resource, error) {
	// No selection arguments means the resource is being constructed
	// internally (e.g. by managedResources); pass through unchanged.
	if args["type"] == nil && args["name"] == nil {
		return args, nil, nil
	}

	matchType, err := llx.StringOrRegexMatcher(args["type"])
	if err != nil {
		return nil, nil, err
	}
	matchName, err := llx.StringOrRegexMatcher(args["name"])
	if err != nil {
		return nil, nil, err
	}

	o, err := CreateResource(runtime, "terraform", map[string]*llx.RawData{})
	if err != nil {
		return nil, nil, err
	}
	tf := o.(*mqlTerraform)
	if err := tf.refreshCache(nil); err != nil {
		return nil, nil, err
	}

	var pool []*mqlTerraformBlock
	if kind == "resource" {
		pool = tf.mqlTerraformInternal.resources
	} else {
		for i := range tf.Datasources.Data {
			pool = append(pool, tf.Datasources.Data[i].(*mqlTerraformBlock))
		}
	}

	for _, b := range pool {
		if matchType != nil && !matchType(labelAt(b, 0)) {
			continue
		}
		if matchName != nil && !matchName(labelAt(b, 1)) {
			continue
		}
		if kind == "resource" {
			r, err := newTerraformResource(tf, b)
			return nil, r, err
		}
		r, err := newTerraformDatasource(tf, b)
		return nil, r, err
	}

	// No match: a bare record (its accessors return empty values).
	return args, nil, nil
}
