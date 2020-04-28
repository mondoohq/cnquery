package motor

import (
	"errors"

	"go.mondoo.io/mondoo/motor/motoros/capabilities"
	"go.mondoo.io/mondoo/motor/motoros/events"
	"go.mondoo.io/mondoo/motor/motoros/local"
	"go.mondoo.io/mondoo/motor/motoros/platform"
	"go.mondoo.io/mondoo/motor/motoros/types"
)

func New(trans types.Transport) (*Motor, error) {
	c := &Motor{Transport: trans}
	return c, nil
}

type Motor struct {
	Transport types.Transport
	platform  *platform.PlatformInfo
	watcher   types.Watcher
	Meta      MetaInfo
}

type MetaInfo struct {
	Name       string
	Identifier []string
	Labels     map[string]string
}

func (m *Motor) Platform() (platform.PlatformInfo, error) {
	// check if platform is in cache
	if m.platform != nil {
		return *m.platform, nil
	}

	detector := &platform.Detector{Transport: m.Transport}
	resolved, di := detector.Resolve()
	if !resolved {
		return platform.PlatformInfo{}, errors.New("could not determine operating system")
	} else {
		// cache value
		m.platform = di
	}
	return *di, nil
}

func (m *Motor) Watcher() types.Watcher {
	// create watcher once
	if m.watcher == nil {
		m.watcher = events.NewWatcher(m.Transport)
	}
	return m.watcher
}

func (m *Motor) Close() {
	if m == nil {
		return
	}
	if m.Transport != nil {
		m.Transport.Close()
	}
}

func (m *Motor) HasCapability(capability capabilities.Capability) bool {
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
