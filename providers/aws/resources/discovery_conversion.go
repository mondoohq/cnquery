// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	ssmtypes "github.com/aws/aws-sdk-go-v2/service/ssm/types"
	"github.com/aws/aws-sdk-go/aws/arn"
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
	platformName := getPlatformName(mqlObject.awsObject)
	if platformName == "" {
		log.Error().Err(errors.New("could not fetch platform info for object")).Msg("missing runtime info")
		return nil
	}
	platformid := MondooObjectID(mqlObject.awsObject)
	t := conn.Conf
	t.PlatformId = platformid
	return &inventory.Asset{
		PlatformIds: []string{platformid, mqlObject.awsObject.arn},
		Name:        mqlObject.name,
		Platform:    connection.GetPlatformForObject(platformName),
		Labels:      mqlObject.labels,
		Connections: []*inventory.Config{cloneInventoryConf(conn.Conf)},
	}
}

func cloneInventoryConf(invConf *inventory.Config) *inventory.Config {
	invConfClone := invConf.Clone()
	// We do not want to run discovery again for the already discovered assets
	invConfClone.Discover = &inventory.Discovery{}
	return invConfClone
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

func getPlatformName(awsObject awsObject) string {
	switch awsObject.service {
	case "s3":
		if awsObject.objectType == "bucket" {
			return "aws-s3-bucket"
		}
	case "cloudtrail":
		if awsObject.objectType == "trail" {
			return "aws-cloudtrail-trail"
		}
	case "rds":
		if awsObject.objectType == "dbinstance" {
			return "aws-rds-dbinstance"
		}
	case "dynamodb":
		if awsObject.objectType == "table" {
			return "aws-dynamodb-table"
		}
	case "redshift":
		if awsObject.objectType == "cluster" {
			return "aws-redshift-cluster"
		}
	case "vpc":
		if awsObject.objectType == "vpc" {
			return "aws-vpc"
		}
	case "ec2":
		switch awsObject.objectType {
		case "securitygroup":
			return "aws-security-group"
		case "volume":
			return "aws-ec2-volume"
		case "snapshot":
			return "aws-ec2-snapshot"
		case "instance":
			return "aws-ec2-instance"
		}
	case "iam":
		switch awsObject.objectType {
		case "user":
			return "aws-iam-user"

		case "group":
			return "aws-iam-group"
		}
	case "cloudwatch":
		if awsObject.objectType == "loggroup" {
			return "aws-cloudwatch-loggroup"
		}
	case "lambda":
		if awsObject.objectType == "function" {
			return "aws-lambda-function"
		}
	case "ecs":
		if awsObject.objectType == "container" {
			return "aws-ecs-container"
		}
		if awsObject.objectType == "instance" {
			return "aws-ecs-instance"
		}
	case "efs":
		if awsObject.objectType == "filesystem" {
			return "aws-efs-filesystem"
		}
	case "gateway":
		if awsObject.objectType == "restapi" {
			return "aws-gateway-restapi"
		}
	case "elb":
		if awsObject.objectType == "loadbalancer" {
			return "aws-elb-loadbalancer"
		}
	case "es":
		if awsObject.objectType == "domain" {
			return "aws-es-domain"
		}
	case "kms":
		if awsObject.objectType == "key" {
			return "aws-kms-key"
		}
	case "sagemaker":
		if awsObject.objectType == "notebookinstance" {
			return "aws-sagemaker-notebookinstance"
		}
	case "ssm":
		if awsObject.objectType == "instance" {
			return "aws-ssm-instance"
		}
	case "ecr":
		if awsObject.objectType == "image" {
			return "aws-ecr-image"
		}
	}
	return ""
}

func accountAsset(conn *connection.AwsConnection, awsAccount *mqlAwsAccount) *inventory.Asset {
	var alias string
	aliases := awsAccount.GetAliases()
	if len(aliases.Data) > 0 {
		alias = aliases.Data[0].(string)
	}
	name := AssembleIntegrationName(alias, awsAccount.Id.Data)
	justId := strings.TrimPrefix(awsAccount.Id.Data, "aws.account/")

	id := "//platformid.api.mondoo.app/runtime/aws/accounts/" + justId

	return &inventory.Asset{
		PlatformIds: []string{id},
		Name:        name,
		Platform:    connection.GetPlatformForObject(""),
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

func addConnectionInfoToECSContainerAsset(container *mqlAwsEcsContainer) *inventory.Asset {
	a := &inventory.Asset{}

	runtimeId := container.RuntimeId.Data
	if runtimeId == "" {
		return nil
	}
	state := container.Status.Data
	containerArn := container.Arn.Data
	taskArn := container.TaskArn.Data
	publicIp := container.GetPublicIp().Data
	region := container.Region.Data

	a.Name = container.Name.Data
	a.PlatformIds = []string{containerid.MondooContainerID(runtimeId), MondooECSContainerID(containerArn)}
	a.Platform = &inventory.Platform{
		Kind:    "container",
		Runtime: "aws_ecs",
	}
	a.State = mapContainerState(state)
	taskId := ""
	if arn.IsARN(taskArn) {
		if parsed, err := arn.Parse(taskArn); err == nil {
			if taskIds := strings.Split(parsed.Resource, "/"); len(taskIds) > 1 {
				taskId = taskIds[len(taskIds)-1]
			}
		}
	}

	if publicIp != "" {
		a.Connections = []*inventory.Config{{
			Backend: "ssh",
			Host:    publicIp,
			Options: map[string]string{
				"region":         region,
				"container_name": container.Name.Data,
				"task_id":        taskId,
			},
		}}
	} else {
		log.Warn().Str("asset", a.Name).Msg("no public ip address found")
	}

	return a
}

func addConnectionInfoToECSContainerInstanceAsset(inst *mqlAwsEcsInstance, accountId string, conn *connection.AwsConnection) *inventory.Asset {
	m := mqlObject{
		name: inst.Id.Data, labels: map[string]string{},
		awsObject: awsObject{
			account: accountId, region: inst.Region.Data, arn: inst.Arn.Data,
			id: inst.Id.Data, service: "ecs", objectType: "instance",
		},
	}
	a := MqlObjectToAsset(accountId, m, conn)
	a.Connections = []*inventory.Config{{
		Backend: "ssh", // fallback to ssh
		Options: map[string]string{
			"region": inst.Region.Data,
		},
	}}
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

func MondooECSContainerID(containerArn string) string {
	var account, region, id string
	if arn.IsARN(containerArn) {
		if p, err := arn.Parse(containerArn); err == nil {
			account = p.AccountID
			region = p.Region
			id = p.Resource
		}
	}
	return "//platformid.api.mondoo.app/runtime/aws/ecs/v1/accounts/" + account + "/regions/" + region + "/" + id
}
