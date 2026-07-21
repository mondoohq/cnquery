// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"strings"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/ranger-rpc/codes"
	"go.mondoo.com/ranger-rpc/status"
)

// Registry locations of the effective Schannel TLS configuration. The ordered
// cipher-suite list and the ordered elliptic-curve / supported-group list are
// each stored as the REG_MULTI_SZ `Functions` value under their own subkey of
// the local SSL configuration store.
//
// Verified against Microsoft Learn: the cipher-suite order lives at
// ...\Local\SSL\00010002 as the `Functions` REG_MULTI_SZ value (documented in
// several Microsoft troubleshooting articles). The companion
// ...\Local\SSL\00010003 subkey holds the effective elliptic-curve /
// supported-group order, where post-quantum ML-KEM groups appear on Windows 11
// 24H2 and Windows Server 2025.
const (
	schannelCipherSuitesPath    = `HKEY_LOCAL_MACHINE\SYSTEM\CurrentControlSet\Control\Cryptography\Configuration\Local\SSL\00010002`
	schannelSupportedGroupsPath = `HKEY_LOCAL_MACHINE\SYSTEM\CurrentControlSet\Control\Cryptography\Configuration\Local\SSL\00010003`
	schannelFunctionsValue      = "Functions"
)

func (r *mqlWindowsSchannel) id() (string, error) {
	return "windows.schannel", nil
}

// readMultiString reads the named REG_MULTI_SZ value from a registry key and
// returns its ordered entries. A missing key or a missing value yields an empty
// slice rather than an error, so the field degrades gracefully on older Windows,
// on systems using the default order, and on non-Windows platforms.
func (r *mqlWindowsSchannel) readMultiString(path, name string) ([]string, error) {
	o, err := CreateResource(r.MqlRuntime, "registrykey", map[string]*llx.RawData{
		"path": llx.StringData(path),
	})
	if err != nil {
		return nil, err
	}

	entries, err := o.(*mqlRegistrykey).getEntries()
	if err != nil {
		// a missing key is expected (e.g. the default order is in effect); treat
		// it as empty so the field resolves to an empty list rather than erroring
		if st, ok := status.FromError(err); ok && st.Code() == codes.NotFound {
			return []string{}, nil
		}
		return nil, err
	}

	for i := range entries {
		if strings.EqualFold(entries[i].Key, name) {
			return entries[i].Value.MultiString, nil
		}
	}
	return []string{}, nil
}

func (r *mqlWindowsSchannel) cipherSuites() ([]interface{}, error) {
	suites, err := r.readMultiString(schannelCipherSuitesPath, schannelFunctionsValue)
	if err != nil {
		return nil, err
	}
	return strSliceToAny(suites), nil
}

func (r *mqlWindowsSchannel) ellipticCurves() ([]interface{}, error) {
	curves, err := r.readMultiString(schannelSupportedGroupsPath, schannelFunctionsValue)
	if err != nil {
		return nil, err
	}
	return strSliceToAny(curves), nil
}

func (r *mqlWindowsSchannel) pqcKeyExchangeEnabled() (bool, error) {
	// Derive from the already-resolved ellipticCurves field rather than reading
	// the registry again, so a query for both fields reads the supported-group
	// key once and the two fields stay consistent by construction.
	raw := r.GetEllipticCurves()
	if raw.Error != nil {
		return false, raw.Error
	}

	curves := make([]string, 0, len(raw.Data))
	for _, v := range raw.Data {
		if s, ok := v.(string); ok {
			curves = append(curves, s)
		}
	}
	return pqcKeyExchangeEnabled(curves), nil
}

// pqcKeyExchangeEnabled reports whether any supported group is a post-quantum
// ML-KEM key-exchange group. Windows names these groups with an "mlkem" segment
// (for example x25519_mlkem768, secp256r1_mlkem768, secp384r1_mlkem1024), so the
// match is a case-insensitive substring test for "MLKEM".
func pqcKeyExchangeEnabled(curves []string) bool {
	for _, c := range curves {
		if strings.Contains(strings.ToUpper(c), "MLKEM") {
			return true
		}
	}
	return false
}
