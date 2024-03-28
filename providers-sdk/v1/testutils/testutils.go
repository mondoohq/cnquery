// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package testutils

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnquery/v10"
	"go.mondoo.com/cnquery/v10/llx"
	"go.mondoo.com/cnquery/v10/logger"
	"go.mondoo.com/cnquery/v10/mql"
	"go.mondoo.com/cnquery/v10/mqlc"
	"go.mondoo.com/cnquery/v10/providers"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/lr"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/lr/docs"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/recording"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/resources"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/testutils/mockprovider"
	networkconf "go.mondoo.com/cnquery/v10/providers/network/config"
	networkprovider "go.mondoo.com/cnquery/v10/providers/network/provider"
	osconf "go.mondoo.com/cnquery/v10/providers/os/config"
	osprovider "go.mondoo.com/cnquery/v10/providers/os/provider"
	"sigs.k8s.io/yaml"
)

var (
	Features     cnquery.Features
	TestutilsDir string
)

func init() {
	logger.InitTestEnv()
	Features = getEnvFeatures()

	_, pathToFile, _, ok := runtime.Caller(0)
	if !ok {
		panic("unable to get runtime for testutils for cnquery providers")
	}
	TestutilsDir = path.Dir(pathToFile)
}

func getEnvFeatures() cnquery.Features {
	env := os.Getenv("FEATURES")
	if env == "" {
		return cnquery.Features{byte(cnquery.PiperCode)}
	}

	arr := strings.Split(env, ",")
	var fts cnquery.Features
	for i := range arr {
		v, ok := cnquery.FeaturesValue[arr[i]]
		if ok {
			fmt.Println("--> activate feature: " + arr[i])
			fts = append(Features, byte(v))
		} else {
			panic("cannot find requested feature: " + arr[i])
		}
	}
	return fts
}

type tester struct {
	Runtime llx.Runtime
}

type SchemaProvider struct {
	Provider string
	Path     string
}

func InitTester(runtime llx.Runtime) *tester {
	return &tester{
		Runtime: runtime,
	}
}

func (ctx *tester) Compile(query string) (*llx.CodeBundle, error) {
	return mqlc.Compile(query, nil, mqlc.NewConfig(ctx.Runtime.Schema(), Features))
}

func (ctx *tester) ExecuteCode(bundle *llx.CodeBundle, props map[string]*llx.Primitive) (map[string]*llx.RawResult, error) {
	return mql.ExecuteCode(ctx.Runtime, bundle, props, Features)
}

func (ctx *tester) TestQueryPWithError(t *testing.T, query string, props map[string]*llx.Primitive) ([]*llx.RawResult, error) {
	t.Helper()
	bundle, err := mqlc.Compile(query, props, mqlc.NewConfig(ctx.Runtime.Schema(), Features))
	if err != nil {
		return nil, fmt.Errorf("failed to compile code: %w", err)
	}
	err = mqlc.Invariants.Check(bundle)
	if err != nil {
		return nil, fmt.Errorf("failed to check invariants: %w", err)
	}
	return ctx.TestMqlc(t, bundle, props), nil
}

func (ctx *tester) TestQueryP(t *testing.T, query string, props map[string]*llx.Primitive) []*llx.RawResult {
	t.Helper()
	bundle, err := mqlc.Compile(query, props, mqlc.NewConfig(ctx.Runtime.Schema(), Features))
	if err != nil {
		t.Fatal("failed to compile code: " + err.Error())
	}
	err = mqlc.Invariants.Check(bundle)
	require.NoError(t, err)
	return ctx.TestMqlc(t, bundle, props)
}

func (ctx *tester) TestQuery(t *testing.T, query string) []*llx.RawResult {
	return ctx.TestQueryP(t, query, nil)
}

func (ctx *tester) TestMqlc(t *testing.T, bundle *llx.CodeBundle, props map[string]*llx.Primitive) []*llx.RawResult {
	t.Helper()

	resultMap, err := mql.ExecuteCode(ctx.Runtime, bundle, props, Features)
	require.NoError(t, err)

	lastQueryResult := &llx.RawResult{}
	results := make([]*llx.RawResult, 0, len(resultMap)+1)

	refs := make([]uint64, 0, len(bundle.CodeV2.Checksums))
	for _, datapointArr := range [][]uint64{bundle.CodeV2.Datapoints(), bundle.CodeV2.Entrypoints()} {
		refs = append(refs, datapointArr...)
	}

	sort.Slice(refs, func(i, j int) bool {
		return refs[i] < refs[j]
	})

	for idx, ref := range refs {
		checksum := bundle.CodeV2.Checksums[ref]
		if d, ok := resultMap[checksum]; ok {
			results = append(results, d)
			if idx+1 == len(refs) {
				lastQueryResult.CodeID = d.CodeID
				if d.Data.Error != nil {
					lastQueryResult.Data = &llx.RawData{
						Error: d.Data.Error,
					}
				} else {
					success, valid := d.Data.IsSuccess()
					lastQueryResult.Data = llx.BoolData(success && valid)
				}
			}
		}
	}

	results = append(results, lastQueryResult)
	return results
}

