// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package apache2

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// debianEnvvars is a trimmed version of the envvars file shipped by Debian's
// apache2 package. It includes the real shell control flow and exports so the
// parser is exercised against realistic input.
const debianEnvvars = `
# envvars - default environment variables for apache2ctl

unset HOME

if [ "${APACHE_CONFDIR##/etc/apache2-}" != "${APACHE_CONFDIR}" ] ; then
	SUFFIX="-${APACHE_CONFDIR##/etc/apache2-}"
else
	SUFFIX=
fi

export APACHE_RUN_USER=www-data
export APACHE_RUN_GROUP=www-data

export APACHE_PID_FILE=/var/run/apache2$SUFFIX/apache2.pid
export APACHE_RUN_DIR=/var/run/apache2$SUFFIX
export APACHE_LOCK_DIR=/var/lock/apache2$SUFFIX
export APACHE_LOG_DIR=/var/log/apache2$SUFFIX

export LANG=C
export LANG

# Quoted values
export APACHE_ARGUMENTS="-DFOREGROUND"
`

func TestParseEnvvarsBasic(t *testing.T) {
	vars := ParseEnvvars(debianEnvvars)
	assert.Equal(t, "www-data", vars["APACHE_RUN_USER"])
	assert.Equal(t, "www-data", vars["APACHE_RUN_GROUP"])
	assert.Equal(t, "C", vars["LANG"])
	// Unset SUFFIX expands to empty string (shell default for unknown vars).
	assert.Equal(t, "/var/run/apache2/apache2.pid", vars["APACHE_PID_FILE"])
	assert.Equal(t, "/var/log/apache2", vars["APACHE_LOG_DIR"])
	// Quoted values keep their inner content without the quotes.
	assert.Equal(t, "-DFOREGROUND", vars["APACHE_ARGUMENTS"])
	// Shell control flow is not treated as an assignment.
	assert.NotContains(t, vars, "if")
	assert.NotContains(t, vars, "fi")
}

func TestParseEnvvarsIgnoresCommentsAndBlanks(t *testing.T) {
	vars := ParseEnvvars(`
# pure comment
   # indented comment
FOO=bar # trailing comment
BAZ='qux'
`)
	assert.Equal(t, "bar", vars["FOO"])
	assert.Equal(t, "qux", vars["BAZ"])
}

func TestParseEnvvarsReferencesOtherVars(t *testing.T) {
	vars := ParseEnvvars(`
export BASE=/opt/apache
export LOG=${BASE}/log
export PID=$BASE/apache.pid
`)
	assert.Equal(t, "/opt/apache", vars["BASE"])
	assert.Equal(t, "/opt/apache/log", vars["LOG"])
	assert.Equal(t, "/opt/apache/apache.pid", vars["PID"])
}

// TestParseEnvvarsChainedReferences exercises the fixed-point expansion:
// A depends on B depends on C, and map iteration order can visit them in any
// sequence.
func TestParseEnvvarsChainedReferences(t *testing.T) {
	vars := ParseEnvvars(`
export A=${B}/a
export B=${C}/b
export C=/root
`)
	assert.Equal(t, "/root", vars["C"])
	assert.Equal(t, "/root/b", vars["B"])
	assert.Equal(t, "/root/b/a", vars["A"])
}

// TestParseEnvvarsCyclicReferences ensures the fixed-point loop terminates
// even when the input contains a reference cycle.
func TestParseEnvvarsCyclicReferences(t *testing.T) {
	done := make(chan map[string]string, 1)
	go func() {
		done <- ParseEnvvars(`
export A=${B}
export B=${A}
`)
	}()
	select {
	case <-done:
		// ok — terminated
	case <-time.After(2 * time.Second):
		t.Fatal("ParseEnvvars did not terminate on a reference cycle")
	}
}

