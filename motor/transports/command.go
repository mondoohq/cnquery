package transports

import (
	"io"
	"time"
)

type PerfStats struct {
	Start    time.Time     `json:"start"`
	Duration time.Duration `json:"duration"`
}

type Command struct {
	Command    string
	Stats      PerfStats
	Stdout     io.ReadWriter
	Stderr     io.ReadWriter
	ExitStatus int
}
