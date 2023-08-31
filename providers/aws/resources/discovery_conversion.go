// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	ssmtypes "github.com/aws/aws-sdk-go-v2/service/ssm/types"
	"github.com/cockroachdb/errors"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/providers-sdk/v1/vault"
	"go.mondoo.com/cnquery/providers/aws/connection"
	"go.mondoo.com/cnquery/providers/os/id/awsec2"
	"go.mondoo.com/cnquery/providers/os/id/containerid"
)

type mqlObject struct {
	name      string
	labels    map[string]string
	awsObject awsObject
}

type awsObject struct {
	account    string
	region     string
	id         string
	service    string
	objectType string
	arn        string
}

func MondooObjectID(awsObject awsObject) string {
	return "//platformid.api.mondoo.app/runtime/aws/" + awsObject.service + "/v1/accounts/" + awsObject.account + "/regions/" + awsObject.region + "/" + awsObject.objectType + "/" + awsObject.id
}

func MqlObjectToAsset(account string, mqlObject mqlObject, conn *connection.AwsConnection) *inventory.Asset {
	if name := mqlObject.labels["Name"]; name != "" {
		mqlObject.name = name
	}
	if mqlObject.name == "" {
		mqlObject.name = mqlObject.awsObject.id
	}
	if err := validate(mqlObject); err != nil {
		log.Error().Err(err).Msg("missing values in mql object to asset translation")
		return nil
	}
	info, err := getTitleFamily(mqlObject.awsObject)
	if err != nil {
		log.Error().Err(err).Msg("missing runtime info")
		return nil
	}
	platformid := MondooObjectID(mqlObject.awsObject)
	t := conn.Conf
	t.PlatformId = platformid
	return &inventory.Asset{
		PlatformIds: []string{platformid, mqlObject.awsObject.arn},
		Name:        mqlObject.name,
		Platform: &inventory.Platform{
			Name:    info.Name,
			Title:   info.Title,
			Kind:    "aws-object",
			Runtime: "AWS",
		},
		Labels:      mqlObject.labels,
		Connections: []*inventory.Config{conn.Conf},
	}
}

func validate(m mqlObject) error {
	if m.name == "" {
		return errors.New("name required for mql aws object to asset translation")
	}
	if m.awsObject.id == "" {
		return errors.New("id required for mql aws object to asset translation")
	}
	if m.awsObject.region == "" {
		return errors.New("region required for mql aws object to asset translation")
	}
	if m.awsObject.account == "" {
		return errors.New("account required for mql aws object to asset translation")
	}
	if m.awsObject.arn == "" {
		return errors.New("arn required for mql aws object to asset translation")
	}
	return nil
}

