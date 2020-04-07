package platform_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	mock "go.mondoo.io/mondoo/motor/motoros/mock/toml"
	"go.mondoo.io/mondoo/motor/motoros/platform"
	"go.mondoo.io/mondoo/motor/motoros/types"
)

func newDetector(path string) (*platform.Detector, error) {
	mock, err := mock.New(&types.Endpoint{Backend: "mock", Path: path})
	if err != nil {
		return nil, err
	}
	detector := &platform.Detector{Transport: mock}
	return detector, nil
}

func TestRhel6OSDetector(t *testing.T) {
	detector, err := newDetector("./testdata/detect-rhel6.toml")
	assert.Nil(t, err, "was able to create the transport")
	resolved, di := detector.Resolve()

	assert.Equal(t, true, resolved, "platform should be resolvable")
	assert.Equal(t, "redhat", di.Name, "os name should be identified")
	assert.Equal(t, "Red Hat Linux", di.Title, "os title should be identified")
	assert.Equal(t, "6.2", di.Release, "os version should be identified")
	assert.Equal(t, "x86_64", di.Arch, "os arch should be identified")
	assert.Equal(t, []string{"redhat", "linux", "unix", "os"}, di.Family)
}

func TestRhel7OSDetector(t *testing.T) {
	detector, err := newDetector("./testdata/detect-rhel7.toml")
	assert.Nil(t, err, "was able to create the transport")
	resolved, di := detector.Resolve()

	assert.Equal(t, true, resolved, "platform should be resolvable")
	assert.Equal(t, "redhat", di.Name, "os name should be identified")
	assert.Equal(t, "Red Hat Enterprise Linux Server", di.Title, "os title should be identified")
	assert.Equal(t, "7.2", di.Release, "os version should be identified")
	assert.Equal(t, "x86_64", di.Arch, "os arch should be identified")
	assert.Equal(t, []string{"redhat", "linux", "unix", "os"}, di.Family)
}

func TestRhel7SLESOSDetector(t *testing.T) {
	detector, err := newDetector("./testdata/detect-rhel7-sles.toml")
	assert.Nil(t, err, "was able to create the transport")
	resolved, di := detector.Resolve()

	assert.Equal(t, true, resolved, "platform should be resolvable")
	assert.Equal(t, "redhat", di.Name, "os name should be identified")
	assert.Equal(t, "Red Hat Enterprise Linux Server", di.Title, "os title should be identified")
	assert.Equal(t, "7.4", di.Release, "os version should be identified")
	assert.Equal(t, "x86_64", di.Arch, "os arch should be identified")
	assert.Equal(t, []string{"redhat", "linux", "unix", "os"}, di.Family)
}

func TestRhel8OSDetector(t *testing.T) {
	detector, err := newDetector("./testdata/detect-rhel8.toml")
	assert.Nil(t, err, "was able to create the transport")
	resolved, di := detector.Resolve()

	assert.Equal(t, true, resolved, "platform should be resolvable")
	assert.Equal(t, "redhat", di.Name, "os name should be identified")
	assert.Equal(t, "Red Hat Enterprise Linux", di.Title, "os title should be identified")
	assert.Equal(t, "8.0", di.Release, "os version should be identified")
	assert.Equal(t, "x86_64", di.Arch, "os arch should be identified")
	assert.Equal(t, []string{"redhat", "linux", "unix", "os"}, di.Family)
}

func TestFedora29OSDetector(t *testing.T) {
	detector, err := newDetector("./testdata/detect-fedora29.toml")
	assert.Nil(t, err, "was able to create the transport")
	resolved, di := detector.Resolve()

	assert.Equal(t, true, resolved, "platform should be resolvable")
	assert.Equal(t, "fedora", di.Name, "os name should be identified")
	assert.Equal(t, "Fedora", di.Title, "os title should be identified")
	assert.Equal(t, "29", di.Release, "os version should be identified")
	assert.Equal(t, "x86_64", di.Arch, "os arch should be identified")
	assert.Equal(t, []string{"redhat", "linux", "unix", "os"}, di.Family)
}

func TestFedoraCoreOSDetector(t *testing.T) {
	detector, err := newDetector("./testdata/detect-coreos-fedora.toml")
	assert.Nil(t, err, "was able to create the transport")
	resolved, di := detector.Resolve()

	assert.Equal(t, true, resolved, "platform should be resolvable")
	assert.Equal(t, "fedora", di.Name, "os name should be identified")
	assert.Equal(t, "Fedora", di.Title, "os title should be identified")
	assert.Equal(t, "31", di.Release, "os version should be identified")
	assert.Equal(t, "x86_64", di.Arch, "os arch should be identified")
	assert.Equal(t, []string{"redhat", "linux", "unix", "os"}, di.Family)
}

