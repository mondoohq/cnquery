package platform_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	mock "go.mondoo.io/mondoo/motor/motoros/mock/toml"
	"go.mondoo.io/mondoo/motor/motoros/platform"
	"go.mondoo.io/mondoo/motor/motoros/types"
)

type OsDetectTestSuite struct {
	suite.Suite
}

func (suite *OsDetectTestSuite) SetupSuite() {}

func newDetector(path string) (*platform.Detector, error) {
	mock, err := mock.New(&types.Endpoint{Backend: "mock", Path: path})
	if err != nil {
		return nil, err
	}
	detector := &platform.Detector{Transport: mock}
	return detector, nil
}

func (suite *OsDetectTestSuite) TestRhel6OSDetector() {
	detector, err := newDetector("./testdata/detect-rhel6.toml")
	assert.Nil(suite.T(), err, "was able to create the transport")
	resolved, di := detector.Resolve()

	assert.Equal(suite.T(), true, resolved, "platform should be resolvable")
	assert.Equal(suite.T(), "redhat", di.Name, "os name should be identified")
	assert.Equal(suite.T(), "Red Hat Linux", di.Title, "os title should be identified")
	assert.Equal(suite.T(), "6.2", di.Release, "os version should be identified")
	assert.Equal(suite.T(), "x86_64", di.Arch, "os arch should be identified")
	assert.Equal(suite.T(), []string{"redhat", "linux", "unix", "os"}, di.Family)
}

func (suite *OsDetectTestSuite) TestRhel7OSDetector() {
	detector, err := newDetector("./testdata/detect-rhel7.toml")
	assert.Nil(suite.T(), err, "was able to create the transport")
	resolved, di := detector.Resolve()

	assert.Equal(suite.T(), true, resolved, "platform should be resolvable")
	assert.Equal(suite.T(), "redhat", di.Name, "os name should be identified")
	assert.Equal(suite.T(), "Red Hat Enterprise Linux Server", di.Title, "os title should be identified")
	assert.Equal(suite.T(), "7.2", di.Release, "os version should be identified")
	assert.Equal(suite.T(), "x86_64", di.Arch, "os arch should be identified")
	assert.Equal(suite.T(), []string{"redhat", "linux", "unix", "os"}, di.Family)
}

func (suite *OsDetectTestSuite) TestRhel7SLESOSDetector() {
	detector, err := newDetector("./testdata/detect-rhel7-sles.toml")
	assert.Nil(suite.T(), err, "was able to create the transport")
	resolved, di := detector.Resolve()

	assert.Equal(suite.T(), true, resolved, "platform should be resolvable")
	assert.Equal(suite.T(), "redhat", di.Name, "os name should be identified")
	assert.Equal(suite.T(), "Red Hat Enterprise Linux Server", di.Title, "os title should be identified")
	assert.Equal(suite.T(), "7.4", di.Release, "os version should be identified")
	assert.Equal(suite.T(), "x86_64", di.Arch, "os arch should be identified")
	assert.Equal(suite.T(), []string{"redhat", "linux", "unix", "os"}, di.Family)
}

func (suite *OsDetectTestSuite) TestRhel8OSDetector() {
	detector, err := newDetector("./testdata/detect-rhel8.toml")
	assert.Nil(suite.T(), err, "was able to create the transport")
	resolved, di := detector.Resolve()

	assert.Equal(suite.T(), true, resolved, "platform should be resolvable")
	assert.Equal(suite.T(), "redhat", di.Name, "os name should be identified")
	assert.Equal(suite.T(), "Red Hat Enterprise Linux", di.Title, "os title should be identified")
	assert.Equal(suite.T(), "8.0", di.Release, "os version should be identified")
	assert.Equal(suite.T(), "x86_64", di.Arch, "os arch should be identified")
	assert.Equal(suite.T(), []string{"redhat", "linux", "unix", "os"}, di.Family)
}