func getTitleFamily(awsObject awsObject) (*inventory.Platform, error) {
	switch awsObject.service {
	case "s3":
		if awsObject.objectType == "bucket" {
			return &inventory.Platform{Title: "AWS S3 Bucket", Name: "aws-s3-bucket"}, nil
		}
	case "cloudtrail":
		if awsObject.objectType == "trail" {
			return &inventory.Platform{Title: "AWS CloudTrail Trail", Name: "aws-cloudtrail-trail"}, nil
		}
	case "rds":
		if awsObject.objectType == "dbinstance" {
			return &inventory.Platform{Title: "AWS RDS DB Instance", Name: "aws-rds-dbinstance"}, nil
		}
	case "dynamodb":
		if awsObject.objectType == "table" {
			return &inventory.Platform{Title: "AWS DynamoDB Table", Name: "aws-dynamodb-table"}, nil
		}
	case "redshift":
		if awsObject.objectType == "cluster" {
			return &inventory.Platform{Title: "AWS Redshift Cluster", Name: "aws-redshift-cluster"}, nil
		}
	case "vpc":
		if awsObject.objectType == "vpc" {
			return &inventory.Platform{Title: "AWS VPC", Name: "aws-vpc"}, nil
		}
	case "ec2":
		switch awsObject.objectType {
		case "securitygroup":
			return &inventory.Platform{Title: "AWS Security Group", Name: "aws-security-group"}, nil
		case "volume":
			return &inventory.Platform{Title: "AWS EC2 Volume", Name: "aws-ec2-volume"}, nil
		case "snapshot":
			return &inventory.Platform{Title: "AWS EC2 Snapshot", Name: "aws-ec2-snapshot"}, nil
		case "instance":
			return &inventory.Platform{Title: "AWS EC2 Instance", Name: "aws-ec2-instance"}, nil
		}
	case "iam":
		switch awsObject.objectType {
		case "user":
			return &inventory.Platform{Title: "AWS IAM User", Name: "aws-iam-user"}, nil

		case "group":
			return &inventory.Platform{Title: "AWS IAM Group", Name: "aws-iam-group"}, nil
		}
	case "cloudwatch":
		if awsObject.objectType == "loggroup" {
			return &inventory.Platform{Title: "AWS CloudWatch Log Group", Name: "aws-cloudwatch-loggroup"}, nil
		}
	case "lambda":
		if awsObject.objectType == "function" {
			return &inventory.Platform{Title: "AWS Lambda Function", Name: "aws-lambda-function"}, nil
		}
	case "ecs":
		if awsObject.objectType == "container" {
			return &inventory.Platform{Title: "AWS ECS Container", Name: "aws-ecs-container"}, nil
		}
		if awsObject.objectType == "instance" {
			return &inventory.Platform{Title: "AWS ECS Container Instance", Name: "aws-ecs-instance"}, nil
		}
	case "efs":
		if awsObject.objectType == "filesystem" {
			return &inventory.Platform{Title: "AWS EFS Filesystem", Name: "aws-efs-filesystem"}, nil
		}
	case "gateway":
		if awsObject.objectType == "restapi" {
			return &inventory.Platform{Title: "AWS Gateway REST API", Name: "aws-gateway-restapi"}, nil
		}
	case "elb":
		if awsObject.objectType == "loadbalancer" {
			return &inventory.Platform{Title: "AWS ELB Load Balancer", Name: "aws-elb-loadbalancer"}, nil
		}
	case "es":
		if awsObject.objectType == "domain" {
			return &inventory.Platform{Title: "AWS ES Domain", Name: "aws-es-domain"}, nil
		}
	case "kms":
		if awsObject.objectType == "key" {
			return &inventory.Platform{Title: "AWS KMS Key", Name: "aws-kms-key"}, nil
		}
	case "sagemaker":
		if awsObject.objectType == "notebookinstance" {
			return &inventory.Platform{Title: "AWS SageMaker Notebook Instance", Name: "aws-sagemaker-notebookinstance"}, nil
		}
	case "ssm":
		if awsObject.objectType == "instance" {
			return &inventory.Platform{Title: "AWS SSM Instance", Name: "aws-ssm-instance"}, nil
		}
	case "ecr":
		if awsObject.objectType == "image" {
			return &inventory.Platform{Title: "AWS ECR Image", Name: "aws-ecr-image"}, nil
		}
	}
	return nil, errors.Newf("missing runtime info for aws object service %s type %s", awsObject.service, awsObject.objectType)
}

func accountAsset(conn *connection.AwsConnection, awsAccount *mqlAwsAccount) *inventory.Asset {
	var alias string
	aliases := awsAccount.GetAliases()
	if len(aliases.Data) > 0 {
		alias = aliases.Data[0].(string)
	}
	name := AssembleIntegrationName(alias, awsAccount.Id.Data)

	id := "//platformid.api.mondoo.app/runtime/aws/accounts/" + awsAccount.Id.Data

	return &inventory.Asset{
		PlatformIds: []string{id},
		Name:        name,
		Platform:    &inventory.Platform{Name: "aws", Runtime: "aws"},
		Connections: []*inventory.Config{conn.Conf},
	}
}

func AssembleIntegrationName(alias string, id string) string {
	justId := strings.TrimPrefix(id, "aws.account/")
	if alias == "" {
		return fmt.Sprintf("AWS Account %s", justId)
	}
	return fmt.Sprintf("AWS Account %s (%s)", alias, justId)
}

