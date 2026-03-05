// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

type Runner interface {
	Run() error
	ExitCode() int
	Stdout() []byte
	Stderr() []byte
	Json(v any) error
}

type cliTestRunner struct {
	cmd    *exec.Cmd
	binary string
	args   []string
	stdout bytes.Buffer
	stderr bytes.Buffer
}

func NewCliTestRunner(binary string, args ...string) *cliTestRunner {
	c := &cliTestRunner{
		binary: binary,
		args:   args,
	}
	return c
}

func (c *cliTestRunner) Run() error {
	c.cmd = exec.Command(c.binary, c.args...)
	c.cmd.Env = BuildEnv()
	c.cmd.Stdout = &c.stdout
	c.cmd.Stderr = &c.stderr

	if err := c.cmd.Start(); err != nil {
		return fmt.Errorf("error starting command: %s", err)
	}

	// Wait for the command to finish
	if err := c.cmd.Wait(); err != nil {
		if _, ok := err.(*exec.ExitError); ok {
			// The program has exited with an exit code != 0, but for testing purposes we don't want to fail the test
			return nil
		}
		return fmt.Errorf("command finished with error: %v", err)
	}

	return nil
}

func (c *cliTestRunner) ExitCode() int {
	return c.cmd.ProcessState.ExitCode()
}

func (c *cliTestRunner) Stdout() []byte {
	return c.stdout.Bytes()
}

func (c *cliTestRunner) Stderr() []byte {
	return c.stderr.Bytes()
}

func (c *cliTestRunner) Json(v any) error {
	return json.Unmarshal(c.Stdout(), v)
}

// BuildEnv returns the current environment with GOCOVERDIR removed.
// When tests run with -cover, Go sets GOCOVERDIR which is inherited by child
// processes. Programs spawned via "go run" or "go build" are not built with
// -cover, so they fail at exit when trying to write coverage data.
func BuildEnv() []string {
	var env []string
	for _, e := range os.Environ() {
		if !strings.HasPrefix(e, "GOCOVERDIR=") {
			env = append(env, e)
		}
	}
	return env
}
