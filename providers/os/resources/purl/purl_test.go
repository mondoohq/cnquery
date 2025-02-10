// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package purl_test

import (
	"testing"

	"github.com/package-url/packageurl-go"
	"github.com/stretchr/testify/assert"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v11/providers/os/resources/purl"
)

func TestNewQualifiers(t *testing.T) {
	t.Run("Empty qualifiers map", func(t *testing.T) {
		result := purl.NewQualifiers(map[string]string{})
		assert.Empty(t, result)
	})

	t.Run("Valid qualifiers map", func(t *testing.T) {
		qualifiers := map[string]string{
			"arch": "x86_64",
			"os":   "linux",
		}
		expected := packageurl.Qualifiers{
			{Key: "arch", Value: "x86_64"},
			{Key: "os", Value: "linux"},
		}

		result := purl.NewQualifiers(qualifiers)
		assert.Equal(t, expected, result)
	})

	t.Run("Qualifiers map with empty values", func(t *testing.T) {
		qualifiers := map[string]string{
			"arch": "x86_64",
			"os":   "",
		}
		expected := packageurl.Qualifiers{
			{Key: "arch", Value: "x86_64"},
		}

		result := purl.NewQualifiers(qualifiers)
		assert.Equal(t, expected, result)
	})

	t.Run("Qualifiers map with unsorted keys", func(t *testing.T) {
		qualifiers := map[string]string{
			"os":   "linux",
			"arch": "x86_64",
		}
		expected := packageurl.Qualifiers{
			{Key: "arch", Value: "x86_64"},
			{Key: "os", Value: "linux"},
		}

		result := purl.NewQualifiers(qualifiers)
		assert.Equal(t, expected, result)
	})
}

func TestNewPackageURL(t *testing.T) {
	platform := &inventory.Platform{
		Arch:    "x86_64",
		Version: "22.04",
		Labels: map[string]string{
			"distro-id": "ubuntu",
		},
	}

	t.Run("Basic PackageURL", func(t *testing.T) {
		p := purl.NewPackageURL(platform, purl.TypeApk, "testpkg", "1.0.0")
		assert.Equal(t, purl.TypeApk, p.Type)
		assert.Equal(t, "testpkg", p.Name)
		assert.Equal(t, "1.0.0", p.Version)
		assert.Equal(t, "x86_64", p.Arch)
		assert.Equal(t, "", p.Namespace)
	})

	t.Run("Modifiers applied", func(t *testing.T) {
		p := purl.NewPackageURL(platform, purl.TypeRPM, "testpkg", "1.0.0",
			purl.WithArch("arm64"),
			purl.WithEpoch("1"),
		)
		assert.Equal(t, "arm64", p.Arch)
		assert.Equal(t, "1", p.Epoch)
	})

	t.Run("Nil platform won't discover optional attributes", func(t *testing.T) {
		p := purl.NewPackageURL(nil, purl.TypeDebian, "testpkg", "1.0.0")
		assert.Equal(t, purl.TypeDebian, p.Type)
		assert.Equal(t, "testpkg", p.Name)
		assert.Equal(t, "1.0.0", p.Version)
		assert.Empty(t, p.Arch)
		assert.Empty(t, p.Namespace)
	})
}

