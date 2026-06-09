// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import "strings"

// cipherInfo holds the components and derived security properties parsed from a
// TLS/SSL cipher suite name.
type cipherInfo struct {
	keyExchange    string
	authentication string
	encryption     string
	mac            string
	forwardSecrecy bool
	aead           bool
	export         bool
	null           bool
	anonymous      bool
	cbc            bool
}

// classifyCipher parses an IANA/OpenSSL cipher suite name (e.g.
// "TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256") into its components and derived
// security properties. Parsing is best-effort: unknown components are left
// empty, but the boolean properties are derived from the full name and stay
// reliable even for suites whose component layout is unusual.
func classifyCipher(name string) cipherInfo {
	upper := strings.ToUpper(name)

	info := cipherInfo{
		export:    strings.Contains(upper, "EXPORT"),
		null:      strings.Contains(upper, "NULL"),
		anonymous: strings.Contains(upper, "ANON"),
		cbc:       strings.Contains(upper, "CBC"),
		aead: strings.Contains(upper, "GCM") ||
			strings.Contains(upper, "CCM") ||
			strings.Contains(upper, "POLY1305"),
	}

	body := upper
	for _, prefix := range []string{"TLS_", "SSL_", "SSL2_", "SSL3_"} {
		body = strings.TrimPrefix(body, prefix)
	}

	if kxAuth, encMac, ok := strings.Cut(body, "_WITH_"); ok {
		info.keyExchange, info.authentication = parseKeyExchangeAuth(kxAuth)
		info.encryption, info.mac = parseEncryptionMac(encMac)
		info.forwardSecrecy = isEphemeralKeyExchange(info.keyExchange)
	} else {
		// TLS 1.3 form (e.g. "AES_128_GCM_SHA256"): the key exchange and
		// authentication are negotiated separately, and every suite is AEAD
		// and forward-secret.
		info.encryption, info.mac = parseEncryptionMac(body)
		info.forwardSecrecy = info.aead
	}

	return info
}

func parseKeyExchangeAuth(s string) (keyExchange, authentication string) {
	tokens := strings.Split(s, "_")
	if len(tokens) == 0 || tokens[0] == "" {
		return "", ""
	}
	keyExchange = tokens[0]
	if len(tokens) == 1 {
		// e.g. "RSA" or "PSK": the key exchange also provides authentication
		authentication = tokens[0]
	} else {
		// e.g. "ECDHE_RSA" -> RSA, "DH_anon" -> anon, "SRP_SHA_RSA" -> RSA
		authentication = tokens[len(tokens)-1]
	}
	if strings.EqualFold(authentication, "ANON") {
		authentication = "anon"
	}
	return keyExchange, authentication
}

func parseEncryptionMac(s string) (encryption, mac string) {
	tokens := strings.Split(s, "_")
	if len(tokens) == 0 || tokens[0] == "" {
		return "", ""
	}
	if len(tokens) == 1 {
		return tokens[0], ""
	}
	// the trailing token is the MAC / PRF hash (SHA, SHA256, SHA384, MD5)
	mac = tokens[len(tokens)-1]
	encryption = strings.Join(tokens[:len(tokens)-1], "_")
	return encryption, mac
}

func isEphemeralKeyExchange(kx string) bool {
	switch kx {
	case "ECDHE", "DHE", "EECDH", "EDH":
		return true
	}
	// post-quantum hybrids (e.g. CECPQ1, CECPQ2) are ephemeral
	return strings.HasPrefix(kx, "CECPQ")
}
