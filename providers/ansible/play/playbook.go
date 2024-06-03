// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package play

import (
	"gopkg.in/yaml.v3"
)

// Playbook is a collection of plays
type Playbook []*Play

// Play is a collection of tasks to be executed
// see https://docs.ansible.com/ansible/latest/reference_appendices/playbooks_keywords.html
type Play struct {
	// Name is the name of the play
	Name string `yaml:"name,omitempty"`

	// Hosts is a pattern that matches hosts
	// see https://docs.ansible.com/ansible/latest/inventory_guide/intro_patterns.html
	Hosts interface{} `yaml:"hosts"`

	// RemoteUser sets the user to use for the connection
	// see https://docs.ansible.com/ansible/latest/inventory_guide/connection_details.html
	RemoteUser string `yaml:"remote_user,omitempty"`

	// Become sets to true to activate privilege escalation.
	// see https://docs.ansible.com/ansible/latest/playbook_guide/playbooks_privilege_escalation.html#become
	Become bool `yaml:"become,omitempty"`

	// BecomeUser sets to user with desired privileges
	// see https://docs.ansible.com/ansible/latest/playbook_guide/playbooks_privilege_escalation.html#become
	BecomeUser string `yaml:"become_user,omitempty"`

	// BecomeMethod overrides the default method of privilege escalation
	// see https://docs.ansible.com/ansible/latest/playbook_guide/playbooks_privilege_escalation.html#become
	BecomeMethod string `yaml:"become_method,omitempty"`

	// BecomeFlags permits the use of specific flags
	// see https://docs.ansible.com/ansible/latest/playbook_guide/playbooks_privilege_escalation.html#become
	BecomeFlags string `yaml:"become_flags,omitempty"`

	// Serial sets the batch size
	// see https://docs.ansible.com/ansible/latest/playbook_guide/playbooks_strategies.html#setting-the-batch-size-with-serial
	Serial interface{} `yaml:"serial,omitempty"` // Can be an integer or a string

	// Playbook execution strategy
	// see https://docs.ansible.com/ansible/latest/playbook_guide/playbooks_strategies.html
	Strategy string `yaml:"strategy,omitempty"`

	// MaxFailPercentage sets a maximum failure percentage
	// see https://docs.ansible.com/ansible/latest/playbook_guide/playbooks_error_handling.html#maximum-failure-percentage
	MaxFailPercentage int `yaml:"max_fail_percentage,omitempty"`

	// IgnoreUnreachable sets to true to ignore unreachable hosts
	// see https://docs.ansible.com/ansible/latest/playbook_guide/playbooks_error_handling.html#ignore-unreachable
	IgnoreUnreachable bool `yaml:"ignore_unreachable,omitempty"`

	// AnyErrorsFatal finishes the fatal task on all hosts in the current batch
	// see https://docs.ansible.com/ansible/latest/playbook_guide/playbooks_error_handling.html#aborting-on-the-first-error-any-errors-fatal
	AnyErrorsFatal bool `yaml:"any_errors_fatal,omitempty"`

	// Vars are variables to be used in the play
	// see https://docs.ansible.com/ansible/latest/playbook_guide/playbooks_variables.html
	Vars map[string]interface{} `yaml:"vars,omitempty"`

	// Roles are a list of roles to be applied to the play
	// see https://docs.ansible.com/ansible/latest/playbook_guide/playbooks_reuse_roles.html
	Roles []string `yaml:"roles,omitempty"`

	// Tasks are a list of tasks to be executed
	// see https://docs.ansible.com/ansible/latest/playbook_guide/playbooks_intro.html#id4
	Tasks []*Task `yaml:"tasks"`

	// Handlers are tasks that only run when notified
	Handlers []*Handler `yaml:"handlers,omitempty"`

	GatherFacts string `yaml:"gather_facts,omitempty"`
}

// Tasks is a list of tasks to be executed
type Tasks struct {
	// Tasks are a list of tasks to be executed
	// see https://docs.ansible.com/ansible/latest/playbook_guide/playbooks_intro.html#id4
	Tasks []*Task `yaml:"tasks"`
}

