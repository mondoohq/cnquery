package mount

import (
	"bufio"
	"io"
	"regexp"
	"strings"

	"errors"
	"github.com/spf13/afero"
)

func ParseLinuxMountCmd(r io.Reader) []MountPoint {
	res := []MountPoint{}

	mountEntry := regexp.MustCompile(`^(\S+)\son\s(\S+)\stype\s(\S+)\s\((\S+)\)$`)

	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Text()
		m := mountEntry.FindStringSubmatch(line)
		if len(m) == 5 {
			res = append(res, MountPoint{
				Device:     strings.TrimSpace(m[1]),
				MountPoint: strings.TrimSpace(m[2]),
				FSType:     strings.TrimSpace(m[3]),
				Options:    parseOptions(strings.TrimSpace(m[4])),
			})
		}
	}

	return res
}

// NOTE: we do not handle `map auto_home` on macos
func ParseUnixMountCmd(r io.Reader) []MountPoint {
	res := []MountPoint{}
	mountEntry := regexp.MustCompile(`^(\S+)\son\s(\S+)\s\((.*)\)$`)

	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Text()
		m := mountEntry.FindStringSubmatch(line)
		if len(m) == 4 {
			opts := strings.TrimSpace(m[3])
			fstype := ""
			entries := strings.Split(opts, ",")
			if len(entries) > 1 {
				fstype = strings.TrimSpace(entries[0])
			}

			res = append(res, MountPoint{
				Device:     strings.TrimSpace(m[1]),
				MountPoint: strings.TrimSpace(m[2]),
				FSType:     fstype,
				Options:    parseOptions(opts),
			})
		}
	}

	return res
}

// see https://stackoverflow.com/questions/18122123/how-to-interpret-proc-mounts
func ParseLinuxProcMount(r io.Reader) []MountPoint {
	res := []MountPoint{}

	procMountEntry := regexp.MustCompile(`^(\S+)\s(\S+)\s(\S+)\s(\S+)\s0\s0$`)

	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Text()
		m := procMountEntry.FindStringSubmatch(line)
		if len(m) == 5 {
			res = append(res, MountPoint{
				Device:     strings.TrimSpace(m[1]),
				MountPoint: strings.TrimSpace(m[2]),
				FSType:     strings.TrimSpace(m[3]),
				Options:    parseOptions(strings.TrimSpace(m[4])),
			})
		}
	}

	return res
}

func parseOptions(opts string) map[string]string {
	res := map[string]string{}
	entries := strings.Split(opts, ",")
	for i := range entries {
		entry := entries[i]
		keyval := strings.Split(entry, "=")
		if len(keyval) == 2 {
			res[strings.TrimSpace(keyval[0])] = strings.TrimSpace(keyval[1])
		} else {
			res[strings.TrimSpace(entry)] = ""
		}
	}
	return res
}

func ParseFstab(r io.Reader) ([]MountPoint, error) {
	res := []MountPoint{}

	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" || line[0] == '#' {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 4 {
			// ignoring bad lines
			continue
		}
		res = append(res, MountPoint{
			Device:     fields[0],
			MountPoint: fields[1],
			FSType:     fields[2],
			Options:    parseOptions(fields[3]),
		})
	}
	return res, scanner.Err()
}

func mountsFromFSLinux(fs afero.Fs) ([]MountPoint, error) {
	// Check if we have /proc/mounts
	procMountExists, err := afero.Exists(fs, "/proc/mounts")
	if err != nil {
		return nil, err
	}
	if procMountExists {
		f, err := fs.Open("/proc/mounts")
		if err != nil {
			return nil, err
		}
		defer f.Close()
		return ParseLinuxProcMount(f), nil
	}

	fstabExists, err := afero.Exists(fs, "/etc/fstab")
	if err != nil {
		return nil, err
	}
	if fstabExists {
		f, err := fs.Open("/etc/fstab")
		if err != nil {
			return nil, err
		}
		defer f.Close()
		return ParseFstab(f)
	}
	return nil, errors.New("could not find mounts")
}
