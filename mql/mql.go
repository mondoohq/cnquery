// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package mql

import (
	"errors"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/v11"
	"go.mondoo.com/cnquery/v11/llx"
	"go.mondoo.com/cnquery/v11/mql/internal"
	"go.mondoo.com/cnquery/v11/mqlc"
)

// New creates a new MQL executor instance. It allows you to easily run multiple queries against the
// same runtime
func New(runtime llx.Runtime, features cnquery.Features) *Executor {
	return &Executor{
		runtime:  runtime,
		features: features,
	}
}

type Executor struct {
	runtime  llx.Runtime
	features cnquery.Features
}

// Exec runs a query with properties against the runtime
func (e *Executor) Exec(query string, props map[string]*llx.Primitive) (*llx.RawData, error) {
	return Exec(query, e.runtime, e.features, props)
}

func Exec(query string, runtime llx.Runtime, features cnquery.Features, props map[string]*llx.Primitive) (*llx.RawData, error) {
	bundle, err := mqlc.Compile(query, props, mqlc.NewConfig(runtime.Schema(), features))
	if err != nil {
		return nil, errors.New("failed to compile: " + err.Error())
	}

	var results []*llx.RawResult

	if len(bundle.CodeV2.Entrypoints()) == 0 {
		return llx.NilData, nil
	}

	if !bundle.CodeV2.Blocks[0].SingleValue {
		log.Warn().Str("query", query).Msg("mql> Code must only return one value, but it has many configured. Only returning last result.")
	}

	raw, err := ExecuteCode(runtime, bundle, props, features)
	if err != nil {
		return nil, err
	}

	results = llx.ReturnValuesV2(bundle, func(checksum string) (*llx.RawResult, bool) {
		res, ok := raw[checksum]
		return res, ok
	})

	if len(results) > 1 {
		return nil, errors.New("too many results received")
	}

	rawres := results[0]
	res, err := rawres.Data.Dereference(rawres.CodeID, bundle)
	if err != nil {
		return nil, err
	}

	return res, nil
}

func ExecuteCode(runtime llx.Runtime, codeBundle *llx.CodeBundle, props map[string]*llx.Primitive, features cnquery.Features) (map[string]*llx.RawResult, error) {
	builder := internal.NewBuilder()
	builder.WithFeatureBoolAssertions(features.IsActive(cnquery.BoolAssertions))

	builder.AddQuery(codeBundle, nil, props)
	for _, checksum := range internal.CodepointChecksums(codeBundle) {
		builder.CollectDatapoint(checksum)
	}

	resultMap := map[string]*llx.RawResult{}
	collector := &internal.FuncCollector{
		SinkDataFunc: func(results []*llx.RawResult) {
			for _, d := range results {
				resultMap[d.CodeID] = d
			}
		},
	}
	builder.AddDatapointCollector(collector)

	ge, err := builder.Build(runtime.Schema(), runtime, "")
	if err != nil {
		return nil, err
	}

	if err := ge.Execute(); err != nil {
		return nil, err
	}

	return resultMap, nil
}
