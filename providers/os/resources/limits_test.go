// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseLimitsLine_Wildcard(t *testing.T) {
	content := "* soft core 0"
	matches := limitsEntryRegex.FindStringSubmatch(content)

	require.NotNil(t, matches)
	assert.Equal(t, "*", matches[1])    // domain
	assert.Equal(t, "soft", matches[2]) // type
	assert.Equal(t, "core", matches[3]) // item
	assert.Equal(t, "0", matches[4])    // value
}

func TestParseLimitsLine_HardLimit(t *testing.T) {
	content := "* hard rss 10000"
	matches := limitsEntryRegex.FindStringSubmatch(content)

	require.NotNil(t, matches)
	assert.Equal(t, "*", matches[1])     // domain
	assert.Equal(t, "hard", matches[2])  // type
	assert.Equal(t, "rss", matches[3])   // item
	assert.Equal(t, "10000", matches[4]) // value
}

func TestParseLimitsLine_BothType(t *testing.T) {
	content := "@student - maxlogins 4"
	matches := limitsEntryRegex.FindStringSubmatch(content)

	require.NotNil(t, matches)
	assert.Equal(t, "@student", matches[1])  // domain (group)
	assert.Equal(t, "-", matches[2])         // type (both)
	assert.Equal(t, "maxlogins", matches[3]) // item
	assert.Equal(t, "4", matches[4])         // value
}

func TestParseLimitsLine_Username(t *testing.T) {
	content := "john soft nofile 4096"
	matches := limitsEntryRegex.FindStringSubmatch(content)

	require.NotNil(t, matches)
	assert.Equal(t, "john", matches[1])   // domain (user)
	assert.Equal(t, "soft", matches[2])   // type
	assert.Equal(t, "nofile", matches[3]) // item
	assert.Equal(t, "4096", matches[4])   // value
}

func TestParseLimitsLine_GroupName(t *testing.T) {
	content := "@admin hard nproc 50"
	matches := limitsEntryRegex.FindStringSubmatch(content)

	require.NotNil(t, matches)
	assert.Equal(t, "@admin", matches[1]) // domain (group)
	assert.Equal(t, "hard", matches[2])   // type
	assert.Equal(t, "nproc", matches[3])  // item
	assert.Equal(t, "50", matches[4])     // value
}

func TestParseLimitsLine_Unlimited(t *testing.T) {
	content := "root soft core unlimited"
	matches := limitsEntryRegex.FindStringSubmatch(content)

	require.NotNil(t, matches)
	assert.Equal(t, "root", matches[1])      // domain
	assert.Equal(t, "soft", matches[2])      // type
	assert.Equal(t, "core", matches[3])      // item
	assert.Equal(t, "unlimited", matches[4]) // value
}

func TestParseLimitsLine_MemoryLock(t *testing.T) {
	content := "* hard memlock 64"
	matches := limitsEntryRegex.FindStringSubmatch(content)

	require.NotNil(t, matches)
	assert.Equal(t, "*", matches[1])       // domain
	assert.Equal(t, "hard", matches[2])    // type
	assert.Equal(t, "memlock", matches[3]) // item
	assert.Equal(t, "64", matches[4])      // value
}

func TestParseLimitsLine_Stack(t *testing.T) {
	content := "apache soft stack 8192"
	matches := limitsEntryRegex.FindStringSubmatch(content)

	require.NotNil(t, matches)
	assert.Equal(t, "apache", matches[1]) // domain
	assert.Equal(t, "soft", matches[2])   // type
	assert.Equal(t, "stack", matches[3])  // item
	assert.Equal(t, "8192", matches[4])   // value
}

func TestParseLimitsLine_CPUTime(t *testing.T) {
	content := "@users hard cpu 60"
	matches := limitsEntryRegex.FindStringSubmatch(content)

	require.NotNil(t, matches)
	assert.Equal(t, "@users", matches[1]) // domain
	assert.Equal(t, "hard", matches[2])   // type
	assert.Equal(t, "cpu", matches[3])    // item
	assert.Equal(t, "60", matches[4])     // value
}

