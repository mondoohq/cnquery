// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package providers

import (
	"os/exec"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/mql/v13/test"
)

// rpmTestImages are rpm-based images we exercise both package code paths
// against. They mirror the distros we keep mock fixtures for.
var rpmTestImages = []string{
	"oraclelinux:9",
	"almalinux:8",
}

type rpmPackage struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Arch    string `json:"arch"`
	Format  string `json:"format"`
	Vendor  string `json:"vendor"`
}

type rpmPackagesResult []struct {
	List []rpmPackage `json:"packages.list"`
}

// requireDocker fails the test (it does NOT skip) when docker is unavailable.
// A silently skipped integration test gives the same false-green as the
// hand-edited mock fixtures that caused the rpm delimiter incident: the test
// looks healthy while never exercising the real code path. If docker is
// missing where these tests run, that is a setup error worth a red build.
func requireDocker(t *testing.T) {
	t.Helper()
	out, err := exec.Command("docker", "version", "--format", "{{.Server.Version}}").CombinedOutput()
	require.NoErrorf(t, err, "docker is required for the rpm integration tests: %s", string(out))
}

// dockerRunDetached starts a long-lived container from image and returns its
// id. A *running* container is what routes mql to the runtime code path
// (ContainerConnection.RunCommand -> real `rpm -qa --queryformat`).
func dockerRunDetached(t *testing.T, image string) string {
	t.Helper()
	// Pull explicitly first: `docker run` prints image-pull progress, and if
	// that progress ends up mixed with the container id we read back, the id
	// becomes garbage and mql falls back to printing usage.
	if out, err := exec.Command("docker", "pull", image).CombinedOutput(); err != nil {
		t.Fatalf("pulling %s: %v\n%s", image, err, string(out))
	}
	// --rm so the daemon reaps the container if the test process dies before
	// the cleanup callback runs (SIGKILL, OOM, CI timeout). Read only stdout so
	// the container id is never contaminated by warnings on stderr.
	out, err := exec.Command("docker", "run", "-d", "--rm", image, "sleep", "3600").Output()
	require.NoErrorf(t, err, "starting container for %s", image)
	id := strings.TrimSpace(string(out))
	require.NotEmpty(t, id, "empty container id for %s", image)
	t.Cleanup(func() {
		_ = exec.Command("docker", "rm", "-f", id).Run()
	})
	return id
}

// queryRpmPackages runs `mql run <target...> -c packages -j` and returns the
// parsed package list.
func queryRpmPackages(t *testing.T, target ...string) []rpmPackage {
	t.Helper()
	args := make([]string, 0, 1+len(target)+3)
	args = append(args, "run")
	args = append(args, target...)
	args = append(args, "-c", "packages", "-j")
	r := test.NewCliTestRunner("./mql", args...)
	require.NoError(t, r.Run())
	require.Equalf(t, 0, r.ExitCode(), "mql exited non-zero; stderr: %s", string(r.Stderr()))

	var res rpmPackagesResult
	if err := r.Json(&res); err != nil {
		t.Fatalf("parsing mql json failed: %v\n--- stdout ---\n%s\n--- stderr ---\n%s",
			err, string(r.Stdout()), string(r.Stderr()))
	}
	require.NotEmpty(t, res, "no result object from mql")
	return res[0].List
}

func hasPackage(list []rpmPackage, name string) (rpmPackage, bool) {
	for _, p := range list {
		if p.Name == name {
			return p, true
		}
	}
	return rpmPackage{}, false
}

func packageNames(list []rpmPackage) []string {
	names := make([]string, 0, len(list))
	for _, p := range list {
		names = append(names, p.Name)
	}
	return names
}

// assertRealRpmPackages is the core regression guard for the rpm queryformat
// delimiter. The bug that triggered the revert (#7963 reverting #7818) made
// the runtime path parse zero packages because rpm did not emit the assumed
// delimiter byte. These assertions only pass against genuine rpm output.
func assertRealRpmPackages(t *testing.T, path string, list []rpmPackage) {
	t.Helper()
	// A real rpm distro has hundreds of packages; the delimiter break produced
	// an empty list. 50 is a generous floor that still catches that failure.
	assert.Greaterf(t, len(list), 50, "%s path returned too few packages (delimiter/parse regression?)", path)

	for _, p := range list {
		assert.NotEmptyf(t, p.Name, "%s path: package with empty name", path)
		assert.NotEmptyf(t, p.Version, "%s path: %q has empty version", path, p.Name)
		assert.Equalf(t, "rpm", p.Format, "%s path: %q has unexpected format %q", path, p.Name, p.Format)
	}

	// bash is present on every rpm base image and carries a normal arch, so it
	// proves field splitting worked end to end (gpg-pubkey, by contrast, has an
	// empty arch and would not).
	bash, ok := hasPackage(list, "bash")
	require.Truef(t, ok, "%s path: base package bash missing", path)
	assert.NotEmptyf(t, bash.Arch, "%s path: bash has empty arch", path)
}

// TestRpmPackages exercises both rpm package code paths against real rpm
// images and cross-checks them. It deliberately requires docker rather than
// skipping, because the whole point is to catch divergence between our
// assumptions and what rpm actually emits.
func TestRpmPackages(t *testing.T) {
	once.Do(setup)
	requireDocker(t)

	for _, image := range rpmTestImages {
		t.Run(image, func(t *testing.T) {
			// Runtime path: a running container routes to ContainerConnection,
			// which runs the real `rpm -qa --queryformat '<queryFormat()>'` and
			// parses the output with RPM_REGEX. This is the path the __ -> RS
			// delimiter change broke and the reason mock fixtures could not.
			id := dockerRunDetached(t, image)
			runtime := queryRpmPackages(t, "docker", id)
			assertRealRpmPackages(t, "runtime", runtime)

			// Static path: the same image as an image reference routes to a
			// snapshot connection, which reads the rpm database with the rpmdb
			// library and never touches queryFormat/RPM_REGEX.
			static := queryRpmPackages(t, "docker", image)
			assertRealRpmPackages(t, "static", static)

			// Both code paths enumerate the same installed rpm database, so the
			// set of package names must match. This pins the runtime and static
			// paths to each other and to real rpm, so neither can silently
			// drift again.
			assert.ElementsMatchf(t, packageNames(static), packageNames(runtime),
				"runtime and static package sets diverge for %s", image)
		})
	}
}