func (suite *OsDetectTestSuite) TestFedora29OSDetector() {
	detector, err := newDetector("./testdata/detect-fedora29.toml")
	assert.Nil(suite.T(), err, "was able to create the transport")
	resolved, di := detector.Resolve()

	assert.Equal(suite.T(), true, resolved, "platform should be resolvable")
	assert.Equal(suite.T(), "fedora", di.Name, "os name should be identified")
	assert.Equal(suite.T(), "Fedora", di.Title, "os title should be identified")
	assert.Equal(suite.T(), "29", di.Release, "os version should be identified")
	assert.Equal(suite.T(), "x86_64", di.Arch, "os arch should be identified")
	assert.Equal(suite.T(), []string{"redhat", "linux", "unix", "os"}, di.Family)
}

func (suite *OsDetectTestSuite) TestCentos6Detector() {
	detector, err := newDetector("./testdata/detect-centos6.toml")
	assert.Nil(suite.T(), err, "was able to create the transport")
	resolved, di := detector.Resolve()

	assert.Equal(suite.T(), true, resolved, "platform should be resolvable")
	assert.Equal(suite.T(), "centos", di.Name, "os name should be identified")
	assert.Equal(suite.T(), "CentOS", di.Title, "os title should be identified")
	assert.Equal(suite.T(), "6.9", di.Release, "os version should be identified")
	assert.Equal(suite.T(), "x86_64", di.Arch, "os arch should be identified")
	assert.Equal(suite.T(), []string{"redhat", "linux", "unix", "os"}, di.Family)
}

func (suite *OsDetectTestSuite) TestCentos7OSDetector() {
	detector, err := newDetector("./testdata/detect-centos7.toml")
	assert.Nil(suite.T(), err, "was able to create the transport")
	resolved, di := detector.Resolve()

	assert.Equal(suite.T(), true, resolved, "platform should be resolvable")
	assert.Equal(suite.T(), "centos", di.Name, "os name should be identified")
	assert.Equal(suite.T(), "CentOS Linux", di.Title, "os title should be identified")
	assert.Equal(suite.T(), "7.5.1804", di.Release, "os version should be identified")
	assert.Equal(suite.T(), "x86_64", di.Arch, "os arch should be identified")
	assert.Equal(suite.T(), []string{"redhat", "linux", "unix", "os"}, di.Family)
}

func (suite *OsDetectTestSuite) TestCentos5Detector() {
	detector, err := newDetector("./testdata/detect-centos5.toml")
	assert.Nil(suite.T(), err, "was able to create the transport")
	resolved, di := detector.Resolve()

	assert.Equal(suite.T(), true, resolved, "platform should be resolvable")
	assert.Equal(suite.T(), "centos", di.Name, "os name should be identified")
	assert.Equal(suite.T(), "CentOS", di.Title, "os title should be identified")
	assert.Equal(suite.T(), "5.11", di.Release, "os version should be identified")
	assert.Equal(suite.T(), "x86_64", di.Arch, "os arch should be identified")
	assert.Equal(suite.T(), []string{"redhat", "linux", "unix", "os"}, di.Family)
}

func (suite *OsDetectTestSuite) TestUbuntu1204Detector() {
	detector, err := newDetector("./testdata/detect-ubuntu1204.toml")
	assert.Nil(suite.T(), err, "was able to create the transport")
	resolved, di := detector.Resolve()

	assert.Equal(suite.T(), true, resolved, "platform should be resolvable")
	assert.Equal(suite.T(), "ubuntu", di.Name, "os name should be identified")
	assert.Equal(suite.T(), "Ubuntu", di.Title, "os title should be identified")
	assert.Equal(suite.T(), "12.04", di.Release, "os version should be identified")
	assert.Equal(suite.T(), "x86_64", di.Arch, "os arch should be identified")
	assert.Equal(suite.T(), []string{"debian", "linux", "unix", "os"}, di.Family)
}

func (suite *OsDetectTestSuite) TestUbuntu1404Detector() {
	detector, err := newDetector("./testdata/detect-ubuntu1404.toml")
	assert.Nil(suite.T(), err, "was able to create the transport")
	resolved, di := detector.Resolve()

	assert.Equal(suite.T(), true, resolved, "platform should be resolvable")
	assert.Equal(suite.T(), "ubuntu", di.Name, "os name should be identified")
	assert.Equal(suite.T(), "Ubuntu", di.Title, "os title should be identified")
	assert.Equal(suite.T(), "14.04", di.Release, "os version should be identified")
	assert.Equal(suite.T(), "x86_64", di.Arch, "os arch should be identified")
	assert.Equal(suite.T(), []string{"debian", "linux", "unix", "os"}, di.Family)
}

