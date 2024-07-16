// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package packages

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/package-url/packageurl-go"
	"go.mondoo.com/cnquery/v11/providers/os/resources/cpe"
	"go.mondoo.com/cnquery/v11/providers/os/resources/purl"

	"github.com/cockroachdb/errors"
	_ "github.com/glebarez/go-sqlite" // required for sqlite3 rpm support
	rpmdb "github.com/knqyf263/go-rpmdb/pkg"
	"github.com/rs/zerolog/log"
	"github.com/spf13/afero"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v11/providers/os/connection/shared"
)

const (
	RpmPkgFormat = "rpm"
)

var RPM_REGEX = regexp.MustCompile(`^([\w-+]*)\s(\d*|\(none\)):([\w\d-+.:]+)\s([\w\d]*|\(none\))__([\w\d\s,\.]+)__(.*)$`)

// ParseRpmPackages parses output from:
// rpm -qa --queryformat '%{NAME} %{EPOCHNUM}:%{VERSION}-%{RELEASE} %{ARCH}__%{VENDOR}__%{SUMMARY}\n'
func ParseRpmPackages(pf *inventory.Platform, input io.Reader) []Package {
	pkgs := []Package{}
	scanner := bufio.NewScanner(input)
	for scanner.Scan() {
		line := scanner.Text()
		m := RPM_REGEX.FindStringSubmatch(line)
		if m != nil {
			name := m[1]
			epoch := m[2]
			version := m[3]

			// trim epoch if it is 0 or "(none)"
			if epoch == "0" || strings.TrimSpace(epoch) == "(none)" {
				epoch = ""
			} else {
				// only append the epoch if we have a non-zero value
				version = epoch + ":" + version
			}

			arch := m[4]
			// if no arch provided, remove it completely
			if arch == "(none)" {
				arch = ""
			}
			pkg := newRpmPackage(pf, name, version, arch, epoch, m[5], m[6])
			pkg.FilesAvailable = PkgFilesAsync // when we use commands we need to fetch the files async
			pkgs = append(pkgs, pkg)

		}
	}
	return pkgs
}

func newRpmPackage(pf *inventory.Platform, name, version, arch, epoch, vendor, description string) Package {
	// NOTE that we do not have the vendor of the package itself, we could try to parse it from the vendor
	// but that will also not be reliable. We may incorporate the cpe dictionary in the future but that would
	// increase the binary.
	if epoch == "0" {
		epoch = ""
	}
	cpes, _ := cpe.NewPackage2Cpe(vendor, name, version, epoch, arch)
	cpesWithoutEpoch := []string{}
	if epoch != "" {
		// I searched https://www.redhat.com/security/data/metrics/repository-to-cpe.json for the epoch
		// and it seems that the epoch is not part of the CPE, so we need to add it without the epoch
		cpesWithoutEpoch, _ = cpe.NewPackage2Cpe(vendor, name, version, "", arch)
	}
	cpesWithoutEpochAndArch, _ := cpe.NewPackage2Cpe(vendor, name, version, "", "")
	cpes = append(cpes, cpesWithoutEpoch...)
	cpes = append(cpes, cpesWithoutEpochAndArch...)
	return Package{
		Name:        name,
		Version:     version,
		Epoch:       epoch,
		Arch:        arch,
		Description: description,
		Format:      RpmPkgFormat,
		PUrl:        purl.NewPackageUrl(pf, name, version, arch, epoch, packageurl.TypeRPM),
		CPEs:        cpes,
		Vendor:      vendor,
	}
}

// RpmPkgManager is the package manager for Redhat, CentOS, Oracle, Photon and Suse
// it support two modes: runtime where the rpm command is available and static analysis for images (e.g. container tar)
// If the RpmPkgManager is used in static mode, it extracts the rpm database from the system and copies it to the local
// filesystem to run a local rpm command to extract the data. The static analysis is always slower than using the running
// one since more data need to copied. Therefore the runtime check should be preferred over the static analysis
type RpmPkgManager struct {
	conn          shared.Connection
	platform      *inventory.Platform
	staticChecked bool
	static        bool
}

func (rpm *RpmPkgManager) Name() string {
	return "Rpm Package Manager"
}

func (rpm *RpmPkgManager) Format() string {
	return RpmPkgFormat
}

