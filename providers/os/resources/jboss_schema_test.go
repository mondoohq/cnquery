// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/mql"
	"go.mondoo.com/mql/mqlc"
	"go.mondoo.com/mql/providers-sdk/v1/testutils"
)

// jbossCompilerConfig builds a compiler bound to the same schema a scan uses,
// so a checksum computed here is the one a score would be keyed by.
func jbossCompilerConfig() mqlc.CompilerConfig {
	core := testutils.MustLoadSchema(testutils.SchemaProvider{Provider: "core"})
	os := testutils.MustLoadSchema(testutils.SchemaProvider{Provider: "os"})
	return mqlc.NewConfig(core.Add(os), mql.Features{byte(mql.ResourceContext)})
}

// TestJbossAuditLogPartsAreIndependentlyAssertable is the regression for the
// reason the audit log is modeled attribute by attribute rather than as one
// "auditing is on" boolean.
//
// A score is keyed by the checksum of the compiled query, and the name a
// variable is given is not part of that checksum — only the shape of the
// expression is. Two checks whose bodies compile identically therefore resolve
// to a single score, and the one written first disappears from the report with
// no error.
//
// Hardening guidance for this server states more than a dozen separate
// requirements — that the event type is recorded, that boot-time operations
// are covered, that reads are covered, that the record carries a timestamp,
// that it lands somewhere durable, that it is shipped off the host — and every
// one of them hangs off the same switch. If the switch were all this resource
// exposed, those checks could only be written one way, and all but one of them
// would vanish. Each has to be able to assert the switch *and* the part of the
// block its own evidence depends on, which is what the fields below exist for.
func TestJbossAuditLogPartsAreIndependentlyAssertable(t *testing.T) {
	conf := jbossCompilerConfig()

	// Each entry asserts the switch plus one further condition that a distinct
	// requirement genuinely depends on. None of them can pass while auditing is
	// off, and no two of them are the same assertion.
	queries := []string{
		`jboss.management.auditLog.enabled == true
		 jboss.management.auditLog.handlers.where(type == "file").length > 0`,
		`jboss.management.auditLog.enabled == true
		 jboss.management.auditLog.logger.logBoot == true`,
		`jboss.management.auditLog.enabled == true
		 jboss.management.auditLog.logger.logReadOnly == true`,
		`jboss.management.auditLog.enabled == true
		 jboss.management.auditLog.logger.handlers.length > 0`,
		`jboss.management.auditLog.enabled == true
		 jboss.management.auditLog.formatters.length > 0`,
		`jboss.management.auditLog.enabled == true
		 jboss.management.auditLog.formatters.all(includeDate == true)`,
		`jboss.management.auditLog.enabled == true
		 jboss.management.auditLog.formatters.all(compact == false)`,
		`jboss.management.auditLog.enabled == true
		 jboss.management.auditLog.formatters.all(escapeControlCharacters == true)`,
		`jboss.management.auditLog.enabled == true
		 jboss.management.auditLog.formatters.all(name != "")`,
		`jboss.management.auditLog.enabled == true
		 jboss.management.auditLog.handlers.all(name != "")`,
		`jboss.management.auditLog.enabled == true
		 jboss.management.auditLog.handlers.all(formatter != "")`,
		`jboss.management.auditLog.enabled == true
		 jboss.management.auditLog.handlers.where(type == "file").all(path != "")`,
		`jboss.management.auditLog.enabled == true
		 jboss.management.auditLog.handlers.where(type == "file").all(relativeTo != "")`,
		`jboss.management.auditLog.enabled == true
		 jboss.management.auditLog.handlers.where(type == "file").all(rotateAtStartup == false)`,
		`jboss.management.auditLog.enabled == true
		 jboss.management.auditLog.handlers.all(maxFailureCount > 0)`,
		`jboss.management.auditLog.enabled == true
		 jboss.management.auditLog.handlers.where(type == "syslog").length > 0`,
		`jboss.management.auditLog.enabled == true
		 jboss.management.auditLog.handlers.where(type == "syslog").all(transport == "tcp")`,
		`jboss.management.auditLog.enabled == true
		 jboss.management.auditLog.serverLogger.logReadOnly == true`,
	}

	checksums := map[string]int{}
	for i, query := range queries {
		bundle, err := mqlc.Compile(query, nil, conf)
		require.NoError(t, err, "query %d", i)
		require.NotNil(t, bundle)
		checksums[bundle.CodeV2.Id]++
	}

	duplicates := []string{}
	for checksum, count := range checksums {
		if count > 1 {
			duplicates = append(duplicates, checksum)
		}
	}

	assert.Empty(t, duplicates,
		"two of these compile to the same code and would collapse into one score")
	assert.Len(t, checksums, len(queries))
}

// Renaming a variable does not change the compiled code, which is why the
// distinctness above has to come from the assertions themselves rather than
// from how they are written.
func TestJbossVariableNamesDoNotChangeTheChecksum(t *testing.T) {
	conf := jbossCompilerConfig()

	first, err := mqlc.Compile(
		"auditLog = jboss.management.auditLog\nauditLog.enabled == true", nil, conf)
	require.NoError(t, err)

	second, err := mqlc.Compile(
		"log = jboss.management.auditLog\nlog.enabled == true", nil, conf)
	require.NoError(t, err)

	assert.Equal(t, first.CodeV2.Id, second.CodeV2.Id,
		"the variable name is not part of the checksum")
}
