package packages

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"

	"github.com/rs/zerolog/log"
	"go.mondoo.io/mondoo/lumi/resources/parser"
	"go.mondoo.io/mondoo/motor"
)

type OperatingSystemPkgManager interface {
	Name() string
	Format() string
	List() ([]parser.Package, error)
	Available() ([]parser.PackageUpdate, error)
}

// this will find the right package manager for the operating system
func ResolveSystemPkgManager(motor *motor.Motor) (OperatingSystemPkgManager, error) {
	var pm OperatingSystemPkgManager

	platform, err := motor.Platform()
	if err != nil {
		return nil, err
	}

	// TODO: use OS family and select package manager
	switch platform.Name {
	case "manjaro", "arch": // arch family
		pm = &PacmanPkgManager{motor: motor}
	case "ubuntu", "debian": // debian family
		pm = &DebPkgManager{motor: motor}
	case "redhat", "centos", "amzn", "ol", "scientific": // rhel family
		pm = &RpmPkgManager{motor: motor}
	case "opensuse", "sles": // suse handling
		pm = &SusePkgManager{RpmPkgManager{motor: motor}}
	case "alpine": // alpine family
		pm = &AlpinePkgManager{motor: motor}
	case "mac_os_x": // mac os family
		pm = &MacOSPkgManager{motor: motor}
	default:
		return nil, errors.New("your platform is not supported by packages resource")
	}

	return pm, nil
}

// Debian, Ubuntu
type DebPkgManager struct {
	motor *motor.Motor
}

func (dpm *DebPkgManager) Name() string {
	return "Debian Package Manager"
}

func (dpm *DebPkgManager) Format() string {
	return "deb"
}

func (dpm *DebPkgManager) List() ([]parser.Package, error) {
	fi, err := dpm.motor.Transport.File("/var/lib/dpkg/status")
	if err != nil {
		return nil, fmt.Errorf("could not read package list")
	}

	fReader, err := fi.Open()
	defer fReader.Close()

	return parser.ParseDpkgPackages(fReader)
}

func (dpm *DebPkgManager) Available() ([]parser.PackageUpdate, error) {
	// TODO: run this as a complete shell script in motor
	// DEBIAN_FRONTEND=noninteractive apt-get update >/dev/null 2>&1
	// readlock() { cat /proc/locks | awk '{print $5}' | grep -v ^0 | xargs -I {1} find /proc/{1}/fd -maxdepth 1 -exec readlink {} \; | grep '^/var/lib/dpkg/lock$'; }
	// while test -n "$(readlock)"; do sleep 1; done
	// DEBIAN_FRONTEND=noninteractive apt-get upgrade --dry-run
	dpm.motor.Transport.RunCommand("DEBIAN_FRONTEND=noninteractive apt-get update >/dev/null 2>&1")

	cmd, err := dpm.motor.Transport.RunCommand("DEBIAN_FRONTEND=noninteractive apt-get upgrade --dry-run")
	if err != nil {
		log.Error().Err(err).Msg("lumi[packages]> could not read package updates")
		return nil, fmt.Errorf("could not read package update list")
	}
	return parser.ParseDpkgUpdates(cmd.Stdout)
}

// RpmPkgManager is the pacakge manager for Redhat, CentOS, Oracle and Suse
// it support two modes: runtime where the rpm command is available and static analysis for images (e.g. container tar)
// If the RpmPkgManager is used in static mode, it extracts the rpm database from the system and copies it to the local
// filesystem to run a local rpm command to extract the data. The static analysis is always slower than using the running
// one since more data need to copied. Therefore the runtime check should be preferred over the static analysis
type RpmPkgManager struct {
	motor         *motor.Motor
	staticChecked bool
	static        bool
}

func (rpm *RpmPkgManager) Name() string {
	return "Rpm Package Manager"
}

func (rpm *RpmPkgManager) Format() string {
	return "rpm"
}

// determine if we running against a static image, where we cannot execute the rpm command
// once executed, it caches its result to prevent the execution of the checks many times
func (rpm *RpmPkgManager) isStaticAnalysis() bool {
	if rpm.staticChecked == true {
		return rpm.static
	}

	rpm.static = false

	// check if the rpm command exists, e.g it is not available on tar backend
	c, err := rpm.motor.Transport.RunCommand("command -v rpm")
	if err != nil || c.ExitStatus != 0 {
		log.Debug().Msg("lumi[packages]> fallback to static rpm package manager")
		rpm.static = true
	}

	rpm.staticChecked = true
	return rpm.static
}

