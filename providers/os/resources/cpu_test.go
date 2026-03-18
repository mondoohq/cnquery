// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/mql/v13/providers-sdk/v1/inventory"
	"go.mondoo.com/mql/v13/providers/os/connection/mock"
)

func TestGetCpuInfoLinuxX64(t *testing.T) {
	conn, err := mock.New(0, &inventory.Asset{}, mock.WithPath("./procfs/testdata/cpu-info-x64.toml"))
	require.NoError(t, err)

	info, err := getCpuInfoLinux(conn)
	require.NoError(t, err)

	assert.Equal(t, "Intel", info.Manufacturer)
	assert.Equal(t, "Intel(R) Core(TM) i7-6700K CPU @ 4.00GHz", info.Model)
	assert.Equal(t, int64(2), info.ProcessorCount)
	assert.Equal(t, int64(2), info.Cores)
}

func TestGetCpuInfoMacosAppleSilicon(t *testing.T) {
	conn, err := mock.New(0, &inventory.Asset{}, mock.WithData(&mock.TomlData{
		Commands: map[string]*mock.Command{
			"sysctl -n machdep.cpu.brand_string hw.physicalcpu": {
				Stdout: "Apple M4 Pro\n14\n",
			},
		},
	}))
	require.NoError(t, err)

	info, err := getCpuInfoMacos(conn)
	require.NoError(t, err)

	assert.Equal(t, "Apple", info.Manufacturer)
	assert.Equal(t, "Apple M4 Pro", info.Model)
	assert.Equal(t, int64(1), info.ProcessorCount)
	assert.Equal(t, int64(14), info.Cores)
}

func TestGetCpuInfoMacosIntel(t *testing.T) {
	conn, err := mock.New(0, &inventory.Asset{}, mock.WithData(&mock.TomlData{
		Commands: map[string]*mock.Command{
			"sysctl -n machdep.cpu.brand_string hw.physicalcpu": {
				Stdout: "Intel(R) Core(TM) i9-9980HK CPU @ 2.40GHz\n8\n",
			},
		},
	}))
	require.NoError(t, err)

	info, err := getCpuInfoMacos(conn)
	require.NoError(t, err)

	assert.Equal(t, "Intel", info.Manufacturer)
	assert.Equal(t, "Intel(R) Core(TM) i9-9980HK CPU @ 2.40GHz", info.Model)
	assert.Equal(t, int64(1), info.ProcessorCount)
	assert.Equal(t, int64(8), info.Cores)
}

func TestGetCpuInfoLinuxAMD(t *testing.T) {
	conn, err := mock.New(0, &inventory.Asset{}, mock.WithPath("./procfs/testdata/cpu-info-amd.toml"))
	require.NoError(t, err)

	info, err := getCpuInfoLinux(conn)
	require.NoError(t, err)

	assert.Equal(t, "AMD", info.Manufacturer)
	assert.Equal(t, "AMD EPYC 7763 64-Core Processor", info.Model)
	assert.Equal(t, int64(1), info.ProcessorCount)
	assert.Equal(t, int64(2), info.Cores)
}

func TestGetCpuInfoFreeBSD(t *testing.T) {
	conn, err := mock.New(0, &inventory.Asset{}, mock.WithData(&mock.TomlData{
		Commands: map[string]*mock.Command{
			"sysctl -n hw.model kern.smp.cores": {
				Stdout: "Intel(R) Xeon(R) CPU E3-1220 v5 @ 3.00GHz\n4\n",
			},
		},
	}))
	require.NoError(t, err)

	info, err := getCpuInfoFreeBSD(conn)
	require.NoError(t, err)

	assert.Equal(t, "Intel", info.Manufacturer)
	assert.Equal(t, "Intel(R) Xeon(R) CPU E3-1220 v5 @ 3.00GHz", info.Model)
	assert.Equal(t, int64(1), info.ProcessorCount)
	assert.Equal(t, int64(4), info.Cores)
}

func TestGetCpuInfoAIX(t *testing.T) {
	conn, err := mock.New(0, &inventory.Asset{}, mock.WithData(&mock.TomlData{
		Commands: map[string]*mock.Command{
			"prtconf": {
				Stdout: "System Model: IBM,9009-42A\n" +
					"Processor Type: PowerPC_POWER9\n" +
					"Number Of Processors: 48\n" +
					"Processor Clock Speed: 3800 MHz\n" +
					"CPU Type: 64-bit\n",
			},
			"lsdev -Cc processor": {
				Stdout: "proc0  Available  00-00  Processor\n" +
					"proc4  Available  00-04  Processor\n" +
					"proc8  Available  00-08  Processor\n" +
					"proc12 Available  00-12  Processor\n" +
					"proc16 Available  00-16  Processor\n" +
					"proc20 Available  00-20  Processor\n" +
					"proc24 Available  00-24  Processor\n" +
					"proc28 Available  00-28  Processor\n" +
					"proc32 Available  00-32  Processor\n" +
					"proc36 Available  00-36  Processor\n" +
					"proc40 Available  00-40  Processor\n" +
					"proc44 Available  00-44  Processor\n",
			},
		},
	}))
	require.NoError(t, err)

	info, err := getCpuInfoAIX(conn)
	require.NoError(t, err)

	assert.Equal(t, "IBM", info.Manufacturer)
	assert.Equal(t, "PowerPC_POWER9", info.Model)
	assert.Equal(t, int64(1), info.ProcessorCount)
	// 12 physical cores from lsdev, not 48 logical from prtconf
	assert.Equal(t, int64(12), info.Cores)
}

