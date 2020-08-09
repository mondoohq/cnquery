package resources

import (
	"encoding/json"
	"time"

	"github.com/Azure/go-autorest/autorest/date"
	uuid "github.com/satori/go.uuid"
)

func (a *lumiAzurerm) id() (string, error) {
	return "azurerm", nil
}

func azureTagsToInterface(data map[string]*string) map[string]interface{} {
	labels := make(map[string]interface{})
	for key := range data {
		labels[key] = toString(data[key])
	}
	return labels
}

func azureRmTime(d *date.Time) time.Time {
	if d == nil {
		return time.Time{}
	}
	return d.Time
}

func azureRmUnixTime(d *date.UnixTime) time.Time {
	if d == nil {
		return time.Time{}
	}

	return time.Time(*d)
}

// TODO: double-check if lumi supports float
func toFloat64(i *float64) int64 {
	if i == nil {
		return 0
	}
	return int64(*i)
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

func uuidToString(u *uuid.UUID) string {
	if u == nil {
		return ""
	}
	return u.String()
}

func jsonToDict(v interface{}) (map[string]interface{}, error) {
	res := make(map[string](interface{}))

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
