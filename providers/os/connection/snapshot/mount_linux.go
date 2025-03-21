// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package snapshot

import (
	"errors"
	"strings"

	"github.com/moby/sys/mount"
	"github.com/rs/zerolog/log"
	"golang.org/x/sys/unix"
)

func Mount(attachedFS string, scanDir string, fsType string, opts []string) error {
	if err := mount.Mount(attachedFS, scanDir, fsType, strings.Join(opts, ",")); err != nil && errors.Unwrap(err) != unix.EBUSY {
		log.Error().Err(err).Str("attached-fs", attachedFS).Str("scan-dir", scanDir).Str("fs-type", fsType).Str("opts", strings.Join(opts, ",")).Msg("failed to mount dir")
		return err
	}
	return nil
}

func Unmount(scanDir string) error {
	if err := mount.Unmount(scanDir); err != nil && errors.Unwrap(err) != unix.EBUSY {
		return err
	}
	return nil
}
