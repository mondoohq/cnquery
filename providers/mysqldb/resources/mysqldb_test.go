// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import "testing"

func TestIsYes(t *testing.T) {
	for _, s := range []string{"YES", "Y", "ON", "1"} {
		if !isYes(s) {
			t.Errorf("isYes(%q) = false, want true", s)
		}
	}
	for _, s := range []string{"NO", "N", "OFF", "0", ""} {
		if isYes(s) {
			t.Errorf("isYes(%q) = true, want false", s)
		}
	}
}

func TestGrantee(t *testing.T) {
	if got := grantee("appuser", "%"); got != "'appuser'@'%'" {
		t.Errorf("grantee = %q", got)
	}
	if got := grantee("", "localhost"); got != "''@'localhost'" {
		t.Errorf("anonymous grantee = %q", got)
	}
}

func TestIdentifierBuilders(t *testing.T) {
	if got := userResourceID("SRV", "root", "localhost"); got != "SRV/user/root@localhost" {
		t.Errorf("userResourceID = %q", got)
	}
	if got := schemaResourceID("SRV", "appdb"); got != "SRV/schema/appdb" {
		t.Errorf("schemaResourceID = %q", got)
	}
	// composite privilege id must vary by scope, object, and type
	a := privilegeResourceID("p", "GLOBAL", "", "", "SUPER")
	b := privilegeResourceID("p", "SCHEMA", "appdb", "", "SELECT")
	c := privilegeResourceID("p", "TABLE", "appdb", "t1", "SELECT")
	if a == b || b == c || a == c {
		t.Errorf("privilegeResourceID collides: %q %q %q", a, b, c)
	}
}
