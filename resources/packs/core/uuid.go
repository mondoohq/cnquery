package core

import (
	"errors"

	"github.com/google/uuid"
	"go.mondoo.io/mondoo/resources"
)

func (u *mqlUuid) id() (string, error) {
	value, err := u.Value()
	return "uuid:" + value, err
}

func (s *mqlUuid) init(args *resources.Args) (*resources.Args, Uuid, error) {
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

func (u *mqlUuid) getUuid() (uuid.UUID, error) {
	value, err := u.Value()
	if err != nil {
		return uuid.UUID{}, err
	}
	return uuid.Parse(value)
}

func (u *mqlUuid) GetUrn() (interface{}, error) {
	uid, err := u.getUuid()
	return uid.URN(), err
}

func (u *mqlUuid) GetVersion() (interface{}, error) {
	uid, err := u.getUuid()
	return int64(uid.Version()), err
}

func (u *mqlUuid) GetVariant() (string, error) {
	uid, err := u.getUuid()
	return uid.Variant().String(), err
}
