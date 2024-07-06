// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package multierr

import (
	"strconv"
	"strings"
)

// withMessage and methods are taken from https://github.com/pkg/errors
// under BSD-2-Clause license

type withMessage struct {
	cause error
	msg   string
}

func (w withMessage) Error() string { return w.msg + ": " + w.cause.Error() }
func (w withMessage) Cause() error  { return w.cause }
func (w withMessage) Unwrap() error { return w.cause }

func Wrap(err error, message string) error {
	if err == nil {
		return nil
	}
	return withMessage{
		cause: err,
		msg:   message,
	}
}

type Errors struct {
	Errors []error
}

func (m *Errors) Add(err ...error) {
	for i := range err {
		if err[i] != nil {
			m.Errors = append(m.Errors, err[i])
		}
	}
}

func (m *Errors) Error() string {
	var res strings.Builder

	n := strconv.Itoa(len(m.Errors))
	if n == "1" {
		res.WriteString("1 error occurred:\n")
	} else {
		res.WriteString(n + " errors occurred:\n")
	}

	for i := range m.Errors {
		res.WriteString("\t* ")
		res.WriteString(m.Errors[i].Error())
		res.WriteByte('\n')
	}
	return res.String()
}

func (m Errors) Deduplicate() error {
	if len(m.Errors) == 0 {
		return nil
	}

	track := map[string]error{}
	for i := range m.Errors {
		e := m.Errors[i]
		track[e.Error()] = e
	}

	res := make([]error, len(track))
	i := 0
	for _, v := range track {
		res[i] = v
		i++
	}
	return &Errors{Errors: res}
}

func (m *Errors) IsEmpty() bool {
	if m == nil {
		return true
	}
	return len(m.Errors) == 0
}
