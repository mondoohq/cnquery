package logger

import (
	"fmt"
	"os"

	"github.com/hokaccha/go-prettyjson"
	"github.com/rs/zerolog/log"
)

// DebugJSON prints a prettified JSON of the data to CLI on debug mode
func DebugJSON(obj interface{}) {
	if !log.Debug().Enabled() {
		return
	}

	fmt.Fprintln(os.Stderr, PrettyJSON(obj))
}

// PrettyJSON turns any object into its prettified json representation
func PrettyJSON(obj interface{}) string {
	s, _ := prettyjson.Marshal(obj)
	return string(s)
}
