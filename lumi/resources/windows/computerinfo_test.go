package windows

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWindowsComputerInfo(t *testing.T) {
	data, err := os.Open("./testdata/computer-info.json")
	if err != nil {
		t.Fatal(err)
	}

	items, err := ParseComputerInfo(data)
	assert.Nil(t, err)
	assert.Equal(t, 43, len(items))
}
