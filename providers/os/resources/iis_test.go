// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/inventory"
	"go.mondoo.com/mql/v13/providers-sdk/v1/testutils"
	"go.mondoo.com/mql/v13/providers/os/resources/powershell"
	"go.mondoo.com/mql/v13/providers/os/resources/windows"
)

// The recording format, mirrored here so a recording can be built in the test
// rather than checked in. Checking one in would pin the base64 of the collection
// script as a lookup key, and the first edit to the script would leave a
// recording nothing matches.
type iisRecordingResource struct {
	Resource string
	ID       string
	Fields   map[string]*llx.RawData
}

type iisRecordingConnection struct {
	Url        string `json:"url"`
	ProviderID string `json:"provider"`
	Connector  string `json:"connector"`
	Version    string `json:"version"`
}

type iisRecordingAsset struct {
	Asset       *inventory.Asset         `json:"asset"`
	Connections []iisRecordingConnection `json:"connections"`
	Resources   []iisRecordingResource   `json:"resources"`
}

type iisRecording struct {
	Assets []iisRecordingAsset `json:"assets"`
}

// iisMock builds a Windows runtime whose only recorded command is the IIS
// collection script, answering with the given payload. The command key is
// derived from the shipped script, so it stays correct however the script
// changes.
func iisMock(t *testing.T, payload string) llx.Runtime {
	t.Helper()
	return iisMockCommand(t, 0, payload, "")
}

// iisFailingMock answers the collection script the way a host that refuses it
// does: a non-zero exit code and a message on stderr.
func iisFailingMock(t *testing.T, stderr string) llx.Runtime {
	t.Helper()
	return iisMockCommand(t, 1, "", stderr)
}

func iisMockCommand(t *testing.T, exitcode int64, stdout string, stderr string) llx.Runtime {
	t.Helper()

	rec := iisRecording{
		Assets: []iisRecordingAsset{
			{
				Asset: &inventory.Asset{
					Id:          "windows-iis",
					PlatformIds: []string{"windows"},
					Name:        "windows",
					Platform: &inventory.Platform{
						Name:    "windows",
						Arch:    "x86_64",
						Title:   "Windows Server",
						Family:  []string{"windows", "os"},
						Build:   "rolling",
						Version: "2022",
					},
				},
				Connections: []iisRecordingConnection{
					{Url: "local://", ProviderID: "os", Connector: "local"},
				},
				Resources: []iisRecordingResource{
					{
						Resource: "command",
						// The staged command, not powershell.Encode: the script
						// is far too long for a command line and is written to
						// the target as a file. That command *is* the id of the
						// `command` resource, so it is the key a recording is
						// filed under — and it stays derivable here because the
						// path is a client-side literal plus the script's
						// content hash, with nothing read off a host.
						ID: powershell.StagedCommand(
							powershell.StagedWindowsPath("iis", windows.IIS_CONFIGURATION)),
						Fields: map[string]*llx.RawData{
							"exitcode": llx.IntData(exitcode),
							"stdout":   llx.StringData(stdout),
							"stderr":   llx.StringData(stderr),
						},
					},
				},
			},
		},
	}

	data, err := json.Marshal(rec)
	require.NoError(t, err)

	path := filepath.Join(t.TempDir(), "iis-recording.json")
	require.NoError(t, os.WriteFile(path, data, 0o600))

	abs, err := filepath.Abs(path)
	require.NoError(t, err)
	return testutils.RecordingMock(abs)
}

func iisFixture(t *testing.T, name string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("windows", "testdata", name))
	require.NoError(t, err)
	return string(data)
}

