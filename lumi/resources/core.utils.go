package resources

import "time"

func LumiTime(t time.Time) *time.Time {
	return &t
}

func strSliceToInterface(value []string) []interface{} {
	res := make([]interface{}, len(value))
	for i := range value {
		res[i] = value[i]
	}
	return res
}
