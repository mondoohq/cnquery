// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
)

func newTestModel(id, displayName string) *mqlTogetherModel {
	return &mqlTogetherModel{
		Id:          plugin.TValue[string]{Data: id, State: plugin.StateIsSet},
		DisplayName: plugin.TValue[string]{Data: displayName, State: plugin.StateIsSet},
	}
}

func TestModelFamily(t *testing.T) {
	tests := []struct {
		id   string
		want string
	}{
		{"meta-llama/Llama-3.3-70B-Instruct-Turbo", "Llama"},
		{"meta-llama/Meta-Llama-3.1-8B-Instruct-Turbo", "Llama"},
		{"Qwen/Qwen2.5-72B-Instruct-Turbo", "Qwen"},
		{"mistralai/Mistral-7B-v0.1", "Mistral"},
		{"mistralai/Mixtral-8x7B-Instruct-v0.1", "Mixtral"},
		{"google/Gemma-2-9B-it", "Gemma"},
		{"deepseek-ai/DeepSeek-V3", "DeepSeek"},
		{"microsoft/Phi-3-mini-4k-instruct", "Phi"},
	}
	for _, tt := range tests {
		t.Run(tt.id, func(t *testing.T) {
			m := newTestModel(tt.id, "")
			got, err := m.family()
			assert.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestModelParameterSize(t *testing.T) {
	tests := []struct {
		id   string
		want string
	}{
		{"meta-llama/Llama-3.3-70B-Instruct-Turbo", "70B"},
		{"meta-llama/Meta-Llama-3.1-8B-Instruct-Turbo", "8B"},
		{"Qwen/Qwen2.5-72B-Instruct-Turbo", "72B"},
		{"mistralai/Mistral-7B-v0.1", "7B"},
		{"mistralai/Mixtral-8x7B-Instruct-v0.1", "8x7B"},
		{"google/Gemma-2-9B-it", "9B"},
		{"openai/gpt-4o", ""},
	}
	for _, tt := range tests {
		t.Run(tt.id, func(t *testing.T) {
			m := newTestModel(tt.id, "")
			got, err := m.parameterSize()
			assert.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestModelQuantization(t *testing.T) {
	tests := []struct {
		id   string
		want string
	}{
		{"meta-llama/Llama-3.3-70B-Instruct-Turbo", ""},
		{"org/Model-7B-fp8", "fp8"},
		{"org/Model-7B-int4-variant", "int4"},
		{"org/Model-7B-AWQ", "awq"},
		{"org/Model-7B-GPTQ", "gptq"},
	}
	for _, tt := range tests {
		t.Run(tt.id, func(t *testing.T) {
			m := newTestModel(tt.id, "")
			got, err := m.quantization()
			assert.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestModelDescription(t *testing.T) {
	m := newTestModel("meta-llama/Llama-3.3-70B-Instruct-Turbo", "Llama 3.3 70B Instruct Turbo")
	got, err := m.description()
	assert.NoError(t, err)
	assert.Equal(t, "Llama 3.3 70B Instruct Turbo", got)
}
