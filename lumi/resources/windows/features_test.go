package windows

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWindowsFeatures(t *testing.T) {
	data, err := os.Open("./testdata/features.json")
	if err != nil {
		t.Fatal(err)
	}

	items, err := ParseWindowsFeatures(data)
	assert.Nil(t, err)
	assert.Equal(t, 5, len(items))
}
