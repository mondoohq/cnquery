// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"errors"
	"slices"
	"strconv"
	"strings"
	"sync"

	"github.com/moby/buildkit/frontend/dockerfile/instructions"
	"github.com/moby/buildkit/frontend/dockerfile/parser"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/v11/llx"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v11/providers/os/connection/docker"
	"go.mondoo.com/cnquery/v11/types"
	"go.mondoo.com/cnquery/v11/utils/multierr"
)

func initDockerFile(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
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
		path = dfc.Filename
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
		return err
	}

	content := file.GetContent()
	if content.Error != nil {
		return setError(content.Error)
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

	parsedStages, meta, err := instructions.Parse(ast.AST)
	if err != nil {
		return setError(multierr.Wrap(err, "failed to parse dockerfile instructions in "+file.Path.Data))
	}

	stages := make([]any, len(parsedStages))
	var stagesErr error
	for i := range parsedStages {
		stages[i], err = p.stage2resource(parsedStages[i])
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

	// FIXME: add meta data
	_ = meta

	return nil
}

func (p *mqlDockerFile) stage2resource(stage instructions.Stage) (*mqlDockerFileStage, error) {
	var image string
	var tag string
	var digest string
	if idx := strings.Index(stage.BaseName, ":"); idx != -1 {
		image = stage.BaseName[:idx]
		if len(stage.BaseName) > idx+1 {
			tag = stage.BaseName[idx+1:]
		}
	} else if idx := strings.Index(stage.BaseName, "@"); idx != -1 {
		image = stage.BaseName[:idx]
		if len(stage.BaseName) > idx+1 {
			tag = stage.BaseName[idx+1:]
		}
	} else {
		image = stage.BaseName
	}

	stageID := p.locationID(stage.Location)

	rawFrom, err := CreateResource(p.MqlRuntime, "docker.file.from", map[string]*llx.RawData{
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

	env := map[string]any{}
	var runs []any
	var copy []any
	var add []any
	var unsupported []string
	var entrypointRaw *instructions.EntrypointCommand
	var cmdRaw *instructions.CmdCommand
	for i := range stage.Commands {
		switch v := stage.Commands[i].(type) {
		case *instructions.EnvCommand:
			for _, kv := range v.Env {
				env[kv.Key] = strings.Trim(kv.Value, "\"")
			}

		case *instructions.RunCommand:
			script := strings.Join(v.ShellDependantCmdLine.CmdLine, "\n")
			runResource, err := CreateResource(p.MqlRuntime, "docker.file.run", map[string]*llx.RawData{
				"__id":   llx.StringData(p.locationID(v.Location())),
				"script": llx.StringData(script),
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
			src := make([]any, len(v.SourcesAndDest.SourcePaths))
			for i := range v.SourcesAndDest.SourcePaths {
				src[i] = v.SourcesAndDest.SourcePaths[i]
			}
			resource, err := CreateResource(p.MqlRuntime, "docker.file.copy", map[string]*llx.RawData{
				"src": llx.ArrayData(src, types.String),
				"dst": llx.StringData(v.SourcesAndDest.DestPath),
			})
			if err != nil {
				return nil, err
			}
			copy = append(copy, resource)

		case *instructions.AddCommand:
			src := make([]any, len(v.SourcesAndDest.SourcePaths))
			for i := range v.SourcesAndDest.SourcePaths {
				src[i] = v.SourcesAndDest.SourcePaths[i]
			}
			resource, err := CreateResource(p.MqlRuntime, "docker.file.add", map[string]*llx.RawData{
				"src":   llx.ArrayData(src, types.String),
				"dst":   llx.StringData(v.SourcesAndDest.DestPath),
				"chown": llx.StringData(v.Chown),
				"chmod": llx.StringData(v.Chmod),
			})
			if err != nil {
				return nil, err
			}
			add = append(add, resource)

		default:
			cmd := stage.Commands[i]
			unsupported = append(unsupported, cmd.Name())
		}
	}

	if len(unsupported) != 0 {
		slices.Sort(unsupported)
		log.Warn().Strs("commands", slices.Compact(unsupported)).Msg("unsuppoprted dockerfile commands")
	}

	args := map[string]*llx.RawData{
		"__id": llx.StringData(stageID),
		"from": llx.ResourceData(rawFrom, "docker.file.from"),
		"file": llx.ResourceData(p, "docker.file"),
		"env":  llx.MapData(env, types.String),
		"run":  llx.ArrayData(runs, types.Resource("docker.file.run")),
		"add":  llx.ArrayData(add, types.Resource("docker.file.add")),
		"copy": llx.ArrayData(copy, types.Resource("docker.file.copy")),
	}

	if entrypointRaw != nil {
		script := strings.Join(entrypointRaw.ShellDependantCmdLine.CmdLine, "\n")
		runResource, err := CreateResource(p.MqlRuntime, "docker.file.run", map[string]*llx.RawData{
			"__id":   llx.StringData(p.locationID(entrypointRaw.Location())),
			"script": llx.StringData(script),
		})
		if err != nil {
			return nil, err
		}
		args["entrypoint"] = llx.ResourceData(runResource, "docker.file.run")
	} else {
		args["entrypoint"] = llx.NilData
	}

	if cmdRaw != nil {
		script := strings.Join(cmdRaw.ShellDependantCmdLine.CmdLine, "\n")
		cmdResource, err := CreateResource(p.MqlRuntime, "docker.file.run", map[string]*llx.RawData{
			"__id":   llx.StringData(p.locationID(cmdRaw.Location())),
			"script": llx.StringData(script),
		})
		if err != nil {
			return nil, err
		}
		args["cmd"] = llx.ResourceData(cmdResource, "docker.file.run")
	} else {
		args["cmd"] = llx.NilData
	}

	rawStage, err := CreateResource(p.MqlRuntime, "docker.file.stage", args)
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
