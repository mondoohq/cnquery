// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package snmpd

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParse(t *testing.T) {
	content := `# /etc/snmp/snmpd.conf
   # indented comment
agentAddress  udp:161,udp6:[::1]:161

rocommunity  public  10.0.0.0/24 -V systemonly
rocommunity6 public6
rwcommunity  private             # inline comment after the directive
rouser   authOnlyUser
rwuser   adminUser priv

includeDir /etc/snmp/snmpd.conf.d
`

	directives := Parse(content)

	t.Run("comments and blank lines are skipped", func(t *testing.T) {
		for _, d := range directives {
			assert.NotEqual(t, "#", d.Keyword)
		}
		require.Len(t, directives, 7)
	})

	t.Run("keyword and args are tokenized", func(t *testing.T) {
		assert.Equal(t, "agentAddress", directives[0].Keyword)
		assert.Equal(t, []string{"udp:161,udp6:[::1]:161"}, directives[0].Args)
		assert.Equal(t, 3, directives[0].Line)
	})

	t.Run("trailing arguments are preserved", func(t *testing.T) {
		assert.Equal(t, "rocommunity", directives[1].Keyword)
		assert.Equal(t, []string{"public", "10.0.0.0/24", "-V", "systemonly"}, directives[1].Args)
	})

	t.Run("inline comment is stripped", func(t *testing.T) {
		assert.Equal(t, "rwcommunity", directives[3].Keyword)
		assert.Equal(t, []string{"private"}, directives[3].Args)
	})

	t.Run("includeDir is a normal directive", func(t *testing.T) {
		last := directives[len(directives)-1]
		assert.Equal(t, "includeDir", last.Keyword)
		assert.Equal(t, []string{"/etc/snmp/snmpd.conf.d"}, last.Args)
	})
}

func TestParseQuoted(t *testing.T) {
	directives := Parse(`rocommunity "my secret community" 127.0.0.1`)
	require.Len(t, directives, 1)
	assert.Equal(t, "rocommunity", directives[0].Keyword)
	assert.Equal(t, []string{"my secret community", "127.0.0.1"}, directives[0].Args)
}

func TestParseHashInQuotesIsNotComment(t *testing.T) {
	directives := Parse(`rocommunity "pa#ss"`)
	require.Len(t, directives, 1)
	assert.Equal(t, []string{"pa#ss"}, directives[0].Args)
}

func TestParseEmpty(t *testing.T) {
	assert.Empty(t, Parse(""))
	assert.Empty(t, Parse("# only comments\n\n   \n"))
}
