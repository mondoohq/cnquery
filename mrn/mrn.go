package mrn

import (
	"errors"
	"net/url"
	"strings"
)

func ServiceID(serviceName string, baseDomain string) string {
	res := strings.TrimSuffix(serviceName, baseDomain)
	return res
}

func NewMRN(fullResourceName string) (*MRN, error) {
	u, err := url.Parse(fullResourceName)
	if err != nil {
		return nil, err
	}

	path := strings.TrimLeft(u.EscapedPath(), "/")

	return &MRN{
		ServiceName:          u.Host,
		RelativeResourceName: path,
	}, nil
}

// MRN follows Google's Design for resource names
// see https://cloud.google.com/apis/design/resource_names
type MRN struct {
	ServiceName          string
	RelativeResourceName string
}

func (mrn *MRN) String() string {
	return "//" + mrn.ServiceName + "/" + mrn.RelativeResourceName
}

func (mrn *MRN) ResourceID(collectionId string) (string, error) {
	keyValues := strings.Split(mrn.RelativeResourceName, "/")
	for i := 0; i < len(keyValues); {
		if keyValues[i] == collectionId {
			if i+1 < len(keyValues) {
				return keyValues[i+1], nil
			} else {
				return "", errors.New("invalid mrn collection id scheme: " + mrn.String())
			}
		}
		i = i + 2
	}
	return "", errors.New("could not find collection id: " + collectionId)
}

func (mrn *MRN) Equals(resource string) bool {
	parsed, err := NewMRN(resource)
	if err != nil {
		return false
	}
	if parsed.ServiceName == mrn.ServiceName && parsed.RelativeResourceName == mrn.RelativeResourceName {
		return true
	}
	return false
}

// SafeComponentString sanitizes a string so that it can be safely used as a uri component for mrns
func SafeComponentString(s string) string {
	if s == "" {
		return s
	}
	s = strings.ReplaceAll(s, " ", "_")
	s = strings.ReplaceAll(s, "/", "-")
	return s
}