// Task is a single task to be executed
// see https://docs.ansible.com/ansible/latest/reference_appendices/playbooks_keywords.html#task
type Task struct {
	// Name is the name of the task
	Name string `yaml:"name,omitempty"`

	// Action is the module to be executed
	Action map[string]interface{} `yaml:",inline"` // Use inline to handle dynamic task modules

	// Vars are variables to be used in the play
	// see https://docs.ansible.com/ansible/latest/playbook_guide/playbooks_variables.html
	Vars map[string]interface{} `yaml:"vars,omitempty"`

	// see https://docs.ansible.com/ansible/latest/playbook_guide/playbooks_tags.html
	Tags []string `yaml:"tags,omitempty"`

	// Register is a variable to store the result of the task
	// see https://docs.ansible.com/ansible/latest/playbook_guide/playbooks_variables.html#registering-variables
	Register string `yaml:"register,omitempty"`

	// Conditional statement to execute the task
	// see https://docs.ansible.com/ansible/latest/playbook_guide/playbooks_conditionals.html#basic-conditionals-with-when
	When string `yaml:"when,omitempty"`

	// Failed condition
	// see https://docs.ansible.com/ansible/latest/playbook_guide/playbooks_error_handling.html#defining-failure
	FailedWhen string `yaml:"failed_when,omitempty"`

	// Changed condition
	// see https://docs.ansible.com/ansible/latest/playbook_guide/playbooks_error_handling.html#defining-changed
	ChangedWhen string `yaml:"changed_when,omitempty"`

	// Notify is a list of handlers to notify
	// see https://docs.ansible.com/ansible/latest/playbook_guide/playbooks_handlers.html
	Notify []string `yaml:"notify,omitempty"`

	// Importing Playbooks
	// see https://docs.ansible.com/ansible/2.9/user_guide/playbooks_reuse_includes.html
	ImportPlaybook string `yaml:"import_playbook,omitempty"`

	// Include Playbooks
	// see https://docs.ansible.com/ansible/2.9/user_guide/playbooks_reuse_includes.html
	IncludePlaybook string `yaml:"include_playbook,omitempty"`

	// Import statements are pre-processed at the time playbooks are parsed
	// see https://docs.ansible.com/ansible/2.9/user_guide/playbooks_reuse_includes.html
	ImportTasks string `yaml:"import_tasks,omitempty"`

	// Include statements are processed at the time the play is executed
	// see https://docs.ansible.com/ansible/2.9/user_guide/playbooks_reuse_includes.html
	IncludeTasks string `yaml:"include_tasks,omitempty"`

	// Task grouping with blocks
	// see https://docs.ansible.com/ansible/latest/playbook_guide/playbooks_blocks.html
	Block []*Task `yaml:"block,omitempty"`

	// Handle error in block
	// see https://docs.ansible.com/ansible/latest/playbook_guide/playbooks_blocks.html
	Rescue []*Task `yaml:"rescue,omitempty"`

	// Always runs regardless of the results of the block
	// see https://docs.ansible.com/ansible/latest/playbook_guide/playbooks_blocks.html
	Always []*Task `yaml:"always,omitempty"`
}

// Handler is a task that only runs when notified
// see https://docs.ansible.com/ansible/latest/playbook_guide/playbooks_handlers.html
type Handler struct {
	// Name is the name of the handler
	Name string `yaml:"name,omitempty"`
	// Action is the module to be executed
	Action map[string]interface{} `yaml:",inline"` // Use inline to handle dynamic handler modules
}

func DecodeTasks(data []byte) (Tasks, error) {
	var tasks Tasks
	err := yaml.Unmarshal(data, &tasks)
	if err != nil {
		return tasks, err
	}
	return tasks, nil
}

func DecodePlaybook(data []byte) (Playbook, error) {
	var playbook Playbook
	err := yaml.Unmarshal(data, &playbook)
	if err != nil {
		return nil, err
	}
	return playbook, nil
}
