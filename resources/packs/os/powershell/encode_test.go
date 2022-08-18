package powershell_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.mondoo.io/mondoo/resources/packs/os/powershell"
)

func TestPowershellEncoding(t *testing.T) {
	expected := "powershell.exe -NoProfile -EncodedCommand JABQAHIAbwBnAHIAZQBzAHMAUAByAGUAZgBlAHIAZQBuAGMAZQA9ACcAUwBpAGwAZQBuAHQAbAB5AEMAbwBuAHQAaQBuAHUAZQAnADsAZABpAHIAIAAiAGMAOgBcAHAAcgBvAGcAcgBhAG0AIABmAGkAbABlAHMAIgAgAA=="
	cmd := string("dir \"c:\\program files\" ")
	assert.Equal(t, expected, powershell.Encode(cmd))
}