func (suite *OsDetectTestSuite) TestUbuntu1604Detector() {
	detector, err := newDetector("./testdata/detect-ubuntu1604.toml")
	assert.Nil(suite.T(), err, "was able to create the transport")
	resolved, di := detector.Resolve()

	assert.Equal(suite.T(), true, resolved, "platform should be resolvable")
	assert.Equal(suite.T(), "ubuntu", di.Name, "os name should be identified")
	assert.Equal(suite.T(), "Ubuntu", di.Title, "os title should be identified")
	assert.Equal(suite.T(), "16.04", di.Release, "os version should be identified")
	assert.Equal(suite.T(), "x86_64", di.Arch, "os arch should be identified")
	assert.Equal(suite.T(), []string{"debian", "linux", "unix", "os"}, di.Family)
}

func (suite *OsDetectTestSuite) TestUbuntu1804Detector() {
	detector, err := newDetector("./testdata/detect-ubuntu1804.toml")
	assert.Nil(suite.T(), err, "was able to create the transport")
	resolved, di := detector.Resolve()

	assert.Equal(suite.T(), true, resolved, "platform should be resolvable")
	assert.Equal(suite.T(), "ubuntu", di.Name, "os name should be identified")
	assert.Equal(suite.T(), "Ubuntu", di.Title, "os title should be identified")
	assert.Equal(suite.T(), "18.04", di.Release, "os version should be identified")
	assert.Equal(suite.T(), "x86_64", di.Arch, "os arch should be identified")
	assert.Equal(suite.T(), []string{"debian", "linux", "unix", "os"}, di.Family)
}

func (suite *OsDetectTestSuite) TestDebian7Detector() {
	detector, err := newDetector("./testdata/detect-debian7.toml")
	assert.Nil(suite.T(), err, "was able to create the transport")
	resolved, di := detector.Resolve()

	assert.Equal(suite.T(), true, resolved, "platform should be resolvable")
	assert.Equal(suite.T(), "debian", di.Name, "os name should be identified")
	assert.Equal(suite.T(), "Debian GNU/Linux", di.Title, "os title should be identified")
	assert.Equal(suite.T(), "7.11", di.Release, "os version should be identified")
	assert.Equal(suite.T(), "x86_64", di.Arch, "os arch should be identified")
	assert.Equal(suite.T(), []string{"debian", "linux", "unix", "os"}, di.Family)
}

func (suite *OsDetectTestSuite) TestDebian8Detector() {
	detector, err := newDetector("./testdata/detect-debian8.toml")
	assert.Nil(suite.T(), err, "was able to create the transport")
	resolved, di := detector.Resolve()

	assert.Equal(suite.T(), true, resolved, "platform should be resolvable")
	assert.Equal(suite.T(), "debian", di.Name, "os name should be identified")
	assert.Equal(suite.T(), "Debian GNU/Linux", di.Title, "os title should be identified")
	assert.Equal(suite.T(), "8.11", di.Release, "os version should be identified")
	assert.Equal(suite.T(), "x86_64", di.Arch, "os arch should be identified")
	assert.Equal(suite.T(), []string{"debian", "linux", "unix", "os"}, di.Family)
}

func (suite *OsDetectTestSuite) TestDebian9Detector() {
	detector, err := newDetector("./testdata/detect-debian9.toml")
	assert.Nil(suite.T(), err, "was able to create the transport")
	resolved, di := detector.Resolve()

	assert.Equal(suite.T(), true, resolved, "platform should be resolvable")
	assert.Equal(suite.T(), "debian", di.Name, "os name should be identified")
	assert.Equal(suite.T(), "Debian GNU/Linux", di.Title, "os title should be identified")
	assert.Equal(suite.T(), "9.4", di.Release, "os version should be identified")
	assert.Equal(suite.T(), "x86_64", di.Arch, "os arch should be identified")
	assert.Equal(suite.T(), []string{"debian", "linux", "unix", "os"}, di.Family)
}

