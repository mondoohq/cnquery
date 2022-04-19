package windows

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseSecurityProductState(t *testing.T) {
	code := uint32(397568)
	res := parseProductState(code)
	assert.Equal(t, uint32(1), res.Owner)     // microsoft
	assert.Equal(t, uint32(0), res.Product)   // on
	assert.Equal(t, uint32(1), res.Signature) // up to date

	code = uint32(393216)
	res = parseProductState(code)
	assert.Equal(t, uint32(0), res.Owner)     // other
	assert.Equal(t, uint32(1), res.Product)   // off
	assert.Equal(t, uint32(1), res.Signature) // up to date

	code = uint32(397584)
	res = parseProductState(code)
	assert.Equal(t, uint32(1), res.Owner)     // microsoft
	assert.Equal(t, uint32(0), res.Product)   // on
	assert.Equal(t, uint32(0), res.Signature) // ouf to date
}

func TestSecurityProductsPowershell(t *testing.T) {
	r, err := os.Open("./testdata/security_products.json")
	require.NoError(t, err)

	products, err := ParseWindowsSecurityProducts(r)
	require.NoError(t, err)
	assert.True(t, len(products) == 1)

	assert.Equal(t, "Windows Defender", products[0].Name)
	assert.Equal(t, int64(397568), products[0].State)
	assert.Equal(t, "UP-TO-DATE", products[0].SignatureStatus)
	assert.Equal(t, "ON", products[0].ProductStatus)
}
