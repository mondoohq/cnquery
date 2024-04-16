// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package explorer

import (
	"encoding/json"
	"errors"

	"go.mondoo.com/cnquery/v11/checksums"
	"gopkg.in/yaml.v3"
)

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
	case v.Value.Value > 0:
		return "low"
	default:
		return "info"
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

func (v *Impact) UnmarshalJSON(data []byte) error {
	var res int32
	if err := json.Unmarshal(data, &res); err == nil {
		v.Value = &ImpactValue{Value: res}

		if v.Value.Value < 0 || v.Value.Value > 100 {
			return errors.New("impact must be between 0 and 100")
		}
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
		case "highest impact":
			*s = ScoringSystem_WORST
		case "weighted":
			*s = ScoringSystem_WEIGHTED
		case "average", "":
			*s = ScoringSystem_AVERAGE
		case "banded":
			*s = ScoringSystem_BANDED
		case "decayed":
			*s = ScoringSystem_DECAYED
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
		case "highest impact":
			*s = ScoringSystem_WORST
		case "weighted":
			*s = ScoringSystem_WEIGHTED
		case "average", "":
			*s = ScoringSystem_AVERAGE
		case "banded":
			*s = ScoringSystem_BANDED
		case "decayed":
			*s = ScoringSystem_DECAYED
		default:
			return errors.New("unknown scoring system: " + string(name))
		}
	}
	return nil
}

func (s *ScoringSystem) MarshalYAML() (interface{}, error) {
	switch *s {
	case ScoringSystem_WORST:
		return "highest impact", nil
	case ScoringSystem_WEIGHTED:
		return "weighted", nil
	case ScoringSystem_AVERAGE:
		return "average", nil
	case ScoringSystem_BANDED:
		return "banded", nil
	case ScoringSystem_DECAYED:
		return "decayed", nil
	default:
		return *s, nil
	}
}
