// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package vault

import (
	"encoding/json"
	"errors"
	"strings"
)

// supports both the protobuf string and the custom defined string
// representations of a vault type
func NewVaultType(name string) (VaultType, error) {
	entry := strings.TrimSpace(strings.ToLower(name))
	var code VaultType
	marshalMapOk := false
	for k := range vaultMarshalNameMap {
		if vaultMarshalNameMap[k] == entry {
			marshalMapOk = true
			code = k
			break
		}
	}
	if !marshalMapOk {
		// also support the auto-generated protobuf string values for all the enum values
		protoMapOk := false
		for k, v := range VaultType_value {
			if k == name {
				protoMapOk = true
				code = VaultType(v)
				break
			}
		}
		if !protoMapOk {
			return VaultType_None, errors.New("unknown type value: " + string(name))
		}
	}

	return code, nil
}

var vaultMarshalNameMap = map[VaultType]string{
	VaultType_None:               "none",
	VaultType_KeyRing:            "keyring",
	VaultType_LinuxKernelKeyring: "linux-kernel-keyring",
	VaultType_EncryptedFile:      "encrypted-file",
	VaultType_HashiCorp:          "hashicorp-vault",
	VaultType_GCPSecretsManager:  "gcp-secret-manager",
	VaultType_AWSSecretsManager:  "aws-secrets-manager",
	VaultType_AWSParameterStore:  "aws-parameter-store",
	VaultType_GCPBerglas:         "gcp-berglas",
	VaultType_Memory:             "memory",
}

func (t *VaultType) Value() string {
	if t == nil {
		return ""
	}
	return vaultMarshalNameMap[*t]
}

func TypeIds() []string {
	var types []string
	for _, v := range vaultMarshalNameMap {
		types = append(types, v)
	}
	return types
}

// UnmarshalJSON parses either an int or a string representation of
// VaultType into the struct
func (t *VaultType) UnmarshalJSON(data []byte) error {
	// check if we have a number
	var code int32
	err := json.Unmarshal(data, &code)
	if err == nil {
		*t = VaultType(code)
	}
	if err != nil {
		var name string
		err = json.Unmarshal(data, &name)
		if err != nil {
			return err
		}
		c, err := NewVaultType(name)
		if err != nil {
			return err
		}
		*t = c
	}
	return nil
}

// MarshalJSON returns the JSON representation of VaultType
// NOTE: we do not use pointers here to ensure its converted properly
// even if the struct is used directly
func (t VaultType) MarshalJSON() ([]byte, error) {
	v, ok := vaultMarshalNameMap[t]
	if !ok {
		return nil, errors.New("could not marshal CredentialType")
	}
	return json.Marshal(v)
}
