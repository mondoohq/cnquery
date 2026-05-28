// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"errors"
	"slices"
	"strconv"
	"strings"
	"sync"

	"github.com/moby/buildkit/frontend/dockerfile/instructions"
	"github.com/moby/buildkit/frontend/dockerfile/linter"
	"github.com/moby/buildkit/frontend/dockerfile/parser"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/os/connection/docker"
	"go.mondoo.com/mql/v13/providers/os/connection/local"
	"go.mondoo.com/mql/v13/providers/os/connection/ssh"
	"go.mondoo.com/mql/v13/types"
	"go.mondoo.com/mql/v13/utils/multierr"
)

func initDockerFile(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	// the dockerfile connection is a wrapper around the local one
	// NOTE: we might have to extend this in the future if we start supporting docker files from other connections (e.g. tar)
	_, isDockerConn := runtime.Connection.(*docker.DockerfileConnection)
	_, isSshConn := runtime.Connection.(*ssh.Connection)
	_, isLocalConn := runtime.Connection.(*local.LocalConnection)
	// if neither, we set the file to nil.
	if !isDockerConn && !isSshConn && !isLocalConn {
		return args, nil, nil
	}

	// if users supply a file, we don't have to run any fancy initialization,
	// since most of this function deals with trying to find the dockerfile
	if _, ok := args["file"]; ok {
		return args, nil, nil
	}

	var path string

	// init from path
	if rawPath, ok := args["path"]; ok {
		delete(args, "path")
		path, ok = rawPath.Value.(string)
		if !ok {
			return nil, nil, errors.New("path must be supplied as a string")
		}
	} else if dfc, ok := runtime.Connection.(*docker.DockerfileConnection); ok {
		path = dfc.FileAbsSrc
	}

	// we assume the default name for the dockerfile if it was not provided
	if path == "" {
		path = "Dockerfile"
	}

	raw, err := CreateResource(runtime, "file", map[string]*llx.RawData{
		"path": llx.StringData(path),
	})
	if err != nil {
		return nil, nil, err
	}
	mqlFile, _ := raw.(*mqlFile)
	args["file"] = llx.ResourceData(mqlFile, "file")
	return args, nil, nil
}

type mqlDockerFileInternal struct {
	lock sync.Mutex
}

func (p *mqlDockerFile) id() (string, error) {
	if p.File.Data == nil {
		return "", errors.New("no file provided, can't determine ID for dockerfile")
	}
	return p.File.Data.id()
}

func (p *mqlDockerFile) file() (*mqlFile, error) {
	return nil, errors.New("missing underlying file, please specify a path of file")
}

