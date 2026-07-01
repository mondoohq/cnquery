// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws/arn"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
)

// managedByFromResourceTags returns the infrastructure-management system that
// owns a resource, inferred from the provenance tags AWS injects on managed
// resources. It propagates any error from resolving computed tags.
func managedByFromResourceTags(tags *plugin.TValue[map[string]any]) (string, error) {
	if tags.Error != nil {
		return "", tags.Error
	}
	return managedByFromTags(tags.Data), nil
}

// cloudformationStackFromResourceTags resolves the CloudFormation stack that
// provisioned a resource from its AWS-injected stack-name tag, setting the
// field to null when the resource carries no such tag.
func cloudformationStackFromResourceTags(runtime *plugin.Runtime, region string, tags *plugin.TValue[map[string]any], field *plugin.TValue[*mqlAwsCloudformationStack]) (*mqlAwsCloudformationStack, error) {
	if tags.Error != nil {
		return nil, tags.Error
	}
	stack, err := cloudformationStackForTags(runtime, region, tags.Data)
	if err != nil {
		return nil, err
	}
	if stack == nil {
		field.State = plugin.StateIsNull | plugin.StateIsSet
		return nil, nil
	}
	return stack, nil
}

func (a *mqlAwsRdsDbinstance) managedBy() (string, error) {
	return managedByFromResourceTags(a.GetTags())
}

func (a *mqlAwsRdsDbinstance) cloudformationStack() (*mqlAwsCloudformationStack, error) {
	return cloudformationStackFromResourceTags(a.MqlRuntime, a.Region.Data, a.GetTags(), &a.CloudformationStack)
}

func (a *mqlAwsRdsDbcluster) managedBy() (string, error) {
	return managedByFromResourceTags(a.GetTags())
}

func (a *mqlAwsRdsDbcluster) cloudformationStack() (*mqlAwsCloudformationStack, error) {
	return cloudformationStackFromResourceTags(a.MqlRuntime, a.Region.Data, a.GetTags(), &a.CloudformationStack)
}

func (a *mqlAwsEc2Volume) managedBy() (string, error) {
	return managedByFromResourceTags(a.GetTags())
}

func (a *mqlAwsEc2Volume) cloudformationStack() (*mqlAwsCloudformationStack, error) {
	return cloudformationStackFromResourceTags(a.MqlRuntime, a.Region.Data, a.GetTags(), &a.CloudformationStack)
}

func (a *mqlAwsEc2Snapshot) managedBy() (string, error) {
	return managedByFromResourceTags(a.GetTags())
}

func (a *mqlAwsEc2Snapshot) cloudformationStack() (*mqlAwsCloudformationStack, error) {
	return cloudformationStackFromResourceTags(a.MqlRuntime, a.Region.Data, a.GetTags(), &a.CloudformationStack)
}

func (a *mqlAwsEfsFilesystem) managedBy() (string, error) {
	owner, err := managedByFromResourceTags(a.GetTags())
	if err != nil {
		return "", err
	}
	if owner != "" {
		return owner, nil
	}
	// Terraform injects no provenance tag, but when creation_token is left
	// unset the Terraform AWS provider auto-generates one with a "terraform-"
	// prefix. Fall back to that heuristic only when no tag-based owner matched.
	ct := a.GetCreationToken()
	if ct.Error != nil {
		return "", ct.Error
	}
	if strings.HasPrefix(ct.Data, "terraform-") {
		return "terraform", nil
	}
	return "", nil
}

func (a *mqlAwsEfsFilesystem) cloudformationStack() (*mqlAwsCloudformationStack, error) {
	return cloudformationStackFromResourceTags(a.MqlRuntime, a.Region.Data, a.GetTags(), &a.CloudformationStack)
}

func (a *mqlAwsEcsInstance) managedBy() (string, error) {
	return managedByFromResourceTags(a.GetTags())
}

func (a *mqlAwsEcsInstance) cloudformationStack() (*mqlAwsCloudformationStack, error) {
	return cloudformationStackFromResourceTags(a.MqlRuntime, a.Region.Data, a.GetTags(), &a.CloudformationStack)
}

