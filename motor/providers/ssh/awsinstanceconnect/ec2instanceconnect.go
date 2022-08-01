package awsinstanceconnect

import (
	"context"

	"github.com/cockroachdb/errors"
	"github.com/sethvargo/go-password/password"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2instanceconnect"
	"go.mondoo.io/mondoo/motor/providers/ssh/keypair"
)

type generator struct {
	cfg aws.Config
}

func New(cfg aws.Config) *generator {
	return &generator{cfg: cfg}
}

type InstanceCredentials struct {
	KeyPair         *keypair.SSH
	PublicDnsName   string
	PrivateDnsName  string
	PublicIpAddress string
}

func (c *generator) GenerateCredentials(instanceID string, user string) (*InstanceCredentials, error) {
	ctx := context.Background()
	ec2srv := ec2.NewFromConfig(c.cfg)
	resp, err := ec2srv.DescribeInstances(ctx, &ec2.DescribeInstancesInput{
		InstanceIds: []string{instanceID},
	})
	if err != nil {
		return nil, err
	}

	if len(resp.Reservations) != 1 || len(resp.Reservations[0].Instances) != 1 {
		return nil, errors.New("could not find the instance")
	}

	instance := resp.Reservations[0].Instances[0]

	// generate random passphrase
	passphrase, err := password.Generate(64, 10, 10, false, false)
	if err != nil {
		return nil, err
	}

	sshkeypair, err := keypair.NewRSAKeys(keypair.DefaultRsaBits, []byte(passphrase))
	if err != nil {
		return nil, err
	}

	ec2ic := ec2instanceconnect.NewFromConfig(c.cfg)
	_, err = ec2ic.SendSSHPublicKey(ctx, &ec2instanceconnect.SendSSHPublicKeyInput{
		InstanceId:       instance.InstanceId,
		AvailabilityZone: instance.Placement.AvailabilityZone,
		InstanceOSUser:   aws.String(user),
		SSHPublicKey:     aws.String(string(sshkeypair.PublicKey)),
	})
	if err != nil {
		return nil, err
	}

	ic := &InstanceCredentials{
		KeyPair: sshkeypair,
	}

	if instance.PublicDnsName != nil {
		ic.PublicDnsName = *instance.PublicDnsName
	}

	if instance.PrivateDnsName != nil {
		ic.PrivateDnsName = *instance.PrivateDnsName
	}

	if instance.PublicIpAddress != nil {
		ic.PublicIpAddress = *instance.PublicIpAddress
	}

	return ic, nil
}
