package arista

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tj/assert"
)

func TestConfigParser(t *testing.T) {
	f, err := os.Open("./testdata/config")
	require.NoError(t, err)
	defer f.Close()

	dict := ParseConfig(f)
	assert.NotNil(t, dict)

	data, err := json.Marshal(dict)
	require.NoError(t, err)
	fmt.Printf("%v\n", string(data))

	entry := dict["management telnet"].(map[string]interface{})
	assert.NotNil(t, entry)

	_, ok := entry["shutdown"]
	assert.True(t, ok)
}

func TestGetSection(t *testing.T) {
	f, err := os.Open("./testdata/config")
	require.NoError(t, err)
	defer f.Close()

	data, err := ioutil.ReadAll(f)
	require.NoError(t, err)

	section := GetSection(bytes.NewReader(data), "cvx service openstack")
	expected := "shutdown\ngrace-period 60\nnetwork type-driver vlan default\nname-resolution interval 21600\n"
	assert.Equal(t, expected, section)

	section = GetSection(bytes.NewReader(data), "management telnet")
	expected = "shutdown\nidle-timeout 0\nsession-limit 20\nsession-limit per-host 20\n"
	assert.Equal(t, expected, section)
}
