// Copyright Mondoo, Inc. 2026
// SPDX-License-Identifier: BUSL-1.1

package providers

import (
	"fmt"
	"io"
	"os"
	"sync/atomic"
	"time"

	"github.com/rs/zerolog/log"
)

const (
	// EnvDownloadTimeout configures the idle timeout for provider and binary
	// downloads. The timeout resets every time data is received, so slow but
	// active transfers are not interrupted. Format: Go duration string
	// (e.g. "5m", "120s"). Default: 2 minutes.
	EnvDownloadTimeout     = "MONDOO_DOWNLOAD_TIMEOUT"
	defaultDownloadTimeout = 2 * time.Minute
)

// downloadTimeout returns the configured idle timeout for downloads.
// It reads MONDOO_DOWNLOAD_TIMEOUT and falls back to defaultDownloadTimeout.
func downloadTimeout() time.Duration {
	v := os.Getenv(EnvDownloadTimeout)
	if v == "" {
		return defaultDownloadTimeout
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		log.Warn().Str("value", v).Msg("invalid " + EnvDownloadTimeout + " value, using default")
		return defaultDownloadTimeout
	}
	if d <= 0 {
		log.Warn().Str("value", v).Msg(EnvDownloadTimeout + " must be positive, using default")
		return defaultDownloadTimeout
	}
	return d
}

// idleTimeoutReader wraps an io.ReadCloser and enforces an idle timeout.
// If no data is received for the configured duration, the underlying reader
// is closed, causing any blocked Read to return an error.
type idleTimeoutReader struct {
	body     io.ReadCloser
	timeout  time.Duration
	timer    *time.Timer
	timedOut atomic.Bool
}

// newIdleTimeoutReader wraps body with an idle timeout. The timer starts
// immediately; each successful Read that returns data resets it.
func newIdleTimeoutReader(body io.ReadCloser, timeout time.Duration) *idleTimeoutReader {
	itr := &idleTimeoutReader{
		body:    body,
		timeout: timeout,
	}
	itr.timer = time.AfterFunc(timeout, func() {
		itr.timedOut.Store(true)
		body.Close()
	})
	return itr
}

func (itr *idleTimeoutReader) Read(p []byte) (int, error) {
	n, err := itr.body.Read(p)
	if n > 0 {
		itr.timer.Reset(itr.timeout)
	}
	if err != nil && itr.timedOut.Load() {
		return n, fmt.Errorf("download stalled: no data received for %s (configure with %s)", itr.timeout, EnvDownloadTimeout)
	}
	return n, err
}

func (itr *idleTimeoutReader) Close() error {
	itr.timer.Stop()
	if itr.timedOut.Load() {
		return nil // body already closed by timer callback
	}
	return itr.body.Close()
}
