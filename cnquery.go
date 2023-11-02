// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package cnquery

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
)

// Version is set via ldflags
var Version string

// Build version is set via ldflags
var Build string

// Date is set via ldflags
var Date string

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

// Release represents a GitHub release
type Release struct {
	Name string `json:"name"`
}

var cnqueryGithubReleaseUrl = "https://api.github.com/repos/mondoohq/cnquery/releases/latest"

// GetLatestReleaseName fetches the name of the latest release from the specified GitHub repository
func GetLatestReleaseName(url string) (string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", fmt.Errorf("error fetching latest release: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("received non-OK response status: %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("error reading response body: %v", err)
	}

	var release Release
	if err := json.Unmarshal(body, &release); err != nil {
		return "", fmt.Errorf("error unmarshalling response: %v", err)
	}

	return release.Name, nil
}

// GetLatestVersion returns the latest version available on Github
func GetLatestVersion() (string, error) {
	releaseName, err := GetLatestReleaseName(cnqueryGithubReleaseUrl)
	if err != nil {
		return "", err
	}
	cleanVersion := strings.TrimPrefix(releaseName, "v")
	return cleanVersion, nil
}

var coreSemverRegex = regexp.MustCompile(`^(\d+.\d+.\d+)`)

// GetCoreVersion returns the semver core (i.e. major.minor.patch)
func GetCoreVersion() string {
	v := Version

	if v != "" {
		v = coreSemverRegex.FindString(v)
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

// GetDate returns the date of this build
func GetDate() string {
	d := Date
	if len(d) == 0 {
		d = "unknown"
	}
	return d
}

var majorVersionRegex = regexp.MustCompile(`^(\d+)`)

// APIVersion is the major version of the version string (e.g. 4)
func APIVersion() string {
	v := Version

	if v != "" {
		v = majorVersionRegex.FindString(v)
	}

	if v == "" {
		return "unstable"
	}
	return v
}

// Info on this application with version and build
func Info() string {
	return "cnquery " + GetVersion() + " (" + GetBuild() + ", " + GetDate() + ")"
}

// LatestMQLVersion returns the current version of MQL
func LatestMQLVersion() string {
	return "v2"
}
