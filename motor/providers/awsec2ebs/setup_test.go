// +build debugtest

package awsec2ebs

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"gotest.tools/assert"
)

func awsTestConfig() aws.Config {
	cfg, err := config.LoadDefaultConfig(context.Background(),
		config.WithSharedConfigProfile("mondoo-demo"),
		config.WithRegion("us-east-1"),
	)
	if err != nil {
		panic(err)
	}

	return cfg
}

func TestFindRecentSnapshot(t *testing.T) {
	ec2svc := ec2.NewFromConfig(awsTestConfig())
	e := Ec2EbsTransport{scannerRegionEc2svc: ec2svc}
	found, _ := e.FindRecentSnapshotForVolume(context.Background(), VolumeId{Id: "vol-0c04d709ea3e59096", Region: "us-east-1", Account: "185972265011"})
	assert.Equal(t, found, true)
	// found, _ = e.FindRecentSnapshotForVolume(context.Background(), VolumeId{Id: "vol-0d5df63d656ac4d9c", Region: "us-east-1", Account: "185972265011"})
	// assert.Equal(t, found, true)
}
