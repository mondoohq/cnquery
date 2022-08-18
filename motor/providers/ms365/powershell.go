package ms365

import (
	"encoding/json"
	"errors"
	"os"

	ms356_resources "go.mondoo.io/mondoo/resources/packs/services/ms365"
)

// NOTE: this is a temporary solution and will be replaced with logic that calls powershell directly and
// hopefully provides more flexibility in the future
func (p *Provider) GetMs365DataReport() (*ms356_resources.Microsoft365Report, error) {
	p.ms365PowershellReportLoader.Lock()
	defer p.ms365PowershellReportLoader.Unlock()

	if p.ms365PowershellReport != nil {
		return p.ms365PowershellReport, nil
	}

	if _, err := os.Stat(p.powershellDataReportFile); os.IsNotExist(err) {
		return nil, errors.New("could not load powershell data report from: " + p.powershellDataReportFile)
	}

	// get path from transport option
	data, err := os.ReadFile(p.powershellDataReportFile)
	if err != nil {
		return nil, err
	}

	p.ms365PowershellReport = &ms356_resources.Microsoft365Report{}
	err = json.Unmarshal(data, p.ms365PowershellReport)
	if err != nil {
		return nil, err
	}
	return p.ms365PowershellReport, nil
}