func (suite *OsDetectTestSuite) TestDebian10Detector() {
	detector, err := newDetector("./testdata/detect-debian10.toml")
	assert.Nil(suite.T(), err, "was able to create the transport")
	resolved, di := detector.Resolve()

	assert.Equal(suite.T(), true, resolved, "platform should be resolvable")
	assert.Equal(suite.T(), "debian", di.Name, "os name should be identified")
	assert.Equal(suite.T(), "Debian GNU/Linux", di.Title, "os title should be identified")
	assert.Equal(suite.T(), "10.0", di.Release, "os version should be identified")
	assert.Equal(suite.T(), "x86_64", di.Arch, "os arch should be identified")
	assert.Equal(suite.T(), []string{"debian", "linux", "unix", "os"}, di.Family)
}

func (suite *OsDetectTestSuite) TestRaspian10Detector() {
	detector, err := newDetector("./testdata/detect-raspbian.toml")
	assert.Nil(suite.T(), err, "was able to create the transport")
	resolved, di := detector.Resolve()

	assert.Equal(suite.T(), true, resolved, "platform should be resolvable")
	assert.Equal(suite.T(), "raspbian", di.Name, "os name should be identified")
	assert.Equal(suite.T(), "Raspbian GNU/Linux", di.Title, "os title should be identified")
	assert.Equal(suite.T(), "10", di.Release, "os version should be identified")
	assert.Equal(suite.T(), "armv7l", di.Arch, "os arch should be identified")
	assert.Equal(suite.T(), []string{"debian", "linux", "unix", "os"}, di.Family)
}

func (suite *OsDetectTestSuite) TestOpenSuseLeap42Detector() {
	detector, err := newDetector("./testdata/detect-opensuse-leap-42.3.toml")
	assert.Nil(suite.T(), err, "was able to create the transport")
	resolved, di := detector.Resolve()

	assert.Equal(suite.T(), true, resolved, "platform should be resolvable")
	assert.Equal(suite.T(), "opensuse", di.Name, "os name should be identified")
	assert.Equal(suite.T(), "openSUSE Leap", di.Title, "os title should be identified")
	assert.Equal(suite.T(), "42.3", di.Release, "os version should be identified")
	assert.Equal(suite.T(), "x86_64", di.Arch, "os arch should be identified")
	assert.Equal(suite.T(), []string{"suse", "linux", "unix", "os"}, di.Family)
}

func (suite *OsDetectTestSuite) TestOpenSuseLeap15Detector() {
	detector, err := newDetector("./testdata/detect-opensuse-leap-15.toml")
	assert.Nil(suite.T(), err, "was able to create the transport")
	resolved, di := detector.Resolve()

	assert.Equal(suite.T(), true, resolved, "platform should be resolvable")
	assert.Equal(suite.T(), "opensuse-leap", di.Name, "os name should be identified")
	assert.Equal(suite.T(), "openSUSE Leap", di.Title, "os title should be identified")
	assert.Equal(suite.T(), "15.0", di.Release, "os version should be identified")
	assert.Equal(suite.T(), "x86_64", di.Arch, "os arch should be identified")
	assert.Equal(suite.T(), []string{"suse", "linux", "unix", "os"}, di.Family)
}

func (suite *OsDetectTestSuite) TestSuse12Detector() {
	detector, err := newDetector("./testdata/detect-suse-12.toml")
	assert.Nil(suite.T(), err, "was able to create the transport")
	resolved, di := detector.Resolve()

	assert.Equal(suite.T(), true, resolved, "platform should be resolvable")
	assert.Equal(suite.T(), "sles", di.Name, "os name should be identified")
	assert.Equal(suite.T(), "SLES", di.Title, "os title should be identified")
	assert.Equal(suite.T(), "12.3", di.Release, "os version should be identified")
	assert.Equal(suite.T(), "x86_64", di.Arch, "os arch should be identified")
	assert.Equal(suite.T(), []string{"suse", "linux", "unix", "os"}, di.Family)
}

