package plugin

import "go.mondoo.com/cnquery/motor/platform"

func IdentifyPlatform[T any](connection T, p *platform.Platform, identifiers ...func(T, *platform.Platform) (string, bool)) []string {
	res := []string{}
	for i := range identifiers {
		if x, found := identifiers[i](connection, p); found {
			res = append(res, x)
		}
	}
	return res
}
