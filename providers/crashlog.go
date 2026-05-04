// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package providers

import (
	"bytes"
	"io"
	"strings"
	"sync"
)

// defaultCrashLogLines is the number of stderr lines we retain per provider.
// 200 is enough to cover a typical Go panic stack trace (a goroutine + a few
// runtime frames is usually <50 lines, and we want headroom for unrelated log
// chatter that happened to land in the buffer right before the crash).
const defaultCrashLogLines = 200

// crashLogBuffer is an io.Writer that tees writes to an underlying writer
// while keeping the most recent lines in a ring buffer. We use it to capture
// a provider subprocess's stderr so that, on crash, we can surface the most
// recent stderr (typically a Go runtime fatal or panic stack trace) in the
// error attached to Runtime.CriticalErrors.
//
// hashicorp/go-plugin already writes the plugin's raw stderr to the configured
// Stderr writer line-by-line before any hclog wrapping. We sit in front of
// that path so the captured bytes are exactly what the plugin printed.
type crashLogBuffer struct {
	out io.Writer

	mu      sync.Mutex
	pending bytes.Buffer
	ring    [][]byte
	head    int
	full    bool
}

func newCrashLogBuffer(out io.Writer, lines int) *crashLogBuffer {
	if lines <= 0 {
		lines = defaultCrashLogLines
	}
	return &crashLogBuffer{
		out:  out,
		ring: make([][]byte, lines),
	}
}

func (b *crashLogBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	b.absorbLocked(p)
	b.mu.Unlock()

	if b.out != nil {
		return b.out.Write(p)
	}
	return len(p), nil
}

// absorbLocked splits p on '\n' and appends each completed line to the ring.
// Any trailing bytes (no '\n' yet) are kept in pending and prepended to the
// next write.
func (b *crashLogBuffer) absorbLocked(p []byte) {
	for len(p) > 0 {
		i := bytes.IndexByte(p, '\n')
		if i < 0 {
			b.pending.Write(p)
			return
		}
		b.pending.Write(p[:i])
		// drop a trailing '\r' so Windows-style line endings don't leak in
		line := b.pending.Bytes()
		if n := len(line); n > 0 && line[n-1] == '\r' {
			line = line[:n-1]
		}
		b.appendLineLocked(line)
		b.pending.Reset()
		p = p[i+1:]
	}
}

func (b *crashLogBuffer) appendLineLocked(line []byte) {
	cp := make([]byte, len(line))
	copy(cp, line)
	b.ring[b.head] = cp
	b.head = (b.head + 1) % len(b.ring)
	if b.head == 0 {
		b.full = true
	}
}

// Snapshot returns a copy of the current buffer contents in chronological order.
// Safe to call concurrently with Write.
//
// Any unterminated bytes still sitting in pending (a partial line that hasn't
// seen a '\n' yet) are appended as a final line. This matters for crashes
// where the subprocess is killed mid-write — the panic line that didn't get
// to flush its newline is exactly what we want to surface.
func (b *crashLogBuffer) Snapshot() []string {
	b.mu.Lock()
	defer b.mu.Unlock()

	var lines [][]byte
	if b.full {
		lines = make([][]byte, 0, len(b.ring)+1)
		lines = append(lines, b.ring[b.head:]...)
		lines = append(lines, b.ring[:b.head]...)
	} else {
		lines = b.ring[:b.head]
	}

	out := make([]string, 0, len(lines)+1)
	for _, l := range lines {
		out = append(out, string(l))
	}
	if b.pending.Len() > 0 {
		out = append(out, b.pending.String())
	}
	return out
}

// CrashTail returns the suffix of recent stderr lines that look like a Go
// runtime fatal or panic stack trace. The detection mirrors hashicorp/go-plugin's
// own panic detection (matching the same prefixes it uses) plus a few common
// runtime markers. Returns nil if no panic-like marker was found.
func (b *crashLogBuffer) CrashTail() []string {
	lines := b.Snapshot()

	// Walk forward, remember the last index where a panic/fatal marker started.
	// If a provider has multiple recovered panics earlier that landed in stderr
	// (e.g. via debug.PrintStack from inside recoverPanic), we want the most
	// recent one — that's the unrecoverable fatal that actually killed the
	// process.
	start := -1
	for i, raw := range lines {
		l := strings.TrimSpace(raw)
		switch {
		case strings.HasPrefix(l, "panic:"),
			strings.HasPrefix(l, "fatal error:"),
			strings.HasPrefix(l, "runtime: "),
			strings.HasPrefix(l, "SIG"),
			strings.HasPrefix(l, "unexpected fault address"):
			start = i
		}
	}
	if start < 0 {
		return nil
	}
	return lines[start:]
}
