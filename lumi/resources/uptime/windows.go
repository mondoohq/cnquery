package uptime

import (
	"encoding/json"
	"time"
)

type WindowsUptime struct {
	TotalMilliseconds float64 `json:"TotalMilliseconds"`
}

// ParseWindowsUptime parses the json output of gcim LastBootUpTime
// (Get-Date) - (gcim Win32_OperatingSystem).LastBootUpTime | ConvertTo-Json
func ParseWindowsUptime(uptime string) (time.Duration, error) {

	var winUptime WindowsUptime
	err := json.Unmarshal([]byte(uptime), &winUptime)
	if err != nil {
		return 0, err
	}

	milli := winUptime.TotalMilliseconds * float64(time.Millisecond)
	return time.Duration(int64(milli)), nil
}
