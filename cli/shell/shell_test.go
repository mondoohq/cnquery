package shell_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.mondoo.com/cnquery/cli/shell"
	"go.mondoo.com/cnquery/providers"
	"go.mondoo.com/cnquery/providers/mock"
	"go.mondoo.com/cnquery/providers/os/provider"
)

func localShell() *shell.Shell {
	runtime := providers.Coordinator.NewRuntime()
	schema, err := os.ReadFile("../../providers/os/dist/os.resources.json")
	if err != nil {
		panic(err.Error())
	}

	runtime.SchemaData = providers.MustLoadSchema("os", schema)
	runtime.Connection, err = provider.Init().Connect(provider.LocalAssetReq)
	if err != nil {
		panic(err.Error())
	}

	res, err := shell.New(runtime)
	if err != nil {
		panic(err.Error())
	}

	return res
}

func mockShell(filename string, opts ...shell.ShellOption) *shell.Shell {
	path, _ := filepath.Abs(filename)
	runtime, err := mock.NewFromTomlFile(path)
	if err != nil {
		panic(err.Error())
	}

	res, err := shell.New(runtime, opts...)
	if err != nil {
		panic(err.Error())
	}

	return res
}

func TestShell_RunOnce(t *testing.T) {
	shell := localShell()
	assert.NotPanics(t, func() {
		shell.RunOnce("mondoo.build")
	}, "should not panic on partial queries")

	assert.NotPanics(t, func() {
		shell.RunOnce("mondoo { build version }")
	}, "should not panic on partial queries")

	assert.NotPanics(t, func() {
		shell.RunOnce("mondoo { _.version }")
	}, "should not panic on partial queries")
}

func TestShell_Help(t *testing.T) {
	shell := localShell()
	assert.NotPanics(t, func() {
		shell.ExecCmd("help")
	}, "should not panic on help command")

	assert.NotPanics(t, func() {
		shell.ExecCmd("help platform")
	}, "should not panic on help subcommand")
}

func TestShell_Centos8(t *testing.T) {
	shell := mockShell("../../mql/testdata/centos8.toml")
	assert.NotPanics(t, func() {
		shell.RunOnce("platform { title name release arch }")
	}, "should not panic on partial queries")
}
