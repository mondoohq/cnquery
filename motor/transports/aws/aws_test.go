package aws

// import (
// 	"testing"

// 	"github.com/aws/aws-sdk-go-v2/aws/endpoints"
// 	"github.com/stretchr/testify/require"
// 	"go.mondoo.io/mondoo/motor/transports"
// )

// func TestAwsTransport(t *testing.T) {

// 	tc := &transports.TransportConfig{
// 		Backend: transports.TransportBackend_CONNECTION_AWS,
// 		Options: map[string]string{
// 			"profile": "mondoo-inc",
// 			"region":  endpoints.UsEast1RegionID,
// 		},
// 	}

// 	trans, err := New(tc)
// 	require.NoError(t, err)

// 	info, err := trans.Account()
// 	require.NoError(t, err)
// }
