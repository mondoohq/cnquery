package windows

import (
	"github.com/stretchr/testify/assert"
	"strings"
	"testing"
)

// UBR - Update Build Revision
func TestParseWinRegistryCurrentVersion(t *testing.T) {

	data := `{
    "CurrentBuild":  "17763",
		"UBR":  720,
		"EditionID": "ServerDatacenterEval",
		"ReleaseId": "1809"
	}`

	m, err := ParseWinRegistryCurrentVersion(strings.NewReader(data))
	assert.Nil(t, err)

	assert.Equal(t, "17763", m.CurrentBuild, "buildnumber should be parsed properly")
	assert.Equal(t, 720, m.UBR, "ubr should be parsed properly")

}
