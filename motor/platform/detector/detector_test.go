package detector_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.io/mondoo/motor/platform"
	"go.mondoo.io/mondoo/motor/platform/detector"
	"go.mondoo.io/mondoo/motor/providers/mock"
)

func newDetector(filepath string) (*detector.Detector, error) {
	mock, err := mock.NewFromTomlFile(filepath)
	if err != nil {
		return nil, err
	}
	detector := detector.New(mock)
	return detector, nil
}

func TestRhel6OSDetector(t *testing.T) {
	detector, err := newDetector("./testdata/detect-rhel-6.toml")
	assert.Nil(t, err, "was able to create the transport")
	di, err := detector.Platform()
	require.NoError(t, err)

	assert.Equal(t, "redhat", di.Name, "os name should be identified")
	assert.Equal(t, "Red Hat Linux", di.Title, "os title should be identified")
	assert.Equal(t, "6.2", di.Version, "os version should be identified")
	assert.Equal(t, "x86_64", di.Arch, "os arch should be identified")
	assert.Equal(t, []string{"redhat", "linux", "unix", "os"}, di.Family)
}

func TestRhel7OSDetector(t *testing.T) {
	detector, err := newDetector("./testdata/detect-rhel-7.toml")
	assert.Nil(t, err, "was able to create the transport")
	di, err := detector.Platform()
	require.NoError(t, err)

	assert.Equal(t, "redhat", di.Name, "os name should be identified")
	assert.Equal(t, "Red Hat Enterprise Linux Server 7.2 (Maipo)", di.Title, "os title should be identified")
	assert.Equal(t, "7.2", di.Version, "os version should be identified")
	assert.Equal(t, "x86_64", di.Arch, "os arch should be identified")
	assert.Equal(t, []string{"redhat", "linux", "unix", "os"}, di.Family)
}

func TestRhel7SLESOSDetector(t *testing.T) {
	detector, err := newDetector("./testdata/detect-rhel-7-sles.toml")
	assert.Nil(t, err, "was able to create the transport")
	di, err := detector.Platform()
	require.NoError(t, err)

	assert.Equal(t, "redhat", di.Name, "os name should be identified")
	assert.Equal(t, "Red Hat Enterprise Linux Server 7.4", di.Title, "os title should be identified")
	assert.Equal(t, "7.4", di.Version, "os version should be identified")
	assert.Equal(t, "x86_64", di.Arch, "os arch should be identified")
	assert.Equal(t, []string{"redhat", "linux", "unix", "os"}, di.Family)
}

func TestRhel8OSDetector(t *testing.T) {
	detector, err := newDetector("./testdata/detect-rhel-8.toml")
	assert.Nil(t, err, "was able to create the transport")
	di, err := detector.Platform()
	require.NoError(t, err)

	assert.Equal(t, "redhat", di.Name, "os name should be identified")
	assert.Equal(t, "Red Hat Enterprise Linux 8.0 (Ootpa)", di.Title, "os title should be identified")
	assert.Equal(t, "8.0", di.Version, "os version should be identified")
	assert.Equal(t, "x86_64", di.Arch, "os arch should be identified")
	assert.Equal(t, []string{"redhat", "linux", "unix", "os"}, di.Family)
}

func TestRhel9OSDetector(t *testing.T) {
	detector, err := newDetector("./testdata/detect-rhel-9.toml")
	assert.Nil(t, err, "was able to create the transport")
	di, err := detector.Platform()
	require.NoError(t, err)

	assert.Equal(t, "redhat", di.Name, "os name should be identified")
	assert.Equal(t, "Red Hat Enterprise Linux 9.0 (Plow)", di.Title, "os title should be identified")
	assert.Equal(t, "9.0", di.Version, "os version should be identified")
	assert.Equal(t, "aarch64", di.Arch, "os arch should be identified")
	assert.Equal(t, []string{"redhat", "linux", "unix", "os"}, di.Family)
}

func TestFedora29OSDetector(t *testing.T) {
	detector, err := newDetector("./testdata/detect-fedora29.toml")
	assert.Nil(t, err, "was able to create the transport")
	di, err := detector.Platform()
	require.NoError(t, err)

	assert.Equal(t, "fedora", di.Name, "os name should be identified")
	assert.Equal(t, "Fedora 29 (Container Image)", di.Title, "os title should be identified")
	assert.Equal(t, "29", di.Version, "os version should be identified")
	assert.Equal(t, "x86_64", di.Arch, "os arch should be identified")
	assert.Equal(t, []string{"redhat", "linux", "unix", "os"}, di.Family)
}

func TestFedoraCoreOSDetector(t *testing.T) {
	detector, err := newDetector("./testdata/detect-coreos-fedora.toml")
	assert.Nil(t, err, "was able to create the transport")
	di, err := detector.Platform()
	require.NoError(t, err)

	assert.Equal(t, "fedora", di.Name, "os name should be identified")
	assert.Equal(t, "Fedora CoreOS 31.20200310.3.0", di.Title, "os title should be identified")
	assert.Equal(t, "31", di.Version, "os version should be identified")
	assert.Equal(t, "x86_64", di.Arch, "os arch should be identified")
	assert.Equal(t, []string{"redhat", "linux", "unix", "os"}, di.Family)
}

func TestCoreOSDetector(t *testing.T) {
	detector, err := newDetector("./testdata/detect-coreos.toml")
	assert.Nil(t, err, "was able to create the transport")
	di, err := detector.Platform()
	require.NoError(t, err)

	assert.Equal(t, "flatcar", di.Name, "os name should be identified")
	assert.Equal(t, "Flatcar Container Linux by Kinvolk 2430.0.0 (Rhyolite)", di.Title, "os title should be identified")
	assert.Equal(t, "2430.0.0", di.Version, "os version should be identified")
	assert.Equal(t, "x86_64", di.Arch, "os arch should be identified")
	assert.Equal(t, []string{"linux", "unix", "os"}, di.Family)
}

