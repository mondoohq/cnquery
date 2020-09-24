// +build windows

// Package eventlog provides a io.Writer to send the logs
// to Windows event log.
package eventlog

import (
	"bytes"
	"encoding/json"
	"io"

	"github.com/rs/zerolog"
	cbor "github.com/toravir/csd/libs"

	"golang.org/x/sys/windows/svc/debug"
	"golang.org/x/sys/windows/svc/eventlog"
)

// NewEventlogWriter returns a zerolog log destination
// to be used as parameter to New() calls. Writing logs
// to this writer will send the log messages to windows
// event log running on this system.
func NewEventlogWriter(svcName string) (io.WriteCloser, error) {
	elog, err := eventlog.Open(svcName)
	if err != nil {
		return nil, err
	}

	return eventlogWriter{
		elog: elog,
	}, nil
}

type eventlogWriter struct {
	elog debug.Log
}

func levelToJEventLevel(zLevel string) int {
	lvl, _ := zerolog.ParseLevel(zLevel)

	switch lvl {
	case zerolog.TraceLevel:
		return eventlog.Info
	case zerolog.DebugLevel:
		return eventlog.Info
	case zerolog.InfoLevel:
		return eventlog.Info
	case zerolog.WarnLevel:
		return eventlog.Warning
	case zerolog.ErrorLevel:
		return eventlog.Error
	case zerolog.FatalLevel:
		return eventlog.Error
	case zerolog.PanicLevel:
		return eventlog.Error
	case zerolog.NoLevel:
		return eventlog.Info
	}
	return eventlog.Info
}

func (w eventlogWriter) Close() error {
	return w.elog.Close()
}

func (w eventlogWriter) Write(p []byte) (n int, err error) {
	var event map[string]interface{}
	p = cbor.DecodeIfBinaryToBytes(p)
	d := json.NewDecoder(bytes.NewReader(p))
	d.UseNumber()
	err = d.Decode(&event)
	jPrio := eventlog.Info
	if err != nil {
		return
	}
	if l, ok := event[zerolog.LevelFieldName].(string); ok {
		jPrio = levelToJEventLevel(l)
	}

	switch jPrio {
	case eventlog.Error:
		w.elog.Error(1, string(p))
	case eventlog.Warning:
		w.elog.Warning(1, string(p))
	default:
		w.elog.Info(1, string(p))
	}

	return
}
