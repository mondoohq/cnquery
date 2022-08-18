package core_test

import (
	"testing"

	"go.mondoo.io/mondoo/resources/packs/testutils"
)

func TestUUID(t *testing.T) {
	x.TestSimple(t, []testutils.SimpleTest{
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
