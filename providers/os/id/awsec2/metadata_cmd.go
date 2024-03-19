// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package awsec2

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/ec2/imds"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/cockroachdb/errors"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v10/providers/os/connection/shared"
	"go.mondoo.com/cnquery/v10/providers/os/resources/powershell"
)

const (
	identityUrl = `-H "X-aws-ec2-metadata-token: %s" -v http://169.254.169.254/latest/dynamic/instance-identity/document`
	tokenUrl    = `-H "X-aws-ec2-metadata-token-ttl-seconds: 21600" -X PUT "http://169.254.169.254/latest/api/token"`
	tagNameUrl  = `-H "X-aws-ec2-metadata-token: %s" -v http://169.254.169.254/latest/meta-data/tags/instance/Name`

	identityUrlWindows = `
$Headers = @{
    "X-aws-ec2-metadata-token" = %s
}
Invoke-RestMethod -TimeoutSec 1 -Headers $Headers -URI http://169.254.169.254/latest/dynamic/instance-identity/document -UseBasicParsing | ConvertTo-Json
`

	tokenUrlWindows = `
$Headers = @{
    "X-aws-ec2-metadata-token-ttl-seconds" = "21600"
}
Invoke-RestMethod -Method Put -Uri "http://169.254.169.254/latest/api/token" -Headers $Headers -TimeoutSec 1 -UseBasicParsing
`
	tagNameUrlWindows = `
$Headers = @{
    "X-aws-ec2-metadata-token" = %s
}
Invoke-RestMethod -Method Put -Uri "http://169.254.169.254/latest/meta-data/tags/instance/Name" -Headers $Headers -TimeoutSec 1 -UseBasicParsing
`
)

func NewCommandInstanceMetadata(conn shared.Connection, pf *inventory.Platform, config *aws.Config) *CommandInstanceMetadata {
	return &CommandInstanceMetadata{
		conn:     conn,
		platform: pf,
		config:   config,
	}
}

type CommandInstanceMetadata struct {
	conn     shared.Connection
	platform *inventory.Platform
	config   *aws.Config
}

func (m *CommandInstanceMetadata) Identify() (Identity, error) {
	instanceDocument, err := m.instanceIdentityDocument()
	if err != nil {
		return Identity{}, err
	}
	// parse into struct
	doc := imds.InstanceIdentityDocument{}
	if err := json.NewDecoder(strings.NewReader(instanceDocument)).Decode(&doc); err != nil {
		return Identity{}, errors.Wrap(err, "failed to decode EC2 instance identity document")
	}

	name := doc.InstanceID
	// Note that the tags metadata service has to be enabled for this to work. If not, we fallback to trying to get the name
	// via the aws API (if there's a config provided).
	taggedName, err := m.instanceNameTag()
	if err == nil {
		name = taggedName
	} else if m.config != nil {
		ec2svc := ec2.NewFromConfig(*m.config)
		ctx := context.Background()
		filters := []ec2types.Filter{
			{
				Name:   aws.String("resource-id"),
				Values: []string{doc.InstanceID},
			},
		}
		tags, err := ec2svc.DescribeTags(ctx, &ec2.DescribeTagsInput{Filters: filters})
		if err == nil {
			for _, t := range tags.Tags {
				if t.Key != nil && *t.Key == "Name" && t.Value != nil {
					name = *t.Value
				}
			}
		}
	}
	return Identity{
		InstanceName: name,
		InstanceID:   MondooInstanceID(doc.AccountID, doc.Region, doc.InstanceID),
		AccountID:    "//platformid.api.mondoo.app/runtime/aws/accounts/" + doc.AccountID,
	}, nil
}

type metadataType int

const (
	document metadataType = iota
	instanceNameTag
)

func (m *CommandInstanceMetadata) curlDocument(metadataType metadataType) (string, error) {
	switch {
	case m.platform.IsFamily(inventory.FAMILY_UNIX):
		cmd, err := m.conn.RunCommand("curl " + tokenUrl)
		if err != nil {
			return "", err
		}
		data, err := io.ReadAll(cmd.Stdout)
		if err != nil {
			return "", err
		}
		tokenString := strings.TrimSpace(string(data))

		commandScript := ""
		switch metadataType {
		case document:
			commandScript = "curl " + fmt.Sprintf(identityUrl, tokenString)
		case instanceNameTag:
			commandScript = "curl " + fmt.Sprintf(tagNameUrl, tokenString)
		}

		cmd, err = m.conn.RunCommand(commandScript)
		if err != nil {
			return "", err
		}
		data, err = io.ReadAll(cmd.Stdout)
		if err != nil {
			return "", err
		}

		return strings.TrimSpace(string(data)), nil
	case m.platform.IsFamily(inventory.FAMILY_WINDOWS):
		tokenPwshEncoded := powershell.Encode(tokenUrlWindows)
		cmd, err := m.conn.RunCommand(tokenPwshEncoded)
		if err != nil {
			return "", err
		}
		data, err := io.ReadAll(cmd.Stdout)
		if err != nil {
			return "", err
		}
		tokenString := strings.TrimSpace(string(data))

		commandScript := ""
		switch metadataType {
		case document:
			commandScript = powershell.Encode(fmt.Sprintf(identityUrlWindows, tokenString))
		case instanceNameTag:
			commandScript = powershell.Encode(fmt.Sprintf(tagNameUrlWindows, tokenString))
		}

		cmd, err = m.conn.RunCommand(commandScript)
		if err != nil {
			return "", err
		}
		data, err = io.ReadAll(cmd.Stdout)
		if err != nil {
			return "", err
		}

		return strings.TrimSpace(string(data)), nil
	default:
		return "", errors.New("your platform is not supported by aws metadata identifier resource")
	}
}

func (m *CommandInstanceMetadata) instanceNameTag() (string, error) {
	res, err := m.curlDocument(instanceNameTag)
	if err != nil {
		return "", err
	}
	if strings.Contains(res, "Not Found") {
		return "", errors.New("metadata tags not enabled")
	}
	return res, nil
}

func (m *CommandInstanceMetadata) instanceIdentityDocument() (string, error) {
	return m.curlDocument(document)
}