func (p *mqlDockerFile) parse(file *mqlFile) error {
	p.lock.Lock()
	defer p.lock.Unlock()

	setError := func(err error) error {
		p.Instructions.Error = err
		p.Stages.Error = err
		p.Directives.Error = err
		p.MultiStage.Error = err
		p.HasSyntaxDirective.Error = err
		p.FinalStage.Error = err
		return err
	}

	content := file.GetContent()
	if content.Error != nil {
		return setError(content.Error)
	}

	directives := map[string]any{}
	dp := parser.DirectiveParser{}
	parsed, derr := dp.ParseAll([]byte(content.Data))
	if derr == nil {
		for _, d := range parsed {
			directives[d.Name] = d.Value
		}
	}
	p.Directives = plugin.TValue[map[string]any]{
		Data:  directives,
		State: plugin.StateIsSet,
	}

	reader := strings.NewReader(content.Data)
	ast, err := parser.Parse(reader)
	if err != nil {
		return setError(multierr.Wrap(err, "failed to parse dockerfile "+file.Path.Data))
	}

	if ast.AST != nil {
		instructions := make([]any, len(ast.AST.Children))
		for i := range ast.AST.Children {
			node := ast.AST.Children[i]
			instructions[i] = map[string]any{
				"original": node.Original,
			}
		}
		p.Instructions = plugin.TValue[any]{
			Data:  instructions,
			State: plugin.StateIsSet,
		}
	} else {
		p.Instructions = plugin.TValue[any]{
			Data:  []any{},
			State: plugin.StateIsSet,
		}
	}

	parsedStages, meta, err := instructions.Parse(ast.AST, linter.New(&linter.Config{}))
	if err != nil {
		return setError(multierr.Wrap(err, "failed to parse dockerfile instructions in "+file.Path.Data))
	}

	stages := make([]any, len(parsedStages))
	var stagesErr error
	for i := range parsedStages {
		stages[i], err = p.stage2resource(parsedStages[i], i == len(parsedStages)-1)
		if err != nil {
			stagesErr = multierr.Wrap(err, "failed to parse stage in dockerfile "+file.Path.Data)
			break
		}
	}
	p.Stages = plugin.TValue[[]any]{
		Data:  stages,
		Error: stagesErr,
		State: plugin.StateIsSet,
	}

	p.MultiStage = plugin.TValue[bool]{
		Data:  len(stages) > 1,
		Error: stagesErr,
		State: plugin.StateIsSet,
	}
	_, hasSyntax := directives["syntax"]
	p.HasSyntaxDirective = plugin.TValue[bool]{
		Data:  hasSyntax,
		State: plugin.StateIsSet,
	}
	if stagesErr == nil && len(stages) > 0 {
		p.FinalStage = plugin.TValue[*mqlDockerFileStage]{
			Data:  stages[len(stages)-1].(*mqlDockerFileStage),
			State: plugin.StateIsSet,
		}
	} else {
		p.FinalStage = plugin.TValue[*mqlDockerFileStage]{
			Error: stagesErr,
			State: plugin.StateIsSet | plugin.StateIsNull,
		}
	}

	// FIXME: add meta data
	_ = meta

	return nil
}

