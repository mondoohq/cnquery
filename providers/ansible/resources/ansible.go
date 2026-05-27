// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"strconv"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
	"go.mondoo.com/mql/v13/providers/ansible/connection"
	"go.mondoo.com/mql/v13/providers/ansible/play"
	"go.mondoo.com/mql/v13/types"
)

func (r *mqlAnsible) id() (string, error) {
	return "ansible", nil
}

func (r *mqlAnsible) plays() ([]any, error) {
	conn := r.MqlRuntime.Connection.(*connection.AnsibleConnection)
	playbook := conn.Playbook()

	plays := make([]any, 0, len(playbook))
	for i, p := range playbook {
		mqlPlay, err := newMqlAnsiblePlay(r.MqlRuntime, "play["+strconv.Itoa(i)+"]", p)
		if err != nil {
			return nil, err
		}
		plays = append(plays, mqlPlay)
	}
	return plays, nil
}

func newMqlAnsiblePlay(runtime *plugin.Runtime, id string, p *play.Play) (*mqlAnsiblePlay, error) {
	res, err := CreateResource(runtime, "ansible.play", map[string]*llx.RawData{
		"__id":              llx.StringData(id),
		"name":              llx.StringData(p.Name),
		"hosts":             llx.DictData(p.Hosts),
		"remoteUser":        llx.StringData(p.RemoteUser),
		"become":            llx.BoolData(p.Become),
		"becomeUser":        llx.StringData(p.BecomeUser),
		"becomeMethod":      llx.StringData(p.BecomeMethod),
		"becomeFlags":       llx.StringData(p.BecomeFlags),
		"serial":            llx.DictData(p.Serial),
		"strategy":          llx.StringData(p.Strategy),
		"maxFailPercentage": llx.IntData(p.MaxFailPercentage),
		"ignoreUnreachable": llx.BoolData(p.IgnoreUnreachable),
		"anyErrorsFatal":    llx.BoolData(p.AnyErrorsFatal),
		"gatherFacts":       llx.StringData(p.GatherFacts),
		"vars":              llx.MapData(p.Vars, types.Dict),
		"tags":              llx.ArrayData(convert.SliceAnyToInterface(p.Tags), types.String),
		"roles":             llx.ArrayData(convert.SliceAnyToInterface(p.Roles), types.String),
	})
	if err != nil {
		return nil, err
	}
	mqlPlay := res.(*mqlAnsiblePlay)
	mqlPlay.play = p
	return mqlPlay, nil
}

type mqlAnsiblePlayInternal struct {
	play *play.Play
}

func newMqlAnsibleHandler(runtime *plugin.Runtime, id string, handler *play.Handler) (*mqlAnsibleHandler, error) {
	res, err := CreateResource(runtime, "ansible.handler", map[string]*llx.RawData{
		"__id":   llx.StringData(id),
		"name":   llx.StringData(handler.Name),
		"action": llx.DictData(toAny(handler.Action)),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlAnsibleHandler), nil
}

func newMqlAnsibleHandlers(runtime *plugin.Runtime, parentID string, handlers []*play.Handler) ([]any, error) {
	mqlHandlers := make([]any, 0, len(handlers))
	for i, h := range handlers {
		id := parentID + "/handlers[" + strconv.Itoa(i) + "]"
		mqlHandler, err := newMqlAnsibleHandler(runtime, id, h)
		if err != nil {
			return nil, err
		}
		mqlHandlers = append(mqlHandlers, mqlHandler)
	}
	return mqlHandlers, nil
}

func (r *mqlAnsiblePlay) handlers() ([]any, error) {
	return newMqlAnsibleHandlers(r.MqlRuntime, r.MqlID(), r.play.Handlers)
}

func newMqlAnsibleTask(runtime *plugin.Runtime, id string, task *play.Task) (*mqlAnsibleTask, error) {
	res, err := CreateResource(runtime, "ansible.task", map[string]*llx.RawData{
		"__id":            llx.StringData(id),
		"name":            llx.StringData(task.Name),
		"action":          llx.DictData(toAny(task.Action)),
		"vars":            llx.MapData(task.Vars, types.Dict),
		"tags":            llx.ArrayData(convert.SliceAnyToInterface(task.Tags), types.String),
		"register":        llx.StringData(task.Register),
		"become":          llx.BoolData(task.Become),
		"becomeUser":      llx.StringData(task.BecomeUser),
		"becomeMethod":    llx.StringData(task.BecomeMethod),
		"becomeFlags":     llx.StringData(task.BecomeFlags),
		"delegateTo":      llx.StringData(task.DelegateTo),
		"environment":     llx.MapData(task.Environment, types.Dict),
		"noLog":           llx.DictData(task.NoLog),
		"ignoreErrors":    llx.DictData(task.IgnoreErrors),
		"runOnce":         llx.DictData(task.RunOnce),
		"when":            llx.StringData(task.When),
		"failedWhen":      llx.StringData(task.FailedWhen),
		"changedWhen":     llx.StringData(task.ChangedWhen),
		"notify":          llx.ArrayData(convert.SliceAnyToInterface(task.Notify), types.String),
		"loop":            llx.DictData(task.Loop),
		"loopControl":     llx.DictData(toAny(task.LoopControl)),
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

func newMqlAnsibleTasks(runtime *plugin.Runtime, parentID, group string, tasks []*play.Task) ([]any, error) {
	mqlTasks := make([]any, 0, len(tasks))
	for i, t := range tasks {
		id := parentID + "/" + group + "[" + strconv.Itoa(i) + "]"
		mqlTask, err := newMqlAnsibleTask(runtime, id, t)
		if err != nil {
			return nil, err
		}
		mqlTasks = append(mqlTasks, mqlTask)
	}
	return mqlTasks, nil
}

type mqlAnsibleTaskInternal struct {
	task *play.Task
}

func (r *mqlAnsiblePlay) preTasks() ([]any, error) {
	return newMqlAnsibleTasks(r.MqlRuntime, r.MqlID(), "preTasks", r.play.PreTasks)
}

func (r *mqlAnsiblePlay) tasks() ([]any, error) {
	return newMqlAnsibleTasks(r.MqlRuntime, r.MqlID(), "tasks", r.play.Tasks)
}

func (r *mqlAnsiblePlay) postTasks() ([]any, error) {
	return newMqlAnsibleTasks(r.MqlRuntime, r.MqlID(), "postTasks", r.play.PostTasks)
}

func (r *mqlAnsibleTask) block() ([]any, error) {
	return newMqlAnsibleTasks(r.MqlRuntime, r.MqlID(), "block", r.task.Block)
}

func (r *mqlAnsibleTask) rescue() ([]any, error) {
	return newMqlAnsibleTasks(r.MqlRuntime, r.MqlID(), "rescue", r.task.Rescue)
}

func (r *mqlAnsibleTask) always() ([]any, error) {
	return newMqlAnsibleTasks(r.MqlRuntime, r.MqlID(), "always", r.task.Always)
}

// toAny converts a typed map to an untyped `any` for llx.DictData. Passing the
// typed map directly would box the map type into the interface, which the dict
// runtime cannot iterate the same way as map[string]any.
func toAny(m map[string]any) any {
	if m == nil {
		return nil
	}
	return m
}