func (rpm *RpmPkgManager) List() ([]parser.Package, error) {
	if rpm.isStaticAnalysis() {
		return rpm.staticList()
	} else {
		return rpm.runtimeList()
	}
}

func (rpm *RpmPkgManager) Available() ([]parser.PackageUpdate, error) {
	if rpm.isStaticAnalysis() {
		return rpm.staticAvailable()
	} else {
		return rpm.runtimeAvailable()
	}
}

func (rpm *RpmPkgManager) queryFormat() string {
	// this format should work everywhere
	// fall-back to epoch instead of epochnum for 6 ish platforms, latest 6 platforms also support epochnum, but we
	// save 1 call by not detecting the available keyword via rpm --querytags
	format := "%{NAME} %{EPOCH}:%{VERSION}-%{RELEASE} %{ARCH} %{SUMMARY}\\n"

	// ATTENTION: EPOCHNUM is only available since later version of rpm in RedHat 6 and Suse 12
	// we can only expect if for rhel 7+, therefore we need to run an extra test
	info, err := rpm.motor.Platform()
	if err != nil {
		return format
	}

	// be aware that this method is also used for non-redhat systems like suse
	i, err := strconv.ParseInt(info.Release, 0, 32)
	if err == nil && (info.Name == "centos" || info.Name == "redhat") && i >= 7 {
		format = "%{NAME} %{EPOCHNUM}:%{VERSION}-%{RELEASE} %{ARCH} %{SUMMARY}\\n"
	}

	return format
}

func (rpm *RpmPkgManager) runtimeList() ([]parser.Package, error) {

	command := fmt.Sprintf("rpm -qa --queryformat '%s'", rpm.queryFormat())
	cmd, err := rpm.motor.Transport.RunCommand(command)
	if err != nil {
		return nil, fmt.Errorf("could not read package list")
	}
	return parser.ParseRpmPackages(cmd.Stdout), nil
}

// fetch all available packages, is that working with centos 6?
func (rpm *RpmPkgManager) runtimeAvailable() ([]parser.PackageUpdate, error) {
	// python script:
	// import sys;sys.path.insert(0, "/usr/share/yum-cli");import cli;list = cli.YumBaseCli().returnPkgLists(["updates"]);
	// print ''.join(["{\"name\":\""+x.name+"\", \"available\":\""+x.evr+"\",\"arch\":\""+x.arch+"\",\"repo\":\""+x.repo.id+"\"}\n" for x in list.updates]);
	script := "python -c 'import sys;sys.path.insert(0, \"/usr/share/yum-cli\");import cli;list = cli.YumBaseCli().returnPkgLists([\"updates\"]);print \"\".join([ \"{\\\"name\\\":\\\"\"+x.name+\"\\\",\\\"available\\\":\\\"\"+x.evr+\"\\\",\\\"arch\\\":\\\"\"+x.arch+\"\\\",\\\"repo\\\":\\\"\"+x.repo.id+\"\\\"}\\n\" for x in list.updates]);'"

	cmd, err := rpm.motor.Transport.RunCommand(script)
	if err != nil {
		log.Error().Err(err).Msg("lumi[packages]> could not read package updates")
		return nil, fmt.Errorf("could not read package update list")
	}
	return parser.ParseRpmUpdates(cmd.Stdout)
}

