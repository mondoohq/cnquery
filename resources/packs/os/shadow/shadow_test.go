package shadow_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnquery/resources/packs/os/shadow"
	"go.mondoo.com/cnquery/motor/providers/mock"
)

func TestParseShadow(t *testing.T) {
	mock, err := mock.NewFromTomlFile("./testdata/debian.toml")
	require.NoError(t, err)

	f, err := mock.FS().Open("/etc/shadow")
	require.NoError(t, err)
	defer f.Close()

	shadowEntries, err := shadow.ParseShadow(f)
	require.NoError(t, err)

	assert.Equal(t, 27, len(shadowEntries))

	// 18368 days + jan 1 1970 = 2020-04-16 00:00:00 +0000 UTC
	date := time.Date(2020, 0o4, 16, 0, 0, 0, 0, time.UTC)
	expected := &shadow.ShadowEntry{
		User:         "chris",
		Password:     "*",
		LastChanged:  &date,
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