func (a *mqlAwsApigatewayRestapi) managedBy() (string, error) {
	return managedByFromResourceTags(a.GetTags())
}

func (a *mqlAwsApigatewayRestapi) cloudformationStack() (*mqlAwsCloudformationStack, error) {
	return cloudformationStackFromResourceTags(a.MqlRuntime, a.Region.Data, a.GetTags(), &a.CloudformationStack)
}

func (a *mqlAwsEsDomain) managedBy() (string, error) {
	return managedByFromResourceTags(a.GetTags())
}

func (a *mqlAwsEsDomain) cloudformationStack() (*mqlAwsCloudformationStack, error) {
	return cloudformationStackFromResourceTags(a.MqlRuntime, a.Region.Data, a.GetTags(), &a.CloudformationStack)
}

func (a *mqlAwsMskCluster) managedBy() (string, error) {
	return managedByFromResourceTags(a.GetTags())
}

func (a *mqlAwsMskCluster) cloudformationStack() (*mqlAwsCloudformationStack, error) {
	return cloudformationStackFromResourceTags(a.MqlRuntime, a.Region.Data, a.GetTags(), &a.CloudformationStack)
}

func (a *mqlAwsCloudtrailTrail) managedBy() (string, error) {
	return managedByFromResourceTags(a.GetTags())
}

func (a *mqlAwsCloudtrailTrail) cloudformationStack() (*mqlAwsCloudformationStack, error) {
	return cloudformationStackFromResourceTags(a.MqlRuntime, a.Region.Data, a.GetTags(), &a.CloudformationStack)
}

func (a *mqlAwsDynamodbTable) managedBy() (string, error) {
	return managedByFromResourceTags(a.GetTags())
}

func (a *mqlAwsDynamodbTable) cloudformationStack() (*mqlAwsCloudformationStack, error) {
	return cloudformationStackFromResourceTags(a.MqlRuntime, a.Region.Data, a.GetTags(), &a.CloudformationStack)
}

func (a *mqlAwsCloudwatchLoggroup) managedBy() (string, error) {
	return managedByFromResourceTags(a.GetTags())
}

func (a *mqlAwsCloudwatchLoggroup) cloudformationStack() (*mqlAwsCloudformationStack, error) {
	return cloudformationStackFromResourceTags(a.MqlRuntime, a.Region.Data, a.GetTags(), &a.CloudformationStack)
}

func (a *mqlAwsLambdaFunction) managedBy() (string, error) {
	return managedByFromResourceTags(a.GetTags())
}

func (a *mqlAwsLambdaFunction) cloudformationStack() (*mqlAwsCloudformationStack, error) {
	return cloudformationStackFromResourceTags(a.MqlRuntime, a.Region.Data, a.GetTags(), &a.CloudformationStack)
}

func (a *mqlAwsElbLoadbalancer) managedBy() (string, error) {
	return managedByFromResourceTags(a.GetTags())
}

func (a *mqlAwsElbLoadbalancer) cloudformationStack() (*mqlAwsCloudformationStack, error) {
	return cloudformationStackFromResourceTags(a.MqlRuntime, a.Region.Data, a.GetTags(), &a.CloudformationStack)
}

func (a *mqlAwsKmsKey) managedBy() (string, error) {
	return managedByFromResourceTags(a.GetTags())
}

func (a *mqlAwsKmsKey) cloudformationStack() (*mqlAwsCloudformationStack, error) {
	return cloudformationStackFromResourceTags(a.MqlRuntime, a.Region.Data, a.GetTags(), &a.CloudformationStack)
}

func (a *mqlAwsSagemakerNotebookinstance) managedBy() (string, error) {
	return managedByFromResourceTags(a.GetTags())
}

func (a *mqlAwsSagemakerNotebookinstance) cloudformationStack() (*mqlAwsCloudformationStack, error) {
	return cloudformationStackFromResourceTags(a.MqlRuntime, a.Region.Data, a.GetTags(), &a.CloudformationStack)
}

func (a *mqlAwsSagemakerProcessingjob) managedBy() (string, error) {
	return managedByFromResourceTags(a.GetTags())
}