func TestCentos5Detector(t *testing.T) {
	detector, err := newDetector("./testdata/detect-centos-5.toml")
	assert.Nil(t, err, "was able to create the transport")
	di, err := detector.Platform()
	require.NoError(t, err)

	assert.Equal(t, "centos", di.Name, "os name should be identified")
	assert.Equal(t, "CentOS", di.Title, "os title should be identified")
	assert.Equal(t, "5.11", di.Version, "os version should be identified")
	assert.Equal(t, "x86_64", di.Arch, "os arch should be identified")
	assert.Equal(t, []string{"redhat", "linux", "unix", "os"}, di.Family)
}

func TestCentos6Detector(t *testing.T) {
	detector, err := newDetector("./testdata/detect-centos-6.toml")
	assert.Nil(t, err, "was able to create the transport")
	di, err := detector.Platform()
	require.NoError(t, err)

	assert.Equal(t, "centos", di.Name, "os name should be identified")
	assert.Equal(t, "CentOS", di.Title, "os title should be identified")
	assert.Equal(t, "6.9", di.Version, "os version should be identified")
	assert.Equal(t, "x86_64", di.Arch, "os arch should be identified")
	assert.Equal(t, []string{"redhat", "linux", "unix", "os"}, di.Family)
}

func TestCentos7OSDetector(t *testing.T) {
	detector, err := newDetector("./testdata/detect-centos-7.toml")
	assert.Nil(t, err, "was able to create the transport")
	di, err := detector.Platform()
	require.NoError(t, err)

	assert.Equal(t, "centos", di.Name, "os name should be identified")
	assert.Equal(t, "CentOS Linux 7 (Core)", di.Title, "os title should be identified")
	assert.Equal(t, "7.5.1804", di.Version, "os version should be identified")
	assert.Equal(t, "x86_64", di.Arch, "os arch should be identified")
	assert.Equal(t, []string{"redhat", "linux", "unix", "os"}, di.Family)
}

func TestCentos8OSDetector(t *testing.T) {
	detector, err := newDetector("./testdata/detect-centos-8.toml")
	assert.Nil(t, err, "was able to create the transport")
	di, err := detector.Platform()
	require.NoError(t, err)

	assert.Equal(t, "centos", di.Name, "os name should be identified")
	assert.Equal(t, "CentOS Linux 8 (Core)", di.Title, "os title should be identified")
	assert.Equal(t, "8.2.2004", di.Version, "os version should be identified")
	assert.Equal(t, "x86_64", di.Arch, "os arch should be identified")
	assert.Equal(t, []string{"redhat", "linux", "unix", "os"}, di.Family)
}

func TestCentos8StreamOSDetector(t *testing.T) {
	detector, err := newDetector("./testdata/detect-centos-8-stream.toml")
	assert.Nil(t, err, "was able to create the transport")
	di, err := detector.Platform()
	require.NoError(t, err)

	assert.Equal(t, "centos", di.Name, "os name should be identified")
	assert.Equal(t, "CentOS Stream 8", di.Title, "os title should be identified")
	assert.Equal(t, "8", di.Version, "os version should be identified")
	assert.Equal(t, "x86_64", di.Arch, "os arch should be identified")
	assert.Equal(t, []string{"redhat", "linux", "unix", "os"}, di.Family)
}

func TestCentos9StreamOSDetector(t *testing.T) {
	detector, err := newDetector("./testdata/detect-centos-9-stream.toml")
	assert.Nil(t, err, "was able to create the transport")
	di, err := detector.Platform()
	require.NoError(t, err)

	assert.Equal(t, "centos", di.Name, "os name should be identified")
	assert.Equal(t, "CentOS Stream 9", di.Title, "os title should be identified")
	assert.Equal(t, "9", di.Version, "os version should be identified")
	assert.Equal(t, "x86_64", di.Arch, "os arch should be identified")
	assert.Equal(t, []string{"redhat", "linux", "unix", "os"}, di.Family)
}

func TestAlmaLinux8OSDetector(t *testing.T) {
	detector, err := newDetector("./testdata/detect-almalinux-8.toml")
	assert.Nil(t, err, "was able to create the transport")
	di, err := detector.Platform()
	require.NoError(t, err)

	assert.Equal(t, "almalinux", di.Name, "os name should be identified")
	assert.Equal(t, "AlmaLinux 8.3 Beta (Purple Manul)", di.Title, "os title should be identified")
	assert.Equal(t, "8.3", di.Version, "os version should be identified")
	assert.Equal(t, "x86_64", di.Arch, "os arch should be identified")
	assert.Equal(t, []string{"redhat", "linux", "unix", "os"}, di.Family)
}

func TestAlmaLinux9OSDetector(t *testing.T) {
	detector, err := newDetector("./testdata/detect-almalinux-9.toml")
	assert.Nil(t, err, "was able to create the transport")
	di, err := detector.Platform()
	require.NoError(t, err)

	assert.Equal(t, "almalinux", di.Name, "os name should be identified")
	assert.Equal(t, "AlmaLinux 9.0 Beta (Emerald Puma)", di.Title, "os title should be identified")
	assert.Equal(t, "9.0", di.Version, "os version should be identified")
	assert.Equal(t, "aarch64", di.Arch, "os arch should be identified")
	assert.Equal(t, []string{"redhat", "linux", "unix", "os"}, di.Family)
}

