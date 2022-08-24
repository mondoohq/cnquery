package mrn

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMrnParser(t *testing.T) {
	var mrn *MRN
	var err error

	fullResourceName := "//service.example.com/key1/:value1:/key2/:value2:/key3/:value3:"
	mrn, err = NewMRN(fullResourceName)
	assert.Nil(t, err)
	assert.Equal(t, "service.example.com", mrn.ServiceName)
	assert.Equal(t, "service", ServiceID(mrn.ServiceName, ".example.com"))
	assert.Equal(t, "key1/:value1:/key2/:value2:/key3/:value3:", mrn.RelativeResourceName)
	assert.Equal(t, fullResourceName, mrn.String())
}

func TestCollectionID(t *testing.T) {
	fullResourceName := "//service.example.com/key1/:value1:/key2/:value2:/key3/:value3:"
	mrn, err := NewMRN(fullResourceName)
	assert.Nil(t, err)

	space, err := mrn.ResourceID("key1")
	require.NoError(t, err)
	assert.Equal(t, ":value1:", space)
	region, err := mrn.ResourceID("key2")
	require.NoError(t, err)
	assert.Equal(t, ":value2:", region)
	asset, err := mrn.ResourceID("key3")
	require.NoError(t, err)
	assert.Equal(t, ":value3:", asset)
}

func TestEquals(t *testing.T) {
	fullResourceName := "//service.example.com/key1/:value1:/key2/:value2:/key3/:value3:"
	mrn, err := NewMRN(fullResourceName)
	require.NoError(t, err)
	assert.True(t, mrn.Equals(fullResourceName))

	mrn, err = NewMRN("//service.example.com/key1/:value1:/key2/:value2:/key3/:value4:")
	require.NoError(t, err)
	assert.False(t, mrn.Equals(fullResourceName))
}
