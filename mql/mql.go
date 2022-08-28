package mql

import (
	"errors"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery"
	"go.mondoo.com/cnquery/llx"
	"go.mondoo.com/cnquery/motor"
	"go.mondoo.com/cnquery/motor/providers/mock"
	"go.mondoo.com/cnquery/mql/internal"
	"go.mondoo.com/cnquery/mqlc"
	"go.mondoo.com/cnquery/resources"
)

// New creates a new MQL executor instance. It allows you to easily run multiple queries against the
// same runtime
func New(runtime *resources.Runtime, features cnquery.Features) *Executor {
	return &Executor{
		runtime:  runtime,
		features: features,
	}
}

type Executor struct {
	runtime  *resources.Runtime
	features cnquery.Features
}

// Exec runs a query with properties against the runtime
func (e *Executor) Exec(query string, props map[string]*llx.Primitive) (*llx.RawData, error) {
	return Exec(query, e.runtime, e.features, props)
}

func Exec(query string, runtime *resources.Runtime, features cnquery.Features, props map[string]*llx.Primitive) (*llx.RawData, error) {
	bundle, err := mqlc.Compile(query, runtime.Registry.Schema(), features, props)
	if err != nil {
		return nil, errors.New("failed to compile: " + err.Error())
	}

	var results []*llx.RawResult

	if !features.IsActive(cnquery.PiperCode) {
		if len(bundle.DeprecatedV5Code.Entrypoints) == 0 {
			return llx.NilData, nil
		}

		if !bundle.DeprecatedV5Code.SingleValue {
			log.Warn().Str("query", query).Msg("mql> Code must only return one value, but it has many configured. Only returning last result.")
		}

		raw, err := llx.RunOnceSyncV1(bundle.DeprecatedV5Code, runtime, props)
		if err != nil {
			return nil, err
		}

		results = llx.ReturnValuesV1(bundle, func(checksum string) (*llx.RawResult, bool) {
			for i := range raw {
				if raw[i].CodeID == checksum {
					return raw[i], true
				}
			}
			return nil, false
		})
	} else {
		if len(bundle.CodeV2.Entrypoints()) == 0 {
			return llx.NilData, nil
		}

		if !bundle.CodeV2.Blocks[0].SingleValue {
			log.Warn().Str("query", query).Msg("mql> Code must only return one value, but it has many configured. Only returning last result.")
		}

		raw, err := ExecuteCode(runtime.Registry.Schema(), runtime, bundle, props, features)
		if err != nil {
			return nil, err
		}

		results = llx.ReturnValuesV2(bundle, func(checksum string) (*llx.RawResult, bool) {
			res, ok := raw[checksum]
			return res, ok
		})
	}

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

func MockRuntime() (*resources.Runtime, error) {
	provider, err := mock.New()
	if err != nil {
		return nil, err
	}
	m, err := motor.New(provider)
	if err != nil {
		return nil, err
	}

	registry := resources.NewRegistry()
	return resources.NewRuntime(registry, m), nil
}

func ExecuteCode(schema *resources.Schema, runtime *resources.Runtime, codeBundle *llx.CodeBundle, props map[string]*llx.Primitive, features cnquery.Features) (map[string]*llx.RawResult, error) {
	useV2Code := features.IsActive(cnquery.PiperCode)

	builder := internal.NewBuilder()
	builder.WithUseV2Code(useV2Code)
	builder.WithFeatureBoolAssertions(features.IsActive(cnquery.BoolAssertions))

	builder.AddQuery(codeBundle, nil, props)
	for _, checksum := range internal.CodepointChecksums(codeBundle, useV2Code) {
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

	ge, err := builder.Build(schema, runtime, "")
	if err != nil {
		return nil, err
	}

	if err := ge.Execute(); err != nil {
		return nil, err
	}

	return resultMap, nil
}
