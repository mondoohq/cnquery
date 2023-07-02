package statutil

import (
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"time"

	"errors"
	"github.com/rs/zerolog/log"
	os_provider "go.mondoo.com/cnquery/motor/providers/os"
)

type CommandRunner interface {
	RunCommand(command string) (*os_provider.Command, error)
}

var ESCAPEREGEX = regexp.MustCompile(`[^\w@%+=:,./-]`)

func ShellEscape(s string) string {
	if len(s) == 0 {
		return "''"
	}
	if ESCAPEREGEX.MatchString(s) {
		return "'" + strings.Replace(s, "'", "'\"'\"'", -1) + "'"
	}

	return s
}

func New(cmdRunner CommandRunner) *statHelper {
	return &statHelper{
		commandRunner: cmdRunner,
	}
}

// Stat helper implements the stat command for various unix systems
// since this helper is used by transports itself, we cannot rely on the
// platform detection mechanism (since it may rely on stat to determine the system)
// therefore we implement the minimum required to detect the right stat parser
type statHelper struct {
	commandRunner CommandRunner
	detected      bool
	isUnix        bool
}

var bsdunix = map[string]bool{
	"openbsd":   true,
	"dragonfly": true,
	"freebsd":   true,
	"netbsd":    true,
	"darwin":    true, // use bsd stat for macOS
}

func (s *statHelper) Stat(name string) (os.FileInfo, error) {
	// detect stat version
	if !s.detected {
		cmd, err := s.commandRunner.RunCommand("uname -s")
		if err != nil {
			log.Debug().Err(err).Str("file", name).Msg("could not detect platform for file stat")
			return nil, err
		}

		data, err := io.ReadAll(cmd.Stdout)
		if err != nil {
			return nil, err
		}

		// only switch to unix if we properly detected it, otherwise fallback to linux
		val := strings.ToLower(strings.TrimSpace(string(data)))

		isUnix, ok := bsdunix[val]
		if ok && isUnix {
			s.isUnix = true
		}
		s.detected = true
	}

	if s.isUnix {
		return s.unix(name)
	}
	return s.linux(name)
}

func (s *statHelper) linux(name string) (os.FileInfo, error) {
	path := ShellEscape(name)

	// check if file exists
	cmd, err := s.commandRunner.RunCommand("test -e " + path)
	if err != nil || cmd.ExitStatus != 0 {
		return nil, os.ErrNotExist
	}

	// run stat
	lstat := "-L"
	format := "--printf"

	var sb strings.Builder

	sb.WriteString("stat ")
	sb.WriteString(lstat)
	sb.WriteString(" ")
	sb.WriteString(path)
	sb.WriteString(" ")
	sb.WriteString(format)
	sb.WriteString(" '%s\n%f\n%u\n%g\n%X\n%Y\n%C'")

	// NOTE: handling the exit code here does not work for all cases
	// sometimes stat returns something like: failed to get security context of '/etc/ssh/sshd_config': No data available
	// Therefore we continue after this command and try to parse the result and focus on making the parsing more robust
	command := sb.String()
	cmd, err = s.commandRunner.RunCommand(command)

	// we get stderr content in cases where we could not gather the security context via failed to get security context of
	// it could also include: No such file or directory
	if err != nil {
		log.Debug().Str("path", path).Str("command", command).Err(err).Send()
	}

	if cmd == nil {
		return nil, errors.New("could not parse file stat: " + path)
	}

	data, err := io.ReadAll(cmd.Stdout)
	if err != nil {
		return nil, err
	}

	statsData := strings.Split(string(data), "\n")
	if len(statsData) != 7 {
		log.Debug().Str("path", path).Msg("could not parse file stat information")
		// TODO: we may need to parse the returning error to better distinguish between a real error and file not found
		// if we are going to check for file not found, we probably run into the issue that the error message is returned in
		// multiple languages
		return nil, errors.New("could not parse file stat: " + path)
	}

	size, err := strconv.Atoi(statsData[0])
	if err != nil {
		return nil, errors.Join(err, errors.New("could not stat "+name))
	}

	uid, err := strconv.ParseInt(statsData[2], 10, 64)
	if err != nil {
		return nil, errors.Join(err, errors.New("could not stat "+name))
	}

	gid, err := strconv.ParseInt(statsData[3], 10, 64)
	if err != nil {
		return nil, errors.Join(err, errors.New("could not stat "+name))
	}

	mask, err := strconv.ParseUint(statsData[1], 16, 32)
	if err != nil {
		return nil, errors.Join(err, errors.New("could not stat "+name))
	}

	mtime, err := strconv.ParseInt(statsData[4], 10, 64)
	if err != nil {
		return nil, errors.Join(err, errors.New("could not stat "+name))
	}

	// extract file modes
	mapMode := toFileMode(mask)

	return &os_provider.FileInfo{
		FName:    filepath.Base(path),
		FSize:    int64(size),
		FMode:    mapMode,
		FIsDir:   mapMode.IsDir(),
		FModTime: time.Unix(mtime, 0),
		Uid:      uid,
		Gid:      gid,
	}, nil
}

