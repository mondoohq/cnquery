// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"strings"

	"go.mondoo.com/cnquery/v11/providers-sdk/v1/util/convert"
	"go.mondoo.com/cnquery/v11/providers/os/connection/shared"
	"go.mondoo.com/cnquery/v11/providers/os/resources/macos"
	"howett.net/plist"
)

func (m *mqlMacos) userPreferences() (map[string]interface{}, error) {
	conn := m.MqlRuntime.Connection.(shared.Connection)
	preferences, err := macos.NewPreferences(conn).UserPreferences()
	if err != nil {
		return nil, err
	}

	return convert.JsonToDict(preferences)
}

func (m *mqlMacos) userHostPreferences() (map[string]interface{}, error) {
	conn := m.MqlRuntime.Connection.(shared.Connection)

	preferences, err := macos.NewPreferences(conn).UserHostPreferences()
	if err != nil {
		return nil, err
	}

	return convert.JsonToDict(preferences)
}

func (m *mqlMacos) globalAccountPolicies() (map[string]interface{}, error) {
	conn := m.MqlRuntime.Connection.(shared.Connection)

	cmd, err := conn.RunCommand("pwpolicy -getaccountpolicies")
	if err != nil {
		return nil, err
	}

	data, err := io.ReadAll(cmd.Stdout)
	if err != nil {
		return nil, err
	}

	return Decode(bytes.NewReader(data))
}