// TestParseWithGlobExpandsEnvvars is the regression test for
// https://github.com/mondoohq/mql/issues/7173 — Debian's apache2.conf references
// `${APACHE_RUN_USER}`, which must be resolved from envvars.
func TestParseWithGlobExpandsEnvvars(t *testing.T) {
	files := map[string]string{
		"/etc/apache2/apache2.conf": `
User ${APACHE_RUN_USER}
Group ${APACHE_RUN_GROUP}
ErrorLog ${APACHE_LOG_DIR}/error.log
PidFile ${APACHE_PID_FILE}
`,
	}

	fileContent := func(path string) (string, error) {
		if c, ok := files[path]; ok {
			return c, nil
		}
		return "", &fileNotFoundError{path: path}
	}
	globExpand := func(string) ([]string, error) { return nil, nil }

	envvars := ParseEnvvars(`
export APACHE_RUN_USER=www-data
export APACHE_RUN_GROUP=www-data
export APACHE_LOG_DIR=/var/log/apache2
export APACHE_PID_FILE=/var/run/apache2/apache2.pid
`)

	cfg, err := ParseWithGlob("/etc/apache2/apache2.conf", fileContent, globExpand, envvars)
	require.NoError(t, err)

	assert.Equal(t, "www-data", cfg.Params["User"])
	assert.Equal(t, "www-data", cfg.Params["Group"])
	assert.Equal(t, "/var/log/apache2/error.log", cfg.Params["ErrorLog"])
	assert.Equal(t, "/var/run/apache2/apache2.pid", cfg.Params["PidFile"])
}

func TestParseWithGlobUnresolvedVarsPreserved(t *testing.T) {
	// When we can't resolve a variable, keep the original reference text so
	// users can still see the problem instead of getting a silent empty value.
	files := map[string]string{
		"/etc/apache2/apache2.conf": `User ${APACHE_RUN_USER}`,
	}
	fileContent := func(path string) (string, error) { return files[path], nil }
	globExpand := func(string) ([]string, error) { return nil, nil }

	cfg, err := ParseWithGlob("/etc/apache2/apache2.conf", fileContent, globExpand, nil)
	require.NoError(t, err)
	assert.Equal(t, "${APACHE_RUN_USER}", cfg.Params["User"])
}

func TestParseWithGlobExpandsInVirtualHost(t *testing.T) {
	// ${VAR} references inside <VirtualHost> blocks must also be expanded,
	// since Debian's default site configs rely on that (e.g. SSLCertificateFile
	// ${APACHE_SSL_DIR}/cert.pem).
	files := map[string]string{
		"/etc/apache2/apache2.conf": `
<VirtualHost *:443>
    ServerName ${SITE_HOST}
    DocumentRoot ${SITE_ROOT}
    SSLEngine on
</VirtualHost>
`,
	}
	fileContent := func(path string) (string, error) { return files[path], nil }
	globExpand := func(string) ([]string, error) { return nil, nil }

	vars := map[string]string{
		"SITE_HOST": "secure.example.com",
		"SITE_ROOT": "/var/www/secure",
	}

	cfg, err := ParseWithGlob("/etc/apache2/apache2.conf", fileContent, globExpand, vars)
	require.NoError(t, err)
	require.Len(t, cfg.VHosts, 1)

	vh := cfg.VHosts[0]
	assert.Equal(t, "secure.example.com", vh.ServerName)
	assert.Equal(t, "/var/www/secure", vh.DocumentRoot)
	assert.Equal(t, "secure.example.com", vh.Params["ServerName"])
}

func TestParseWithGlobApacheDefineDirective(t *testing.T) {
	// Apache's own Define directive creates variables usable as ${NAME}.
	files := map[string]string{
		"/etc/apache2/apache2.conf": `
Define SITE_ROOT /var/www/site
DocumentRoot ${SITE_ROOT}
`,
	}
	fileContent := func(path string) (string, error) { return files[path], nil }
	globExpand := func(string) ([]string, error) { return nil, nil }

	cfg, err := ParseWithGlob("/etc/apache2/apache2.conf", fileContent, globExpand, nil)
	require.NoError(t, err)
	assert.Equal(t, "/var/www/site", cfg.Params["DocumentRoot"])
}