func addConnectionInfoToEc2Asset(instance *mqlAwsEc2Instance, accountId string) *inventory.Asset {
	asset := &inventory.Asset{}
	asset.PlatformIds = []string{awsec2.MondooInstanceID(accountId, instance.Region.Data, instance.InstanceId.Data)}
	asset.IdDetector = []string{"aws-ec2"}
	asset.Platform = &inventory.Platform{
		Kind:    "virtual_machine",
		Runtime: "aws_ec2",
	}
	asset.State = mapEc2InstanceStateCode(instance.State.Data)
	asset.Labels = mapStringInterfaceToStringString(instance.Tags.Data)
	asset.Name = instance.InstanceId.Data
	if name := asset.Labels["Name"]; name != "" {
		asset.Name = name
	}
	// if there is a public ip, we assume ssh is an option
	if instance.PublicIp.Data != "" {
		imageName := ""
		if instance.GetImage().Data != nil {
			imageName = instance.GetImage().Data.Name.Data
		}
		asset.Connections = []*inventory.Config{{
			Backend: "ssh",
			Host:    instance.PublicIp.Data,
			// Insecure: ,
			Runtime: "aws_ec2",
			Credentials: []*vault.Credential{
				{
					Type: vault.CredentialType_aws_ec2_instance_connect,
					User: getProbableUsernameFromImageName(imageName),
				},
			},
			Options: map[string]string{
				"region": instance.Region.Data,
				// "profile":  ec2i.profile,
				"instance": instance.InstanceId.Data,
			},
		}}
	} else {
		log.Warn().Str("asset", asset.Name).Msg("no public ip address found")
	}
	return asset
}

func addSSMConnectionInfoToEc2Asset(instance *mqlAwsEc2Instance, accountId string, profile string) *inventory.Asset {
	asset := &inventory.Asset{}
	asset.PlatformIds = []string{awsec2.MondooInstanceID(accountId, instance.Region.Data, instance.InstanceId.Data)}
	asset.IdDetector = []string{"aws-ec2"}
	asset.Platform = &inventory.Platform{
		Kind:    "virtual_machine",
		Runtime: "aws_ec2",
	}
	ssm := ""
	if s := instance.GetSsm().Data.(map[string]interface{})["PingStatus"]; s != nil {
		ssm = s.(string)
	}
	asset.State = mapSmmManagedPingStateCode(ssm)
	asset.Labels = mapStringInterfaceToStringString(instance.Tags.Data)
	asset.Name = instance.InstanceId.Data
	if name := asset.Labels["Name"]; name != "" {
		asset.Name = name
	}
	imageName := ""
	if instance.GetImage().Data != nil {
		imageName = instance.GetImage().Data.Name.Data
	}
	creds := []*vault.Credential{
		{
			User: getProbableUsernameFromImageName(imageName),
			Type: vault.CredentialType_aws_ec2_ssm_session,
		},
	}
	host := instance.InstanceId.Data
	if instance.PublicIp.Data != "" {
		host = instance.PublicIp.Data
	}

	asset.Connections = []*inventory.Config{{
		Backend:     "ssh",
		Host:        host,
		Insecure:    true,
		Runtime:     "aws_ec2",
		Credentials: creds,
		Options: map[string]string{
			"region":   instance.Region.Data,
			"profile":  profile,
			"instance": instance.InstanceId.Data,
		},
	}}
	return asset
}

func mapEc2InstanceStateCode(state string) inventory.State {
	switch state {
	case string(types.InstanceStateNameRunning):
		return inventory.State_STATE_RUNNING
	case string(types.InstanceStateNamePending):
		return inventory.State_STATE_PENDING
	case string(types.InstanceStateNameShuttingDown): // 32 is shutting down, which is the step before terminated, assume terminated if we get shutting down
		return inventory.State_STATE_TERMINATED
	case string(types.InstanceStateNameStopping):
		return inventory.State_STATE_STOPPING
	case string(types.InstanceStateNameStopped):
		return inventory.State_STATE_STOPPED
	case string(types.InstanceStateNameTerminated):
		return inventory.State_STATE_TERMINATED
	default:
		log.Warn().Str("state", string(state)).Msg("unknown ec2 state")
		return inventory.State_STATE_UNKNOWN
	}
}

func getProbableUsernameFromImageName(name string) string {
	if strings.Contains(name, "centos") {
		return "centos"
	}
	if strings.Contains(name, "ubuntu") {
		return "ubuntu"
	}
	return "ec2-user"
}

