package shell

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.mondoo.com/cnquery/motor"
	"go.mondoo.com/cnquery/motor/providers/local"
	"go.mondoo.com/cnquery/motor/providers/mock"
)

func localShell() *Shell {
	transport, err := local.New()
	if err != nil {
		panic(err.Error())
	}

	motor, err := motor.New(transport)
	if err != nil {
		panic(err.Error())
	}

	res, err := New(motor)
	if err != nil {
		panic(err.Error())
	}

	return res
}

func mockShell(filename string, opts ...ShellOption) *Shell {
	filepath, _ := filepath.Abs(filename)
	provider, err := mock.NewFromTomlFile(filepath)
	if err != nil {
		panic(err.Error())
	}

	motor, err := motor.New(provider)
	if err != nil {
		panic(err.Error())
	}

	res, err := New(motor, opts...)
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
		shell.execCmd("help")
	}, "should not panic on help command")

	assert.NotPanics(t, func() {
		shell.execCmd("help platform")
	}, "should not panic on help subcommand")
}

func TestShell_Centos8(t *testing.T) {
	shell := mockShell("../../mql/testdata/centos8.toml")
	assert.NotPanics(t, func() {
		shell.RunOnce("platform { title name release arch }")
	}, "should not panic on partial queries")
}
