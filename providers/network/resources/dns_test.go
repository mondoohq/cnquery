package resources_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResource_DNS(t *testing.T) {
	res := x.TestQuery(t, "dns(\"mondoo.com\").mx")
	assert.NotEmpty(t, res)
}

func TestResource_DomainName(t *testing.T) {
	res := x.TestQuery(t, "domainName")
	assert.NotEmpty(t, res)
	res = x.TestQuery(t, "domainName(\"mondoo.com\").tld")
	assert.Equal(t, "com", string(res[0].Result().Data.Value))
}
