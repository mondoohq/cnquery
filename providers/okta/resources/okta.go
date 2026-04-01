// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

func (k *mqlOkta) id() (string, error) {
	return "okta", nil
}

const queryLimit = 200
