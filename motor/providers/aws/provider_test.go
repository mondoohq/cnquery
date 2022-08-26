package aws

// import (
// 	"testing"

// 	"github.com/aws/aws-sdk-go-v2/aws/endpoints"
// 	"github.com/stretchr/testify/require"
// 	"go.mondoo.com/cnquery/motor/transports"
// )

// func TestAwsTransport(t *testing.T) {

// 	pCfg := &transports.TransportConfig{
// 		Type: transports.TransportBackend_CONNECTION_AWS,
// 		Options: map[string]string{
// 			"profile": "mondoo-inc",
// 			"region":  endpoints.UsEast1RegionID,
// 		},
// 	}

// 	p, err := New(pCfg)
// 	require.NoError(t, err)

// 	info, err := p.Account()
// 	require.NoError(t, err)
// }
