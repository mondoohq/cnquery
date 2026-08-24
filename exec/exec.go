// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package exec

import (
	"errors"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/mql"
	"go.mondoo.com/mql/exec/internal"
	"go.mondoo.com/mql/llx"
	"go.mondoo.com/mql/mqlc"
)

// New creates a new MQL executor instance. It allows you to easily run multiple queries against the
// same runtime
func New(runtime llx.Runtime, features mql.Features) *Executor {
	return &Executor{
		runtime:  runtime,
		features: features,
	}
}

type Executor struct {
	runtime  llx.Runtime
	features mql.Features
}

// Exec runs a query with properties against the runtime
func (e *Executor) Exec(query string, props mqlc.PropsHandler) (*llx.RawData, error) {
	return Exec(query, e.runtime, e.features, props)
}

func Exec(query string, runtime llx.Runtime, features mql.Features, props mqlc.PropsHandler) (*llx.RawData, error) {
	return ExecStrict(query, runtime, features, props, false)
}

// ExecStrict is Exec with an explicit ADR 043 strict-mode setting. Exec keeps its
// signature and stays non-strict, which is the right default for a caller that
// has no content declaring otherwise.
func ExecStrict(query string, runtime llx.Runtime, features mql.Features, props mqlc.PropsHandler, strict bool) (*llx.RawData, error) {
	if props == nil {
		props = mqlc.EmptyPropsHandler
	}

	conf := mqlc.NewConfig(runtime.Schema(), features)
	conf.Strict = strict
	bundle, err := mqlc.Compile(query, props, conf)
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

	raw, err := ExecuteCode(runtime, bundle, props.Available(), features)
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

func ExecuteCode(runtime llx.Runtime, codeBundle *llx.CodeBundle, props map[string]*llx.Primitive, features mql.Features) (map[string]*llx.RawResult, error) {
	builder := internal.NewBuilder()

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
