package types

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTypes(t *testing.T) {
	list := []struct {
		T             Type
		ExpectedLabel string
	}{
		{
			T: Unset, ExpectedLabel: "unset",
		}, {
			T: Any, ExpectedLabel: "any",
		}, {
			T: Nil, ExpectedLabel: "null",
		}, {
			T: Ref, ExpectedLabel: "ref",
		}, {
			T: Bool, ExpectedLabel: "bool",
		}, {
			T: Int, ExpectedLabel: "int",
		}, {
			T: Float, ExpectedLabel: "float",
		}, {
			T: String, ExpectedLabel: "string",
		}, {
			T: Regex, ExpectedLabel: "regex",
		}, {
			T: Time, ExpectedLabel: "time",
		}, {
			T: Dict, ExpectedLabel: "dict",
		}, {
			T: Score, ExpectedLabel: "score",
		}, {
			T: Array(String), ExpectedLabel: "[]string",
		}, {
			T: Map(String, String), ExpectedLabel: "map[string]string",
		}, {
			T: Resource("mockresource"), ExpectedLabel: "mockresource",
		}, {
			T: Function('f', []Type{String, Int}), ExpectedLabel: "function(..??..)",
		},
	}

	for i := range list {
		test := list[i]

		// test for human friendly name
		assert.Equal(t, test.ExpectedLabel, test.T.Label())
	}
}
