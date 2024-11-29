// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package processes

import (
	"bufio"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/kballard/go-shellquote"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v11/providers/os/connection/shared"
)

var (
	LINUX_PS_REGEX = regexp.MustCompile(`^\s*([^ ]+)\s+([^ ]+)\s+([^ ]+)\s+([^ ]+)\s+([^ ]+)\s+([^ ]+)\s+([^ ]+)\s+([^ ]+)\s+([^ ]+)\s+([^ ]+)\s+([^ ].*)?$`)
	UNIX_PS_REGEX  = regexp.MustCompile(`^\s*([^ ]+)\s+([^ ]+)\s+([^ ]+)\s+([^ ]+)\s+([^ ]+)\s+([^ ]+)\s+([^ ]+)\s+([^ ]+)\s+([^ ]+)\s+([^ ].*)$`)
	AIX_PS_REGEX   = regexp.MustCompile(`^\s*([^ ]+)\s+([^ ]+)\s+([^ ]+)\s+([^ ]+)\s+([^ ]+)\s+([^ ]+)\s+([^ ]+)\s+([^ ].*)$`)

	// "lrwx------ 1 0 0 64 Dec  6 13:56 /proc/1/fd/12 -> socket:[37364]"
	reFindSockets = regexp.MustCompile(
		"^[lrwx-]+\\.?\\s+" +
			"\\d+\\s+" +
			"\\d+\\s+" + // uid
			"\\d+\\s+" + // gid
			"\\d+\\s+" +
			"[^ ]+\\s+" + // month, e.g. Dec
			"\\d+\\s+" + // day
			"\\d+:\\d+\\s+" + // time
			"/proc/(\\d+)/fd/\\d+\\s+" + // path
			"->\\s+" +
			".*socket:\\[(\\d+)\\].*\\s*") // target
)

type ProcessEntry struct {
	Pid     int64
	CPU     string
	Mem     string
	Vsz     string
	Rss     string
	Tty     string
	Stat    string
	Start   string
	Time    string
	Uid     int64
	Command string
}

func (p ProcessEntry) ToOSProcess() *OSProcess {
	executablePath := ""
	args, err := shellquote.Split(p.Command)
	if err == nil && len(args) > 0 {
		executablePath = args[0]
	}

	executablePathParts := strings.Split(executablePath, "/")
	return &OSProcess{
		Pid:        p.Pid,
		Command:    p.Command,
		Executable: executablePathParts[len(executablePathParts)-1],
		State:      "",
	}
}

func ParseLinuxPsResult(input io.Reader) ([]*ProcessEntry, error) {
	processes := []*ProcessEntry{}
	scanner := bufio.NewScanner(input)
	for scanner.Scan() {
		line := scanner.Text()

		m := LINUX_PS_REGEX.FindStringSubmatch(line)
		if len(m) != 12 {
			log.Fatal().Str("psoutput", line).Msg("unexpected result while trying to parse process output")
		}
		if m[1] == "PID" {
			// header
			continue
		}

		pid, err := strconv.ParseInt(m[1], 10, 64)
		if err != nil {
			log.Error().Err(err).Msg("cannot parse ps pid " + m[1])
			continue
		}
		uid, err := strconv.ParseInt(m[10], 10, 64)
		if err != nil {
			log.Error().Err(err).Msg("cannot parse ps uid " + m[10])
			continue
		}

		// PID %CPU %MEM    VSZ   RSS TT       STAT  STARTED     TIME   UID COMMAND
		p := &ProcessEntry{
			Pid:     pid,
			CPU:     m[2],
			Mem:     m[3],
			Vsz:     m[4],
			Rss:     m[5],
			Tty:     m[6],
			Stat:    m[7],
			Start:   m[8],
			Time:    m[9],
			Uid:     uid,
			Command: m[11],
		}
		processes = append(processes, p)
	}

	return processes, nil
}

