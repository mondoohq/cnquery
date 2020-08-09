package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type azureIdTestCase struct {
	url      string
	expected keyvaultid
}

func TestAzureKeyvaultIdParser(t *testing.T) {

	testCases := []azureIdTestCase{
		{url: "https://superdupertestkey.vault.azure.net/certificates/testcertificate",
			expected: keyvaultid{
				BaseUrl: "https://superdupertestkey.vault.azure.net",
				Vault:   "superdupertestkey",
				Type:    "certificates",
				Name:    "testcertificate",
			}},
		{url: "https://superdupertestkey.vault.azure.net/certificates/testcertificate/c2fcb0ffb06d4cfead8240b4a06b7c63",
			expected: keyvaultid{
				BaseUrl: "https://superdupertestkey.vault.azure.net",
				Vault:   "superdupertestkey",
				Type:    "certificates",
				Name:    "testcertificate",
				Version: "c2fcb0ffb06d4cfead8240b4a06b7c63",
			}},
		{url: "https://superdupertestkey.vault.azure.net/secrets/testcertificate",
			expected: keyvaultid{
				BaseUrl: "https://superdupertestkey.vault.azure.net",
				Vault:   "superdupertestkey",
				Type:    "secrets",
				Name:    "testcertificate",
			}},
		{url: "https://superdupertestkey.vault.azure.net/secrets/Test",
			expected: keyvaultid{
				BaseUrl: "https://superdupertestkey.vault.azure.net",
				Vault:   "superdupertestkey",
				Type:    "secrets",
				Name:    "Test",
			}},
		{url: "https://superdupertestkey.vault.azure.net/keys/test",
			expected: keyvaultid{
				BaseUrl: "https://superdupertestkey.vault.azure.net",
				Vault:   "superdupertestkey",
				Type:    "keys",
				Name:    "test",
			}},
	}

	for i := range testCases {
		val, err := parseKeyVaultId(testCases[i].url)
		require.NoError(t, err, testCases[i].url)
		assert.Equal(t, testCases[i].expected, *val)
	}
}