// TestIisResourceOverrides drives the whole resource through MQL. It is the
// counterpart to the parser tests: it proves the schema wires up, the typed
// application pool reference resolves, and the values a policy would read are
// the resolved ones.
//
// It runs on the **hand-authored** payload rather than on the capture, because
// it is about the override case — a site whose configuration disagrees with the
// server's — and no host this suite has captured has one. See the comment on
// iisSynthetic in windows/iis_test.go. TestIisResourceOnCapture below is the
// same wiring driven by a payload a real server produced.
func TestIisResourceOverrides(t *testing.T) {
	runtime := iisMock(t, iisFixture(t, "iis_synthetic.json"))
	tester := testutils.InitTester(runtime)

	tester.TestSimple(t, []testutils.SimpleTest{
		{Code: "iis.installed", ResultIndex: 0, Expectation: true},
		{Code: "iis.version", ResultIndex: 0, Expectation: "10.0"},
		{Code: "iis.sites.length", ResultIndex: 0, Expectation: int64(2)},
		{Code: "iis.appPools.length", ResultIndex: 0, Expectation: int64(2)},

		// The server scope declares directory browsing off. Reading only this
		// would report the site below as compliant.
		{Code: "iis.config.path", ResultIndex: 0, Expectation: "MACHINE/WEBROOT/APPHOST"},
		{Code: "iis.config.directoryBrowsingEnabled", ResultIndex: 0, Expectation: false},

		// The legacy site overrides it, and the resource reports the override.
		{
			Code:        `iis.sites.where(name == "legacy")[0].config.directoryBrowsingEnabled`,
			ResultIndex: 0, Expectation: true,
		},
		{
			Code:        `iis.sites.where(name == "legacy")[0].config.path`,
			ResultIndex: 0, Expectation: "MACHINE/WEBROOT/APPHOST/legacy",
		},
		{
			Code:        `iis.sites.where(name == "legacy")[0].config.machineKeyValidation`,
			ResultIndex: 0, Expectation: "SHA1",
		},
		{
			Code:        `iis.sites.where(name == "legacy")[0].config.sessionStateTimeout`,
			ResultIndex: 0, Expectation: int64(3600),
		},
		{
			Code:        `iis.sites.where(name == "Default Web Site")[0].config.machineKeyValidation`,
			ResultIndex: 0, Expectation: "HMACSHA256",
		},

		// A section this payload does not declare reads null, not false. Note
		// what is *not* being claimed: a real server does declare these at
		// server scope, and TestIisResourceOnCapture reads values for them.
		// This payload omits them, and the point is only that an absent section
		// yields null rather than a made-up zero value.
		{Code: "iis.config.compilationDebug", ResultIndex: 0, Expectation: nil},
		{Code: "iis.config.sessionStateMode", ResultIndex: 0, Expectation: nil},

		// The typed application pool reference resolves to the pool itself.
		{
			Code:        `iis.sites.where(name == "legacy")[0].appPool.identityType`,
			ResultIndex: 0, Expectation: "SpecificUser",
		},
		{
			Code:        `iis.sites.where(name == "Default Web Site")[0].appPool.identityType`,
			ResultIndex: 0, Expectation: "ApplicationPoolIdentity",
		},
		{
			Code:        `iis.appPools.where(name == "LegacyPool")[0].idleTimeout`,
			ResultIndex: 0, Expectation: int64(0),
		},

		// An application resolves at its own scope, below the site's.
		{
			Code:        `iis.sites.where(name == "legacy")[0].applications.where(path == "/shop")[0].config.sessionStateTimeout`,
			ResultIndex: 0, Expectation: int64(900),
		},
		{
			Code:        `iis.sites.where(name == "legacy")[0].applications.where(path == "/shop")[0].config.path`,
			ResultIndex: 0, Expectation: "MACHINE/WEBROOT/APPHOST/legacy/shop",
		},
		// The root application shares the site's scope, so it reports the site's
		// value rather than resolving a second time.
		{
			Code:        `iis.sites.where(name == "legacy")[0].applications.where(path == "/")[0].config.sessionStateTimeout`,
			ResultIndex: 0, Expectation: int64(3600),
		},

		{
			Code:        `iis.sites.where(name == "legacy")[0].bindings.where(protocol == "https")[0].port`,
			ResultIndex: 0, Expectation: int64(443),
		},
		{
			Code:        `iis.sites.where(name == "legacy")[0].config.customHeaders["X-Powered-By"]`,
			ResultIndex: 0, Expectation: "ASP.NET",
		},
		// Every application declares a virtual directory at "/", so the ids have
		// to carry the site and the application as well as the path. If they did
		// not, the second one would resolve to the cached first and report its
		// physical path.
		{
			Code:        `iis.sites.where(name == "legacy")[0].applications.where(path == "/")[0].virtualDirectories[0].physicalPath`,
			ResultIndex: 0, Expectation: `D:\sites\legacy`,
		},
		{
			Code:        `iis.sites.where(name == "legacy")[0].applications.where(path == "/shop")[0].virtualDirectories[0].physicalPath`,
			ResultIndex: 0, Expectation: `D:\sites\legacy\shop`,
		},
		{
			Code:        `iis.sites.where(name == "Default Web Site")[0].applications[0].virtualDirectories[0].path`,
			ResultIndex: 0, Expectation: "/",
		},

		// The escape hatch reaches a section with no field of its own.
		{
			Code:        `iis.config.sections["system.webServer/security/requestFiltering"]["requestLimits"]["maxUrl"]`,
			ResultIndex: 0, Expectation: float64(4096),
		},
	})
}

