package motor

import (
	"sync"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/motor/asset"
	"go.mondoo.com/cnquery/motor/platform"
	"go.mondoo.com/cnquery/motor/platform/detector"
	"go.mondoo.com/cnquery/motor/providers"
	"go.mondoo.com/cnquery/motor/providers/local"
	"go.mondoo.com/cnquery/motor/providers/mock"
	"go.mondoo.com/cnquery/motor/providers/os"
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
	localProviderLock     = &sync.Mutex{}
	localProviderDetector *detector.Detector
)

func New(provider providers.Instance, motorOpts ...MotorOption) (*Motor, error) {
	m := &Motor{
		Provider: provider,
	}

	for i := range motorOpts {
		motorOpts[i](m)
	}

	// set the detector after the opts have been applied to ensure its going via the recorder
	// if activated
	_, ok := m.Provider.(*local.Provider)
	if ok && !m.isRecording {
		localProviderLock.Lock()
		if localProviderDetector == nil {
			localProviderDetector = detector.New(m.Provider)
		}
		m.detector = localProviderDetector
		localProviderLock.Unlock()
	} else {
		m.detector = detector.New(m.Provider)
	}

	return m, nil
}

type Motor struct {
	l sync.Mutex

	Provider    providers.Instance
	asset       *asset.Asset
	detector    *detector.Detector
	watcher     providers.Watcher
	isRecording bool
	isClosed    bool
}

func (m *Motor) Platform() (*platform.Platform, error) {
	m.l.Lock()
	defer m.l.Unlock()
	return m.detector.Platform()
}

func (m *Motor) Watcher() providers.Watcher {
	m.l.Lock()
	defer m.l.Unlock()

	osProvider, isOSprovider := m.Provider.(os.OperatingSystemProvider)
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

	osProvider, isOSprovider := m.Provider.(os.OperatingSystemProvider)
	if !isOSprovider {
		return
	}

	mockT, _ := mock.NewRecordProvider(osProvider)
	m.Provider = mockT
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

	if m.isRecording {
		rt := m.Provider.(*mock.MockRecordProvider)
		data, err := rt.ExportData()
		if err != nil {
			log.Error().Err(err).Msg("could not export data")
			return nil
		}
		return data
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

	_, ok := m.Provider.(*local.Provider)
	if !ok {
		return false
	}
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
