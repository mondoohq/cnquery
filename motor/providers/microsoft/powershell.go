package microsoft

import (
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"runtime"

	"go.mondoo.com/cnquery/motor/providers/microsoft/ms365/ms365report"
	os_provider "go.mondoo.com/cnquery/motor/providers/os"
	"go.mondoo.com/cnquery/motor/providers/os/cmd"
)

//go:embed mondoo-data-report.ps1
var report []byte

// NOTE: this is a temporary solution and will be replaced with logic that calls powershell directly and
// hopefully provides more flexibility in the future
func (p *Provider) GetMs365DataReport() (*ms365report.Microsoft365Report, error) {
	if p.assetType != ms365 {
		return nil, errors.New("ms365 data report not supported on this transport")
	}

	// if the report is cached, return that directly
	if p.ms365PowershellReport != nil {
		return p.ms365PowershellReport, nil
	}

	res, err := p.RunCommand(string(report))
	if err != nil {
		return nil, err
	}
	var data []byte
	if res.ExitStatus == 0 {
		data, err = io.ReadAll(res.Stdout)
	} else {
		data, err = io.ReadAll(res.Stderr)
		if err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("failed to generate ms365 report: %s", string(data))
	}

	p.ms365PowershellReport = &ms365report.Microsoft365Report{}
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(data, p.ms365PowershellReport)
	if err != nil {
		return nil, err
	}
	return p.ms365PowershellReport, nil
}

// todo: shall we inject that via an os provider?
func (p *Provider) RunCommand(command string) (*os_provider.Command, error) {
	var shell []string
	if runtime.GOOS == "windows" {
		// It does not make any sense to use cmd as default shell
		// shell = []string{"cmd", "/C"}
		shell = []string{"powershell", "-c", "Invoke-Expression", "-Command"}
	} else {
		shell = []string{"pwsh", "-c", "Invoke-Expression", "-Command"}
	}
	c := &cmd.CommandRunner{Shell: shell}
	args := []string{}
	res, err := c.Exec(command, args)
	return res, err
}