func addConnectionInfoToSSMAsset(instance *mqlAwsSsmInstance, accountId string, profile string) *inventory.Asset {
	asset := &inventory.Asset{}
	asset.Name = instance.InstanceId.Data
	asset.Labels = mapStringInterfaceToStringString(instance.Tags.Data)
	if name := asset.Labels["Name"]; name != "" {
		asset.Name = name
	}
	creds := []*vault.Credential{
		{
			User: getProbableUsernameFromSSMPlatformName(strings.ToLower(instance.PlatformName.Data)),
		},
	}
	if strings.HasPrefix(instance.InstanceId.Data, "i-") {
		creds[0].Type = vault.CredentialType_aws_ec2_ssm_session // this will only work for ec2 instances
	} else {
		log.Warn().Str("asset", asset.Name).Str("id", instance.InstanceId.Data).Msg("cannot use ssm session credentials")
	}
	host := instance.InstanceId.Data
	if instance.IpAddress.Data != "" {
		host = instance.IpAddress.Data
	}
	asset.PlatformIds = []string{awsec2.MondooInstanceID(accountId, instance.Region.Data, instance.InstanceId.Data)}
	asset.Platform = &inventory.Platform{
		Kind:    "virtual_machine",
		Runtime: "ssm_managed",
	}
	asset.Connections = []*inventory.Config{{
		Backend:     "ssh",
		Host:        host,
		Insecure:    true,
		Runtime:     "aws_ec2",
		Credentials: creds,
		Options: map[string]string{
			"region":   instance.Region.Data,
			"profile":  profile,
			"instance": instance.InstanceId.Data,
		},
	}}
	asset.State = mapSmmManagedPingStateCode(instance.PingStatus.Data)
	return asset
}

func getProbableUsernameFromSSMPlatformName(name string) string {
	if strings.HasPrefix(name, "centos") {
		return "centos"
	}
	if strings.HasPrefix(name, "ubuntu") {
		return "ubuntu"
	}
	return "ec2-user"
}

func mapSmmManagedPingStateCode(pingStatus string) inventory.State {
	switch pingStatus {
	case string(ssmtypes.PingStatusOnline):
		return inventory.State_STATE_RUNNING
	case string(ssmtypes.PingStatusConnectionLost):
		return inventory.State_STATE_PENDING
	case string(ssmtypes.PingStatusInactive):
		return inventory.State_STATE_STOPPED
	default:
		return inventory.State_STATE_UNKNOWN
	}
}

func MondooImageRegistryID(id string) string {
	return "//platformid.api.mondoo.app/runtime/docker/registry/" + id
}

func addConnectionInfoToEcrAsset(image *mqlAwsEcrImage, profile string) *inventory.Asset {
	a := &inventory.Asset{}
	a.PlatformIds = []string{containerid.MondooContainerImageID(image.Digest.Data)}
	a.Platform = &inventory.Platform{
		Kind:    "container_image",
		Runtime: "aws_ecr",
	}
	a.Name = ecrImageName(image.RepoName.Data, image.Digest.Data)
	a.State = inventory.State_STATE_ONLINE
	imageTags := []string{}
	for i := range image.Tags.Data {
		tag := image.Tags.Data[i].(string)
		imageTags = append(imageTags, tag)
		a.Connections = append(a.Connections, &inventory.Config{
			Backend: "container_image",
			Host:    image.Uri.Data + ":" + tag,
			Options: map[string]string{
				"region":  image.Region.Data,
				"profile": profile,
			},
		})

	}
	a.Labels = make(map[string]string)
	// store digest
	a.Labels[fmt.Sprintf("ecr.%s.amazonaws.com/digest", image.Region.Data)] = image.Digest.Data
	a.Labels[fmt.Sprintf("ecr.%s.amazonaws.com/tags", image.Region.Data)] = strings.Join(imageTags, ",")

	// store repo digest
	repoDigests := []string{image.Uri.Data + "@" + image.Digest.Data}
	a.Labels[fmt.Sprintf("ecr.%s.amazonaws.com/repo-digests", image.Region.Data)] = strings.Join(repoDigests, ",")

	return a
}

func ecrImageName(repoName string, digest string) string {
	return repoName + "@" + digest
}