func TestCoreOSDetector(t *testing.T) {
	detector, err := newDetector("./testdata/detect-coreos.toml")
	assert.Nil(t, err, "was able to create the transport")
	resolved, di := detector.Resolve()

	assert.Equal(t, true, resolved, "platform should be resolvable")
	assert.Equal(t, "flatcar", di.Name, "os name should be identified")
	assert.Equal(t, "Flatcar Container Linux by Kinvolk", di.Title, "os title should be identified")
	assert.Equal(t, "2430.0.0", di.Release, "os version should be identified")
	assert.Equal(t, "x86_64", di.Arch, "os arch should be identified")
	assert.Equal(t, []string{"linux", "unix", "os"}, di.Family)
}

func TestCentos6Detector(t *testing.T) {
	detector, err := newDetector("./testdata/detect-centos6.toml")
	assert.Nil(t, err, "was able to create the transport")
	resolved, di := detector.Resolve()

	assert.Equal(t, true, resolved, "platform should be resolvable")
	assert.Equal(t, "centos", di.Name, "os name should be identified")
	assert.Equal(t, "CentOS", di.Title, "os title should be identified")
	assert.Equal(t, "6.9", di.Release, "os version should be identified")
	assert.Equal(t, "x86_64", di.Arch, "os arch should be identified")
	assert.Equal(t, []string{"redhat", "linux", "unix", "os"}, di.Family)
}

func TestCentos7OSDetector(t *testing.T) {
	detector, err := newDetector("./testdata/detect-centos7.toml")
	assert.Nil(t, err, "was able to create the transport")
	resolved, di := detector.Resolve()

	assert.Equal(t, true, resolved, "platform should be resolvable")
	assert.Equal(t, "centos", di.Name, "os name should be identified")
	assert.Equal(t, "CentOS Linux", di.Title, "os title should be identified")
	assert.Equal(t, "7.5.1804", di.Release, "os version should be identified")
	assert.Equal(t, "x86_64", di.Arch, "os arch should be identified")
	assert.Equal(t, []string{"redhat", "linux", "unix", "os"}, di.Family)
}

func TestCentos5Detector(t *testing.T) {
	detector, err := newDetector("./testdata/detect-centos5.toml")
	assert.Nil(t, err, "was able to create the transport")
	resolved, di := detector.Resolve()

	assert.Equal(t, true, resolved, "platform should be resolvable")
	assert.Equal(t, "centos", di.Name, "os name should be identified")
	assert.Equal(t, "CentOS", di.Title, "os title should be identified")
	assert.Equal(t, "5.11", di.Release, "os version should be identified")
	assert.Equal(t, "x86_64", di.Arch, "os arch should be identified")
	assert.Equal(t, []string{"redhat", "linux", "unix", "os"}, di.Family)
}

func TestUbuntu1204Detector(t *testing.T) {
	detector, err := newDetector("./testdata/detect-ubuntu1204.toml")
	assert.Nil(t, err, "was able to create the transport")
	resolved, di := detector.Resolve()

	assert.Equal(t, true, resolved, "platform should be resolvable")
	assert.Equal(t, "ubuntu", di.Name, "os name should be identified")
	assert.Equal(t, "Ubuntu", di.Title, "os title should be identified")
	assert.Equal(t, "12.04", di.Release, "os version should be identified")
	assert.Equal(t, "x86_64", di.Arch, "os arch should be identified")
	assert.Equal(t, []string{"debian", "linux", "unix", "os"}, di.Family)
}

func TestUbuntu1404Detector(t *testing.T) {
	detector, err := newDetector("./testdata/detect-ubuntu1404.toml")
	assert.Nil(t, err, "was able to create the transport")
	resolved, di := detector.Resolve()

	assert.Equal(t, true, resolved, "platform should be resolvable")
	assert.Equal(t, "ubuntu", di.Name, "os name should be identified")
	assert.Equal(t, "Ubuntu", di.Title, "os title should be identified")
	assert.Equal(t, "14.04", di.Release, "os version should be identified")
	assert.Equal(t, "x86_64", di.Arch, "os arch should be identified")
	assert.Equal(t, []string{"debian", "linux", "unix", "os"}, di.Family)
}

