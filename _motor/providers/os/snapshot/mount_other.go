//go:build !linux
// +build !linux

package snapshot

import "errors"

func Mount(attachedFS string, scanDir string, fsType string, opts []string) error {
	return errors.New("unsupported platform")
}

func Unmount(scanDir string) error {
	return errors.New("unsupported platform")
}
