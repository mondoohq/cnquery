package custommount

import (
	"syscall"

	"github.com/rs/zerolog/log"
	"golang.org/x/sys/unix"
)

func Mount(attachedFS string, scanDir string, fsType string, opts string) error {
	if err := unix.Mount(attachedFS, scanDir, fsType, syscall.MS_MGC_VAL, opts); err != nil && err != unix.EBUSY {
		log.Error().Err(err).Msg("failed to mount dir")
		return err
	}
	return nil
}

func Unmount(scanDir string) error {
	if err := unix.Unmount(scanDir, unix.MNT_DETACH); err != nil && err != unix.EBUSY {
		log.Error().Err(err).Msg("failed to unmount dir")
		return err
	}
	return nil
}
