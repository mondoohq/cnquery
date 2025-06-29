// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package kernel

import (
	"bufio"
	"io"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/spf13/afero"
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

func ParseGenkex(stdout io.Reader) ([]*KernelModule, error) {
	res := []*KernelModule{}

	// genkex output is like this:
	// Text address     Size File

	// f10009d5b06d8000     d000 /usr/lib/drivers/bpf
	// f10009d5b06a2000    36000 /usr/lib/drivers/autofs.ext
	// f10009d5b0685000    1d000 /usr/lib/drivers/ahafs.ext

	genkexEntry := regexp.MustCompile(`^\s*(\S+)\s+(\S+)\s+(\S+)\s*$`)

	scanner := bufio.NewScanner(stdout)
	scanner.Scan()
	for scanner.Scan() {
		line := scanner.Text()
		m := genkexEntry.FindStringSubmatch(line)
		if len(m) == 4 {
			// Get the last part file name
			// /usr/lib/drivers/bpf -> bpf
			name := path.Base(m[3])
			res = append(res, &KernelModule{
				Name: name,
				Size: strings.TrimSpace(m[2]),
			})
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return res, nil
}

// ParseLinuxSysModule parses kernel modules from /sys/module directory structure
// This is used as a fallback when lsmod and /proc/modules are not available
func ParseLinuxSysModule(fs afero.Fs) ([]*KernelModule, error) {
	res := []*KernelModule{}

	// Check if /sys/module exists
	exists, err := afero.DirExists(fs, "/sys/module")
	if err != nil || !exists {
		return res, err
	}

	// Walk through /sys/module directory
	err = afero.Walk(fs, "/sys/module", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Continue walking even if there are errors
		}

		// We only care about directories that are direct children of /sys/module
		if !info.IsDir() {
			return nil
		}

		// Skip the root /sys/module directory itself
		if path == "/sys/module" {
			return nil
		}

		// Skip subdirectories (we only want direct module directories)
		relPath, err := filepath.Rel("/sys/module", path)
		if err != nil || strings.Contains(relPath, "/") {
			return nil
		}

		moduleName := info.Name()

		// Check if module is live by reading initstate file
		initstatePath := filepath.Join(path, "initstate")
		initstateFile, err := fs.Open(initstatePath)
		if err != nil {
			// If initstate doesn't exist, skip this module
			return nil
		}
		defer initstateFile.Close()

		initstateContent, err := io.ReadAll(initstateFile)
		if err != nil {
			return nil
		}

		initstate := strings.TrimSpace(string(initstateContent))
		if initstate != "live" {
			// Only include live modules
			return nil
		}

		// Try to get module size from coresize file
		size := "0"
		coresizePath := filepath.Join(path, "coresize")
		if coresizeFile, err := fs.Open(coresizePath); err == nil {
			defer coresizeFile.Close()
			if coresizeContent, err := io.ReadAll(coresizeFile); err == nil {
				if coresize, err := strconv.Atoi(strings.TrimSpace(string(coresizeContent))); err == nil {
					size = strconv.Itoa(coresize)
				}
			}
		}

		// Try to get reference count from refcnt file
		usedBy := "0"
		refcntPath := filepath.Join(path, "refcnt")
		if refcntFile, err := fs.Open(refcntPath); err == nil {
			defer refcntFile.Close()
			if refcntContent, err := io.ReadAll(refcntFile); err == nil {
				usedBy = strings.TrimSpace(string(refcntContent))
			}
		}

		res = append(res, &KernelModule{
			Name:   moduleName,
			Size:   size,
			UsedBy: usedBy,
		})

		return nil
	})

	return res, err
}
