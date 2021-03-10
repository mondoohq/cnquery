package resources

import (
	"encoding/json"
	"time"
)

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

func toFloat64(i *float64) float64 {
	if i == nil {
		return 0
	}
	return float64(*i)
}

func toStringSlice(in *[]string) []interface{} {
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

func jsonToDict(v interface{}) (map[string]interface{}, error) {
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

func jsonToDictSlice(v interface{}) ([]interface{}, error) {
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

func strMapToInterface(data map[string]string) map[string]interface{} {
	labels := make(map[string]interface{})
	for key := range data {
		labels[key] = data[key]
	}
	return labels
}
