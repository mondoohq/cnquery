package awspolicy

import (
	"encoding/json"
	"fmt"
)

type IamPolicyDocument struct {
	Version   string           `json:"Version,omitempty"`
	Statement policyStatements `json:"Statement,omitempty"`
}

type IamPolicyStatement struct {
	Sid      string           `json:"Sid,omitempty"`
	Effect   string           `json:"Effect,omitempty"`
	Action   statementSection `json:"Action,omitempty"`
	Resource statementSection `json:"Resource,omitempty"`
}

type policyStatements []IamPolicyStatement

type statementSection []string

// can be string or []string
func (v *statementSection) UnmarshalJSON(b []byte) error {
	var raw interface{}
	err := json.Unmarshal(b, &raw)
	if err != nil {
		return err
	}

	var section []string
	switch v := raw.(type) {
	case string:
		section = []string{v}
	case []interface{}:
		for _, item := range v {
			section = append(section, item.(string))
		}
	default:
		return fmt.Errorf("invalid %T value element, policy action and resource only support string or []string", v)
	}
	*v = section
	return nil
}

// can be single object or array
func (v *policyStatements) UnmarshalJSON(b []byte) error {
	var raw interface{}
	err := json.Unmarshal(b, &raw)
	if err != nil {
		return err
	}

	statements := []IamPolicyStatement{}

	switch raw.(type) {
	case []interface{}:
		err = json.Unmarshal(b, &statements)
		if err != nil {
			return err
		}
	case interface{}:
		statement := IamPolicyStatement{}
		err = json.Unmarshal(b, &statement)
		if err != nil {
			return err
		}
		statements = append(statements, statement)
	}
	*v = statements
	return nil
}