func MustLoadSchema(provider SchemaProvider) *resources.Schema {
	if provider.Path == "" && provider.Provider == "" {
		panic("cannot load schema without provider name or path")
	}
	var path string
	// path towards the .yaml manifest, containing metadata about the resources
	var manifestPath string
	if provider.Provider != "" {
		switch provider.Provider {
		// special handling for the mockprovider
		case "mockprovider":
			path = filepath.Join(TestutilsDir, "mockprovider/resources/mockprovider.lr")
		default:
			manifestPath = filepath.Join(TestutilsDir, "../../../providers/"+provider.Provider+"/resources/"+provider.Provider+".lr.manifest.yaml")
			path = filepath.Join(TestutilsDir, "../../../providers/"+provider.Provider+"/resources/"+provider.Provider+".lr")
		}
	} else if provider.Path != "" {
		path = provider.Path
	}

	res, err := lr.Resolve(path, func(path string) ([]byte, error) { return os.ReadFile(path) })
	if err != nil {
		panic(err.Error())
	}
	schema, err := lr.Schema(res)
	if err != nil {
		panic(err.Error())
	}
	// TODO: we should make a function that takes the Schema and the metadata and merges those.
	// Then we can use that in the LR code and the testutils code too
	if manifestPath != "" {
		// we will attempt to auto-detect the manifest to inject some metadata
		// into the schema
		raw, err := os.ReadFile(manifestPath)
		if err == nil {
			var lrDocsData docs.LrDocs
			err = yaml.Unmarshal(raw, &lrDocsData)
			if err == nil {
				docs.InjectMetadata(schema, &lrDocsData)
			}
		}
	}

	return schema
}

func Local() llx.Runtime {
	osSchema := MustLoadSchema(SchemaProvider{Provider: "os"})
	coreSchema := MustLoadSchema(SchemaProvider{Provider: "core"})
	networkSchema := MustLoadSchema(SchemaProvider{Provider: "network"})
	mockSchema := MustLoadSchema(SchemaProvider{Provider: "mockprovider"})

	schema := providers.Coordinator.Schema().(providers.ExtensibleSchema)
	schema.Add(providers.BuiltinCoreID, coreSchema)
	schema.Add(osconf.Config.Name, osSchema)
	schema.Add(networkconf.Config.Name, networkSchema)
	schema.Add(mockprovider.Config.Name, mockSchema)

	runtime := providers.Coordinator.NewRuntime()

	provider := &providers.RunningProvider{
		Name:   osconf.Config.Name,
		ID:     osconf.Config.ID,
		Plugin: osprovider.Init(),
		Schema: osSchema.Add(coreSchema),
	}
	runtime.Provider = &providers.ConnectedProvider{Instance: provider}
	runtime.AddConnectedProvider(runtime.Provider)

	provider = &providers.RunningProvider{
		Name:   networkconf.Config.Name,
		ID:     networkconf.Config.ID,
		Plugin: networkprovider.Init(),
		Schema: networkSchema,
	}
	runtime.AddConnectedProvider(&providers.ConnectedProvider{Instance: provider})

	provider = &providers.RunningProvider{
		Name:   mockprovider.Config.Name,
		ID:     mockprovider.Config.ID,
		Plugin: mockprovider.Init(),
		Schema: mockSchema,
	}
	runtime.AddConnectedProvider(&providers.ConnectedProvider{Instance: provider})

	// Since the testutils runtime is meant to be used with in-memory
	// providers only, we deactivate any type of discovery on the system.
	// This prevents us from accidentally pulling locally installed providers
	// which may not work with the current dependencies. The task of testing
	// those falls to an integration environment, not to unit tests.
	providers.Coordinator.DeactivateProviderDiscovery()

	return runtime
}

