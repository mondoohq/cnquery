package resources

import (
	"errors"

	"go.mondoo.io/mondoo/motor/transports"
	aws_transport "go.mondoo.io/mondoo/motor/transports/aws"
)

func awstransport(t transports.Transport) (*aws_transport.Transport, error) {
	at, ok := t.(*aws_transport.Transport)
	if !ok {
		return nil, errors.New("aws resource is not supported on this transport")
	}
	return at, nil
}

func toString(i *string) string {
	if i == nil {
		return ""
	}
	return *i
}

func toBool(i *bool) bool {
	if i == nil {
		return false
	}
	return *i
}

func toInt64(i *int64) int64 {
	if i == nil {
		return 0
	}
	return *i
}

func toInt(i *int) int64 {
	if i == nil {
		return int64(0)
	}
	return int64(*i)
}
