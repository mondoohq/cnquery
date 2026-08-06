// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"encoding/json"
	"testing"

	"github.com/okta/okta-sdk-golang/v5/okta"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDecodeOktaLogStreamAws covers the AWS EventBridge variant of the log
// stream union.
func TestDecodeOktaLogStreamAws(t *testing.T) {
	t.Parallel()
	wire := `{
		"id": "ls1",
		"name": "eventbridge",
		"type": "aws_eventbridge",
		"status": "ACTIVE",
		"created": "2026-01-02T03:04:05.000Z",
		"lastUpdated": "2026-02-03T04:05:06.000Z",
		"settings": {"accountId": "123456789012", "eventSourceName": "okta", "region": "us-east-1"}
	}`

	stream, err := decodeOktaLogStream(unmarshalLogStream(t, wire))
	require.NoError(t, err)

	assert.Equal(t, "ls1", stream.Id)
	assert.Equal(t, "eventbridge", stream.Name)
	assert.Equal(t, "aws_eventbridge", stream.Type)
	assert.Equal(t, "ACTIVE", stream.Status)
	assert.Equal(t, "123456789012", stream.Settings["accountId"])
	assert.Equal(t, "us-east-1", stream.Settings["region"])
	require.NotNil(t, stream.Created)
	assert.Equal(t, 2026, stream.Created.Year())
}

// TestDecodeOktaLogStreamSplunkRedactsToken is the one that matters: the Splunk
// HEC token is a credential capable of writing into the customer's Splunk
// index, and it must never reach a resource field.
func TestDecodeOktaLogStreamSplunkRedactsToken(t *testing.T) {
	t.Parallel()
	wire := `{
		"id": "ls2",
		"name": "splunk",
		"type": "splunk_cloud_logstreaming",
		"status": "ACTIVE",
		"settings": {
			"host": "acme.splunkcloud.com",
			"edition": "gcp",
			"token": "super-secret-hec-token"
		}
	}`

	stream, err := decodeOktaLogStream(unmarshalLogStream(t, wire))
	require.NoError(t, err)

	assert.Equal(t, "acme.splunkcloud.com", stream.Settings["host"])
	assert.Equal(t, "gcp", stream.Settings["edition"])
	assert.NotContains(t, stream.Settings, "token", "the Splunk HEC token must never be exposed")

	// Belt and braces: the token value must not survive anywhere in settings.
	encoded, err := json.Marshal(stream.Settings)
	require.NoError(t, err)
	assert.NotContains(t, string(encoded), "super-secret-hec-token")
}

// TestDecodeOktaLogStreamNoSettings documents what the SDK's union decoder does
// with a response that omits `settings` and the timestamps. It does not produce
// an absent settings block: it selects a variant and zero-fills that variant's
// typed fields, so the keys are present with empty values, and the non-pointer
// `created`/`lastUpdated` come back as the zero time rather than null.
//
// The decode must not fail or panic on that shape. Okta always sends these
// fields for a real log stream, so no coercion is applied; this test exists to
// make the behavior visible if that ever stops being true.
func TestDecodeOktaLogStreamNoSettings(t *testing.T) {
	t.Parallel()
	stream, err := decodeOktaLogStream(unmarshalLogStream(t,
		`{"id":"ls3","name":"bare","type":"aws_eventbridge","status":"INACTIVE"}`))
	require.NoError(t, err)

	assert.Equal(t, "ls3", stream.Id)
	assert.Equal(t, "INACTIVE", stream.Status)
	assert.Equal(t, "", stream.Settings["accountId"], "zero-filled by the union decoder")
	assert.NotContains(t, stream.Settings, "token")
	require.NotNil(t, stream.Created)
	assert.True(t, stream.Created.IsZero(), "omitted timestamps decode to the zero time")
}

func unmarshalLogStream(t *testing.T, wire string) *okta.ListLogStreams200ResponseInner {
	t.Helper()
	entry := &okta.ListLogStreams200ResponseInner{}
	require.NoError(t, json.Unmarshal([]byte(wire), entry))
	return entry
}
