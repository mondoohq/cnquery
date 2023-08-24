// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

func (k *mqlGcp) id() (string, error) {
	return "gcp", nil
}

func (s *mqlGcp) field() (string, error) {
	return "example", nil
}
