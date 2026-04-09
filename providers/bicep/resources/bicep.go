// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/bicep/connection"
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
	parsed *parsedBicepFile
}

func newMqlBicepFile(runtime *plugin.Runtime, f *connection.BicepFile) (*mqlBicepFile, error) {
	parsed := parseBicep(f.Content)

	res, err := CreateResource(runtime, "bicep.file", map[string]*llx.RawData{
		"__id":        llx.StringData("bicep.file:" + f.Path),
		"path":        llx.StringData(f.Path),
		"targetScope": llx.StringData(parsed.targetScope),
		"content":     llx.StringData(f.Content),
	})
	if err != nil {
		return nil, err
	}
	mqlF := res.(*mqlBicepFile)
	mqlF.parsed = parsed
	return mqlF, nil
}

func (f *mqlBicepFile) id() (string, error) {
	return "bicep.file:" + f.Path.Data, nil
}

func (f *mqlBicepFile) parameters() ([]any, error) {
	return createMqlParameters(f.MqlRuntime, f.Path.Data, f.parsed.parameters)
}

func (f *mqlBicepFile) variables() ([]any, error) {
	return createMqlVariables(f.MqlRuntime, f.Path.Data, f.parsed.variables)
}

func (f *mqlBicepFile) resources() ([]any, error) {
	return createMqlResources(f.MqlRuntime, f.Path.Data, f.parsed.resources)
}

func (f *mqlBicepFile) modules() ([]any, error) {
	return createMqlModules(f.MqlRuntime, f.Path.Data, f.parsed.modules)
}

func (f *mqlBicepFile) outputs() ([]any, error) {
	return createMqlOutputs(f.MqlRuntime, f.Path.Data, f.parsed.outputs)
}
