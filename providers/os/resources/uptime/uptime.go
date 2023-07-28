package uptime

import (
	"errors"
	"time"

	"go.mondoo.com/cnquery/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/providers/os/connection/shared"
)

type Uptime interface {
	Name() string
	Duration() (time.Duration, error)
}

func New(conn shared.Connection) (Uptime, error) {
	pf := conn.Asset().Platform

	switch {
	case pf.IsFamily(inventory.FAMILY_UNIX):
		return &Unix{conn: conn}, nil
	case pf.IsFamily(inventory.FAMILY_WINDOWS):
		return &Windows{conn: conn}, nil
	default:
		return nil, errors.New("your platform is not supported by reboot resource")
	}
}