func (suite *OsDetectTestSuite) TestAmazonLinuxDetector() {
	detector, err := newDetector("./testdata/detect-amazonlinux-2017.09.toml")
	assert.Nil(suite.T(), err, "was able to create the transport")
	resolved, di := detector.Resolve()

	assert.Equal(suite.T(), true, resolved, "platform should be resolvable")
	assert.Equal(suite.T(), "amzn", di.Name, "os name should be identified")
	assert.Equal(suite.T(), "Amazon Linux AMI", di.Title, "os title should be identified")
	assert.Equal(suite.T(), "2017.09", di.Release, "os version should be identified")
	assert.Equal(suite.T(), "x86_64", di.Arch, "os arch should be identified")
	assert.Equal(suite.T(), []string{"linux", "unix", "os"}, di.Family)
}

func (suite *OsDetectTestSuite) TestScientificLinuxDetector() {
	detector, err := newDetector("./testdata/detect-scientific.toml")
	assert.Nil(suite.T(), err, "was able to create the transport")
	resolved, di := detector.Resolve()

	assert.Equal(suite.T(), true, resolved, "platform should be resolvable")
	assert.Equal(suite.T(), "scientific", di.Name, "os name should be identified")
	assert.Equal(suite.T(), "Scientific Linux CERN SLC", di.Title, "os title should be identified")
	assert.Equal(suite.T(), "6.9", di.Release, "os version should be identified")
	assert.Equal(suite.T(), "x86_64", di.Arch, "os arch should be identified")
	assert.Equal(suite.T(), []string{"redhat", "linux", "unix", "os"}, di.Family)
}

func (suite *OsDetectTestSuite) TestArchLinuxDetector() {
	detector, err := newDetector("./testdata/detect-arch.toml")
	assert.Nil(suite.T(), err, "was able to create the transport")
	resolved, di := detector.Resolve()

	assert.Equal(suite.T(), true, resolved, "platform should be resolvable")
	assert.Equal(suite.T(), "arch", di.Name, "os name should be identified")
	assert.Equal(suite.T(), "Arch Linux", di.Title, "os title should be identified")
	assert.Equal(suite.T(), "x86_64", di.Arch, "os arch should be identified")
	assert.Equal(suite.T(), []string{"linux", "unix", "os"}, di.Family)
}

func (suite *OsDetectTestSuite) TestOracleLinux6Detector() {
	detector, err := newDetector("./testdata/detect-oracle6.toml")
	assert.Nil(suite.T(), err, "was able to create the transport")
	resolved, di := detector.Resolve()

	assert.Equal(suite.T(), true, resolved, "platform should be resolvable")
	assert.Equal(suite.T(), "ol", di.Name, "os name should be identified")
	assert.Equal(suite.T(), "Oracle Linux Server", di.Title, "os title should be identified")
	assert.Equal(suite.T(), "6.9", di.Release, "os version should be identified")
	assert.Equal(suite.T(), "x86_64", di.Arch, "os arch should be identified")
	assert.Equal(suite.T(), []string{"redhat", "linux", "unix", "os"}, di.Family)
}

func (suite *OsDetectTestSuite) TestOracleLinux7Detector() {
	detector, err := newDetector("./testdata/detect-oracle7.toml")
	assert.Nil(suite.T(), err, "was able to create the transport")
	resolved, di := detector.Resolve()

	assert.Equal(suite.T(), true, resolved, "platform should be resolvable")
	assert.Equal(suite.T(), "ol", di.Name, "os name should be identified")
	assert.Equal(suite.T(), "Oracle Linux Server", di.Title, "os title should be identified")
	assert.Equal(suite.T(), "7.5", di.Release, "os version should be identified")
	assert.Equal(suite.T(), "x86_64", di.Arch, "os arch should be identified")
	assert.Equal(suite.T(), []string{"redhat", "linux", "unix", "os"}, di.Family)
}