func TestUbuntu1604Detector(t *testing.T) {
	detector, err := newDetector("./testdata/detect-ubuntu1604.toml")
	assert.Nil(t, err, "was able to create the transport")
	resolved, di := detector.Resolve()

	assert.Equal(t, true, resolved, "platform should be resolvable")
	assert.Equal(t, "ubuntu", di.Name, "os name should be identified")
	assert.Equal(t, "Ubuntu", di.Title, "os title should be identified")
	assert.Equal(t, "16.04", di.Release, "os version should be identified")
	assert.Equal(t, "x86_64", di.Arch, "os arch should be identified")
	assert.Equal(t, []string{"debian", "linux", "unix", "os"}, di.Family)
}

func TestUbuntu1804Detector(t *testing.T) {
	detector, err := newDetector("./testdata/detect-ubuntu1804.toml")
	assert.Nil(t, err, "was able to create the transport")
	resolved, di := detector.Resolve()

	assert.Equal(t, true, resolved, "platform should be resolvable")
	assert.Equal(t, "ubuntu", di.Name, "os name should be identified")
	assert.Equal(t, "Ubuntu", di.Title, "os title should be identified")
	assert.Equal(t, "18.04", di.Release, "os version should be identified")
	assert.Equal(t, "x86_64", di.Arch, "os arch should be identified")
	assert.Equal(t, []string{"debian", "linux", "unix", "os"}, di.Family)
}

func TestUbuntu2004Detector(t *testing.T) {
	detector, err := newDetector("./testdata/detect-ubuntu2004.toml")
	assert.Nil(t, err, "was able to create the transport")
	resolved, di := detector.Resolve()

	assert.Equal(t, true, resolved, "platform should be resolvable")
	assert.Equal(t, "ubuntu", di.Name, "os name should be identified")
	assert.Equal(t, "Ubuntu", di.Title, "os title should be identified")
	assert.Equal(t, "20.04", di.Release, "os version should be identified")
	assert.Equal(t, "x86_64", di.Arch, "os arch should be identified")
	assert.Equal(t, []string{"debian", "linux", "unix", "os"}, di.Family)
}

func TestWindriver7Detector(t *testing.T) {
	detector, err := newDetector("./testdata/detect-windriver7.toml")
	assert.Nil(t, err, "was able to create the transport")
	resolved, di := detector.Resolve()

	assert.Equal(t, true, resolved, "platform should be resolvable")
	assert.Equal(t, "wrlinux", di.Name, "os name should be identified")
	assert.Equal(t, "Wind River Linux", di.Title, "os title should be identified")
	assert.Equal(t, "7.0.0.2", di.Release, "os version should be identified")
	assert.Equal(t, "x86_64", di.Arch, "os arch should be identified")
	assert.Equal(t, []string{"linux", "unix", "os"}, di.Family)
}

func TestOpenWrtDetector(t *testing.T) {
	detector, err := newDetector("./testdata/detect-openwrt.toml")
	assert.Nil(t, err, "was able to create the transport")
	resolved, di := detector.Resolve()

	assert.Equal(t, true, resolved, "platform should be resolvable")
	assert.Equal(t, "openwrt", di.Name, "os name should be identified")
	assert.Equal(t, "OpenWrt", di.Title, "os title should be identified")
	assert.Equal(t, "Bleeding Edge", di.Release, "os version should be identified")
	assert.Equal(t, "x86_64", di.Arch, "os arch should be identified")
	assert.Equal(t, []string{"linux", "unix", "os"}, di.Family)
}

func TestDebian7Detector(t *testing.T) {
	detector, err := newDetector("./testdata/detect-debian7.toml")
	assert.Nil(t, err, "was able to create the transport")
	resolved, di := detector.Resolve()

	assert.Equal(t, true, resolved, "platform should be resolvable")
	assert.Equal(t, "debian", di.Name, "os name should be identified")
	assert.Equal(t, "Debian GNU/Linux", di.Title, "os title should be identified")
	assert.Equal(t, "7.11", di.Release, "os version should be identified")
	assert.Equal(t, "x86_64", di.Arch, "os arch should be identified")
	assert.Equal(t, []string{"debian", "linux", "unix", "os"}, di.Family)
}

func TestDebian8Detector(t *testing.T) {
	detector, err := newDetector("./testdata/detect-debian8.toml")
	assert.Nil(t, err, "was able to create the transport")
	resolved, di := detector.Resolve()

	assert.Equal(t, true, resolved, "platform should be resolvable")
	assert.Equal(t, "debian", di.Name, "os name should be identified")
	assert.Equal(t, "Debian GNU/Linux", di.Title, "os title should be identified")
	assert.Equal(t, "8.11", di.Release, "os version should be identified")
	assert.Equal(t, "x86_64", di.Arch, "os arch should be identified")
	assert.Equal(t, []string{"debian", "linux", "unix", "os"}, di.Family)
}

