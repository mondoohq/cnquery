// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"math/big"
	"net"
	"testing"
	"time"
)

// makeCert builds a self-signed certificate carrying the given SAN DNS names,
// SAN IP addresses, and Subject Common Name.
func makeCert(t *testing.T, commonName string, dnsNames []string, ips []net.IP) *x509.Certificate {
	t.Helper()

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: commonName},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
		DNSNames:     dnsNames,
		IPAddresses:  ips,
	}

	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("create certificate: %v", err)
	}

	cert, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatalf("parse certificate: %v", err)
	}
	return cert
}

func TestCertMatchesDomain(t *testing.T) {
	testCases := []struct {
		name       string
		commonName string
		dnsNames   []string
		ips        []net.IP
		domain     string
		want       bool
	}{
		{
			name:     "exact SAN match",
			dnsNames: []string{"example.com"},
			domain:   "example.com",
			want:     true,
		},
		{
			name:     "SAN mismatch",
			dnsNames: []string{"example.com"},
			domain:   "other.com",
			want:     false,
		},
		{
			name:     "wildcard covers subdomain",
			dnsNames: []string{"*.example.com"},
			domain:   "api.example.com",
			want:     true,
		},
		{
			name:     "wildcard does not cover bare apex",
			dnsNames: []string{"*.example.com"},
			domain:   "example.com",
			want:     false,
		},
		{
			name:     "wildcard does not cover nested subdomain",
			dnsNames: []string{"*.example.com"},
			domain:   "a.b.example.com",
			want:     false,
		},
		{
			name:     "case insensitive match",
			dnsNames: []string{"example.com"},
			domain:   "EXAMPLE.COM",
			want:     true,
		},
		{
			name:       "common name is not consulted when SANs present",
			commonName: "example.com",
			dnsNames:   []string{"other.com"},
			domain:     "example.com",
			want:       false,
		},
		{
			name:       "common name alone never matches",
			commonName: "example.com",
			domain:     "example.com",
			want:       false,
		},
		{
			name:   "ip SAN match",
			ips:    []net.IP{net.ParseIP("10.0.0.1")},
			domain: "10.0.0.1",
			want:   true,
		},
		{
			name:   "ip SAN mismatch",
			ips:    []net.IP{net.ParseIP("10.0.0.1")},
			domain: "10.0.0.2",
			want:   false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cert := makeCert(t, tc.commonName, tc.dnsNames, tc.ips)
			if got := certMatchesDomain(cert, tc.domain); got != tc.want {
				t.Errorf("certMatchesDomain(%q) = %v, want %v", tc.domain, got, tc.want)
			}
		})
	}
}
