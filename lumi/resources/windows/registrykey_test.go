package windows

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWindowsRegistryKeyItemParser(t *testing.T) {
	data, err := os.Open("./testdata/registrykey.json")
	if err != nil {
		t.Fatal(err)
	}

	items, err := ParseRegistryKeyItems(data)
	assert.Nil(t, err)
	assert.Equal(t, 10, len(items))
	assert.Equal(t, "ConsentPromptBehaviorAdmin", items[0].Key)
	assert.Equal(t, 4, items[0].Value.Kind)
	assert.Equal(t, int64(5), items[0].Value.Number)
	assert.Equal(t, "5", items[0].GetValue())
}

func TestWindowsRegistryKeyChildParser(t *testing.T) {
	data, err := os.Open("./testdata/registrykey-children.json")
	if err != nil {
		t.Fatal(err)
	}

	items, err := ParseRegistryKeyChildren(data)
	assert.Nil(t, err)
	assert.Equal(t, 5, len(items))
}
