// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package purl_test

import (
	"testing"

	"github.com/package-url/packageurl-go"
	"github.com/stretchr/testify/assert"
	"go.mondoo.com/mql/providers-sdk/v1/inventory"
	"go.mondoo.com/mql/providers/os/detector"
	"go.mondoo.com/mql/providers/os/resources/purl"
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

	t.Run("Photon package", func(t *testing.T) {
		platform := &inventory.Platform{
			Name:    "photon",
			Arch:    "x86_64",
			Version: "4.0",
			Labels:  nil,
		}
		p := purl.NewPackageURL(platform, purl.TypeRPM, "testpkg", "1.0.0")
		expected := "pkg:rpm/photon%20os/testpkg@1.0.0?arch=x86_64"
		assert.Equal(t, expected, p.String())
	})

	t.Run("Photon package with distro-id and name", func(t *testing.T) {
		platform := &inventory.Platform{
			Name:    "photon",
			Arch:    "x86_64",
			Version: "4.0",
			Labels: map[string]string{
				"distro-id": "photon",
			},
		}
		p := purl.NewPackageURL(platform, purl.TypeRPM, "testpkg", "1.0.0")
		expected := "pkg:rpm/photon%20os/testpkg@1.0.0?arch=x86_64&distro=photon-4.0"
		assert.Equal(t, expected, p.String())
	})

	t.Run("Rocky Linux package", func(t *testing.T) {
		platform := &inventory.Platform{
			Name:    "rockylinux",
			Arch:    "x86_64",
			Version: "8.6",
			Labels:  nil,
		}
		p := purl.NewPackageURL(platform, purl.TypeRPM, "testpkg", "1.0.0")
		expected := "pkg:rpm/rocky-linux/testpkg@1.0.0?arch=x86_64"
		assert.Equal(t, expected, p.String())
	})

	t.Run("openSUSE package", func(t *testing.T) {
		platform := &inventory.Platform{
			Name:    "opensuse-leap",
			Arch:    "x86_64",
			Version: "15.4",
			Labels:  nil,
		}
		p := purl.NewPackageURL(platform, purl.TypeRPM, "testpkg", "1.0.0")
		expected := "pkg:rpm/opensuse/testpkg@1.0.0?arch=x86_64"
		assert.Equal(t, expected, p.String())
	})

	t.Run("SUSE package", func(t *testing.T) {
		platform := &inventory.Platform{
			Name:    "sles",
			Arch:    "x86_64",
			Version: "15.4",
			Labels:  nil,
		}
		p := purl.NewPackageURL(platform, purl.TypeRPM, "testpkg", "1.0.0")
		expected := "pkg:rpm/suse/testpkg@1.0.0?arch=x86_64"
		assert.Equal(t, expected, p.String())
	})

	// matches what openSUSE's own build service emits for MicroOS packages:
	// the namespace collapses to opensuse and the product stays in the distro
	// qualifier.
	t.Run("openSUSE MicroOS package", func(t *testing.T) {
		platform := &inventory.Platform{
			Name:    "opensuse-microos",
			Arch:    "x86_64",
			Version: "20260822",
			Labels:  map[string]string{detector.LabelDistroID: "opensuse-microos"},
		}
		p := purl.NewPackageURL(platform, purl.TypeRPM, "testpkg", "1.0.0")
		expected := "pkg:rpm/opensuse/testpkg@1.0.0?arch=x86_64&distro=opensuse-microos-20260822"
		assert.Equal(t, expected, p.String())
	})

	// Arch and its derivatives ship no VERSION_ID, only BUILD_ID=rolling, so
	// the distro qualifier has to come from the build id.
	t.Run("rolling release package", func(t *testing.T) {
		platform := &inventory.Platform{
			Name:  "arch",
			Arch:  "x86_64",
			Build: "rolling",
			Labels: map[string]string{
				"distro-id": "arch",
			},
		}
		p := purl.NewPackageURL(platform, purl.TypeAlpm, "testpkg", "1.0.0")
		expected := "pkg:alpm/arch/testpkg@1.0.0?arch=x86_64&distro=arch-rolling"
		assert.Equal(t, expected, p.String())
	})

	t.Run("package without version or build", func(t *testing.T) {
		platform := &inventory.Platform{
			Name: "arch",
			Arch: "x86_64",
			Labels: map[string]string{
				"distro-id": "arch",
			},
		}
		p := purl.NewPackageURL(platform, purl.TypeAlpm, "testpkg", "1.0.0")
		expected := "pkg:alpm/arch/testpkg@1.0.0?arch=x86_64&distro=arch"
		assert.Equal(t, expected, p.String())
	})
}