func (suite *OsDetectTestSuite) TestOracleLinux8Detector() {
	detector, err := newDetector("./testdata/detect-oracle8.toml")
	assert.Nil(suite.T(), err, "was able to create the transport")
	resolved, di := detector.Resolve()

	assert.Equal(suite.T(), true, resolved, "platform should be resolvable")
	assert.Equal(suite.T(), "ol", di.Name, "os name should be identified")
	assert.Equal(suite.T(), "Oracle Linux Server", di.Title, "os title should be identified")
	assert.Equal(suite.T(), "8.0", di.Release, "os version should be identified")
	assert.Equal(suite.T(), "x86_64", di.Arch, "os arch should be identified")
	assert.Equal(suite.T(), []string{"redhat", "linux", "unix", "os"}, di.Family)
}

func (suite *OsDetectTestSuite) TestGentooLinuxDetector() {
	detector, err := newDetector("./testdata/detect-gentoo.toml")
	assert.Nil(suite.T(), err, "was able to create the transport")
	resolved, di := detector.Resolve()

	assert.Equal(suite.T(), true, resolved, "platform should be resolvable")
	assert.Equal(suite.T(), "gentoo", di.Name, "os name should be identified")
	assert.Equal(suite.T(), "Gentoo", di.Title, "os title should be identified")
	assert.Equal(suite.T(), "2.4.1", di.Release, "os version should be identified")
	assert.Equal(suite.T(), "x86_64", di.Arch, "os arch should be identified")
	assert.Equal(suite.T(), []string{"linux", "unix", "os"}, di.Family)
}

func (suite *OsDetectTestSuite) TestAlpineLinuxDetector() {
	detector, err := newDetector("./testdata/detect-alpine.toml")
	assert.Nil(suite.T(), err, "was able to create the transport")
	resolved, di := detector.Resolve()

	assert.Equal(suite.T(), true, resolved, "platform should be resolvable")
	assert.Equal(suite.T(), "alpine", di.Name, "os name should be identified")
	assert.Equal(suite.T(), "Alpine Linux", di.Title, "os title should be identified")
	assert.Equal(suite.T(), "3.7.0", di.Release, "os version should be identified")
	assert.Equal(suite.T(), "x86_64", di.Arch, "os arch should be identified")
	assert.Equal(suite.T(), []string{"linux", "unix", "os"}, di.Family)
}

func (suite *OsDetectTestSuite) TestBusyboxLinuxDetector() {
	detector, err := newDetector("./testdata/detect-busybox.toml")
	assert.Nil(suite.T(), err, "was able to create the transport")
	resolved, di := detector.Resolve()

	assert.Equal(suite.T(), true, resolved, "platform should be resolvable")
	assert.Equal(suite.T(), "busybox", di.Name, "os name should be identified")
	assert.Equal(suite.T(), "BusyBox", di.Title, "os title should be identified")
	assert.Equal(suite.T(), "v1.28.4", di.Release, "os version should be identified")
	assert.Equal(suite.T(), "x86_64", di.Arch, "os arch should be identified")
	assert.Equal(suite.T(), []string{"linux", "unix", "os"}, di.Family)
}

func (suite *OsDetectTestSuite) TestWindows2016Detector() {
	detector, err := newDetector("./testdata/detect-windows2016.toml")
	assert.Nil(suite.T(), err, "was able to create the transport")
	resolved, di := detector.Resolve()

	assert.Equal(suite.T(), true, resolved, "platform should be resolvable")
	assert.Equal(suite.T(), "windows", di.Name, "os name should be identified")
	assert.Equal(suite.T(), "Microsoft Windows Server 2016 Standard Evaluation", di.Title, "os title should be identified")
	assert.Equal(suite.T(), "14393", di.Release, "os version should be identified")
	assert.Equal(suite.T(), "64-bit", di.Arch, "os arch should be identified")
	assert.Equal(suite.T(), []string{"windows", "os"}, di.Family)
}

func (suite *OsDetectTestSuite) TestWindows2019Detector() {
	detector, err := newDetector("./testdata/detect-windows2019.toml")
	assert.Nil(suite.T(), err, "was able to create the transport")
	resolved, di := detector.Resolve()

	assert.Equal(suite.T(), true, resolved, "platform should be resolvable")
	assert.Equal(suite.T(), "windows", di.Name, "os name should be identified")
	assert.Equal(suite.T(), "Microsoft Windows Server 2019 Datacenter Evaluation", di.Title, "os title should be identified")
	assert.Equal(suite.T(), "17763.720", di.Release, "os version should be identified")
	assert.Equal(suite.T(), "64-bit", di.Arch, "os arch should be identified")
	assert.Equal(suite.T(), []string{"windows", "os"}, di.Family)
}

