//go:build windows
// +build windows

package platformid

import (
	"errors"
	"runtime"

	wmi "github.com/StackExchange/wmi"
	"go.mondoo.com/cnquery/motor/providers/local"
	"go.mondoo.com/cnquery/motor/providers/os"
)

func windowsMachineId(p os.OperatingSystemProvider) (string, error) {
	// if we are running locally on windows, we want to avoid using powershell to be faster
	_, ok := p.(*local.Provider)
	if ok && runtime.GOOS == "windows" {

		// we always get a list or entries
		type win32ComputerSystemProduct struct {
			UUID *string
		}

		// query wmi to retrieve information
		var entries []win32ComputerSystemProduct
		if err := wmi.Query(wmiMachineIDQuery, &entries); err != nil {
			return "", err
		}

		if len(entries) != 1 || entries[0].UUID == nil {
			return "", errors.New("could not query machine id on windows")
		}

		return *entries[0].UUID, nil
	}
	return PowershellWindowsMachineId(p)
}