func (p *mqlDockerFile) stage2resource(stage instructions.Stage, isFinal bool) (*mqlDockerFileStage, error) {
	var image, tag, digest string
	rest := stage.BaseName
	if before, after, ok := strings.Cut(rest, "@"); ok {
		rest = before
		digest = after
	}
	if before, after, ok := strings.Cut(rest, ":"); ok {
		image = before
		tag = after
	} else {
		image = rest
	}

	stageID := p.locationID(stage.Location)

	rawFrom, err := CreateResource(p.MqlRuntime, ResourceDockerFileFrom, map[string]*llx.RawData{
		"__id":     llx.StringData(stageID),
		"platform": llx.StringData(stage.Platform),
		"image":    llx.StringData(image),
		"tag":      llx.StringData(tag),
		"digest":   llx.StringData(digest),
		"name":     llx.StringData(stage.Name),
	})
	if err != nil {
		return nil, err
	}

	var env []any
	var arg []any
	labels := map[string]string{}
	var expose []any
	var runs []any
	var copy []any
	var add []any
	var volumes []any
	var workdir []any
	var onbuild []any
	var unsupported []string
	var entrypointRaw *instructions.EntrypointCommand
	var cmdRaw *instructions.CmdCommand
	var userRaw *instructions.UserCommand
	var healthcheckRaw *instructions.HealthCheckCommand
	var shellRaw *instructions.ShellCommand
	var stopsignalRaw *instructions.StopSignalCommand
	for i := range stage.Commands {
		switch v := stage.Commands[i].(type) {
		case *instructions.EnvCommand:
			for _, kv := range v.Env {
				envResource, err := CreateResource(p.MqlRuntime, ResourceDockerFileEnv, map[string]*llx.RawData{
					"__id":  llx.StringData(p.locationID(v.Location())),
					"name":  llx.StringData(kv.Key),
					"value": llx.StringData(kv.Value),
				})
				if err != nil {
					return nil, err
				}
				env = append(env, envResource)
			}
		case *instructions.ArgCommand:
			for _, kv := range v.Args {
				argResource, err := CreateResource(p.MqlRuntime, ResourceDockerFileArg, map[string]*llx.RawData{
					"__id":    llx.StringData(p.locationID(v.Location())),
					"name":    llx.StringData(kv.Key),
					"default": llx.StringDataPtr(kv.Value),
				})
				if err != nil {
					return nil, err
				}
				arg = append(arg, argResource)
			}
		case *instructions.LabelCommand:
			for _, kv := range v.Labels {
				labels[kv.Key] = kv.Value
			}
		case *instructions.UserCommand:
			userRaw = v

		case *instructions.RunCommand:
			script := strings.Join(v.CmdLine, "\n")
			mounts, err := p.mountResources(v)
			if err != nil {
				return nil, err
			}
			mountsSecret, mountsSsh := mountTypeFlags(mounts)
			runResource, err := CreateResource(p.MqlRuntime, ResourceDockerFileRun, map[string]*llx.RawData{
				"__id":         llx.StringData(p.locationID(v.Location())),
				"script":       llx.StringData(script),
				"mounts":       llx.ArrayData(mounts, types.Resource(ResourceDockerFileRunMount)),
				"network":      llx.StringData(runFlagValue(v, "network")),
				"security":     llx.StringData(runFlagValue(v, "security")),
				"isShellForm":  llx.BoolData(v.PrependShell),
				"isExecForm":   llx.BoolData(!v.PrependShell),
				"mountsSecret": llx.BoolData(mountsSecret),
				"mountsSsh":    llx.BoolData(mountsSsh),
			})
			if err != nil {
				return nil, err
			}
			runs = append(runs, runResource)

		case *instructions.EntrypointCommand:
			entrypointRaw = v

		case *instructions.CmdCommand:
			cmdRaw = v

		case *instructions.CopyCommand:
			src := make([]any, len(v.SourcePaths))
			for i := range v.SourcePaths {
				src[i] = v.SourcePaths[i]
			}
			excludes := stringsToAny(v.ExcludePatterns)
			resource, err := CreateResource(p.MqlRuntime, ResourceDockerFileCopy, map[string]*llx.RawData{
				"__id":     llx.StringData(p.locationID(v.Location())),
				"src":      llx.ArrayData(src, types.String),
				"dst":      llx.StringData(v.DestPath),
				"chown":    llx.StringData(v.Chown),
				"chmod":    llx.StringData(v.Chmod),
				"from":     llx.StringData(v.From),
				"link":     llx.BoolData(v.Link),
				"excludes": llx.ArrayData(excludes, types.String),
				"parents":  llx.BoolData(v.Parents),
			})
			if err != nil {
				return nil, err
			}
			copy = append(copy, resource)

		case *instructions.AddCommand:
			src := make([]any, len(v.SourcePaths))
			for i := range v.SourcePaths {
				src[i] = v.SourcePaths[i]
			}
			excludes := stringsToAny(v.ExcludePatterns)
			resource, err := CreateResource(p.MqlRuntime, ResourceDockerFileAdd, map[string]*llx.RawData{
				"__id":     llx.StringData(p.locationID(v.Location())),
				"src":      llx.ArrayData(src, types.String),
				"dst":      llx.StringData(v.DestPath),
				"chown":    llx.StringData(v.Chown),
				"chmod":    llx.StringData(v.Chmod),
				"link":     llx.BoolData(v.Link),
				"checksum": llx.StringData(v.Checksum),
				"excludes": llx.ArrayData(excludes, types.String),
			})
			if err != nil {
				return nil, err
			}
			add = append(add, resource)

		case *instructions.ExposeCommand:
			for _, port := range v.Ports {
				arr := strings.Split(port, "/")
				var protocol string
				if len(arr) < 2 {
					protocol = "tcp"
				} else {
					protocol = arr[1]
				}
				portNum, _ := strconv.Atoi(arr[0])
				id := arr[0] + "/" + protocol

				resource, err := CreateResource(p.MqlRuntime, ResourceDockerFileExpose, map[string]*llx.RawData{
					"__id":     llx.StringData(id),
					"port":     llx.IntData(portNum),
					"protocol": llx.StringData(protocol),
				})
				if err != nil {
					return nil, err
				}
				expose = append(expose, resource)

			}

		case *instructions.HealthCheckCommand:
			healthcheckRaw = v

		case *instructions.VolumeCommand:
			for _, vol := range v.Volumes {
				resource, err := CreateResource(p.MqlRuntime, ResourceDockerFileVolume, map[string]*llx.RawData{
					"__id": llx.StringData(p.locationID(v.Location()) + ":" + vol),
					"path": llx.StringData(vol),
				})
				if err != nil {
					return nil, err
				}
				volumes = append(volumes, resource)
			}

		case *instructions.ShellCommand:
			shellRaw = v

		case *instructions.WorkdirCommand:
			resource, err := CreateResource(p.MqlRuntime, ResourceDockerFileWorkdir, map[string]*llx.RawData{
				"__id": llx.StringData(p.locationID(v.Location())),
				"path": llx.StringData(v.Path),
			})
			if err != nil {
				return nil, err
			}
			workdir = append(workdir, resource)

		case *instructions.StopSignalCommand:
			stopsignalRaw = v

		case *instructions.OnbuildCommand:
			resource, err := CreateResource(p.MqlRuntime, ResourceDockerFileOnbuild, map[string]*llx.RawData{
				"__id":       llx.StringData(p.locationID(v.Location())),
				"expression": llx.StringData(v.Expression),
			})
			if err != nil {
				return nil, err
			}
			onbuild = append(onbuild, resource)

		default:
			cmd := stage.Commands[i]
			unsupported = append(unsupported, cmd.Name())
		}
	}

	if len(unsupported) != 0 {
		slices.Sort(unsupported)
		log.Debug().Strs("commands", slices.Compact(unsupported)).Msg("unsupported dockerfile commands")
	}

	var userValue, groupValue string
	if userRaw != nil {
		parts := strings.SplitN(userRaw.User, ":", 2)
		if len(parts) > 0 && parts[0] != "" {
			userValue = parts[0]
		}
		if len(parts) > 1 && parts[1] != "" {
			groupValue = parts[1]
		}
	}

	args := map[string]*llx.RawData{
		"__id":           llx.StringData(stageID),
		"from":           llx.ResourceData(rawFrom, ResourceDockerFileFrom),
		"file":           llx.ResourceData(p, ResourceDockerFile),
		"env":            llx.ArrayData(env, types.Resource(ResourceDockerFileEnv)),
		"arg":            llx.ArrayData(arg, types.Resource(ResourceDockerFileArg)),
		"labels":         llx.MapData(llx.TMap2Raw(labels), types.String),
		"run":            llx.ArrayData(runs, types.Resource(ResourceDockerFileRun)),
		"add":            llx.ArrayData(add, types.Resource(ResourceDockerFileAdd)),
		"copy":           llx.ArrayData(copy, types.Resource(ResourceDockerFileCopy)),
		"expose":         llx.ArrayData(expose, types.Resource(ResourceDockerFileExpose)),
		"volumes":        llx.ArrayData(volumes, types.Resource(ResourceDockerFileVolume)),
		"workdir":        llx.ArrayData(workdir, types.Resource(ResourceDockerFileWorkdir)),
		"onbuild":        llx.ArrayData(onbuild, types.Resource(ResourceDockerFileOnbuild)),
		"runsAsRoot":     llx.BoolData(userRaw == nil || isRootUser(userValue)),
		"hasHealthcheck": llx.BoolData(healthcheckRaw != nil),
		"final":          llx.BoolData(isFinal),
	}

	if stopsignalRaw != nil {
		stopResource, err := CreateResource(p.MqlRuntime, ResourceDockerFileStopsignal, map[string]*llx.RawData{
			"__id":   llx.StringData(p.locationID(stopsignalRaw.Location())),
			"signal": llx.StringData(stopsignalRaw.Signal),
		})
		if err != nil {
			return nil, err
		}
		args["stopsignal"] = llx.ResourceData(stopResource, ResourceDockerFileStopsignal)
	} else {
		args["stopsignal"] = llx.NilData
	}

	if entrypointRaw != nil {
		script := strings.Join(entrypointRaw.CmdLine, "\n")
		runResource, err := CreateResource(p.MqlRuntime, ResourceDockerFileRun, map[string]*llx.RawData{
			"__id":         llx.StringData(p.locationID(entrypointRaw.Location())),
			"script":       llx.StringData(script),
			"mounts":       llx.ArrayData(nil, types.Resource(ResourceDockerFileRunMount)),
			"network":      llx.StringData(""),
			"security":     llx.StringData(""),
			"isShellForm":  llx.BoolData(entrypointRaw.PrependShell),
			"isExecForm":   llx.BoolData(!entrypointRaw.PrependShell),
			"mountsSecret": llx.BoolData(false),
			"mountsSsh":    llx.BoolData(false),
		})
		if err != nil {
			return nil, err
		}
		args["entrypoint"] = llx.ResourceData(runResource, ResourceDockerFileRun)
	} else {
		args["entrypoint"] = llx.NilData
	}

	if cmdRaw != nil {
		script := strings.Join(cmdRaw.CmdLine, "\n")
		cmdResource, err := CreateResource(p.MqlRuntime, ResourceDockerFileRun, map[string]*llx.RawData{
			"__id":         llx.StringData(p.locationID(cmdRaw.Location())),
			"script":       llx.StringData(script),
			"mounts":       llx.ArrayData(nil, types.Resource(ResourceDockerFileRunMount)),
			"network":      llx.StringData(""),
			"security":     llx.StringData(""),
			"isShellForm":  llx.BoolData(cmdRaw.PrependShell),
			"isExecForm":   llx.BoolData(!cmdRaw.PrependShell),
			"mountsSecret": llx.BoolData(false),
			"mountsSsh":    llx.BoolData(false),
		})
		if err != nil {
			return nil, err
		}
		args["cmd"] = llx.ResourceData(cmdResource, ResourceDockerFileRun)
	} else {
		args["cmd"] = llx.NilData
	}

	if userRaw != nil {
		userResource, err := CreateResource(p.MqlRuntime, ResourceDockerFileUser, map[string]*llx.RawData{
			"__id":   llx.StringData(p.locationID(userRaw.Location())),
			"user":   llx.StringData(userValue),
			"group":  llx.StringData(groupValue),
			"isRoot": llx.BoolData(isRootUser(userValue)),
		})
		if err != nil {
			return nil, err
		}
		args["user"] = llx.ResourceData(userResource, ResourceDockerFileUser)
	} else {
		args["user"] = llx.NilData
	}

	if healthcheckRaw != nil && healthcheckRaw.Health != nil {
		h := healthcheckRaw.Health
		isNone := len(h.Test) > 0 && h.Test[0] == "NONE"
		test := make([]any, len(h.Test))
		for i, t := range h.Test {
			test[i] = t
		}
		hcResource, err := CreateResource(p.MqlRuntime, ResourceDockerFileHealthcheck, map[string]*llx.RawData{
			"__id":          llx.StringData(p.locationID(healthcheckRaw.Location())),
			"test":          llx.ArrayData(test, types.String),
			"interval":      llx.IntData(int64(h.Interval)),
			"timeout":       llx.IntData(int64(h.Timeout)),
			"startPeriod":   llx.IntData(int64(h.StartPeriod)),
			"startInterval": llx.IntData(int64(h.StartInterval)),
			"retries":       llx.IntData(int64(h.Retries)),
			"none":          llx.BoolData(isNone),
		})
		if err != nil {
			return nil, err
		}
		args["healthcheck"] = llx.ResourceData(hcResource, ResourceDockerFileHealthcheck)
	} else {
		args["healthcheck"] = llx.NilData
	}

	if shellRaw != nil {
		shell := make([]any, len(shellRaw.Shell))
		for i, s := range shellRaw.Shell {
			shell[i] = s
		}
		shellResource, err := CreateResource(p.MqlRuntime, ResourceDockerFileShell, map[string]*llx.RawData{
			"__id":    llx.StringData(p.locationID(shellRaw.Location())),
			"command": llx.ArrayData(shell, types.String),
		})
		if err != nil {
			return nil, err
		}
		args["shell"] = llx.ResourceData(shellResource, ResourceDockerFileShell)
	} else {
		args["shell"] = llx.NilData
	}

	rawStage, err := CreateResource(p.MqlRuntime, ResourceDockerFileStage, args)
	if err != nil {
		return nil, err
	}

	return rawStage.(*mqlDockerFileStage), nil
}

