// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestShouldCheckEarlierDateForReport(t *testing.T) {
	err := errors.New("some random error message")
	require.False(t, shouldCheckEarlierDateForReport(err))

	err = errors.New("Error 400: Another err (bad request)")
	require.False(t, shouldCheckEarlierDateForReport(err))

	err = errors.New("Error 400: Start date can not be later than 2024-07-29, invalid")
	require.True(t, shouldCheckEarlierDateForReport(err))

	err = errors.New("Error 400: Data for dates later than 2024-07-26 is not yet available. Please check back later, invalid")
	require.True(t, shouldCheckEarlierDateForReport(err))
}
