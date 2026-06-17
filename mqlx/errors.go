// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package mqlx

// CompileError wraps a compilation failure together with the query source.
// Unwrap exposes the underlying compiler error, so typed errors such as
// mqlc.ErrIdentifierNotFound remain accessible via errors.As.
type CompileError struct {
	Source string
	Err    error
}

func (e *CompileError) Error() string {
	return "failed to compile MQL: " + e.Err.Error()
}

func (e *CompileError) Unwrap() error {
	return e.Err
}
