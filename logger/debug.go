package logger

import (
	"fmt"
	"github.com/hokaccha/go-prettyjson"
)

// DebugJson prints a prettified JSON of the data to CLI
func DebugJson(obj interface{}) {
	s, _ := prettyjson.Marshal(obj)
	fmt.Println(string(s))
}