func (rpm *RpmPkgManager) staticList() ([]parser.Package, error) {

	rpmTmpDir, err := ioutil.TempDir(os.TempDir(), "mondoo-rpmdb")
	if err != nil {
		return nil, fmt.Errorf("could not read package list")
	}
	defer os.RemoveAll(rpmTmpDir)

	// fetch rpm database files and store it in local tmp directory
	f, err := rpm.motor.Transport.File("/var/lib/rpm")
	if err != nil {
		return nil, fmt.Errorf("could not read package list")
	}

	fReader, err := f.Open()
	if err != nil {
		return nil, fmt.Errorf("could not read package list")
	}

	fWriter, err := os.Create(filepath.Join(rpmTmpDir, "Packages"))
	if err != nil {
		log.Error().Err(err).Msg("lumi[packages]> could not create tmp file for rpm database")
		return nil, fmt.Errorf("could not read package list")
	}
	_, err = io.Copy(fWriter, fReader)
	if err != nil {
		log.Error().Err(err).Msg("lumi[packages]> could not copy rpm to tmp file")
		return nil, fmt.Errorf("could not read package list")
	}

	log.Debug().Str("rpmdb", rpmTmpDir).Msg("cached rpm database locally")

	// call local rpm tool to extract the packages
	c := exec.Command("rpm", "--dbpath", rpmTmpDir, "-qa", "--queryformat", rpm.queryFormat())

	stdoutBuffer := bytes.Buffer{}
	stderrBuffer := bytes.Buffer{}

	c.Stdout = &stdoutBuffer
	c.Stderr = &stderrBuffer

	err = c.Run()
	if err != nil {
		log.Error().Err(err).Msg("lumi[packages]> could not execute rpm locally")
		return nil, fmt.Errorf("could not read package list")
	}

	return parser.ParseRpmPackages(&stdoutBuffer), nil
}

// TODO: Available() not implemented for RpmFileSystemManager
// for now this is not an error since we can easily determine available packages
func (rpm *RpmPkgManager) staticAvailable() ([]parser.PackageUpdate, error) {
	return []parser.PackageUpdate{}, nil
}

// Suse, overwrites the Centos handler
type SusePkgManager struct {
	RpmPkgManager
}

func (spm *SusePkgManager) Available() ([]parser.PackageUpdate, error) {
	cmd, err := spm.motor.Transport.RunCommand("zypper --xmlout list-updates")
	if err != nil {
		log.Error().Err(err).Msg("lumi[packages]> could not read package updates")
		return nil, fmt.Errorf("could not read package update list")
	}
	return parser.ParseZypperUpdates(cmd.Stdout)
}

// Arch, Manjaro
type PacmanPkgManager struct {
	motor *motor.Motor
}

func (ppm *PacmanPkgManager) Name() string {
	return "Pacman Package Manager"
}

func (ppm *PacmanPkgManager) Format() string {
	return "pacman"
}

func (ppm *PacmanPkgManager) List() ([]parser.Package, error) {
	cmd, err := ppm.motor.Transport.RunCommand("pacman -Q")
	if err != nil {
		return nil, fmt.Errorf("could not read package list")
	}

	return parser.ParsePacmanPackages(cmd.Stdout), nil
}

func (ppm *PacmanPkgManager) Available() ([]parser.PackageUpdate, error) {
	return nil, errors.New("Available() not implemented for PacmanPkgManager")
}

// Arch, Manjaro
type AlpinePkgManager struct {
	motor *motor.Motor
}

func (apm *AlpinePkgManager) Name() string {
	return "Alpine Package Manager"
}

func (apm *AlpinePkgManager) Format() string {
	return "apk"
}

func (apm *AlpinePkgManager) List() ([]parser.Package, error) {
	fi, err := apm.motor.Transport.File("/lib/apk/db/installed")
	if err != nil {
		return nil, fmt.Errorf("could not read package list")
	}

	fReader, err := fi.Open()
	defer fReader.Close()

	return parser.ParseApkDbPackages(fReader), nil
}

func (apm *AlpinePkgManager) Available() ([]parser.PackageUpdate, error) {
	// it only works if apk is updated
	apm.motor.Transport.RunCommand("apk update")

	// determine package updates
	cmd, err := apm.motor.Transport.RunCommand("apk version -v -l '<'")
	if err != nil {
		log.Error().Err(err).Msg("lumi[packages]> could not read package updates")
		return nil, fmt.Errorf("could not read package update list")
	}
	return parser.ParseApkUpdates(cmd.Stdout)
}

// MacOS
type MacOSPkgManager struct {
	motor *motor.Motor
}

func (mpm *MacOSPkgManager) Name() string {
	return "macOS Package Manager"
}

func (mpm *MacOSPkgManager) Format() string {
	return "macos"
}

func (mpm *MacOSPkgManager) List() ([]parser.Package, error) {
	cmd, err := mpm.motor.Transport.RunCommand("system_profiler SPApplicationsDataType -xml")
	if err != nil {
		return nil, fmt.Errorf("could not read package list")
	}

	return parser.ParseMacOSPackages(cmd.Stdout)
}

func (mpm *MacOSPkgManager) Available() ([]parser.PackageUpdate, error) {
	return nil, errors.New("Available() not implemented for MacOSPkgManager")
}
