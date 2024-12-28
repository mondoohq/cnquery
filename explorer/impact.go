// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package explorer

import (
	"encoding/json"
	"errors"

	"go.mondoo.com/cnquery/v11/checksums"
	"gopkg.in/yaml.v3"
)

// Impact represents severity rating scale when impact is provided as human-readable string value
var impactMapping = map[string]int32{
	"none":     0,
	"low":      10,
	"medium":   40,
	"high":     70,
	"critical": 100,
}

func (v *Impact) HumanReadable() string {
	if v.Value == nil {
		return "unknown"
	}
	switch {
	case v.Value.Value >= 90:
		return "critical"
	case v.Value.Value >= 70:
		return "high"
	case v.Value.Value >= 40:
		return "medium"
	case v.Value.Value >= 10:
		return "low"
	default:
		return "none"
	}
}

func (v *Impact) AddBase(base *Impact) {
	if base == nil {
		return
	}

	if v.Scoring == ScoringSystem_SCORING_UNSPECIFIED {
		v.Scoring = base.Scoring
	}
	if v.Value == nil {
		v.Value = base.Value
	}
	if v.Weight < 1 {
		v.Weight = base.Weight
	}
	if v.Action == Action_UNSPECIFIED {
		v.Action = base.Action
	}
}

func (v *Impact) Checksum() uint64 {
	res := checksums.New
	if v == nil {
		return uint64(res)
	}

	res = res.AddUint(uint64(v.Scoring)).
		AddUint(uint64(v.Weight)).
		AddUint(uint64(v.Action))

	if v.Value != nil {
		res = res.AddUint(uint64(v.Value.Value))
	}

	return uint64(res)
}

// UnmarshalJSON implements the json.Unmarshaler interface for impact value. It supports human-readable string, int and
// complex struct.
func (v *Impact) UnmarshalJSON(data []byte) error {
	var intRes int32
	if err := json.Unmarshal(data, &intRes); err == nil {
		v.Value = &ImpactValue{Value: intRes}

		if v.Value.Value < 0 || v.Value.Value > 100 {
			return errors.New("impact must be between 0 and 100")
		}
		return nil
	}

	var stringRes string
	if err := json.Unmarshal(data, &stringRes); err == nil {
		val, ok := impactMapping[stringRes]
		if !ok {
			return errors.New("impact must use critical, high, medium, low or none")
		}
		v.Value = &ImpactValue{Value: val}
		return nil
	}

	type tmp Impact
	return json.Unmarshal(data, (*tmp)(v))
}

func (v *ImpactValue) MarshalJSON() ([]byte, error) {
	if v == nil {
		return []byte{}, nil
	}
	return json.Marshal(v.Value)
}

func (v *ImpactValue) UnmarshalJSON(data []byte) error {
	var res int32
	if err := json.Unmarshal(data, &res); err == nil {
		v.Value = res
	} else {
		vInternal := &struct {
			Value int32 `json:"value"`
		}{}
		if err := json.Unmarshal(data, &vInternal); err != nil {
			return err
		}
		v.Value = vInternal.Value
	}

	if v.Value < 0 || v.Value > 100 {
		return errors.New("impact must be between 0 and 100")
	}

	return nil
}

func (s *ScoringSystem) UnmarshalJSON(data []byte) error {
	// check if we have a number
	var code int32
	err := json.Unmarshal(data, &code)
	if err == nil {
		*s = ScoringSystem(code)
	} else {
		var name string
		_ = json.Unmarshal(data, &name)

		switch name {
		case "unspecified":
			*s = ScoringSystem_SCORING_UNSPECIFIED
		case "weighted":
			*s = ScoringSystem_WEIGHTED
		case "highest impact":
			*s = ScoringSystem_WORST
		case "average", "":
			*s = ScoringSystem_AVERAGE
		case "data only":
			*s = ScoringSystem_DATA_ONLY
		case "ignore score":
			*s = ScoringSystem_IGNORE_SCORE
		case "banded":
			*s = ScoringSystem_BANDED
		case "decayed":
			*s = ScoringSystem_DECAYED
		case "disabled":
			*s = ScoringSystem_DISABLED
		default:
			return errors.New("unknown scoring system: " + string(data))
		}
	}
	return nil
}

func (s *ScoringSystem) UnmarshalYAML(node *yaml.Node) error {
	// check if we have a number
	var code int32
	err := node.Decode(&code)
	if err == nil {
		*s = ScoringSystem(code)
	} else {
		var name string
		_ = node.Decode(&name)

		switch name {
		case "unspecified":
			*s = ScoringSystem_SCORING_UNSPECIFIED
		case "weighted":
			*s = ScoringSystem_WEIGHTED
		case "highest impact":
			*s = ScoringSystem_WORST
		case "average", "":
			*s = ScoringSystem_AVERAGE
		case "data only":
			*s = ScoringSystem_DATA_ONLY
		case "ignore score":
			*s = ScoringSystem_IGNORE_SCORE
		case "banded":
			*s = ScoringSystem_BANDED
		case "decayed":
			*s = ScoringSystem_DECAYED
		case "disabled":
			*s = ScoringSystem_DISABLED
		default:
			return errors.New("unknown scoring system: " + string(name))
		}
	}
	return nil
}

func (s ScoringSystem) MarshalJSON() ([]byte, error) {
	var result string
	switch s {
	case ScoringSystem_SCORING_UNSPECIFIED:
		result = "unspecified"
	case ScoringSystem_WEIGHTED:
		result = "weighted"
	case ScoringSystem_WORST:
		result = "highest impact"
	case ScoringSystem_AVERAGE:
		result = "average"
	case ScoringSystem_DATA_ONLY:
		result = "data only"
	case ScoringSystem_IGNORE_SCORE:
		result = "ignore score"
	case ScoringSystem_BANDED:
		result = "banded"
	case ScoringSystem_DECAYED:
		result = "decayed"
	case ScoringSystem_DISABLED:
		result = "disabled"
	default:
		result = "unknown"
	}

	return json.Marshal(result) // will add quotes and escape if needed
}

func (s ScoringSystem) MarshalYAML() (interface{}, error) {
	switch s {
	case ScoringSystem_SCORING_UNSPECIFIED:
		return "unspecified", nil
	case ScoringSystem_WEIGHTED:
		return "weighted", nil
	case ScoringSystem_WORST:
		return "highest impact", nil
	case ScoringSystem_AVERAGE:
		return "average", nil
	case ScoringSystem_DATA_ONLY:
		return "data only", nil
	case ScoringSystem_IGNORE_SCORE:
		return "ignore score", nil
	case ScoringSystem_BANDED:
		return "banded", nil
	case ScoringSystem_DECAYED:
		return "decayed", nil
	case ScoringSystem_DISABLED:
		return "disabled", nil
	default:
		return s, nil
	}
}
