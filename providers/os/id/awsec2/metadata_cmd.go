// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package awsec2

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/ec2/imds"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/cockroachdb/errors"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v11/providers/os/connection/shared"
)

const (
	identityURLPath = "dynamic/instance-identity/document"
	metadataURLPath = "meta-data/"
	tagNameURLPath  = "meta-data/tags/instance/Name"
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

	// used internally to avoid fetching a token multiple times
	token string
}

func (m *CommandInstanceMetadata) RawMetadata() (any, error) {
	return recursive{m.curlDocument}.Crawl(metadataURLPath)
}

func (m *CommandInstanceMetadata) Identify() (Identity, error) {
	instanceDocument, err := m.instanceIdentityDocument()
	if err != nil {
		return Identity{}, err
	}
	log.Debug().Str("instance_document", instanceDocument).Msg("identity")

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

func (m *CommandInstanceMetadata) curlDocument(metadataPath string) (string, error) {
	token, err := m.getToken()
	if err != nil {
		return "", err
	}

	var commandString string
	switch {
	case m.platform.IsFamily(inventory.FAMILY_UNIX):
		commandString = unixMetadataCmdString(token, metadataPath)
	case m.platform.IsFamily(inventory.FAMILY_WINDOWS):
		commandString = windowsMetadataCmdString(token, metadataPath)
	default:
		return "", errors.New("your platform is not supported by aws metadata identifier resource")
	}

	log.Debug().Str("command_string", commandString).Msg("running os command")
	cmd, err := m.conn.RunCommand(commandString)
	if err != nil {
		return "", err
	}
	log.Debug().Str("hash", hashCmd(commandString)).Msg("executed")
	data, err := io.ReadAll(cmd.Stdout)
	log.Debug().Msg("read")
	return strings.TrimSpace(string(data)), err
}

func hashCmd(message string) string {
	hash := sha256.New()
	hash.Write([]byte(message))
	return hex.EncodeToString(hash.Sum(nil))
}
func (m *CommandInstanceMetadata) getToken() (string, error) {
	if m.token != "" {
		return m.token, nil
	}

	var commandString string
	switch {
	case m.platform.IsFamily(inventory.FAMILY_UNIX):
		commandString = unixTokenCmdString()
	case m.platform.IsFamily(inventory.FAMILY_WINDOWS):
		commandString = windowsTokenCmdString()
	default:
		return "", errors.New("your platform is not supported by aws metadata identifier resource")
	}

	cmd, err := m.conn.RunCommand(commandString)
	if err != nil {
		return "", err
	}
	data, err := io.ReadAll(cmd.Stdout)
	return strings.TrimSpace(string(data)), err
}

func (m *CommandInstanceMetadata) instanceNameTag() (string, error) {
	res, err := m.curlDocument(tagNameURLPath)
	if err != nil {
		return "", err
	}
	if strings.Contains(res, "Not Found") {
		return "", errors.New("metadata tags not enabled")
	}
	return res, nil
}

func (m *CommandInstanceMetadata) instanceIdentityDocument() (string, error) {
	return m.curlDocument(identityURLPath)
}
