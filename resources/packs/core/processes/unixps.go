package processes

import (
	"bufio"
	"fmt"
	"io"
	"regexp"
	"strconv"

	"github.com/kballard/go-shellquote"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/motor/platform"
	"go.mondoo.com/cnquery/motor/providers/os"
	"go.mondoo.com/cnquery/resources/packs/core/lsof"
)

var (
	LINUX_PS_REGEX = regexp.MustCompile(`^\s*([^ ]+)\s+([^ ]+)\s+([^ ]+)\s+([^ ]+)\s+([^ ]+)\s+([^ ]+)\s+([^ ]+)\s+([^ ]+)\s+([^ ]+)\s+([^ ]+)\s+([^ ].*)$`)
	UNIX_PS_REGEX  = regexp.MustCompile(`^\s*([^ ]+)\s+([^ ]+)\s+([^ ]+)\s+([^ ]+)\s+([^ ]+)\s+([^ ]+)\s+([^ ]+)\s+([^ ]+)\s+([^ ]+)\s+([^ ].*)$`)
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
	executable := ""
	args, err := shellquote.Split(p.Command)
	if err == nil && len(args) > 0 {
		executable = args[0]
	}

	return &OSProcess{
		Pid:        p.Pid,
		Command:    p.Command,
		Executable: executable,
		State:      "",
	}
}

func ParseLinuxPsResult(input io.Reader) ([]*ProcessEntry, error) {
	processes := []*ProcessEntry{}
	scanner := bufio.NewScanner(input)
	for scanner.Scan() {
		line := scanner.Text()

		m := LINUX_PS_REGEX.FindStringSubmatch(line)
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

		// PID %CPU %MEM    VSZ   RSS TT       STAT  STARTED     TIME   UID COMMAND
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

type UnixProcessManager struct {
	provider os.OperatingSystemProvider
	platform *platform.Platform
}

func (upm *UnixProcessManager) Name() string {
	return "Unix Process Manager"
}

func (upm *UnixProcessManager) List() ([]*OSProcess, error) {
	var entries []*ProcessEntry
	// NOTE: improve proc parser instead of supporting multiple ps commands
	if upm.platform.IsFamily("linux") {
		c, err := upm.provider.RunCommand("ps axo pid,pcpu,pmem,vsz,rss,tty,stat,stime,time,uid,command")
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
		c, err := upm.provider.RunCommand("ps Axo pid,pcpu,pmem,vsz,rss,tty,stat,stime,time,uid,command")
		if err != nil {
			return nil, fmt.Errorf("processes> could not run command")
		}

		entries, err = ParseLinuxPsResult(c.Stdout)
		if err != nil {
			return nil, err
		}
	} else {
		// TODO: consider using different ps calls for different platforms to determine max information
		// do not use stime since it is not available on FreeBSD
		c, err := upm.provider.RunCommand("ps axo pid,pcpu,pmem,vsz,rss,tty,stat,time,uid,command")
		if err != nil {
			return nil, fmt.Errorf("processes> could not run command")
		}

		entries, err = ParseUnixPsResult(c.Stdout)
		if err != nil {
			return nil, err
		}
	}

	log.Debug().Int("processes", len(entries)).Msg("found processes")

	// get socket information to enrich the process list
	sockets, err := upm.getSockets()
	if err != nil {
		log.Error().Err(err).Msg("processes> cannot get sockets")
	}

	var ps []*OSProcess
	for i := range entries {
		osProcess := entries[i].ToOSProcess()
		osProcess.SocketInodes = sockets[osProcess.Pid]
		ps = append(ps, osProcess)
	}
	return ps, nil
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

func (upm *UnixProcessManager) getSockets() (map[int64][]int64, error) {
	c, err := upm.provider.RunCommand("lsof -nP -i -F")
	if err != nil {
		return nil, fmt.Errorf("processes> could not run command: %v", err)
	}

	lsofProcesses, err := lsof.Parse(c.Stdout)
	if err != nil {
		return nil, err
	}

	sockets := map[int64][]int64{}
	for i := range lsofProcesses {

		pid, err := strconv.ParseInt(lsofProcesses[i].PID, 10, 64)
		if err != nil {
			log.Error().Err(err).Msg("cannot parse unix pid " + lsofProcesses[i].PID)
			continue
		}
		inode, err := strconv.ParseInt(lsofProcesses[i].SocketInode, 10, 64)
		if err != nil {
			log.Error().Err(err).Msg("cannot parse socket inode " + lsofProcesses[i].SocketInode)
			continue
		}

		sockets[pid] = append(sockets[pid], inode)
	}

	return sockets, nil
}
