package resources

import (
	"errors"
	"time"

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

func toTime(i *time.Time) int64 {
	if i == nil {
		return 0
	}
	return i.UnixNano()
}

func toInt64(i *int64) int64 {
	if i == nil {
		return 0
	}
	return *i
}
