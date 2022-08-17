package motor

import (
	"sync"

	"github.com/rs/zerolog/log"
	"go.mondoo.io/mondoo/motor/asset"
	"go.mondoo.io/mondoo/motor/platform"
	"go.mondoo.io/mondoo/motor/platform/detector"
	"go.mondoo.io/mondoo/motor/providers"
	"go.mondoo.io/mondoo/motor/providers/local"
	"go.mondoo.io/mondoo/motor/providers/mock"
	"go.mondoo.io/mondoo/motor/providers/os/events"
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
	localTransportLock     = &sync.Mutex{}
	localTransportDetector *detector.Detector
)

func New(trans providers.Transport, motorOpts ...MotorOption) (*Motor, error) {
	m := &Motor{
		Transport: trans,
	}

	for i := range motorOpts {
		motorOpts[i](m)
	}

	// set the detector after the opts have been applied to ensure its going via the recorder
	// if activated
	_, ok := m.Transport.(*local.Provider)
	if ok && !m.isRecording {
		localTransportLock.Lock()
		if localTransportDetector == nil {
			localTransportDetector = detector.New(m.Transport)
		}
		m.detector = localTransportDetector
		localTransportLock.Unlock()
	} else {
		m.detector = detector.New(m.Transport)
	}

	return m, nil
}

type Motor struct {
	l sync.Mutex

	Transport   providers.Transport
	asset       *asset.Asset
	detector    *detector.Detector
	watcher     providers.Watcher
	isRecording bool
}

func (m *Motor) Platform() (*platform.Platform, error) {
	m.l.Lock()
	defer m.l.Unlock()
	return m.detector.Platform()
}

func (m *Motor) Watcher() providers.Watcher {
	m.l.Lock()
	defer m.l.Unlock()

	// create watcher once
	if m.watcher == nil {
		m.watcher = events.NewWatcher(m.Transport)
	}
	return m.watcher
}

func (m *Motor) ActivateRecorder() {
	m.l.Lock()
	defer m.l.Unlock()

	if m.isRecording {
		return
	}

	mockT, _ := mock.NewRecordProvider(m.Transport)
	m.Transport = mockT
	m.isRecording = true
}

func (m *Motor) IsRecording() bool {
	m.l.Lock()
	defer m.l.Unlock()

	return m.isRecording
}

// returns marshaled toml stucture
func (m *Motor) Recording() []byte {
	m.l.Lock()
	defer m.l.Unlock()

	if m.isRecording {
		rt := m.Transport.(*mock.MockRecordProvider)
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

	if m.Transport != nil {
		m.Transport.Close()
	}
	if m.watcher != nil {
		if err := m.watcher.TearDown(); err != nil {
			log.Warn().Err(err).Msg("failed to tear down watcher")
		}
	}
}

func (m *Motor) HasCapability(capability providers.Capability) bool {
	m.l.Lock()
	defer m.l.Unlock()

	list := m.Transport.Capabilities()
	for i := range list {
		if list[i] == capability {
			return true
		}
	}
	return false
}

func (m *Motor) IsLocalTransport() bool {
	m.l.Lock()
	defer m.l.Unlock()

	_, ok := m.Transport.(*local.Provider)
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
