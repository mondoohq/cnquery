// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package metadata_test

import (
	"errors"
	"testing"

	subject "go.mondoo.com/cnquery/v12/providers/os/id/metadata"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Mock implementation of the recursive interface
type mockRecursive struct {
	mock.Mock
}

func (m *mockRecursive) GetMetadataValue(path string) (string, error) {
	args := m.Called(path)
	return args.String(0), args.Error(1)
}

func TestCrawl_Mock(t *testing.T) {
	m := new(mockRecursive)

	m.On("GetMetadataValue", "valid/path").Return("{\"key\": \"value\"}", nil)
	m.On("GetMetadataValue", "nested/path/").Return("subpath1\nsubpath2", nil)
	m.On("GetMetadataValue", "nested/path/subpath1").Return("value1", nil)
	m.On("GetMetadataValue", "nested/path/subpath2").Return("value2", nil)
	m.On("GetMetadataValue", "error").Return("", errors.New("error"))
	m.On("GetMetadataValue", "empty").Return("", nil)
	m.On("GetMetadataValue", "instance/attributes/ssh-keys").Return("line1\nline2\nline3", nil)

	result, err := subject.Crawl(m, "valid/path")
	assert.NoError(t, err)
	assert.Equal(t, map[string]any{"key": "value"}, result)

	result, err = subject.Crawl(m, "nested/path/")
	assert.NoError(t, err)
	assert.Equal(t, map[string]any{"subpath1": "value1", "subpath2": "value2"}, result)

	_, err = subject.Crawl(m, "error")
	assert.Error(t, err)

	result, err = subject.Crawl(m, "empty")
	assert.NoError(t, err)
	assert.Equal(t, "", result)

	result, err = subject.Crawl(m, "instance/attributes/ssh-keys")
	assert.NoError(t, err)
	assert.Equal(t, "line1\nline2\nline3", result)
}

// Mock is an alternative way to implementation of the recursive interface via map
type mockRecursiveMap struct {
	data map[string]string
}

func (m *mockRecursiveMap) GetMetadataValue(path string) (string, error) {
	if val, exists := m.data[path]; exists {
		return val, nil
	}
	return "", errors.New("not found")
}

func TestCrawl_Map(t *testing.T) {
	mockData := map[string]string{
		"root/":                                  "sub1\nsub2\n",
		"root/sub1":                              "value1",
		"root/sub2":                              "{\"key\": \"value2\"}",
		"json":                                   "{\"field\": 42}",
		"meta-data/managed-ssh-keys/signer-cert": "line1\nline2\n",
		"not-json":                               "random text",
	}
	mock := &mockRecursiveMap{data: mockData}

	tests := []struct {
		name     string
		path     string
		expected any
		hasError bool
	}{
		{"Valid single value", "root/sub1", "value1", false},
		{"Valid JSON object", "root/sub2", map[string]any{"key": "value2"}, false},
		{"Valid JSON field", "json", map[string]any{"field": float64(42)}, false},
		{"Invalid path", "invalid", nil, true},
		{"Non-JSON string", "not-json", "random text", false},
		{"Multiline string", "meta-data/managed-ssh-keys/signer-cert", "line1\nline2\n", false},
		{"Nested structure", "root/", map[string]any{"sub1": "value1", "sub2": map[string]any{"key": "value2"}}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := subject.Crawl(mock, tt.path)
			if tt.hasError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}
