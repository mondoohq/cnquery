package awsec2ebs

import (
	"path"
	"regexp"
	"strings"

	"github.com/cockroachdb/errors"
)

type InstanceId struct {
	Id      string
	Region  string
	Account string
	Zone    string
}

func NewInstanceId(account string, region string, id string) (*InstanceId, error) {
	if region == "" || id == "" || account == "" {
		return nil, errors.New("invalid instance id. account, region and instance id required.")
	}
	return &InstanceId{Account: account, Region: region, Id: id}, nil
}

func (s *InstanceId) String() string {
	// e.g. account/999000999000/region/us-east-2/instance/i-0989478343232
	return path.Join("account", s.Account, "region", s.Region, "instance", s.Id)
}

func ParseInstanceId(path string) (*InstanceId, error) {
	if !IsValidInstanceId(path) {
		return nil, errors.New("invalid instance id. expected account/<id>/region/<region-val>/instance/<instance-id>")
	}
	keyValues := strings.Split(path, "/")
	if len(keyValues) != 6 {
		return nil, errors.New("invalid instance id. expected account/<id>/region/<region-val>/instance/<instance-id>")
	}
	return NewInstanceId(keyValues[1], keyValues[3], keyValues[5])
}

var VALID_INSTANCE_ID = regexp.MustCompile(`^account/\d{12}/region\/(us(-gov)?|ap|ca|cn|eu|sa)-(central|(north|south)?(east|west)?)-\d\/instance\/.+$`)

func IsValidInstanceId(path string) bool {
	return VALID_INSTANCE_ID.MatchString(path)
}

type SnapshotId struct {
	Id      string
	Region  string
	Account string
}

type VolumeId struct {
	Id      string
	Region  string
	Account string
}

type FsType int64

const (
	Xfs FsType = iota
	Ext4
)

func (v FsType) String() string {
	switch v {
	case Xfs:
		return "xfs"
	case Ext4:
		return "ext4"
	}
	return "xfs"
}