func TestDebian9Detector(t *testing.T) {
	detector, err := newDetector("./testdata/detect-debian9.toml")
	assert.Nil(t, err, "was able to create the transport")
	resolved, di := detector.Resolve()

	assert.Equal(t, true, resolved, "platform should be resolvable")
	assert.Equal(t, "debian", di.Name, "os name should be identified")
	assert.Equal(t, "Debian GNU/Linux", di.Title, "os title should be identified")
	assert.Equal(t, "9.4", di.Release, "os version should be identified")
	assert.Equal(t, "x86_64", di.Arch, "os arch should be identified")
	assert.Equal(t, []string{"debian", "linux", "unix", "os"}, di.Family)
}

func TestDebian10Detector(t *testing.T) {
	detector, err := newDetector("./testdata/detect-debian10.toml")
	assert.Nil(t, err, "was able to create the transport")
	resolved, di := detector.Resolve()

	assert.Equal(t, true, resolved, "platform should be resolvable")
	assert.Equal(t, "debian", di.Name, "os name should be identified")
	assert.Equal(t, "Debian GNU/Linux", di.Title, "os title should be identified")
	assert.Equal(t, "10.0", di.Release, "os version should be identified")
	assert.Equal(t, "x86_64", di.Arch, "os arch should be identified")
	assert.Equal(t, []string{"debian", "linux", "unix", "os"}, di.Family)
}

func TestRaspian10Detector(t *testing.T) {
	detector, err := newDetector("./testdata/detect-raspbian.toml")
	assert.Nil(t, err, "was able to create the transport")
	resolved, di := detector.Resolve()

	assert.Equal(t, true, resolved, "platform should be resolvable")
	assert.Equal(t, "raspbian", di.Name, "os name should be identified")
	assert.Equal(t, "Raspbian GNU/Linux", di.Title, "os title should be identified")
	assert.Equal(t, "10", di.Release, "os version should be identified")
	assert.Equal(t, "armv7l", di.Arch, "os arch should be identified")
	assert.Equal(t, []string{"debian", "linux", "unix", "os"}, di.Family)
}

func TestKaliRollingDetector(t *testing.T) {
	detector, err := newDetector("./testdata/detect-kalirolling.toml")
	assert.Nil(t, err, "was able to create the transport")
	resolved, di := detector.Resolve()

	assert.Equal(t, true, resolved, "platform should be resolvable")
	assert.Equal(t, "kali", di.Name, "os name should be identified")
	assert.Equal(t, "Kali GNU/Linux", di.Title, "os title should be identified")
	assert.Equal(t, "2019.4", di.Release, "os version should be identified")
	assert.Equal(t, "x86_64", di.Arch, "os arch should be identified")
	assert.Equal(t, []string{"debian", "linux", "unix", "os"}, di.Family)
}

func TestOpenSuse13Detector(t *testing.T) {
	detector, err := newDetector("./testdata/detect-opensuse-13.toml")
	assert.Nil(t, err, "was able to create the transport")
	resolved, di := detector.Resolve()

	assert.Equal(t, true, resolved, "platform should be resolvable")
	assert.Equal(t, "opensuse", di.Name, "os name should be identified")
	assert.Equal(t, "openSUSE", di.Title, "os title should be identified")
	assert.Equal(t, "13.2", di.Release, "os version should be identified")
	assert.Equal(t, "x86_64", di.Arch, "os arch should be identified")
	assert.Equal(t, []string{"suse", "linux", "unix", "os"}, di.Family)
}

func TestOpenSuseLeap42Detector(t *testing.T) {
	detector, err := newDetector("./testdata/detect-opensuse-leap-42.3.toml")
	assert.Nil(t, err, "was able to create the transport")
	resolved, di := detector.Resolve()

	assert.Equal(t, true, resolved, "platform should be resolvable")
	assert.Equal(t, "opensuse", di.Name, "os name should be identified")
	assert.Equal(t, "openSUSE Leap", di.Title, "os title should be identified")
	assert.Equal(t, "42.3", di.Release, "os version should be identified")
	assert.Equal(t, "x86_64", di.Arch, "os arch should be identified")
	assert.Equal(t, []string{"suse", "linux", "unix", "os"}, di.Family)
}

