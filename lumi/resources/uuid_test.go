package resources_test

import (
	"testing"
)

func TestUUID(t *testing.T) {
	runSimpleTests(t, []simpleTest{
		{
			"uuid('6ba7b810-9dad-11d1-80b4-00c04fd430c8').value",
			0, "6ba7b810-9dad-11d1-80b4-00c04fd430c8",
		},
		{
			"uuid('6ba7b810-9dad-11d1-80b4-00c04fd430c8').variant",
			0, "RFC4122",
		},
		{
			"uuid('6ba7b810-9dad-11d1-80b4-00c04fd430c8').version",
			0, int64(1),
		},
	})
}
