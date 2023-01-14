package core_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResource_OpenPgg(t *testing.T) {
	res := x.TestQuery(t, "parse.openpgp(path: \"./testdata/expires.asc\").all( identities.all( signatures.all( keyExpiresIn.days > 30 )))")
	assert.NotEmpty(t, res)
}
