package docker_engine

import (
	"bufio"
	"bytes"
	"context"
	"encoding/binary"
	"io"
	"io/ioutil"
	"time"

	docker "github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"go.mondoo.io/mondoo/motor/transports"
)

type Command struct {
	transports.Command
	Container    string
	dockerClient *client.Client
}

func (c *Command) Exec(command string) (*transports.Command, error) {
	c.Command.Command = command
	c.Command.Stats.Start = time.Now()

	res, err := c.dockerClient.ContainerExecCreate(context.Background(), c.Container, docker.ExecConfig{
		Cmd:          []string{"/bin/sh", "-c", c.Command.Command},
		Detach:       true,
		Tty:          false,
		AttachStdin:  false,
		AttachStderr: true,
		AttachStdout: true,
	})
	if err != nil {
		return nil, err
	}

	resp, err := c.dockerClient.ContainerExecAttach(context.Background(), res.ID, docker.ExecStartCheck{
		Detach: false,
		Tty:    false,
	})
	if err != nil {
		return nil, err
	}

	// TODO: transformHijack breaks for long stdout, but not if we read stdout/stderr in upfront
	content, err := ioutil.ReadAll(resp.Reader)

	var stdoutBuffer bytes.Buffer
	var stderrBuffer bytes.Buffer

	// create buffered stream
	c.Command.Stdout = &stdoutBuffer
	c.Command.Stderr = &stderrBuffer

	stdOutWriter := bufio.NewWriter(&stdoutBuffer)
	stdErrWriter := bufio.NewWriter(&stderrBuffer)

	// extract stdout, stderr
	c.transformHijack(bytes.NewReader(content), stdOutWriter, stdErrWriter)

	defer stdOutWriter.Flush()
	defer stdErrWriter.Flush()

	c.Command.Stats.Duration = time.Since(c.Command.Stats.Start)

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
