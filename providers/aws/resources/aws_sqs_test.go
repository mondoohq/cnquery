// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSqsQueueName(t *testing.T) {
	tests := []struct {
		name     string
		queueURL string
		expected string
	}{
		{"standard url", "https://sqs.us-east-1.amazonaws.com/123456789012/MyQueue", "MyQueue"},
		{"fifo queue", "https://sqs.eu-west-1.amazonaws.com/123456789012/MyQueue.fifo", "MyQueue.fifo"},
		{"china partition", "https://sqs.cn-north-1.amazonaws.com.cn/123456789012/MyQueue", "MyQueue"},
		{"legacy host", "https://us-east-1.queue.amazonaws.com/123456789012/MyQueue", "MyQueue"},
		{"no path separator", "MyQueue", "MyQueue"},
		{"empty", "", ""},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.expected, sqsQueueName(test.queueURL))
		})
	}
}

func TestRegionFromSqsQueueURL(t *testing.T) {
	tests := []struct {
		name     string
		queueURL string
		expected string
	}{
		{"standard url", "https://sqs.us-east-1.amazonaws.com/123456789012/MyQueue", "us-east-1"},
		{"fips url", "https://sqs-fips.us-gov-west-1.amazonaws.com/123456789012/MyQueue", "us-gov-west-1"},
		{"china partition", "https://sqs.cn-north-1.amazonaws.com.cn/123456789012/MyQueue", "cn-north-1"},
		{"legacy host", "https://us-east-1.queue.amazonaws.com/123456789012/MyQueue", "us-east-1"},
		{"region-less legacy host", "https://queue.amazonaws.com/123456789012/MyQueue", ""},
		{"unrecognized host", "https://example.com/123456789012/MyQueue", ""},
		{"not a url", "MyQueue", ""},
		{"empty", "", ""},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.expected, regionFromSqsQueueURL(test.queueURL))
		})
	}
}
