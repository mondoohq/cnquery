package powershell

import (
	"encoding/base64"
	"fmt"
)

// base64 encoding for long powershell script
func Encode(cmd string) string {

	// powershall uses two bytes chars :-(
	withSpaceCmd := ""
	for _, b := range []byte(cmd) {
		withSpaceCmd += string(b) + "\x00"
	}

	// encode the command as base64
	input := []uint8(withSpaceCmd)
	return fmt.Sprintf("powershell.exe -EncodedCommand %s", base64.StdEncoding.EncodeToString(input))
}

func Wrap(cmd string) string {
	return fmt.Sprintf("powershell -c \"%s\"", cmd)
}
