package tar

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	docker_types "github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/stretchr/testify/assert"
	"go.mondoo.io/mondoo/motor/motorutil"
	"go.mondoo.io/mondoo/motor/types"
)

func TestTarCommand(t *testing.T) {
	dc, _ := dockerClient()
	err := exportContainer(dc)
	assert.Equal(t, nil, err, "should create tar without error")
	if err != nil {
		return
	}

	filepath, _ := filepath.Abs("./centos-container.tar")
	tarTransport, err := New(&types.Endpoint{Backend: "tar", Path: filepath})
	assert.Equal(t, nil, err, "should create tar without error")

	cmd, err := tarTransport.RunCommand("ls /")
	if assert.NotNil(t, cmd) {
		assert.Equal(t, nil, err, "should execute without error")
		assert.Equal(t, -1, cmd.ExitStatus, "command should not be executed")
		stdoutContent, _ := ioutil.ReadAll(cmd.Stdout)
		assert.Equal(t, "", string(stdoutContent), "output should be correct")
		stderrContent, _ := ioutil.ReadAll(cmd.Stdout)
		assert.Equal(t, "", string(stderrContent), "output should be correct")
	}
}
func TestTarSymlinkFile(t *testing.T) {
	dc, _ := dockerClient()
	err := exportContainer(dc)
	assert.Equal(t, nil, err, "should create tar without error")
	if err != nil {
		return
	}

	filepath, _ := filepath.Abs("./centos-container.tar")
	tarTransport, err := New(&types.Endpoint{Backend: "tar", Path: filepath})
	assert.Equal(t, nil, err, "should create tar without error")

	f, err := tarTransport.File("/etc/redhat-release")

	if assert.NotNil(t, f) {
		assert.Equal(t, nil, err, "should execute without error")

		p := f.Name()
		assert.Equal(t, "/etc/redhat-release", p, "path should be correct")

		stat, err := f.Stat()
		assert.Equal(t, int64(38), stat.Size(), "should read file size")
		assert.Equal(t, nil, err, "should execute without error")

		reader, err := f.Open()
		content, err := ioutil.ReadAll(reader)
		assert.Equal(t, nil, err, "should execute without error")
		assert.Equal(t, 38, len(content), "should read the full content")

		// ensure the same works with tar()
		content, err = motorutil.ReadFile(f)
		assert.Equal(t, nil, err, "should execute without error")
		assert.Equal(t, 38, len(content), "should read the full content")
	}
}
func TestTarFile(t *testing.T) {
	dc, _ := dockerClient()
	err := exportContainer(dc)
	assert.Equal(t, nil, err, "should create tar without error")
	if err != nil {
		return
	}

	filepath, _ := filepath.Abs("./centos-container.tar")
	tarTransport, err := New(&types.Endpoint{Backend: "tar", Path: filepath})
	assert.Equal(t, nil, err, "should create tar without error")

	f, err := tarTransport.File("/etc/centos-release")

	if assert.NotNil(t, f) {
		assert.Equal(t, nil, err, "should execute without error")

		p := f.Name()
		assert.Equal(t, "/etc/centos-release", p, "path should be correct")

		stat, err := f.Stat()
		assert.Equal(t, int64(38), stat.Size(), "should read file size")
		assert.Equal(t, nil, err, "should execute without error")

		reader, err := f.Open()
		content, err := ioutil.ReadAll(reader)
		assert.Equal(t, nil, err, "should execute without error")
		assert.Equal(t, 38, len(content), "should read the full content")
	}
}

func dockerClient() (*client.Client, error) {
	// set docker api version for macos
	os.Setenv("DOCKER_API_VERSION", "1.26")
	// Start new docker container
	return client.NewEnvClient()
}

func exportContainer(dc *client.Client) error {
	image := "centos:7"
	filename := "./centos-container.tar"
	containerName := "centos-container"

	if _, err := os.Stat(filename); err == nil {
		fmt.Println("use cached container file")
		return nil
	}

	ctx := context.TODO()

	// ensure the image is available
	out, err := dc.ImagePull(ctx, image, docker_types.ImagePullOptions{})
	if err != nil {
		return err
	}
	io.Copy(os.Stdout, out)

	// we ignore errors for container kill and remove
	dc.ContainerKill(ctx, containerName, "SIGKILL")
	dc.ContainerRemove(ctx, containerName, docker_types.ContainerRemoveOptions{Force: true})

	// create a new container
	resp, err := dc.ContainerCreate(ctx, &container.Config{
		Image: image,
		Cmd:   []string{},
		Tty:   false,
	}, nil, nil, containerName)
	if err != nil {
		return err
	}

	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	// store container locally
	reader, err := dc.ContainerExport(ctx, resp.ID)
	if err != nil {
		return err
	}

	if _, err := io.Copy(f, reader); err != nil {
		return err
	}
	return nil
}
