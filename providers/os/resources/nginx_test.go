// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"github.com/nginxinc/nginx-go-crossplane"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseNginxServerBlock(t *testing.T) {
	directives := crossplane.Directives{
		{Directive: "listen", Args: []string{"80"}},
		{Directive: "listen", Args: []string{"443", "ssl"}},
		{Directive: "server_name", Args: []string{"example.com", "www.example.com"}},
		{Directive: "root", Args: []string{"/var/www/html"}},
		{Directive: "ssl_certificate", Args: []string{"/etc/ssl/cert.pem"}},
		{Directive: "ssl_certificate_key", Args: []string{"/etc/ssl/key.pem"}},
		{Directive: "location", Args: []string{"/"}, Block: crossplane.Directives{
			{Directive: "proxy_pass", Args: []string{"http://backend"}},
		}},
		{Directive: "location", Args: []string{"/static"}, Block: crossplane.Directives{
			{Directive: "root", Args: []string{"/var/www/static"}},
			{Directive: "expires", Args: []string{"30d"}},
		}},
	}

	srv := parseNginxServerBlock(directives)
	assert.Equal(t, "example.com www.example.com", srv.ServerName)
	assert.Equal(t, "80,443 ssl", srv.Listen)
	assert.Equal(t, "/var/www/html", srv.Root)
	assert.True(t, srv.SSL)
	assert.Equal(t, "/etc/ssl/cert.pem", srv.Params["ssl_certificate"])
	assert.Equal(t, "/etc/ssl/key.pem", srv.Params["ssl_certificate_key"])

	require.Len(t, srv.Locations, 2)
	assert.Equal(t, "/", srv.Locations[0].Path)
	assert.Equal(t, "http://backend", srv.Locations[0].ProxyPass)
	assert.Equal(t, "/static", srv.Locations[1].Path)
	assert.Equal(t, "/var/www/static", srv.Locations[1].Root)
	assert.Equal(t, "30d", srv.Locations[1].Params["expires"])
}

func TestParseNginxServerBlockSSLViaListen(t *testing.T) {
	directives := crossplane.Directives{
		{Directive: "listen", Args: []string{"443", "ssl"}},
		{Directive: "server_name", Args: []string{"secure.example.com"}},
	}

	srv := parseNginxServerBlock(directives)
	assert.True(t, srv.SSL)
}

func TestParseNginxServerBlockNoSSL(t *testing.T) {
	directives := crossplane.Directives{
		{Directive: "listen", Args: []string{"80"}},
		{Directive: "server_name", Args: []string{"plain.example.com"}},
	}

	srv := parseNginxServerBlock(directives)
	assert.False(t, srv.SSL)
}

func TestParseNginxUpstreamBlock(t *testing.T) {
	directives := crossplane.Directives{
		{Directive: "least_conn"},
		{Directive: "server", Args: []string{"127.0.0.1:8080"}},
		{Directive: "server", Args: []string{"127.0.0.1:8081", "weight=3"}},
		{Directive: "keepalive", Args: []string{"32"}},
	}

	up := parseNginxUpstreamBlock("backend", directives)
	assert.Equal(t, "backend", up.Name)
	require.Len(t, up.Servers, 2)
	assert.Equal(t, "127.0.0.1:8080", up.Servers[0])
	assert.Equal(t, "127.0.0.1:8081 weight=3", up.Servers[1])
	assert.Equal(t, "32", up.Params["keepalive"])
	assert.Equal(t, "", up.Params["least_conn"])
}

func TestParseNginxLocationBlock(t *testing.T) {
	directives := crossplane.Directives{
		{Directive: "proxy_pass", Args: []string{"http://backend"}},
		{Directive: "proxy_set_header", Args: []string{"Host", "$host"}},
		{Directive: "proxy_set_header", Args: []string{"X-Real-IP", "$remote_addr"}},
		{Directive: "root", Args: []string{"/var/www"}},
	}

	loc := parseNginxLocationBlock("/api", directives)
	assert.Equal(t, "/api", loc.Path)
	assert.Equal(t, "http://backend", loc.ProxyPass)
	assert.Equal(t, "/var/www", loc.Root)
	assert.Equal(t, "Host $host,X-Real-IP $remote_addr", loc.Params["proxy_set_header"])
}

