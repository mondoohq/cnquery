package testutils

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnquery"
	"go.mondoo.com/cnquery/llx"
	"go.mondoo.com/cnquery/logger"
	"go.mondoo.com/cnquery/mql"
	"go.mondoo.com/cnquery/mqlc"
	"go.mondoo.com/cnquery/providers"
	"go.mondoo.com/cnquery/providers/mock"
	osconf "go.mondoo.com/cnquery/providers/os/config"
	osprovider "go.mondoo.com/cnquery/providers/os/provider"
)

var Features cnquery.Features

func init() {
	logger.InitTestEnv()
	Features = getEnvFeatures()
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

func TomlMock(filepath string) llx.Runtime {
	trans, err := mock.NewFromTomlFile(filepath)
	if err != nil {
		panic(err.Error())
	}

	return trans
}

type tester struct {
	Runtime llx.Runtime
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

func Local(pathToTestutils string) llx.Runtime {
	raw, err := os.ReadFile(filepath.Join(pathToTestutils, "../../../providers/os/resources/os.resources.json"))
	if err != nil {
		panic("failed to load os resources for testing: " + err.Error())
	}
	osSchema := providers.MustLoadSchema("os", raw)

	raw, err = os.ReadFile(filepath.Join(pathToTestutils, "../../../providers/core/resources/core.resources.json"))
	if err != nil {
		panic("failed to load core resources for testing: " + err.Error())
	}
	coreSchema := providers.MustLoadSchema("core", raw)

	raw, err = os.ReadFile(filepath.Join(pathToTestutils, "../../../providers/network/resources/network.resources.json"))
	if err != nil {
		panic("failed to load network resources for testing: " + err.Error())
	}
	networkSchema := providers.MustLoadSchema("network", raw)

	provider := &providers.RunningProvider{
		Name:   osconf.Config.Name,
		ID:     osconf.Config.ID,
		Plugin: osprovider.Init(),
		Schema: osSchema.Add(coreSchema).Add(networkSchema),
	}

	runtime := providers.DefaultRuntime()
	runtime.Provider = &providers.ConnectedProvider{Instance: provider}
	runtime.AddConnectedProvider(runtime.Provider)

	return runtime
}

func mockRuntime(pathToTestutils string, testdata string) llx.Runtime {
	runtime := Local(pathToTestutils).(*providers.Runtime)

	abs, _ := filepath.Abs(filepath.Join(pathToTestutils, testdata))
	recording, err := providers.LoadRecordingFile(abs)
	if err != nil {
		panic("failed to load recording: " + err.Error())
	}

	err = runtime.SetRecording(recording, runtime.Provider.Instance.ID, true, true)
	if err != nil {
		panic("failed to set recording: " + err.Error())
	}

	return runtime
}

func LinuxMock(pathToTestutils string) llx.Runtime {
	return mockRuntime(pathToTestutils, "testdata/arch.json")
}

func KubeletMock() llx.Runtime {
	return TomlMock("./k8s/testdata/kubelet.toml")
}

func KubeletAKSMock() llx.Runtime {
	return TomlMock("./k8s/testdata/kubelet-aks.toml")
}

func WindowsMock() llx.Runtime {
	return TomlMock("./testdata/windows.toml")
}

func CustomMock(path string) llx.Runtime {
	return TomlMock(path)
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
