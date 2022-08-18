package reboot

import (
	"io/ioutil"
	"strings"

	"go.mondoo.io/mondoo/motor/providers"
	"go.mondoo.io/mondoo/motor/providers/os"
	"go.mondoo.io/mondoo/resources/packs/core/packages"
	"go.mondoo.io/mondoo/vadvisor/versions/rpm"
)

// RpmNewestKernel works on all machines running rpm
type RpmNewestKernel struct {
	provider os.OperatingSystemProvider
}

func (s *RpmNewestKernel) Name() string {
	return "RPM Latest Kernel"
}

func (s *RpmNewestKernel) RebootPending() (bool, error) {
	// if it is a static asset, no reboot is pending
	if !s.provider.Capabilities().HasCapability(providers.Capability_RunCommand) {
		return false, nil
	}

	// get installed kernel version
	installedKernelCmd, err := s.provider.RunCommand("rpm -q kernel --queryformat '%{NAME} %{EPOCHNUM}:%{VERSION}-%{RELEASE} %{ARCH} %{SUMMARY}\n'")
	if err != nil {
		return false, err
	}

	pkgs := packages.ParseRpmPackages(installedKernelCmd.Stdout)
	// this case is valid in container
	if len(pkgs) == 0 {
		return false, nil
	}

	// check running kernel version
	unamerCmd, err := s.provider.RunCommand("uname -r")
	if err != nil {
		return false, err
	}

	unameR, err := ioutil.ReadAll(unamerCmd.Stdout)
	if err != nil {
		return false, err
	}

	// check if any kernel is newer
	kernelVersion := strings.TrimSpace(string(unameR))

	var parser rpm.Parser

	for i := range pkgs {
		cmp, err := parser.Compare(pkgs[i].Version, kernelVersion)
		if err != nil {
			return false, err
		}
		if cmp >= 1 {
			return true, nil
		}
	}
	return false, nil
}
