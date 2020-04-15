package logger

import (
	"fmt"
	"os"

	"github.com/hokaccha/go-prettyjson"
)

// DebugJson prints a prettified JSON of the data to CLI
func DebugJson(obj interface{}) {
	s, _ := prettyjson.Marshal(obj)
	fmt.Fprintln(os.Stderr, string(s))
}
