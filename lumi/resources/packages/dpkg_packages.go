package packages

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/spf13/afero"
	os_provider "go.mondoo.io/mondoo/motor/providers/os"
)

const (
	DpkgPkgFormat = "deb"
)

var (
	DPKG_REGEX        = regexp.MustCompile(`^(.+):\s(.+)$`)
	DPKG_ORIGIN_REGEX = regexp.MustCompile(`^\s*([^\(]*)(?:\((.*)\))?\s*$`)
)

// parse the dpkg database content located in /var/lib/dpkg/status
func ParseDpkgPackages(input io.Reader) ([]Package, error) {
	const STATE_RESET = 0
	const STATE_DESC = 1
	pkgs := []Package{}

	add := func(pkg Package) {
		// do sanitization checks to ensure we have minimal information
		if pkg.Name != "" && pkg.Version != "" {
			pkgs = append(pkgs, pkg)
		} else {
			log.Debug().Msg("ignored deb packages since information is missing")
		}
	}

	scanner := bufio.NewScanner(input)
	pkg := Package{Format: DpkgPkgFormat}
	state := STATE_RESET
	var key string
	for scanner.Scan() {
		line := scanner.Text()

		// reset package definition once we reach a newline
		if len(line) == 0 {
			add(pkg)
			pkg = Package{Format: DpkgPkgFormat}
		}

		m := DPKG_REGEX.FindStringSubmatch(line)
		key = ""
		if m != nil {
			key = m[1]
			state = STATE_RESET
		}
		switch {
		case key == "Package":
			pkg.Name = strings.TrimSpace(m[2])
		case key == "Version":
			pkg.Version = strings.TrimSpace(m[2])
		case key == "Architecture":
			pkg.Arch = strings.TrimSpace(m[2])
		case key == "Status":
			pkg.Status = strings.TrimSpace(m[2])
		case key == "Source":
			o := DPKG_ORIGIN_REGEX.FindStringSubmatch(m[2])
			if o != nil && len(o) >= 1 {
				pkg.Origin = strings.TrimSpace(o[1])
			} else {
				log.Error().Str("origin", m[2]).Msg("cannot parse dpkg origin")
			}
		// description supports multi-line statements, start desc
		case key == "Description":
			pkg.Description = strings.TrimSpace(m[2])
			state = STATE_DESC
		// next desc line, append to previous one
		case state == STATE_DESC:
			pkg.Description += "\n" + strings.TrimSpace(line)
		}
	}

	// if the last line is not an empty line we have things in flight, lets check it
	add(pkg)

	return pkgs, nil
}

var DPKG_UPDATE_REGEX = regexp.MustCompile(`^Inst\s([a-zA-Z0-9.\-_]+)\s\[([a-zA-Z0-9.\-\+]+)\]\s\(([a-zA-Z0-9.\-\+]+)\s*(.*)\)(.*)$`)

func ParseDpkgUpdates(input io.Reader) (map[string]PackageUpdate, error) {
	pkgs := map[string]PackageUpdate{}
	scanner := bufio.NewScanner(input)
	for scanner.Scan() {
		line := scanner.Text()
		m := DPKG_UPDATE_REGEX.FindStringSubmatch(line)
		if m != nil {
			pkgs[m[1]] = PackageUpdate{
				Name:      m[1],
				Version:   m[2],
				Available: m[3],
			}
		}
	}
	return pkgs, nil
}

// Debian, Ubuntu
type DebPkgManager struct {
	provider os_provider.OperatingSystemProvider
}

func (dpm *DebPkgManager) Name() string {
	return "Debian Package Manager"
}

func (dpm *DebPkgManager) Format() string {
	return DpkgPkgFormat
}

func (dpm *DebPkgManager) List() ([]Package, error) {
	fs := dpm.provider.FS()
	dpkgStatusFile := "/var/lib/dpkg/status"
	dpkgStatusDir := "/var/lib/dpkg/status.d"
	_, fErr := fs.Stat(dpkgStatusFile)
	dStat, dErr := fs.Stat(dpkgStatusDir)

	if fErr != nil && dErr != nil {
		log.Debug().Err(fErr).Str("path", dpkgStatusFile).Msg("cannot find status file")
		log.Debug().Err(dErr).Str("path", dpkgStatusDir).Msg("cannot find status dir")
		return nil, fmt.Errorf("could not find dpkg package list")
	}

	pkgList := []Package{}
	// main pkg file for debian systems
	if fErr == nil {
		log.Debug().Str("file", dpkgStatusFile).Msg("parse dpkg status file")
		fi, err := dpm.provider.FS().Open(dpkgStatusFile)
		if err != nil {
			return nil, fmt.Errorf("could not read dpkg package list")
		}
		defer fi.Close()

		list, err := ParseDpkgPackages(fi)
		if err != nil {
			return nil, fmt.Errorf("could not parse dpkg package list")
		}
		pkgList = append(pkgList, list...)
	}

	// e.g. google distroless images stores their pkg data in /var/lib/dpkg/status.d/
	if dErr == nil && dStat.IsDir() == true {
		afutil := afero.Afero{Fs: fs}
		wErr := afutil.Walk(dpkgStatusDir, func(path string, f os.FileInfo, fErr error) error {
			if f == nil || f.IsDir() {
				return nil
			}

			log.Debug().Str("path", path).Msg("walk file")
			fi, err := dpm.provider.FS().Open(path)
			if err != nil {
				log.Debug().Err(err).Str("path", path).Msg("could open file")
				return fmt.Errorf("could not read dpkg package list")
			}

			list, err := ParseDpkgPackages(fi)
			fi.Close()
			if err != nil {
				log.Debug().Err(err).Str("path", path).Msg("could not parse")
				return fmt.Errorf("could not parse dpkg package list")
			}

			log.Debug().Int("pkgs", len(list)).Msg("completed parsing")
			pkgList = append(pkgList, list...)
			return nil
		})
		if wErr != nil {
			return nil, wErr
		}
	}

	return pkgList, nil
}

func (dpm *DebPkgManager) Available() (map[string]PackageUpdate, error) {
	// TODO: run this as a complete shell script in motor
	// DEBIAN_FRONTEND=noninteractive apt-get update >/dev/null 2>&1
	// readlock() { cat /proc/locks | awk '{print $5}' | grep -v ^0 | xargs -I {1} find /proc/{1}/fd -maxdepth 1 -exec readlink {} \; | grep '^/var/lib/dpkg/lock$'; }
	// while test -n "$(readlock)"; do sleep 1; done
	// DEBIAN_FRONTEND=noninteractive apt-get upgrade --dry-run
	dpm.provider.RunCommand("DEBIAN_FRONTEND=noninteractive apt-get update >/dev/null 2>&1")

	cmd, err := dpm.provider.RunCommand("DEBIAN_FRONTEND=noninteractive apt-get upgrade --dry-run")
	if err != nil {
		log.Debug().Err(err).Msg("lumi[packages]> could not read package updates")
		return nil, fmt.Errorf("could not read package update list")
	}
	return ParseDpkgUpdates(cmd.Stdout)
}
