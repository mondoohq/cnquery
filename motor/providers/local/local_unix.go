//go:build !windows
// +build !windows

package local

import (
	"os"
	"syscall"

	pos "go.mondoo.com/cnquery/motor/providers/os"
)

func (p *Provider) fileowner(stat os.FileInfo) (int64, int64) {
	uid := int64(-1)
	gid := int64(-1)
	if p.Sudo != nil {
		if stat, ok := stat.Sys().(*pos.FileInfo); ok {
			uid = stat.Uid
			gid = stat.Gid
		}
	}
	if stat, ok := stat.Sys().(*syscall.Stat_t); ok {
		uid = int64(stat.Uid)
		gid = int64(stat.Gid)
	}
	return uid, gid
}
