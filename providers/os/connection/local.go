package connection

import (
	"bytes"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/spf13/afero"
)

type LocalConnection struct {
	shell   []string
	fs      afero.Fs
	Sudo    *Sudo
	runtime string
	id      uint32
}

func NewLocalConnection(id uint32) *LocalConnection {
	// expect unix shell by default
	res := LocalConnection{
		id: id,
	}

	if runtime.GOOS == "windows" {
		// It does not make any sense to use cmd as default shell
		// shell = []string{"cmd", "/C"}
		res.shell = []string{"powershell", "-c"}
	} else {
		res.shell = []string{"sh", "-c"}
	}

	return &res
}

func (p *LocalConnection) ID() uint32 {
	return p.id
}

func (p *LocalConnection) Type() ConnectionType {
	return Local
}

func (p *LocalConnection) RunCommand(command string) (*Command, error) {
	log.Debug().Msgf("local> run command %s", command)
	if p.Sudo != nil {
		command = p.Sudo.Build(command)
	}
	c := &commandRunner{Shell: p.shell}
	args := []string{}

	res, err := c.Exec(command, args)
	return res, err
}

func (p *LocalConnection) FileSystem() afero.Fs {
	if p.fs != nil {
		return p.fs
	}

	if p.Sudo != nil {
		// p.fs = cat.New(p)
		panic("NOT MIGRATED")
	} else {
		p.fs = afero.NewOsFs()
	}

	return p.fs
}

func (p *LocalConnection) FileInfo(path string) (FileInfoDetails, error) {
	fs := p.FileSystem()
	afs := &afero.Afero{Fs: fs}
	stat, err := afs.Stat(path)
	if err != nil {
		return FileInfoDetails{}, err
	}

	uid, gid := p.fileowner(stat)

	mode := stat.Mode()
	return FileInfoDetails{
		Mode: FileModeDetails{mode},
		Size: stat.Size(),
		Uid:  uid,
		Gid:  gid,
	}, nil
}

type FileInfo struct {
	FName    string
	FSize    int64
	FIsDir   bool
	FModTime time.Time
	FMode    os.FileMode
	Uid      int64
	Gid      int64
}

func (f *FileInfo) Name() string {
	return f.FName
}

func (f *FileInfo) Size() int64 {
	return f.FSize
}

func (f *FileInfo) Mode() os.FileMode {
	return f.FMode
}

func (f *FileInfo) ModTime() time.Time {
	return f.FModTime
}

func (f *FileInfo) IsDir() bool {
	return f.FIsDir
}

func (f *FileInfo) Sys() interface{} {
	return f
}

func (p *LocalConnection) Close() {
	// TODO: we need to close all commands and file handles
}

type commandRunner struct {
	Command
	cmdExecutor *exec.Cmd
	Shell       []string
}

func (c *commandRunner) Exec(usercmd string, args []string) (*Command, error) {
	c.Command.Stats.Start = time.Now()

	var cmd string
	cmdArgs := []string{}

	if len(c.Shell) > 0 {
		shellCommand, shellArgs := c.Shell[0], c.Shell[1:]
		cmd = shellCommand
		cmdArgs = append(cmdArgs, shellArgs...)
		cmdArgs = append(cmdArgs, usercmd)
	} else {
		cmd = usercmd
	}
	cmdArgs = append(cmdArgs, args...)

	// this only stores the user command, not the shell
	c.Command.Command = usercmd + " " + strings.Join(args, " ")
	c.cmdExecutor = exec.Command(cmd, cmdArgs...)

	var stdoutBuffer bytes.Buffer
	var stderrBuffer bytes.Buffer

	// create buffered stream
	c.Command.Stdout = &stdoutBuffer
	c.Command.Stderr = &stderrBuffer

	c.cmdExecutor.Stdout = c.Command.Stdout
	c.cmdExecutor.Stderr = c.Command.Stderr

	err := c.cmdExecutor.Run()
	c.Command.Stats.Duration = time.Since(c.Command.Stats.Start)

	// command completed successfully, great :-)
	if err == nil {
		return &c.Command, nil
	}

	// if the program failed, we do not return err but its exit code
	if exiterr, ok := err.(*exec.ExitError); ok {
		if status, ok := exiterr.Sys().(syscall.WaitStatus); ok {
			c.Command.ExitStatus = status.ExitStatus()
		}
		return &c.Command, nil
	}

	// all other errors are real errors and not expected
	return &c.Command, err
}
