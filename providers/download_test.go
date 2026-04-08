// Copyright Mondoo, Inc. 2026
// SPDX-License-Identifier: BUSL-1.1

package providers

import (
	"bytes"
	"io"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// blockingReadCloser blocks on Read until the done channel is closed, then
// returns the provided error.
type blockingReadCloser struct {
	done chan struct{}
	err  error
	once sync.Once
}

func (b *blockingReadCloser) Read(p []byte) (int, error) {
	<-b.done
	return 0, b.err
}

func (b *blockingReadCloser) Close() error {
	b.once.Do(func() { close(b.done) })
	return nil
}

// slowReadCloser returns data in small chunks with a delay between each Read.
type slowReadCloser struct {
	data  []byte
	pos   int
	chunk int
	delay time.Duration
}

func (s *slowReadCloser) Read(p []byte) (int, error) {
	if s.pos >= len(s.data) {
		return 0, io.EOF
	}
	time.Sleep(s.delay)
	end := s.pos + s.chunk
	if end > len(s.data) {
		end = len(s.data)
	}
	n := copy(p, s.data[s.pos:end])
	s.pos += n
	return n, nil
}

func (s *slowReadCloser) Close() error { return nil }

func TestIdleTimeoutReader_NormalRead(t *testing.T) {
	data := []byte("hello world provider data")
	body := io.NopCloser(bytes.NewReader(data))

	itr := newIdleTimeoutReader(body, 5*time.Second)
	defer itr.Close()

	got, err := io.ReadAll(itr)
	require.NoError(t, err)
	assert.Equal(t, data, got)
}

func TestIdleTimeoutReader_StalledReadTimesOut(t *testing.T) {
	blocker := &blockingReadCloser{
		done: make(chan struct{}),
		err:  io.EOF,
	}

	itr := newIdleTimeoutReader(blocker, 100*time.Millisecond)
	defer itr.Close()

	_, err := io.ReadAll(itr)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "download stalled")
	assert.Contains(t, err.Error(), "MONDOO_DOWNLOAD_TIMEOUT")
}

func TestIdleTimeoutReader_SlowButActiveSucceeds(t *testing.T) {
	data := bytes.Repeat([]byte("x"), 100)
	slow := &slowReadCloser{
		data:  data,
		chunk: 10,
		delay: 50 * time.Millisecond, // 50ms between chunks
	}

	// Idle timeout is 500ms — each chunk arrives every 50ms, well within limit
	itr := newIdleTimeoutReader(slow, 500*time.Millisecond)
	defer itr.Close()

	got, err := io.ReadAll(itr)
	require.NoError(t, err)
	assert.Equal(t, data, got)
}

func TestIdleTimeoutReader_CloseStopsTimer(t *testing.T) {
	blocker := &blockingReadCloser{
		done: make(chan struct{}),
		err:  io.EOF,
	}

	itr := newIdleTimeoutReader(blocker, 1*time.Hour)
	// Close should stop the timer without panic
	require.NoError(t, itr.Close())
	// The timer should not fire after close
	assert.False(t, itr.timedOut.Load())
}

func TestDownloadTimeout_Default(t *testing.T) {
	t.Setenv(EnvDownloadTimeout, "")
	assert.Equal(t, defaultDownloadTimeout, downloadTimeout())
}

func TestDownloadTimeout_CustomValue(t *testing.T) {
	t.Setenv(EnvDownloadTimeout, "5m")
	assert.Equal(t, 5*time.Minute, downloadTimeout())
}

func TestDownloadTimeout_InvalidFallsBackToDefault(t *testing.T) {
	t.Setenv(EnvDownloadTimeout, "not-a-duration")
	assert.Equal(t, defaultDownloadTimeout, downloadTimeout())
}

func TestDownloadTimeout_NegativeFallsBackToDefault(t *testing.T) {
	t.Setenv(EnvDownloadTimeout, "-5s")
	assert.Equal(t, defaultDownloadTimeout, downloadTimeout())
}

func TestIdleTimeoutReader_LargePayload(t *testing.T) {
	// 1MB of data to ensure the reader handles larger payloads
	data := []byte(strings.Repeat("abcdefghij", 100_000))
	body := io.NopCloser(bytes.NewReader(data))

	itr := newIdleTimeoutReader(body, 5*time.Second)
	defer itr.Close()

	got, err := io.ReadAll(itr)
	require.NoError(t, err)
	assert.Equal(t, len(data), len(got))
}
