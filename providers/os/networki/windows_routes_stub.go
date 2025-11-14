//go:build windows

package networki

import (
	"github.com/cockroachdb/errors"
)

// Here we are stubbing out the Darwin and Linux route detection for windows builds
// to avoid compile errors because golang.org/x/sys/unix and golang.org/x/net/route and is excluded on Windows.
func (n *neti) detectDarwinRoutes() ([]Route, error) {
	return nil, errors.New("Darwin route detection is not available on Linux builds")
}

func (n *neti) detectLinuxRoutes() ([]Route, error) {
	return nil, errors.New("Linux route detection is not available on Windows builds")
}
