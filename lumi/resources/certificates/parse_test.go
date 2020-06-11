package certificates

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"gotest.tools/assert"
)

func TestCertificates(t *testing.T) {
	file := "./testdata/ca-bundle.crt"

	f, err := os.Open(file)
	require.NoError(t, err)

	certs, err := ParseCertFromPEM(f)
	require.NoError(t, err)

	assert.Equal(t, 17, len(certs))
}