// determine if we running against a static image, where we cannot execute the rpm command
// once executed, it caches its result to prevent the execution of the checks many times
func (rpm *RpmPkgManager) isStaticAnalysis() bool {
	if rpm.staticChecked {
		return rpm.static
	}

	rpm.static = false

	// check if the rpm command exists, e.g it is not available on tar backend
	c, err := rpm.conn.RunCommand("command -v rpm")
	if err != nil || c.ExitStatus != 0 {
		log.Debug().Msg("mql[packages]> fallback to static rpm package manager")
		rpm.static = true
	}

	// the root problem is that the docker transport (for running containers) cannot easily get the exit code so
	// we cannot always rely on that, a running photon container return non-zero exit code but it will be -1 on the system
	// we probably cannot fix this easily, see dockers approach:
	// https://docs.docker.com/engine/reference/commandline/attach/#get-the-exit-code-of-the-containers-command
	if c != nil {
		rpmCmdPath, err := io.ReadAll(c.Stdout)
		if err != nil || len(rpmCmdPath) == 0 {
			rpm.static = true
		}
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
	format := "%{NAME} %{EPOCH}:%{VERSION}-%{RELEASE} %{ARCH} %{VENDOR} %{SUMMARY}\\n"

	// ATTENTION: EPOCHNUM is only available since later version of rpm in RedHat 6 and Suse 12
	// we can only expect if for rhel 7+, therefore we need to run an extra test
	// be aware that this method is also used for non-redhat systems like suse
	i, err := strconv.ParseInt(rpm.platform.Version, 0, 32)
	if err == nil && (rpm.platform.Name == "centos" || rpm.platform.Name == "redhat") && i >= 7 {
		format = "%{NAME} %{EPOCHNUM}:%{VERSION}-%{RELEASE} %{ARCH} %{VENDOR} %{SUMMARY}\\n"
	}

	return format
}

func (rpm *RpmPkgManager) runtimeList() ([]Package, error) {
	command := fmt.Sprintf("rpm -qa --queryformat '%s'", rpm.queryFormat())
	cmd, err := rpm.conn.RunCommand(command)
	if err != nil {
		return nil, errors.Wrap(err, "could not read package list")
	}
	return ParseRpmPackages(rpm.platform, cmd.Stdout), nil
}

// fetch all available packages, is that working with centos 6?
func (rpm *RpmPkgManager) runtimeAvailable() (map[string]PackageUpdate, error) {
	// python script:
	// import sys;sys.path.insert(0, "/usr/share/yum-cli");import cli;list = cli.YumBaseCli().returnPkgLists(["updates"]);
	// print ''.join(["{\"name\":\""+x.name+"\", \"available\":\""+x.evr+"\",\"arch\":\""+x.arch+"\",\"repo\":\""+x.repo.id+"\"}\n" for x in list.updates]);
	script := "python -c 'import sys;sys.path.insert(0, \"/usr/share/yum-cli\");import cli;list = cli.YumBaseCli().returnPkgLists([\"updates\"]);print \"\".join([ \"{\\\"name\\\":\\\"\"+x.name+\"\\\",\\\"available\\\":\\\"\"+x.evr+\"\\\",\\\"arch\\\":\\\"\"+x.arch+\"\\\",\\\"repo\\\":\\\"\"+x.repo.id+\"\\\"}\\n\" for x in list.updates]);'"

	cmd, err := rpm.conn.RunCommand(script)
	if err != nil {
		log.Debug().Err(err).Msg("mql[packages]> could not read package updates")
		return nil, errors.Wrap(err, "could not read package update list")
	}
	return ParseRpmUpdates(cmd.Stdout)
}

func (rpm *RpmPkgManager) staticList() ([]Package, error) {
	rpmTmpDir, err := os.MkdirTemp(os.TempDir(), "mondoo-rpmdb")
	if err != nil {
		return nil, errors.Wrap(err, "could not create local temp directory")
	}
	log.Debug().Str("path", rpmTmpDir).Msg("mql[packages]> cache rpm library locally")
	defer os.RemoveAll(rpmTmpDir)

	fs := rpm.conn.FileSystem()
	afs := &afero.Afero{Fs: fs}

	// fetch rpm database file and store it in local tmp file
	// iterate over file paths to check if one exists
	files := []string{
		"/usr/lib/sysimage/rpm/Packages",     // used on opensuse container
		"/usr/lib/sysimage/rpm/Packages.db",  // used on SLES bci-base container
		"/usr/lib/sysimage/rpm/rpmdb.sqlite", // used on fedora 36+ and photon4
		"/var/lib/rpm/rpmdb.sqlite",          // used on fedora 33-35
		"/var/lib/rpm/Packages",              // used on fedora 32
	}
	var tmpRpmDBFile string
	var detectedPath string
	for i := range files {
		ok, err := afs.Exists(files[i])
		if err == nil && ok {
			splitPath := strings.Split(files[i], "/")
			tmpRpmDBFile = filepath.Join(rpmTmpDir, splitPath[len(splitPath)-1])
			detectedPath = files[i]
			break
		}
	}

	if len(detectedPath) == 0 {
		return nil, errors.Wrap(err, "could not find rpm packages location on : "+rpm.platform.Name)
	}
	log.Debug().Str("path", detectedPath).Msg("found rpm packages location")

	f, err := fs.Open(detectedPath)
	if err != nil {
		return nil, errors.Wrap(err, "could not fetch rpm package list")
	}
	defer f.Close()
	fWriter, err := os.Create(tmpRpmDBFile)
	if err != nil {
		log.Error().Err(err).Msg("mql[packages]> could not create tmp file for rpm database")
		return nil, errors.Wrap(err, "could not create local temp file")
	}
	defer fWriter.Close()
	_, err = io.Copy(fWriter, f)
	if err != nil {
		log.Error().Err(err).Msg("mql[packages]> could not copy rpm to tmp file")
		return nil, fmt.Errorf("could not cache rpm package list")
	}

	log.Debug().Str("rpmdb", rpmTmpDir).Msg("mql[packages]> cached rpm database locally")
	db, err := rpmdb.Open(tmpRpmDBFile)
	if err != nil {
		return nil, err
	}
	defer db.Close()
	pkgList, err := db.ListPackages()
	if err != nil {
		return nil, err
	}

	resultList := []Package{}
	for _, pkg := range pkgList {
		version := pkg.Version
		epoch := strconv.Itoa(pkg.EpochNum())
		version = epoch + ":" + version
		if pkg.Release != "" {
			version = version + "-" + pkg.Release
		}

		rpmPkg := newRpmPackage(rpm.platform, pkg.Name, version, pkg.Arch, epoch, pkg.Vendor, pkg.Summary)

		// determine all files attached
		records := []FileRecord{}
		files, err := pkg.InstalledFiles()
		if err == nil {
			for _, record := range files {
				records = append(records, FileRecord{
					Path: record.Path,
					Digest: PkgDigest{
						Value:     record.Digest,
						Algorithm: pkg.DigestAlgorithm.String(),
					},
					FileInfo: PkgFileInfo{
						Mode:  record.Mode,
						Flags: int32(record.Flags),
						Owner: record.Username,
						Group: record.Groupname,
						Size:  int64(record.Size),
					},
				})
			}
		}

		rpmPkg.FilesAvailable = PkgFilesIncluded
		rpmPkg.Files = records
		resultList = append(resultList, rpmPkg)
	}

	return resultList, nil
}

// TODO: Available() not implemented for RpmFileSystemManager
// for now this is not an error since we can easily determine available packages
func (rpm *RpmPkgManager) staticAvailable() (map[string]PackageUpdate, error) {
	return map[string]PackageUpdate{}, nil
}

func (rpm *RpmPkgManager) Files(name string, version string, arch string) ([]FileRecord, error) {
	if rpm.isStaticAnalysis() {
		// nothing to do since the data is already attached to the package
		return nil, nil
	} else {
		// we need to fetch the files from the running system
		cmd, err := rpm.conn.RunCommand("rpm -ql " + name)
		if err != nil {
			return nil, errors.Wrap(err, "could not read package files")
		}
		fileRecords := []FileRecord{}
		scanner := bufio.NewScanner(cmd.Stdout)
		for scanner.Scan() {
			line := scanner.Text()
			fileRecords = append(fileRecords, FileRecord{
				Path: line,
			})
		}
		return fileRecords, nil
	}
}

// SusePkgManager overwrites the normal RPM handler
type SusePkgManager struct {
	RpmPkgManager
}

func (spm *SusePkgManager) Available() (map[string]PackageUpdate, error) {
	if spm.isStaticAnalysis() {
		return spm.staticAvailable()
	}
	cmd, err := spm.conn.RunCommand("zypper -n --xmlout list-updates")
	if err != nil {
		log.Debug().Err(err).Msg("mql[packages]> could not read package updates")
		return nil, fmt.Errorf("could not read package update list")
	}
	return ParseZypperUpdates(cmd.Stdout)
}