func (a *mqlAwsSagemakerProcessingjob) cloudformationStack() (*mqlAwsCloudformationStack, error) {
	return cloudformationStackFromResourceTags(a.MqlRuntime, a.Region.Data, a.GetTags(), &a.CloudformationStack)
}

func (a *mqlAwsSagemakerTrainingjob) managedBy() (string, error) {
	return managedByFromResourceTags(a.GetTags())
}

func (a *mqlAwsSagemakerTrainingjob) cloudformationStack() (*mqlAwsCloudformationStack, error) {
	return cloudformationStackFromResourceTags(a.MqlRuntime, a.Region.Data, a.GetTags(), &a.CloudformationStack)
}

func (a *mqlAwsSsmInstance) managedBy() (string, error) {
	return managedByFromResourceTags(a.GetTags())
}

func (a *mqlAwsSsmInstance) cloudformationStack() (*mqlAwsCloudformationStack, error) {
	return cloudformationStackFromResourceTags(a.MqlRuntime, a.Region.Data, a.GetTags(), &a.CloudformationStack)
}

func (a *mqlAwsEcrRepository) managedBy() (string, error) {
	return managedByFromResourceTags(a.GetTags())
}

func (a *mqlAwsEcrRepository) cloudformationStack() (*mqlAwsCloudformationStack, error) {
	return cloudformationStackFromResourceTags(a.MqlRuntime, a.Region.Data, a.GetTags(), &a.CloudformationStack)
}

func (a *mqlAwsMqBroker) managedBy() (string, error) {
	return managedByFromResourceTags(a.GetTags())
}

func (a *mqlAwsMqBroker) cloudformationStack() (*mqlAwsCloudformationStack, error) {
	return cloudformationStackFromResourceTags(a.MqlRuntime, a.Region.Data, a.GetTags(), &a.CloudformationStack)
}

func (a *mqlAwsEksCluster) managedBy() (string, error) {
	return managedByFromResourceTags(a.GetTags())
}

func (a *mqlAwsEksCluster) cloudformationStack() (*mqlAwsCloudformationStack, error) {
	return cloudformationStackFromResourceTags(a.MqlRuntime, a.Region.Data, a.GetTags(), &a.CloudformationStack)
}

func (a *mqlAwsElasticacheCluster) managedBy() (string, error) {
	return managedByFromResourceTags(a.GetTags())
}

func (a *mqlAwsElasticacheCluster) cloudformationStack() (*mqlAwsCloudformationStack, error) {
	return cloudformationStackFromResourceTags(a.MqlRuntime, a.Region.Data, a.GetTags(), &a.CloudformationStack)
}

func (a *mqlAwsDocumentdbCluster) managedBy() (string, error) {
	return managedByFromResourceTags(a.GetTags())
}

func (a *mqlAwsDocumentdbCluster) cloudformationStack() (*mqlAwsCloudformationStack, error) {
	return cloudformationStackFromResourceTags(a.MqlRuntime, a.Region.Data, a.GetTags(), &a.CloudformationStack)
}

func (a *mqlAwsIamUser) managedBy() (string, error) {
	return managedByFromResourceTags(a.GetTags())
}

func (a *mqlAwsSecretsmanagerSecret) managedBy() (string, error) {
	return managedByFromResourceTags(a.GetTags())
}

func (a *mqlAwsSecretsmanagerSecret) cloudformationStack() (*mqlAwsCloudformationStack, error) {
	return cloudformationStackFromResourceTags(a.MqlRuntime, a.GetPrimaryRegion().Data, a.GetTags(), &a.CloudformationStack)
}

func (a *mqlAwsEmrCluster) managedBy() (string, error) {
	return managedByFromResourceTags(a.GetTags())
}

func (a *mqlAwsEmrCluster) cloudformationStack() (*mqlAwsCloudformationStack, error) {
	region := ""
	if parsed, err := arn.Parse(a.Arn.Data); err == nil {
		region = parsed.Region
	}
	return cloudformationStackFromResourceTags(a.MqlRuntime, region, a.GetTags(), &a.CloudformationStack)
}
