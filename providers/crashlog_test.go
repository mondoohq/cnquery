// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package providers

import (
	"bytes"
	"io"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCrashLogBuffer_TeesToUnderlying(t *testing.T) {
	var sink bytes.Buffer
	b := newCrashLogBuffer(&sink, 10)

	n, err := b.Write([]byte("hello\nworld\n"))
	assert.NoError(t, err)
	assert.Equal(t, 12, n)
	assert.Equal(t, "hello\nworld\n", sink.String())
}

func TestCrashLogBuffer_NilUnderlyingIsAllowed(t *testing.T) {
	b := newCrashLogBuffer(nil, 10)
	n, err := b.Write([]byte("hi\n"))
	assert.NoError(t, err)
	assert.Equal(t, 3, n)
	assert.Equal(t, []string{"hi"}, b.Snapshot())
}

func TestCrashLogBuffer_SplitsOnNewlines(t *testing.T) {
	b := newCrashLogBuffer(io.Discard, 10)
	_, _ = b.Write([]byte("one\ntwo\nthree\n"))
	assert.Equal(t, []string{"one", "two", "three"}, b.Snapshot())
}

func TestCrashLogBuffer_HandlesPartialLinesAcrossWrites(t *testing.T) {
	b := newCrashLogBuffer(io.Discard, 10)
	_, _ = b.Write([]byte("partial "))
	// not yet a complete line
	assert.Empty(t, b.Snapshot())
	_, _ = b.Write([]byte("line\nnext\n"))
	assert.Equal(t, []string{"partial line", "next"}, b.Snapshot())
}

func TestCrashLogBuffer_StripsCRLF(t *testing.T) {
	b := newCrashLogBuffer(io.Discard, 10)
	_, _ = b.Write([]byte("windows\r\nstyle\r\n"))
	assert.Equal(t, []string{"windows", "style"}, b.Snapshot())
}

func TestCrashLogBuffer_RingOverflowKeepsMostRecent(t *testing.T) {
	b := newCrashLogBuffer(io.Discard, 3)
	for _, l := range []string{"a", "b", "c", "d", "e"} {
		_, _ = b.Write([]byte(l + "\n"))
	}
	assert.Equal(t, []string{"c", "d", "e"}, b.Snapshot())
}

func TestCrashLogBuffer_CrashTailFindsPanic(t *testing.T) {
	b := newCrashLogBuffer(io.Discard, 50)
	stderr := strings.Join([]string{
		"2026-05-04T10:00:00Z INF starting up",
		"2026-05-04T10:00:01Z DBG querying foo",
		"panic: runtime error: invalid memory address or nil pointer dereference",
		"[signal SIGSEGV: segmentation violation code=0x1 addr=0x0]",
		"",
		"goroutine 1 [running]:",
		"main.main()",
		"\t/src/main.go:42 +0x10",
		"",
	}, "\n") + "\n"
	_, _ = b.Write([]byte(stderr))

	tail := b.CrashTail()
	assert.NotEmpty(t, tail)
	assert.Equal(t, "panic: runtime error: invalid memory address or nil pointer dereference", tail[0])
	// The DBG/INF lines from before the panic must not be included.
	for _, l := range tail {
		assert.NotContains(t, l, "starting up")
	}
}

func TestCrashLogBuffer_CrashTailFindsFatalError(t *testing.T) {
	b := newCrashLogBuffer(io.Discard, 50)
	_, _ = b.Write([]byte("regular log\nfatal error: concurrent map writes\n\ngoroutine 17:\n"))
	tail := b.CrashTail()
	assert.Equal(t, []string{
		"fatal error: concurrent map writes",
		"",
		"goroutine 17:",
	}, tail)
}

func TestCrashLogBuffer_CrashTailReturnsNilWhenNoMarker(t *testing.T) {
	b := newCrashLogBuffer(io.Discard, 10)
	_, _ = b.Write([]byte("just some logs\nand more logs\n"))
	assert.Nil(t, b.CrashTail())
}

func TestCrashLogBuffer_CrashTailPicksMostRecentPanic(t *testing.T) {
	// Two panic markers in the buffer: we want the second (most recent) one,
	// since that's the one that actually killed the process.
	b := newCrashLogBuffer(io.Discard, 50)
	_, _ = b.Write([]byte("panic: first\nrecovered\npanic: second fatal\ngoroutine 1:\n"))
	tail := b.CrashTail()
	assert.Equal(t, []string{"panic: second fatal", "goroutine 1:"}, tail)
}

func TestCrashLogBuffer_ConcurrentWrites(t *testing.T) {
	b := newCrashLogBuffer(io.Discard, 1000)
	var wg sync.WaitGroup
	for i := range 10 {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			for range 50 {
				_, _ = b.Write([]byte("line\n"))
			}
		}(i)
	}
	wg.Wait()
	// We don't care about ordering, just that it didn't race or panic.
	assert.Len(t, b.Snapshot(), 500)
}
