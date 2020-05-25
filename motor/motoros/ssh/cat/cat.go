package cat

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"

	"github.com/spf13/afero"
	"go.mondoo.io/mondoo/motor/motoros/types"
)

type CommandRunner interface {
	RunCommand(command string) (*types.Command, error)
}

func New(cmdRunner CommandRunner) *CatFs {
	return &CatFs{
		commandRunner: cmdRunner,
	}
}

type CatFs struct {
	commandRunner CommandRunner
}

func (cat *CatFs) Name() string {
	return "Cat FS"
}

func (cat *CatFs) Open(name string) (afero.File, error) {
	cmd, err := cat.commandRunner.RunCommand(fmt.Sprintf("cat %s", name))
	if err != nil {
		return nil, err
	}

	data, err := ioutil.ReadAll(cmd.Stdout)
	if err != nil {
		return nil, err
	}

	return NewFile(name, bytes.NewBuffer(data)), nil
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

func (cat *CatFs) Stat(name string) (os.FileInfo, error) {

	lstat := "-L"
	format := "--printf"
	path := ShellEscape(name)

	var sb strings.Builder

	sb.WriteString("stat ")
	sb.WriteString(lstat)
	sb.WriteString(" ")
	sb.WriteString(path)
	sb.WriteString(" 2>/dev/null ")
	sb.WriteString(format)
	sb.WriteString(" '%s\n%f\n%u\n%g\n%X\n%Y\n%C'")

	cmd, err := cat.commandRunner.RunCommand(sb.String())
	if err != nil {
		return nil, os.ErrNotExist
	}

	data, err := ioutil.ReadAll(cmd.Stdout)
	if err != nil {
		return nil, err
	}

	statsData := strings.Split(string(data), "\n")
	if len(statsData) != 7 {
		return nil, errors.New("could not stat " + name)
	}

	size, err := strconv.Atoi(statsData[0])
	if err != nil {
		return nil, errors.Wrap(err, "could not stat "+name)
	}

	uid, err := strconv.ParseInt(statsData[2], 10, 64)
	if err != nil {
		return nil, errors.Wrap(err, "could not stat "+name)
	}

	gid, err := strconv.ParseInt(statsData[3], 10, 64)
	if err != nil {
		return nil, errors.Wrap(err, "could not stat "+name)
	}

	mask, err := strconv.ParseUint(statsData[1], 16, 32)
	if err != nil {
		return nil, errors.Wrap(err, "could not stat "+name)
	}

	mode := os.FileMode(uint32(mask) & 07777)

	mtime, err := strconv.ParseInt(statsData[4], 10, 64)
	if err != nil {
		return nil, errors.Wrap(err, "could not stat "+name)
	}

	return &types.FileInfo{
		FSize:    int64(size),
		FMode:    mode,
		FIsDir:   mode.IsDir(),
		FModTime: time.Unix(mtime, 0),
		Uid:      uid,
		Gid:      gid,
	}, nil
}

func (cat *CatFs) Create(name string) (afero.File, error) {
	return nil, errors.New("not implemented")
}
func (cat *CatFs) Mkdir(name string, perm os.FileMode) error {
	return errors.New("not implemented")
}
func (cat *CatFs) MkdirAll(path string, perm os.FileMode) error {
	return errors.New("not implemented")
}

func (cat *CatFs) OpenFile(name string, flag int, perm os.FileMode) (afero.File, error) {
	return nil, errors.New("not implemented")
}

func (cat *CatFs) Remove(name string) error {
	return errors.New("not implemented")
}

func (cat *CatFs) RemoveAll(path string) error {
	return errors.New("not implemented")
}

func (cat *CatFs) Rename(oldname, newname string) error {
	return errors.New("not implemented")
}

func (cat *CatFs) Chmod(name string, mode os.FileMode) error {
	return errors.New("not implemented")
}

func (cat *CatFs) Chtimes(name string, atime time.Time, mtime time.Time) error {
	return errors.New("not implemented")
}
