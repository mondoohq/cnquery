package resources

import (
	"encoding/json"
	"errors"

	vt "github.com/VirusTotal/vt-go"
	"go.mondoo.com/cnquery/v12/providers-sdk/v1/util/convert"
)

var errUnexpectedAnalysisStatsFormat = errors.New("unexpected format for last_analysis_stats")

func stringMapAttribute(obj *vt.Object, attribute string) (map[string]any, bool, error) {
	rawValue, err := obj.Get(attribute)
	if err != nil {
		return nil, true, err
	}

	if rawValue == nil {
		return nil, true, nil
	}

	typed, ok := rawValue.(map[string]any)
	if !ok {
		return nil, true, nil
	}

	normalized := make(map[string]string, len(typed))
	for key, val := range typed {
		if val == nil {
			continue
		}

		strVal, ok := val.(string)
		if !ok {
			return nil, true, nil
		}
		normalized[key] = strVal
	}

	return convert.MapToInterfaceMap(normalized), false, nil
}

func analysisStatsAttribute(obj *vt.Object) (map[string]any, int64, bool, error) {
	rawValue, err := obj.Get("last_analysis_stats")
	if err != nil {
		return nil, 0, true, err
	}

	if rawValue == nil {
		return nil, 0, true, nil
	}

	typed, ok := rawValue.(map[string]any)
	if !ok {
		return nil, 0, false, errUnexpectedAnalysisStatsFormat
	}

	normalized := make(map[string]int64, len(typed))
	var detections int64

	for key, val := range typed {
		if val == nil {
			continue
		}

		num, ok := toInt64(val)
		if !ok {
			return nil, 0, false, errUnexpectedAnalysisStatsFormat
		}

		normalized[key] = num

		if key == "malicious" || key == "suspicious" {
			detections += num
		}
	}

	return convert.MapToInterfaceMap(normalized), detections, false, nil
}

func toInt64(value any) (int64, bool) {
	switch v := value.(type) {
	case int:
		return int64(v), true
	case int8:
		return int64(v), true
	case int16:
		return int64(v), true
	case int32:
		return int64(v), true
	case int64:
		return v, true
	case uint:
		return int64(v), true
	case uint8:
		return int64(v), true
	case uint16:
		return int64(v), true
	case uint32:
		return int64(v), true
	case uint64:
		return int64(v), true
	case float32:
		return int64(v), true
	case float64:
		return int64(v), true
	case json.Number:
		n, err := v.Int64()
		return n, err == nil
	case string:
		num, err := json.Number(v).Int64()
		return num, err == nil
	default:
		return 0, false
	}
}

func isNotFoundError(err error) bool {
	if err == nil {
		return false
	}

	var apiErr vt.Error
	if errors.As(err, &apiErr) {
		return apiErr.Code == "NotFoundError"
	}

	return false
}
