package resources_test

import (
	"bytes"
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/client-go/util/jsonpath"
	"sigs.k8s.io/yaml"
)

func TestResource_Yaml(t *testing.T) {
	t.Run("extract content", func(t *testing.T) {
		res := testQuery(t, "yaml.path(filepath: '/root/pod.yaml', jsonpath: '{.kind}').result")
		assert.NotEmpty(t, res)
		assert.Empty(t, res[0].Result().Error)
		assert.Equal(t, "Pod", res[0].Data.Value)
	})
}

func TestJsonPath(t *testing.T) {
	mContent, err := ioutil.ReadFile("./testdata/pod.yaml")
	require.NoError(t, err)

	// load data
	var jsonInterface interface{}
	err = yaml.Unmarshal(mContent, &jsonInterface)
	require.NoError(t, err)

	// parse json path expression
	j := jsonpath.New("jsonpath")
	j.AllowMissingKeys(false)
	err = j.Parse("{.kind}")
	require.NoError(t, err)

	buf := new(bytes.Buffer)
	err = j.Execute(buf, jsonInterface)
	require.NoError(t, err)
	jsonpathResult := buf.String()
	assert.Equal(t, "Pod", jsonpathResult)
}
