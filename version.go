package cnquery

import (
	"fmt"
	"regexp"
)

const Name = "cnquery"

// Version is set via ldflags
var Version string

// Build version is set via ldflags
var Build string

// Date is set via ldflags
var Date string

// DumpLocal configures if resolved policies are dumped locally. If it is
// a non-empty string, it will be used as the path to store dumps into.
var DumpLocal string

/*
 versioning follows semver guidelines: https://semver.org/

<valid semver> ::= <version core>
                 | <version core> "-" <pre-release>
                 | <version core> "+" <build>
                 | <version core> "-" <pre-release> "+" <build>

<version core> ::= <major> "." <minor> "." <patch>

<major> ::= <numeric identifier>

<minor> ::= <numeric identifier>

<patch> ::= <numeric identifier>
*/

// GetVersion returns the version of the build
// valid semver version including build version (e.g. 4.10.0+4900), where 4900 is a forward rolling int
func GetVersion() string {
	if Version == "" {
		return "unstable"
	}
	return Version
}

var majorVersionRegex = regexp.MustCompile(`^(\d+)`)

// the API version is the major version of the version string
// major version number only (e.g. 4)
func ApiVersion() string {
	v := Version

	if v != "" {
		v = majorVersionRegex.FindString(v)
	}

	if v == "" {
		return "unstable"
	}
	return v
}

var fullVersionRegex = regexp.MustCompile(`^(\d+.\d+.\d+)`)

// major and minor version (e.g. 4.10.0)
func GetCoreVersion() string {
	v := Version

	if v != "" {
		v = fullVersionRegex.FindString(v)
	}

	if v == "" {
		return "unstable"
	}
	return v
}

// GetBuild returns the git sha of the build
func GetBuild() string {
	b := Build
	if len(b) == 0 {
		b = "development"
	}
	return b
}

func GetDate() string {
	d := Date
	if len(d) == 0 {
		d = "unknown"
	}
	return d
}

// Info on this application with version and build
func Info() string {
	return fmt.Sprintf("%s %s (%s, %s)", Name, GetVersion(), GetBuild(), GetDate())
}

func LatestMQLVersion() string {
	return "v2"
}
