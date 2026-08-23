// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIpsecProposalArgs(t *testing.T) {
	row := map[string]string{
		"name":            "default",
		"enc-algorithms":  "3des,aes-128-cbc",
		"auth-algorithms": "md5,sha1",
		"pfs-group":       "modp1024",
		"lifetime":        "30m",
		"default":         "true",
		"disabled":        "false",
	}
	args := ipsecProposalArgs(row)

	assert.Equal(t, "mikrotik.ip.ipsec.proposal/default", args["__id"].Value)
	// 3des/md5/modp1024 is the combination worth finding
	assert.Equal(t, []any{"3des", "aes-128-cbc"}, args["encAlgorithms"].Value)
	assert.Equal(t, []any{"md5", "sha1"}, args["authAlgorithms"].Value)
	assert.Equal(t, "modp1024", args["pfsGroup"].Value)
	assert.Equal(t, true, args["default"].Value)
	assert.Equal(t, false, args["disabled"].Value)
}

func TestIpsecProposalArgsAbsentAttributes(t *testing.T) {
	args := ipsecProposalArgs(map[string]string{"name": "bare"})

	assert.Nil(t, args["encAlgorithms"].Value)
	assert.Nil(t, args["authAlgorithms"].Value)
	assert.Nil(t, args["default"].Value)
	assert.Nil(t, args["disabled"].Value)
}

func TestIpsecProfileArgs(t *testing.T) {
	row := map[string]string{
		"name":                 "default",
		"hash-algorithm":       "sha1",
		"enc-algorithm":        "aes-128,3des",
		"dh-group":             "modp1024,modp2048",
		"prf-algorithm":        "auto",
		"proposal-check":       "obey",
		"lifetime":             "1d",
		"nat-traversal":        "yes",
		"dpd-interval":         "2m",
		"dpd-maximum-failures": "5",
	}
	args := ipsecProfileArgs(row)

	assert.Equal(t, "mikrotik.ip.ipsec.profile/default", args["__id"].Value)
	// RouterOS names both attributes in the singular on this menu
	assert.Equal(t, []any{"aes-128", "3des"}, args["encAlgorithms"].Value)
	assert.Equal(t, []any{"modp1024", "modp2048"}, args["dhGroups"].Value)
	assert.Equal(t, "sha1", args["hashAlgorithm"].Value)
	assert.Equal(t, true, args["natTraversal"].Value)
	assert.Equal(t, int64(5), args["dpdMaximumFailures"].Value)
}

func TestIpsecPeerArgs(t *testing.T) {
	row := map[string]string{
		".id":                  "*2",
		"name":                 "branch-office",
		"address":              "203.0.113.10/32",
		"local-address":        "198.51.100.2",
		"port":                 "500",
		"exchange-mode":        "aggressive",
		"profile":              "default",
		"passive":              "no",
		"send-initial-contact": "yes",
		"disabled":             "false",
	}
	args := ipsecPeerArgs(row)

	assert.Equal(t, "mikrotik.ip.ipsec.peer/*2", args["__id"].Value)
	assert.Equal(t, "branch-office", args["name"].Value)
	// aggressive mode exposes a hash an attacker can grind offline
	assert.Equal(t, "aggressive", args["exchangeMode"].Value)
	assert.Equal(t, "default", args["profile"].Value)
	assert.Equal(t, int64(500), args["port"].Value)
	assert.Equal(t, false, args["passive"].Value)
}

func TestIpsecIdentityArgs(t *testing.T) {
	row := map[string]string{
		".id":                   "*4",
		"peer":                  "branch-office",
		"auth-method":           "pre-shared-key",
		"secret":                "not-a-real-key",
		"generate-policy":       "port-override",
		"policy-template-group": "default",
		"match-by":              "remote-id",
		"my-id-type":            "auto",
		"remote-id-type":        "auto",
		"disabled":              "false",
	}
	args := ipsecIdentityArgs(row)

	assert.Equal(t, "mikrotik.ip.ipsec.identity/*4", args["__id"].Value)
	// pre-shared-key plus generate-policy is the pair that matters
	assert.Equal(t, "pre-shared-key", args["authMethod"].Value)
	assert.Equal(t, "port-override", args["generatePolicy"].Value)
	assert.Equal(t, true, args["hasSecret"].Value)

	// the pre-shared key is never carried into the resource, under any field
	assert.NotContains(t, args, "secret")
	assert.NotContains(t, args, "password")
	assert.NotContains(t, args, "key")
	for name, v := range args {
		assert.NotEqual(t, "not-a-real-key", v.Value, "field %q leaked the secret", name)
	}
}

func TestIpsecIdentityArgsNoSecret(t *testing.T) {
	args := ipsecIdentityArgs(map[string]string{
		"peer":        "branch-office",
		"auth-method": "digital-signature",
		"secret":      "",
		"certificate": "ipsec-local",
	})

	assert.Equal(t, "mikrotik.ip.ipsec.identity/branch-office", args["__id"].Value)
	assert.Equal(t, false, args["hasSecret"].Value)
	assert.Equal(t, "ipsec-local", args["certificate"].Value)
}

func TestIpsecIdentityArgsAbsentAttributes(t *testing.T) {
	args := ipsecIdentityArgs(map[string]string{"peer": "branch-office"})

	// an identity RouterOS reported nothing about must not read as one with
	// no key configured
	assert.Nil(t, args["hasSecret"].Value)
	assert.Nil(t, args["disabled"].Value)
	assert.Equal(t, "", args["generatePolicy"].Value)
}