// TestPackageURLEncoding pins how each purl segment is percent-encoded.
//
// These live in their own function, with their own fixtures, deliberately.
// TestPackageURLString shares one `platform` value that later subtests mutate
// in place (Build, Version, Name), so anything added mid-chain is sensitive to
// its position and would break if the subtests were reordered or parallelized.
//
// The cases cover characters that a plausible encoder regression would treat
// differently. That rules out the values used elsewhere in this file --
// "debian", "ubuntu", "x86_64", "arm64" -- which survive any encoding scheme
// and so cannot detect one changing.
func TestPackageURLEncoding(t *testing.T) {
	// Name and version. packageurl-go v0.1.6 stopped percent-encoding the RFC
	// 3986 sub-delimiters here and emitted "... Redistributable (x64)@..."
	// instead of "%28x64%29"; we pinned to v0.1.5 until the upstream fix
	// (package-url/packageurl-go#93) shipped in v0.1.7. These fail on v0.1.6.
	t.Run("Name and version", func(t *testing.T) {
		t.Run("Every RFC 3986 sub-delimiter", func(t *testing.T) {
			p := purl.NewPackageURL(nil, purl.TypeGeneric, "a!b$c&d'e(f)g*h+i,j;k=l", "1;2=3")
			expected := "pkg:generic/a%21b%24c%26d%27e%28f%29g%2Ah%2Bi%2Cj%3Bk%3Dl@1%3B2%3D3"
			assert.Equal(t, expected, p.String())
		})

		t.Run("Windows display name with parentheses", func(t *testing.T) {
			p := purl.NewPackageURL(nil, purl.TypeWindows,
				"Microsoft Visual C++ 2015-2022 Redistributable (x64)", "14.38.33130.0")
			expected := "pkg:windows/Microsoft%20Visual%20C%2B%2B%202015-2022%20Redistributable%20%28x64%29@14.38.33130.0"
			assert.Equal(t, expected, p.String())
		})
	})

	// Namespace. v0.1.7 replaced the namespace encoder wholesale (split-and-
	// encode became an index-scanning loop), and this is not a hypothetical
	// path for us: NewPackageURL maps the photon platform to the namespace
	// "photon os", which contains a space, and windows-driver namespaces are
	// arbitrary vendor strings.
	t.Run("Namespace", func(t *testing.T) {
		t.Run("Space in a platform-derived namespace", func(t *testing.T) {
			p := purl.NewPackageURL(nil, purl.TypeGeneric, "vim", "9.0",
				purl.WithNamespace("photon os"))
			assert.Equal(t, "pkg:generic/photon%20os/vim@9.0", p.String())
		})

		t.Run("Sub-delimiters in a vendor namespace", func(t *testing.T) {
			p := purl.NewPackageURL(nil, purl.TypeWindowsDriver, "PCL 6 Driver", "1.0",
				purl.WithNamespace("Brother & Co."))
			assert.Equal(t, "pkg:windows-driver/Brother%20%26%20Co./PCL%206%20Driver@1.0", p.String())
		})
	})

	// Qualifier values. v0.1.7 replaced the qualifier encoder too. We put
	// free-form text here: macos_packages.go sets `remoting-name` from an
	// app's RemotingName and windows_m365_channel.go sets `channel`.
	t.Run("Qualifier values", func(t *testing.T) {
		p := purl.NewPackageURL(nil, purl.TypeMacos, "app", "1.0",
			purl.WithQualifiers(map[string]string{"remoting-name": "Photo & Video Editor"}))
		assert.Equal(t, "pkg:macos/app@1.0?remoting-name=Photo%20%26%20Video%20Editor", p.String())
	})
}

// TestPackageURLStringDoesNotMutateQualifiers guards the map clone in String().
//
// The qualifier map belongs to the caller. String() adds the derived
// arch/epoch/distro entries, and when it wrote them into the caller's map, a
// second package rendered from the same map inherited the first one's platform
// -- reporting an arch and distro it never had, with no error anywhere.
//
// No caller reuses a map today (each builds a fresh one per package), so this
// pins the contract before someone hoists an allocation out of a loop.
func TestPackageURLStringDoesNotMutateQualifiers(t *testing.T) {
	shared := map[string]string{"custom": "value"}

	platform := &inventory.Platform{
		Arch:    "x86_64",
		Version: "22.04",
		Name:    "ubuntu",
		Labels:  map[string]string{"distro-id": "ubuntu"},
	}

	withPlatform := purl.NewPackageURL(platform, purl.TypeDebian, "pkgA", "1.0",
		purl.WithQualifiers(shared))
	assert.Equal(t, "pkg:deb/ubuntu/pkgA@1.0?arch=x86_64&custom=value&distro=ubuntu-22.04",
		withPlatform.String())

	assert.Equal(t, map[string]string{"custom": "value"}, shared,
		"String() must not write its derived qualifiers into the caller's map")

	// The leak was observable here: rendered with no platform, this package
	// used to come back carrying the first one's arch and distro.
	noPlatform := purl.NewPackageURL(nil, purl.TypeDebian, "pkgB", "2.0",
		purl.WithQualifiers(shared))
	assert.Equal(t, "pkg:deb/pkgB@2.0?custom=value", noPlatform.String())
}
