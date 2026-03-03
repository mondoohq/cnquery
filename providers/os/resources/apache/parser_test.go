// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package apache

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const basicConfig = `
# Basic Apache configuration
ServerRoot "/etc/httpd"
ServerName www.example.com
ServerAdmin admin@example.com
ServerTokens Prod
ServerSignature Off
TraceEnable Off
Timeout 60
KeepAlive On
MaxKeepAliveRequests 100
KeepAliveTimeout 5
User apache
Group apache

Listen 80
Listen 443

DocumentRoot "/var/www/html"

LogLevel warn
ErrorLog "logs/error_log"

LoadModule ssl_module modules/mod_ssl.so
LoadModule rewrite_module modules/mod_rewrite.so
LoadModule headers_module modules/mod_headers.so
`

func TestParseBasicDirectives(t *testing.T) {
	cfg := Parse(basicConfig)
	require.NotNil(t, cfg)

	assert.Equal(t, "www.example.com", cfg.Params["ServerName"])
	assert.Equal(t, "admin@example.com", cfg.Params["ServerAdmin"])
	assert.Equal(t, "Prod", cfg.Params["ServerTokens"])
	assert.Equal(t, "Off", cfg.Params["ServerSignature"])
	assert.Equal(t, "Off", cfg.Params["TraceEnable"])
	assert.Equal(t, "60", cfg.Params["Timeout"])
	assert.Equal(t, "On", cfg.Params["KeepAlive"])
	assert.Equal(t, "apache", cfg.Params["User"])
	assert.Equal(t, "apache", cfg.Params["Group"])
	assert.Equal(t, "warn", cfg.Params["LogLevel"])
	assert.Equal(t, "/var/www/html", cfg.Params["DocumentRoot"])
}

func TestParseListenDirectives(t *testing.T) {
	cfg := Parse(basicConfig)
	require.NotNil(t, cfg)

	// Listen is a multi-param, so both values are comma-joined
	assert.Equal(t, "80,443", cfg.Params["Listen"])
}

func TestParseModules(t *testing.T) {
	cfg := Parse(basicConfig)
	require.NotNil(t, cfg)

	require.Len(t, cfg.Modules, 3)
	assert.Equal(t, "ssl_module", cfg.Modules[0].Name)
	assert.Equal(t, "modules/mod_ssl.so", cfg.Modules[0].Path)
	assert.Equal(t, "rewrite_module", cfg.Modules[1].Name)
	assert.Equal(t, "headers_module", cfg.Modules[2].Name)
}

const virtualHostConfig = `
ServerName default.example.com

<VirtualHost *:80>
    ServerName http.example.com
    DocumentRoot /var/www/http
    Redirect permanent / https://http.example.com/
</VirtualHost>

<VirtualHost *:443>
    ServerName secure.example.com
    DocumentRoot /var/www/secure
    SSLEngine on
    SSLProtocol all -SSLv3 -TLSv1 -TLSv1.1
    SSLCipherSuite HIGH:!aNULL:!MD5
    SSLCertificateFile /etc/pki/tls/certs/server.crt
    SSLCertificateKeyFile /etc/pki/tls/private/server.key
    Header always set Strict-Transport-Security "max-age=31536000; includeSubDomains"
    Header always set X-Frame-Options DENY
</VirtualHost>
`

func TestParseVirtualHosts(t *testing.T) {
	cfg := Parse(virtualHostConfig)
	require.NotNil(t, cfg)

	require.Len(t, cfg.VHosts, 2)

	// HTTP VHost
	http := cfg.VHosts[0]
	assert.Equal(t, "*:80", http.Address)
	assert.Equal(t, "http.example.com", http.ServerName)
	assert.Equal(t, "/var/www/http", http.DocumentRoot)
	assert.False(t, http.SSL)

	// HTTPS VHost
	https := cfg.VHosts[1]
	assert.Equal(t, "*:443", https.Address)
	assert.Equal(t, "secure.example.com", https.ServerName)
	assert.Equal(t, "/var/www/secure", https.DocumentRoot)
	assert.True(t, https.SSL)
	assert.Equal(t, "all -SSLv3 -TLSv1 -TLSv1.1", https.Params["SSLProtocol"])
	assert.Equal(t, "HIGH:!aNULL:!MD5", https.Params["SSLCipherSuite"])
}

const directoryConfig = `
<Directory />
    AllowOverride none
    Require all denied
    Options None
</Directory>

<Directory "/var/www/html">
    Options -Indexes +FollowSymLinks
    AllowOverride None
    Require all granted
</Directory>

<Directory "/var/www/cgi-bin">
    Options +ExecCGI
    AllowOverride None
    Require all granted
</Directory>
`

