package awsiam

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCredentialReport(t *testing.T) {
	f, err := os.Open("./testdata/report.csv")
	require.NoError(t, err)

	entries, err := Parse(f)
	require.NoError(t, err)
	assert.Equal(t, 2, len(entries))

	assert.Equal(t, "<root_account>", entries[0]["user"])
}