func ParseUnixPsResult(input io.Reader) ([]*ProcessEntry, error) {
	processes := []*ProcessEntry{}
	scanner := bufio.NewScanner(input)
	for scanner.Scan() {
		line := scanner.Text()
		m := UNIX_PS_REGEX.FindStringSubmatch(line)
		if len(m) != 11 {
			log.Fatal().Str("psoutput", line).Msg("unexpected result while trying to parse process output")
		}
		if m[1] == "PID" {
			// header
			continue
		}

		pid, err := strconv.ParseInt(m[1], 10, 64)
		if err != nil {
			log.Error().Err(err).Msg("cannot parse unix pid " + m[1])
			continue
		}
		uid, err := strconv.ParseInt(m[9], 10, 64)
		if err != nil {
			log.Error().Err(err).Msg("cannot parse unix uid " + m[9])
			continue
		}

		// PID %CPU %MEM    VSZ   RSS TTY       STAT  TIME   UID COMMAND
		p := &ProcessEntry{
			Pid:     pid,
			CPU:     m[2],
			Mem:     m[3],
			Vsz:     m[4],
			Rss:     m[5],
			Tty:     m[6],
			Stat:    m[7],
			Time:    m[8],
			Uid:     uid,
			Command: m[10],
		}
		processes = append(processes, p)
	}

	return processes, nil
}

func ParseAixPsResult(input io.Reader) ([]*ProcessEntry, error) {
	processes := []*ProcessEntry{}
	scanner := bufio.NewScanner(input)
	for scanner.Scan() {
		line := scanner.Text()
		// skip defunct processes
		if strings.Contains(line, "defunct") {
			continue
		}

		m := AIX_PS_REGEX.FindStringSubmatch(line)
		if len(m) != 9 {
			log.Fatal().Str("psoutput", line).Msg("unexpected result while trying to parse process output")
		}
		if m[1] == "PID" {
			// header
			continue
		}

		pid, err := strconv.ParseInt(m[1], 10, 64)
		if err != nil {
			log.Error().Err(err).Msg("cannot parse unix pid " + m[1])
			continue
		}
		uid, err := strconv.ParseInt(m[7], 10, 64)
		if err != nil {
			log.Error().Err(err).Msg("cannot parse unix uid " + m[9])
			continue
		}

		// PID  %CPU  %MEM   VSZ     TT        TIME UID COMMAND
		p := &ProcessEntry{
			Pid:     pid,
			CPU:     m[2],
			Mem:     m[3],
			Vsz:     m[4],
			Tty:     m[5],
			Time:    m[6],
			Uid:     uid,
			Command: m[8],
		}
		processes = append(processes, p)
	}

	return processes, nil
}

type UnixProcessManager struct {
	conn     shared.Connection
	platform *inventory.Platform
}

func (upm *UnixProcessManager) Name() string {
	return "Unix Process Manager"
}

