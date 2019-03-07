package parser

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPowershellEncoding(t *testing.T) {
	expected := "powershell.exe -EncodedCommand ZABpAHIAIAAiAGMAOgBcAHAAcgBvAGcAcgBhAG0AIABmAGkAbABlAHMAIgAgAA=="
	cmd := string("dir \"c:\\program files\" ")
	assert.Equal(t, expected, EncodePowershell(cmd))
}
