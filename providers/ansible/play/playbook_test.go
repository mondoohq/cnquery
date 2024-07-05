// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package play

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"os"
	"testing"
)

func TestPlaybookDecoding(t *testing.T) {
	t.Run("load default playbook", func(t *testing.T) {
		data, err := os.ReadFile("./testdata/play_default.yml")
		require.NoError(t, err)

		playbook, err := DecodePlaybook(data)
		require.NoError(t, err)
		require.NotNil(t, playbook)

		play := playbook[0]
		assert.Equal(t, "webservers", play.Hosts)
		assert.Equal(t, "root", play.RemoteUser)
		assert.Equal(t, 80, play.Vars["http_port"])

		assert.Equal(t, 3, len(play.Tasks))
		assert.Equal(t, "ensure apache is at the latest version", play.Tasks[0].Name)

		action := play.Tasks[0].Action["yum"].(map[string]interface{})
		assert.Equal(t, "httpd", action["name"])

		assert.Equal(t, 1, len(play.Handlers))
		assert.Equal(t, "restart apache", play.Handlers[0].Name)

		handler := play.Handlers[0].Action["service"].(map[string]interface{})
		assert.Equal(t, "httpd", handler["name"])
	})

	t.Run("load playbook with roles", func(t *testing.T) {
		data, err := os.ReadFile("./testdata/play_role.yml")
		require.NoError(t, err)

		playbook, err := DecodePlaybook(data)
		require.NoError(t, err)
		require.NotNil(t, playbook)

		play := playbook[0]
		assert.Equal(t, "webservers", play.Hosts)
		assert.Equal(t, []string{"common", "webservers"}, play.Roles)
	})

	t.Run("load playbook with vars", func(t *testing.T) {
		data, err := os.ReadFile("./testdata/play_vars.yml")
		require.NoError(t, err)

		playbook, err := DecodePlaybook(data)
		require.NoError(t, err)
		require.NotNil(t, playbook)

		play := playbook[0]
		list := play.Vars["list1"]
		assert.Equal(t, []interface{}{"apple", "banana", "fig"}, list)
	})

	t.Run("load playbook with serial", func(t *testing.T) {
		data, err := os.ReadFile("./testdata/play_serial.yml")
		require.NoError(t, err)

		playbook, err := DecodePlaybook(data)
		require.NoError(t, err)
		require.NotNil(t, playbook)

		play := playbook[0]
		assert.Equal(t, 3, play.Serial)
		assert.Equal(t, "False", play.GatherFacts)
	})

	t.Run("load playbook with multiple plays", func(t *testing.T) {
		data, err := os.ReadFile("./testdata/play_multi.yml")
		require.NoError(t, err)

		playbook, err := DecodePlaybook(data)
		require.NoError(t, err)
		require.NotNil(t, playbook)

		assert.Equal(t, 2, len(playbook))
	})

	t.Run("load playbook with blocks and errors", func(t *testing.T) {
		data, err := os.ReadFile("./testdata/play_blocks_errors.yml")
		require.NoError(t, err)

		playbook, err := DecodePlaybook(data)
		require.NoError(t, err)
		require.NotNil(t, playbook)

		play := playbook[0]
		assert.Equal(t, "Accumulate failure", play.Tasks[0].Rescue[0].Name)
		assert.Equal(t, 1, len(play.Tasks[0].Rescue))
	})
}

func TestTaskDecoding(t *testing.T) {
	t.Run("load task with blocks", func(t *testing.T) {
		data, err := os.ReadFile("./testdata/tasks_blocks.yml")
		require.NoError(t, err)

		tasks, err := DecodeTasks(data)
		require.NoError(t, err)
		require.NotNil(t, tasks)

		task := tasks.Tasks[0]
		assert.Equal(t, "install httpd and memcached", task.Block[0].Name)
		assert.Equal(t, 3, len(task.Block))
	})

	t.Run("load task with vars", func(t *testing.T) {
		data, err := os.ReadFile("./testdata/tasks_vars.yml")
		require.NoError(t, err)

		tasks, err := DecodeTasks(data)
		require.NoError(t, err)
		require.NotNil(t, tasks)

		task := tasks.Tasks[0]
		assert.Equal(t, "copy a file from a fileshare with custom credentials", task.Name)
		assert.Equal(t, 1, len(task.Action))
		assert.Equal(t, 5, len(task.Vars))
	})
}
