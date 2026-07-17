// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package providers

import (
	"os"
	"os/exec"
	"strconv"
	"sync"
	"syscall"

	"github.com/hashicorp/go-plugin"
)

// processTracker pairs the plugin subprocess command with the go-plugin
// client supervising it, so crash diagnostics can read the subprocess's exit
// disposition (exit code vs. signal, peak RSS) after it dies.
//
// go-plugin's exit-watcher goroutine calls cmd.Wait() — which populates
// exec.Cmd.ProcessState — and only afterwards marks Client.Exited() under the
// client's mutex. Reading ProcessState is therefore only safe once Exited()
// reports true for the client that owns this exact cmd; exitState enforces
// that pairing. The tracker is updated in lock-step by the coordinator's
// connect function, including across RestartableProvider restarts.
type processTracker struct {
	lock   sync.Mutex
	client *plugin.Client
	cmd    *exec.Cmd
}

func (t *processTracker) track(client *plugin.Client, cmd *exec.Cmd) {
	t.lock.Lock()
	defer t.lock.Unlock()
	t.client = client
	t.cmd = cmd
}

// exitState returns the exit state of the tracked plugin subprocess, or nil
// while it is still running (or if nothing is tracked). A non-nil result is
// safe to read: Exited() == true means go-plugin's Wait on this cmd returned.
func (t *processTracker) exitState() *os.ProcessState {
	t.lock.Lock()
	defer t.lock.Unlock()
	if t.client == nil || t.cmd == nil {
		return nil
	}
	if !t.client.Exited() {
		return nil
	}
	return t.cmd.ProcessState
}

// formatExitStatus renders a process's exit disposition for the crash
// diagnostics meta block: "code:<n>" for a regular exit, "signal:<SIG>" when
// the process was killed by a signal (e.g. "signal:SIGKILL", the typical OOM
// killer fingerprint), or "unknown" if the state carries neither.
func formatExitStatus(ps *os.ProcessState) string {
	if ps == nil {
		return "unknown"
	}
	if ws, ok := ps.Sys().(syscall.WaitStatus); ok && ws.Signaled() {
		return "signal:" + signalName(ws.Signal())
	}
	if code := ps.ExitCode(); code >= 0 {
		return "code:" + strconv.Itoa(code)
	}
	return "unknown"
}
