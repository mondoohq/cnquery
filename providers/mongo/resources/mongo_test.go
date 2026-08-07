// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"
)

func TestDeepGet(t *testing.T) {
	m := bson.M{"net": bson.M{"tls": bson.M{"mode": "requireTLS"}}, "security": bson.M{"authorization": "enabled"}}
	if got := toStr(deepGet(m, "net", "tls", "mode")); got != "requireTLS" {
		t.Errorf("deepGet mode = %q", got)
	}
	if got := toStr(deepGet(m, "security", "authorization")); got != "enabled" {
		t.Errorf("deepGet authorization = %q", got)
	}
	// missing paths return nil without panicking
	if deepGet(m, "net", "ssl", "mode") != nil {
		t.Error("expected nil for missing path")
	}
	if deepGet(nil, "a", "b") != nil {
		t.Error("expected nil for nil root")
	}
}

func TestToIntBool(t *testing.T) {
	if toInt(int32(5)) != 5 || toInt(int64(7)) != 7 || toInt(float64(9)) != 9 || toInt("x") != 0 {
		t.Error("toInt conversions wrong")
	}
	if !toBool(true) || toBool("nope") {
		t.Error("toBool conversions wrong")
	}
}

func TestPrivilegedAndBuiltinRoles(t *testing.T) {
	for _, r := range []string{"root", "readWriteAnyDatabase", "userAdminAnyDatabase"} {
		if _, ok := privilegedRoles[r]; !ok {
			t.Errorf("%q should be a privileged role", r)
		}
	}
	if _, ok := privilegedRoles["read"]; ok {
		t.Error("read should not be a privileged role")
	}
	if _, ok := builtinRoles["read"]; !ok {
		t.Error("read should be a built-in role")
	}
	if _, ok := builtinRoles["appReadMetrics"]; ok {
		t.Error("a custom role should not be in builtinRoles")
	}
}

func TestIDBuilders(t *testing.T) {
	if got := roleResourceID("SRV", "appdb", "appReadMetrics"); got != "SRV/role/appdb.appReadMetrics" {
		t.Errorf("roleResourceID = %q", got)
	}
	if got := userResourceID("SRV", "admin", "opsadmin"); got != "SRV/user/admin.opsadmin" {
		t.Errorf("userResourceID = %q", got)
	}
}
