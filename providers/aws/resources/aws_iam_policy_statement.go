// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
	"go.mondoo.com/mql/v13/providers/aws/resources/awspolicy"
	"go.mondoo.com/mql/v13/types"
)

// newPolicyStatementResources parses an IAM-grammar policy document (an IAM
// policy, role trust policy, or any resource-based policy) into a slice of
// aws.iam.policyStatement resources. The document may be a plain JSON string
// or, as the IAM API returns policy and trust documents, URL-encoded JSON.
// parentID is used to build a stable, synthetic __id for each statement.
func newPolicyStatementResources(runtime *plugin.Runtime, parentID string, policyJSON string) ([]any, error) {
	if strings.TrimSpace(policyJSON) == "" {
		return []any{}, nil
	}

	policy, err := parsePolicyStatements(policyJSON)
	if err != nil {
		return nil, err
	}

	res := make([]any, 0, len(policy.Statements))
	for i := range policy.Statements {
		stmt := policy.Statements[i]

		var conditions any
		if len(stmt.Condition) > 0 {
			if err := json.Unmarshal(stmt.Condition, &conditions); err != nil {
				return nil, err
			}
		}

		mqlStatement, err := CreateResource(runtime, ResourceAwsIamPolicyStatement,
			map[string]*llx.RawData{
				"__id":          llx.StringData(fmt.Sprintf("%s/statement/%d", parentID, i)),
				"sid":           llx.StringData(stmt.Sid),
				"effect":        llx.StringData(stmt.Effect),
				"actions":       llx.ArrayData(convert.SliceAnyToInterface(stmt.Action.Value()), types.String),
				"notActions":    llx.ArrayData(convert.SliceAnyToInterface(stmt.NotAction.Value()), types.String),
				"resources":     llx.ArrayData(convert.SliceAnyToInterface(stmt.Resource.Value()), types.String),
				"notResources":  llx.ArrayData(convert.SliceAnyToInterface(stmt.NotResource.Value()), types.String),
				"principals":    llx.MapData(stringSliceMapToAny(stmt.Principal.Data()), types.Array(types.String)),
				"notPrincipals": llx.MapData(stringSliceMapToAny(stmt.NotPrincipal.Data()), types.Array(types.String)),
				"conditions":    llx.DictData(conditions),
			})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlStatement)
	}
	return res, nil
}

// parsePolicyStatements unmarshals a policy document into the shared
// S3BucketPolicy shape, which preserves the full principal map and the Not*
// elements across every AWS policy grammar. It first tries the raw bytes and
// falls back to URL-decoding, since IAM policy and role trust documents are
// returned URL-encoded while resource-based policies are not.
func parsePolicyStatements(policyJSON string) (*awspolicy.S3BucketPolicy, error) {
	var policy awspolicy.S3BucketPolicy
	if err := json.Unmarshal([]byte(policyJSON), &policy); err != nil {
		decoded, decodeErr := url.QueryUnescape(policyJSON)
		if decodeErr != nil {
			return nil, err
		}
		if err := json.Unmarshal([]byte(decoded), &policy); err != nil {
			return nil, err
		}
	}
	return &policy, nil
}

// policyStatementsFromString parses a resource policy held as a raw JSON
// string field (e.g. an EFS file system policy or VPC endpoint policy).
func policyStatementsFromString(runtime *plugin.Runtime, parentID string, doc *plugin.TValue[string]) ([]any, error) {
	if doc.Error != nil {
		return nil, doc.Error
	}
	return newPolicyStatementResources(runtime, parentID, doc.Data)
}

// policyStatementsFromDict parses a resource policy held as a parsed dict
// field (e.g. an SNS topic or SQS queue policy). The dict is round-tripped
// back to JSON, which is loss-free because these policies are stored as the
// faithful raw document rather than a reduced shape.
func policyStatementsFromDict(runtime *plugin.Runtime, parentID string, doc *plugin.TValue[any]) ([]any, error) {
	if doc.Error != nil {
		return nil, doc.Error
	}
	if doc.Data == nil {
		return []any{}, nil
	}
	raw, err := json.Marshal(doc.Data)
	if err != nil {
		return nil, err
	}
	return newPolicyStatementResources(runtime, parentID, string(raw))
}

func (a *mqlAwsIamPolicy) statements() ([]any, error) {
	defaultVersion, err := a.defaultVersion()
	if err != nil {
		return nil, err
	}
	return defaultVersion.statements()
}

func (a *mqlAwsIamPolicyversion) statements() ([]any, error) {
	rawDoc, err := a.rawDocument()
	if err != nil {
		return nil, err
	}
	parentID := a.Arn.Data + "/" + a.VersionId.Data
	return newPolicyStatementResources(a.MqlRuntime, parentID, rawDoc)
}

func (a *mqlAwsIamRole) assumeRolePolicyStatements() ([]any, error) {
	doc := a.GetAssumeRolePolicyDocument()
	if doc.Error != nil {
		return nil, doc.Error
	}
	if doc.Data == nil {
		return []any{}, nil
	}
	raw, err := json.Marshal(doc.Data)
	if err != nil {
		return nil, err
	}
	return newPolicyStatementResources(a.MqlRuntime, a.Arn.Data+"/trust", string(raw))
}

func (a *mqlAwsKmsKey) policyStatements() ([]any, error) {
	policy := a.GetPolicy()
	if policy.Error != nil {
		return nil, policy.Error
	}
	return newPolicyStatementResources(a.MqlRuntime, a.Arn.Data, policy.Data)
}

func (a *mqlAwsVpcEndpoint) policyStatements() ([]any, error) {
	return policyStatementsFromString(a.MqlRuntime, a.Id.Data, a.GetPolicyDocument())
}

func (a *mqlAwsEfsFilesystem) policyStatements() ([]any, error) {
	return policyStatementsFromString(a.MqlRuntime, a.Arn.Data, a.GetFileSystemPolicy())
}

func (a *mqlAwsSnsTopic) policyStatements() ([]any, error) {
	return policyStatementsFromDict(a.MqlRuntime, a.Arn.Data, a.GetPolicy())
}

func (a *mqlAwsSqsQueue) policyStatements() ([]any, error) {
	return policyStatementsFromDict(a.MqlRuntime, a.Arn.Data, a.GetPolicy())
}

func (a *mqlAwsEcrRepository) policyStatements() ([]any, error) {
	return policyStatementsFromDict(a.MqlRuntime, a.Arn.Data, a.GetPolicy())
}

func (a *mqlAwsLambdaFunction) policyStatements() ([]any, error) {
	return policyStatementsFromDict(a.MqlRuntime, a.Arn.Data, a.GetPolicy())
}

func (a *mqlAwsSecretsmanagerSecret) policyStatements() ([]any, error) {
	return policyStatementsFromString(a.MqlRuntime, a.Arn.Data, a.GetResourcePolicy())
}

func (a *mqlAwsS3Bucket) policyStatements() ([]any, error) {
	policy := a.GetPolicy()
	if policy.Error != nil {
		return nil, policy.Error
	}
	if policy.Data == nil {
		return []any{}, nil
	}
	return policyStatementsFromString(a.MqlRuntime, a.Name.Data, policy.Data.GetDocument())
}
