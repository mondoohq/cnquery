// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package connection

import "testing"

func TestClassifyFlavor(t *testing.T) {
	cases := []struct {
		comment, version, want string
	}{
		{"MySQL Community Server - GPL", "8.0.40", "mysql"},
		{"mariadb.org binary distribution", "11.4.2-MariaDB", "mariadb"},
		{"MySQL Community Server (GPL)", "10.11.8-MariaDB-1:10.11.8+maria~ubu2204", "mariadb"},
		{"Percona Server (GPL), Release 30", "8.0.36-28", "percona"},
		{"", "8.4.0", "mysql"},
	}
	for _, tc := range cases {
		if got := classifyFlavor(tc.comment, tc.version); got != tc.want {
			t.Errorf("classifyFlavor(%q, %q) = %q, want %q", tc.comment, tc.version, got, tc.want)
		}
	}
}

func TestTLSParamKeyword(t *testing.T) {
	for _, mode := range []string{"false", "skip-verify", "preferred", "true"} {
		c := &MysqldbConnection{tlsMode: mode}
		got, err := c.tlsParam()
		if err != nil {
			t.Fatalf("tlsParam(%q) error: %v", mode, err)
		}
		if got != mode {
			t.Errorf("tlsParam(%q) = %q, want passthrough", mode, got)
		}
	}
}
