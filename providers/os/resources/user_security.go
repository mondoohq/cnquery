// Copyright Mondoo, Inc. 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"strconv"
	"strings"
	"time"

	"github.com/spf13/afero"
	"go.mondoo.com/mql/llx"
	"go.mondoo.com/mql/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/providers/os/connection/shared"
)

// joinHome joins a home directory with a relative path using forward slashes,
// which the connection filesystem accepts on every platform (including Windows).
func joinHome(home, rel string) string {
	return strings.TrimRight(home, `/\`) + "/" + rel
}

// --- user.knownHosts ---

func (u *mqlUser) knownHosts(home string) (*mqlKnownhosts, error) {
	path := joinHome(home, ".ssh/known_hosts")
	kh, err := CreateResource(u.MqlRuntime, "knownhosts", map[string]*llx.RawData{
		"path": llx.StringData(path),
	})
	if err != nil {
		return nil, err
	}
	return kh.(*mqlKnownhosts), nil
}

func initKnownhosts(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if _, ok := args["path"]; !ok {
		// standalone default: the system-wide known_hosts
		args["path"] = llx.StringData("/etc/ssh/ssh_known_hosts")
	}
	return args, nil, nil
}

func (x *mqlKnownhosts) id() (string, error)     { return "knownhosts:" + x.Path.Data, nil }
func (x *mqlKnownhosts) file() (*mqlFile, error) { return networkDBFile(x.MqlRuntime, x.Path.Data) }
func (x *mqlKnownhosts) content() (string, error) {
	content, _, err := readNetworkDB(x.MqlRuntime, x.Path.Data)
	return content, err
}

func (x *mqlKnownhosts) list() ([]any, error) {
	res := []any{}
	content, exists, err := readNetworkDB(x.MqlRuntime, x.Path.Data)
	if err != nil || !exists {
		return res, err
	}
	for i, raw := range strings.Split(content, "\n") {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Fields(line)
		idx := 0
		// optional marker: @cert-authority / @revoked
		if strings.HasPrefix(fields[0], "@") {
			idx = 1
		}
		if len(fields) < idx+3 {
			continue
		}
		host := fields[idx]
		entry, err := CreateResource(x.MqlRuntime, "knownhosts.entry", map[string]*llx.RawData{
			"line":     llx.IntData(int64(i + 1)),
			"host":     llx.StringData(host),
			"isHashed": llx.BoolData(strings.HasPrefix(host, "|1|")),
			"type":     llx.StringData(fields[idx+1]),
			"key":      llx.StringData(fields[idx+2]),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, entry)
	}
	return res, nil
}

func (x *mqlKnownhostsEntry) id() (string, error) {
	return "knownhosts.entry/" + strconv.FormatInt(x.Line.Data, 10) + "/" + x.Host.Data, nil
}

// --- user.shellHistory ---

type historyCommand struct {
	command string
	ts      *time.Time
}

func (u *mqlUser) shellHistory(home string) ([]any, error) {
	res := []any{}
	conn, ok := u.MqlRuntime.Connection.(shared.Connection)
	if !ok {
		return res, nil
	}
	afs := &afero.Afero{Fs: conn.FileSystem()}

	sources := []struct {
		path string
		kind string
	}{
		{joinHome(home, ".bash_history"), "bash"},
		{joinHome(home, ".zsh_history"), "zsh"},
		{joinHome(home, ".local/share/fish/fish_history"), "fish"},
		{joinHome(home, "AppData/Roaming/Microsoft/Windows/PowerShell/PSReadLine/ConsoleHost_history.txt"), "powershell"},
	}

	for _, src := range sources {
		exists, err := afs.Exists(src.path)
		if err != nil || !exists {
			continue
		}
		b, err := afs.ReadFile(src.path)
		if err != nil {
			continue
		}
		for i, cmd := range parseHistory(string(b), src.kind) {
			entry, err := CreateResource(u.MqlRuntime, "shellHistory.command", map[string]*llx.RawData{
				"user":    llx.StringData(u.Name.Data),
				"line":    llx.IntData(int64(i + 1)),
				"command": llx.StringData(cmd.command),
				"time":    llx.TimeDataPtr(cmd.ts),
				"file":    llx.StringData(src.path),
			})
			if err != nil {
				return nil, err
			}
			res = append(res, entry)
		}
	}
	return res, nil
}

func (x *mqlShellHistoryCommand) id() (string, error) {
	return "shellHistory/" + x.User.Data + "/" + x.File.Data + ":" + strconv.FormatInt(x.Line.Data, 10), nil
}

// parseHistory extracts commands (and timestamps where the shell records them)
// from a shell history file. bash and PowerShell PSReadLine store no timestamps
// by default; zsh only with EXTENDED_HISTORY; fish always does.
func parseHistory(content, kind string) []historyCommand {
	switch kind {
	case "zsh":
		return parseZshHistory(content)
	case "fish":
		return parseFishHistory(content)
	default: // bash, powershell: plain command-per-line (bash may carry #<epoch> markers)
		return parsePlainHistory(content)
	}
}

func parsePlainHistory(content string) []historyCommand {
	var out []historyCommand
	var pending *time.Time
	for _, raw := range strings.Split(content, "\n") {
		line := strings.TrimRight(raw, "\r")
		if line == "" {
			continue
		}
		// bash HISTTIMEFORMAT writes a "#<epoch>" line before each command
		if strings.HasPrefix(line, "#") {
			if epoch, err := strconv.ParseInt(strings.TrimSpace(line[1:]), 10, 64); err == nil {
				t := time.Unix(epoch, 0).UTC()
				pending = &t
				continue
			}
		}
		out = append(out, historyCommand{command: line, ts: pending})
		pending = nil
	}
	return out
}

func parseZshHistory(content string) []historyCommand {
	var out []historyCommand
	for _, raw := range strings.Split(content, "\n") {
		line := strings.TrimRight(raw, "\r")
		if line == "" {
			continue
		}
		// extended: ": <epoch>:<elapsed>;<command>"
		if strings.HasPrefix(line, ": ") {
			if semi := strings.IndexByte(line, ';'); semi >= 0 {
				meta := line[2:semi]
				cmd := line[semi+1:]
				var ts *time.Time
				if colon := strings.IndexByte(meta, ':'); colon >= 0 {
					if epoch, err := strconv.ParseInt(strings.TrimSpace(meta[:colon]), 10, 64); err == nil {
						t := time.Unix(epoch, 0).UTC()
						ts = &t
					}
				}
				out = append(out, historyCommand{command: cmd, ts: ts})
				continue
			}
		}
		out = append(out, historyCommand{command: line})
	}
	return out
}

func parseFishHistory(content string) []historyCommand {
	var out []historyCommand
	var cur *historyCommand
	flush := func() {
		if cur != nil {
			out = append(out, *cur)
			cur = nil
		}
	}
	for _, raw := range strings.Split(content, "\n") {
		line := strings.TrimRight(raw, "\r")
		switch {
		case strings.HasPrefix(line, "- cmd:"):
			flush()
			cur = &historyCommand{command: strings.TrimSpace(strings.TrimPrefix(line, "- cmd:"))}
		case cur != nil && strings.HasPrefix(strings.TrimSpace(line), "when:"):
			v := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line), "when:"))
			if epoch, err := strconv.ParseInt(v, 10, 64); err == nil {
				t := time.Unix(epoch, 0).UTC()
				cur.ts = &t
			}
		}
	}
	flush()
	return out
}
