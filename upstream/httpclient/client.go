package httpclient

import (
	"net/http"

	"go.mondoo.com/cnquery/apps/cnquery/cmd/proxy"
	"go.mondoo.com/ranger-rpc"
)

// NewClient will set up the underlyig ranger http client
// (with the appropriate proxy if needed)
func NewClient() (*http.Client, error) {
	rangerClient, err := ranger.NewHttpClient(&ranger.HttpClientOpts{
		Proxy: proxy.GetAPIProxy(),
	})
	if err != nil {
		return nil, err
	}

	return rangerClient, nil
}