func (p *mqlDockerFile) locationID(location []parser.Range) string {
	var line int
	var char int
	if len(location) != 0 {
		line = location[0].Start.Line
		char = location[0].Start.Character
	}
	return "dockerfile/" + p.File.Data.Path.Data + "/" + strconv.FormatInt(int64(line), 10) + ":" + strconv.FormatInt(int64(char), 10)
}

func (p *mqlDockerFile) instructions(file *mqlFile) (any, error) {
	return nil, p.parse(file)
}

func (p *mqlDockerFile) stages(file *mqlFile) ([]any, error) {
	return nil, p.parse(file)
}

func (p *mqlDockerFile) directives(file *mqlFile) (map[string]any, error) {
	return nil, p.parse(file)
}

func (p *mqlDockerFile) multiStage(file *mqlFile) (bool, error) {
	return false, p.parse(file)
}

func (p *mqlDockerFile) hasSyntaxDirective(file *mqlFile) (bool, error) {
	return false, p.parse(file)
}

func (p *mqlDockerFile) finalStage(file *mqlFile) (*mqlDockerFileStage, error) {
	return nil, p.parse(file)
}

// isRootUser reports whether the USER value resolves to root. Only the user
// portion is considered — the group is ignored.
func isRootUser(user string) bool {
	return user == "0" || user == "root"
}

