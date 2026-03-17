// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/inventory"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/os/connection/shared"
	"go.mondoo.com/mql/v13/providers/os/resources/powershell"
	"go.mondoo.com/mql/v13/providers/os/resources/procfs"
)

type cpuInfo struct {
	Cores          int64
	Manufacturer   string
	Model          string
	ProcessorCount int64
}

func initMachineCpu(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 2 {
		return args, nil, nil
	}

	conn := runtime.Connection.(shared.Connection)
	pf := conn.Asset().Platform

	var info *cpuInfo
	var err error

	switch {
	case pf.IsFamily("darwin"):
		info, err = getCpuInfoMacos(conn)
	case pf.IsFamily("windows"):
		info, err = getCpuInfoWindows(conn)
	case pf.IsFamily(inventory.FAMILY_UNIX):
		info, err = getCpuInfoLinux(conn)
	default:
		return nil, nil, fmt.Errorf("unsupported platform for cpu info: %s", pf.Name)
	}
	if err != nil {
		return nil, nil, err
	}

	return map[string]*llx.RawData{
		"cores":          llx.IntData(info.Cores),
		"manufacturer":   llx.StringData(info.Manufacturer),
		"model":          llx.StringData(info.Model),
		"processorCount": llx.IntData(info.ProcessorCount),
	}, nil, nil
}

func getCpuInfoLinux(conn shared.Connection) (*cpuInfo, error) {
	f, err := conn.FileSystem().Open("/proc/cpuinfo")
	if err != nil {
		return nil, err
	}
	defer f.Close()

	parsed, err := procfs.ParseCpuInfo(f)
	if err != nil {
		return nil, err
	}

	if len(parsed.Processors) == 0 {
		return nil, errors.New("no processors found in /proc/cpuinfo")
	}

	// Count physical CPU packages (sockets) by unique physical_id values.
	// Count physical cores by deduplicating (physical_id, core_id) pairs.
	// On ARM or in containers where these fields aren't populated,
	// fall back to 1 socket and using processor count as core count.
	type coreKey struct {
		physicalID uint
		coreID     uint
	}
	seenCores := map[coreKey]struct{}{}
	seenSockets := map[uint]struct{}{}
	hasCoreInfo := false
	for _, p := range parsed.Processors {
		if p.CPUCores > 0 || p.Siblings > 0 {
			hasCoreInfo = true
		}
		seenCores[coreKey{p.PhysicalID, p.CoreID}] = struct{}{}
		seenSockets[p.PhysicalID] = struct{}{}
	}

	manufacturer := parsed.Processors[0].VendorID
	if manufacturer == "GenuineIntel" {
		manufacturer = "Intel"
	}

	info := &cpuInfo{
		Manufacturer: manufacturer,
		Model:        parsed.Processors[0].ModelName,
	}

	if hasCoreInfo {
		info.Cores = int64(len(seenCores))
		info.ProcessorCount = int64(len(seenSockets))
	} else {
		// ARM or minimal /proc/cpuinfo: each processor entry is a core, assume 1 socket
		info.Cores = int64(len(parsed.Processors))
		info.ProcessorCount = 1
	}

	return info, nil
}

func getCpuInfoMacos(conn shared.Connection) (*cpuInfo, error) {
	cmd, err := conn.RunCommand("sysctl -n machdep.cpu.brand_string hw.physicalcpu")
	if err != nil {
		return nil, err
	}
	if cmd.ExitStatus != 0 {
		stderr, _ := io.ReadAll(cmd.Stderr)
		return nil, fmt.Errorf("sysctl failed: %s", string(stderr))
	}

	data, err := io.ReadAll(cmd.Stdout)
	if err != nil {
		return nil, err
	}

	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) < 2 {
		return nil, fmt.Errorf("unexpected sysctl output: %s", string(data))
	}

	brandString := strings.TrimSpace(lines[0])
	physicalCPU, err := strconv.ParseInt(strings.TrimSpace(lines[1]), 10, 64)
	if err != nil {
		return nil, fmt.Errorf("failed to parse hw.physicalcpu: %w", err)
	}

	// Strip "Apple " prefix from brand string (e.g. "Apple M4 Pro" -> "M4 Pro")
	model := strings.TrimPrefix(brandString, "Apple ")

	info := &cpuInfo{
		Model:          model,
		Cores:          physicalCPU,
		ProcessorCount: 1, // Macs always have a single CPU package
	}

	// Extract manufacturer from brand string
	if strings.Contains(brandString, "Intel") {
		info.Manufacturer = "Intel"
	} else if strings.Contains(brandString, "Apple") {
		info.Manufacturer = "Apple"
	} else if strings.Contains(brandString, "AMD") {
		info.Manufacturer = "AMD"
	}

	return info, nil
}

const cpuWindowsScript = `
$cpu = @(Get-CimInstance -ClassName Win32_Processor)
$result = @{
    Name = $cpu[0].Name
    Manufacturer = $cpu[0].Manufacturer
    NumberOfCores = ($cpu | Measure-Object -Property NumberOfCores -Sum).Sum
    ProcessorCount = $cpu.Count
}
$result | ConvertTo-Json
`

func getCpuInfoWindows(conn shared.Connection) (*cpuInfo, error) {
	cmd, err := conn.RunCommand(powershell.Encode(cpuWindowsScript))
	if err != nil {
		return nil, err
	}
	if cmd.ExitStatus != 0 {
		stderr, _ := io.ReadAll(cmd.Stderr)
		return nil, fmt.Errorf("failed to retrieve cpu info: %s", string(stderr))
	}

	var result struct {
		Name           string `json:"Name"`
		Manufacturer   string `json:"Manufacturer"`
		NumberOfCores  int64  `json:"NumberOfCores"`
		ProcessorCount int64  `json:"ProcessorCount"`
	}

	data, err := io.ReadAll(cmd.Stdout)
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("failed to parse cpu info: %w", err)
	}

	return &cpuInfo{
		Model:          strings.TrimSpace(result.Name),
		Manufacturer:   strings.TrimSpace(result.Manufacturer),
		Cores:          result.NumberOfCores,
		ProcessorCount: result.ProcessorCount,
	}, nil
}
