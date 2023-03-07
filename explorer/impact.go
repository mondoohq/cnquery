package explorer

import (
	"encoding/json"

	"github.com/pkg/errors"
	"go.mondoo.com/cnquery/checksums"
)

func (v *Impact) AddBase(base *Impact) {
	if base == nil {
		return
	}

	if v.Scoring == Impact_SCORING_UNSPECIFIED {
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
