package awsecsid

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws/arn"
	"github.com/cockroachdb/errors"
	"go.mondoo.com/cnquery/motor/platform"
	"go.mondoo.com/cnquery/motor/providers/local"
	"go.mondoo.com/cnquery/motor/providers/mock"
	"go.mondoo.com/cnquery/motor/providers/os"
)

func MondooECSContainerID(containerArn string) string {
	var account, region, id string
	if arn.IsARN(containerArn) {
		if p, err := arn.Parse(containerArn); err == nil {
			account = p.AccountID
			region = p.Region
			id = p.Resource
		}
	}
	return "//platformid.api.mondoo.app/runtime/aws/ecs/v1/accounts/" + account + "/regions/" + region + "/" + id
}

type Identity struct {
	ContainerArn      string
	Name              string
	RuntimeID         string
	PlatformIds       []string
	AccountPlatformID string
}
type InstanceIdentifier interface {
	Identify() (Identity, error)
}

func Resolve(provider os.OperatingSystemProvider, pf *platform.Platform) (InstanceIdentifier, error) {
	_, ok := provider.(*local.Provider)
	if ok {
		return NewContainerMetadata(provider, pf), nil
	}
	_, ok = provider.(*mock.Provider)
	if ok {
		return NewContainerMetadata(provider, pf), nil
	}

	return nil, errors.New(fmt.Sprintf("awsecs id detector is not supported for your asset: %s %s", pf.Name, pf.Version))
}
