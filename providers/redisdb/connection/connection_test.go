// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package connection

import "testing"

func TestParseInfo(t *testing.T) {
	info := "# Server\r\nredis_version:7.4.0\r\nredis_mode:standalone\r\n\r\n# Clients\r\nconnected_clients:1\r\n"
	got := ParseInfo(info)
	if got["redis_version"] != "7.4.0" {
		t.Errorf("redis_version = %q, want 7.4.0", got["redis_version"])
	}
	if got["redis_mode"] != "standalone" {
		t.Errorf("redis_mode = %q, want standalone", got["redis_mode"])
	}
	if got["connected_clients"] != "1" {
		t.Errorf("connected_clients = %q, want 1", got["connected_clients"])
	}
	if _, ok := got["# Server"]; ok {
		t.Error("section headers should be skipped")
	}
}