func TestRocky8OSDetector(t *testing.T) {
	detector, err := newDetector("./testdata/detect-rocky-8.toml")
	assert.Nil(t, err, "was able to create the transport")
	di, err := detector.Platform()
	require.NoError(t, err)

	assert.Equal(t, "rockylinux", di.Name, "os name should be identified")
	assert.Equal(t, "Rocky Linux 8.5 (Green Obsidian)", di.Title, "os title should be identified")
	assert.Equal(t, "8.5", di.Version, "os version should be identified")
	assert.Equal(t, "aarch64", di.Arch, "os arch should be identified")
	assert.Equal(t, []string{"redhat", "linux", "unix", "os"}, di.Family)
}

func TestUbuntu1204Detector(t *testing.T) {
	detector, err := newDetector("./testdata/detect-ubuntu1204.toml")
	assert.Nil(t, err, "was able to create the transport")
	di, err := detector.Platform()
	require.NoError(t, err)

	assert.Equal(t, "ubuntu", di.Name, "os name should be identified")
	assert.Equal(t, "Ubuntu precise (12.04.5 LTS)", di.Title, "os title should be identified")
	assert.Equal(t, "12.04", di.Version, "os version should be identified")
	assert.Equal(t, "x86_64", di.Arch, "os arch should be identified")
	assert.Equal(t, []string{"debian", "linux", "unix", "os"}, di.Family)
}

func TestUbuntu1404Detector(t *testing.T) {
	detector, err := newDetector("./testdata/detect-ubuntu1404.toml")
	assert.Nil(t, err, "was able to create the transport")
	di, err := detector.Platform()
	require.NoError(t, err)

	assert.Equal(t, "ubuntu", di.Name, "os name should be identified")
	assert.Equal(t, "Ubuntu 14.04.6 LTS", di.Title, "os title should be identified")
	assert.Equal(t, "14.04", di.Version, "os version should be identified")
	assert.Equal(t, "x86_64", di.Arch, "os arch should be identified")
	assert.Equal(t, []string{"debian", "linux", "unix", "os"}, di.Family)
}

func TestUbuntu1604Detector(t *testing.T) {
	detector, err := newDetector("./testdata/detect-ubuntu1604.toml")
	assert.Nil(t, err, "was able to create the transport")
	di, err := detector.Platform()
	require.NoError(t, err)

	assert.Equal(t, "ubuntu", di.Name, "os name should be identified")
	assert.Equal(t, "Ubuntu 16.04.4 LTS", di.Title, "os title should be identified")
	assert.Equal(t, "16.04", di.Version, "os version should be identified")
	assert.Equal(t, "x86_64", di.Arch, "os arch should be identified")
	assert.Equal(t, []string{"debian", "linux", "unix", "os"}, di.Family)
}

func TestUbuntu1804Detector(t *testing.T) {
	detector, err := newDetector("./testdata/detect-ubuntu1804.toml")
	assert.Nil(t, err, "was able to create the transport")
	di, err := detector.Platform()
	require.NoError(t, err)

	assert.Equal(t, "ubuntu", di.Name, "os name should be identified")
	assert.Equal(t, "Ubuntu 18.04.3 LTS", di.Title, "os title should be identified")
	assert.Equal(t, "18.04", di.Version, "os version should be identified")
	assert.Equal(t, "x86_64", di.Arch, "os arch should be identified")
	assert.Equal(t, []string{"debian", "linux", "unix", "os"}, di.Family)
}

func TestUbuntu2004Detector(t *testing.T) {
	detector, err := newDetector("./testdata/detect-ubuntu2004.toml")
	assert.Nil(t, err, "was able to create the transport")
	di, err := detector.Platform()
	require.NoError(t, err)

	assert.Equal(t, "ubuntu", di.Name, "os name should be identified")
	assert.Equal(t, "Ubuntu Focal Fossa (development branch)", di.Title, "os title should be identified")
	assert.Equal(t, "20.04", di.Version, "os version should be identified")
	assert.Equal(t, "x86_64", di.Arch, "os arch should be identified")
	assert.Equal(t, []string{"debian", "linux", "unix", "os"}, di.Family)
}

func TestUbuntu2204Detector(t *testing.T) {
	detector, err := newDetector("./testdata/detect-ubuntu2204.toml")
	assert.Nil(t, err, "was able to create the transport")
	di, err := detector.Platform()
	require.NoError(t, err)

	assert.Equal(t, "ubuntu", di.Name, "os name should be identified")
	assert.Equal(t, "Ubuntu Jammy Jellyfish (development branch)", di.Title, "os title should be identified")
	assert.Equal(t, "22.04", di.Version, "os version should be identified")
	assert.Equal(t, "aarch64", di.Arch, "os arch should be identified")
}

func TestPoposDetector(t *testing.T) {
	detector, err := newDetector("./testdata/detect-popos.toml")
	assert.Nil(t, err, "was able to create the transport")
	di, err := detector.Platform()
	require.NoError(t, err)

	assert.Equal(t, "pop", di.Name, "os name should be identified")
	assert.Equal(t, "Pop!_OS 20.04 LTS", di.Title, "os title should be identified")
	assert.Equal(t, "20.04", di.Version, "os version should be identified")
	assert.Equal(t, "x86_64", di.Arch, "os arch should be identified")
	assert.Equal(t, []string{"debian", "linux", "unix", "os"}, di.Family)
}

func TestWindriver7Detector(t *testing.T) {
	detector, err := newDetector("./testdata/detect-windriver7.toml")
	assert.Nil(t, err, "was able to create the transport")
	di, err := detector.Platform()
	require.NoError(t, err)

	assert.Equal(t, "wrlinux", di.Name, "os name should be identified")
	assert.Equal(t, "Wind River Linux 7.0.0.2", di.Title, "os title should be identified")
	assert.Equal(t, "7.0.0.2", di.Version, "os version should be identified")
	assert.Equal(t, "x86_64", di.Arch, "os arch should be identified")
	assert.Equal(t, []string{"linux", "unix", "os"}, di.Family)
}