func TestParseDirectories(t *testing.T) {
	cfg := Parse(directoryConfig)
	require.NotNil(t, cfg)

	require.Len(t, cfg.Dirs, 3)

	root := cfg.Dirs[0]
	assert.Equal(t, "/", root.Path)
	assert.Equal(t, "None", root.Options)
	assert.Equal(t, "none", root.AllowOverride)
	assert.Equal(t, "all denied", root.Params["Require"])

	html := cfg.Dirs[1]
	assert.Equal(t, "/var/www/html", html.Path)
	assert.Equal(t, "-Indexes +FollowSymLinks", html.Options)
	assert.Equal(t, "None", html.AllowOverride)

	cgi := cfg.Dirs[2]
	assert.Equal(t, "/var/www/cgi-bin", cgi.Path)
	assert.Equal(t, "+ExecCGI", cgi.Options)
}

const includeConfig = `
ServerRoot "/etc/httpd"
Include conf.d/*.conf
IncludeOptional sites-enabled/*.conf
`

func TestParseIncludes(t *testing.T) {
	cfg := Parse(includeConfig)
	require.NotNil(t, cfg)

	require.Len(t, cfg.Includes, 2)
	assert.Equal(t, "conf.d/*.conf", cfg.Includes[0])
	assert.Equal(t, "sites-enabled/*.conf", cfg.Includes[1])
}

func TestParseComments(t *testing.T) {
	content := `
# This is a comment
ServerName example.com # inline comment
# Another comment
Timeout 30
`
	cfg := Parse(content)
	require.NotNil(t, cfg)

	assert.Equal(t, "example.com", cfg.Params["ServerName"])
	assert.Equal(t, "30", cfg.Params["Timeout"])
}

func TestParseContinuationLines(t *testing.T) {
	content := `SSLCipherSuite HIGH:MEDIUM:\
!aNULL:!MD5
Timeout 30
`
	cfg := Parse(content)
	require.NotNil(t, cfg)

	assert.Equal(t, "HIGH:MEDIUM: !aNULL:!MD5", cfg.Params["SSLCipherSuite"])
	assert.Equal(t, "30", cfg.Params["Timeout"])
}

func TestParseQuotedValues(t *testing.T) {
	content := `
DocumentRoot "/var/www/my site"
ErrorLog "logs/error_log"
`
	cfg := Parse(content)
	require.NotNil(t, cfg)

	assert.Equal(t, "/var/www/my site", cfg.Params["DocumentRoot"])
	assert.Equal(t, "logs/error_log", cfg.Params["ErrorLog"])
}

func TestParseWithGlob(t *testing.T) {
	files := map[string]string{
		"/etc/httpd/conf/httpd.conf": `
ServerRoot "/etc/httpd"
ServerName main.example.com
Include conf.d/*.conf
`,
		"/etc/httpd/conf.d/ssl.conf": `
SSLProtocol all -SSLv3
SSLCipherSuite HIGH:!aNULL
`,
		"/etc/httpd/conf.d/security.conf": `
ServerTokens Prod
ServerSignature Off
`,
	}

	fileContent := func(path string) (string, error) {
		if c, ok := files[path]; ok {
			return c, nil
		}
		return "", &fileNotFoundError{path: path}
	}

	globExpand := func(pattern string) ([]string, error) {
		if pattern == "conf.d/*.conf" {
			return []string{"/etc/httpd/conf.d/ssl.conf", "/etc/httpd/conf.d/security.conf"}, nil
		}
		return nil, nil
	}

	cfg, err := ParseWithGlob("/etc/httpd/conf/httpd.conf", fileContent, globExpand)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	assert.Equal(t, "main.example.com", cfg.Params["ServerName"])
	assert.Equal(t, "all -SSLv3", cfg.Params["SSLProtocol"])
	assert.Equal(t, "Prod", cfg.Params["ServerTokens"])
	assert.Equal(t, "Off", cfg.Params["ServerSignature"])
}

func TestParseEmptyContent(t *testing.T) {
	cfg := Parse("")
	require.NotNil(t, cfg)
	assert.Empty(t, cfg.Params)
	assert.Empty(t, cfg.Modules)
	assert.Empty(t, cfg.VHosts)
	assert.Empty(t, cfg.Dirs)
}

type fileNotFoundError struct {
	path string
}

func (e *fileNotFoundError) Error() string {
	return "file not found: " + e.path
}
