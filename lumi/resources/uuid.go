package resources

import (
	"errors"

	"github.com/google/uuid"
	"go.mondoo.io/mondoo/lumi"
)

func (u *lumiUuid) id() (string, error) {
	value, err := u.Value()
	return "uuid:" + value, err
}

func (s *lumiUuid) init(args *lumi.Args) (*lumi.Args, Uuid, error) {
	if x, ok := (*args)["value"]; ok {
		value, ok := x.(string)
		if !ok {
			return nil, nil, errors.New("Wrong type for 'value' in uuid initialization, it must be a string")
		}

		// ensure the value is a proper uuid
		u, err := uuid.Parse(value)
		if err != nil {
			return nil, nil, errors.New("invalid uuid")
		}

		(*args)["value"] = u.String()
	}

	return args, nil, nil
}

func (u *lumiUuid) getUuid() (uuid.UUID, error) {
	value, err := u.Value()
	if err != nil {
		return uuid.UUID{}, err
	}
	return uuid.Parse(value)
}

func (u *lumiUuid) GetUrn() (interface{}, error) {
	uid, err := u.getUuid()
	return uid.URN(), err
}

func (u *lumiUuid) GetVersion() (interface{}, error) {
	uid, err := u.getUuid()
	return int64(uid.Version()), err
}

func (u *lumiUuid) GetVariant() (string, error) {
	uid, err := u.getUuid()
	return uid.Variant().String(), err
}
