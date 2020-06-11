package resources_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResource_AuthorizedKeys(t *testing.T) {
	t.Run("view authorized keys file", func(t *testing.T) {
		res := testQuery(t, "authorizedkeys('/home/chris/.ssh/authorized_keys').content")
		assert.NotEmpty(t, res)
		assert.Equal(t, 755, len(res[0].Data.Value.(string)))
	})

	// t.Run("test authorized keys type", func(t *testing.T) {
	// 	res := testQuery(t, "authorizedkeys('/home/chris/.ssh/authorized_keys').list[0].type")
	// 	assert.NotEmpty(t, res)
	// 	assert.Empty(t, res[0].Result().Error)
	// 	assert.Equal(t, "ssh-rsa", res[0].Data.Value)
	// })

	// t.Run("test authorized keys type", func(t *testing.T) {
	// 	res := testQuery(t, "authorizedkeys('/home/chris/.ssh/authorized_keys').list[0].label")
	// 	assert.NotEmpty(t, res)
	// 	assert.Empty(t, res[0].Result().Error)
	// 	assert.Equal(t, "chris@lollyrock.com", res[0].Data.Value)
	// })

	// t.Run("test that the user exists", func(t *testing.T) {
	// 	res := testQuery(t, "users.where( name == 'chris' ).list[0].authorizedkeys.list[0].type")
	// 	assert.NotEmpty(t, res)
	// 	assert.Empty(t, res[0].Result().Error)
	// 	assert.Equal(t, "ssh-rsa", res[0].Data.Value)
	// })
}