func TestOpenSuseLeap15Detector(t *testing.T) {
	detector, err := newDetector("./testdata/detect-opensuse-leap-15.toml")
	assert.Nil(t, err, "was able to create the transport")
	resolved, di := detector.Resolve()

	assert.Equal(t, true, resolved, "platform should be resolvable")
	assert.Equal(t, "opensuse-leap", di.Name, "os name should be identified")
	assert.Equal(t, "openSUSE Leap", di.Title, "os title should be identified")
	assert.Equal(t, "15.0", di.Release, "os version should be identified")
	assert.Equal(t, "x86_64", di.Arch, "os arch should be identified")
	assert.Equal(t, []string{"suse", "linux", "unix", "os"}, di.Family)
}

func TestOpenSuseTumbleweedDetector(t *testing.T) {
	detector, err := newDetector("./testdata/detect-opensuse-tumbleweed.toml")
	assert.Nil(t, err, "was able to create the transport")
	resolved, di := detector.Resolve()

	assert.Equal(t, true, resolved, "platform should be resolvable")
	assert.Equal(t, "opensuse-tumbleweed", di.Name, "os name should be identified")
	assert.Equal(t, "openSUSE Tumbleweed", di.Title, "os title should be identified")
	assert.Equal(t, "20200305", di.Release, "os version should be identified")
	assert.Equal(t, "x86_64", di.Arch, "os arch should be identified")
	assert.Equal(t, []string{"suse", "linux", "unix", "os"}, di.Family)
}

func TestSuse12Detector(t *testing.T) {
	detector, err := newDetector("./testdata/detect-suse-sles-12.toml")
	assert.Nil(t, err, "was able to create the transport")
	resolved, di := detector.Resolve()

	assert.Equal(t, true, resolved, "platform should be resolvable")
	assert.Equal(t, "sles", di.Name, "os name should be identified")
	assert.Equal(t, "SLES", di.Title, "os title should be identified")
	assert.Equal(t, "12.3", di.Release, "os version should be identified")
	assert.Equal(t, "x86_64", di.Arch, "os arch should be identified")
	assert.Equal(t, []string{"suse", "linux", "unix", "os"}, di.Family)
}

func TestSuse125Detector(t *testing.T) {
	detector, err := newDetector("./testdata/detect-suse-sles-12.5.toml")
	assert.Nil(t, err, "was able to create the transport")
	resolved, di := detector.Resolve()

	assert.Equal(t, true, resolved, "platform should be resolvable")
	assert.Equal(t, "sles", di.Name, "os name should be identified")
	assert.Equal(t, "SLES", di.Title, "os title should be identified")
	assert.Equal(t, "12.5", di.Release, "os version should be identified")
	assert.Equal(t, "x86_64", di.Arch, "os arch should be identified")
	assert.Equal(t, []string{"suse", "linux", "unix", "os"}, di.Family)
}

func TestSuse15Detector(t *testing.T) {
	detector, err := newDetector("./testdata/detect-suse-sles-15.toml")
	assert.Nil(t, err, "was able to create the transport")
	resolved, di := detector.Resolve()

	assert.Equal(t, true, resolved, "platform should be resolvable")
	assert.Equal(t, "sles", di.Name, "os name should be identified")
	assert.Equal(t, "SLES", di.Title, "os title should be identified")
	assert.Equal(t, "15.1", di.Release, "os version should be identified")
	assert.Equal(t, "x86_64", di.Arch, "os arch should be identified")
	assert.Equal(t, []string{"suse", "linux", "unix", "os"}, di.Family)
}

func TestAmazonLinuxDetector(t *testing.T) {
	detector, err := newDetector("./testdata/detect-amazonlinux-2017.09.toml")
	assert.Nil(t, err, "was able to create the transport")
	resolved, di := detector.Resolve()

	assert.Equal(t, true, resolved, "platform should be resolvable")
	assert.Equal(t, "amzn", di.Name, "os name should be identified")
	assert.Equal(t, "Amazon Linux AMI", di.Title, "os title should be identified")
	assert.Equal(t, "2017.09", di.Release, "os version should be identified")
	assert.Equal(t, "x86_64", di.Arch, "os arch should be identified")
	assert.Equal(t, []string{"linux", "unix", "os"}, di.Family)
}

func TestScientificLinuxDetector(t *testing.T) {
	detector, err := newDetector("./testdata/detect-scientific.toml")
	assert.Nil(t, err, "was able to create the transport")
	resolved, di := detector.Resolve()

	assert.Equal(t, true, resolved, "platform should be resolvable")
	assert.Equal(t, "scientific", di.Name, "os name should be identified")
	assert.Equal(t, "Scientific Linux CERN SLC", di.Title, "os title should be identified")
	assert.Equal(t, "6.9", di.Release, "os version should be identified")
	assert.Equal(t, "x86_64", di.Arch, "os arch should be identified")
	assert.Equal(t, []string{"redhat", "linux", "unix", "os"}, di.Family)
}

