package rangerclient

import (
	"net/http"

	"go.mondoo.com/ranger-rpc"
)

type RangerClientOpts struct {
	Proxy string
}

// NewRangerClient will set up the underlyig ranger client
// with the appropriate proxy if needed.
func NewRangerClient(opts *RangerClientOpts) (*http.Client, error) {
	var proxy string
	if opts != nil {
		proxy = opts.Proxy
	}

	rangerClient, err := ranger.NewHttpClient(&ranger.HttpClientOpts{
		Proxy: proxy,
	})
	if err != nil {
		return nil, err
	}

	return rangerClient, nil
}