func TestOpenWrtDetector(t *testing.T) {
	detector, err := newDetector("./testdata/detect-openwrt.toml")
	assert.Nil(t, err, "was able to create the transport")
	di, err := detector.Platform()
	require.NoError(t, err)

	assert.Equal(t, "openwrt", di.Name, "os name should be identified")
	assert.Equal(t, "OpenWrt", di.Title, "os title should be identified")
	assert.Equal(t, "Bleeding Edge", di.Version, "os version should be identified")
	assert.Equal(t, "x86_64", di.Arch, "os arch should be identified")
	assert.Equal(t, []string{"linux", "unix", "os"}, di.Family)
}

func TestDebian7Detector(t *testing.T) {
	detector, err := newDetector("./testdata/detect-debian7.toml")
	assert.Nil(t, err, "was able to create the transport")
	di, err := detector.Platform()
	require.NoError(t, err)

	assert.Equal(t, "debian", di.Name, "os name should be identified")
	assert.Equal(t, "Debian GNU/Linux 7 (wheezy)", di.Title, "os title should be identified")
	assert.Equal(t, "7.11", di.Version, "os version should be identified")
	assert.Equal(t, "x86_64", di.Arch, "os arch should be identified")
	assert.Equal(t, []string{"debian", "linux", "unix", "os"}, di.Family)
}

func TestDebian8Detector(t *testing.T) {
	detector, err := newDetector("./testdata/detect-debian8.toml")
	assert.Nil(t, err, "was able to create the transport")
	di, err := detector.Platform()
	require.NoError(t, err)

	assert.Equal(t, "debian", di.Name, "os name should be identified")
	assert.Equal(t, "Debian GNU/Linux 8 (jessie)", di.Title, "os title should be identified")
	assert.Equal(t, "8.11", di.Version, "os version should be identified")
	assert.Equal(t, "x86_64", di.Arch, "os arch should be identified")
	assert.Equal(t, []string{"debian", "linux", "unix", "os"}, di.Family)
}

func TestDebian9Detector(t *testing.T) {
	detector, err := newDetector("./testdata/detect-debian9.toml")
	assert.Nil(t, err, "was able to create the transport")
	di, err := detector.Platform()
	require.NoError(t, err)

	assert.Equal(t, "debian", di.Name, "os name should be identified")
	assert.Equal(t, "Debian GNU/Linux 9 (stretch)", di.Title, "os title should be identified")
	assert.Equal(t, "9.4", di.Version, "os version should be identified")
	assert.Equal(t, "x86_64", di.Arch, "os arch should be identified")
	assert.Equal(t, []string{"debian", "linux", "unix", "os"}, di.Family)
}

func TestDebian10Detector(t *testing.T) {
	detector, err := newDetector("./testdata/detect-debian10.toml")
	assert.Nil(t, err, "was able to create the transport")
	di, err := detector.Platform()
	require.NoError(t, err)

	assert.Equal(t, "debian", di.Name, "os name should be identified")
	assert.Equal(t, "Debian GNU/Linux 10 (buster)", di.Title, "os title should be identified")
	assert.Equal(t, "10.0", di.Version, "os version should be identified")
	assert.Equal(t, "x86_64", di.Arch, "os arch should be identified")
	assert.Equal(t, []string{"debian", "linux", "unix", "os"}, di.Family)
}

func TestRaspian10Detector(t *testing.T) {
	detector, err := newDetector("./testdata/detect-raspbian.toml")
	assert.Nil(t, err, "was able to create the transport")
	di, err := detector.Platform()
	require.NoError(t, err)

	assert.Equal(t, "raspbian", di.Name, "os name should be identified")
	assert.Equal(t, "Raspbian GNU/Linux 10 (buster)", di.Title, "os title should be identified")
	assert.Equal(t, "10", di.Version, "os version should be identified")
	assert.Equal(t, "armv7l", di.Arch, "os arch should be identified")
	assert.Equal(t, []string{"debian", "linux", "unix", "os"}, di.Family)
}

func TestKaliRollingDetector(t *testing.T) {
	detector, err := newDetector("./testdata/detect-kalirolling.toml")
	assert.Nil(t, err, "was able to create the transport")
	di, err := detector.Platform()
	require.NoError(t, err)

	assert.Equal(t, "kali", di.Name, "os name should be identified")
	assert.Equal(t, "Kali GNU/Linux Rolling", di.Title, "os title should be identified")
	assert.Equal(t, "2019.4", di.Version, "os version should be identified")
	assert.Equal(t, "x86_64", di.Arch, "os arch should be identified")
	assert.Equal(t, []string{"debian", "linux", "unix", "os"}, di.Family)
}

func TestOpenSuse13Detector(t *testing.T) {
	detector, err := newDetector("./testdata/detect-opensuse-13.toml")
	assert.Nil(t, err, "was able to create the transport")
	di, err := detector.Platform()
	require.NoError(t, err)

	assert.Equal(t, "opensuse", di.Name, "os name should be identified")
	assert.Equal(t, "openSUSE 13.2 (Harlequin) (x86_64)", di.Title, "os title should be identified")
	assert.Equal(t, "13.2", di.Version, "os version should be identified")
	assert.Equal(t, "x86_64", di.Arch, "os arch should be identified")
	assert.Equal(t, []string{"suse", "linux", "unix", "os"}, di.Family)
}

func TestOpenSuseLeap42Detector(t *testing.T) {
	detector, err := newDetector("./testdata/detect-opensuse-leap-42.3.toml")
	assert.Nil(t, err, "was able to create the transport")
	di, err := detector.Platform()
	require.NoError(t, err)

	assert.Equal(t, "opensuse", di.Name, "os name should be identified")
	assert.Equal(t, "openSUSE Leap 42.3", di.Title, "os title should be identified")
	assert.Equal(t, "42.3", di.Version, "os version should be identified")
	assert.Equal(t, "x86_64", di.Arch, "os arch should be identified")
	assert.Equal(t, []string{"suse", "linux", "unix", "os"}, di.Family)
}

