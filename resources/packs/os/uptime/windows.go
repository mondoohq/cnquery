package uptime

import (
	"encoding/json"
	"io"
	"io/ioutil"
	"time"

	"go.mondoo.com/cnquery/motor/providers/os"
	"go.mondoo.com/cnquery/motor/providers/os/powershell"
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

const WindowsUptimeCmd = "(Get-Date) - (gcim Win32_OperatingSystem).LastBootUpTime | ConvertTo-Json"

type Windows struct {
	provider os.OperatingSystemProvider
}

func (s *Windows) Name() string {
	return "Windows Uptime"
}

func (s *Windows) Duration() (time.Duration, error) {
	cmd, err := s.provider.RunCommand(powershell.Wrap(WindowsUptimeCmd))
	if err != nil {
		return 0, err
	}

	return s.parse(cmd.Stdout)
}

func (s *Windows) parse(r io.Reader) (time.Duration, error) {
	content, err := ioutil.ReadAll(r)
	if err != nil {
		return 0, err
	}
	return ParseWindowsUptime(string(content))
}
