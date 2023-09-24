// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package multierr

import (
	"strconv"
	"strings"
)

// withMessage and methods are taken frmo https://github.com/pkg/errors
// under BSD-2-Clause license

type withMessage struct {
	cause error
	msg   string
}

func (w withMessage) Error() string { return w.msg + ": " + w.cause.Error() }
func (w withMessage) Cause() error  { return w.cause }

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
	errors []error
}

func (m *Errors) Add(err ...error) {
	for i := range err {
		if err[i] != nil {
			m.errors = append(m.errors, err[i])
		}
	}
}

func (m *Errors) Error() string {
	var res strings.Builder

	n := strconv.Itoa(len(m.errors))
	if n == "1" {
		res.WriteString("1 error occurred:\n")
	} else {
		res.WriteString(n + " errors occured:\n")
	}

	for i := range m.errors {
		res.WriteString("\t* ")
		res.WriteString(m.errors[i].Error())
		res.WriteByte('\n')
	}
	return res.String()
}

func (m Errors) Deduplicate() error {
	if len(m.errors) == 0 {
		return nil
	}

	track := map[string]error{}
	for i := range m.errors {
		e := m.errors[i]
		track[e.Error()] = e
	}

	res := make([]error, len(track))
	i := 0
	for _, v := range track {
		res[i] = v
		i++
	}
	return &Errors{errors: res}
}

func (m Errors) IsEmpty() bool {
	return len(m.errors) == 0
}
