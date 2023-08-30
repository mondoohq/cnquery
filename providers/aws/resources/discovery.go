package resources

import (
	"fmt"

	"github.com/cockroachdb/errors"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/llx"
	"go.mondoo.com/cnquery/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/providers/aws/config"
	"go.mondoo.com/cnquery/providers/aws/connection"
)

func Discover(runtime *plugin.Runtime) (*inventory.Inventory, error) {
	conn := runtime.Connection.(*connection.AwsConnection)

	in := &inventory.Inventory{Spec: &inventory.InventorySpec{
		Assets: []*inventory.Asset{},
	}}
	if len(conn.Conf.Discover.Targets) == 0 {
		// default to account discovery if none is defined.
		conn.Conf.Discover.Targets = []string{config.DiscoveryAccounts}
	}
	res, err := runtime.CreateResource(runtime, "aws.account", map[string]*llx.RawData{})
	if err != nil {
		return nil, err
	}

	awsAccount := res.(*mqlAwsAccount)

	for i := range conn.Conf.Discover.Targets {
		target := conn.Conf.Discover.Targets[i]

		list, err := discover(runtime, awsAccount, target)
		if err != nil {
			log.Error().Err(err).Msg("error during discovery")
			continue
		}
		in.Spec.Assets = append(in.Spec.Assets, list...)
	}

	return in, nil
}

func accountAsset(conn *connection.AwsConnection, awsAccount *mqlAwsAccount) *inventory.Asset {
	var name string
	if len(awsAccount.Aliases.Data) > 0 {
		alias := awsAccount.Aliases.Data[0].(string)
		name = AssembleIntegrationName(alias, awsAccount.Id.Data)
	}
	id := "//platformid.api.mondoo.app/runtime/aws/accounts/" + awsAccount.Id.Data

	return &inventory.Asset{
		PlatformIds: []string{id},
		Name:        name,
		Platform:    &inventory.Platform{Name: "aws", Runtime: "aws"},
		Connections: []*inventory.Config{conn.Conf},
	}
}

func AssembleIntegrationName(alias string, id string) string {
	if alias == "" {
		return fmt.Sprintf("AWS Account %s", id)
	}
	return fmt.Sprintf("AWS Account %s (%s)", alias, id)
}

func discover(runtime *plugin.Runtime, awsAccount *mqlAwsAccount, target string) ([]*inventory.Asset, error) {
	conn := runtime.Connection.(*connection.AwsConnection)
	accountId := awsAccount.Id.Data
	assetList := []*inventory.Asset{}
	switch target {
	case config.DiscoveryAuto:
		assetList = append(assetList, accountAsset(conn, awsAccount))
	case config.DiscoveryAccounts:
		assetList = append(assetList, accountAsset(conn, awsAccount))
	case config.DiscoveryIAMUsers:
		res, err := NewResource(runtime, "aws.iam", map[string]*llx.RawData{})
		if err != nil {
			return nil, err
		}

		iam := res.(*mqlAwsIam)

		users := iam.GetUsers()
		if users == nil {
			return assetList, nil
		}

		for i := range users.Data {
			user := users.Data[i].(*mqlAwsIamUser)
			labels := map[string]string{}

			m := mqlObject{
				name: user.Name.Data, labels: labels,
				awsObject: awsObject{
					account: accountId, region: "us-east-1", arn: user.Arn.Data,
					id: user.Id.Data, service: "iam", objectType: "user",
				},
			}
			assetList = append(assetList, MqlObjectToAsset(accountId, m, conn))
		}
	}
	return assetList, nil
}

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
	if mqlObject.name == "" {
		mqlObject.name = mqlObject.awsObject.id
	}
	// if err := validate(mqlObject); err != nil {
	// 	log.Error().Err(err).Msg("missing values in mql object to asset translation")
	// 	return nil
	// }
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
			Kind:    "aws_object",
			Runtime: "aws",
		},
		Labels:      mqlObject.labels,
		Connections: []*inventory.Config{conn.Conf},
	}
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
