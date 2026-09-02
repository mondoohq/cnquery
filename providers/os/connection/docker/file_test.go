// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package docker

import (
	"testing"

	"github.com/kballard/go-shellquote"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListDirsCommand(t *testing.T) {
	// directory names come from the image, so they can contain anything a
	// filename is allowed to contain
	paths := []string{
		"/etc",
		"/opt/my app",
		`/opt/od'd`,
		`/opt/x'$(id)'`,
		`/opt/"quoted"`,
		"/opt/semi;colon",
		"/opt/back\\slash",
	}

	for _, path := range paths {
		t.Run(path, func(t *testing.T) {
			args, err := shellquote.Split(listDirsCommand(path))
			require.NoError(t, err)
			assert.Equal(t, []string{"find", path, "-maxdepth", "1", "-type", "d"}, args)
		})
	}
}
