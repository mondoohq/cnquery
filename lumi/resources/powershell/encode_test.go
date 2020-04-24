package powershell_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.mondoo.io/mondoo/lumi/resources/powershell"
)

func TestPowershellEncoding(t *testing.T) {
	expected := "powershell.exe -EncodedCommand ZABpAHIAIAAiAGMAOgBcAHAAcgBvAGcAcgBhAG0AIABmAGkAbABlAHMAIgAgAA=="
	cmd := string("dir \"c:\\program files\" ")
	assert.Equal(t, expected, powershell.Encode(cmd))
}
