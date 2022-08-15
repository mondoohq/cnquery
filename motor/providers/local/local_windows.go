//go:build windows
// +build windows

package local

import "os"

func (p *Provider) fileowner(stat os.FileInfo) (int64, int64) {
	uid := int64(-1)
	gid := int64(-1)

	return uid, gid
}
