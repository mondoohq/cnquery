// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package windows

import (
	"encoding/json"
	"io"
	"strings"
)

// SCHEDULED_TASKS projects every registered Task Scheduler task into a stable
// JSON shape. Enum-typed properties (State, LogonType, RunLevel,
// MultipleInstances, Compatibility) are stringified via "$(...)" so they
// serialize as their names rather than raw integers, and the polymorphic
// trigger objects are flattened into a single union shape keyed by their CIM
// class name.
const SCHEDULED_TASKS = `
$ErrorActionPreference = 'Stop'
Get-ScheduledTask | ForEach-Object {
  $t = $_
  $p = $t.Principal
  $s = $t.Settings
  [PSCustomObject]@{
    Name               = $t.TaskName
    Path               = $t.TaskPath
    URI                = $t.URI
    State              = "$($t.State)"
    Description        = $t.Description
    Author             = $t.Author
    Documentation      = $t.Documentation
    SecurityDescriptor = $t.SecurityDescriptor
    Source             = $t.Source
    Date               = $t.Date
    Principal          = if ($p) {
      [PSCustomObject]@{
        UserId      = $p.UserId
        GroupId     = $p.GroupId
        DisplayName = $p.DisplayName
        LogonType   = "$($p.LogonType)"
        RunLevel    = "$($p.RunLevel)"
      }
    } else { $null }
    Actions = @($t.Actions | ForEach-Object {
      [PSCustomObject]@{
        Execute          = $_.Execute
        Arguments        = $_.Arguments
        WorkingDirectory = $_.WorkingDirectory
      }
    })
    Triggers = @($t.Triggers | ForEach-Object {
      $tr = $_
      [PSCustomObject]@{
        Type                        = $tr.CimClass.CimClassName
        Enabled                     = $tr.Enabled
        StartBoundary               = $tr.StartBoundary
        EndBoundary                 = $tr.EndBoundary
        ExecutionTimeLimit          = $tr.ExecutionTimeLimit
        RepetitionInterval          = $tr.Repetition.Interval
        RepetitionDuration          = $tr.Repetition.Duration
        RepetitionStopAtDurationEnd = $tr.Repetition.StopAtDurationEnd
        Delay                       = $tr.Delay
        RandomDelay                 = $tr.RandomDelay
        DaysInterval                = $tr.DaysInterval
        WeeksInterval               = $tr.WeeksInterval
        DaysOfWeek                  = $tr.DaysOfWeek
        UserId                      = $tr.UserId
      }
    })
    Settings = if ($s) {
      [PSCustomObject]@{
        Enabled                         = $s.Enabled
        Hidden                          = $s.Hidden
        AllowDemandStart                = $s.AllowDemandStart
        AllowHardTerminate              = $s.AllowHardTerminate
        StartWhenAvailable              = $s.StartWhenAvailable
        RunOnlyIfNetworkAvailable       = $s.RunOnlyIfNetworkAvailable
        RunOnlyIfIdle                   = $s.RunOnlyIfIdle
        WakeToRun                       = $s.WakeToRun
        DisallowStartIfOnBatteries      = $s.DisallowStartIfOnBatteries
        StopIfGoingOnBatteries          = $s.StopIfGoingOnBatteries
        DisallowStartOnRemoteAppSession = $s.DisallowStartOnRemoteAppSession
        RestartCount                    = $s.RestartCount
        RestartInterval                 = $s.RestartInterval
        ExecutionTimeLimit              = $s.ExecutionTimeLimit
        MultipleInstances               = "$($s.MultipleInstances)"
        Priority                        = $s.Priority
        DeleteExpiredTaskAfter          = $s.DeleteExpiredTaskAfter
        Compatibility                   = "$($s.Compatibility)"
        IdleDuration                    = $s.IdleSettings.IdleDuration
        IdleWaitTimeout                 = $s.IdleSettings.WaitTimeout
        IdleStopOnIdleEnd               = $s.IdleSettings.StopOnIdleEnd
        IdleRestartOnIdle               = $s.IdleSettings.RestartOnIdle
        NetworkId                       = $s.NetworkSettings.Id
        NetworkName                     = $s.NetworkSettings.Name
      }
    } else { $null }
  }
} | ConvertTo-Json -Depth 5`

// WindowsScheduledTask is the projected JSON shape of a single Task Scheduler
// task. Pointer fields distinguish "absent" from a zero value so the resource
// layer can surface MQL null for properties the task does not set.
type WindowsScheduledTask struct {
	Name               string                         `json:"Name"`
	Path               string                         `json:"Path"`
	URI                string                         `json:"URI"`
	State              string                         `json:"State"`
	Description        string                         `json:"Description"`
	Author             string                         `json:"Author"`
	Documentation      string                         `json:"Documentation"`
	SecurityDescriptor string                         `json:"SecurityDescriptor"`
	Source             string                         `json:"Source"`
	Date               string                         `json:"Date"`
	Principal          *WindowsScheduledTaskPrincipal `json:"Principal"`
	Actions            []WindowsScheduledTaskAction   `json:"Actions"`
	Triggers           []WindowsScheduledTaskTrigger  `json:"Triggers"`
	Settings           *WindowsScheduledTaskSettings  `json:"Settings"`
}

type WindowsScheduledTaskPrincipal struct {
	UserId      string `json:"UserId"`
	GroupId     string `json:"GroupId"`
	DisplayName string `json:"DisplayName"`
	LogonType   string `json:"LogonType"`
	RunLevel    string `json:"RunLevel"`
}

type WindowsScheduledTaskAction struct {
	Execute          string `json:"Execute"`
	Arguments        string `json:"Arguments"`
	WorkingDirectory string `json:"WorkingDirectory"`
}

