package logger

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/hokaccha/go-prettyjson"
	"github.com/rs/zerolog/log"
	"go.mondoo.io/mondoo"
)

// DebugJSON prints a prettified JSON of the data to CLI on debug mode
func DebugJSON(obj interface{}) {
	if !log.Debug().Enabled() {
		return
	}

	fmt.Fprintln(os.Stderr, PrettyJSON(obj))
}

func TraceJSON(obj interface{}) {
	if !log.Trace().Enabled() {
		return
	}

	fmt.Fprintln(os.Stderr, PrettyJSON(obj))
}

// PrettyJSON turns any object into its prettified json representation
func PrettyJSON(obj interface{}) string {
	s, _ := prettyjson.Marshal(obj)
	return string(s)
}

// DebugDumpJSON will write a JSON dump if the Debug mode is active and
// the DumpLocal prefix is defined.
func DebugDumpJSON(name string, obj interface{}) {
	if !log.Debug().Enabled() || mondoo.DumpLocal == "" {
		return
	}

	raw, err := json.MarshalIndent(obj, "", "  ")
	if err != nil {
		log.Error().Err(err).Msg("failed to dump JSON for resolved policy")
	}

	err = ioutil.WriteFile(mondoo.DumpLocal+name+".json", []byte(raw), 0644)
	if err != nil {
		log.Error().Err(err).Msg("failed to dump JSON for resolved policy")
	}
}
