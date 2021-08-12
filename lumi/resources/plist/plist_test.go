package plist

import (
	"bytes"
	"encoding/base64"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBase64DecodePlist(t *testing.T) {
	data := "YnBsaXN0MDDaAQIDBAUGBwgJCgsMDQ4XGBgaGxhcR3Vlc3RFbmFibGVkXxAZT3B0aW1pemVyTGFzdFJ1bkZvclN5c3RlbVxsYXN0VXNlck5hbWVbQWNjb3VudEluZm9fEBhPcHRpbWl6ZXJMYXN0UnVuRm9yQnVpbGRfEBpVc2VWb2ljZU92ZXJMZWdhY3lNaWdyYXRlZF8QE0Rpc2FibGVGREVBdXRvTG9naW5YbGFzdFVzZXJfEA9Mb2dpbndpbmRvd1RleHRcU0hPV0ZVTExOQU1FCBILBQEAXWFkbWluaXN0cmF0b3LTDxAREhMVXE1heGltdW1Vc2Vyc1lPbkNvbnNvbGVbRmlyc3RMb2dpbnMQAdEUEl1hZG1pbmlzdHJhdG9y0RYSXWFkbWluaXN0cmF0b3ISAoYKAAkJWGxvZ2dlZEluWmxvZ2luIHRleHQJAAgAHQAqAEYAUwBfAHoAlwCtALYAyADVANYA2wDpAPAA/QEHARMBFQEYASYBKQE3ATwBPQE+AUcBUgAAAAAAAAIBAAAAAAAAAB0AAAAAAAAAAAAAAAAAAAFT"

	decodedData, err := base64.StdEncoding.DecodeString(data)
	require.NoError(t, err)

	plistData, err := Decode(bytes.NewReader(decodedData))
	require.NoError(t, err)
	assert.NotNil(t, plistData)
	assert.Equal(t, "login text", plistData["LoginwindowText"])

	// convert binary to xml and parse again
	plistXmlData, err := ToXml(bytes.NewReader(decodedData))
	require.NoError(t, err)
	assert.NotNil(t, plistXmlData)

	plistData, err = Decode(bytes.NewReader(decodedData))
	require.NoError(t, err)
	assert.NotNil(t, plistData)
	assert.Equal(t, "login text", plistData["LoginwindowText"])
}
