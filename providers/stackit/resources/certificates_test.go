// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"reflect"
	"testing"

	"github.com/stackitcloud/stackit-sdk-go/services/certificates"
)

func TestCertificateUsage(t *testing.T) {
	strPtr := func(s string) *string { return &s }

	t.Run("empty usage yields empty slice", func(t *testing.T) {
		got := certificateUsage(certificates.Usage{})
		if got == nil || len(got) != 0 {
			t.Fatalf("expected non-nil empty slice, got %#v", got)
		}
	})

	t.Run("maps load balancers and listeners", func(t *testing.T) {
		listenersA := []string{"https", "http"}
		listenersB := []string{"tls"}
		u := certificates.Usage{
			Items: &[]certificates.UsageItem{
				{LoadBalancerName: strPtr("lb-a"), ListenerNames: &listenersA},
				{LoadBalancerName: strPtr("lb-b"), ListenerNames: &listenersB},
			},
		}

		got := certificateUsage(u)
		want := []any{
			map[string]any{"loadBalancerName": "lb-a", "listenerNames": []any{"https", "http"}},
			map[string]any{"loadBalancerName": "lb-b", "listenerNames": []any{"tls"}},
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("mismatch\n got: %#v\nwant: %#v", got, want)
		}
	})

	t.Run("item without listeners yields empty listener slice", func(t *testing.T) {
		u := certificates.Usage{
			Items: &[]certificates.UsageItem{
				{LoadBalancerName: strPtr("lb-c")},
			},
		}

		got := certificateUsage(u)
		entry, ok := got[0].(map[string]any)
		if !ok {
			t.Fatalf("expected map entry, got %T", got[0])
		}
		listeners, ok := entry["listenerNames"].([]any)
		if !ok {
			t.Fatalf("expected []any listeners, got %T", entry["listenerNames"])
		}
		if len(listeners) != 0 {
			t.Fatalf("expected empty listeners, got %#v", listeners)
		}
	})
}
