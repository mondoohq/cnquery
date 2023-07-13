package kernel

import (
	"bufio"
	"io"
	"regexp"
	"strings"
)

func ParseLsmod(r io.Reader) []*KernelModule {
	res := []*KernelModule{}

	lsmodEntry := regexp.MustCompile(`^(\S+)\s+(\S+)\s+(\S+)\s*(\S*)$`)

	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Text()
		m := lsmodEntry.FindStringSubmatch(line)
		if len(m) == 5 {
			if m[1] == "Module" {
				continue
			}

			res = append(res, &KernelModule{
				Name:   strings.TrimSpace(m[1]),
				Size:   strings.TrimSpace(m[2]),
				UsedBy: strings.TrimSpace(m[3]),
			})
		}
	}

	return res
}

func ParseLinuxProcModules(r io.Reader) []*KernelModule {
	res := []*KernelModule{}

	procModulesEntry := regexp.MustCompile(`^(\S+)\s(\S+)\s(\S+)\s(.*)$`)

	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Text()
		m := procModulesEntry.FindStringSubmatch(line)
		if len(m) == 5 {
			res = append(res, &KernelModule{
				Name:   strings.TrimSpace(m[1]),
				Size:   strings.TrimSpace(m[2]),
				UsedBy: strings.TrimSpace(m[3]),
			})
		}
	}

	return res
}

func ParseKldstat(r io.Reader) []*KernelModule {
	res := []*KernelModule{}

	lsmodEntry := regexp.MustCompile(`^\s+(\S+)\s+(\S+)\s+(\S+)\s*(\S*)\s*(\S*)$`)

	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Text()
		m := lsmodEntry.FindStringSubmatch(line)
		if len(m) == 6 {
			res = append(res, &KernelModule{
				Name:   strings.TrimSpace(m[5]),
				Size:   strings.TrimSpace(m[4]),
				UsedBy: strings.TrimSpace(m[2]),
			})
		}
	}

	return res
}

func ParseKextstat(r io.Reader) []*KernelModule {
	res := []*KernelModule{}

	lsmodEntry := regexp.MustCompile(`^\s+(\S+)\s+(\S+)\s+(\S+)\s*(\S*)\s*(\S*)\s*(\S*)`)

	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Text()
		m := lsmodEntry.FindStringSubmatch(line)
		if len(m) == 7 {
			res = append(res, &KernelModule{
				Name:   strings.TrimSpace(m[6]),
				Size:   strings.TrimSpace(m[4]),
				UsedBy: strings.TrimSpace(m[2]),
			})
		}
	}

	return res
}
