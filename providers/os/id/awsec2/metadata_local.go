// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package awsec2

import (
	"context"
	"encoding/json"
	"io"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/ec2/imds"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/rs/zerolog/log"

	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

func NewLocal(cfg aws.Config) *LocalEc2InstanceMetadata {
	return &LocalEc2InstanceMetadata{config: cfg}
}

// Ec2InstanceMetadata returns the instance id
// TODO: we may want to implement instance verification as documented in
// https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/instance-identity-documents.html
type LocalEc2InstanceMetadata struct {
	config aws.Config
}

func (m *LocalEc2InstanceMetadata) RawMetadata() (any, error) {
	client := imds.NewFromConfig(m.config)
	return m.getMetadataRecursively(client, "")
}

// isJSON checks if a string is valid JSON
func isJSON(data string) bool {
	var js json.RawMessage
	return json.Unmarshal([]byte(data), &js) == nil
}

// isMultilineString checks if a path should be treated as a raw multiline string
func isMultilineString(path string) bool {
	// Add any additional paths that should be treated as multiline strings
	return path == "managed-ssh-keys/signer-cert"
}

func (m *LocalEc2InstanceMetadata) getMetadataRecursively(client *imds.Client, path string) (any, error) {
	log.Trace().
		Str("path", path).
		Msg("os.id.awsec2> metadata")
	data, err := m.getMetadataValue(client, path)
	if err != nil {
		return nil, err
	}

	// If the response is JSON, parse it
	if isJSON(data) {
		var jsonData interface{}
		if err := json.Unmarshal([]byte(data), &jsonData); err != nil {
			return nil, err
		}
		return jsonData, nil
	}

	// Handle specific paths that return multiline strings (e.g., "managed-ssh-keys/signer-cert")
	if isMultilineString(path) {
		return data, nil // Preserve as a raw string
	}

	lines := strings.Split(data, "\n")

	// If the data contains sub-paths, fetch them recursively
	if len(lines) > 1 || strings.HasSuffix(data, "/") {
		result := make(map[string]any)

		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}

			subPath := path + line

			subData, err := m.getMetadataRecursively(client, subPath)
			if err != nil {
				log.Trace().Err(err).
					Str("path", path).
					Str("line", line).
					Msg("os.id.awsec2> failed to get sub-path metadata")
				continue
			}

			result[strings.TrimSuffix(line, "/")] = subData
		}

		return result, nil
	}

	// If it's a single value, return it as a string
	return data, nil
}

func (m *LocalEc2InstanceMetadata) Identify() (Identity, error) {
	metadata := imds.NewFromConfig(m.config)
	ec2svc := ec2.NewFromConfig(m.config)
	ctx := context.Background()
	doc, err := metadata.GetInstanceIdentityDocument(ctx, &imds.GetInstanceIdentityDocumentInput{})
	if err != nil {
		return Identity{}, err
	}
	name := ""
	// try and fetch this from the metadata, if the tag metadata service is enabled.
	nameTag, err := m.getMetadataValue(metadata, "tags/instance/Name")
	if err == nil {
		name = nameTag
	} else {
		// if not enabled, try and use the aws api as a fallback. this only works if the aws config is setup
		// correctly on the ec2 instance.
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
	if name == "" {
		name = doc.InstanceID
	}
	return Identity{
		InstanceName: name,
		InstanceID:   MondooInstanceID(doc.AccountID, doc.Region, doc.InstanceID),
		AccountID:    "//platformid.api.mondoo.app/runtime/aws/accounts/" + doc.AccountID,
	}, nil
}

// gets the metadata at the relative specified path. The base path is /latest/meta-data
// so the path param needs to only specify which metadata path is requested
func (m *LocalEc2InstanceMetadata) getMetadataValue(client *imds.Client, path string) (string, error) {
	output, err := client.GetMetadata(context.TODO(), &imds.GetMetadataInput{
		Path: path,
	})
	if err != nil {
		return "", err
	}
	defer output.Content.Close()
	bytes, err := io.ReadAll(output.Content)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}
