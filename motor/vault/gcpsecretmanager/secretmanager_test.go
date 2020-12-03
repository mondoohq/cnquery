package gcpsecretmanager

// func TestGcpSecretmanager(t *testing.T) {

// 	projectID := "mondoo-dev-262313"
// 	v := New(projectID)
// 	ctx := context.Background()

// 	cred := &vault.Credential{
// 		Key: vault.Mrn2secretKey("//platformid.api.mondoo.app/runtime/aws/ec2/v1/accounts/675173580680/regions/eu-west-1/instances/i-0e11b0762369fbefa"),
// 		Fields: map[string]string{
// 			"key":  "value",
// 			"key2": "value2",
// 		},
// 	}

// 	id, err := v.Set(ctx, cred)
// 	require.NoError(t, err)

// 	newCred, err := v.Get(ctx, id)
// 	require.NoError(t, err)
// 	assert.Equal(t, cred, newCred)
// }