func mapContainerInstanceState(status *string) inventory.State {
	if status == nil {
		return inventory.State_STATE_UNKNOWN
	}
	switch *status {
	case "REGISTERING":
		return inventory.State_STATE_PENDING
	case "REGISTRATION_FAILED":
		return inventory.State_STATE_ERROR
	case "ACTIVE":
		return inventory.State_STATE_ONLINE
	case "INACTIVE":
		return inventory.State_STATE_OFFLINE
	case "DEREGISTERING":
		return inventory.State_STATE_STOPPING
	case "DRAINING":
		return inventory.State_STATE_STOPPING
	default:
		return inventory.State_STATE_UNKNOWN
	}
}

func addConnectionInfoToECSContainerInstanceAsset(containerInstance *mqlAwsEcsInstance) *inventory.Asset {
	a := &inventory.Asset{}

	// if asset == nil {
	// 	return nil
	// }
	// if strings.HasPrefix(asset.Id, "i-") {
	// 	ec2i, err := NewEc2Discovery(ecs.mqlDiscovery, ecs.providerConfig, ecs.account)
	// 	if err == nil {
	// 		return ec2i.addConnectionInfoToEc2Asset(asset)
	// 	}
	// }
	// asset.Connections = []*providers.Config{{
	// 	Backend: providers.ProviderType_SSH, // fallback to ssh
	// 	Options: map[string]string{
	// 		"region": asset.Labels[RegionLabel],
	// 	},
	// }}
	// if len(ecs.PassInLabels) > 0 {
	// 	for k, v := range ecs.PassInLabels {
	// 		asset.Labels[k] = v
	// 	}
	// }
	return a
}

func addConnectionInfoToECSContainerAsset(container *mqlAwsEcsContainer) *inventory.Asset {
	a := &inventory.Asset{}
	// runtimeId := asset.Labels[RuntimeIdLabel]
	// if runtimeId == "" {
	// 	return nil
	// }
	// state := asset.Labels[StateLabel]
	// containerArn := asset.Labels[ArnLabel]
	// taskArn := asset.Labels[TaskDefinitionArnLabel]
	// publicIp := asset.Labels[common.IPLabel]
	// region := asset.Labels[RegionLabel]

	// asset.PlatformIds = []string{containerid.MondooContainerID(runtimeId), awsecsid.MondooECSContainerID(containerArn)}
	// asset.Platform = &platform.Platform{
	// 	Kind:    providers.Kind_KIND_CONTAINER,
	// 	Runtime: providers.RUNTIME_AWS_ECS,
	// }
	// asset.State = mapContainerState(state)
	// taskId := ""
	// if arn.IsARN(taskArn) {
	// 	if parsed, err := arn.Parse(taskArn); err == nil {
	// 		if taskIds := strings.Split(parsed.Resource, "/"); len(taskIds) > 1 {
	// 			taskId = taskIds[len(taskIds)-1]
	// 		}
	// 	}
	// }

	// if publicIp != "" {
	// 	asset.Connections = []*providers.Config{{
	// 		Backend: providers.ProviderType_SSH, // looking into ecs-exec for this, if we leave this out the scan assumes its local
	// 		Host:    publicIp,
	// 		Options: map[string]string{
	// 			"region":      region,
	// 			ContainerName: asset.Labels[ContainerName],
	// 			TaskId:        taskId,
	// 		},
	// 	}}
	// } else {
	// 	log.Warn().Str("asset", asset.Name).Msg("no public ip address found")
	// }

	// if len(ecs.PassInLabels) > 0 {
	// 	for k, v := range ecs.PassInLabels {
	// 		asset.Labels[k] = v
	// 	}
	// }

	return a
}

func mapContainerState(state string) inventory.State {
	switch strings.ToLower(state) {
	case "running":
		return inventory.State_STATE_RUNNING
	case "created":
		return inventory.State_STATE_PENDING
	case "paused":
		return inventory.State_STATE_STOPPED
	case "exited":
		return inventory.State_STATE_TERMINATED
	case "restarting":
		return inventory.State_STATE_PENDING
	case "dead":
		return inventory.State_STATE_ERROR
	default:
		log.Warn().Str("state", state).Msg("unknown container state")
		return inventory.State_STATE_UNKNOWN
	}
}
