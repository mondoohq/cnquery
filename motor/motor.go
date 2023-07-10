package motor

import (
	"os"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/motor/asset"
	"go.mondoo.com/cnquery/motor/platform"
	"go.mondoo.com/cnquery/motor/providers"
	os_provider "go.mondoo.com/cnquery/motor/providers/os"
	"go.mondoo.com/cnquery/motor/providers/os/events"
)

type MotorOption func(m *Motor)

func WithRecoding(record bool) MotorOption {
	return func(m *Motor) {
		if record {
			m.ActivateRecorder()
		}
	}
}

// implement special case for local platform to speed things up, this is especially important on windows where
// powershell calls are pretty expensive and slow
var (
	localProviderLock = &sync.Mutex{}
)

func New(provider providers.Instance, motorOpts ...MotorOption) (*Motor, error) {
	panic("marked for deletion")
	return nil, nil
}

type Motor struct {
	l sync.Mutex

	Provider    providers.Instance
	asset       *asset.Asset
	watcher     providers.Watcher
	isRecording bool
	isClosed    bool
}

func (m *Motor) Platform() (*platform.Platform, error) {
	m.l.Lock()
	defer m.l.Unlock()
	return nil, nil
}

func (m *Motor) Watcher() providers.Watcher {
	m.l.Lock()
	defer m.l.Unlock()

	osProvider, isOSprovider := m.Provider.(os_provider.OperatingSystemProvider)
	if !isOSprovider {
		return nil
	}

	// create watcher once
	if m.watcher == nil {
		m.watcher = events.NewWatcher(osProvider)
	}
	return m.watcher
}

func (m *Motor) ActivateRecorder() {
	m.l.Lock()
	defer m.l.Unlock()

	if m.isRecording {
		return
	}

	m.isRecording = true
}

func (m *Motor) IsRecording() bool {
	m.l.Lock()
	defer m.l.Unlock()

	return m.isRecording
}

// returns marshaled toml structure
func (m *Motor) Recording() []byte {
	m.l.Lock()
	defer m.l.Unlock()
	return nil
}

// StoreRecording stores tracked commands and files into the recording file
// If no filename is provided, it generates a filename
func (m *Motor) StoreRecording(filename string) error {
	if m.IsRecording() {
		if filename == "" {
			filename = "recording-" + time.Now().Format("20060102150405") + ".toml"
		}
		log.Info().Str("filename", filename).Msg("store recording")
		data := m.Recording()
		return os.WriteFile(filename, data, 0o700)
	}
	return nil
}

func (m *Motor) Close() {
	if m == nil {
		return
	}
	m.l.Lock()
	defer m.l.Unlock()
	if m.isClosed {
		return
	}
	m.isClosed = true

	if m.Provider != nil {
		m.Provider.Close()
	}
	if m.watcher != nil {
		if err := m.watcher.TearDown(); err != nil {
			log.Warn().Err(err).Msg("failed to tear down watcher")
		}
	}
}

func (m *Motor) IsLocalProvider() bool {
	m.l.Lock()
	defer m.l.Unlock()
	return true
}

// SetAsset sets the asset that this Motor was created for
func (m *Motor) SetAsset(a *asset.Asset) {
	m.l.Lock()
	defer m.l.Unlock()
	m.asset = a
}

// GetAsset returns the asset that this motor was created for.
// The caller must check that the return value is not nil before
// using
func (m *Motor) GetAsset() *asset.Asset {
	m.l.Lock()
	defer m.l.Unlock()
	return m.asset
}