func TestArchLinuxVmDetector(t *testing.T) {
	detector, err := newDetector("./testdata/detect-arch-vm.toml")
	assert.Nil(t, err, "was able to create the transport")
	resolved, di := detector.Resolve()

	assert.Equal(t, true, resolved, "platform should be resolvable")
	assert.Equal(t, "arch", di.Name, "os name should be identified")
	assert.Equal(t, "Arch Linux", di.Title, "os title should be identified")
	assert.Equal(t, "x86_64", di.Arch, "os arch should be identified")
	assert.Equal(t, []string{"arch", "linux", "unix", "os"}, di.Family)
}

func TestArchLinuxContainerDetector(t *testing.T) {
	detector, err := newDetector("./testdata/detect-arch-container.toml")
	assert.Nil(t, err, "was able to create the transport")
	resolved, di := detector.Resolve()

	assert.Equal(t, true, resolved, "platform should be resolvable")
	assert.Equal(t, "arch", di.Name, "os name should be identified")
	assert.Equal(t, "Arch Linux", di.Title, "os title should be identified")
	assert.Equal(t, "x86_64", di.Arch, "os arch should be identified")
	assert.Equal(t, []string{"arch", "linux", "unix", "os"}, di.Family)
}

func TestManjaroLinuxContainerDetector(t *testing.T) {
	detector, err := newDetector("./testdata/detect-manjaro.toml")
	assert.Nil(t, err, "was able to create the transport")
	resolved, di := detector.Resolve()

	assert.Equal(t, true, resolved, "platform should be resolvable")
	assert.Equal(t, "manjaro", di.Name, "os name should be identified")
	assert.Equal(t, "Manjaro Linux", di.Title, "os title should be identified")
	assert.Equal(t, "x86_64", di.Arch, "os arch should be identified")
	assert.Equal(t, []string{"arch", "linux", "unix", "os"}, di.Family)
}

func TestOracleLinux6Detector(t *testing.T) {
	detector, err := newDetector("./testdata/detect-oracle6.toml")
	assert.Nil(t, err, "was able to create the transport")
	resolved, di := detector.Resolve()

	assert.Equal(t, true, resolved, "platform should be resolvable")
	assert.Equal(t, "ol", di.Name, "os name should be identified")
	assert.Equal(t, "Oracle Linux Server", di.Title, "os title should be identified")
	assert.Equal(t, "6.9", di.Release, "os version should be identified")
	assert.Equal(t, "x86_64", di.Arch, "os arch should be identified")
	assert.Equal(t, []string{"redhat", "linux", "unix", "os"}, di.Family)
}

func TestOracleLinux7Detector(t *testing.T) {
	detector, err := newDetector("./testdata/detect-oracle7.toml")
	assert.Nil(t, err, "was able to create the transport")
	resolved, di := detector.Resolve()

	assert.Equal(t, true, resolved, "platform should be resolvable")
	assert.Equal(t, "ol", di.Name, "os name should be identified")
	assert.Equal(t, "Oracle Linux Server", di.Title, "os title should be identified")
	assert.Equal(t, "7.5", di.Release, "os version should be identified")
	assert.Equal(t, "x86_64", di.Arch, "os arch should be identified")
	assert.Equal(t, []string{"redhat", "linux", "unix", "os"}, di.Family)
}

func TestOracleLinux8Detector(t *testing.T) {
	detector, err := newDetector("./testdata/detect-oracle8.toml")
	assert.Nil(t, err, "was able to create the transport")
	resolved, di := detector.Resolve()

	assert.Equal(t, true, resolved, "platform should be resolvable")
	assert.Equal(t, "ol", di.Name, "os name should be identified")
	assert.Equal(t, "Oracle Linux Server", di.Title, "os title should be identified")
	assert.Equal(t, "8.0", di.Release, "os version should be identified")
	assert.Equal(t, "x86_64", di.Arch, "os arch should be identified")
	assert.Equal(t, []string{"redhat", "linux", "unix", "os"}, di.Family)
}

