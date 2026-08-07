// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"reflect"
	"testing"

	"github.com/Snowflake-Labs/terraform-provider-snowflake/pkg/sdk"
)

func TestSQLTagName(t *testing.T) {
	cases := []struct {
		name string
		tag  string
		want string
	}{
		{"bare name", "SKIP_HEADER", "SKIP_HEADER"},
		{"surrounding space trimmed", "  SKIP_HEADER  ", "SKIP_HEADER"},
		// The SDK's sql tags are bare today, but its ddl tags in the same
		// structs are comma-separated. If that convention ever spreads, the
		// option name must survive rather than becoming a wrong dict key.
		{"comma options dropped", "SKIP_HEADER,keyword", "SKIP_HEADER"},
		{"comma options with space", "SKIP_HEADER, keyword", "SKIP_HEADER"},
		{"empty tag", "", ""},
		{"leading comma", ",keyword", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := sqlTagName(tc.tag); got != tc.want {
				t.Errorf("sqlTagName(%q) = %q, want %q", tc.tag, got, tc.want)
			}
		})
	}
}

func TestFileFormatOptionsToDict(t *testing.T) {
	t.Run("csv keeps only csv options", func(t *testing.T) {
		opts := sdk.FileFormatTypeOptions{
			CSVSkipHeader:                 sdk.Int(1),
			CSVFieldDelimiter:             sdk.String(","),
			CSVParseHeader:                sdk.Bool(true),
			CSVErrorOnColumnCountMismatch: sdk.Bool(false),
			CSVCompression:                &[]sdk.CSVCompression{sdk.CSVCompressionGzip}[0],
			// options of other types must not leak into a CSV format
			JSONStripOuterArray: sdk.Bool(true),
			ParquetBinaryAsText: sdk.Bool(true),
		}
		got := fileFormatOptionsToDict(sdk.FileFormatTypeCSV, opts)

		want := map[string]any{
			"SKIP_HEADER":                    int64(1),
			"FIELD_DELIMITER":                ",",
			"PARSE_HEADER":                   true,
			"ERROR_ON_COLUMN_COUNT_MISMATCH": false,
			"COMPRESSION":                    string(sdk.CSVCompressionGzip),
		}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("got %#v, want %#v", got, want)
		}
	})

	t.Run("unset options are omitted, not null", func(t *testing.T) {
		got := fileFormatOptionsToDict(sdk.FileFormatTypeCSV, sdk.FileFormatTypeOptions{})
		if len(got) != 0 {
			t.Errorf("got %#v, want an empty map", got)
		}
	})

	t.Run("null_if flattens to plain strings", func(t *testing.T) {
		opts := sdk.FileFormatTypeOptions{
			JSONNullIf: []sdk.NullString{{S: "NULL"}, {S: "\\N"}},
		}
		got := fileFormatOptionsToDict(sdk.FileFormatTypeJSON, opts)
		want := []any{"NULL", "\\N"}
		if !reflect.DeepEqual(got["NULL_IF"], want) {
			t.Errorf("NULL_IF = %#v, want %#v", got["NULL_IF"], want)
		}
	})

	t.Run("unknown format type yields an empty map", func(t *testing.T) {
		got := fileFormatOptionsToDict(sdk.FileFormatType("SOMETHING_NEW"), sdk.FileFormatTypeOptions{
			CSVSkipHeader: sdk.Int(1),
		})
		if len(got) != 0 {
			t.Errorf("got %#v, want an empty map", got)
		}
	})

	// Every value a dict carries must be JSON-native, or the field errors at
	// query time rather than at compile time. Exercise every format type with a
	// fully populated option struct and assert nothing else slips through.
	t.Run("all emitted values are json native", func(t *testing.T) {
		opts := sdk.FileFormatTypeOptions{
			CSVSkipHeader:       sdk.Int(1),
			CSVFieldDelimiter:   sdk.String(","),
			CSVParseHeader:      sdk.Bool(true),
			CSVNullIf:           &[]sdk.NullString{{S: "NULL"}},
			JSONNullIf:          []sdk.NullString{{S: "NULL"}},
			AvroNullIf:          &[]sdk.NullString{{S: "NULL"}},
			ORCNullIf:           &[]sdk.NullString{{S: "NULL"}},
			ParquetNullIf:       &[]sdk.NullString{{S: "NULL"}},
			XMLPreserveSpace:    sdk.Bool(true),
			JSONEnableOctal:     sdk.Bool(true),
			ParquetBinaryAsText: sdk.Bool(true),
		}
		for _, formatType := range []sdk.FileFormatType{
			sdk.FileFormatTypeCSV, sdk.FileFormatTypeJSON, sdk.FileFormatTypeAvro,
			sdk.FileFormatTypeORC, sdk.FileFormatTypeParquet, sdk.FileFormatTypeXML,
		} {
			for key, value := range fileFormatOptionsToDict(formatType, opts) {
				assertJSONNative(t, string(formatType)+"."+key, value)
			}
		}
	})
}

// assertJSONNative fails when a value is not one a dict can carry.
func assertJSONNative(t *testing.T, path string, value any) {
	t.Helper()
	switch v := value.(type) {
	case string, bool, int64, float64, nil:
	case []any:
		for i := range v {
			assertJSONNative(t, path, v[i])
		}
	default:
		t.Errorf("%s is %T, which a dict cannot carry", path, value)
	}
}
