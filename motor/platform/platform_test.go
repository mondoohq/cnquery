package platform

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestFamilyParents(t *testing.T) {

	test := []struct {
		Platform string
		Expected []string
	}{
		{
			Platform: "redhat",
			Expected: []string{"os", "unix", "linux", "redhat"},
		},
		{
			Platform: "centos",
			Expected: []string{"os", "unix", "linux", "redhat"},
		},
		{
			Platform: "debian",
			Expected: []string{"os", "unix", "linux", "debian"},
		},
		{
			Platform: "ubuntu",
			Expected: []string{"os", "unix", "linux", "debian"},
		},
	}

	for i := range test {
		assert.Equal(t, test[i].Expected, Family(test[i].Platform), test[i].Platform)
	}
}

func TestIsFamily(t *testing.T) {

	test := []struct {
		Val      bool
		Expected bool
	}{
		{
			Val:      IsFamily("redhat", FAMILY_LINUX),
			Expected: true,
		},
		{
			Val:      IsFamily("redhat", FAMILY_UNIX),
			Expected: true,
		},
		{
			Val:      IsFamily("redhat", "redhat"),
			Expected: true,
		},
		{
			Val:      IsFamily("centos", FAMILY_LINUX),
			Expected: true,
		},
		{
			Val:      IsFamily("centos", "redhat"),
			Expected: true,
		},
	}

	for i := range test {
		assert.Equal(t, test[i].Expected, test[i].Val, i)
	}

}
