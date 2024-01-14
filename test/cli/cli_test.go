// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package cli

import (
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnquery/v10/apps/cnquery/cmd"
	"os"
	"sync"
	"testing"

	cmdtest "github.com/google/go-cmdtest"
)

var once sync.Once
var cnqueryCmd *cobra.Command

func setup() {
	var err error
	cnqueryCmd, err = cmd.BuildRootCmd()
	if err != nil {
		panic(err)
	}
}

func TestMain(m *testing.M) {
	ret := m.Run()
	os.Exit(ret)
}

func TestCompare(t *testing.T) {
	once.Do(setup)
	ts, err := cmdtest.Read("testdata")
	require.NoError(t, err)

	ts.DisableLogging = true
	ts.Commands["cnquery"] = cmdtest.InProcessProgram("cnquery", func() int {
		err := cnqueryCmd.Execute()
		if err != nil {
			return 1
		}
		return 0
	})
	ts.Run(t, false) // set to true to update test files
}
