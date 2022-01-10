package logger

import (
	"bytes"
	"io"
	"sync"
)

func NewBufferedWriter(out io.Writer) io.Writer {
	return &BufferedWriter{
		out: out,
	}
}

type BufferedWriter struct {
	out    io.Writer
	buf    bytes.Buffer
	paused bool
	lock   sync.RWMutex
}

func (bw *BufferedWriter) Pause() {
	bw.lock.Lock()
	defer bw.lock.Unlock()

	bw.paused = true
}

func (bw *BufferedWriter) Resume() {
	bw.lock.Lock()
	defer bw.lock.Unlock()

	if bw.paused == false {
		return
	}
	bw.paused = false
	bw.out.Write(bw.buf.Bytes())
	bw.buf = bytes.Buffer{}
}

func (bw *BufferedWriter) Write(p []byte) (n int, err error) {
	bw.lock.RLock()
	defer bw.lock.RUnlock()

	if bw.paused {
		return bw.buf.Write(p)
	}
	return bw.out.Write(p)
}
