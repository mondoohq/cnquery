// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"testing"
	"time"
)

// idracNetworkProtocol is a recorded ManagerNetworkProtocol document of a Dell
// iDRAC 9. It reports every protocol, with IPMI over LAN and SNMP enabled.
const idracNetworkProtocol = `{
  "@odata.id": "/redfish/v1/Managers/iDRAC.Embedded.1/NetworkProtocol",
  "@odata.type": "#ManagerNetworkProtocol.v1_5_0.ManagerNetworkProtocol",
  "FQDN": "bmc-01.example.com",
  "HostName": "bmc-01",
  "HTTP": {"Port": 80, "ProtocolEnabled": false},
  "HTTPS": {
    "Certificates": {"@odata.id": "/redfish/v1/Managers/iDRAC.Embedded.1/NetworkProtocol/HTTPS/Certificates"},
    "Port": 443,
    "ProtocolEnabled": true
  },
  "IPMI": {"Port": 623, "ProtocolEnabled": true},
  "KVMIP": {"Port": 5900, "ProtocolEnabled": true},
  "SNMP": {"Port": 161, "ProtocolEnabled": true},
  "SSH": {"Port": 22, "ProtocolEnabled": true},
  "Telnet": {"Port": 23, "ProtocolEnabled": false},
  "VirtualMedia": {"Port": 17990, "ProtocolEnabled": true}
}`

// iloNetworkProtocol is a recorded ManagerNetworkProtocol document of an HPE
// iLO 5. It leaves out telnet and IPMI, which the controller does not report.
const iloNetworkProtocol = `{
  "@odata.id": "/redfish/v1/Managers/1/NetworkProtocol",
  "@odata.type": "#ManagerNetworkProtocol.v1_5_0.ManagerNetworkProtocol",
  "FQDN": "bmc-02.example.com",
  "HostName": "bmc-02",
  "HTTP": {"ProtocolEnabled": false},
  "HTTPS": {"Port": 443, "ProtocolEnabled": true},
  "KVMIP": {"Port": 17990, "ProtocolEnabled": true},
  "SNMP": {"Port": 161, "ProtocolEnabled": false},
  "SSH": {"Port": 22, "ProtocolEnabled": true},
  "VirtualMedia": {"Port": 17988, "ProtocolEnabled": true}
}`

func TestParseNetworkProtocolReportedProtocols(t *testing.T) {
	np, err := parseNetworkProtocol([]byte(idracNetworkProtocol))
	if err != nil {
		t.Fatalf("parseNetworkProtocol() error = %v", err)
	}

	if np.HostName != "bmc-01" || np.FQDN != "bmc-01.example.com" {
		t.Errorf("got identity (%q, %q)", np.HostName, np.FQDN)
	}
	if np.ODataID != "/redfish/v1/Managers/iDRAC.Embedded.1/NetworkProtocol" {
		t.Errorf("got odata id %q", np.ODataID)
	}

	tests := []struct {
		name    string
		setting *protocolSettingJSON
		enabled bool
		port    int64
	}{
		{"http", np.HTTP, false, 80},
		{"https", np.HTTPS, true, 443},
		{"ssh", np.SSH, true, 22},
		{"telnet", np.Telnet, false, 23},
		{"ipmi", np.IPMI, true, 623},
		{"snmp", np.SNMP, true, 161},
		{"kvmip", np.KVMIP, true, 5900},
		{"virtualMedia", np.VirtualMedia, true, 17990},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			enabled := protocolEnabled(tt.setting)
			if enabled == nil {
				t.Fatal("enabled is null, want a reported value")
			}
			if *enabled != tt.enabled {
				t.Errorf("enabled = %v, want %v", *enabled, tt.enabled)
			}
			port := protocolPort(tt.setting)
			if port == nil {
				t.Fatal("port is null, want a reported value")
			}
			if *port != tt.port {
				t.Errorf("port = %d, want %d", *port, tt.port)
			}
		})
	}

	if np.HTTPS.Certificates == nil || np.HTTPS.Certificates.ODataID == "" {
		t.Error("HTTPS carries no certificate collection link")
	}
}

func TestParseNetworkProtocolUnreportedProtocolsAreNull(t *testing.T) {
	np, err := parseNetworkProtocol([]byte(iloNetworkProtocol))
	if err != nil {
		t.Fatalf("parseNetworkProtocol() error = %v", err)
	}

	// The controller does not describe telnet or IPMI, so neither may read as
	// disabled. A silent false would hide an IPMI over LAN interface.
	if got := protocolEnabled(np.Telnet); got != nil {
		t.Errorf("telnet enabled = %v, want null", *got)
	}
	if got := protocolEnabled(np.IPMI); got != nil {
		t.Errorf("ipmi enabled = %v, want null", *got)
	}
	if got := protocolPort(np.IPMI); got != nil {
		t.Errorf("ipmi port = %d, want null", *got)
	}

	// HTTP is reported without a port, so the state is known and the port is not.
	enabled := protocolEnabled(np.HTTP)
	if enabled == nil || *enabled {
		t.Errorf("http enabled = %v, want false", enabled)
	}
	if got := protocolPort(np.HTTP); got != nil {
		t.Errorf("http port = %d, want null", *got)
	}

	if np.HTTPS.Certificates != nil {
		t.Error("HTTPS reports a certificate collection that the document does not contain")
	}
}

