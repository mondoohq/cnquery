package testutils

import (
	"errors"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tj/assert"
	"go.mondoo.com/cnquery/llx"
	"go.mondoo.com/cnquery/logger"
	"go.mondoo.com/cnquery/motor"
	"go.mondoo.com/cnquery/motor/providers/local"
	"go.mondoo.com/cnquery/motor/providers/mock"
	"go.mondoo.com/cnquery/mqlc"
	"go.mondoo.com/cnquery/policy"
	"go.mondoo.com/cnquery/policy/executor"
	"go.mondoo.com/cnquery/resources"
	"go.mondoo.io/mondoo"
)

var Features mondoo.Features

func init() {
	logger.InitTestEnv()
	Features = getEnvFeatures()
}

func getEnvFeatures() mondoo.Features {
	env := os.Getenv("FEATURES")
	if env == "" {
		return mondoo.Features{byte(mondoo.PiperCode)}
	}

	arr := strings.Split(env, ",")
	var fts mondoo.Features
	for i := range arr {
		v, ok := mondoo.FeaturesValue[arr[i]]
		if ok {
			fmt.Println("--> activate feature: " + arr[i])
			fts = append(Features, byte(v))
		} else {
			panic("cannot find requested feature: " + arr[i])
		}
	}
	return fts
}

func OnlyV1(t *testing.T) {
	t.Helper()
	if Features.IsActive(mondoo.PiperCode) {
		t.SkipNow()
	}
}

func OnlyPiper(t *testing.T) {
	t.Helper()
	if !Features.IsActive(mondoo.PiperCode) {
		t.SkipNow()
	}
}

func mockTransport(filepath string) (*motor.Motor, error) {
	trans, err := mock.NewFromTomlFile(filepath)
	if err != nil {
		panic(err.Error())
	}

	return motor.New(trans)
}

type tester struct {
	runtime *resources.Runtime
}

func InitTester(motor *motor.Motor, registry *resources.Registry) *tester {
	return &tester{
		runtime: resources.NewRuntime(registry, motor),
	}
}

func (ctx *tester) Compile(query string) (*llx.CodeBundle, error) {
	return mqlc.Compile(query, ctx.runtime.Registry.Schema(), Features, nil)
}

func (ctx *tester) TestQueryP(t *testing.T, query string, props map[string]*llx.Primitive) []*llx.RawResult {
	t.Helper()
	bundle, err := mqlc.Compile(query, ctx.runtime.Registry.Schema(), Features, props)
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

	score, resultMap, err := executor.ExecuteQuery(ctx.runtime.Registry.Schema(), ctx.runtime, bundle, props, Features)
	require.NoError(t, err)

	results := make([]*llx.RawResult, 0, len(resultMap)+1)
	i := 0

	if Features.IsActive(mondoo.PiperCode) {
		refs := make([]uint64, 0, len(bundle.CodeV2.Checksums))
		for _, datapointArr := range [][]uint64{bundle.CodeV2.Datapoints(), bundle.CodeV2.Entrypoints()} {
			for _, v := range datapointArr {
				refs = append(refs, v)
			}
		}

		sort.Slice(refs, func(i, j int) bool {
			return refs[i] < refs[j]
		})

		for _, ref := range refs {
			checksum := bundle.CodeV2.Checksums[ref]
			if d, ok := resultMap[checksum]; ok {
				results = append(results, d)
				i++
			}
		}

	} else {
		refs := make([]int, 0, len(bundle.DeprecatedV5Code.Checksums))
		for _, datapointArr := range [][]int32{bundle.DeprecatedV5Code.Datapoints, bundle.DeprecatedV5Code.Entrypoints} {
			for _, v := range datapointArr {
				refs = append(refs, int(v))
			}
		}

		sort.Ints(refs)

		for _, ref := range refs {
			checksum := bundle.DeprecatedV5Code.Checksums[int32(ref)]
			if d, ok := resultMap[checksum]; ok {
				results = append(results, d)
				i++
			}
		}
	}

	success := score.Value == 100
	queryResult := &llx.RawResult{
		CodeID: score.QrId,
	}
	if score.Type == policy.ScoreType_Result {
		queryResult.Data = llx.BoolData(success)
	} else if score.Type == policy.ScoreType_Error {
		queryResult.Data = &llx.RawData{
			Error: errors.New(score.Message),
		}
	} else if score.Type == policy.ScoreType_Skip {
		queryResult.Data = llx.NilData
	}

	results = append(results, queryResult)

	return results
}

func Local() *motor.Motor {
	transport, err := local.New()
	if err != nil {
		panic(err.Error())
	}

	m, err := motor.New(transport)
	if err != nil {
		panic(err.Error())
	}

	return m
}

func Mock(path string) *motor.Motor {
	m, err := mockTransport(path)
	if err != nil {
		panic(err.Error())
	}

	return m
}

func LinuxMock() *motor.Motor {
	return Mock("../testdata/arch.toml")
}

func WindowsMock() *motor.Motor {
	return Mock("../testdata/windows.toml")
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
