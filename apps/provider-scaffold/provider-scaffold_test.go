// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package main

import (
	"testing"
)

func TestGenerator(t *testing.T) {
	cfg := config{
		Path:                "../../cnquery/providers/oci",
		ProviderID:          "oci",
		ProviderName:        "Oracle Cloud Infrastructure",
		GoPackage:           "go.mondoo.com/cnquery/v11/providers/oci",
		CamelcaseProviderID: "Oci",
	}

	err := generateProvider(cfg)
	if err != nil {
		t.Fatal(err)
	}
}
