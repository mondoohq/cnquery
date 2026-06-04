// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package project

import (
	"os"
	"path/filepath"
	"sort"
)

// Plugin is a custom module or plugin shipped inside the project — code that
// executes during a run and therefore forms part of its supply chain.
type Plugin struct {
	Name string // file name without extension
	Type string // plugin type, e.g. module, filter, lookup, module_utils
	Path string // absolute path of the plugin file
}

// pluginDirTypes maps the conventional top-level plugin directories to the
// plugin type they hold. The `plugins/<type>/` collection layout is handled
// separately.
var pluginDirTypes = map[string]string{
	"library":            "module",
	"module_utils":       "module_utils",
	"filter_plugins":     "filter",
	"lookup_plugins":     "lookup",
	"action_plugins":     "action",
	"callback_plugins":   "callback",
	"strategy_plugins":   "strategy",
	"inventory_plugins":  "inventory",
	"vars_plugins":       "vars",
	"connection_plugins": "connection",
	"test_plugins":       "test",
	"cache_plugins":      "cache",
}

// loadPlugins discovers custom plugins and modules in the project, both in the
// classic top-level directories (library/, filter_plugins/, …) and under the
// collection-style plugins/<type>/ layout.
func loadPlugins(root string) ([]*Plugin, error) {
	var plugins []*Plugin

	for dir, ptype := range pluginDirTypes {
		plugins = append(plugins, collectPlugins(filepath.Join(root, dir), ptype)...)
	}

	// plugins/<type>/... — the layout collections and newer projects use.
	pluginsDir := filepath.Join(root, "plugins")
	if entries, err := os.ReadDir(pluginsDir); err == nil {
		for _, e := range entries {
			if e.IsDir() {
				plugins = append(plugins, collectPlugins(filepath.Join(pluginsDir, e.Name()), pluginTypeFromDir(e.Name()))...)
			}
		}
	}

	sort.Slice(plugins, func(i, j int) bool {
		if plugins[i].Type != plugins[j].Type {
			return plugins[i].Type < plugins[j].Type
		}
		return plugins[i].Path < plugins[j].Path
	})
	return plugins, nil
}

// collectPlugins returns the plugin files (recursively) under dir, classified as
// the given type. Non-code files (__init__.py, READMEs) are skipped.
func collectPlugins(dir, ptype string) []*Plugin {
	var plugins []*Plugin
	_ = filepath.WalkDir(dir, func(p string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		name := d.Name()
		if !isPluginFile(name) {
			return nil
		}
		plugins = append(plugins, &Plugin{
			Name: trimExt(name),
			Type: ptype,
			Path: p,
		})
		return nil
	})
	return plugins
}

func isPluginFile(name string) bool {
	if name == "__init__.py" || name == "__pycache__" {
		return false
	}
	switch filepath.Ext(name) {
	case ".py", ".ps1", ".yml", ".yaml":
		return true
	}
	return false
}

// pluginTypeFromDir maps a plugins/<dir> name to a plugin type, normalizing the
// "modules" directory to the singular "module".
func pluginTypeFromDir(dir string) string {
	if dir == "modules" {
		return "module"
	}
	return dir
}

func trimExt(name string) string {
	return name[:len(name)-len(filepath.Ext(name))]
}
