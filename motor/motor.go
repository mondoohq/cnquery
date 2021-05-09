package motor

import (
	"github.com/rs/zerolog/log"
	"go.mondoo.io/mondoo/motor/platform"
	"go.mondoo.io/mondoo/motor/transports"
	"go.mondoo.io/mondoo/motor/transports/events"
	"go.mondoo.io/mondoo/motor/transports/local"
	"go.mondoo.io/mondoo/motor/transports/mock"
	"sync"
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
var localTransportLock = &sync.Mutex{}
var localTransportDetector *platform.Detector

func New(trans transports.Transport, motorOpts ...MotorOption) (*Motor, error) {
	m := &Motor{
		Transport: trans,
	}

	for i := range motorOpts {
		motorOpts[i](m)
	}

	// set the detector after the opts have been applied to ensure its going via the recorder
	// if activated
	_, ok := m.Transport.(*local.LocalTransport)
	if ok && !m.isRecording {
		localTransportLock.Lock()
		if localTransportDetector == nil {
			localTransportDetector = platform.NewDetector(m.Transport)
		}
		m.detector = localTransportDetector
		localTransportLock.Unlock()
	} else {
		m.detector = platform.NewDetector(m.Transport)
	}

	return m, nil
}

type Motor struct {
	Transport   transports.Transport
	detector    *platform.Detector
	watcher     transports.Watcher
	Meta        MetaInfo
	isRecording bool
}

type MetaInfo struct {
	Name       string
	Identifier []string
	Labels     map[string]string
}

func (m *Motor) Platform() (*platform.Platform, error) {
	return m.detector.Platform()
}

func (m *Motor) Watcher() transports.Watcher {
	// create watcher once
	if m.watcher == nil {
		m.watcher = events.NewWatcher(m.Transport)
	}
	return m.watcher
}

func (m *Motor) ActivateRecorder() {
	if m.isRecording {
		return
	}

	mockT, _ := mock.NewRecordTransport(m.Transport)
	m.Transport = mockT
	m.isRecording = true
}

func (m *Motor) IsRecording() bool {
	return m.isRecording
}

// returns marshaled toml stucture
func (m *Motor) Recording() []byte {
	if m.isRecording {
		rt := m.Transport.(*mock.RecordTransport)
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
	if m.Transport != nil {
		m.Transport.Close()
	}
}

func (m *Motor) HasCapability(capability transports.Capability) bool {
	list := m.Transport.Capabilities()
	for i := range list {
		if list[i] == capability {
			return true
		}
	}
	return false
}

func (m *Motor) IsLocalTransport() bool {
	_, ok := m.Transport.(*local.LocalTransport)
	if !ok {
		return false
	}
	return true
}
