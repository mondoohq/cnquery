// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"go.mondoo.com/cnquery/v11/llx"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/util/convert"
	"go.mondoo.com/cnquery/v11/providers/ansible/connection"
	"go.mondoo.com/cnquery/v11/providers/ansible/play"
	"go.mondoo.com/cnquery/v11/types"
	"strconv"
)

func (r *mqlAnsible) id() (string, error) {
	return "ansible", nil
}

func (r *mqlAnsible) plays() ([]interface{}, error) {
	conn := r.MqlRuntime.Connection.(*connection.AnsibleConnection)
	playbook := conn.Playbook()

	var plays []interface{}
	for _, play := range playbook {

		p, err := newMqlAnsiblePlay(r.MqlRuntime, play)
		if err != nil {
			return nil, err
		}
		plays = append(plays, p)
	}
	return plays, nil
}

func newMqlAnsiblePlay(runtime *plugin.Runtime, play *play.Play) (*mqlAnsiblePlay, error) {
	varsDict, err := convert.JsonToDict(play.Vars)
	if err != nil {
		return nil, err
	}

	res, err := CreateResource(runtime, "ansible.play", map[string]*llx.RawData{
		"name":              llx.StringData(play.Name),
		"hosts":             llx.DictData(play.Hosts),
		"remoteUser":        llx.StringData(play.RemoteUser),
		"become":            llx.BoolData(play.Become),
		"becomeUser":        llx.StringData(play.BecomeUser),
		"becomeMethod":      llx.StringData(play.BecomeMethod),
		"becomeFlags":       llx.StringData(play.BecomeFlags),
		"strategy":          llx.StringData(play.Strategy),
		"maxFailPercentage": llx.IntData(play.MaxFailPercentage),
		"ignoreUnreachable": llx.BoolData(play.IgnoreUnreachable),
		"anyErrorsFatal":    llx.BoolData(play.AnyErrorsFatal),
		"vars":              llx.DictData(varsDict),
		"roles":             llx.ArrayData(convert.SliceAnyToInterface(play.Roles), types.String),
	})
	if err != nil {
		return nil, err
	}
	mqlPlay := res.(*mqlAnsiblePlay)
	mqlPlay.play = play
	return mqlPlay, nil
}

type mqlAnsiblePlayInternal struct {
	play *play.Play
}

func (r *mqlAnsiblePlay) id() (string, error) {
	return r.Name.Data, nil
}

func newMqlAnsibleHandler(runtime *plugin.Runtime, id string, task *play.Handler) (*mqlAnsibleHandler, error) {
	dict, err := convert.JsonToDict(task.Action)
	if err != nil {
		return nil, err
	}

	res, err := CreateResource(runtime, "ansible.handler", map[string]*llx.RawData{
		"__id":   llx.StringData(id),
		"name":   llx.StringData(task.Name),
		"action": llx.DictData(dict),
	})
	if err != nil {
		return nil, err
	}
	mqlHandler := res.(*mqlAnsibleHandler)
	return mqlHandler, nil
}

func newMqlAnsibleHandlers(runtime *plugin.Runtime, idPrefix string, handler []*play.Handler) ([]interface{}, error) {
	var mqlTasks []interface{}
	for i, t := range handler {
		id := idPrefix + strconv.Itoa(i)
		if t.Name != "" {
			id = t.Name
		}

		t, err := newMqlAnsibleHandler(runtime, id, t)
		if err != nil {
			return nil, err
		}
		mqlTasks = append(mqlTasks, t)
	}
	return mqlTasks, nil
}

func (r *mqlAnsiblePlay) handlers() ([]interface{}, error) {
	return newMqlAnsibleHandlers(r.MqlRuntime, "handlers", r.play.Handlers)
}

func newMqlAnsibleTask(runtime *plugin.Runtime, id string, task *play.Task) (*mqlAnsibleTask, error) {
	actionDict, err := convert.JsonToDict(task.Action)
	if err != nil {
		return nil, err
	}

	varDict, err := convert.JsonToDict(task.Vars)
	if err != nil {
		return nil, err
	}

	res, err := CreateResource(runtime, "ansible.task", map[string]*llx.RawData{
		"__id":            llx.StringData(id),
		"name":            llx.StringData(task.Name),
		"action":          llx.DictData(actionDict),
		"vars":            llx.DictData(varDict),
		"register":        llx.StringData(task.Register),
		"when":            llx.StringData(task.When),
		"failedWhen":      llx.StringData(task.FailedWhen),
		"changedWhen":     llx.StringData(task.ChangedWhen),
		"notify":          llx.ArrayData(convert.SliceAnyToInterface(task.Notify), types.String),
		"importPlaybook":  llx.StringData(task.ImportPlaybook),
		"includePlaybook": llx.StringData(task.IncludePlaybook),
		"importTasks":     llx.StringData(task.ImportTasks),
		"includeTasks":    llx.StringData(task.IncludeTasks),
	})
	if err != nil {
		return nil, err
	}
	mqlTask := res.(*mqlAnsibleTask)
	mqlTask.task = task
	return mqlTask, nil
}

func newMqlAnsibleTasks(runtime *plugin.Runtime, idPrefix string, tasks []*play.Task) ([]interface{}, error) {
	var mqlTasks []interface{}
	for i, t := range tasks {
		id := idPrefix + strconv.Itoa(i)
		if t.Name != "" {
			id = t.Name
		}

		t, err := newMqlAnsibleTask(runtime, id, t)
		if err != nil {
			return nil, err
		}
		mqlTasks = append(mqlTasks, t)
	}
	return mqlTasks, nil
}

type mqlAnsibleTaskInternal struct {
	task *play.Task
}

func (r *mqlAnsiblePlay) tasks() ([]interface{}, error) {
	return newMqlAnsibleTasks(r.MqlRuntime, "tasks", r.play.Tasks)
}

func (r *mqlAnsibleTask) block() ([]interface{}, error) {
	return newMqlAnsibleTasks(r.MqlRuntime, "block", r.task.Block)
}

func (r *mqlAnsibleTask) rescue() ([]interface{}, error) {
	return newMqlAnsibleTasks(r.MqlRuntime, "rescue", r.task.Rescue)
}
