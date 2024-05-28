// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"fmt"

	"go.mondoo.com/cnquery/v11/providers-sdk/v1/util/convert"
	"go.mondoo.com/ranger-rpc/codes"
	"go.mondoo.com/ranger-rpc/status"
	"gopkg.in/yaml.v3"
)

func gatherMapValue(n *yaml.Node, key string) (*yaml.Node, *yaml.Node, error) {
	if n == nil {
		return nil, nil, status.Error(codes.InvalidArgument, "node is nil for key "+key)
	}

	// check that we have a map
	if n.Kind != yaml.MappingNode {
		return nil, nil, status.Error(codes.InvalidArgument, fmt.Sprintf("invalid node kind %v for key %s", n.Kind, key))
	}

	// check if content is even
	if len(n.Content)%2 != 0 {
		return nil, nil, status.Error(codes.InvalidArgument, fmt.Sprintf("uneven length %v for key %s", len(n.Content), key))
	}

	// search for key
	for i := 0; i < len(n.Content); i += 2 {
		keyNode := n.Content[i]
		valueNode := n.Content[i+1]

		if keyNode.Value == key {
			return keyNode, valueNode, nil
		}
	}

	return nil, nil, status.Error(codes.NotFound, fmt.Sprintf("key %s not found", key))
}

func convertYamlToDict(valueNode *yaml.Node) (map[string]interface{}, error) {
	data, err := yaml.Marshal(valueNode)
	if err != nil {
		return nil, err
	}

	dict := make(map[string](interface{}))
	err = yaml.Unmarshal(data, &dict)
	if err != nil {
		return nil, err
	}

	return convert.JsonToDict(dict)
}