func mockRuntime(testdata string) llx.Runtime {
	return mockRuntimeAbs(filepath.Join(TestutilsDir, testdata))
}

func MockFromRecording(recording llx.Recording) llx.Runtime {
	runtime := Local().(*providers.Runtime)

	err := runtime.SetMockRecording(recording, runtime.Provider.Instance.ID, true)
	if err != nil {
		panic("failed to set recording: " + err.Error())
	}
	err = runtime.SetMockRecording(recording, networkconf.Config.ID, true)
	if err != nil {
		panic("failed to set recording: " + err.Error())
	}
	err = runtime.SetMockRecording(recording, mockprovider.Config.ID, true)
	if err != nil {
		panic("failed to set recording: " + err.Error())
	}

	return runtime
}

func mockRuntimeAbs(testdata string) llx.Runtime {
	abs, _ := filepath.Abs(testdata)
	recording, err := recording.LoadRecordingFile(abs)
	if err != nil {
		panic("failed to load recording: " + err.Error())
	}
	roRecording := recording.ReadOnly()

	return MockFromRecording(roRecording)
}

func RecordingFromAsset(a *inventory.Asset) llx.Recording {
	recording, err := recording.FromAsset(a)
	if err != nil {
		panic("failed to create recording from an asset: " + err.Error())
	}
	roRecording := recording.ReadOnly()

	return roRecording
}

func MockFromAsset(a *inventory.Asset) llx.Runtime {
	return MockFromRecording(RecordingFromAsset(a))
}

func LinuxMock() llx.Runtime {
	return mockRuntime("testdata/arch.json")
}

func KubeletMock() llx.Runtime {
	return mockRuntime("testdata/kubelet.json")
}

func KubeletAKSMock() llx.Runtime {
	return mockRuntime("testdata/kubelet-aks.json")
}

func KubeletEKSMock() llx.Runtime {
	return mockRuntime("testdata/kubelet-eks.json")
}

func WindowsMock() llx.Runtime {
	return mockRuntime("testdata/windows.json")
}

func RecordingMock(absTestdataPath string) llx.Runtime {
	return mockRuntimeAbs(absTestdataPath)
}

type SimpleTest struct {
	Code        string
	ResultIndex int
	Expectation interface{}
}

func (ctx *tester) TestSimple(t *testing.T, tests []SimpleTest) {
	t.Helper()
	for i := range tests {
		cur := tests[i]
		t.Run(cur.Code, func(t *testing.T) {
			res := ctx.TestQuery(t, cur.Code)
			assert.NotEmpty(t, res)

			if len(res) <= cur.ResultIndex {
				t.Error("insufficient results, looking for result idx " + strconv.Itoa(cur.ResultIndex))
				return
			}

			data := res[cur.ResultIndex].Data
			require.NoError(t, data.Error)
			assert.Equal(t, cur.Expectation, data.Value)
		})
	}
}

func (ctx *tester) TestNoErrorsNonEmpty(t *testing.T, tests []SimpleTest) {
	t.Helper()
	for i := range tests {
		cur := tests[i]
		t.Run(cur.Code, func(t *testing.T) {
			res := ctx.TestQuery(t, cur.Code)
			assert.NotEmpty(t, res)
		})
	}
}

func (ctx *tester) TestSimpleErrors(t *testing.T, tests []SimpleTest) {
	for i := range tests {
		cur := tests[i]
		t.Run(cur.Code, func(t *testing.T) {
			res := ctx.TestQuery(t, cur.Code)
			assert.NotEmpty(t, res)
			assert.Equal(t, cur.Expectation, res[cur.ResultIndex].Result().Error)
			assert.Nil(t, res[cur.ResultIndex].Data.Value)
		})
	}
}

func (ctx *tester) TestCompileErrors(t *testing.T, tests []SimpleTest) {
	for i := range tests {
		cur := tests[i]
		t.Run(cur.Code, func(t *testing.T) {
			res := ctx.TestQuery(t, cur.Code)
			assert.NotEmpty(t, res)
			assert.Equal(t, cur.Expectation, res[cur.ResultIndex].Result().Error)
			assert.Nil(t, res[cur.ResultIndex].Data.Value)
		})
	}
}

func TestNoResultErrors(t *testing.T, r []*llx.RawResult) bool {
	var found bool
	for i := range r {
		err := r[i].Data.Error
		if err != nil {
			t.Error("result has error: " + err.Error())
			found = true
		}
	}
	return found
}
