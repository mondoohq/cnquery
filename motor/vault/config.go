package vault

import (
	"encoding/json"
	"errors"
	"strings"
)

func NewVaultType(name string) (VaultType, error) {
	entry := strings.TrimSpace(strings.ToLower(name))
	var code VaultType
	ok := false
	for k := range vaultMarshalNameMap {
		if vaultMarshalNameMap[k] == entry {
			ok = true
			code = k
			break
		}
	}
	if !ok {
		return VaultType_None, errors.New("unknown type value: " + string(name))
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
}

func (t *VaultType) Value() string {
	if t == nil {
		return ""
	}
	return vaultMarshalNameMap[*t]
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
		entry := strings.TrimSpace(strings.ToLower(name))

		var code VaultType
		ok := false
		for k := range vaultMarshalNameMap {
			if vaultMarshalNameMap[k] == entry {
				ok = true
				code = k
				break
			}
		}
		if !ok {
			return errors.New("unknown type value: " + string(data))
		}
		*t = code
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