func TestOpenSuseLeap15Detector(t *testing.T) {
	detector, err := newDetector("./testdata/detect-opensuse-leap-15.toml")
	assert.Nil(t, err, "was able to create the transport")
	di, err := detector.Platform()
	require.NoError(t, err)

	assert.Equal(t, "opensuse-leap", di.Name, "os name should be identified")
	assert.Equal(t, "openSUSE Leap 15.0", di.Title, "os title should be identified")
	assert.Equal(t, "15.0", di.Version, "os version should be identified")
	assert.Equal(t, "x86_64", di.Arch, "os arch should be identified")
	assert.Equal(t, []string{"suse", "linux", "unix", "os"}, di.Family)
}

func TestOpenSuseTumbleweedDetector(t *testing.T) {
	detector, err := newDetector("./testdata/detect-opensuse-tumbleweed.toml")
	assert.Nil(t, err, "was able to create the transport")
	di, err := detector.Platform()
	require.NoError(t, err)

	assert.Equal(t, "opensuse-tumbleweed", di.Name, "os name should be identified")
	assert.Equal(t, "openSUSE Tumbleweed", di.Title, "os title should be identified")
	assert.Equal(t, "20200305", di.Version, "os version should be identified")
	assert.Equal(t, "x86_64", di.Arch, "os arch should be identified")
	assert.Equal(t, []string{"suse", "linux", "unix", "os"}, di.Family)
}

func TestSuse12Detector(t *testing.T) {
	detector, err := newDetector("./testdata/detect-suse-sles-12.toml")
	assert.Nil(t, err, "was able to create the transport")
	di, err := detector.Platform()
	require.NoError(t, err)

	assert.Equal(t, "sles", di.Name, "os name should be identified")
	assert.Equal(t, "SUSE Linux Enterprise Server 12 SP3", di.Title, "os title should be identified")
	assert.Equal(t, "12.3", di.Version, "os version should be identified")
	assert.Equal(t, "x86_64", di.Arch, "os arch should be identified")
	assert.Equal(t, []string{"suse", "linux", "unix", "os"}, di.Family)
}

func TestSuse125Detector(t *testing.T) {
	detector, err := newDetector("./testdata/detect-suse-sles-12.5.toml")
	assert.Nil(t, err, "was able to create the transport")
	di, err := detector.Platform()
	require.NoError(t, err)

	assert.Equal(t, "sles", di.Name, "os name should be identified")
	assert.Equal(t, "SUSE Linux Enterprise Server 12 SP5", di.Title, "os title should be identified")
	assert.Equal(t, "12.5", di.Version, "os version should be identified")
	assert.Equal(t, "x86_64", di.Arch, "os arch should be identified")
	assert.Equal(t, []string{"suse", "linux", "unix", "os"}, di.Family)
}

func TestSuse15Detector(t *testing.T) {
	detector, err := newDetector("./testdata/detect-suse-sles-15.toml")
	assert.Nil(t, err, "was able to create the transport")
	di, err := detector.Platform()
	require.NoError(t, err)

	assert.Equal(t, "sles", di.Name, "os name should be identified")
	assert.Equal(t, "SUSE Linux Enterprise Server 15 SP1", di.Title, "os title should be identified")
	assert.Equal(t, "15.1", di.Version, "os version should be identified")
	assert.Equal(t, "x86_64", di.Arch, "os arch should be identified")
	assert.Equal(t, []string{"suse", "linux", "unix", "os"}, di.Family)
}

func TestSuse5MicroDetector(t *testing.T) {
	detector, err := newDetector("./testdata/detect-suse-micro-5.toml")
	assert.Nil(t, err, "was able to create the transport")
	di, err := detector.Platform()
	require.NoError(t, err)

	assert.Equal(t, "suse-microos", di.Name, "os name should be identified")
	assert.Equal(t, "SUSE Linux Enterprise Micro 5.1", di.Title, "os title should be identified")
	assert.Equal(t, "5.1", di.Version, "os version should be identified")
	assert.Equal(t, "aarch64", di.Arch, "os arch should be identified")
	assert.Equal(t, []string{"suse", "linux", "unix", "os"}, di.Family)
}

func TestAmazon1LinuxDetector(t *testing.T) {
	detector, err := newDetector("./testdata/detect-amazonlinux-2017.09.toml")
	assert.Nil(t, err, "was able to create the transport")
	di, err := detector.Platform()
	require.NoError(t, err)

	assert.Equal(t, "amazonlinux", di.Name, "os name should be identified")
	assert.Equal(t, "Amazon Linux AMI 2017.09", di.Title, "os title should be identified")
	assert.Equal(t, "2017.09", di.Version, "os version should be identified")
	assert.Equal(t, "x86_64", di.Arch, "os arch should be identified")
	assert.Equal(t, []string{"linux", "unix", "os"}, di.Family)
}

func TestAmazon2LinuxDetector(t *testing.T) {
	detector, err := newDetector("./testdata/detect-amzn-2.toml")
	assert.Nil(t, err, "was able to create the transport")
	di, err := detector.Platform()
	require.NoError(t, err)

	assert.Equal(t, "amazonlinux", di.Name, "os name should be identified")
	assert.Equal(t, "Amazon Linux 2", di.Title, "os title should be identified")
	assert.Equal(t, "2", di.Version, "os version should be identified")
	assert.Equal(t, "x86_64", di.Arch, "os arch should be identified")
	assert.Equal(t, []string{"linux", "unix", "os"}, di.Family)
}

