// +build windows

package platformid

import (
	"errors"
	"runtime"

	"github.com/StackExchange/wmi"

	"go.mondoo.io/mondoo/motor/transports"
	"go.mondoo.io/mondoo/motor/transports/local"
)

func windowsMachineId(t transports.Transport) (string, error) {
	// if we are running locally on windows, we want to avoid using powershell to be faster
	_, ok := t.(*local.LocalTransport)
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
	return PowershellWindowsMachineId(t)
}
