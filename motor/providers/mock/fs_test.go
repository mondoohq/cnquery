package mock_test

import (
	"io/ioutil"
	"path/filepath"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnquery/motor/providers/mock"
)

func TestGlobCommand(t *testing.T) {
	filepath, _ := filepath.Abs("./testdata/mock.toml")
	p, err := mock.NewFromTomlFile(filepath)
	assert.Equal(t, nil, err, "should create mock without error")

	filesystem := p.Fs
	matches, err := filesystem.Glob("*ssh/*_config")
	require.NoError(t, err)

	assert.True(t, len(matches) == 1)
	assert.Contains(t, matches, "/etc/ssh/sshd_config")
}

func TestLoadFile(t *testing.T) {
	filepath, _ := filepath.Abs("./testdata/mock.toml")
	p, err := mock.NewFromTomlFile(filepath)
	assert.Equal(t, nil, err, "should create mock without error")

	f, err := p.FS().Open("/etc/os-release")
	require.NoError(t, err)

	data, err := ioutil.ReadAll(f)
	require.NoError(t, err)

	assert.Equal(t, 382, len(data))
}

func TestReadDirnames(t *testing.T) {
	filepath, _ := filepath.Abs("./testdata/mock.toml")
	p, err := mock.NewFromTomlFile(filepath)
	require.NoError(t, err)

	dir, err := p.FS().Open("/sys/class/dmi/id")
	require.NoError(t, err)
	stat, err := dir.Stat()
	require.NoError(t, err)
	assert.True(t, stat.IsDir())

	names, err := dir.Readdirnames(100)
	require.NoError(t, err)

	assert.Equal(t, 2, len(names))
	assert.Contains(t, names, "bios_vendor")
	assert.Contains(t, names, "bios_date")
}

func TestConcurrent(t *testing.T) {
	wg := sync.WaitGroup{}
	filepath, _ := filepath.Abs("./testdata/mock.toml")
	p, err := mock.NewFromTomlFile(filepath)
	require.NoError(t, err)

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for _, f := range []string{
				"/etc/os-release",
				"/etc/ssh/sshd_config",
				"/sys/class/dmi/id/bios_date",
				"/sys/class/dmi/id/bios_vendor",
			} {

				_, err := p.FS().Open(f)
				if err != nil {
					t.Errorf("unexpected error in Open: %v", err)
					return
				}

				err = p.FS().Rename(f, f+".new")
				if err != nil {
					t.Errorf("unexpected error in Rename: %v", err)
					return
				}
			}
		}()
	}
	wg.Wait()
}
