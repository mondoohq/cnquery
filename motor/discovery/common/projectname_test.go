package common

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestProjectName(t *testing.T) {
	// absolute path where the file does not exist locally
	assert.Equal(t, "projectname", ProjectNameFromPath("/testdata/terraform/projectname"))
	assert.Equal(t, "file", ProjectNameFromPath("/testdata/terraform/projectname/file.tf"))
	assert.Equal(t, "manifest", ProjectNameFromPath("/testdata/terraform/projectname/manifest.yaml"))
	// relative path where the file does not exist locally
	assert.Equal(t, "manifest", ProjectNameFromPath("./projectname/manifest.yaml"))
	assert.Equal(t, "manifest", ProjectNameFromPath("./manifest.yaml"))
	// if we get a dot, use the current directory since . does not make any sense
	assert.Equal(t, "common", ProjectNameFromPath("."))
}
