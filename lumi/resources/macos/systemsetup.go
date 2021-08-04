package macos

import "strings"

type SystemSetupCmdOutput struct{}

func (s SystemSetupCmdOutput) ParseDate(in string) string {
	return strings.TrimSpace(strings.TrimPrefix(in, "Time:"))
}

func (s SystemSetupCmdOutput) ParseTime(in string) string {
	return strings.TrimSpace(strings.TrimPrefix(in, "Time:"))
}

func (s SystemSetupCmdOutput) ParseTimeZone(in string) string {
	return strings.TrimSpace(strings.TrimPrefix(in, "Time Zone:"))
}

func (s SystemSetupCmdOutput) ParseUsingNetworktTime(in string) string {
	return strings.TrimSpace(strings.TrimPrefix(in, "Network Time:"))
}

func (s SystemSetupCmdOutput) ParseNetworkTimeServer(in string) string {
	return strings.TrimSpace(strings.TrimPrefix(in, "Network Time Server:"))
}

func (s SystemSetupCmdOutput) ParseSleep(in string) []string {
	entries := strings.Split(strings.TrimSpace(in), "\n")
	for i := range entries {
		entries[i] = strings.TrimSpace(strings.TrimPrefix(entries[i], "Sleep:"))
	}
	return entries
}

func (s SystemSetupCmdOutput) ParseDisplaySleep(in string) string {
	return strings.TrimSpace(strings.TrimPrefix(in, "Display Sleep:"))
}

func (s SystemSetupCmdOutput) ParseHardDiskSleep(in string) string {
	return strings.TrimSpace(strings.TrimPrefix(in, "Hard Disk Sleep:"))
}

func (s SystemSetupCmdOutput) ParseWakeOnModem(in string) string {
	data := strings.TrimSpace(strings.TrimPrefix(in, "Wake On Modem:"))
	data = strings.TrimSuffix(data, ".")
	return data
}

func (s SystemSetupCmdOutput) ParseWakeOnNetwork(in string) string {
	return strings.TrimSpace(strings.TrimPrefix(in, "Wake On Network Access:"))
}

func (s SystemSetupCmdOutput) ParseRestartPowerFailure(in string) string {
	data := strings.TrimSpace(strings.TrimPrefix(in, "Restart After Power Failure:"))
	data = strings.TrimSuffix(data, ".")
	return data
}

func (s SystemSetupCmdOutput) ParseRestartFreeze(in string) string {
	return strings.TrimSpace(strings.TrimPrefix(in, "Restart After Freeze:"))
}

func (s SystemSetupCmdOutput) ParseAllowPowerButtonToSleep(in string) string {
	return strings.TrimSpace(strings.TrimPrefix(in, "getAllowPowerButtonToSleepComputer:"))
}

func (s SystemSetupCmdOutput) ParseRemoteLogin(in string) string {
	return strings.TrimSpace(strings.TrimPrefix(in, "Remote Login:"))
}

func (s SystemSetupCmdOutput) ParseRemoteAppleEvents(in string) string {
	return strings.TrimSpace(strings.TrimPrefix(in, "Remote Apple Events:"))
}

func (s SystemSetupCmdOutput) ParseComputerName(in string) string {
	return strings.TrimSpace(strings.TrimPrefix(in, "Computer Name:"))
}

func (s SystemSetupCmdOutput) ParseLocalSubnetname(in string) string {
	return strings.TrimSpace(strings.TrimPrefix(in, "Local Subnet Name:"))
}

func (s SystemSetupCmdOutput) ParseWaitForStartupAfterPowerFailure(in string) string {
	return strings.TrimSpace(strings.TrimPrefix(in, "getwaitforstartupafterpowerfailure:"))
}

func (s SystemSetupCmdOutput) ParseDisableKeyboardWhenEnclosureLockIsEngaged(in string) string {
	return strings.TrimSpace(strings.TrimPrefix(in, "getdisablekeyboardwhenenclosurelockisengaged:"))
}