type WindowsScheduledTaskTrigger struct {
	Type                        string `json:"Type"`
	Enabled                     *bool  `json:"Enabled"`
	StartBoundary               string `json:"StartBoundary"`
	EndBoundary                 string `json:"EndBoundary"`
	ExecutionTimeLimit          string `json:"ExecutionTimeLimit"`
	RepetitionInterval          string `json:"RepetitionInterval"`
	RepetitionDuration          string `json:"RepetitionDuration"`
	RepetitionStopAtDurationEnd *bool  `json:"RepetitionStopAtDurationEnd"`
	Delay                       string `json:"Delay"`
	RandomDelay                 string `json:"RandomDelay"`
	DaysInterval                *int64 `json:"DaysInterval"`
	WeeksInterval               *int64 `json:"WeeksInterval"`
	DaysOfWeek                  *int64 `json:"DaysOfWeek"`
	UserId                      string `json:"UserId"`
}

type WindowsScheduledTaskSettings struct {
	Enabled                         *bool  `json:"Enabled"`
	Hidden                          *bool  `json:"Hidden"`
	AllowDemandStart                *bool  `json:"AllowDemandStart"`
	AllowHardTerminate              *bool  `json:"AllowHardTerminate"`
	StartWhenAvailable              *bool  `json:"StartWhenAvailable"`
	RunOnlyIfNetworkAvailable       *bool  `json:"RunOnlyIfNetworkAvailable"`
	RunOnlyIfIdle                   *bool  `json:"RunOnlyIfIdle"`
	WakeToRun                       *bool  `json:"WakeToRun"`
	DisallowStartIfOnBatteries      *bool  `json:"DisallowStartIfOnBatteries"`
	StopIfGoingOnBatteries          *bool  `json:"StopIfGoingOnBatteries"`
	DisallowStartOnRemoteAppSession *bool  `json:"DisallowStartOnRemoteAppSession"`
	RestartCount                    *int64 `json:"RestartCount"`
	RestartInterval                 string `json:"RestartInterval"`
	ExecutionTimeLimit              string `json:"ExecutionTimeLimit"`
	MultipleInstances               string `json:"MultipleInstances"`
	Priority                        *int64 `json:"Priority"`
	DeleteExpiredTaskAfter          string `json:"DeleteExpiredTaskAfter"`
	Compatibility                   string `json:"Compatibility"`
	IdleDuration                    string `json:"IdleDuration"`
	IdleWaitTimeout                 string `json:"IdleWaitTimeout"`
	IdleStopOnIdleEnd               *bool  `json:"IdleStopOnIdleEnd"`
	IdleRestartOnIdle               *bool  `json:"IdleRestartOnIdle"`
	NetworkId                       string `json:"NetworkId"`
	NetworkName                     string `json:"NetworkName"`
}

// ParseWindowsScheduledTasks decodes the JSON emitted by SCHEDULED_TASKS,
// handling the PowerShell quirk where a single task is emitted as a bare
// object rather than a one-element array.
func ParseWindowsScheduledTasks(input io.Reader) ([]WindowsScheduledTask, error) {
	return streamDecodeJSONArray[WindowsScheduledTask](input)
}

// WindowsScheduledTaskInfo is the projected JSON shape of Get-ScheduledTaskInfo
// for a single task.
type WindowsScheduledTaskInfo struct {
	LastRunTime        string `json:"LastRunTime"`
	NextRunTime        string `json:"NextRunTime"`
	LastTaskResult     int64  `json:"LastTaskResult"`
	NumberOfMissedRuns int64  `json:"NumberOfMissedRuns"`
}

// ScheduledTaskInfoScript builds a per-task Get-ScheduledTaskInfo query. Run-time
// values aren't part of Get-ScheduledTask, so they are fetched lazily, one task
// at a time, on first access of a run-time field. DateTime values are formatted
// round-trip ("o") for stable parsing.
func ScheduledTaskInfoScript(taskPath, taskName string) string {
	var b strings.Builder
	b.WriteString("Get-ScheduledTaskInfo -TaskPath '")
	b.WriteString(escapePSSingleQuote(taskPath))
	b.WriteString("' -TaskName '")
	b.WriteString(escapePSSingleQuote(taskName))
	b.WriteString("' | ForEach-Object { [PSCustomObject]@{")
	b.WriteString(" LastRunTime = if ($_.LastRunTime) { $_.LastRunTime.ToString('o') } else { $null };")
	b.WriteString(" NextRunTime = if ($_.NextRunTime) { $_.NextRunTime.ToString('o') } else { $null };")
	b.WriteString(" LastTaskResult = $_.LastTaskResult;")
	b.WriteString(" NumberOfMissedRuns = $_.NumberOfMissedRuns")
	b.WriteString(" } } | ConvertTo-Json")
	return b.String()
}

// escapePSSingleQuote escapes a value for inclusion inside a PowerShell
// single-quoted string literal, where the only metacharacter is the single
// quote itself (doubled to escape).
func escapePSSingleQuote(s string) string {
	return strings.ReplaceAll(s, "'", "''")
}

// ParseWindowsScheduledTaskInfo decodes the JSON emitted by
// ScheduledTaskInfoScript. An empty payload (task without run-time info)
// yields a zero-valued struct.
func ParseWindowsScheduledTaskInfo(input io.Reader) (*WindowsScheduledTaskInfo, error) {
	data, err := io.ReadAll(input)
	if err != nil {
		return nil, err
	}
	if len(strings.TrimSpace(string(data))) == 0 {
		return &WindowsScheduledTaskInfo{}, nil
	}
	info := &WindowsScheduledTaskInfo{}
	if err := json.Unmarshal(data, info); err != nil {
		return nil, err
	}
	return info, nil
}
