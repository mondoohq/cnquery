package motor

import (
	"errors"

	"go.mondoo.io/mondoo/motor/events"
	"go.mondoo.io/mondoo/motor/platform"
	"go.mondoo.io/mondoo/motor/types"
)

func New(trans types.Transport) (*Motor, error) {
	c := &Motor{Transport: trans}
	return c, nil
}

type Motor struct {
	Transport types.Transport
	platform  *platform.Info
	watcher   types.Watcher
}

func (m *Motor) Platform() (platform.Info, error) {
	// check if platform is in cache
	if m.platform != nil {
		return *m.platform, nil
	}

	detector := &platform.Detector{Transport: m.Transport}
	resolved, di := detector.Resolve()
	if !resolved {
		return platform.Info{}, errors.New("could not determine operating system")
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