// GetPreferences returns the time machine preferences
//
// NOTE: this cannot be implemented via:
// parse.plist('/Library/Preferences/com.apple.TimeMachine.plist').params['AutoBackup'] == 1
// since the binary is missing the Full Disk Access (FDA), therefore even applications with
// sudo permissions cannot access the file. Instead we need to call
// defaults read /Library/Preferences/com.apple.TimeMachine.plist which has FDA
// see https://developer.apple.com/forums/thread/108348
func (m *mqlMacosTimemachine) preferences() (map[string]interface{}, error) {
	conn := m.MqlRuntime.Connection.(shared.Connection)

	cmd, err := conn.RunCommand("defaults read /Library/Preferences/com.apple.TimeMachine.plist")
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	scanner := bufio.NewScanner(cmd.Stdout)
	for scanner.Scan() {
		line := scanner.Text()
		// skip the BackupAlias since they are not parsable when returned by the `defaults` command
		if strings.HasPrefix(strings.TrimSpace(line), "BackupAlias") {
			continue
		}
		buf.WriteString(line)
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return Decode(bytes.NewReader(buf.Bytes()))
}

func (m *mqlMacosSystemsetup) runCmd(command string) (string, error) {
	conn := m.MqlRuntime.Connection.(shared.Connection)

	cmd, err := conn.RunCommand(command)
	if err != nil {
		return "", err
	}

	data, err := io.ReadAll(cmd.Stdout)
	if err != nil {
		return "", err
	}

	// NOTE: systemsetup returns exit 0 even if it does not have enough permissions
	// Therefore we need to handle this case here
	if strings.TrimSpace(string(data)) == "You need administrator access to run this tool... exiting!" {
		return "", errors.New("macos.systemsetup needs elevated permissions")
	}

	return string(data), nil
}

func (m *mqlMacosSystemsetup) date() (string, error) {
	data, err := m.runCmd("systemsetup -getdate")
	return macos.SystemSetupCmdOutput{}.ParseDate(data), err
}

func (m *mqlMacosSystemsetup) time() (string, error) {
	data, err := m.runCmd("systemsetup -gettime")
	return macos.SystemSetupCmdOutput{}.ParseTime(data), err
}

func (m *mqlMacosSystemsetup) timeZone() (string, error) {
	data, err := m.runCmd("systemsetup -gettimezone")
	return macos.SystemSetupCmdOutput{}.ParseTimeZone(data), err
}

func (m *mqlMacosSystemsetup) usingNetworkTime() (string, error) {
	data, err := m.runCmd("systemsetup -getusingnetworktime")
	return macos.SystemSetupCmdOutput{}.ParseUsingNetworktTime(data), err
}

func (m *mqlMacosSystemsetup) networkTimeServer() (string, error) {
	data, err := m.runCmd("systemsetup -getnetworktimeserver")
	return macos.SystemSetupCmdOutput{}.ParseNetworkTimeServer(data), err
}

func (m *mqlMacosSystemsetup) sleep() ([]interface{}, error) {
	data, err := m.runCmd("systemsetup -getsleep")
	return convert.SliceAnyToInterface(macos.SystemSetupCmdOutput{}.ParseSleep(data)), err
}

func (m *mqlMacosSystemsetup) displaySleep() (string, error) {
	data, err := m.runCmd("systemsetup -getdisplaysleep")
	return macos.SystemSetupCmdOutput{}.ParseDisplaySleep(data), err
}

func (m *mqlMacosSystemsetup) harddiskSleep() (string, error) {
	data, err := m.runCmd("systemsetup -getdisplaysleep")
	return macos.SystemSetupCmdOutput{}.ParseHardDiskSleep(data), err
}

func (m *mqlMacosSystemsetup) wakeOnModem() (string, error) {
	data, err := m.runCmd("systemsetup -getwakeonmodem")
	return macos.SystemSetupCmdOutput{}.ParseWakeOnModem(data), err
}

func (m *mqlMacosSystemsetup) wakeOnNetworkAccess() (string, error) {
	data, err := m.runCmd("systemsetup -getwakeonnetworkaccess")
	return macos.SystemSetupCmdOutput{}.ParseWakeOnNetwork(data), err
}

func (m *mqlMacosSystemsetup) restartPowerFailure() (string, error) {
	data, err := m.runCmd("systemsetup -getrestartpowerfailure")
	return macos.SystemSetupCmdOutput{}.ParseRestartPowerFailure(data), err
}

func (m *mqlMacosSystemsetup) restartFreeze() (string, error) {
	data, err := m.runCmd("systemsetup -getrestartfreeze")
	return macos.SystemSetupCmdOutput{}.ParseRestartFreeze(data), err
}

func (m *mqlMacosSystemsetup) allowPowerButtonToSleepComputer() (string, error) {
	data, err := m.runCmd("systemsetup -getallowpowerbuttontosleepcomputer")
	return macos.SystemSetupCmdOutput{}.ParseAllowPowerButtonToSleep(data), err
}

func (m *mqlMacosSystemsetup) remoteLogin() (string, error) {
	data, err := m.runCmd("systemsetup -getremotelogin")
	return macos.SystemSetupCmdOutput{}.ParseRemoteLogin(data), err
}

func (m *mqlMacosSystemsetup) remoteAppleEvents() (string, error) {
	data, err := m.runCmd("systemsetup -getremoteappleevents")
	return macos.SystemSetupCmdOutput{}.ParseRemoteAppleEvents(data), err
}

func (m *mqlMacosSystemsetup) computerName() (string, error) {
	data, err := m.runCmd("systemsetup -getcomputername")
	return macos.SystemSetupCmdOutput{}.ParseComputerName(data), err
}

func (m *mqlMacosSystemsetup) localSubnetName() (string, error) {
	data, err := m.runCmd("systemsetup -getlocalsubnetname")
	return macos.SystemSetupCmdOutput{}.ParseLocalSubnetname(data), err
}

func (m *mqlMacosSystemsetup) startupDisk() (string, error) {
	data, err := m.runCmd("systemsetup -getstartupdisk")
	return data, err
}

func (m *mqlMacosSystemsetup) waitForStartupAfterPowerFailure() (string, error) {
	data, err := m.runCmd("systemsetup -getwaitforstartupafterpowerfailure")
	return macos.SystemSetupCmdOutput{}.ParseWaitForStartupAfterPowerFailure(data), err
}

func (m *mqlMacosSystemsetup) disableKeyboardWhenEnclosureLockIsEngaged() (string, error) {
	data, err := m.runCmd("systemsetup -getdisablekeyboardwhenenclosurelockisengaged")
	return macos.SystemSetupCmdOutput{}.ParseDisableKeyboardWhenEnclosureLockIsEngaged(data), err
}

func Decode(r io.ReadSeeker) (map[string]interface{}, error) {
	var data map[string]interface{}
	decoder := plist.NewDecoder(r)
	err := decoder.Decode(&data)
	if err != nil {
		return nil, err
	}

	// NOTE: we need to do the extra conversion here to make sure we use supported
	// values by our dict structure: string, float64, int64
	// plist also uses uint64 heavily which we do not support
	// TODO: we really do not want to use the poor-man's json conversion version
	jsondata, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	var dataJson map[string]interface{}
	err = json.Unmarshal(jsondata, &dataJson)
	if err != nil {
		return nil, err
	}

	return dataJson, nil
}
