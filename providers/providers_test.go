// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package providers

import (
	"bytes"
	"context"
	"io"
	"sync"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/mql/v13/cli/config"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
)

type testPlugin struct {
	plugin.Service
}

func (t *testPlugin) Connect(req *plugin.ConnectReq, callback plugin.ProviderCallback) (*plugin.ConnectRes, error) {
	return nil, nil
}

func (t *testPlugin) MockConnect(req *plugin.ConnectReq, callback plugin.ProviderCallback) (*plugin.ConnectRes, error) {
	return nil, nil
}

func (t *testPlugin) ParseCLI(req *plugin.ParseCLIReq) (*plugin.ParseCLIRes, error) {
	return nil, nil
}

func (t *testPlugin) Shutdown(req *plugin.ShutdownReq) (*plugin.ShutdownRes, error) {
	// sleep more than the heartbeat interval to ensure that even if shutting down
	// the provider can still respond to heartbeats
	time.Sleep(10 * time.Second)
	return &plugin.ShutdownRes{}, nil
}

func (t *testPlugin) GetData(req *plugin.DataReq) (*plugin.DataRes, error) {
	return nil, nil
}

func (t *testPlugin) StoreData(req *plugin.StoreReq) (*plugin.StoreRes, error) {
	return nil, nil
}

func TestProviderShutdown(t *testing.T) {
	s := &RunningProvider{
		Plugin:      &testPlugin{},
		interval:    500 * time.Millisecond,
		gracePeriod: 500 * time.Millisecond,
	}
	hbtCtx, hbtCancel := context.WithCancel(context.Background())
	s.hbCancelFunc = hbtCancel
	err := s.heartbeat(hbtCtx, hbtCancel)
	require.NoError(t, err)
	require.False(t, s.isCloseOrShutdown())
	// the shutdown here takes 10 seconds, whereas the heartbeat interval is every second.
	// this means that this provider gets multiple heartbeats while shutting down
	err = s.Shutdown()
	require.NoError(t, err)
	require.True(t, s.isCloseOrShutdown())
}

// TestProvider_LoadResources_ConcurrentIsRaceFree exercises the lazy
// schema load path that callers like installDependencies and the
// coordinator now share. Run with -race to verify the per-Provider mutex
// inside LoadResources serializes the write to p.Schema and the reads
// after the call observe a single consistent schema value.
func TestProvider_LoadResources_ConcurrentIsRaceFree(t *testing.T) {
	origFs := config.AppFs
	config.AppFs = afero.NewMemMapFs()
	t.Cleanup(func() { config.AppFs = origFs })

	const providerDir = "/providers/test"
	const providerName = "test"
	const resourcesJSON = `{"resources":{"test.foo":{"id":"test.foo","name":"test.foo","fields":{}}}}`
	require.NoError(t, afero.WriteFile(
		config.AppFs,
		providerDir+"/"+providerName+".resources.json",
		[]byte(resourcesJSON),
		0o644,
	))

	p := &Provider{
		Provider: &plugin.Provider{Name: providerName},
		Path:     providerDir,
	}

	const goroutines = 32
	var (
		wg       sync.WaitGroup
		start    = make(chan struct{})
		errCount atomic.Int32
	)
	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			<-start
			if err := p.LoadResources(); err != nil {
				errCount.Add(1)
				t.Errorf("LoadResources: %v", err)
			}
		}()
	}
	close(start)
	wg.Wait()

	require.Zero(t, errCount.Load(), "no goroutine should fail")
	require.NotNil(t, p.Schema, "Schema must be populated after LoadResources")
	// AllResources is one of the methods callers invoke on the loaded
	// schema; calling it here lets the race detector flag any read that
	// isn't synchronized-after the (single) write inside the lock.
	require.NotEmpty(t, p.Schema.AllResources(), "schema resources must be parsed")
}

func TestOsRetry_RetryableError(t *testing.T) {
	funcCounter := 0
	testFunc := func() error {
		funcCounter++
		return syscall.EAGAIN
	}
	assert.NoError(t, osRetry(testFunc, 2))
	assert.Equal(t, 2, funcCounter)
}

func TestInstallIO(t *testing.T) {
	confJSON := []byte(`{"Name":"testp","Version":"1.2.3"}`)
	schemaJSON := []byte(`{"resources":{}}`)

	t.Run("schema-only archive", func(t *testing.T) {
		archive, err := buildTarXz(map[string][]byte{
			"testp.json":           confJSON,
			"testp.resources.json": schemaJSON,
		})
		require.NoError(t, err)

		installed, err := InstallIO(io.NopCloser(bytes.NewReader(archive)), InstallConf{Dst: t.TempDir()})
		require.NoError(t, err)
		require.Len(t, installed, 1)
		assert.Equal(t, "testp", installed[0].Name)
		assert.Equal(t, "1.2.3", installed[0].Version)
		assert.False(t, installed[0].HasBinary)
	})

	t.Run("archive with binary", func(t *testing.T) {
		archive, err := buildTarXz(map[string][]byte{
			"testp":                []byte(`fake-binary`),
			"testp.json":           confJSON,
			"testp.resources.json": schemaJSON,
		})
		require.NoError(t, err)

		installed, err := InstallIO(io.NopCloser(bytes.NewReader(archive)), InstallConf{Dst: t.TempDir()})
		require.NoError(t, err)
		require.Len(t, installed, 1)
		assert.Equal(t, "testp", installed[0].Name)
		assert.True(t, installed[0].HasBinary)
	})

	t.Run("archive without config errors", func(t *testing.T) {
		archive, err := buildTarXz(map[string][]byte{
			"testp.resources.json": schemaJSON,
		})
		require.NoError(t, err)

		_, err = InstallIO(io.NopCloser(bytes.NewReader(archive)), InstallConf{Dst: t.TempDir()})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "cannot find testp.json")
	})
}
