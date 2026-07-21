// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"reflect"
	"testing"

	"github.com/stackitcloud/stackit-sdk-go/services/loadbalancer"
)

func TestListenerSNINames(t *testing.T) {
	if got := listenerSNINames(nil); len(got) != 0 {
		t.Fatalf("nil input: expected empty slice, got %#v", got)
	}

	n1, n2 := "a.example.com", "b.example.com"
	in := []loadbalancer.ServerNameIndicator{{Name: &n1}, {Name: &n2}}
	got := listenerSNINames(in)
	want := []string{"a.example.com", "b.example.com"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v, want %#v", got, want)
	}
}