func TestParseNetworkProtocolMalformed(t *testing.T) {
	if _, err := parseNetworkProtocol([]byte(`{not json`)); err == nil {
		t.Error("parseNetworkProtocol() accepted malformed input")
	}
}

func TestParseManagerNetworkProtocolLink(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want string
	}{
		{
			name: "manager with link",
			raw:  `{"Id":"1","NetworkProtocol":{"@odata.id":"/redfish/v1/Managers/1/NetworkProtocol"}}`,
			want: "/redfish/v1/Managers/1/NetworkProtocol",
		},
		{name: "manager without link", raw: `{"Id":"1"}`},
		{name: "empty link", raw: `{"NetworkProtocol":{"@odata.id":""}}`},
		{name: "empty document", raw: ``},
		{name: "malformed json", raw: `{`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := parseManagerNetworkProtocolLink([]byte(tt.raw)); got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestParseCollectionMembers(t *testing.T) {
	raw := `{
      "@odata.id": "/redfish/v1/Managers/1/NetworkProtocol/HTTPS/Certificates",
      "Members": [
        {"@odata.id": "/redfish/v1/Managers/1/NetworkProtocol/HTTPS/Certificates/1"},
        {"@odata.id": ""},
        {"@odata.id": "/redfish/v1/Managers/1/NetworkProtocol/HTTPS/Certificates/2"}
      ],
      "Members@odata.count": 3
    }`

	members, err := parseCollectionMembers([]byte(raw))
	if err != nil {
		t.Fatalf("parseCollectionMembers() error = %v", err)
	}
	if len(members) != 2 {
		t.Fatalf("got %d members, want 2", len(members))
	}
	if members[0] != "/redfish/v1/Managers/1/NetworkProtocol/HTTPS/Certificates/1" {
		t.Errorf("got first member %q", members[0])
	}

	if _, err := parseCollectionMembers([]byte(`[`)); err == nil {
		t.Error("parseCollectionMembers() accepted malformed input")
	}
}

// selfSignedPEM returns a self-signed certificate in PEM form.
func selfSignedPEM(t *testing.T, key any, pub any) string {
	t.Helper()
	template := &x509.Certificate{
		SerialNumber: big.NewInt(4096),
		Subject: pkix.Name{
			CommonName:   "bmc-01.example.com",
			Organization: []string{"Example"},
		},
		NotBefore: time.Now().Add(-time.Hour),
		NotAfter:  time.Now().Add(time.Hour),
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, pub, key)
	if err != nil {
		t.Fatalf("could not create certificate: %v", err)
	}
	return string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}))
}

func TestParseCertificate(t *testing.T) {
	raw := `{
      "@odata.id": "/redfish/v1/Managers/1/NetworkProtocol/HTTPS/Certificates/1",
      "@odata.type": "#Certificate.v1_6_0.Certificate",
      "CertificateString": "",
      "CertificateType": "PEM",
      "CertificateUsageTypes": ["Web"],
      "Fingerprint": "3A:7B:11",
      "FingerprintHashAlgorithm": "SHA256",
      "Issuer": {"CommonName": "Example Issuing CA", "Organization": "Example"},
      "SerialNumber": "1000",
      "SignatureAlgorithm": "1.2.840.113549.1.1.11",
      "Subject": {"CommonName": "bmc-01.example.com", "Organization": "Example"},
      "ValidNotAfter": "2027-01-01T00:00:00Z",
      "ValidNotBefore": "2026-01-01T00:00:00Z"
    }`

	cert, err := parseCertificate([]byte(raw))
	if err != nil {
		t.Fatalf("parseCertificate() error = %v", err)
	}
	if cert.Subject.CommonName != "bmc-01.example.com" {
		t.Errorf("got subject %q", cert.Subject.CommonName)
	}
	if cert.Issuer.CommonName != "Example Issuing CA" {
		t.Errorf("got issuer %q", cert.Issuer.CommonName)
	}
	if cert.ValidNotAfter != "2027-01-01T00:00:00Z" {
		t.Errorf("got expiry %q", cert.ValidNotAfter)
	}
	if len(cert.CertificateUsageTypes) != 1 || cert.CertificateUsageTypes[0] != "Web" {
		t.Errorf("got usage types %v", cert.CertificateUsageTypes)
	}

	if _, err := parseCertificate([]byte(`{`)); err == nil {
		t.Error("parseCertificate() accepted malformed input")
	}
}

