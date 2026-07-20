// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"encoding/json"
	"fmt"
	"strconv"

	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
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

func convertYamlToDict(valueNode *yaml.Node) (map[string]any, error) {
	data, err := yaml.Marshal(valueNode)
	if err != nil {
		return nil, err
	}

	dict := make(map[string](any))
	err = yaml.Unmarshal(data, &dict)
	if err != nil {
		return nil, err
	}

	return convert.JsonToDict(dict)
}

// nodeToDict resolves the child node under `key` to a Go value suitable for
// `llx.DictData`. Returns nil when the key is absent. The field stays `dict`
// in the schema because CloudFormation values can be scalars (literal default,
// intrinsic function evaluating to a string), lists, or full mapping bodies.
func nodeToDict(parent *yaml.Node, key string) (any, error) {
	_, val, err := gatherMapValue(parent, key)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, nil
		}
		return nil, err
	}
	return convertYamlNodeToValue(val)
}

// nodeToInt extracts a YAML integer scalar at `key`, returning nil when the key
// is absent so callers can distinguish "no constraint" from an explicit 0 (a
// legitimate CloudFormation MinValue). Non-integer scalars return an error so
// callers see malformed input rather than silently defaulting.
func nodeToInt(parent *yaml.Node, key string) (*int64, error) {
	_, val, err := gatherMapValue(parent, key)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, nil
		}
		return nil, err
	}
	if val.Kind != yaml.ScalarNode {
		return nil, fmt.Errorf("expected scalar for %s, got kind %v", key, val.Kind)
	}
	n, err := strconv.ParseInt(val.Value, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid integer for %s: %w", key, err)
	}
	return &n, nil
}

// nodeToDictList walks the sequence at `key` and returns each entry as a
// dict (so heterogeneous strings/numbers/objects survive the round-trip).
func nodeToDictList(parent *yaml.Node, key string) ([]any, error) {
	_, val, err := gatherMapValue(parent, key)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, nil
		}
		return nil, err
	}
	if val.Kind != yaml.SequenceNode {
		return nil, fmt.Errorf("expected sequence for %s, got kind %v", key, val.Kind)
	}
	out := make([]any, 0, len(val.Content))
	for _, item := range val.Content {
		dict, err := convertYamlNodeToValue(item)
		if err != nil {
			return nil, err
		}
		out = append(out, dict)
	}
	return out, nil
}

// convertYamlNodeToValue handles a single YAML node — scalar, sequence, or
// mapping — and returns the matching Go value, normalized for the llx dict
// primitive (ints become float64, nested maps become map[string]any). The
// round-trip through JSON mirrors what convert.JsonToDict does for maps but
// also accepts scalars and lists at the top level.
func convertYamlNodeToValue(n *yaml.Node) (any, error) {
	data, err := yaml.Marshal(n)
	if err != nil {
		return nil, err
	}
	var v any
	if err := yaml.Unmarshal(data, &v); err != nil {
		return nil, err
	}
	jsonBytes, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	var out any
	if err := json.Unmarshal(jsonBytes, &out); err != nil {
		return nil, err
	}
	return out, nil
}
