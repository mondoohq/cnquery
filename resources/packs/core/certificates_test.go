package core_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// Example use for certificate parser:
// parse.certificates('/etc/ssl/cert.pem').list {
// 		fingerprints
// 		serial
// 		subjectKeyID
// 		authorityKeyID
// 		isCA
// 		version
// 		keyUsage
// 		extendedKeyUsage
// 		crlDistributionPoints
// 		ocspServer
// 		issuingCertificateUrl
// 		issuer { serialNumber commonName }
// 		subject {serialNumber commonName}
// 		policyidentifier
// 		extensions { identifier }
// }

func TestResource_ParseCertificates(t *testing.T) {
	t.Run("view authorized keys file", func(t *testing.T) {
		res := x.TestQuery(t, "parse.certificates('/etc/ssl/cert.pem').content")
		assert.NotEmpty(t, res)
		assert.Equal(t, 1207, len(res[0].Data.Value.(string)))
	})

	t.Run("test certificate serial", func(t *testing.T) {
		res := x.TestQuery(t, "parse.certificates('/etc/ssl/cert.pem').list[0].serial")
		assert.NotEmpty(t, res)
		assert.Empty(t, res[0].Result().Error)
		assert.Equal(t, "06:6c:9f:cf:99:bf:8c:0a:39:e2:f0:78:8a:43:e6:96:36:5b:ca", res[0].Data.Value)
	})

	t.Run("test certificate issuer commonname", func(t *testing.T) {
		res := x.TestQuery(t, "parse.certificates('/etc/ssl/cert.pem').list[0].issuer.commonName")
		assert.NotEmpty(t, res)
		assert.Empty(t, res[0].Result().Error)
		assert.Equal(t, "Amazon Root CA 1", res[0].Data.Value)
	})

	t.Run("test certificate issuer dn", func(t *testing.T) {
		res := x.TestQuery(t, "parse.certificates('/etc/ssl/cert.pem').list[0].issuer.dn")
		assert.NotEmpty(t, res)
		assert.Empty(t, res[0].Result().Error)
		assert.Equal(t, "CN=Amazon Root CA 1,O=Amazon,C=US", res[0].Data.Value)
	})

	t.Run("test certificate subjectkeyid", func(t *testing.T) {
		res := x.TestQuery(t, "parse.certificates('/etc/ssl/cert.pem').list[0].subjectKeyID")
		assert.NotEmpty(t, res)
		assert.Empty(t, res[0].Result().Error)
		assert.Equal(t, "84:18:cc:85:34:ec:bc:0c:94:94:2e:08:59:9c:c7:b2:10:4e:0a:08", res[0].Data.Value)
	})

	t.Run("test certificate authoritykeyid", func(t *testing.T) {
		res := x.TestQuery(t, "parse.certificates('/etc/ssl/cert.pem').list[0].authorityKeyID")
		assert.NotEmpty(t, res)
		assert.Empty(t, res[0].Result().Error)
		assert.Equal(t, "", res[0].Data.Value)
	})

	t.Run("test certificate version", func(t *testing.T) {
		res := x.TestQuery(t, "parse.certificates('/etc/ssl/cert.pem').list[0].version")
		assert.NotEmpty(t, res)
		assert.Empty(t, res[0].Result().Error)
		assert.Equal(t, int64(3), res[0].Data.Value)
	})

	t.Run("test certificate isca", func(t *testing.T) {
		res := x.TestQuery(t, "parse.certificates('/etc/ssl/cert.pem').list[0].isCA")
		assert.NotEmpty(t, res)
		assert.Empty(t, res[0].Result().Error)
		assert.Equal(t, true, res[0].Data.Value)
	})

	t.Run("test certificate keyusage", func(t *testing.T) {
		res := x.TestQuery(t, "parse.certificates('/etc/ssl/cert.pem').list[0].keyUsage")
		assert.NotEmpty(t, res)
		assert.Empty(t, res[0].Result().Error)
		list := res[0].Data.Value.([]interface{})
		assert.Contains(t, list, "CRLSign")
		assert.Contains(t, list, "DigitalSignature")
		assert.Contains(t, list, "CertificateSign")
	})

	t.Run("test certificate extendedkeyusage", func(t *testing.T) {
		res := x.TestQuery(t, "parse.certificates('/etc/ssl/cert.pem').list[0].extendedKeyUsage")
		assert.NotEmpty(t, res)
		assert.Empty(t, res[0].Result().Error)
		assert.Equal(t, []interface{}{}, res[0].Data.Value)
	})

	t.Run("test certificate crldistributionpoints", func(t *testing.T) {
		res := x.TestQuery(t, "parse.certificates('/etc/ssl/cert.pem').list[0].crlDistributionPoints")
		assert.NotEmpty(t, res)
		assert.Empty(t, res[0].Result().Error)
		assert.Equal(t, []interface{}{}, res[0].Data.Value)
	})

	t.Run("test certificate ocspserver", func(t *testing.T) {
		res := x.TestQuery(t, "parse.certificates('/etc/ssl/cert.pem').list[0].ocspServer")
		assert.NotEmpty(t, res)
		assert.Empty(t, res[0].Result().Error)
		assert.Equal(t, []interface{}{}, res[0].Data.Value)
	})

	t.Run("test certificate issuingcertificateurl", func(t *testing.T) {
		res := x.TestQuery(t, "parse.certificates('/etc/ssl/cert.pem').list[0].issuingCertificateUrl")
		assert.NotEmpty(t, res)
		assert.Empty(t, res[0].Result().Error)
		assert.Equal(t, []interface{}{}, res[0].Data.Value)
	})

	t.Run("test certificate loading from content", func(t *testing.T) {
		cert := `-----BEGIN CERTIFICATE-----
MIIFWDCCBECgAwIBAgIQaMJ5PP8vl9sQAAAAAAEvHjANBgkqhkiG9w0BAQsFADBG
MQswCQYDVQQGEwJVUzEiMCAGA1UEChMZR29vZ2xlIFRydXN0IFNlcnZpY2VzIExM
QzETMBEGA1UEAxMKR1RTIENBIDFENDAeFw0yMjAyMDYwOTI3MzJaFw0yMjA1MDcw
OTI3MzFaMBUxEzARBgNVBAMTCm1vbmRvby5jb20wggEiMA0GCSqGSIb3DQEBAQUA
A4IBDwAwggEKAoIBAQC4oVPC4ORJlZt/FEfrJ4g8gCBPKW0m9rH/e4J78jZTrsye
7w7tXFY7ZeHGQizEsJtfpsipwsldTOoCygDKWI/7xnx9AKe79wRfZecijV11s5MN
TfSlNSgaKZ5DAha8oVszAmPDxD6dDWqMPGL0XHw86aaBimnrh48930qBFwoKyf5I
cWCz77McF0PYNk57VDMB7BVIlthEvVmrSp9zloHOa78LoiexPOTHQSjAZTvnUiMn
EMRL3J9ZFYyshw56oE9hR3getBvlpwOKpS+5MSorOI5/ZSApn6ZF8c0F5IJVlTNR
T3ffKYz02Y4Rz348cgZkpo8t8Gp5/5OYoxjBRm81AgMBAAGjggJxMIICbTAOBgNV
HQ8BAf8EBAMCBaAwEwYDVR0lBAwwCgYIKwYBBQUHAwEwDAYDVR0TAQH/BAIwADAd
BgNVHQ4EFgQU5TBHEo55zzpw6/s3QckdsaprbtYwHwYDVR0jBBgwFoAUJeIYDrJX
kZQq5dRdhpCD3lOzuJIweAYIKwYBBQUHAQEEbDBqMDUGCCsGAQUFBzABhilodHRw
Oi8vb2NzcC5wa2kuZ29vZy9zL2d0czFkNC9za0xzTXRrWUpUczAxBggrBgEFBQcw
AoYlaHR0cDovL3BraS5nb29nL3JlcG8vY2VydHMvZ3RzMWQ0LmRlcjAVBgNVHREE
DjAMggptb25kb28uY29tMCEGA1UdIAQaMBgwCAYGZ4EMAQIBMAwGCisGAQQB1nkC
BQMwPAYDVR0fBDUwMzAxoC+gLYYraHR0cDovL2NybHMucGtpLmdvb2cvZ3RzMWQ0
L0VVQzBtUTR5TVBjLmNybDCCAQQGCisGAQQB1nkCBAIEgfUEgfIA8AB2AFGjsPX9
AXmcVm24N3iPDKR6zBsny/eeiEKaDf7UiwXlAAABfs6aMmoAAAQDAEcwRQIhAMy2
aufiYVITPFDElL1aWVMTo0rBEmQ520rXbTcfzI4JAiAawIFvNix2Vp3Ybuk7doHp
q/sICyNRt+Zrz/wNNfziegB2AEalVet1+pEgMLWiiWn0830RLEF0vv1JuIWr8vxw
/m1HAAABfs6aMoMAAAQDAEcwRQIhAJXJReJyMJskegnWDmfq0ovGZ90A7c9lYebj
7jfJyGGlAiABVuFTV0/jxdAV5XNOyUxN3Y3qhdeSfVM/82qPTub26zANBgkqhkiG
9w0BAQsFAAOCAQEAagCxD1/ctRgSA96MLhIKAey6CHmkECgGb4B+liuO1PwG+Ft9
x4KigQjZ193+z7aSb6CSxIEzUyDfGTMqmER1MOmN5wJhzw7pnZ0VXDLePcTJPqtA
q5uRwWdrXRKsoXPbizcs25btZNgcswHLOzNYxCT5Qf9pprxTcMoIlROFF6WT0wxq
pmYrmQ+eJ9Ny8Fi6ovMWlUch4qg3bcj6QQ0FZ3zPX/6kI9FXGvJ+4rL/WE3Ouc+b
XjazfGmfrd3uVevgxgkfeMsKtKgHCpr7f0qpqgko9F5De68JZg+lV/ganyOxKi5M
ym+AS505m2l07i2SYbM82nyP74qYD3b3QmrZSQ==
-----END CERTIFICATE-----`

		res := x.TestQuery(t, "parse.certificates(content: '"+cert+"').list[0].issuer.commonName")
		assert.NotEmpty(t, res)
		assert.Empty(t, res[0].Result().Error)
		assert.Equal(t, "GTS CA 1D4", res[0].Data.Value)
	})
}
