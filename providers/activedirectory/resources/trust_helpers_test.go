// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import "testing"

func TestParseTrustType(t *testing.T) {
	tests := []struct {
		name         string
		sourceDomain string
		targetDomain string
		trustType    int64
		trustAttrs   int64
		want         string
	}{
		{name: "downlevel", trustType: trustTypeDownlevel, want: "Downlevel"},
		{name: "mit", trustType: trustTypeMIT, want: "MIT"},
		{name: "aad", trustType: trustTypeAAD, want: "AzureAD"},
		{name: "forest", trustType: trustTypeUplevel, trustAttrs: trustAttrForestTransitive, want: "Forest"},
		{name: "parent child", sourceDomain: "child.example.com", targetDomain: "example.com", trustType: trustTypeUplevel, trustAttrs: trustAttrWithinForest, want: "ParentChild"},
		{name: "cross link", sourceDomain: "emea.example.com", targetDomain: "apac.example.com", trustType: trustTypeUplevel, trustAttrs: trustAttrWithinForest, want: "CrossLink"},
		{name: "external", trustType: trustTypeUplevel, want: "External"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseTrustType(tt.sourceDomain, tt.targetDomain, tt.trustType, tt.trustAttrs)
			if got != tt.want {
				t.Fatalf("parseTrustType() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestTrustAttributeHelpers(t *testing.T) {
	attrs := int64(trustAttrCrossOrganization | trustAttrTreatAsExternal | trustAttrUsesAESKeys | trustAttrCrossOrganizationEnableTGTDelegation)
	if !trustUsesSelectiveAuthentication(attrs) {
		t.Fatal("expected selective authentication")
	}
	if !trustUsesAES(attrs) {
		t.Fatal("expected AES")
	}
	if !trustAllowsTGTDelegation(attrs) {
		t.Fatal("expected TGT delegation")
	}
	if !trustHasSIDHistoryEnabled(attrs) {
		t.Fatal("expected SID history enabled")
	}
	if trustHasSIDFilteringEnabled(attrs) {
		t.Fatal("did not expect SID filtering")
	}
	if trustUsesRC4(trustTypeUplevel, attrs) {
		t.Fatal("did not expect RC4 for uplevel trust with AES enabled")
	}
	if !trustUsesRC4(trustTypeUplevel, int64(trustAttrWithinForest)) {
		t.Fatal("expected RC4 for uplevel trust without AES keys")
	}
	if !trustUsesRC4(trustTypeMIT, int64(trustAttrUsesRC4Encryption)) {
		t.Fatal("expected RC4 for MIT trust")
	}
}
