package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResource_Secpol(t *testing.T) {
	t.Run("list systemaccess", func(t *testing.T) {
		res := testQuery(t, "secpol.systemaccess")
		assert.NotEmpty(t, res)
	})

	// works in shell but not here
	// t.Run("test a specific secpol systemaccess entry", func(t *testing.T) {
	// 	res := testQuery(t, "secpol.systemaccess['PasswordHistorySize']")
	// 	assert.NotEmpty(t, res)
	// 	assert.Empty(t, res[0].Result().Error)
	// 	assert.Equal(t, 0, res[0].Data.Value)
	// })

	// // panics here and on shell
	// t.Run("test a specific secpol systemaccess entry", func(t *testing.T) {
	// 	res := testQuery(t, "secpol.systemaccess['PasswordHistorySize'] >=24")
	// 	assert.NotEmpty(t, res)
	// 	assert.Empty(t, res[0].Result().Error)
	// 	assert.Equal(t, 0, res[0].Data.Value)
	// })

	// // panics here and on shell, should be false
	// t.Run("test a specific secpol systemaccess entry", func(t *testing.T) {
	// 	res := testQuery(t, "secpol.systemaccess['PasswordHistorySize'] == 24")
	// 	assert.NotEmpty(t, res)
	// 	assert.Empty(t, res[0].Result().Error)
	// 	assert.Equal(t, 0, res[0].Data.Value)
	// })

	// // panics
	// // {
	// // 	"CodeID": "pQ0D0cMlTKOiL2kwOB1xPYwbatifTb6GEtjZA4xMZZNzQRVa3pvFwG9flmEeu8ok63LUfj0jfwHL3NnPw9Fd+g==",
	// // 	"Data": {
	// // 		"Error": null,
	// // 		"Type": "\u0007",
	// // 		"Value": [
	// // 			"S-1-1-0",
	// // 			"S-1-5-32-544",
	// // 			"S-1-5-32-545",
	// // 			"S-1-5-32-551"
	// // 		]
	// // 	}
	// // }
	// t.Run("test a specific secpol systemaccess entry", func(t *testing.T) {
	// 	res := testQuery(t, "secpol.privilegerights['SeNetworkLogonRight']")
	// 	assert.NotEmpty(t, res)
	// 	assert.Empty(t, res[0].Result().Error)
	// 	assert.Equal(t, []string{
	// 		"S-1-1-0",
	// 		"S-1-5-32-544",
	// 		"S-1-5-32-545",
	// 		"S-1-5-32-551",
	// 	}, res[0].Data.Value)
	// })

	// // this should be equal but is not
	// t.Run("test a specific secpol systemaccess entry", func(t *testing.T) {
	// 	res := testQuery(t, "secpol.privilegerights['SeNetworkLogonRight'] == ['S-1-1-0', 'S-1-5-32-544', 'S-1-5-32-545', 'S-1-5-32-551']")
	// 	assert.NotEmpty(t, res)
	// 	assert.Empty(t, res[0].Result().Error)
	// 	assert.Equal(t, true , res[0].Data.Value)
	// })
	// // also how to make array comparison evectively
	// // how to check array includes one value?
}
