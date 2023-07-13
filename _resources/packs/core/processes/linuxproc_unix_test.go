package processes

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseReadlink(t *testing.T) {
	fi, err := os.Open("./testdata/linux_readlink.txt")
	require.NoError(t, err)
	defer fi.Close()

	inode, err := readInodeFromOutput(fi)
	require.NoError(t, err)
	require.Equal(t, int64(41866700), inode)
}
