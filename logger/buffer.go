package logger

import (
	"bytes"
	"io"
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
}

func (bw *BufferedWriter) Pause() {
	bw.paused = true
}

func (bw *BufferedWriter) Resume() {
	if bw.paused == false {
		return
	}
	bw.paused = false
	bw.out.Write(bw.buf.Bytes())
	bw.buf = bytes.Buffer{}
}

func (bw *BufferedWriter) Write(p []byte) (n int, err error) {
	if bw.paused {
		return bw.buf.Write(p)
	}
	return bw.out.Write(p)
}
