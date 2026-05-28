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
// against. We deliberately use the *minimal* variants (~15-20 MB compressed)
// to keep the pull cheap; they still ship the rpm CLI and a real rpm database,
// which is all both code paths need.
var rpmTestImages = []string{
	"redhat/ubi9-minimal",
	"almalinux:9-minimal",
}

type rpmPackage struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Arch    string `json:"arch"`
	Format  string `json:"format"`
	Vendor  string `json:"vendor"`
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

// requireRpmBinary asserts the running container actually has the rpm CLI so
// the runtime code path (`rpm -qa --queryformat`) is genuinely exercised. Some
// minimal/micro images ship rpm-libs without /usr/bin/rpm; on those mql would
// silently fall back to the static path, making this test pass while covering
// nothing. Fail loudly instead so a bad image choice is obvious.
func requireRpmBinary(t *testing.T, containerID string) {
	t.Helper()
	out, err := exec.Command("docker", "exec", containerID, "sh", "-c", "command -v rpm").CombinedOutput()
	require.NoErrorf(t, err, "rpm CLI missing in container, runtime path would fall back to static: %s", string(out))
}

// queryRpmPackages runs mql against the target and returns the parsed package
// list. It projects the fields explicitly with a block: a bare `packages`
// query only serializes the resource's @defaults (name, version) under -j, so
// arch and format would come back empty.
func queryRpmPackages(t *testing.T, target ...string) []rpmPackage {
	t.Helper()
	args := make([]string, 0, 1+len(target)+3)
	args = append(args, "run")
	args = append(args, target...)
	args = append(args, "-c", "packages.list { name version arch format }", "-j")
	r := test.NewCliTestRunner("./mql", args...)
	require.NoError(t, r.Run())
	require.Equalf(t, 0, r.ExitCode(), "mql exited non-zero; stderr: %s", string(r.Stderr()))

	// The -j output is an array with one object per scanned asset; that object
	// has a single key (the queried block) whose value is the package list.
	// Parse key-agnostically so we don't depend on how mql labels the block.
	var assets []map[string][]rpmPackage
	if err := r.Json(&assets); err != nil {
		t.Fatalf("parsing mql json failed: %v\n--- stdout ---\n%s\n--- stderr ---\n%s",
			err, string(r.Stdout()), string(r.Stderr()))
	}
	require.Lenf(t, assets, 1, "expected exactly one asset result; stdout: %s", string(r.Stdout()))
	for _, list := range assets[0] {
		return list
	}
	t.Fatalf("no package list in mql output; stdout: %s", string(r.Stdout()))
	return nil
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
	// The delimiter break produced an empty list. Even a minimal image has
	// dozens of packages, so 20 is a safe floor that still catches that failure.
	assert.Greaterf(t, len(list), 20, "%s path returned too few packages (delimiter/parse regression?)", path)

	for _, p := range list {
		assert.NotEmptyf(t, p.Name, "%s path: package with empty name", path)
		assert.NotEmptyf(t, p.Version, "%s path: %q has empty version", path, p.Name)
	}

	// glibc is present on every rpm image (including minimal variants) and
	// carries a normal arch and format, so it proves field splitting worked end
	// to end. We assert these on glibc rather than every package because
	// gpg-pubkey (a GPG-key pseudo-package, not real software) has neither.
	glibc, ok := hasPackage(list, "glibc")
	require.Truef(t, ok, "%s path: base package glibc missing", path)
	assert.NotEmptyf(t, glibc.Arch, "%s path: glibc has empty arch", path)
	assert.Equalf(t, "rpm", glibc.Format, "%s path: glibc has unexpected format %q", path, glibc.Format)
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
			requireRpmBinary(t, id)
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
