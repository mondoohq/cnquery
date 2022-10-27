package docker_engine

import (
	"bufio"
	"bytes"
	"context"
	"encoding/binary"
	"io"
	"time"

	"github.com/containers/podman/v4/pkg/api/handlers"
	"github.com/containers/podman/v4/pkg/bindings/containers"
	"go.mondoo.com/cnquery/motor/providers/os"

	docker "github.com/docker/docker/api/types"
)

type Command struct {
	os.Command
	Container    string
	podmanClient context.Context
}

func (c *Command) Exec(command string) (*os.Command, error) {
	c.Command.Command = command
	c.Command.Stats.Start = time.Now()

	res, err := containers.ExecCreate(c.podmanClient, c.Container, &handlers.ExecCreateConfig{
		ExecConfig: docker.ExecConfig{
			Cmd:          []string{"/bin/sh", "-c", c.Command.Command},
			Detach:       true,
			Tty:          false,
			AttachStdin:  false,
			AttachStderr: true,
			AttachStdout: true,
		},
	})
	if err != nil {
		return nil, err
	}
	var stdoutBuffer bytes.Buffer
	var stderrBuffer bytes.Buffer

	stdOutWriter := bufio.NewWriter(&stdoutBuffer)
	stdErrWriter := bufio.NewWriter(&stderrBuffer)

	err = containers.ExecStartAndAttach(c.podmanClient, res, &containers.ExecStartAndAttachOptions{
		OutputStream: stdOutWriter,
		ErrorStream:  stdErrWriter,
	})

	if err != nil {
		return nil, err
	}

	// TODO: transformHijack breaks for long stdout, but not if we read stdout/stderr in upfront
	content, err := io.ReadAll(resp.Reader)
	if err != nil {
		return nil, err
	}

	// extract stdout, stderr
	c.transformHijack(bytes.NewReader(content), stdOutWriter, stdErrWriter)

	defer stdOutWriter.Flush()
	defer stdErrWriter.Flush()

	c.Command.Stats.Duration = time.Since(c.Command.Stats.Start)

	info, err := containers.ExecInspect(c.podmanClient, res, &containers.ExecInspectOptions{})
	if err != nil {
		return nil, err
	}
	c.Command.ExitStatus = info.ExitCode

	return &c.Command, nil
}

const (
	STDIN  byte = 0
	STDOUT byte = 1
	STDERR byte = 2
)

// Format is defined in https://docs.docker.com/engine/api/v1.33/#operation/ContainerAttach
func (c *Command) transformHijack(docker io.Reader, stdout io.Writer, stderr io.Writer) {
	header := make([]byte, 8)
	for {
		// read header
		_, err := docker.Read(header)

		// end reached
		if err == io.EOF {
			break
		}

		size := binary.BigEndian.Uint32(header[4:8])
		content := make([]byte, size)
		_, err = docker.Read(content)

		if header[0] == STDIN || header[0] == STDOUT {
			stdout.Write(content)
		} else if header[0] == STDERR {
			stderr.Write(content)
		}

		// end reached
		if err == io.EOF {
			break
		}
	}
}
