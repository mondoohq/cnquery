package powershell

import (
	"encoding/base64"
	"fmt"
)

// Encode encodes a long powershell script as base64 and returns the wrapped command
//
// wraps a script to deactivate progress listener
// https://docs.microsoft.com/en-us/powershell/module/microsoft.powershell.core/about/about_preference_variables?view=powershell-7
//
// deactivates loading powershell profile
// https://docs.microsoft.com/en-us/windows-server/administration/windows-commands/powershell
func Encode(cmd string) string {
	// avoids messages to stderr that are not required in our execution
	script := "$ProgressPreference='SilentlyContinue';" + cmd

	// powershall uses two bytes chars :-(
	withSpaceScript := ""
	for _, b := range []byte(script) {
		withSpaceScript += string(b) + "\x00"
	}

	// encode the command as base64 and wrap it in a powershell command
	input := []uint8(withSpaceScript)
	return fmt.Sprintf("powershell.exe -NoProfile -EncodedCommand %s", base64.StdEncoding.EncodeToString(input))
}

func Wrap(cmd string) string {
	return fmt.Sprintf("powershell -c \"%s\"", cmd)
}