func TestPackageURLString(t *testing.T) {
	platform := &inventory.Platform{
		Arch:    "x86_64",
		Version: "22.04",
		Labels: map[string]string{
			"distro-id": "ubuntu",
		},
	}

	t.Run("Basic PackageURL string", func(t *testing.T) {
		p := purl.NewPackageURL(platform, purl.TypeDebian, "testpkg", "1.0.0")
		expected := "pkg:deb/testpkg@1.0.0?arch=x86_64&distro=ubuntu-22.04"
		assert.Equal(t, expected, p.String())
	})

	t.Run("With Epoch", func(t *testing.T) {
		p := purl.NewPackageURL(platform, purl.TypeDebian, "testpkg", "1.0.0",
			purl.WithEpoch("2"),
		)
		expected := "pkg:deb/testpkg@1.0.0?arch=x86_64&distro=ubuntu-22.04&epoch=2"
		assert.Equal(t, expected, p.String())
	})

	t.Run("Without Namespace from platform", func(t *testing.T) {
		platform := &inventory.Platform{
			Arch:    "x86_64",
			Version: "11",
			Labels:  nil,
		}
		p := purl.NewPackageURL(platform, purl.TypeDebian, "testpkg", "1.0.0")
		expected := "pkg:deb/testpkg@1.0.0?arch=x86_64"
		assert.Equal(t, expected, p.String())

		t.Run("But Namespace from modifiers", func(t *testing.T) {
			platform := &inventory.Platform{
				Arch:    "x86_64",
				Version: "11",
				Labels:  nil,
			}
			p := purl.NewPackageURL(platform, purl.TypeDebian, "testpkg", "1.0.0",
				purl.WithNamespace("debian"),
			)
			expected := "pkg:deb/debian/testpkg@1.0.0?arch=x86_64"
			assert.Equal(t, expected, p.String())
		})
	})

	t.Run("Modifiers overriding platform values", func(t *testing.T) {
		p := purl.NewPackageURL(platform, purl.TypeDebian, "testpkg", "1.0.0",
			purl.WithArch("arm64"),
		)
		expected := "pkg:deb/testpkg@1.0.0?arch=arm64&distro=ubuntu-22.04"
		assert.Equal(t, expected, p.String())
	})

	t.Run("Empty Platform and Qualifiers", func(t *testing.T) {
		p := purl.NewPackageURL(nil, purl.TypeApk, "testpkg", "1.0.0")
		expected := "pkg:apk/testpkg@1.0.0"
		assert.Equal(t, expected, p.String())
	})

	t.Run("Non-standard Type", func(t *testing.T) {
		p := purl.NewPackageURL(nil, "customtype", "testpkg", "1.0.0")
		expected := "pkg:customtype/testpkg@1.0.0"
		assert.Equal(t, expected, p.String())
	})

	t.Run("Special characters in fields", func(t *testing.T) {
		p := purl.NewPackageURL(nil, purl.TypeApk, "pkg@123", "1.0.0")
		expected := "pkg:apk/pkg%40123@1.0.0"
		assert.Equal(t, expected, p.String())
	})

	t.Run("Empty name and version", func(t *testing.T) {
		p := purl.NewPackageURL(nil, purl.TypeGeneric, "", "")
		assert.Equal(t, purl.TypeGeneric, p.Type)
		assert.Empty(t, p.Name)
		assert.Empty(t, p.Version)
		assert.Empty(t, p.Namespace)
		assert.Empty(t, p.Arch)
	})

	t.Run("Both version and build specified, we prefer version", func(t *testing.T) {
		platform.Build = "20.04" // just for testing
		p := purl.NewPackageURL(platform, purl.TypeDebian, "testpkg", "1.0.0")
		expected := "pkg:deb/testpkg@1.0.0?arch=x86_64&distro=ubuntu-22.04"
		assert.Equal(t, expected, p.String())
		t.Run("Only build specified", func(t *testing.T) {
			platform.Version = ""
			p := purl.NewPackageURL(platform, purl.TypeDebian, "testpkg", "1.0.0")
			expected := "pkg:deb/testpkg@1.0.0?arch=x86_64&distro=ubuntu-20.04"
			assert.Equal(t, expected, p.String())
		})
	})

	t.Run("Set platform name", func(t *testing.T) {
		platform.Name = "ubuntu"
		p := purl.NewPackageURL(platform, purl.TypeDebian, "testpkg", "1.0.0")
		expected := "pkg:deb/ubuntu/testpkg@1.0.0?arch=x86_64&distro=ubuntu-20.04"
		assert.Equal(t, expected, p.String())
	})

	t.Run("Red Hat package", func(t *testing.T) {
		platform := &inventory.Platform{
			Arch:    "x86_64",
			Version: "9.2",
			Labels: map[string]string{
				"distro-id": "rhel",
			},
		}
		p := purl.NewPackageURL(platform, purl.TypeRPM, "testpkg", "1.0.0")
		expected := "pkg:rpm/testpkg@1.0.0?arch=x86_64&distro=rhel-9.2"
		assert.Equal(t, expected, p.String())
	})

	t.Run("Red Hat package without distro-id", func(t *testing.T) {
		platform := &inventory.Platform{
			Name:    "redhat",
			Arch:    "x86_64",
			Version: "9.2",
			Labels:  nil,
		}
		p := purl.NewPackageURL(platform, purl.TypeRPM, "testpkg", "1.0.0")
		expected := "pkg:rpm/redhat/testpkg@1.0.0?arch=x86_64"
		assert.Equal(t, expected, p.String())
	})

	t.Run("Red Hat package with distro-id and name", func(t *testing.T) {
		platform := &inventory.Platform{
			Name:    "redhat",
			Arch:    "x86_64",
			Version: "9.2",
			Labels: map[string]string{
				"distro-id": "rhel",
			},
		}
		p := purl.NewPackageURL(platform, purl.TypeRPM, "testpkg", "1.0.0")
		expected := "pkg:rpm/redhat/testpkg@1.0.0?arch=x86_64&distro=rhel-9.2"
		assert.Equal(t, expected, p.String())
	})
}
