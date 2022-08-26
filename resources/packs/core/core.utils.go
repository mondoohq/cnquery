package core

import (
	"encoding/json"
	"fmt"
	"time"

	"go.mondoo.com/cnquery/motor"
	"go.mondoo.com/cnquery/motor/providers/os"
)

func osProvider(m *motor.Motor) (os.OperatingSystemProvider, error) {
	provider, ok := m.Provider.(os.OperatingSystemProvider)
	if !ok {
		return nil, fmt.Errorf("provider is not an operating system provider")
	}
	return provider, nil
}

func MqlTime(t time.Time) *time.Time {
	return &t
}

func ToString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func ToBool(b *bool) bool {
	if b == nil {
		return false
	}
	return *b
}

func StrSliceToInterface(value []string) []interface{} {
	res := make([]interface{}, len(value))
	for i := range value {
		res[i] = value[i]
	}
	return res
}

func ToStringSlice(in *[]string) []interface{} {
	if in == nil {
		return []interface{}{}
	}
	slice := *in

	res := []interface{}{}
	for i := range slice {
		res = append(res, slice[i])
	}

	return res
}

func ToFloat64(i *float64) float64 {
	if i == nil {
		return 0
	}
	return float64(*i)
}

func ToInt64From32(i *int32) int64 {
	if i == nil {
		return int64(0)
	}
	return int64(*i)
}

func ToInt(i *int) int64 {
	if i == nil {
		return int64(0)
	}
	return int64(*i)
}

func ToIntFrom32(i *int32) int {
	if i == nil {
		return int(0)
	}
	return int(*i)
}

func ToInt64(i *int64) int64 {
	if i == nil {
		return 0
	}
	return *i
}

func ToInt32(i *int32) int32 {
	if i == nil {
		return 0
	}
	return *i
}

// JsonToDict converts a raw golang object (typically loaded from JSON)
// into a `dict` type
func JsonToDict(v interface{}) (map[string]interface{}, error) {
	res := make(map[string]interface{})

	data, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal([]byte(data), &res)
	if err != nil {
		return nil, err
	}
	return res, nil
}

// JsonToDict converts a raw golang object (typically loaded from JSON)
// into an array of `dict` types
func JsonToDictSlice(v interface{}) ([]interface{}, error) {
	res := []interface{}{}

	data, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal([]byte(data), &res)
	if err != nil {
		return nil, err
	}
	return res, nil
}

func StrMapToInterface(data map[string]string) map[string]interface{} {
	labels := make(map[string]interface{}, len(data))
	for key := range data {
		labels[key] = data[key]
	}
	return labels
}