func TestGetCpuInfoSolaris(t *testing.T) {
	conn, err := mock.New(0, &inventory.Asset{}, mock.WithData(&mock.TomlData{
		Commands: map[string]*mock.Command{
			"psrinfo -pv": {
				Stdout: "The physical processor has 4 cores and 8 virtual processors (0-7)\n" +
					"  x86 (GenuineIntel 206D7 family 6 model 45 step 7 clock 2600 MHz)\n" +
					"        Intel(r) Xeon(r) CPU E5-2670 0 @ 2.60GHz\n" +
					"The physical processor has 4 cores and 8 virtual processors (8-15)\n" +
					"  x86 (GenuineIntel 206D7 family 6 model 45 step 7 clock 2600 MHz)\n" +
					"        Intel(r) Xeon(r) CPU E5-2670 0 @ 2.60GHz\n",
			},
		},
	}))
	require.NoError(t, err)

	info, err := getCpuInfoSolaris(conn)
	require.NoError(t, err)

	assert.Equal(t, "Intel", info.Manufacturer)
	assert.Equal(t, "Intel(r) Xeon(r) CPU E5-2670 0 @ 2.60GHz", info.Model)
	assert.Equal(t, int64(2), info.ProcessorCount)
	assert.Equal(t, int64(8), info.Cores)
}

func TestGetCpuInfoLinuxArmWithLscpu(t *testing.T) {
	conn, err := mock.New(0, &inventory.Asset{}, mock.WithData(&mock.TomlData{
		Files: map[string]*mock.MockFileData{
			"/proc/cpuinfo": {
				Content: "processor\t: 0\n" +
					"BogoMIPS\t: 108.00\n" +
					"Features\t: fp asimd evtstrm\n" +
					"CPU implementer\t: 0x41\n" +
					"CPU architecture: 8\n" +
					"CPU variant\t: 0x0\n" +
					"CPU part\t: 0xd08\n" +
					"CPU revision\t: 3\n" +
					"\n" +
					"processor\t: 1\n" +
					"BogoMIPS\t: 108.00\n" +
					"Features\t: fp asimd evtstrm\n" +
					"CPU implementer\t: 0x41\n" +
					"CPU architecture: 8\n" +
					"CPU variant\t: 0x0\n" +
					"CPU part\t: 0xd08\n" +
					"CPU revision\t: 3\n" +
					"\n" +
					"processor\t: 2\n" +
					"BogoMIPS\t: 108.00\n" +
					"Features\t: fp asimd evtstrm\n" +
					"CPU implementer\t: 0x41\n" +
					"CPU architecture: 8\n" +
					"CPU variant\t: 0x0\n" +
					"CPU part\t: 0xd08\n" +
					"CPU revision\t: 3\n" +
					"\n" +
					"processor\t: 3\n" +
					"BogoMIPS\t: 108.00\n" +
					"Features\t: fp asimd evtstrm\n" +
					"CPU implementer\t: 0x41\n" +
					"CPU architecture: 8\n" +
					"CPU variant\t: 0x0\n" +
					"CPU part\t: 0xd08\n" +
					"CPU revision\t: 3\n",
			},
		},
		Commands: map[string]*mock.Command{
			"lscpu": {
				Stdout: "Architecture:                            aarch64\n" +
					"Byte Order:                              Little Endian\n" +
					"CPU(s):                                  4\n" +
					"On-line CPU(s) list:                     0-3\n" +
					"Thread(s) per core:                      1\n" +
					"Core(s) per socket:                      4\n" +
					"Socket(s):                               1\n" +
					"Vendor ID:                               ARM\n" +
					"Model:                                   3\n" +
					"Model name:                              Cortex-A72\n" +
					"Stepping:                                r0p3\n" +
					"CPU max MHz:                             1500.0000\n" +
					"CPU min MHz:                             600.0000\n" +
					"BogoMIPS:                                108.00\n",
			},
		},
	}))
	require.NoError(t, err)

	info, err := getCpuInfoLinux(conn)
	require.NoError(t, err)

	assert.Equal(t, "ARM", info.Manufacturer)
	assert.Equal(t, "Cortex-A72", info.Model)
	assert.Equal(t, int64(1), info.ProcessorCount)
	assert.Equal(t, int64(4), info.Cores)
}

func TestGetCpuInfoLinuxArmNoLscpu(t *testing.T) {
	// ARM system where lscpu is not available — manufacturer/model stay empty
	conn, err := mock.New(0, &inventory.Asset{}, mock.WithPath("./procfs/testdata/cpu-info-aarch64.toml"))
	require.NoError(t, err)

	info, err := getCpuInfoLinux(conn)
	require.NoError(t, err)

	assert.Equal(t, "", info.Manufacturer)
	assert.Equal(t, "", info.Model)
	assert.Equal(t, int64(1), info.ProcessorCount)
	// ARM: no CPUCores info, falls back to processor count for cores
	assert.Equal(t, int64(2), info.Cores)
}