func TestAmazon2022LinuxDetector(t *testing.T) {
	detector, err := newDetector("./testdata/detect-amzn-2022.toml")
	assert.Nil(t, err, "was able to create the transport")
	di, err := detector.Platform()
	require.NoError(t, err)

	assert.Equal(t, "amazonlinux", di.Name, "os name should be identified")
	assert.Equal(t, "Amazon Linux 2022", di.Title, "os title should be identified")
	assert.Equal(t, "2022", di.Version, "os version should be identified")
	assert.Equal(t, "aarch64", di.Arch, "os arch should be identified")
	assert.Equal(t, []string{"linux", "unix", "os"}, di.Family)
}

func TestScientificLinuxDetector(t *testing.T) {
	detector, err := newDetector("./testdata/detect-scientific.toml")
	assert.Nil(t, err, "was able to create the transport")
	di, err := detector.Platform()
	require.NoError(t, err)

	assert.Equal(t, "scientific", di.Name, "os name should be identified")
	assert.Equal(t, "Scientific Linux CERN SLC", di.Title, "os title should be identified")
	assert.Equal(t, "6.9", di.Version, "os version should be identified")
	assert.Equal(t, "x86_64", di.Arch, "os arch should be identified")
	assert.Equal(t, []string{"redhat", "linux", "unix", "os"}, di.Family)
}

func TestArchLinuxVmDetector(t *testing.T) {
	detector, err := newDetector("./testdata/detect-arch-vm.toml")
	assert.Nil(t, err, "was able to create the transport")
	di, err := detector.Platform()
	require.NoError(t, err)

	assert.Equal(t, "arch", di.Name, "os name should be identified")
	assert.Equal(t, "Arch Linux", di.Title, "os title should be identified")
	assert.Equal(t, "x86_64", di.Arch, "os arch should be identified")
	assert.Equal(t, []string{"arch", "linux", "unix", "os"}, di.Family)
}

func TestArchLinuxContainerDetector(t *testing.T) {
	detector, err := newDetector("./testdata/detect-arch-container.toml")
	assert.Nil(t, err, "was able to create the transport")
	di, err := detector.Platform()
	require.NoError(t, err)

	assert.Equal(t, "arch", di.Name, "os name should be identified")
	assert.Equal(t, "Arch Linux", di.Title, "os title should be identified")
	assert.Equal(t, "x86_64", di.Arch, "os arch should be identified")
	assert.Equal(t, []string{"arch", "linux", "unix", "os"}, di.Family)
}

func TestManjaroLinuxContainerDetector(t *testing.T) {
	detector, err := newDetector("./testdata/detect-manjaro.toml")
	assert.Nil(t, err, "was able to create the transport")
	di, err := detector.Platform()
	require.NoError(t, err)

	assert.Equal(t, "manjaro", di.Name, "os name should be identified")
	assert.Equal(t, "Manjaro Linux", di.Title, "os title should be identified")
	assert.Equal(t, "x86_64", di.Arch, "os arch should be identified")
	assert.Equal(t, []string{"arch", "linux", "unix", "os"}, di.Family)
}

func TestOracleLinux6Detector(t *testing.T) {
	detector, err := newDetector("./testdata/detect-oracle6.toml")
	assert.Nil(t, err, "was able to create the transport")
	di, err := detector.Platform()
	require.NoError(t, err)

	assert.Equal(t, "oraclelinux", di.Name, "os name should be identified")
	assert.Equal(t, "Oracle Linux Server 6.9", di.Title, "os title should be identified")
	assert.Equal(t, "6.9", di.Version, "os version should be identified")
	assert.Equal(t, "x86_64", di.Arch, "os arch should be identified")
	assert.Equal(t, []string{"redhat", "linux", "unix", "os"}, di.Family)
}

func TestOracleLinux7Detector(t *testing.T) {
	detector, err := newDetector("./testdata/detect-oracle7.toml")
	assert.Nil(t, err, "was able to create the transport")
	di, err := detector.Platform()
	require.NoError(t, err)

	assert.Equal(t, "oraclelinux", di.Name, "os name should be identified")
	assert.Equal(t, "Oracle Linux Server 7.5", di.Title, "os title should be identified")
	assert.Equal(t, "7.5", di.Version, "os version should be identified")
	assert.Equal(t, "x86_64", di.Arch, "os arch should be identified")
	assert.Equal(t, []string{"redhat", "linux", "unix", "os"}, di.Family)
}

func TestOracleLinux8Detector(t *testing.T) {
	detector, err := newDetector("./testdata/detect-oracle8.toml")
	assert.Nil(t, err, "was able to create the transport")
	di, err := detector.Platform()
	require.NoError(t, err)

	assert.Equal(t, "oraclelinux", di.Name, "os name should be identified")
	assert.Equal(t, "Oracle Linux Server 8.0", di.Title, "os title should be identified")
	assert.Equal(t, "8.0", di.Version, "os version should be identified")
	assert.Equal(t, "x86_64", di.Arch, "os arch should be identified")
	assert.Equal(t, []string{"redhat", "linux", "unix", "os"}, di.Family)
}

func TestGentooLinuxDetector(t *testing.T) {
	detector, err := newDetector("./testdata/detect-gentoo.toml")
	assert.Nil(t, err, "was able to create the transport")
	di, err := detector.Platform()
	require.NoError(t, err)

	assert.Equal(t, "gentoo", di.Name, "os name should be identified")
	assert.Equal(t, "Gentoo/Linux", di.Title, "os title should be identified")
	assert.Equal(t, "2.4.1", di.Version, "os version should be identified")
	assert.Equal(t, "x86_64", di.Arch, "os arch should be identified")
	assert.Equal(t, []string{"linux", "unix", "os"}, di.Family)
}

