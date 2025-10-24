// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.mondoo.com/cnquery/v12/providers-sdk/v1/testutils"
)

func TestResource_AuditdConfig(t *testing.T) {
	x.TestSimpleErrors(t, []testutils.SimpleTest{
		{
			Code:        "auditd.config('nopath').params",
			ResultIndex: 0,
			Expectation: "file 'nopath' not found",
		},
	})

	t.Run("auditd file path", func(t *testing.T) {
		res := x.TestQuery(t, "auditd.config.file.path")
		assert.NotEmpty(t, res)
		assert.NoError(t, res[0].Data.Error)
	})

	t.Run("auditd params", func(t *testing.T) {
		res := x.TestQuery(t, "auditd.config.params")
		assert.NotEmpty(t, res)
		assert.NoError(t, res[0].Data.Error)
	})

	t.Run("auditd is downcasing relevant params", func(t *testing.T) {
		res := x.TestQuery(t, "auditd.config.params.log_format")
		assert.NotEmpty(t, res)
		assert.NoError(t, res[0].Data.Error)
		assert.Equal(t, "enriched", res[0].Data.Value)
	})

	t.Run("auditd is NOT downcasing other params", func(t *testing.T) {
		res := x.TestQuery(t, "auditd.config.params.log_file")
		assert.NotEmpty(t, res)
		assert.NoError(t, res[0].Data.Error)
		assert.Equal(t, "/var/log/audit/AuDiT.log", res[0].Data.Value)
	})
}

func TestResource_AuditdRules(t *testing.T) {
	t.Run("auditd rules path", func(t *testing.T) {
		x.TestSimple(t, []testutils.SimpleTest{
			{
				Code:        "auditd.rules.path",
				ResultIndex: 0,
				Expectation: "/etc/audit/rules.d",
			},
			{
				Code:        "auditd.rules.files.first.path",
				ResultIndex: 0,
				Expectation: "/etc/sudoers",
			},
			{
				Code:        "auditd.rules.controls[0].flag",
				ResultIndex: 0,
				Expectation: "-D",
			},
			{
				Code:        "auditd.rules.syscalls.where(action==\"always\" && fields.contains(key==\"path\" && value==\"/usr/bin/systemd-run\")).length",
				ResultIndex: 0,
				Expectation: int64(2),
			},
		})
	})

	t.Run("auditd rules default source", func(t *testing.T) {
		res := x.TestQuery(t, "auditd.rules.source")
		assert.NotEmpty(t, res)
		assert.NoError(t, res[0].Data.Error)
		assert.Equal(t, "both", res[0].Data.Value)
	})

	t.Run("auditd rules with explicit filesystem source", func(t *testing.T) {
		x.TestSimple(t, []testutils.SimpleTest{
			{
				Code:        "auditd.rules(source: \"filesystem\").source",
				ResultIndex: 0,
				Expectation: "filesystem",
			},
			{
				Code:        "auditd.rules(source: \"filesystem\").files.first.path",
				ResultIndex: 0,
				Expectation: "/etc/sudoers",
			},
		})
	})

	t.Run("auditd rules backward compatibility", func(t *testing.T) {
		// Test that existing queries without source parameter still work
		res := x.TestQuery(t, "auditd.rules.files.length")
		assert.NotEmpty(t, res)
		assert.NoError(t, res[0].Data.Error)
	})
}

func TestResource_AuditdRules_SourceParameter(t *testing.T) {
	t.Run("invalid source parameter", func(t *testing.T) {
		x.TestSimpleErrors(t, []testutils.SimpleTest{
			{
				Code:        "auditd.rules(source: \"invalid\").files",
				ResultIndex: 0,
				Expectation: "source must be 'filesystem', 'runtime', or 'both'",
			},
		})
	})

	t.Run("source parameter values", func(t *testing.T) {
		x.TestSimple(t, []testutils.SimpleTest{
			{
				Code:        "auditd.rules(source: \"filesystem\").source",
				ResultIndex: 0,
				Expectation: "filesystem",
			},
			{
				Code:        "auditd.rules(source: \"both\").source",
				ResultIndex: 0,
				Expectation: "both",
			},
		})
	})
}
