package platform

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.mondoo.io/mondoo/motor/transports"
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

func TestPrettyTitle(t *testing.T) {
	test := []struct {
		Platform *Platform
		Expected string
	}{
		{
			Platform: &Platform{
				Title:   "Kali GNU/Linux Rolling",
				Version: "2019.4",
			},
			Expected: "Kali GNU/Linux Rolling, Unknown",
		},
		{
			Platform: &Platform{
				Title:   "Red Hat Enterprise Linux",
				Runtime: transports.RUNTIME_AWS,
				Kind:    transports.Kind_KIND_API,
				Version: "7",
			},
			Expected: "Red Hat Enterprise Linux, Amazon Web Services",
		},
		{
			Platform: &Platform{
				Title:   "Red Hat Enterprise Linux",
				Kind:    transports.Kind_KIND_API,
				Version: "7",
			},
			Expected: "Red Hat Enterprise Linux, API",
		},
		{
			Platform: &Platform{
				Title:   "Red Hat Enterprise Linux 8",
				Runtime: transports.RUNTIME_AWS,
				Kind:    transports.Kind_KIND_API,
				Version: "8",
			},
			Expected: "Red Hat Enterprise Linux 8, Amazon Web Services",
		},
		{
			Platform: &Platform{
				Title:   "Amazon Web Services 8",
				Runtime: transports.RUNTIME_AWS,
				Kind:    transports.Kind_KIND_API,
				Version: "8",
			},
			Expected: "Amazon Web Services 8",
		},
	}

	for i := range test {
		assert.Equal(t, test[i].Expected, test[i].Platform.PrettyTitle(), i)
	}
}
