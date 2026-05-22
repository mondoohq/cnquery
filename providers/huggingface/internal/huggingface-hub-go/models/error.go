// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package models

type Error struct {
	Error string `json:"error"`
	Code  int    `json:"code"`
}
