package awsiam

import (
	"encoding/csv"
	"io"
)

// https://docs.aws.amazon.com/IAM/latest/UserGuide/id_credentials_getting-report.html
// keys:
// user
// arn
// user_creation_time
// password_enabled
// password_last_used
// password_last_changed
// password_next_rotation
// mfa_active
// access_key_1_active
// access_key_1_last_rotated
// access_key_1_last_used_date
// access_key_1_last_used_region
// access_key_1_last_used_service
// access_key_2_active
// access_key_2_last_rotated
// access_key_2_last_used_date
// access_key_2_last_used_region
// access_key_2_last_used_service
// cert_1_active
// cert_1_last_rotated
// cert_2_active
// cert_2_last_rotated

func Parse(r io.Reader) ([]map[string]interface{}, error) {
	csvr := csv.NewReader(r)
	csvr.Comma = ','

	idx := 0

	index := map[int]string{}

	result := []map[string]interface{}{}
	for {
		record, err := csvr.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		// parse headline
		if idx == 0 {
			for i := range record {
				index[i] = record[i]
			}
		} else {
			entry := map[string]interface{}{}
			for i := range record {
				entry[index[i]] = record[i]
			}
			result = append(result, entry)
		}
		idx++
	}

	return result, nil
}
