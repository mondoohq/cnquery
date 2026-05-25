// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package windows

import (
	"bytes"
	"encoding/binary"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseScheduledTask_Defrag(t *testing.T) {
	f, err := os.Open("./testdata/scheduledtask_defrag.xml")
	require.NoError(t, err)
	defer f.Close()

	task, err := ParseScheduledTask(f)
	require.NoError(t, err)

	assert.Equal(t, `\Microsoft\Windows\Defrag\ScheduledDefrag`, task.URI)
	assert.Equal(t, "Microsoft Corporation", task.Author)
	assert.Equal(t, "This task optimizes local storage drives.", task.Description)
	assert.Equal(t, "$(@%SystemRoot%\\system32\\defragsvc.dll,-801)", task.Source)
	require.NotNil(t, task.Date)
	assert.Equal(t, 2018, task.Date.Year())

	assert.Equal(t, "S-1-5-18", task.RunAsUser)
	assert.Equal(t, "LeastPrivilege", task.RunLevel)

	assert.True(t, task.Enabled)
	assert.False(t, task.Hidden)
	assert.True(t, task.AllowStartOnDemand)
	assert.True(t, task.DisallowStartIfOnBatteries)
	assert.True(t, task.StopIfGoingOnBatteries)
	assert.Equal(t, "IgnoreNew", task.MultipleInstancesPolicy)
	assert.Equal(t, "P3D", task.ExecutionTimeLimit)
	assert.Equal(t, 7, task.Priority)

	require.Len(t, task.Triggers, 2)
	timeTrigger := task.Triggers[1]
	calendarTrigger := task.Triggers[0]
	// Note: collection order follows xmlTriggers field ordering (Calendar before Time).
	if timeTrigger["type"] == "CalendarTrigger" {
		timeTrigger, calendarTrigger = calendarTrigger, timeTrigger
	}
	assert.Equal(t, "TimeTrigger", timeTrigger["type"])
	assert.Equal(t, true, timeTrigger["enabled"])
	assert.Equal(t, "2004-08-01T00:00:00", timeTrigger["startBoundary"])
	rep, ok := timeTrigger["repetition"].(map[string]any)
	require.True(t, ok, "expected repetition map")
	assert.Equal(t, "PT1H", rep["interval"])
	assert.Equal(t, "P1D", rep["duration"])
	assert.Equal(t, false, rep["stopAtDurationEnd"])

	assert.Equal(t, "CalendarTrigger", calendarTrigger["type"])
	assert.Equal(t, "Weekly", calendarTrigger["scheduleType"])
	assert.Equal(t, int64(1), calendarTrigger["weeksInterval"])

	require.Len(t, task.Actions, 1)
	exec := task.Actions[0]
	assert.Equal(t, "Exec", exec["type"])
	assert.Equal(t, "%windir%\\system32\\defrag.exe", exec["command"])
	assert.Equal(t, "-c -h -o -$(Arg0)", exec["arguments"])
}

func TestParseScheduledTask_UTF16LE_BOM(t *testing.T) {
	utf8, err := os.ReadFile("./testdata/scheduledtask_defrag.xml")
	require.NoError(t, err)

	// Synthesize the UTF-16 LE BOM form Windows actually writes to disk.
	encoded := utf8ToUTF16LE(utf8)

	task, err := ParseScheduledTask(bytes.NewReader(encoded))
	require.NoError(t, err)
	assert.Equal(t, `\Microsoft\Windows\Defrag\ScheduledDefrag`, task.URI)
	assert.True(t, task.Enabled)
}

func TestParseScheduledTask_MissingOptionalSettings(t *testing.T) {
	xmlStr := `<?xml version="1.0" encoding="UTF-8"?>
<Task version="1.4" xmlns="http://schemas.microsoft.com/windows/2004/02/mit/task">
  <RegistrationInfo>
    <URI>\Custom\Task</URI>
  </RegistrationInfo>
  <Triggers/>
  <Principals>
    <Principal id="Author">
      <UserId>S-1-5-18</UserId>
    </Principal>
  </Principals>
  <Settings/>
  <Actions Context="Author">
    <Exec>
      <Command>cmd.exe</Command>
    </Exec>
  </Actions>
</Task>`

	task, err := ParseScheduledTask(bytes.NewReader([]byte(xmlStr)))
	require.NoError(t, err)
	// Defaults from Microsoft's documented Settings element values.
	assert.True(t, task.Enabled)
	assert.True(t, task.AllowStartOnDemand)
	assert.True(t, task.DisallowStartIfOnBatteries)
	assert.True(t, task.StopIfGoingOnBatteries)
	assert.False(t, task.Hidden)
	assert.Equal(t, 7, task.Priority)
}

func TestTaskPathFromFilePath(t *testing.T) {
	root := `C:\Windows\System32\Tasks`
	cases := []struct {
		full string
		want string
	}{
		{`C:\Windows\System32\Tasks\Microsoft\Windows\Defrag\ScheduledDefrag`, `\Microsoft\Windows\Defrag\ScheduledDefrag`},
		{`C:/Windows/System32/Tasks/MyTask`, `\MyTask`},
		{`C:\Windows\System32\Tasks\`, ``},
		{`D:\elsewhere\file`, ``},
	}
	for _, c := range cases {
		assert.Equal(t, c.want, TaskPathFromFilePath(root, c.full), c.full)
	}
}

func TestTaskLeafName(t *testing.T) {
	assert.Equal(t, "ScheduledDefrag", TaskLeafName(`\Microsoft\Windows\Defrag\ScheduledDefrag`))
	assert.Equal(t, "MyTask", TaskLeafName(`\MyTask`))
	assert.Equal(t, "", TaskLeafName(""))
}

func utf8ToUTF16LE(in []byte) []byte {
	out := bytes.NewBuffer(nil)
	out.Write([]byte{0xFF, 0xFE})
	for _, r := range string(in) {
		// only handle BMP characters in test fixtures
		var b [2]byte
		binary.LittleEndian.PutUint16(b[:], uint16(r))
		out.Write(b[:])
	}
	return out.Bytes()
}
