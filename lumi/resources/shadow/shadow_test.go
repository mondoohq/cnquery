package shadow_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.io/mondoo/lumi/resources/shadow"
	"go.mondoo.io/mondoo/motor/motoros/mock"
	"go.mondoo.io/mondoo/motor/motoros/types"
)

func TestParseShadow(t *testing.T) {
	mock, err := mock.NewFromToml(&types.Endpoint{Backend: "mock", Path: "./testdata/debian.toml"})
	require.NoError(t, err)

	f, err := mock.File("/etc/shadow")
	require.NoError(t, err)
	defer f.Close()

	shadowEntries, err := shadow.ParseShadow(f)
	require.NoError(t, err)

	assert.Equal(t, 27, len(shadowEntries))

	expected := &shadow.ShadowEntry{
		User:         "chris",
		Password:     "*",
		LastChanges:  "18368",
		MinDays:      "0",
		MaxDays:      "99999",
		WarnDays:     "7",
		InactiveDays: "",
		ExpiryDates:  "",
		Reserved:     "",
	}
	found := findUser(shadowEntries, "chris")
	assert.Equal(t, expected, found)

}

func findUser(shadowEntries []shadow.ShadowEntry, user string) *shadow.ShadowEntry {
	for i := range shadowEntries {
		if shadowEntries[i].User == user {
			return &shadowEntries[i]
		}
	}
	return nil
}
