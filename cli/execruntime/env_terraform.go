// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package execruntime

const TERRAFORM = "terraform"

var terraformEnv = &RuntimeEnv{
	Id:        TERRAFORM,
	Name:      "Terraform",
	Namespace: "terraform.io",
	Prefix:    "TERRAFORM",
	Identify: []Variable{
		{
			Name: "TERRAFORM_PIPELINE",
		},
	},
	Variables: []Variable{},
}
