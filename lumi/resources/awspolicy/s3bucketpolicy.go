package awspolicy

import (
	"encoding/json"
	"errors"
	"fmt"
)

// Threre are two policy versions available:
// - `2012-10-17` this is the current version of the policy language
// - `2008-10-17` an earlier version of the policy language
//
// see https://docs.aws.amazon.com/IAM/latest/UserGuide/reference_policies_elements_version.html
//
// policy examples are available here:
// see https://aws.amazon.com/blogs/security/back-to-school-understanding-the-iam-policy-grammar/

type S3BucketPolicy struct {
	Version    string                    `json:"Version"`
	Id         string                    `json:"Id,omitempty"`
	Statements []S3BucketPolicyStatement `json:"Statement"`
}

// the policy statement includes many different aspects including the Not* elements, they are used to exlclude
// things from the previous inlcude, see https://docs.aws.amazon.com/IAM/latest/UserGuide/reference_policies_examples_s3_deny-except-bucket.html
type S3BucketPolicyStatement struct {
	Sid          string                       `json:"Sid,omitempty"`          // statement ID, optional
	Effect       string                       `json:"Effect"`                 // `Allow` or `Deny`
	Principal    interface{}                  `json:"Principal,omitempty"`    // principal that is allowed or denied
	NotPrincipal interface{}                  `json:"NotPrincipal,omitempty"` // excluded principal
	Action       S3BucketPolicyStatementValue `json:"Action"`                 // allowed or denied action
	NotAction    S3BucketPolicyStatementValue `json:"NotAction,omitempty"`    // excluded action
	Resource     S3BucketPolicyStatementValue `json:"Resource,omitempty"`     // object or objects that the statement covers
	NotResource  S3BucketPolicyStatementValue `json:"NotResource,omitempty"`  // excluded resources
	Condition    json.RawMessage              `json:"Condition,omitempty"`    // conditions for when a policy is in effect
}

// AWS allows string or []string as value, we convert everything to []string to avoid casting
type S3BucketPolicyStatementValue []string

func (v *S3BucketPolicyStatementValue) Value() []string {
	return []string(*v)
}

//  value can be string or []string, convert everything to []string
func (v *S3BucketPolicyStatementValue) UnmarshalJSON(b []byte) error {
	var raw interface{}
	err := json.Unmarshal(b, &raw)
	if err != nil {
		return err
	}

	var p []string
	switch v := raw.(type) {
	case string:
		p = []string{v}
	case []interface{}:
		var items []string
		for _, item := range v {
			items = append(items, fmt.Sprintf("%v", item))
		}
		p = items
	default:
		return errors.New("invalid %s value element, s3 bucket policy only supports string or []string")
	}
	*v = p
	return nil
}