func (suite *OsDetectTestSuite) TestPhoton1Detector() {
	detector, err := newDetector("./testdata/detect-photon1.toml")
	assert.Nil(suite.T(), err, "was able to create the transport")
	resolved, di := detector.Resolve()

	assert.Equal(suite.T(), true, resolved, "platform should be resolvable")
	assert.Equal(suite.T(), "photon", di.Name, "os name should be identified")
	assert.Equal(suite.T(), "VMware Photon", di.Title, "os title should be identified")
	assert.Equal(suite.T(), "1.0", di.Release, "os version should be identified")
	assert.Equal(suite.T(), "x86_64", di.Arch, "os arch should be identified")
	assert.Equal(suite.T(), []string{"linux", "unix", "os"}, di.Family)
}

func (suite *OsDetectTestSuite) TestPhoton2Detector() {
	detector, err := newDetector("./testdata/detect-photon2.toml")
	assert.Nil(suite.T(), err, "was able to create the transport")
	resolved, di := detector.Resolve()

	assert.Equal(suite.T(), true, resolved, "platform should be resolvable")
	assert.Equal(suite.T(), "photon", di.Name, "os name should be identified")
	assert.Equal(suite.T(), "VMware Photon OS", di.Title, "os title should be identified")
	assert.Equal(suite.T(), "2.0", di.Release, "os version should be identified")
	assert.Equal(suite.T(), "x86_64", di.Arch, "os arch should be identified")
	assert.Equal(suite.T(), []string{"linux", "unix", "os"}, di.Family)
}

func (suite *OsDetectTestSuite) TestPhoton3Detector() {
	detector, err := newDetector("./testdata/detect-photon3.toml")
	assert.Nil(suite.T(), err, "was able to create the transport")
	resolved, di := detector.Resolve()

	assert.Equal(suite.T(), true, resolved, "platform should be resolvable")
	assert.Equal(suite.T(), "photon", di.Name, "os name should be identified")
	assert.Equal(suite.T(), "VMware Photon OS", di.Title, "os title should be identified")
	assert.Equal(suite.T(), "3.0", di.Release, "os version should be identified")
	assert.Equal(suite.T(), "x86_64", di.Arch, "os arch should be identified")
	assert.Equal(suite.T(), []string{"linux", "unix", "os"}, di.Family)
}

func (suite *OsDetectTestSuite) TestMacOSsDetector() {
	detector, err := newDetector("./testdata/detect-macos.toml")
	assert.Nil(suite.T(), err, "was able to create the transport")
	resolved, di := detector.Resolve()

	assert.Equal(suite.T(), true, resolved, "platform should be resolvable")
	assert.Equal(suite.T(), "mac_os_x", di.Name, "os name should be identified")
	assert.Equal(suite.T(), "Mac OS X", di.Title, "os title should be identified")
	assert.Equal(suite.T(), "10.14.5", di.Release, "os version should be identified")
	assert.Equal(suite.T(), "x86_64", di.Arch, "os arch should be identified")
	assert.Equal(suite.T(), []string{"darwin", "bsd", "unix", "os"}, di.Family)
}

func (suite *OsDetectTestSuite) TestFamilies() {

	di := &platform.Info{}
	di.Family = []string{"unix", "bsd", "darwin"}

	assert.Equal(suite.T(), true, di.IsFamily("unix"), "unix should be a family")
	assert.Equal(suite.T(), true, di.IsFamily("bsd"), "bsd should be a family")
	assert.Equal(suite.T(), true, di.IsFamily("darwin"), "darwin should be a family")

}

func (suite *OsDetectTestSuite) TearDownSuite() {}

func TestOsDetectTestSuite(t *testing.T) {
	suite.Run(t, new(OsDetectTestSuite))
}

type TestContainer struct {
	Name  string
	Image string
}
