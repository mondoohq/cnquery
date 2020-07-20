package packages

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"go.mondoo.io/mondoo/motor"
	"go.mondoo.io/mondoo/motor/platform"
)

var (
	RPM_REGEX = regexp.MustCompile(`^([\w-+]*)\s(\d*|\(none\)):([\w\d-+.:]+)\s([\w\d]*|\(none\))\s(.*)$`)
)

// ParseRpmPackages parses output from:
// rpm -qa --queryformat '%{NAME} %{EPOCHNUM}:%{VERSION}-%{RELEASE} %{ARCH} %{SUMMARY}\n'
func ParseRpmPackages(input io.Reader) []Package {
	pkgs := []Package{}
	scanner := bufio.NewScanner(input)
	for scanner.Scan() {
		line := scanner.Text()
		m := RPM_REGEX.FindStringSubmatch(line)
		if m != nil {
			var version string
			// only append the epoch if we have a non-zero value
			if m[2] == "0" || strings.TrimSpace(m[2]) == "(none)" {
				version = m[3]
			} else {
				version = m[2] + ":" + m[3]
			}

			arch := m[4]
			// if no arch provided, remove it completely
			if arch == "(none)" {
				arch = ""
			}

			pkgs = append(pkgs, Package{
				Name:        m[1],
				Version:     version,
				Arch:        arch,
				Description: m[5],
				Format:      "rpm",
			})
		}
	}
	return pkgs
}

// RpmPkgManager is the pacakge manager for Redhat, CentOS, Oracle and Suse
// it support two modes: runtime where the rpm command is available and static analysis for images (e.g. container tar)
// If the RpmPkgManager is used in static mode, it extracts the rpm database from the system and copies it to the local
// filesystem to run a local rpm command to extract the data. The static analysis is always slower than using the running
// one since more data need to copied. Therefore the runtime check should be preferred over the static analysis
type RpmPkgManager struct {
	motor         *motor.Motor
	platform      *platform.PlatformInfo
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

func (rpm *RpmPkgManager) List() ([]Package, error) {
	if rpm.isStaticAnalysis() {
		return rpm.staticList()
	} else {
		return rpm.runtimeList()
	}
}

func (rpm *RpmPkgManager) Available() (map[string]PackageUpdate, error) {
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

func (rpm *RpmPkgManager) runtimeList() ([]Package, error) {

	command := fmt.Sprintf("rpm -qa --queryformat '%s'", rpm.queryFormat())
	cmd, err := rpm.motor.Transport.RunCommand(command)
	if err != nil {
		return nil, errors.Wrap(err, "could not read package list")
	}
	return ParseRpmPackages(cmd.Stdout), nil
}

// fetch all available packages, is that working with centos 6?
func (rpm *RpmPkgManager) runtimeAvailable() (map[string]PackageUpdate, error) {
	// python script:
	// import sys;sys.path.insert(0, "/usr/share/yum-cli");import cli;list = cli.YumBaseCli().returnPkgLists(["updates"]);
	// print ''.join(["{\"name\":\""+x.name+"\", \"available\":\""+x.evr+"\",\"arch\":\""+x.arch+"\",\"repo\":\""+x.repo.id+"\"}\n" for x in list.updates]);
	script := "python -c 'import sys;sys.path.insert(0, \"/usr/share/yum-cli\");import cli;list = cli.YumBaseCli().returnPkgLists([\"updates\"]);print \"\".join([ \"{\\\"name\\\":\\\"\"+x.name+\"\\\",\\\"available\\\":\\\"\"+x.evr+\"\\\",\\\"arch\\\":\\\"\"+x.arch+\"\\\",\\\"repo\\\":\\\"\"+x.repo.id+\"\\\"}\\n\" for x in list.updates]);'"

	cmd, err := rpm.motor.Transport.RunCommand(script)
	if err != nil {
		log.Debug().Err(err).Msg("lumi[packages]> could not read package updates")
		return nil, errors.Wrap(err, "could not read package update list")
	}
	return ParseRpmUpdates(cmd.Stdout)
}

func (rpm *RpmPkgManager) staticList() ([]Package, error) {
	rpmTmpDir, err := ioutil.TempDir(os.TempDir(), "mondoo-rpmdb")
	if err != nil {
		return nil, errors.Wrap(err, "could not create local temp directory")
	}
	defer os.RemoveAll(rpmTmpDir)

	// fetch rpm database file and store it in local tmp file
	f, err := rpm.motor.Transport.FS().Open("/var/lib/rpm/Packages")

	// on opensuse, the directory usr/lib/sysimage/rpm/Packages is used in tar
	if err != nil && rpm.platform != nil && rpm.platform.IsFamily("suse") {
		log.Debug().Msg("fallback to opensuse rpm package location")
		f, err = rpm.motor.Transport.FS().Open("/usr/lib/sysimage/rpm/Packages")
	}

	// throw error if we stil couldn't find the packages file
	if err != nil {
		return nil, errors.Wrap(err, "could not fetch rpm package list")
	}

	fWriter, err := os.Create(filepath.Join(rpmTmpDir, "Packages"))
	if err != nil {
		log.Error().Err(err).Msg("lumi[packages]> could not create tmp file for rpm database")
		return nil, errors.Wrap(err, "could not create local temp file")
	}
	_, err = io.Copy(fWriter, f)
	if err != nil {
		log.Error().Err(err).Msg("lumi[packages]> could not copy rpm to tmp file")
		return nil, fmt.Errorf("could not cache rpm package list")
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
		return nil, errors.Wrap(err, "could not read package list")
	}

	return ParseRpmPackages(&stdoutBuffer), nil
}

// TODO: Available() not implemented for RpmFileSystemManager
// for now this is not an error since we can easily determine available packages
func (rpm *RpmPkgManager) staticAvailable() (map[string]PackageUpdate, error) {
	return map[string]PackageUpdate{}, nil
}

// Suse, overwrites the Centos handler
type SusePkgManager struct {
	RpmPkgManager
}

func (spm *SusePkgManager) Available() (map[string]PackageUpdate, error) {
	cmd, err := spm.motor.Transport.RunCommand("zypper --xmlout list-updates")
	if err != nil {
		log.Debug().Err(err).Msg("lumi[packages]> could not read package updates")
		return nil, fmt.Errorf("could not read package update list")
	}
	return ParseZypperUpdates(cmd.Stdout)
}