func TestCertificateKeyInfo(t *testing.T) {
	rsaKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("could not generate an RSA key: %v", err)
	}
	ecKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("could not generate an ECDSA key: %v", err)
	}

	tests := []struct {
		name     string
		pemBody  string
		wantBits int64
		wantAlgo string
	}{
		{
			name:     "RSA 2048",
			pemBody:  selfSignedPEM(t, rsaKey, &rsaKey.PublicKey),
			wantBits: 2048,
			wantAlgo: "RSA",
		},
		{
			name:     "ECDSA P-256",
			pemBody:  selfSignedPEM(t, ecKey, &ecKey.PublicKey),
			wantBits: 256,
			wantAlgo: "ECDSA",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bits, algo := certificateKeyInfo(parsePEMCertificate(tt.pemBody))
			if bits == nil {
				t.Fatal("key size is null, want a value")
			}
			if *bits != tt.wantBits || algo != tt.wantAlgo {
				t.Errorf("got (%d, %q), want (%d, %q)", *bits, algo, tt.wantBits, tt.wantAlgo)
			}
		})
	}
}

func TestParsePEMCertificateRejectsUnusableBodies(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{name: "empty string"},
		{name: "not PEM", body: "MIIB..."},
		{name: "wrong PEM type", body: "-----BEGIN PRIVATE KEY-----\nAAAA\n-----END PRIVATE KEY-----\n"},
		{name: "PEM with broken body", body: "-----BEGIN CERTIFICATE-----\nAAAA\n-----END CERTIFICATE-----\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := parsePEMCertificate(tt.body); got != nil {
				t.Error("parsePEMCertificate() returned a certificate, want nil")
			}
			bits, algo := certificateKeyInfo(nil)
			if bits != nil || algo != "" {
				t.Errorf("certificateKeyInfo(nil) = (%v, %q), want (null, \"\")", bits, algo)
			}
		})
	}
}

func TestCertificateSelfSigned(t *testing.T) {
	rsaKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("could not generate an RSA key: %v", err)
	}
	parsed := parsePEMCertificate(selfSignedPEM(t, rsaKey, &rsaKey.PublicKey))
	if parsed == nil {
		t.Fatal("could not parse the generated certificate")
	}

	t.Run("encoded certificate wins", func(t *testing.T) {
		// Redfish reports a different issuer, the encoded certificate decides.
		cert := &certificateJSON{
			Issuer:  certIdentifierJSON{CommonName: "Example Issuing CA"},
			Subject: certIdentifierJSON{CommonName: "bmc-01.example.com"},
		}
		got := certificateSelfSigned(cert, parsed)
		if got == nil || !*got {
			t.Errorf("got %v, want true", got)
		}
	})

	t.Run("falls back to reported identity", func(t *testing.T) {
		cert := &certificateJSON{
			Issuer:  certIdentifierJSON{CommonName: "Example Issuing CA", Organization: "Example"},
			Subject: certIdentifierJSON{CommonName: "bmc-01.example.com", Organization: "Example"},
		}
		got := certificateSelfSigned(cert, nil)
		if got == nil || *got {
			t.Errorf("got %v, want false", got)
		}
	})

	t.Run("same reported identity is self signed", func(t *testing.T) {
		cert := &certificateJSON{
			Issuer:  certIdentifierJSON{CommonName: "bmc-01", Organization: "Example"},
			Subject: certIdentifierJSON{CommonName: "bmc-01", Organization: "Example"},
		}
		got := certificateSelfSigned(cert, nil)
		if got == nil || !*got {
			t.Errorf("got %v, want true", got)
		}
	})

	t.Run("null without identity", func(t *testing.T) {
		if got := certificateSelfSigned(&certificateJSON{}, nil); got != nil {
			t.Errorf("got %v, want null", *got)
		}
		if got := certificateSelfSigned(nil, nil); got != nil {
			t.Errorf("got %v, want null", *got)
		}
	})
}

func TestIsDefaultVendorAccountName(t *testing.T) {
	tests := []struct {
		userName string
		want     bool
	}{
		{userName: "root", want: true},
		{userName: "Administrator", want: true},
		{userName: "ADMIN", want: true},
		{userName: "USERID", want: true},
		{userName: "sysadmin", want: true},
		{userName: "anonymous", want: true},
		{userName: " root ", want: true},
		{userName: "rootca"},
		{userName: "svc-mondoo"},
		{userName: "administrators"},
		{userName: ""},
	}

	for _, tt := range tests {
		t.Run(tt.userName, func(t *testing.T) {
			if got := isDefaultVendorAccountName(tt.userName); got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsPersistentBootOverride(t *testing.T) {
	tests := []struct {
		name    string
		enabled string
		target  string
		want    bool
	}{
		{name: "continuous pxe", enabled: "Continuous", target: "Pxe", want: true},
		{name: "continuous usb", enabled: "Continuous", target: "Usb", want: true},
		{name: "continuous lower case", enabled: "continuous", target: "Hdd", want: true},
		{name: "continuous without target", enabled: "Continuous", target: "None"},
		{name: "continuous with empty target", enabled: "Continuous"},
		{name: "once pxe", enabled: "Once", target: "Pxe"},
		{name: "disabled", enabled: "Disabled", target: "Pxe"},
		{name: "unreported"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isPersistentBootOverride(tt.enabled, tt.target); got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}
