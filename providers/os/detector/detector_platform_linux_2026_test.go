// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package detector

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRhel10Detector(t *testing.T) {
	di, err := detectPlatformFromMock("./testdata/detect-rhel-10.toml")
	assert.Nil(t, err, "was able to create the provider")

	assert.Equal(t, "redhat", di.Name, "os name should be identified")
	assert.Equal(t, "Red Hat Enterprise Linux 10.0 (Coughlan)", di.Title, "os title should be identified")
	assert.Equal(t, "10.0", di.Version, "os version should be identified")
	assert.Equal(t, "x86_64", di.Arch, "os arch should be identified")
	assert.Equal(t, []string{"redhat", "linux", "unix", "os"}, di.Family)
}

func TestAmazon2023LinuxDetector(t *testing.T) {
	di, err := detectPlatformFromMock("./testdata/detect-amzn-2023.toml")
	assert.Nil(t, err, "was able to create the provider")

	assert.Equal(t, "amazonlinux", di.Name, "os name should be identified")
	assert.Equal(t, "Amazon Linux 2023.12.20260817", di.Title, "os title should be identified")
	assert.Equal(t, "2023", di.Version, "os version should be identified")
	assert.Equal(t, "x86_64", di.Arch, "os arch should be identified")
	assert.Equal(t, []string{"linux", "unix", "os"}, di.Family)
}

func TestDebian11Detector(t *testing.T) {
	di, err := detectPlatformFromMock("./testdata/detect-debian11.toml")
	assert.Nil(t, err, "was able to create the provider")

	assert.Equal(t, "debian", di.Name, "os name should be identified")
	assert.Equal(t, "Debian GNU/Linux 11 (bullseye)", di.Title, "os title should be identified")
	assert.Equal(t, "11.11", di.Version, "os version should be identified")
	assert.Equal(t, "x86_64", di.Arch, "os arch should be identified")
	assert.Equal(t, []string{"debian", "linux", "unix", "os"}, di.Family)
}

func TestDebian12Detector(t *testing.T) {
	di, err := detectPlatformFromMock("./testdata/detect-debian12.toml")
	assert.Nil(t, err, "was able to create the provider")

	assert.Equal(t, "debian", di.Name, "os name should be identified")
	assert.Equal(t, "Debian GNU/Linux 12 (bookworm)", di.Title, "os title should be identified")
	assert.Equal(t, "12.15", di.Version, "os version should be identified")
	assert.Equal(t, "x86_64", di.Arch, "os arch should be identified")
	assert.Equal(t, []string{"debian", "linux", "unix", "os"}, di.Family)
}

func TestDebian13Detector(t *testing.T) {
	di, err := detectPlatformFromMock("./testdata/detect-debian13.toml")
	assert.Nil(t, err, "was able to create the provider")

	assert.Equal(t, "debian", di.Name, "os name should be identified")
	assert.Equal(t, "Debian GNU/Linux 13 (trixie)", di.Title, "os title should be identified")
	assert.Equal(t, "13.6", di.Version, "os version should be identified")
	assert.Equal(t, "x86_64", di.Arch, "os arch should be identified")
	assert.Equal(t, []string{"debian", "linux", "unix", "os"}, di.Family)
}

func TestUbuntu2404Detector(t *testing.T) {
	di, err := detectPlatformFromMock("./testdata/detect-ubuntu2404.toml")
	assert.Nil(t, err, "was able to create the provider")

	assert.Equal(t, "ubuntu", di.Name, "os name should be identified")
	assert.Equal(t, "Ubuntu 24.04.4 LTS", di.Title, "os title should be identified")
	assert.Equal(t, "24.04", di.Version, "os version should be identified")
	assert.Equal(t, "x86_64", di.Arch, "os arch should be identified")
	assert.Equal(t, []string{"debian", "linux", "unix", "os"}, di.Family)
}

func TestUbuntu2604Detector(t *testing.T) {
	di, err := detectPlatformFromMock("./testdata/detect-ubuntu2604.toml")
	assert.Nil(t, err, "was able to create the provider")

	assert.Equal(t, "ubuntu", di.Name, "os name should be identified")
	assert.Equal(t, "Ubuntu 26.04 LTS", di.Title, "os title should be identified")
	assert.Equal(t, "26.04", di.Version, "os version should be identified")
	assert.Equal(t, "x86_64", di.Arch, "os arch should be identified")
	assert.Equal(t, []string{"debian", "linux", "unix", "os"}, di.Family)
}

func TestSles16Detector(t *testing.T) {
	di, err := detectPlatformFromMock("./testdata/detect-suse-sles-16.toml")
	assert.Nil(t, err, "was able to create the provider")

	assert.Equal(t, "sles", di.Name, "os name should be identified")
	assert.Equal(t, "SUSE Linux Enterprise Server 16.0", di.Title, "os title should be identified")
	assert.Equal(t, "16.0", di.Version, "os version should be identified")
	assert.Equal(t, "x86_64", di.Arch, "os arch should be identified")
	assert.Equal(t, []string{"suse", "linux", "unix", "os"}, di.Family)
}
