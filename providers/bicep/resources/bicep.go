// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"sync"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/bicep/connection"
	"go.mondoo.com/mql/v13/types"
)

func (r *mqlBicep) id() (string, error) {
	return "bicep", nil
}

func (r *mqlBicep) files() ([]any, error) {
	conn := r.MqlRuntime.Connection.(*connection.BicepConnection)
	bicepFiles := conn.BicepFiles()

	var mqlFiles []any
	for _, f := range bicepFiles {
		mqlF, err := newMqlBicepFile(r.MqlRuntime, f)
		if err != nil {
			return nil, err
		}
		mqlFiles = append(mqlFiles, mqlF)
	}
	return mqlFiles, nil
}

func (r *mqlBicep) template() (*mqlBicepTemplate, error) {
	conn := r.MqlRuntime.Connection.(*connection.BicepConnection)
	armTmpl := conn.ARMTemplate()
	if armTmpl == nil {
		r.Template.State = plugin.StateIsNull | plugin.StateIsSet
		return nil, nil
	}
	return newMqlBicepTemplate(r.MqlRuntime, conn.Path(), armTmpl)
}

type mqlBicepFileInternal struct {
	parseOnce sync.Once
	parsed    *parsedBicepFile
}

func newMqlBicepFile(runtime *plugin.Runtime, f *connection.BicepFile) (*mqlBicepFile, error) {
	parsed := parseBicep(f.Content)

	metadata := make(map[string]any, len(parsed.metadata))
	for k, v := range parsed.metadata {
		metadata[k] = v
	}

	res, err := CreateResource(runtime, "bicep.file", map[string]*llx.RawData{
		"__id":        llx.StringData("bicep.file:" + f.Path),
		"path":        llx.StringData(f.Path),
		"targetScope": llx.StringData(parsed.targetScope),
		"metadata":    llx.MapData(metadata, types.String),
		"content":     llx.StringData(f.Content),
	})
	if err != nil {
		return nil, err
	}
	mqlF := res.(*mqlBicepFile)
	// CreateResource may return a cached instance for the same __id;
	// parseOnce keeps the stamp race-free under concurrent callers and
	// happens-before any subsequent reader.
	mqlF.parseOnce.Do(func() { mqlF.parsed = parsed })
	return mqlF, nil
}

// getParsed returns the parsed Bicep model, parsing on demand if the
// resource was reconstructed across a gRPC boundary (where Internal
// fields are zeroed but the public `content` field survives).
func (f *mqlBicepFile) getParsed() *parsedBicepFile {
	f.parseOnce.Do(func() {
		f.parsed = parseBicep(f.Content.Data)
	})
	return f.parsed
}

func (f *mqlBicepFile) id() (string, error) {
	return "bicep.file:" + f.Path.Data, nil
}

func (f *mqlBicepFile) parameters() ([]any, error) {
	return createMqlParameters(f.MqlRuntime, f.Path.Data, f.getParsed().parameters)
}

func (f *mqlBicepFile) variables() ([]any, error) {
	return createMqlVariables(f.MqlRuntime, f.Path.Data, f.getParsed().variables)
}

func (f *mqlBicepFile) resources() ([]any, error) {
	return createMqlResources(f.MqlRuntime, f.Path.Data, f.getParsed().resources)
}

func (f *mqlBicepFile) modules() ([]any, error) {
	return createMqlModules(f.MqlRuntime, f.Path.Data, f.getParsed().modules)
}

func (f *mqlBicepFile) outputs() ([]any, error) {
	return createMqlOutputs(f.MqlRuntime, f.Path.Data, f.getParsed().outputs)
}

func (f *mqlBicepFile) types() ([]any, error) {
	return createMqlTypes(f.MqlRuntime, f.Path.Data, f.getParsed().types)
}

func (f *mqlBicepFile) functions() ([]any, error) {
	return createMqlFunctions(f.MqlRuntime, f.Path.Data, f.getParsed().functions)
}

func (f *mqlBicepFile) imports() ([]any, error) {
	return createMqlImports(f.MqlRuntime, f.Path.Data, f.getParsed().imports)
}
