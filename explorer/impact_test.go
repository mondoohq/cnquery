// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package explorer

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestImpactParsing(t *testing.T) {
	tests := []struct {
		title string
		data  string
		res   *Impact
	}{
		{
			"simple number",
			"30",
			&Impact{
				Value: &ImpactValue{Value: 30},
			},
		},
		{
			"complex definition",
			`{"weight": 20, "value": 40}`,
			&Impact{
				Weight: 20,
				Value:  &ImpactValue{Value: 40},
			},
		},
	}

	for i := range tests {
		cur := tests[i]
		t.Run(cur.title, func(t *testing.T) {
			var res *Impact
			err := json.Unmarshal([]byte(cur.data), &res)
			require.NoError(t, err)
			assert.Equal(t, cur.res, res)
		})
	}

	errTests := []struct {
		title string
		data  string
		err   string
	}{
		{
			"invalid low impact",
			"-1",
			"impact must be between 0 and 100",
		},
		{
			"invalid high impact",
			"101",
			"impact must be between 0 and 100",
		},
		{
			"invalid low impact in complex struct",
			`{"value": -1}`,
			"impact must be between 0 and 100",
		},
		{
			"invalid high impact in complex struct",
			`{"value": 101, "weight": 90}`,
			"impact must be between 0 and 100",
		},
	}

	for i := range errTests {
		cur := errTests[i]
		t.Run(cur.title, func(t *testing.T) {
			var res *Impact
			err := json.Unmarshal([]byte(cur.data), &res)
			assert.EqualError(t, err, cur.err)
		})
	}
}

func TestImpactMerging(t *testing.T) {
	tests := []struct {
		title string
		base  *Impact
		main  *Impact
		res   *Impact
	}{
		{
			"nil base",
			nil,
			&Impact{Value: &ImpactValue{Value: 20}},
			&Impact{Value: &ImpactValue{Value: 20}},
		},
		{
			"empty base",
			&Impact{},
			&Impact{Value: &ImpactValue{Value: 20}},
			&Impact{Value: &ImpactValue{Value: 20}},
		},
		{
			"empty main",
			&Impact{Value: &ImpactValue{Value: 20}},
			&Impact{},
			&Impact{Value: &ImpactValue{Value: 20}},
		},
		{
			"inherit value",
			&Impact{Value: &ImpactValue{Value: 20}},
			&Impact{Weight: 10, Scoring: ScoringSystem_AVERAGE},
			&Impact{
				Value:  &ImpactValue{Value: 20},
				Weight: 10, Scoring: ScoringSystem_AVERAGE,
			},
		},
		{
			"inherit scoring (explicit)",
			&Impact{Scoring: ScoringSystem_IGNORE_SCORE},
			&Impact{Scoring: ScoringSystem_SCORING_UNSPECIFIED, Value: &ImpactValue{Value: 78}},
			&Impact{Scoring: ScoringSystem_IGNORE_SCORE, Value: &ImpactValue{Value: 78}},
		},
	}

	for i := range tests {
		cur := tests[i]
		t.Run(cur.title, func(t *testing.T) {
			cur.main.AddBase(cur.base)
			assert.Equal(t, cur.res, cur.main)
		})
	}
}

func TestScoringSystemParsingJSON(t *testing.T) {
	s := ScoringSystem_DECAYED
	raw, err := json.Marshal(s)
	require.NoError(t, err)

	assert.Equal(t, `"decayed"`, string(raw))

	err = json.Unmarshal(raw, &s)
	require.NoError(t, err)
	assert.Equal(t, ScoringSystem_DECAYED, s)
}

func TestScoringSystemParsingYAML(t *testing.T) {
	s := ScoringSystem_DECAYED
	raw, err := yaml.Marshal(s)
	require.NoError(t, err)

	assert.Equal(t, `decayed`, strings.Trim(string(raw), "\n"))

	err = yaml.Unmarshal(raw, &s)
	require.NoError(t, err)
	assert.Equal(t, ScoringSystem_DECAYED, s)
}

func TestScoringSystemPointerParsingJSON(t *testing.T) {
	ss := ScoringSystem_DECAYED
	data := struct {
		ScoringSystem *ScoringSystem `json:"scoring_system"`
	}{
		&ss,
	}

	raw, err := json.Marshal(data)
	require.NoError(t, err)

	assert.JSONEq(t, `{"scoring_system":"decayed"}`, string(raw))

	err = json.Unmarshal(raw, &data)
	require.NoError(t, err)
	assert.Equal(t, ScoringSystem_DECAYED, *data.ScoringSystem)
}

func TestScoringSystemPointerParsingYAML(t *testing.T) {
	ss := ScoringSystem_DECAYED
	data := struct {
		ScoringSystem *ScoringSystem `yaml:"scoring_system"`
	}{
		&ss,
	}

	raw, err := yaml.Marshal(data)
	require.NoError(t, err)

	assert.Equal(t, `scoring_system: decayed`, strings.Trim(string(raw), "\n"))

	err = yaml.Unmarshal(raw, &data)
	require.NoError(t, err)
	assert.Equal(t, ScoringSystem_DECAYED, *data.ScoringSystem)
}
