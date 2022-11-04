//go:build debugtest
// +build debugtest

package okta

import (
	"context"
	"fmt"
	"testing"

	"github.com/okta/okta-sdk-golang/v2/okta"
	"github.com/okta/okta-sdk-golang/v2/okta/query"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	org   = "dev-12345.okta.com"
	token = "<token goes here>"
)

func TestOkta(t *testing.T) {
	ctx, client, err := okta.NewClient(
		context.TODO(),
		okta.WithOrgUrl("https://"+org),
		okta.WithToken(token),
	)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	}

	fmt.Printf("Context: %+v\n Client: %+v\n", ctx, client)

	users, _, err := client.User.ListUsers(context.Background(), nil)
	require.NoError(t, err)
	assert.NotNil(t, users)

	// second call
	users, resp, err := client.User.ListUsers(
		ctx,
		query.NewQueryParams(
			query.WithLimit(200),
		),
	)
	require.NoError(t, err)

	for resp != nil && resp.HasNextPage() {
		var userSetSlice []*okta.User
		resp, err = resp.Next(ctx, &userSetSlice)
		require.NoError(t, err)
	}
}
