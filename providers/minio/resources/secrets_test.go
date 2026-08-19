// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"testing"
	"time"

	madmin "github.com/minio/madmin-go/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/mql/v13/llx"
)

// secretFieldPattern names the things a MinIO deployment will hand back if
// asked and that must never reach a scan result. MinIO's admin API returns
// service account and user secret keys in plaintext on some endpoints, so the
// risk is real rather than theoretical.
var secretFieldPattern = regexp.MustCompile(
	`(?i)(secretkey|secret_key|secret$|\.secret\.|password|passwd|authtoken|auth_token|privatekey|private_key|credential|apikey|api_key|sessiontoken)`)

// TestSchemaDeclaresNoSecretField sweeps every field the schema declares. The
// versions file lists one line per resource and per field, so it covers the
// whole schema mechanically and will fail on a field added later.
func TestSchemaDeclaresNoSecretField(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("minio.lr.versions"))
	require.NoError(t, err)

	paths := 0
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		path := strings.Fields(line)[0]
		paths++
		assert.False(t, secretFieldPattern.MatchString(path),
			"schema field %q looks like key material", path)
	}
	require.Greater(t, paths, 100, "the whole schema was swept, not an empty file")
}

// collectStrings walks the values a resource would be created with and returns
// every string in them, however deeply nested.
func collectStrings(args map[string]*llx.RawData) []string {
	out := []string{}
	var walk func(v any)
	walk = func(v any) {
		if v == nil {
			return
		}
		rv := reflect.ValueOf(v)
		switch rv.Kind() {
		case reflect.String:
			out = append(out, rv.String())
		case reflect.Slice, reflect.Array:
			for i := 0; i < rv.Len(); i++ {
				walk(rv.Index(i).Interface())
			}
		case reflect.Map:
			for _, key := range rv.MapKeys() {
				walk(key.Interface())
				walk(rv.MapIndex(key).Interface())
			}
		case reflect.Ptr, reflect.Interface:
			if !rv.IsNil() {
				walk(rv.Elem().Interface())
			}
		}
	}
	for key, value := range args {
		out = append(out, key)
		if value != nil {
			walk(value.Value)
		}
	}
	return out
}

// TestUserSecretKeyNeverReachesTheSchema plants a secret key in the payload the
// admin API can return and proves the mapping drops it.
func TestUserSecretKeyNeverReachesTheSchema(t *testing.T) {
	const planted = "PLANTED-USER-SECRET-KEY"

	args := userSchemaArgs("alice", madmin.UserInfo{
		SecretKey:  planted,
		PolicyName: "readwrite",
		Status:     madmin.AccountEnabled,
		MemberOf:   []string{"devs"},
		UpdatedAt:  time.Now(),
	})

	for _, value := range collectStrings(args) {
		assert.NotContains(t, value, planted, "the user's secret key reached the schema")
	}
	// The secret must not survive as a key either.
	for key := range args {
		assert.False(t, secretFieldPattern.MatchString(key), "field %q looks like key material", key)
	}
}

// TestServiceAccountSchemaCarriesNoSecretField does the same for a service
// account, whose access key is the public half of the pair. The secret half is
// what AddServiceAccount returns and what must never be modeled.
func TestServiceAccountSchemaCarriesNoSecretField(t *testing.T) {
	expiry := time.Now().Add(time.Hour)
	args := serviceAccountSchemaArgs(madmin.ServiceAccountInfo{
		ParentUser:    "alice",
		AccountStatus: "on",
		ImpliedPolicy: true,
		AccessKey:     "AKEXAMPLE",
		Name:          "ci-pipeline",
		Description:   "used by CI",
		Expiration:    &expiry,
	})

	for key := range args {
		assert.False(t, secretFieldPattern.MatchString(key), "field %q looks like key material", key)
	}
	assert.Equal(t, "AKEXAMPLE", args["accessKey"].Value)
}

// TestSecretFieldPatternCatchesWhatItIsFor is the negative control for the
// sweeps above: a pattern that matched nothing would make every one of them
// pass without proving anything.
func TestSecretFieldPatternCatchesWhatItIsFor(t *testing.T) {
	for _, name := range []string{
		"minio.user.secretKey", "secret_key", "minio.webhook.authToken",
		"auth_token", "password", "privateKey", "minio.user.credential",
		"apiKey", "sessionToken",
	} {
		assert.True(t, secretFieldPattern.MatchString(name), "%q must be caught", name)
	}
	for _, name := range []string{
		"minio.serviceAccount.accessKey", "minio.bucket.name", "minio.kmsKey.name",
		"minio.webhook.clientCertConfigured", "minio.policy.document",
	} {
		assert.False(t, secretFieldPattern.MatchString(name), "%q must not be caught", name)
	}
}

// TestWebhookTargetCarriesNoAuthToken proves the log destination mapping drops
// an authentication token even when the deployment does return one, which older
// and future releases may.
func TestWebhookTargetCarriesNoAuthToken(t *testing.T) {
	const planted = "PLANTED-WEBHOOK-TOKEN"

	configs := []madmin.SubsysConfig{{
		SubSystem: subsysAuditWebhook,
		Target:    "primary",
		KV: []madmin.ConfigKV{
			{Key: "endpoint", Value: "https://audit.example/ingest"},
			{Key: "auth_token", Value: planted},
			{Key: "client_key", Value: "/etc/minio/client.key"},
		},
	}}

	targets := webhookTargetsFromConfig(webhookTypeAudit, configs, nil)
	require.Len(t, targets, 1)

	rendered := reflect.ValueOf(targets[0])
	for i := 0; i < rendered.NumField(); i++ {
		field := rendered.Type().Field(i)
		assert.False(t, secretFieldPattern.MatchString(field.Name),
			"webhook field %q looks like key material", field.Name)
		if rendered.Field(i).Kind() == reflect.String {
			assert.NotContains(t, rendered.Field(i).String(), planted,
				"the authentication token reached field %q", field.Name)
		}
	}
}

// TestFixturesCarryNoCredentials guards the fixtures themselves. They were
// captured from a live deployment, and the secret scanner reads every commit,
// so a credential that slipped into one could not be removed by a later commit.
func TestFixturesCarryNoCredentials(t *testing.T) {
	// The credentials the capture deployment was configured with. None of them
	// should appear anywhere in the recorded responses.
	planted := []string{
		"minioadmin123",
		"alicesecret123",
		"bobsecret123",
		"alicesvcacctsecret0001",
		"supersecrettoken",
		"anothersecret",
		"Jr8EBqxOKo3fUJerHzTS3Yz4N3dRmdUDkAAjSkJeMgA=",
	}

	entries, err := os.ReadDir("testdata")
	require.NoError(t, err)
	require.NotEmpty(t, entries)

	checked := 0
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		data, err := os.ReadFile(filepath.Join("testdata", entry.Name()))
		require.NoError(t, err)
		body := string(data)
		checked++
		for _, secret := range planted {
			assert.NotContains(t, body, secret,
				"fixture %s carries a credential", entry.Name())
		}
		assert.NotContains(t, body, `"secretKey"`,
			"fixture %s carries a secretKey field", entry.Name())
	}
	require.Greater(t, checked, 15, "every fixture was swept")
}