// TestIisConfigurationSectionsReachEverySetting exercises the escape hatch on
// settings deliberately left without a field of their own, because a field that
// is only a second name for a value already in `sections` is API we would have
// to keep. These paths are the documented alternative, so they are tested rather
// than assumed: the verbs filter, which shares its attribute name with the file
// extension filter that does have a field, and the machine key cipher, which
// sits beside the validation algorithm that does.
func TestIisConfigurationSectionsReachEverySetting(t *testing.T) {
	runtime := iisMock(t, iisFixture(t, "iis_synthetic.json"))
	tester := testutils.InitTester(runtime)

	tester.TestSimple(t, []testutils.SimpleTest{
		{
			Code:        `iis.config.sections["system.webServer/security/requestFiltering"]["verbs"]["allowUnlisted"]`,
			ResultIndex: 0, Expectation: true,
		},
		{
			Code:        `iis.config.sections["system.webServer/security/requestFiltering"]["fileExtensions"]["allowUnlisted"]`,
			ResultIndex: 0, Expectation: true,
		},
		{
			Code:        `iis.sites.where(name == "legacy")[0].config.sections["system.web/machineKey"]["decryption"]`,
			ResultIndex: 0, Expectation: "Auto",
		},

		// A section the payload does not declare has no key, and reading
		// through the missing key yields null instead of failing the whole
		// query. That is what makes the escape hatch usable against a scope
		// that genuinely declares less than another one.
		{Code: `iis.config.sections["system.web/machineKey"]`, ResultIndex: 0, Expectation: nil},
		{Code: `iis.config.sections["system.web/machineKey"]["decryption"]`, ResultIndex: 0, Expectation: nil},

		// Key material is dropped during collection, so it is in neither a field
		// nor the raw section.
		{
			Code:        `iis.sites.where(name == "legacy")[0].config.sections["system.web/machineKey"]["validationKey"]`,
			ResultIndex: 0, Expectation: nil,
		},
	})
}

