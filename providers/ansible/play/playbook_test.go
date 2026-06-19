// Copyright Mondoo, Inc. 2024, 2026
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

		action := play.Tasks[0].Action["yum"].(map[string]any)
		assert.Equal(t, "httpd", action["name"])

		assert.Equal(t, 1, len(play.Handlers))
		assert.Equal(t, "restart apache", play.Handlers[0].Name)

		handler := play.Handlers[0].Action["service"].(map[string]any)
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
		assert.Equal(t, []string{"common", "webservers"}, play.RoleNames())
	})

	t.Run("load playbook with parameterized roles", func(t *testing.T) {
		playbook, err := DecodePlaybook([]byte(`---
- hosts: web
  roles:
    - common
    - role: nginx
      when: ansible_os_family == "Debian"
      tags: [web, frontend]
      http_port: 8080
`))
		require.NoError(t, err)

		apps := playbook[0].RoleApplications()
		require.Len(t, apps, 2)
		assert.Equal(t, "common", apps[0].Name)
		assert.Equal(t, "nginx", apps[1].Name)
		assert.Equal(t, `ansible_os_family == "Debian"`, apps[1].When)
		assert.Equal(t, []string{"web", "frontend"}, apps[1].Tags)
		assert.Equal(t, 8080, apps[1].Vars["http_port"])
	})

	t.Run("load playbook with vars", func(t *testing.T) {
		data, err := os.ReadFile("./testdata/play_vars.yml")
		require.NoError(t, err)

		playbook, err := DecodePlaybook(data)
		require.NoError(t, err)
		require.NotNil(t, playbook)

		play := playbook[0]
		list := play.Vars["list1"]
		assert.Equal(t, []any{"apple", "banana", "fig"}, list)
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
		// `always` on the outer block contains a single nested block task
		require.Equal(t, 1, len(play.Tasks[0].Always))
		assert.Equal(t, "Tasks that will always run after the main block", play.Tasks[0].Always[0].Name)
		assert.Equal(t, 4, len(play.Tasks[0].Always[0].Block))
	})

	t.Run("load playbook with tags, pre/post_tasks, loop, and always", func(t *testing.T) {
		data, err := os.ReadFile("./testdata/play_tags_loop.yml")
		require.NoError(t, err)

		playbook, err := DecodePlaybook(data)
		require.NoError(t, err)
		require.NotNil(t, playbook)

		p := playbook[0]
		assert.Equal(t, []string{"web", "provisioning"}, p.Tags)

		require.Equal(t, 1, len(p.PreTasks))
		assert.Equal(t, "Wait for the host to come online", p.PreTasks[0].Name)
		assert.Equal(t, []string{"bootstrap"}, p.PreTasks[0].Tags)

		require.Equal(t, 1, len(p.PostTasks))
		assert.Equal(t, "Notify monitoring system", p.PostTasks[0].Name)
		assert.Equal(t, []string{"notify"}, p.PostTasks[0].Tags)

		require.GreaterOrEqual(t, len(p.Tasks), 2)

		installTask := p.Tasks[0]
		assert.Equal(t, "Install packages", installTask.Name)
		assert.Equal(t, []string{"packages"}, installTask.Tags)
		assert.Equal(t, []any{"httpd", "memcached"}, installTask.Loop)
		assert.Equal(t, "pkg", installTask.LoopControl["loop_var"])
		assert.Equal(t, "{{ pkg }}", installTask.LoopControl["label"])

		blockTask := p.Tasks[1]
		require.Equal(t, 1, len(blockTask.Block))
		require.Equal(t, 1, len(blockTask.Rescue))
		require.Equal(t, 1, len(blockTask.Always))
		assert.Equal(t, "Reload firewalld", blockTask.Always[0].Name)
	})

	t.Run("load playbook with task-level security attributes", func(t *testing.T) {
		data, err := os.ReadFile("./testdata/play_security.yml")
		require.NoError(t, err)

		playbook, err := DecodePlaybook(data)
		require.NoError(t, err)
		require.NotNil(t, playbook)

		p := playbook[0]
		require.Equal(t, 2, len(p.Tasks))

		privileged := p.Tasks[0]
		assert.True(t, privileged.Become)
		assert.Equal(t, "root", privileged.BecomeUser)
		assert.Equal(t, "sudo", privileged.BecomeMethod)
		assert.Equal(t, "-H -S -n", privileged.BecomeFlags)
		assert.Equal(t, "localhost", privileged.DelegateTo)
		assert.Equal(t, true, privileged.NoLog)
		assert.Equal(t, true, privileged.RunOnce)
		assert.Equal(t, "http://proxy.example.com:8080", privileged.Environment["HTTP_PROXY"])

		// A templated string for ignore_errors must not break parsing.
		templated := p.Tasks[1]
		assert.Equal(t, "{{ ansible_check_mode }}", templated.IgnoreErrors)
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

func TestMaxFailPercentageValue(t *testing.T) {
	tests := []struct {
		name string
		raw  any
		want int64
	}{
		{"absent", nil, 0},
		{"plain int", 30, 30},
		{"int64", int64(45), 45},
		{"float", 33.3, 33},
		{"percent string", "30%", 30},
		{"percent string with spaces", " 50 % ", 50},
		{"bare numeric string", "20", 20},
		{"empty string", "", 0},
		{"garbage string", "abc", 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Play{MaxFailPercentage: tt.raw}
			assert.Equal(t, tt.want, p.MaxFailPercentageValue())
		})
	}
}

// A percent-suffixed max_fail_percentage is valid Ansible (the field is typed
// as a percent). It must not fail the whole playbook decode the way it would
// when the struct field was a plain int.
func TestMaxFailPercentagePercentStringDecodes(t *testing.T) {
	data := []byte("- name: percent play\n  hosts: all\n  max_fail_percentage: 30%\n")
	pb, err := DecodePlaybook(data)
	require.NoError(t, err)
	require.Len(t, pb, 1)
	assert.Equal(t, int64(30), pb[0].MaxFailPercentageValue())
}
