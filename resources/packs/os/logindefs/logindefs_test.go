package logindefs_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnquery/motor/providers/mock"
	"go.mondoo.com/cnquery/resources/packs/os/logindefs"
)

func TestLoginDefsParser(t *testing.T) {
	mock, err := mock.NewFromTomlFile("./testdata/debian.toml")
	require.NoError(t, err)

	f, err := mock.FS().Open("/etc/login.defs")
	require.NoError(t, err)
	defer f.Close()

	entries := logindefs.Parse(f)

	assert.Equal(t, "tty", entries["TTYGROUP"])
	assert.Equal(t, "PATH=/usr/local/bin:/usr/bin:/bin:/usr/local/games:/usr/games", entries["ENV_PATH"])

	_, ok := entries["SHA_CRYPT_MIN_ROUNDS"]
	assert.False(t, ok)
}
