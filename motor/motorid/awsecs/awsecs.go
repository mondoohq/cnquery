package awsecsid

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/aws/aws-sdk-go/aws/arn"
	"errors"
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

var VALID_MONDOO_ECSCONTAINER_ID = regexp.MustCompile(`^//platformid.api.mondoo.app/runtime/aws/ecs/v1/accounts/\d{12}/regions\/(us(-gov)?|ap|ca|cn|eu|sa)-(central|(north|south)?(east|west)?)-\d\/container\/.+$`)

type ECSContainer struct {
	Account string
	Region  string
	Id      string
}

func ParseMondooECSContainerId(path string) (*ECSContainer, error) {
	if !IsValidMondooECSContainerId(path) {
		return nil, errors.New("invalid aws ecs container id")
	}
	keyValues := strings.Split(path, "/")
	if len(keyValues) != 15 {
		return nil, errors.New("invalid ecs container id length")
	}
	return &ECSContainer{Account: keyValues[8], Region: keyValues[10], Id: strings.Join(keyValues[12:], "/")}, nil
}

func IsValidMondooECSContainerId(path string) bool {
	return VALID_MONDOO_ECSCONTAINER_ID.MatchString(path)
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