func (upm *UnixProcessManager) List() ([]*OSProcess, error) {
	var entries []*ProcessEntry
	// NOTE: improve proc parser instead of supporting multiple ps commands
	if upm.platform.IsFamily("linux") {
		c, err := upm.conn.RunCommand("ps axo pid,pcpu,pmem,vsz,rss,tty,stat,stime,time,uid,command")
		if err != nil {
			return nil, fmt.Errorf("processes> could not run command")
		}

		entries, err = ParseLinuxPsResult(c.Stdout)
		if err != nil {
			return nil, err
		}
	} else if upm.platform.IsFamily("darwin") {
		// NOTE: special case on darwin is that the ps axo only shows processes for users with terminals
		// TODO: the same applies to OpenBSD and may result in missing processes
		c, err := upm.conn.RunCommand("ps Axo pid,pcpu,pmem,vsz,rss,tty,stat,stime,time,uid,command")
		if err != nil {
			return nil, fmt.Errorf("processes> could not run command")
		}

		entries, err = ParseLinuxPsResult(c.Stdout)
		if err != nil {
			return nil, err
		}
	} else if upm.platform.Name == "aix" {
		// special case for aix since it does not understand x
		c, err := upm.conn.RunCommand("ps -A -o pid,pcpu,pmem,vsz,tty,time,uid,args")
		if err != nil {
			return nil, fmt.Errorf("processes> could not run command")
		}

		entries, err = ParseAixPsResult(c.Stdout)
		if err != nil {
			return nil, err
		}
	} else {
		// TODO: consider using different ps calls for different platforms to determine max information
		// do not use stime since it is not available on FreeBSD
		c, err := upm.conn.RunCommand("ps axo pid,pcpu,pmem,vsz,rss,tty,stat,time,uid,command")
		if err != nil {
			return nil, fmt.Errorf("processes> could not run command")
		}

		entries, err = ParseUnixPsResult(c.Stdout)
		if err != nil {
			return nil, err
		}
	}

	log.Debug().Int("processes", len(entries)).Msg("found processes")

	var ps []*OSProcess
	for i := range entries {
		ps = append(ps, entries[i].ToOSProcess())
	}
	return ps, nil
}

// ListSocketInodesByProcess returns a map with a pid as key and a list of socket inodes as value
func (upm *UnixProcessManager) ListSocketInodesByProcess() (map[int64]plugin.TValue[[]int64], error) {
	startTime := time.Now()
	c, err := upm.conn.RunCommand("find /proc -maxdepth 4 -path '/proc/*/fd/*' -exec ls -n {} \\;")
	if err != nil {
		return nil, fmt.Errorf("processes> could not run command: %v", err)
	}

	processesInodesByPid := map[int64]plugin.TValue[[]int64]{}
	scanner := bufio.NewScanner(c.Stdout)
	for scanner.Scan() {
		line := scanner.Text()
		pid, inode, err := ParseLinuxFindLine(line)
		if err != nil || (pid == 0 && inode == 0) {
			pluginValue := processesInodesByPid[pid]
			pluginValue.Error = err
			processesInodesByPid[pid] = pluginValue
			continue
		}
		pluginValue := plugin.TValue[[]int64]{}
		if _, ok := processesInodesByPid[pid]; ok {
			pluginValue = processesInodesByPid[pid]
			pluginValue.Data = append(pluginValue.Data, inode)
		} else {
			pluginValue.Data = []int64{inode}
		}
		processesInodesByPid[pid] = pluginValue
	}
	log.Debug().Int64("duration (ms)", time.Duration(time.Since(startTime)).Milliseconds()).Msg("parsing find for process socket inodes")

	return processesInodesByPid, nil
}

func (upm *UnixProcessManager) Exists(pid int64) (bool, error) {
	process, err := upm.Process(pid)
	if err != nil {
		return false, err
	}

	if process == nil {
		return false, nil
	}

	return true, nil
}

func (upm *UnixProcessManager) Process(pid int64) (*OSProcess, error) {
	processes, err := upm.List()
	if err != nil {
		return nil, err
	}

	for i := range processes {
		if processes[i].Pid == pid {
			return processes[i], nil
		}
	}

	return nil, nil
}

func ParseLinuxFindLine(line string) (int64, int64, error) {
	if strings.HasSuffix(line, "Permission denied") || strings.HasSuffix(line, "No such file or directory") {
		return 0, 0, nil
	}

	m := reFindSockets.FindStringSubmatch(line)
	if len(m) == 0 {
		return 0, 0, nil
	}

	pid, err := strconv.ParseInt(m[1], 10, 64)
	if err != nil {
		log.Error().Err(err).Msg("cannot parse unix pid " + m[1])
		return 0, 0, err
	}

	inode, err := strconv.ParseInt(m[2], 10, 64)
	if err != nil {
		log.Error().Err(err).Msg("cannot parse socket inode " + m[2])
		return 0, 0, err
	}

	return pid, inode, nil
}