func TestAlpineLinuxDetector(t *testing.T) {
	detector, err := newDetector("./testdata/detect-alpine.toml")
	assert.Nil(t, err, "was able to create the transport")
	di, err := detector.Platform()
	require.NoError(t, err)

	assert.Equal(t, "alpine", di.Name, "os name should be identified")
	assert.Equal(t, "Alpine Linux v3.7", di.Title, "os title should be identified")
	assert.Equal(t, "3.7.0", di.Version, "os version should be identified")
	assert.Equal(t, "x86_64", di.Arch, "os arch should be identified")
	assert.Equal(t, []string{"linux", "unix", "os"}, di.Family)
}

func TestAlpineEdgeLinuxDetector(t *testing.T) {
	detector, err := newDetector("./testdata/detect-alpine-edge.toml")
	assert.Nil(t, err, "was able to create the transport")
	di, err := detector.Platform()
	require.NoError(t, err)

	assert.Equal(t, "alpine", di.Name, "os name should be identified")
	assert.Equal(t, "Alpine Linux edge", di.Title, "os title should be identified")
	assert.Equal(t, "edge", di.Version, "os version should be identified")
	assert.Equal(t, "3.13.0_alpha20201218", di.Build, "os build should be identified")
	assert.Equal(t, "x86_64", di.Arch, "os arch should be identified")
	assert.Equal(t, []string{"linux", "unix", "os"}, di.Family)
}

func TestBusyboxLinuxDetector(t *testing.T) {
	detector, err := newDetector("./testdata/detect-busybox.toml")
	assert.Nil(t, err, "was able to create the transport")
	di, err := detector.Platform()
	require.NoError(t, err)

	assert.Equal(t, "busybox", di.Name, "os name should be identified")
	assert.Equal(t, "BusyBox", di.Title, "os title should be identified")
	assert.Equal(t, "v1.34.1", di.Version, "os version should be identified")
	assert.Equal(t, "x86_64", di.Arch, "os arch should be identified")
	assert.Equal(t, []string{"linux", "unix", "os"}, di.Family)
}

func TestWindows2016Detector(t *testing.T) {
	detector, err := newDetector("./testdata/detect-windows2016.toml")
	assert.Nil(t, err, "was able to create the transport")
	di, err := detector.Platform()
	require.NoError(t, err)

	assert.Equal(t, "windows", di.Name, "os name should be identified")
	assert.Equal(t, "Microsoft Windows Server 2016 Standard Evaluation", di.Title, "os title should be identified")
	assert.Equal(t, "14393", di.Version, "os version should be identified")
	assert.Equal(t, "64-bit", di.Arch, "os arch should be identified")
	assert.Equal(t, []string{"windows", "os"}, di.Family)
}

func TestWindows2019Detector(t *testing.T) {
	detector, err := newDetector("./testdata/detect-windows2019.toml")
	assert.Nil(t, err, "was able to create the transport")
	di, err := detector.Platform()
	require.NoError(t, err)

	assert.Equal(t, "windows", di.Name, "os name should be identified")
	assert.Equal(t, "Microsoft Windows Server 2019 Datacenter Evaluation", di.Title, "os title should be identified")
	assert.Equal(t, "17763", di.Version, "os version should be identified")
	assert.Equal(t, "720", di.Build, "os build version should be identified")
	assert.Equal(t, "64-bit", di.Arch, "os arch should be identified")
	assert.Equal(t, []string{"windows", "os"}, di.Family)
}

func TestPhoton1Detector(t *testing.T) {
	detector, err := newDetector("./testdata/detect-photon1.toml")
	assert.Nil(t, err, "was able to create the transport")
	di, err := detector.Platform()
	require.NoError(t, err)

	assert.Equal(t, "photon", di.Name, "os name should be identified")
	assert.Equal(t, "VMware Photon/Linux", di.Title, "os title should be identified")
	assert.Equal(t, "1.0", di.Version, "os version should be identified")
	assert.Equal(t, "x86_64", di.Arch, "os arch should be identified")
	assert.Equal(t, []string{"linux", "unix", "os"}, di.Family)
}

func TestPhoton2Detector(t *testing.T) {
	detector, err := newDetector("./testdata/detect-photon2.toml")
	assert.Nil(t, err, "was able to create the transport")
	di, err := detector.Platform()
	require.NoError(t, err)

	assert.Equal(t, "photon", di.Name, "os name should be identified")
	assert.Equal(t, "VMware Photon OS/Linux", di.Title, "os title should be identified")
	assert.Equal(t, "2.0", di.Version, "os version should be identified")
	assert.Equal(t, "x86_64", di.Arch, "os arch should be identified")
	assert.Equal(t, []string{"linux", "unix", "os"}, di.Family)
}

func TestPhoton3Detector(t *testing.T) {
	detector, err := newDetector("./testdata/detect-photon3.toml")
	assert.Nil(t, err, "was able to create the transport")
	di, err := detector.Platform()
	require.NoError(t, err)

	assert.Equal(t, "photon", di.Name, "os name should be identified")
	assert.Equal(t, "VMware Photon OS/Linux", di.Title, "os title should be identified")
	assert.Equal(t, "3.0", di.Version, "os version should be identified")
	assert.Equal(t, "x86_64", di.Arch, "os arch should be identified")
	assert.Equal(t, []string{"linux", "unix", "os"}, di.Family)
}

func TestMacOSsDetector(t *testing.T) {
	detector, err := newDetector("./testdata/detect-macos.toml")
	assert.Nil(t, err, "was able to create the transport")
	di, err := detector.Platform()
	require.NoError(t, err)

	assert.Equal(t, "macos", di.Name, "os name should be identified")
	assert.Equal(t, "Mac OS X", di.Title, "os title should be identified")
	assert.Equal(t, "10.14.5", di.Version, "os version should be identified")
	assert.Equal(t, "x86_64", di.Arch, "os arch should be identified")
	assert.Equal(t, []string{"darwin", "bsd", "unix", "os"}, di.Family)
}

