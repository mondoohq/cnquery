package resources_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// Example use for certificate parser:
// parse.certificates('/etc/ssl/cert.pem').list {
// 		fingerprints
// 		serial
// 		subjectkeyid
// 		authoritykeyid
// 		isca
// 		version
// 		keyusage
// 		extendedkeyusage
// 		crldistributionpoints
// 		ocspserver
// 		issuingcertificateurl
// 		issuer { serialnumber commonname }
// 		subject {serialnumber commonname}
// 		policyidentifier
// 		extensions { identifier }
// }

func TestResource_ParseCertificates(t *testing.T) {
	t.Run("view authorized keys file", func(t *testing.T) {
		res := testQuery(t, "parse.certificates('/etc/ssl/cert.pem').content")
		assert.NotEmpty(t, res)
		assert.Equal(t, 1207, len(res[0].Data.Value.(string)))
	})

	t.Run("test certificate serial", func(t *testing.T) {
		res := testQuery(t, "parse.certificates('/etc/ssl/cert.pem').list[0].serial")
		assert.NotEmpty(t, res)
		assert.Empty(t, res[0].Result().Error)
		assert.Equal(t, "06:6c:9f:cf:99:bf:8c:0a:39:e2:f0:78:8a:43:e6:96:36:5b:ca", res[0].Data.Value)
	})

	t.Run("test certificate issuer commonname", func(t *testing.T) {
		res := testQuery(t, "parse.certificates('/etc/ssl/cert.pem').list[0].issuer.commonname")
		assert.NotEmpty(t, res)
		assert.Empty(t, res[0].Result().Error)
		assert.Equal(t, "Amazon Root CA 1", res[0].Data.Value)
	})

	t.Run("test certificate issuer dn", func(t *testing.T) {
		res := testQuery(t, "parse.certificates('/etc/ssl/cert.pem').list[0].issuer.dn")
		assert.NotEmpty(t, res)
		assert.Empty(t, res[0].Result().Error)
		assert.Equal(t, "CN=Amazon Root CA 1,O=Amazon,C=US", res[0].Data.Value)
	})

	t.Run("test certificate subjectkeyid", func(t *testing.T) {
		res := testQuery(t, "parse.certificates('/etc/ssl/cert.pem').list[0].subjectkeyid")
		assert.NotEmpty(t, res)
		assert.Empty(t, res[0].Result().Error)
		assert.Equal(t, "84:18:cc:85:34:ec:bc:0c:94:94:2e:08:59:9c:c7:b2:10:4e:0a:08", res[0].Data.Value)
	})

	t.Run("test certificate authoritykeyid", func(t *testing.T) {
		res := testQuery(t, "parse.certificates('/etc/ssl/cert.pem').list[0].authoritykeyid")
		assert.NotEmpty(t, res)
		assert.Empty(t, res[0].Result().Error)
		assert.Equal(t, "", res[0].Data.Value)
	})

	t.Run("test certificate version", func(t *testing.T) {
		res := testQuery(t, "parse.certificates('/etc/ssl/cert.pem').list[0].version")
		assert.NotEmpty(t, res)
		assert.Empty(t, res[0].Result().Error)
		assert.Equal(t, int64(3), res[0].Data.Value)
	})

	t.Run("test certificate isca", func(t *testing.T) {
		res := testQuery(t, "parse.certificates('/etc/ssl/cert.pem').list[0].isca")
		assert.NotEmpty(t, res)
		assert.Empty(t, res[0].Result().Error)
		assert.Equal(t, true, res[0].Data.Value)
	})

	t.Run("test certificate keyusage", func(t *testing.T) {
		res := testQuery(t, "parse.certificates('/etc/ssl/cert.pem').list[0].keyusage")
		assert.NotEmpty(t, res)
		assert.Empty(t, res[0].Result().Error)
		list := res[0].Data.Value.([]interface{})
		assert.Contains(t, list, "CRLSign")
		assert.Contains(t, list, "DigitalSignature")
		assert.Contains(t, list, "CertificateSign")
	})

	t.Run("test certificate extendedkeyusage", func(t *testing.T) {
		res := testQuery(t, "parse.certificates('/etc/ssl/cert.pem').list[0].extendedkeyusage")
		assert.NotEmpty(t, res)
		assert.Empty(t, res[0].Result().Error)
		assert.Equal(t, []interface{}{}, res[0].Data.Value)
	})

	t.Run("test certificate crldistributionpoints", func(t *testing.T) {
		res := testQuery(t, "parse.certificates('/etc/ssl/cert.pem').list[0].crldistributionpoints")
		assert.NotEmpty(t, res)
		assert.Empty(t, res[0].Result().Error)
		assert.Equal(t, []interface{}{}, res[0].Data.Value)
	})

	t.Run("test certificate ocspserver", func(t *testing.T) {
		res := testQuery(t, "parse.certificates('/etc/ssl/cert.pem').list[0].ocspserver")
		assert.NotEmpty(t, res)
		assert.Empty(t, res[0].Result().Error)
		assert.Equal(t, []interface{}{}, res[0].Data.Value)
	})

	t.Run("test certificate issuingcertificateurl", func(t *testing.T) {
		res := testQuery(t, "parse.certificates('/etc/ssl/cert.pem').list[0].issuingcertificateurl")
		assert.NotEmpty(t, res)
		assert.Empty(t, res[0].Result().Error)
		assert.Equal(t, []interface{}{}, res[0].Data.Value)
	})
}
