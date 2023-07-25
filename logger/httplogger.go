package logger

import (
	"net/http"
	"net/http/httputil"

	"github.com/rs/zerolog/log"
)

type loggingTransport struct {
	transport http.RoundTripper
}

func (t *loggingTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	dump, err := httputil.DumpRequestOut(request, true)
	if err != nil {
		return nil, err
	}
	log.Trace().Msg(string(dump))
	return t.transport.RoundTrip(request)
}

func AttachLoggingTransport(client *http.Client) {
	client.Transport = &loggingTransport{
		transport: client.Transport,
	}
}