func (s *statHelper) unix(name string) (os.FileInfo, error) {
	lstat := "-L"
	format := "-f"
	path := ShellEscape(name)

	var sb strings.Builder
	sb.WriteString("stat ")
	sb.WriteString(lstat)
	sb.WriteString(" ")
	sb.WriteString(format)
	sb.WriteString(" '%z:%p:%u:%g:%a:%m'")
	sb.WriteString(" ")
	sb.WriteString(path)

	cmd, err := s.commandRunner.RunCommand(sb.String())
	if err != nil {
		log.Debug().Err(err).Send()
	}

	data, err := io.ReadAll(cmd.Stdout)
	if err != nil {
		return nil, err
	}

	statsData := strings.Split(string(data), ":")
	if len(statsData) != 6 {
		log.Error().Str("name", name).Msg("could not parse file stat information")
		// TODO: there are likely cases where the file exist but we could still not parse it
		return nil, os.ErrNotExist
	}

	size, err := strconv.Atoi(statsData[0])
	if err != nil {
		return nil, errors.Join(err, errors.New("could not stat "+name))
	}

	uid, err := strconv.ParseInt(statsData[2], 10, 64)
	if err != nil {
		return nil, errors.Join(err, errors.New("could not stat "+name))
	}

	gid, err := strconv.ParseInt(statsData[3], 10, 64)
	if err != nil {
		return nil, errors.Join(err, errors.New("could not stat "+name))
	}

	// NOTE: the base is 8 instead of 16 on linux systems
	mask, err := strconv.ParseUint(statsData[1], 8, 32)
	if err != nil {
		return nil, errors.Join(err, errors.New("could not stat "+name))
	}

	mode := toFileMode(mask)

	mtime, err := strconv.ParseInt(statsData[4], 10, 64)
	if err != nil {
		return nil, errors.Join(err, errors.New("could not stat "+name))
	}

	return &os_provider.FileInfo{
		FName:    filepath.Base(path),
		FSize:    int64(size),
		FMode:    mode,
		FIsDir:   mode.IsDir(),
		FModTime: time.Unix(mtime, 0),
		Uid:      uid,
		Gid:      gid,
	}, nil
}

const (
	S_IFMT  = 0o170000
	S_IFBLK = 0o60000
	S_IFCHR = 0o20000
	S_IFDIR = 0o40000
	S_IFIFO = 10000
	S_ISUID = 0o4000
	S_ISGID = 0o2000
	S_ISVTX = 0o1000
)

func toFileMode(mask uint64) os.FileMode {
	mode := os.FileMode(uint32(mask) & 0o0777)

	// taken from https://github.com/golang/go/blob/2ebe77a2fda1ee9ff6fd9a3e08933ad1ebaea039/src/os/stat_linux.go
	switch mask & S_IFMT {
	case S_IFBLK:
		mode |= fs.ModeDevice
	case S_IFCHR:
		mode |= fs.ModeDevice | fs.ModeCharDevice
	case S_IFDIR:
		mode |= fs.ModeDir
	case S_IFIFO:
		mode |= fs.ModeNamedPipe
	case syscall.S_IFLNK:
		mode |= fs.ModeSymlink
	case syscall.S_IFREG:
		// nothing to do
	case syscall.S_IFSOCK:
		mode |= fs.ModeSocket
	}
	if mask&syscall.S_ISGID != 0 {
		mode |= fs.ModeSetgid
	}
	if mask&syscall.S_ISUID != 0 {
		mode |= fs.ModeSetuid
	}
	if mask&syscall.S_ISVTX != 0 {
		mode |= fs.ModeSticky
	}
	return mode
}
