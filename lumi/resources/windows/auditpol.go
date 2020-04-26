package windows

import (
	"encoding/csv"
	"io"
	"strings"
)

// Machine Name,Policy Target,Subcategory,Subcategory GUID,Inclusion Setting,Exclusion Setting
// Test,System,Security System Extension,{0CCE9211-69AE-11D9-BED3-505054503030},No Auditing,
type AuditpolEntry struct {
	MachineName      string
	PolicyTarget     string
	Subcategory      string
	SubcategoryGUID  string
	InclusionSetting string
	ExclusionSetting string
}

// see https://docs.microsoft.com/en-us/openspecs/windows_protocols/ms-gpac/77878370-0712-47cd-997d-b07053429f6d
func ParseAuditpol(r io.Reader) ([]AuditpolEntry, error) {
	res := []AuditpolEntry{}

	csvReader := csv.NewReader(r)
	for {
		record, err := csvReader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		guid := strings.TrimSpace(record[3])
		guid = strings.TrimPrefix(guid, "{")
		guid = strings.TrimSuffix(guid, "}")

		res = append(res, AuditpolEntry{
			MachineName:      strings.TrimSpace(record[0]),
			PolicyTarget:     strings.TrimSpace(record[1]),
			Subcategory:      strings.TrimSpace(record[2]),
			SubcategoryGUID:  guid,
			InclusionSetting: strings.TrimSpace(record[4]),
			ExclusionSetting: strings.TrimSpace(record[5]),
		})
	}

	return res, nil
}
