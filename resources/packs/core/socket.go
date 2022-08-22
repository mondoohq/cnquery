package core

import "strconv"

func (s *mqlSocket) id() (string, error) {
	protocol, err := s.Protocol()
	if err != nil {
		return "", err
	}

	address, err := s.Address()
	if err != nil {
		return "", err
	}

	port, err := s.Port()
	if err != nil {
		return "", err
	}

	return protocol + "://" + address + ":" + strconv.Itoa(int(port)), nil
}
