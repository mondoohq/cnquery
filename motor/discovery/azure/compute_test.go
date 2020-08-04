package azure_test

// import (
// 	"context"
// 	"testing"

// 	"github.com/stretchr/testify/assert"
// 	"github.com/stretchr/testify/require"
// 	"go.mondoo.io/mondoo/motor/discovery/azure"
// )

// func TestAzureInstanceFetch(t *testing.T) {
// 	subscriptionid := "/subscriptions/123456/resourceGroups/name"

// 	client, err := azure.NewCompute(subscriptionid)
// 	require.NoError(t, err)

// 	ctx := context.Background()
// 	instances, err := client.ListInstances(ctx)
// 	require.NoError(t, err)

// 	assert.Equal(t, 1, len(instances))
// }
