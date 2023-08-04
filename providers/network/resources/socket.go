package resources

import "strconv"

func (s *mqlSocket) id() (string, error) {
	return s.Protocol.Data + "://" + s.Address.Data + ":" + strconv.Itoa(int(s.Port.Data)), nil
}
