// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"reflect"
	"testing"
)

func TestParseBackupVMIDs(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want []int64
	}{
		{"empty", "", nil},
		{"single", "100", []int64{100}},
		{"comma-separated", "100,101,200", []int64{100, 101, 200}},
		{"whitespace-tolerated", " 100 , 101 , 200 ", []int64{100, 101, 200}},
		{"all-skipped", "all", nil},
		{"non-numeric-skipped", "garbage,100,more-garbage", []int64{100}},
		{"range-skipped", "100-105", nil},
		{"mixed-with-range", "100,200-205,300", []int64{100, 300}},
		{"trailing-comma", "100,", []int64{100}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseBackupVMIDs(tt.in)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseBackupVMIDs(%q) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}
