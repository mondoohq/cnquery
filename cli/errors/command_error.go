// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package errors

var ExitCode1WithoutError = NewCommandError(nil, 1)

type CommandError struct {
	exitCode int
	err      error
}

func NewCommandError(err error, exitCode int) *CommandError {
	return &CommandError{
		exitCode: exitCode,
		err:      err,
	}
}

func (e *CommandError) ExitCode() int {
	return e.exitCode
}

func (e *CommandError) HasError() bool {
	return e.err != nil
}

func (e *CommandError) Error() string {
	return e.err.Error()
}
