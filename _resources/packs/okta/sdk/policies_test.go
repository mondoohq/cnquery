// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package sdk

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPoliciesParsing(t *testing.T) {
	data := `
[
    {
      "id": "00p8sl4riqkLEoR7p5d7",
      "settings": {
        "password": {
          "complexity": {
            "minLength": 15,
            "minLowerCase": 1,
            "minUpperCase": 1,
            "minNumber": 1,
            "minSymbol": 1,
            "excludeUsername": true
          },
          "age": {
            "maxAgeDays": 90,
            "expireWarnDays": 15,
            "minAgeMinutes": 60,
            "historyCount": 24
          },
          "lockout": {
            "maxAttempts": 5,
            "autoUnlockMinutes": 30,
            "userLockoutNotificationChannels": [],
            "showLockoutFailures": true
          }
        }
      },
      "type": "PASSWORD"
    }
  ]
`
	var policies []PolicyWrapper
	decoder := json.NewDecoder(strings.NewReader(data))
	err := decoder.Decode(&policies)
	require.NoError(t, err)
	assert.NotNil(t, policies[0].Settings.Password)

	jData, err := json.Marshal(policies)
	require.NoError(t, err)
	assert.NotNil(t, jData)
	assert.Contains(t, string(jData), "\"autoUnlockMinutes\":30,")
}
