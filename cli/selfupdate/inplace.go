// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package selfupdate

import (
	"context"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/rs/zerolog/log"
)

// verifyBinary runs "<binary> version" with auto-update disabled and a 5-second
// timeout. It returns an error if the binary fails to execute or exits non-zero.
// This is called BEFORE any rename so the original binary is never touched if
// the staged binary is broken.
func verifyBinary(binaryPath string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, binaryPath, "version")
	cmd.Env = append(os.Environ(), EnvAutoUpdate+"=false")

	if err := cmd.Run(); err != nil {
		return errors.Wrap(err, "staged binary failed verification")
	}
	return nil
}

// swapBinaryInPlace replaces the currently running binary with the staged binary.
// It renames the running executable to <name>.old, then copies the staged binary
// to the original path. If the copy fails, the rename is rolled back.
// Returns the original executable path for re-exec.
func swapBinaryInPlace(stagedBinaryPath string) (string, error) {
	originalPath, err := os.Executable()
	if err != nil {
		return "", errors.Wrap(err, "failed to get current executable path")
	}
	originalPath, err = filepath.EvalSymlinks(originalPath)
	if err != nil {
		return "", errors.Wrap(err, "failed to resolve executable symlinks")
	}

	return swapBinaryInPlaceFrom(stagedBinaryPath, originalPath)
}

// swapBinaryInPlaceFrom is the testable core of swapBinaryInPlace. It takes
// the resolved original path explicitly instead of calling os.Executable().
func swapBinaryInPlaceFrom(stagedBinaryPath, originalPath string) (string, error) {
	// If already running from the same path, no swap needed.
	stagedAbs, err := filepath.Abs(stagedBinaryPath)
	if err == nil {
		if resolved, err2 := filepath.EvalSymlinks(stagedAbs); err2 == nil {
			stagedAbs = resolved
		}
		if stagedAbs == originalPath {
			return originalPath, nil
		}
	}

	oldPath := originalPath + ".old"

	// On Windows os.Rename fails if the destination already exists. Remove a
	// leftover .old file from a previous swap that wasn't cleaned up (e.g.,
	// crash, or file was still locked at cleanup time).
	if err := os.Remove(oldPath); err != nil && !os.IsNotExist(err) {
		return "", errors.Wrap(err, "failed to remove leftover .old binary")
	}

	// Rename the running binary out of the way (safe on Windows for a running exe).
	if err := os.Rename(originalPath, oldPath); err != nil {
		return "", errors.Wrap(err, "failed to rename running binary to .old")
	}

	// Copy the staged binary to the original path (cross-volume safe).
	if err := copyFile(stagedBinaryPath, originalPath); err != nil {
		// Roll back: move .old back to original.
		if rbErr := os.Rename(oldPath, originalPath); rbErr != nil {
			log.Error().Err(rbErr).Msg("in-place swap: rollback failed, manual recovery may be needed")
		}
		return "", errors.Wrap(err, "failed to copy staged binary to original path")
	}

	log.Debug().
		Str("original", originalPath).
		Str("staged", stagedBinaryPath).
		Msg("in-place swap: binary replaced successfully")

	return originalPath, nil
}

// CleanupOldBinary removes a leftover <exe>.old file from a previous in-place
// update. This is best-effort: on Windows the file may still be locked by the
// previous process, so failures are logged and ignored.
func CleanupOldBinary() {
	if !inPlaceUpdateEnabled {
		return
	}

	exePath, err := os.Executable()
	if err != nil {
		return
	}
	exePath, err = filepath.EvalSymlinks(exePath)
	if err != nil {
		return
	}

	oldPath := exePath + ".old"
	if _, err := os.Stat(oldPath); err != nil {
		return // nothing to clean up
	}

	if err := os.Remove(oldPath); err != nil {
		log.Debug().Err(err).Str("path", oldPath).Msg("in-place swap: could not remove old binary")
	} else {
		log.Debug().Str("path", oldPath).Msg("in-place swap: cleaned up old binary")
	}
}

// copyFile copies src to dst, creating dst with the same permissions as src.
func copyFile(src, dst string) (err error) {
	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}

	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, srcInfo.Mode())
	if err != nil {
		return err
	}
	defer func() {
		if cerr := out.Close(); cerr != nil && err == nil {
			err = cerr
		}
	}()

	_, err = io.Copy(out, in)
	return err
}