func TestParseLimitsLine_Priority(t *testing.T) {
	content := "@realtime hard priority 10"
	matches := limitsEntryRegex.FindStringSubmatch(content)

	require.NotNil(t, matches)
	assert.Equal(t, "@realtime", matches[1]) // domain
	assert.Equal(t, "hard", matches[2])      // type
	assert.Equal(t, "priority", matches[3])  // item
	assert.Equal(t, "10", matches[4])        // value
}

func TestParseLimitsLine_Nice(t *testing.T) {
	content := "postgres - nice -10"
	matches := limitsEntryRegex.FindStringSubmatch(content)

	require.NotNil(t, matches)
	assert.Equal(t, "postgres", matches[1]) // domain
	assert.Equal(t, "-", matches[2])        // type
	assert.Equal(t, "nice", matches[3])     // item
	assert.Equal(t, "-10", matches[4])      // value (negative)
}

func TestParseLimitsLine_Comment(t *testing.T) {
	content := "# This is a comment"
	matches := limitsEntryRegex.FindStringSubmatch(content)

	// Comments should not match
	assert.Nil(t, matches)
}

func TestParseLimitsLine_EmptyLine(t *testing.T) {
	content := ""
	matches := limitsEntryRegex.FindStringSubmatch(content)

	// Empty lines should not match
	assert.Nil(t, matches)
}

func TestParseLimitsLine_InvalidFormat(t *testing.T) {
	content := "invalid line without proper format"
	matches := limitsEntryRegex.FindStringSubmatch(content)

	// Invalid format should not match
	assert.Nil(t, matches)
}

func TestParseLimitsLine_ExtraWhitespace(t *testing.T) {
	content := "  *     soft     nofile     65536  "
	matches := limitsEntryRegex.FindStringSubmatch(strings.TrimSpace(content))

	require.NotNil(t, matches)
	assert.Equal(t, "*", matches[1])      // domain
	assert.Equal(t, "soft", matches[2])   // type
	assert.Equal(t, "nofile", matches[3]) // item
	assert.Equal(t, "65536", matches[4])  // value
}

func TestParseLimitsLine_FileSize(t *testing.T) {
	content := "* hard fsize 1000000"
	matches := limitsEntryRegex.FindStringSubmatch(content)

	require.NotNil(t, matches)
	assert.Equal(t, "*", matches[1])       // domain
	assert.Equal(t, "hard", matches[2])    // type
	assert.Equal(t, "fsize", matches[3])   // item
	assert.Equal(t, "1000000", matches[4]) // value
}

func TestParseLimitsLine_DataSegment(t *testing.T) {
	content := "oracle soft data unlimited"
	matches := limitsEntryRegex.FindStringSubmatch(content)

	require.NotNil(t, matches)
	assert.Equal(t, "oracle", matches[1])    // domain
	assert.Equal(t, "soft", matches[2])      // type
	assert.Equal(t, "data", matches[3])      // item
	assert.Equal(t, "unlimited", matches[4]) // value
}

func TestParseLimitsLine_Locks(t *testing.T) {
	content := "database hard locks 2048"
	matches := limitsEntryRegex.FindStringSubmatch(content)

	require.NotNil(t, matches)
	assert.Equal(t, "database", matches[1]) // domain
	assert.Equal(t, "hard", matches[2])     // type
	assert.Equal(t, "locks", matches[3])    // item
	assert.Equal(t, "2048", matches[4])     // value
}

func TestParseLimitsLine_SigPending(t *testing.T) {
	content := "* soft sigpending 1024"
	matches := limitsEntryRegex.FindStringSubmatch(content)

	require.NotNil(t, matches)
	assert.Equal(t, "*", matches[1])          // domain
	assert.Equal(t, "soft", matches[2])       // type
	assert.Equal(t, "sigpending", matches[3]) // item
	assert.Equal(t, "1024", matches[4])       // value
}

func TestParseLimitsLine_MsgQueue(t *testing.T) {
	content := "* soft msgqueue 819200"
	matches := limitsEntryRegex.FindStringSubmatch(content)

	require.NotNil(t, matches)
	assert.Equal(t, "*", matches[1])        // domain
	assert.Equal(t, "soft", matches[2])     // type
	assert.Equal(t, "msgqueue", matches[3]) // item
	assert.Equal(t, "819200", matches[4])   // value
}
