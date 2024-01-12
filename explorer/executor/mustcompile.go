// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package executor

import (
	"go.mondoo.com/cnquery/v10"
	"go.mondoo.com/cnquery/v10/llx"
	"go.mondoo.com/cnquery/v10/mqlc"
	"go.mondoo.com/cnquery/v10/providers"
)

func MustCompile(code string) *llx.CodeBundle {
	codeBundle, err := mqlc.Compile(code, nil,
		mqlc.NewConfig(providers.DefaultRuntime().Schema(), cnquery.DefaultFeatures))
	if err != nil {
		panic(err)
	}
	return codeBundle
}

func MustGetOneDatapoint(codeBundle *llx.CodeBundle) string {
	if len(codeBundle.CodeV2.Entrypoints()) != 1 {
		panic("code bundle has more than 1 entrypoint")
	}

	entrypoint := codeBundle.CodeV2.Entrypoints()[0]
	checksum, ok := codeBundle.CodeV2.Checksums[entrypoint]
	if !ok {
		panic("could not find the data point for the entrypoint")
	}

	return checksum
}
