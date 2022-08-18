package macos

import (
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.io/mondoo/motor/providers/mock"
	os_provider "go.mondoo.io/mondoo/motor/providers/os"
)

func TestSystemSetup(t *testing.T) {
	mock, err := mock.NewFromTomlFile("./testdata/systemsetup.toml")
	require.NoError(t, err)

	so := SystemSetupCmdOutput{}
	assert.Equal(t, "8/4/2021", so.ParseDate(mustRunCmd(mock, "systemsetup -getdate")))
	assert.Equal(t, "20:22:54", so.ParseTime(mustRunCmd(mock, "systemsetup -gettime")))
	assert.Equal(t, "Europe/Berlin", so.ParseTimeZone(mustRunCmd(mock, "systemsetup -gettimezone")))
	assert.Equal(t, "time.euro.apple.com", so.ParseNetworkTimeServer(mustRunCmd(mock, "systemsetup -getnetworktimeserver")))
	assert.Equal(t, "On", so.ParseUsingNetworktTime(mustRunCmd(mock, "systemsetup -getusingnetworktime")))
	assert.Equal(t, []string{"Computer sleeps after 1 minutes", "Display sleeps after 10 minutes", "Disk sleeps after 10 minutes"}, so.ParseSleep(mustRunCmd(mock, "systemsetup -getsleep")))
	assert.Equal(t, "after 10 minutes", so.ParseDisplaySleep(mustRunCmd(mock, "systemsetup -getdisplaysleep")))
	assert.Equal(t, "after 10 minutes", so.ParseHardDiskSleep(mustRunCmd(mock, "systemsetup -getharddisksleep")))
	assert.Equal(t, "Not supported on this machine", so.ParseWakeOnModem(mustRunCmd(mock, "systemsetup -getwakeonmodem")))
	assert.Equal(t, "On", so.ParseWakeOnNetwork(mustRunCmd(mock, "systemsetup -getwakeonnetworkaccess")))
	assert.Equal(t, "Not supported on this machine", so.ParseRestartPowerFailure(mustRunCmd(mock, "systemsetup -getrestartpowerfailure")))
	assert.Equal(t, "On", so.ParseRestartFreeze(mustRunCmd(mock, "systemsetup -getrestartfreeze")))
	assert.Equal(t, "On", so.ParseAllowPowerButtonToSleep(mustRunCmd(mock, "systemsetup -getallowpowerbuttontosleepcomputer")))
	assert.Equal(t, "Off", so.ParseRemoteLogin(mustRunCmd(mock, "systemsetup -getremotelogin")))
	assert.Equal(t, "Off", so.ParseRemoteAppleEvents(mustRunCmd(mock, "systemsetup -getremoteappleevents")))
	assert.Equal(t, "spacerocket", so.ParseComputerName(mustRunCmd(mock, "systemsetup -getcomputername")))
	assert.Equal(t, "spacerocket", so.ParseLocalSubnetname(mustRunCmd(mock, "systemsetup -getlocalsubnetname")))
	assert.Equal(t, "0 seconds", so.ParseWaitForStartupAfterPowerFailure(mustRunCmd(mock, "systemsetup -getwaitforstartupafterpowerfailure")))
	assert.Equal(t, "No", so.ParseDisableKeyboardWhenEnclosureLockIsEngaged(mustRunCmd(mock, "systemsetup -getdisablekeyboardwhenenclosurelockisengaged")))
}

func mustRunCmd(t os_provider.OperatingSystemProvider, command string) string {
	cmd, err := t.RunCommand(command)
	if err != nil {
		panic(err)
	}
	data, err := io.ReadAll(cmd.Stdout)
	if err != nil {
		panic(err)
	}
	return string(data)
}
