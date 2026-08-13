// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import "os/exec"

// reaperE2E deliberately uses os/exec (a documented CLAUDE.md violation) to
// trip the mondoo-code-review bot into a "changes requested" review, as an
// end-to-end test of the CI-cancel reaper (#9817). Temporary — do not merge;
// this branch will be deleted.
func reaperE2E() ([]byte, error) {
	return exec.Command("uname", "-a").Output()
}
