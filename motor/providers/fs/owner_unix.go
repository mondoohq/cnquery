//go:build !windows
// +build !windows

package fs

import (
	"os"
	"syscall"
)

func (t *Provider) fileowner(stat os.FileInfo) (int64, int64) {
	uid := int64(-1)
	gid := int64(-1)
	if stat, ok := stat.Sys().(*syscall.Stat_t); ok {
		uid = int64(stat.Uid)
		gid = int64(stat.Gid)
	}
	return uid, gid
}