func TestWalkHTTPBlock(t *testing.T) {
	directives := crossplane.Directives{
		{Directive: "server_tokens", Args: []string{"off"}},
		{Directive: "sendfile", Args: []string{"on"}},
		{Directive: "upstream", Args: []string{"backend"}, Block: crossplane.Directives{
			{Directive: "server", Args: []string{"127.0.0.1:8080"}},
			{Directive: "server", Args: []string{"127.0.0.1:8081"}},
		}},
		{Directive: "server", Block: crossplane.Directives{
			{Directive: "listen", Args: []string{"80"}},
			{Directive: "server_name", Args: []string{"example.com"}},
			{Directive: "location", Args: []string{"/"}, Block: crossplane.Directives{
				{Directive: "proxy_pass", Args: []string{"http://backend"}},
			}},
		}},
		{Directive: "server", Block: crossplane.Directives{
			{Directive: "listen", Args: []string{"443", "ssl"}},
			{Directive: "server_name", Args: []string{"example.com"}},
		}},
	}

	httpParams := map[string]any{}
	var servers []nginxServer
	var upstreams []nginxUpstream
	var listenAddrs []string

	walkHTTPBlock(directives, httpParams, &servers, &upstreams, &listenAddrs)

	// HTTP params
	assert.Equal(t, "off", httpParams["server_tokens"])
	assert.Equal(t, "on", httpParams["sendfile"])

	// Upstreams
	require.Len(t, upstreams, 1)
	assert.Equal(t, "backend", upstreams[0].Name)
	require.Len(t, upstreams[0].Servers, 2)

	// Servers
	require.Len(t, servers, 2)
	assert.Equal(t, "example.com", servers[0].ServerName)
	assert.Equal(t, "80", servers[0].Listen)
	assert.False(t, servers[0].SSL)
	require.Len(t, servers[0].Locations, 1)
	assert.Equal(t, "example.com", servers[1].ServerName)
	assert.Equal(t, "443 ssl", servers[1].Listen)
	assert.True(t, servers[1].SSL)

	// Listen addresses
	require.Len(t, listenAddrs, 2)
	assert.Equal(t, "80", listenAddrs[0])
	assert.Equal(t, "443 ssl", listenAddrs[1])
}

func TestSetNginxParam(t *testing.T) {
	t.Run("simple param overwrites", func(t *testing.T) {
		m := map[string]any{}
		setNginxParam(m, "worker_processes", "auto")
		assert.Equal(t, "auto", m["worker_processes"])

		setNginxParam(m, "worker_processes", "4")
		assert.Equal(t, "4", m["worker_processes"])
	})

	t.Run("multi-param concatenates", func(t *testing.T) {
		m := map[string]any{}
		setNginxParam(m, "add_header", "X-Frame-Options DENY")
		setNginxParam(m, "add_header", "X-Content-Type-Options nosniff")
		assert.Equal(t, "X-Frame-Options DENY,X-Content-Type-Options nosniff", m["add_header"])
	})

	t.Run("listen multi-param", func(t *testing.T) {
		m := map[string]any{}
		setNginxParam(m, "listen", "80")
		setNginxParam(m, "listen", "443 ssl")
		assert.Equal(t, "80,443 ssl", m["listen"])
	})
}

func TestNginxConfPathDefault(t *testing.T) {
	assert.Equal(t, "/etc/nginx/nginx.conf", defaultNginxConf)
}

func TestExtractNginxVersion(t *testing.T) {
	t.Run("embedded version in binary data", func(t *testing.T) {
		// Simulate binary data with embedded version string
		data := []byte("\x00\x00nginx/1.25.3\x00\x00")
		assert.Equal(t, "1.25.3", extractNginxVersion(data))
	})

	t.Run("four-part version", func(t *testing.T) {
		data := []byte("some binary stuff\x00nginx/1.21.4.2\x00more stuff")
		assert.Equal(t, "1.21.4.2", extractNginxVersion(data))
	})

	t.Run("no version tag", func(t *testing.T) {
		data := []byte("no version here")
		assert.Equal(t, "", extractNginxVersion(data))
	})

	t.Run("empty data", func(t *testing.T) {
		assert.Equal(t, "", extractNginxVersion([]byte{}))
	})

	t.Run("tag without version digits", func(t *testing.T) {
		data := []byte("nginx/\x00rest")
		assert.Equal(t, "", extractNginxVersion(data))
	})
}

func TestNginxVersionRegex(t *testing.T) {
	t.Run("standard command output", func(t *testing.T) {
		output := []byte("nginx version: nginx/1.25.3\n")
		m := reNginxVersion.FindSubmatch(output)
		require.NotNil(t, m)
		assert.Equal(t, "1.25.3", string(m[1]))
	})

	t.Run("openresty variant", func(t *testing.T) {
		// The -v output for openresty still contains nginx/version
		m := reNginxVersion.FindSubmatch([]byte("nginx version: nginx/1.21.4.2\n"))
		require.NotNil(t, m)
		assert.Equal(t, "1.21.4.2", string(m[1]))
	})
}

func TestParseNginxModules(t *testing.T) {
	output := "nginx version: nginx/1.25.3\n" +
		"built by gcc 12.2.0 (Debian 12.2.0-14)\n" +
		"built with OpenSSL 3.0.11 19 Sep 2023\n" +
		"TLS SNI support enabled\n" +
		"configure arguments: --prefix=/etc/nginx --sbin-path=/usr/sbin/nginx " +
		"--with-compat --with-threads " +
		"--with-http_ssl_module --with-http_v2_module " +
		"--with-http_gzip_static_module --with-stream_ssl_module " +
		"--with-mail_ssl_module\n"

	modules := parseNginxModules(output)
	require.Len(t, modules, 5)
	assert.Equal(t, "http_ssl_module", modules[0])
	assert.Equal(t, "http_v2_module", modules[1])
	assert.Equal(t, "http_gzip_static_module", modules[2])
	assert.Equal(t, "stream_ssl_module", modules[3])
	assert.Equal(t, "mail_ssl_module", modules[4])
}

func TestParseNginxModulesEmpty(t *testing.T) {
	output := "nginx version: nginx/1.25.3\nconfigure arguments: --prefix=/etc/nginx\n"
	modules := parseNginxModules(output)
	assert.Empty(t, modules)
}
