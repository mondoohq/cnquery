package windows

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/stretchr/testify/assert"
)

func TestWindowsRegistryKeyItemParser(t *testing.T) {
	r, err := os.Open("./testdata/registrykey.json")
	require.NoError(t, err)

	items, err := ParseRegistryKeyItems(r)
	assert.Nil(t, err)
	assert.Equal(t, 10, len(items))
	assert.Equal(t, "ConsentPromptBehaviorAdmin", items[0].Key)
	assert.Equal(t, 4, items[0].Value.Kind)
	assert.Equal(t, int64(5), items[0].Value.Number)
	assert.Equal(t, "5", items[0].GetValue())
}

func TestWindowsRegistryKeyChildParser(t *testing.T) {
	r, err := os.Open("./testdata/registrykey-children.json")
	require.NoError(t, err)

	items, err := ParseRegistryKeyChildren(r)
	assert.Nil(t, err)
	assert.Equal(t, 5, len(items))
}
