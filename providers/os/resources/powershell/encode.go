package powershell

import (
	"encoding/base64"
	"fmt"

	"github.com/rs/zerolog/log"
	"golang.org/x/text/encoding/unicode"
)

// Encode encodes a long powershell script as base64 and returns the wrapped command
//
// wraps a script to deactivate progress listener
// https://docs.microsoft.com/en-us/powershell/module/microsoft.powershell.core/about/about_preference_variables?view=powershell-7
//
// deactivates loading powershell profile
// https://docs.microsoft.com/en-us/windows-server/administration/windows-commands/powershell
func Encode(cmd string) string {
	// avoid messages to stderr that are not required in our execution
	script := "$ProgressPreference='SilentlyContinue';" + cmd

	encodedScript, err := ToBase64String(script)
	if err != nil {
		// Ignore this for now to keep the method interface identical
		// lets see if this becomes an issue
		log.Error().Err(err).Msg("could not encode powershell command")
	}

	return fmt.Sprintf("powershell.exe -NoProfile -EncodedCommand %s", encodedScript)
}

// ToBase64String encodes a powershell script to a UTF16-LE, base64 encoded string
// The encoded command can be used with powershell.exe -EncodedCommand
//
// $text = Get-Content .\script.ps1 -Raw;
// $encodedScript = [Convert]::ToBase64String([Text.Encoding]::Unicode.GetBytes($text));
// $encodedScript;
func ToBase64String(script string) (string, error) {
	uni := unicode.UTF16(unicode.LittleEndian, unicode.IgnoreBOM)
	encoded, err := uni.NewEncoder().String(script)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString([]byte(encoded)), nil
}

func Wrap(cmd string) string {
	return fmt.Sprintf("powershell -c \"%s\"", cmd)
}