func TestBuildrootDetector(t *testing.T) {
	detector, err := newDetector("./testdata/detect-buildroot.toml")
	assert.Nil(t, err, "was able to create the transport")
	di, err := detector.Platform()
	require.NoError(t, err)

	assert.Equal(t, "buildroot", di.Name, "os name should be identified")
	assert.Equal(t, "Buildroot 2019.02.9", di.Title, "os title should be identified")
	assert.Equal(t, "2019.02.9", di.Version, "os version should be identified")
	assert.Equal(t, "x86_64", di.Arch, "os arch should be identified")
	assert.Equal(t, []string{"linux", "unix", "os"}, di.Family)
}

func TestSolaris11Detector(t *testing.T) {
	detector, err := newDetector("./testdata/detect-solaris11.toml")
	assert.Nil(t, err, "was able to create the transport")
	di, err := detector.Platform()
	require.NoError(t, err)

	assert.Equal(t, "solaris", di.Name, "os name should be identified")
	assert.Equal(t, "Oracle Solaris", di.Title, "os title should be identified")
	assert.Equal(t, "11.1", di.Version, "os version should be identified")
	assert.Equal(t, "i86pc", di.Arch, "os arch should be identified")
	assert.Equal(t, []string{"unix", "os"}, di.Family)
}

func TestNetbsd8Detector(t *testing.T) {
	detector, err := newDetector("./testdata/detect-netbsd8.toml")
	assert.Nil(t, err, "was able to create the transport")
	di, err := detector.Platform()
	require.NoError(t, err)

	assert.Equal(t, "netbsd", di.Name, "os name should be identified")
	assert.Equal(t, "NetBSD", di.Title, "os title should be identified")
	assert.Equal(t, "8.0", di.Version, "os version should be identified")
	assert.Equal(t, "amd64", di.Arch, "os arch should be identified")
	assert.Equal(t, []string{"bsd", "unix", "os"}, di.Family)
}

func TestFreebsd12Detector(t *testing.T) {
	detector, err := newDetector("./testdata/detect-freebsd12.toml")
	assert.Nil(t, err, "was able to create the transport")
	di, err := detector.Platform()
	require.NoError(t, err)

	assert.Equal(t, "freebsd", di.Name, "os name should be identified")
	assert.Equal(t, "FreeBSD", di.Title, "os title should be identified")
	assert.Equal(t, "12.0-CURRENT", di.Version, "os version should be identified")
	assert.Equal(t, "amd64", di.Arch, "os arch should be identified")
	assert.Equal(t, []string{"bsd", "unix", "os"}, di.Family)
}

func TestOpenBsd6Detector(t *testing.T) {
	detector, err := newDetector("./testdata/detect-openbsd6.toml")
	assert.Nil(t, err, "was able to create the transport")
	di, err := detector.Platform()
	require.NoError(t, err)

	assert.Equal(t, "openbsd", di.Name, "os name should be identified")
	assert.Equal(t, "OpenBSD", di.Title, "os title should be identified")
	assert.Equal(t, "6.7", di.Version, "os version should be identified")
	assert.Equal(t, "amd64", di.Arch, "os arch should be identified")
	assert.Equal(t, []string{"bsd", "unix", "os"}, di.Family)
}

func TestDragonFlyBsd5Detector(t *testing.T) {
	detector, err := newDetector("./testdata/detect-dragonflybsd5.toml")
	assert.Nil(t, err, "was able to create the transport")
	di, err := detector.Platform()
	require.NoError(t, err)

	assert.Equal(t, "dragonflybsd", di.Name, "os name should be identified")
	assert.Equal(t, "DragonFly", di.Title, "os title should be identified")
	assert.Equal(t, "5.8-RELEASE", di.Version, "os version should be identified")
	assert.Equal(t, "x86_64", di.Arch, "os arch should be identified")
	assert.Equal(t, []string{"bsd", "unix", "os"}, di.Family)
}

func TestMint20Detector(t *testing.T) {
	detector, err := newDetector("./testdata/detect-mint20.toml")
	assert.Nil(t, err, "was able to create the transport")
	di, err := detector.Platform()
	require.NoError(t, err)

	assert.Equal(t, "linuxmint", di.Name, "os name should be identified")
	assert.Equal(t, "Linux Mint 20", di.Title, "os title should be identified")
	assert.Equal(t, "20", di.Version, "os version should be identified")
	assert.Equal(t, "x86_64", di.Arch, "os arch should be identified")
	assert.Equal(t, []string{"debian", "linux", "unix", "os"}, di.Family)
}

func TestGoogleCOSDetector(t *testing.T) {
	detector, err := newDetector("./testdata/google-cos.toml")
	assert.Nil(t, err, "was able to create the transport")
	di, err := detector.Platform()
	require.NoError(t, err)

	assert.Equal(t, "cos", di.Name, "os name should be identified")
	assert.Equal(t, "Container-Optimized OS from Google", di.Title, "os title should be identified")
	assert.Equal(t, "97", di.Version, "os version should be identified")
	assert.Equal(t, "x86_64", di.Arch, "os arch should be identified")
	assert.Equal(t, "16919.103.16", di.Build, "os build should be identified")
	assert.Equal(t, []string{"linux", "unix", "os"}, di.Family)
}

func TestFamilies(t *testing.T) {
	di := &platform.Platform{}
	di.Family = []string{"unix", "bsd", "darwin"}

	assert.Equal(t, true, di.IsFamily("unix"), "unix should be a family")
	assert.Equal(t, true, di.IsFamily("bsd"), "bsd should be a family")
	assert.Equal(t, true, di.IsFamily("darwin"), "darwin should be a family")
}

type TestContainer struct {
	Name  string
	Image string
}
