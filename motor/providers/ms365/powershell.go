package ms365

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"os"

	ms356_resources "go.mondoo.io/mondoo/lumi/resources/ms365"
)

// NOTE: this is a temporary solution and will be replaced with logic that calls powershell directly and
// hopefully provides more flexibility in the future
func (t *Provider) GetMs365DataReport() (*ms356_resources.Microsoft365Report, error) {
	t.ms365PowershellReportLoader.Lock()
	defer t.ms365PowershellReportLoader.Unlock()

	if t.ms365PowershellReport != nil {
		return t.ms365PowershellReport, nil
	}

	if _, err := os.Stat(t.powershellDataReportFile); os.IsNotExist(err) {
		return nil, errors.New("could not load powershell data report from: " + t.powershellDataReportFile)
	}

	// get path from transport option
	data, err := ioutil.ReadFile(t.powershellDataReportFile)
	if err != nil {
		return nil, err
	}

	t.ms365PowershellReport = &ms356_resources.Microsoft365Report{}
	err = json.Unmarshal(data, t.ms365PowershellReport)
	if err != nil {
		return nil, err
	}
	return t.ms365PowershellReport, nil
}