// TestIisCollectFailurePropagates covers the failure mode the shared collection
// creates. Every field below iis comes out of one script run, but the fields do
// not share a cache entry, so an error handed only to whichever field ran first
// would leave the rest reading an empty result — reporting a host that refused
// the collection as a host that simply does not run IIS, on which every check
// passes. The second and later accessors are the case that matters here.
// TestIisResourceOnCapture drives the resource through MQL on a payload a real
// Windows Server 2022 produced, in its non-hardened state. Its point is the
// enum-valued fields: every one of them used to reach a policy as a bare number
// formatted as a string, which is a value that looks usable and is not.
func TestIisResourceOnCapture(t *testing.T) {
	runtime := iisMock(t, iisFixture(t, "iis.json"))
	tester := testutils.InitTester(runtime)

	tester.TestSimple(t, []testutils.SimpleTest{
		{Code: "iis.installed", ResultIndex: 0, Expectation: true},
		{Code: "iis.version", ResultIndex: 0, Expectation: "10.0"},
		{Code: "iis.sites.length", ResultIndex: 0, Expectation: int64(2)},
		{Code: "iis.appPools.length", ResultIndex: 0, Expectation: int64(2)},

		// A flags attribute: 519 is Read|Write|Execute|Script, which is not a
		// member of its own enum and cannot be resolved by a lookup.
		{Code: "iis.config.handlerAccessPolicy", ResultIndex: 0, Expectation: "Read, Write, Execute, Script"},

		// Plain enums, all of which are Int32 in the payload the API hands over.
		{Code: "iis.config.authenticationMode", ResultIndex: 0, Expectation: "None"},
		{Code: "iis.config.sessionStateMode", ResultIndex: 0, Expectation: "StateServer"},
		{Code: "iis.config.sessionStateCookieless", ResultIndex: 0, Expectation: "UseUri"},
		{Code: "iis.config.machineKeyValidation", ResultIndex: 0, Expectation: "MD5"},
		{Code: "iis.config.customErrorsMode", ResultIndex: 0, Expectation: "Off"},
		{Code: "iis.config.httpErrorsMode", ResultIndex: 0, Expectation: "Detailed"},
		{Code: "iis.config.formsProtection", ResultIndex: 0, Expectation: "None"},

		// The ASP.NET sections read at server scope. The schema used to
		// document them as null here and the tests used to assert it.
		{Code: "iis.config.compilationDebug", ResultIndex: 0, Expectation: true},
		{Code: "iis.config.trustLevel", ResultIndex: 0, Expectation: "Full"},
		{Code: `iis.config.sections["system.web/machineKey"]["decryption"]`, ResultIndex: 0, Expectation: "Auto"},

		// Key material is dropped during collection, so it reaches neither a
		// typed field nor the raw section — on a real payload, not only on one
		// written to omit it.
		{Code: `iis.config.sections["system.web/machineKey"]["validationKey"]`, ResultIndex: 0, Expectation: nil},
		{Code: `iis.config.sections["system.web/machineKey"]["decryptionKey"]`, ResultIndex: 0, Expectation: nil},

		// A site's log format, as everything that names a log format spells it.
		{
			Code:        `iis.sites.where(name == "Default Web Site")[0].logFormat`,
			ResultIndex: 0, Expectation: "NCSA",
		},
		// The one scope in this capture that actually overrides its parent.
		{
			Code:        `iis.sites.where(name == "fixture-b")[0].applications.where(path == "/app1")[0].config.sessionStateTimeout`,
			ResultIndex: 0, Expectation: int64(3600),
		},
		{
			Code:        `iis.sites.where(name == "fixture-b")[0].config.sessionStateTimeout`,
			ResultIndex: 0, Expectation: int64(7200),
		},
	})
}

func TestIisCollectFailurePropagates(t *testing.T) {
	runtime := iisFailingMock(t, "access is denied")
	tester := testutils.InitTester(runtime)

	const expected = "failed to read IIS configuration: access is denied"

	// The first field to ask runs the collection and sees the failure.
	tester.TestSimpleErrors(t, []testutils.SimpleTest{
		{Code: "iis.installed", ResultIndex: 0, Expectation: expected},
	})

	// Every later field reaches the cached outcome and must see the same
	// failure. Reporting false, "" or an empty list here would be the silent
	// pass this test exists to prevent.
	tester.TestSimpleErrors(t, []testutils.SimpleTest{
		{Code: "iis.installed", ResultIndex: 0, Expectation: expected},
		{Code: "iis.version", ResultIndex: 0, Expectation: expected},
		{Code: "iis.config", ResultIndex: 0, Expectation: expected},
		{Code: "iis.applicationHost", ResultIndex: 0, Expectation: expected},
		{Code: "iis.sites", ResultIndex: 0, Expectation: expected},
		{Code: "iis.appPools", ResultIndex: 0, Expectation: expected},
	})
}

// TestIisResourceAbsent is the case the skill calls out: on a host that does not
// run IIS the resource must answer, and it must not answer in a way that makes a
// check pass. installed is false and the collections are empty, so a policy has
// to filter on installed rather than iterate.
func TestIisResourceAbsent(t *testing.T) {
	runtime := iisMock(t, iisFixture(t, "iis_not_installed.json"))
	tester := testutils.InitTester(runtime)

	tester.TestSimple(t, []testutils.SimpleTest{
		{Code: "iis.installed", ResultIndex: 0, Expectation: false},
		{Code: "iis.version", ResultIndex: 0, Expectation: ""},
		{Code: "iis.sites.length", ResultIndex: 0, Expectation: int64(0)},
		{Code: "iis.appPools.length", ResultIndex: 0, Expectation: int64(0)},
		// Null rather than an empty configuration whose every field reads false.
		{Code: "iis.config", ResultIndex: 0, Expectation: nil},
		{Code: "iis.applicationHost", ResultIndex: 0, Expectation: nil},
	})
}
