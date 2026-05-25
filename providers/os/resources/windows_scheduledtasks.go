// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"errors"
	"os"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/spf13/afero"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/os/connection/shared"
	"go.mondoo.com/mql/v13/providers/os/resources/windows"
)

// windowsTasksRoot is the directory the Task Scheduler service writes task
// XML definitions to. mql's filesystem abstraction normalises Windows paths
// using backslashes.
const windowsTasksRoot = `C:\Windows\System32\Tasks`

func (t *mqlWindowsScheduledTask) id() (string, error) {
	if t.Path.Data == "" {
		return "", errors.New("scheduled task path is required")
	}
	return "windows.scheduledTask/" + t.Path.Data, nil
}

func initWindowsScheduledTask(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 1 {
		return args, nil, nil
	}

	pathRaw, ok := args["path"]
	if !ok || pathRaw == nil {
		return args, nil, nil
	}
	pathStr, ok := pathRaw.Value.(string)
	if !ok {
		return args, nil, nil
	}

	obj, err := NewResource(runtime, "windows", nil)
	if err != nil {
		return nil, nil, err
	}
	tasks := obj.(*mqlWindows).GetScheduledTasks()
	if tasks.Error != nil {
		return nil, nil, tasks.Error
	}

	for _, raw := range tasks.Data {
		task := raw.(*mqlWindowsScheduledTask)
		if task.Path.Data == pathStr {
			return nil, task, nil
		}
	}
	return nil, nil, errors.New("could not find scheduled task " + pathStr)
}

func (w *mqlWindows) scheduledTasks() ([]any, error) {
	conn, ok := w.MqlRuntime.Connection.(shared.Connection)
	if !ok {
		return nil, errors.New("wrong connection type")
	}
	fs := conn.FileSystem()
	if fs == nil {
		return nil, errors.New("filesystem not available")
	}
	afs := &afero.Afero{Fs: fs}

	exists, err := afs.DirExists(windowsTasksRoot)
	if err != nil || !exists {
		// On non-Windows systems, or when the Tasks directory simply isn't
		// present, return an empty collection instead of erroring — the same
		// way other Windows-specific resources behave.
		return []any{}, nil
	}

	var result []any
	walkErr := afero.Walk(afs, windowsTasksRoot, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			log.Debug().Err(err).Str("path", filePath).Msg("skipping scheduled task path")
			return nil
		}
		if info.IsDir() {
			return nil
		}
		resource, parseErr := w.parseScheduledTaskFile(afs, filePath)
		if parseErr != nil {
			log.Debug().Err(parseErr).Str("path", filePath).Msg("skipping unparseable scheduled task")
			return nil
		}
		if resource != nil {
			result = append(result, resource)
		}
		return nil
	})
	if walkErr != nil {
		return nil, walkErr
	}
	return result, nil
}

func (w *mqlWindows) parseScheduledTaskFile(afs *afero.Afero, filePath string) (plugin.Resource, error) {
	f, err := afs.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	parsed, err := windows.ParseScheduledTask(f)
	if err != nil {
		return nil, err
	}

	// The URI in RegistrationInfo is authoritative; fall back to deriving from
	// the on-disk path so synthetic / legacy tasks without URI still resolve.
	taskPath := strings.TrimSpace(parsed.URI)
	if taskPath == "" {
		taskPath = windows.TaskPathFromFilePath(windowsTasksRoot, filePath)
	}
	if taskPath == "" {
		log.Debug().Str("path", filePath).Msg("skipping scheduled task with empty URI and no derivable path")
		return nil, nil
	}

	fileRes, err := CreateResource(w.MqlRuntime, "file", map[string]*llx.RawData{
		"path": llx.StringData(filePath),
	})
	if err != nil {
		return nil, err
	}

	return CreateResource(w.MqlRuntime, "windows.scheduledTask", map[string]*llx.RawData{
		"path":                       llx.StringData(taskPath),
		"name":                       llx.StringData(windows.TaskLeafName(taskPath)),
		"author":                     llx.StringData(parsed.Author),
		"description":                llx.StringData(parsed.Description),
		"source":                     llx.StringData(parsed.Source),
		"date":                       llx.TimeDataPtr(parsed.Date),
		"runAsUser":                  llx.StringData(parsed.RunAsUser),
		"runLevel":                   llx.StringData(parsed.RunLevel),
		"logonType":                  llx.StringData(parsed.LogonType),
		"groupId":                    llx.StringData(parsed.GroupID),
		"enabled":                    llx.BoolData(parsed.Enabled),
		"hidden":                     llx.BoolData(parsed.Hidden),
		"allowStartOnDemand":         llx.BoolData(parsed.AllowStartOnDemand),
		"runOnlyIfNetworkAvailable":  llx.BoolData(parsed.RunOnlyIfNetworkAvailable),
		"stopIfGoingOnBatteries":     llx.BoolData(parsed.StopIfGoingOnBatteries),
		"disallowStartIfOnBatteries": llx.BoolData(parsed.DisallowStartIfOnBatteries),
		"multipleInstancesPolicy":    llx.StringData(parsed.MultipleInstancesPolicy),
		"executionTimeLimit":         llx.StringData(parsed.ExecutionTimeLimit),
		"priority":                   llx.IntData(int64(parsed.Priority)),
		"triggers":                   llx.ArrayData(dictArray(parsed.Triggers), "dict"),
		"actions":                    llx.ArrayData(dictArray(parsed.Actions), "dict"),
		"file":                       llx.ResourceData(fileRes, "file"),
	})
}

func dictArray(items []map[string]any) []any {
	out := make([]any, len(items))
	for i, m := range items {
		out[i] = anyMap(m)
	}
	return out
}
