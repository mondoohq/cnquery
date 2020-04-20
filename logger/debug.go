package logger

import (
	"fmt"
	"os"

	"github.com/hokaccha/go-prettyjson"
	"github.com/rs/zerolog/log"
)

// DebugJSON prints a prettified JSON of the data to CLI
func DebugJSON(obj interface{}) {
	if !log.Debug().Enabled() {
		return
	}

	s, _ := prettyjson.Marshal(obj)
	fmt.Fprintln(os.Stderr, string(s))
}