// mountTypeFlags scans the parsed `--mount=...` entries on a RUN and reports
// whether any of them expose a build-time secret or ssh socket.
func mountTypeFlags(mounts []any) (secret bool, ssh bool) {
	for _, m := range mounts {
		rm, ok := m.(*mqlDockerFileRunMount)
		if !ok {
			continue
		}
		switch rm.Type.Data {
		case "secret":
			secret = true
		case "ssh":
			ssh = true
		}
	}
	return
}

// runFlagValue returns the parsed BuildKit value for `--network=...` or
// `--security=...`. Both default to a non-empty string when set explicitly;
// we surface them as empty strings when the flag wasn't used so audits can
// distinguish "default" (empty) from an explicit pin.
func runFlagValue(cmd *instructions.RunCommand, name string) string {
	for _, f := range cmd.FlagsUsed {
		if f != name {
			continue
		}
		switch name {
		case "network":
			return string(instructions.GetNetwork(cmd))
		case "security":
			return instructions.GetSecurity(cmd)
		}
	}
	return ""
}

func (p *mqlDockerFile) mountResources(cmd *instructions.RunCommand) ([]any, error) {
	hasMount := false
	for _, f := range cmd.FlagsUsed {
		if f == "mount" {
			hasMount = true
			break
		}
	}
	if !hasMount {
		return nil, nil
	}
	// instructions.Parse defers mount field resolution until a variable
	// expander is supplied. We don't expand build args during static analysis,
	// so feed an identity expander to populate Type/Target/Source/etc.
	if err := cmd.Expand(func(s string) (string, error) { return s, nil }); err != nil {
		return nil, err
	}
	mounts := instructions.GetMounts(cmd)
	out := make([]any, 0, len(mounts))
	for i, m := range mounts {
		id := p.locationID(cmd.Location()) + "/mount/" + strconv.Itoa(i)
		var mode, uid, gid int64
		if m.Mode != nil {
			mode = int64(*m.Mode)
		}
		if m.UID != nil {
			uid = int64(*m.UID)
		}
		if m.GID != nil {
			gid = int64(*m.GID)
		}
		var env string
		if m.Env != nil {
			env = *m.Env
		}
		resource, err := CreateResource(p.MqlRuntime, ResourceDockerFileRunMount, map[string]*llx.RawData{
			"__id":      llx.StringData(id),
			"type":      llx.StringData(string(m.Type)),
			"target":    llx.StringData(m.Target),
			"source":    llx.StringData(m.Source),
			"from":      llx.StringData(m.From),
			"id":        llx.StringData(m.CacheID),
			"sharing":   llx.StringData(string(m.CacheSharing)),
			"readOnly":  llx.BoolData(m.ReadOnly),
			"required":  llx.BoolData(m.Required),
			"sizeLimit": llx.IntData(m.SizeLimit),
			"mode":      llx.IntData(mode),
			"uid":       llx.IntData(uid),
			"gid":       llx.IntData(gid),
			"env":       llx.StringData(env),
		})
		if err != nil {
			return nil, err
		}
		out = append(out, resource)
	}
	return out, nil
}