func TestGentooLinuxDetector(t *testing.T) {
	detector, err := newDetector("./testdata/detect-gentoo.toml")
	assert.Nil(t, err, "was able to create the transport")
	resolved, di := detector.Resolve()

	assert.Equal(t, true, resolved, "platform should be resolvable")
	assert.Equal(t, "gentoo", di.Name, "os name should be identified")
	assert.Equal(t, "Gentoo", di.Title, "os title should be identified")
	assert.Equal(t, "2.4.1", di.Release, "os version should be identified")
	assert.Equal(t, "x86_64", di.Arch, "os arch should be identified")
	assert.Equal(t, []string{"linux", "unix", "os"}, di.Family)
}

func TestAlpineLinuxDetector(t *testing.T) {
	detector, err := newDetector("./testdata/detect-alpine.toml")
	assert.Nil(t, err, "was able to create the transport")
	resolved, di := detector.Resolve()

	assert.Equal(t, true, resolved, "platform should be resolvable")
	assert.Equal(t, "alpine", di.Name, "os name should be identified")
	assert.Equal(t, "Alpine Linux", di.Title, "os title should be identified")
	assert.Equal(t, "3.7.0", di.Release, "os version should be identified")
	assert.Equal(t, "x86_64", di.Arch, "os arch should be identified")
	assert.Equal(t, []string{"linux", "unix", "os"}, di.Family)
}

func TestBusyboxLinuxDetector(t *testing.T) {
	detector, err := newDetector("./testdata/detect-busybox.toml")
	assert.Nil(t, err, "was able to create the transport")
	resolved, di := detector.Resolve()

	assert.Equal(t, true, resolved, "platform should be resolvable")
	assert.Equal(t, "busybox", di.Name, "os name should be identified")
	assert.Equal(t, "BusyBox", di.Title, "os title should be identified")
	assert.Equal(t, "v1.28.4", di.Release, "os version should be identified")
	assert.Equal(t, "x86_64", di.Arch, "os arch should be identified")
	assert.Equal(t, []string{"linux", "unix", "os"}, di.Family)
}

func TestWindows2016Detector(t *testing.T) {
	detector, err := newDetector("./testdata/detect-windows2016.toml")
	assert.Nil(t, err, "was able to create the transport")
	resolved, di := detector.Resolve()

	assert.Equal(t, true, resolved, "platform should be resolvable")
	assert.Equal(t, "windows", di.Name, "os name should be identified")
	assert.Equal(t, "Microsoft Windows Server 2016 Standard Evaluation", di.Title, "os title should be identified")
	assert.Equal(t, "14393", di.Release, "os version should be identified")
	assert.Equal(t, "64-bit", di.Arch, "os arch should be identified")
	assert.Equal(t, []string{"windows", "os"}, di.Family)
}

func TestWindows2019Detector(t *testing.T) {
	detector, err := newDetector("./testdata/detect-windows2019.toml")
	assert.Nil(t, err, "was able to create the transport")
	resolved, di := detector.Resolve()

	assert.Equal(t, true, resolved, "platform should be resolvable")
	assert.Equal(t, "windows", di.Name, "os name should be identified")
	assert.Equal(t, "Microsoft Windows Server 2019 Datacenter Evaluation", di.Title, "os title should be identified")
	assert.Equal(t, "17763.720", di.Release, "os version should be identified")
	assert.Equal(t, "64-bit", di.Arch, "os arch should be identified")
	assert.Equal(t, []string{"windows", "os"}, di.Family)
}

func TestPhoton1Detector(t *testing.T) {
	detector, err := newDetector("./testdata/detect-photon1.toml")
	assert.Nil(t, err, "was able to create the transport")
	resolved, di := detector.Resolve()

	assert.Equal(t, true, resolved, "platform should be resolvable")
	assert.Equal(t, "photon", di.Name, "os name should be identified")
	assert.Equal(t, "VMware Photon", di.Title, "os title should be identified")
	assert.Equal(t, "1.0", di.Release, "os version should be identified")
	assert.Equal(t, "x86_64", di.Arch, "os arch should be identified")
	assert.Equal(t, []string{"linux", "unix", "os"}, di.Family)
}

func TestPhoton2Detector(t *testing.T) {
	detector, err := newDetector("./testdata/detect-photon2.toml")
	assert.Nil(t, err, "was able to create the transport")
	resolved, di := detector.Resolve()

	assert.Equal(t, true, resolved, "platform should be resolvable")
	assert.Equal(t, "photon", di.Name, "os name should be identified")
	assert.Equal(t, "VMware Photon OS", di.Title, "os title should be identified")
	assert.Equal(t, "2.0", di.Release, "os version should be identified")
	assert.Equal(t, "x86_64", di.Arch, "os arch should be identified")
	assert.Equal(t, []string{"linux", "unix", "os"}, di.Family)
}

func TestPhoton3Detector(t *testing.T) {
	detector, err := newDetector("./testdata/detect-photon3.toml")
	assert.Nil(t, err, "was able to create the transport")
	resolved, di := detector.Resolve()

	assert.Equal(t, true, resolved, "platform should be resolvable")
	assert.Equal(t, "photon", di.Name, "os name should be identified")
	assert.Equal(t, "VMware Photon OS", di.Title, "os title should be identified")
	assert.Equal(t, "3.0", di.Release, "os version should be identified")
	assert.Equal(t, "x86_64", di.Arch, "os arch should be identified")
	assert.Equal(t, []string{"linux", "unix", "os"}, di.Family)
}

func TestMacOSsDetector(t *testing.T) {
	detector, err := newDetector("./testdata/detect-macos.toml")
	assert.Nil(t, err, "was able to create the transport")
	resolved, di := detector.Resolve()

	assert.Equal(t, true, resolved, "platform should be resolvable")
	assert.Equal(t, "mac_os_x", di.Name, "os name should be identified")
	assert.Equal(t, "Mac OS X", di.Title, "os title should be identified")
	assert.Equal(t, "10.14.5", di.Release, "os version should be identified")
	assert.Equal(t, "x86_64", di.Arch, "os arch should be identified")
	assert.Equal(t, []string{"darwin", "bsd", "unix", "os"}, di.Family)
}

func TestBuildrootDetector(t *testing.T) {
	detector, err := newDetector("./testdata/detect-buildroot.toml")
	assert.Nil(t, err, "was able to create the transport")
	resolved, di := detector.Resolve()

	assert.Equal(t, true, resolved, "platform should be resolvable")
	assert.Equal(t, "buildroot", di.Name, "os name should be identified")
	assert.Equal(t, "Buildroot", di.Title, "os title should be identified")
	assert.Equal(t, "2019.02.9", di.Release, "os version should be identified")
	assert.Equal(t, "x86_64", di.Arch, "os arch should be identified")
	assert.Equal(t, []string{"linux", "unix", "os"}, di.Family)
}

func TestSolaris11Detector(t *testing.T) {
	detector, err := newDetector("./testdata/detect-solaris11.toml")
	assert.Nil(t, err, "was able to create the transport")
	resolved, di := detector.Resolve()

	assert.Equal(t, true, resolved, "platform should be resolvable")
	assert.Equal(t, "solaris", di.Name, "os name should be identified")
	assert.Equal(t, "Oracle Solaris", di.Title, "os title should be identified")
	assert.Equal(t, "11.1", di.Release, "os version should be identified")
	assert.Equal(t, "i86pc", di.Arch, "os arch should be identified")
	assert.Equal(t, []string{"unix", "os"}, di.Family)
}

func TestNetbsd8Detector(t *testing.T) {
	detector, err := newDetector("./testdata/detect-netbsd8.toml")
	assert.Nil(t, err, "was able to create the transport")
	resolved, di := detector.Resolve()

	assert.Equal(t, true, resolved, "platform should be resolvable")
	assert.Equal(t, "netbsd", di.Name, "os name should be identified")
	assert.Equal(t, "NetBSD", di.Title, "os title should be identified")
	assert.Equal(t, "8.0", di.Release, "os version should be identified")
	assert.Equal(t, "amd64", di.Arch, "os arch should be identified")
	assert.Equal(t, []string{"bsd", "unix", "os"}, di.Family)
}

func TestFreebsd12Detector(t *testing.T) {
	detector, err := newDetector("./testdata/detect-freebsd12.toml")
	assert.Nil(t, err, "was able to create the transport")
	resolved, di := detector.Resolve()

	assert.Equal(t, true, resolved, "platform should be resolvable")
	assert.Equal(t, "freebsd", di.Name, "os name should be identified")
	assert.Equal(t, "FreeBSD", di.Title, "os title should be identified")
	assert.Equal(t, "12.0-CURRENT", di.Release, "os version should be identified")
	assert.Equal(t, "amd64", di.Arch, "os arch should be identified")
	assert.Equal(t, []string{"bsd", "unix", "os"}, di.Family)
}

func TestFamilies(t *testing.T) {
	di := &platform.PlatformInfo{}
	di.Family = []string{"unix", "bsd", "darwin"}

	assert.Equal(t, true, di.IsFamily("unix"), "unix should be a family")
	assert.Equal(t, true, di.IsFamily("bsd"), "bsd should be a family")
	assert.Equal(t, true, di.IsFamily("darwin"), "darwin should be a family")
}

type TestContainer struct {
	Name  string
	Image string
}
