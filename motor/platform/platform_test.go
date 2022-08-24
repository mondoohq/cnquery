package platform

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.mondoo.com/cnquery/motor/providers"
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
				Family:  []string{"linux", "unix", "os"},
			},
			Expected: "Kali GNU/Linux Rolling",
		},
		{
			Platform: &Platform{
				Title:   "Red Hat Enterprise Linux",
				Runtime: providers.RUNTIME_AWS_EC2,
				Version: "7",
				Family:  []string{"linux", "unix", "os"},
			},
			Expected: "Red Hat Enterprise Linux, AWS EC2 Instance",
		},
		{
			Platform: &Platform{
				Title:   "Red Hat Enterprise Linux",
				Kind:    providers.Kind_KIND_API,
				Version: "7",
				Family:  []string{"linux", "unix"},
			},
			Expected: "Red Hat Enterprise Linux",
		},
		{
			Platform: &Platform{
				Title:   "Red Hat Enterprise Linux",
				Kind:    providers.Kind_KIND_BARE_METAL,
				Version: "7",
				Family:  []string{"linux", "unix", "os"},
			},
			Expected: "Red Hat Enterprise Linux, bare metal",
		},
		{
			Platform: &Platform{
				Title:   "Red Hat Enterprise Linux 8",
				Kind:    providers.Kind_KIND_CONTAINER_IMAGE,
				Version: "8",
				Family:  []string{"linux", "unix", "os"},
			},
			Expected: "Red Hat Enterprise Linux 8, Container Image",
		},
		{
			Platform: &Platform{
				Title:   "Amazon Web Services 8",
				Runtime: providers.RUNTIME_AWS,
				Kind:    providers.Kind_KIND_API,
				Version: "8",
			},
			Expected: "Amazon Web Services 8",
		},
		{
			Platform: &Platform{
				Title:   "Test Deployment",
				Runtime: providers.RUNTIME_KUBERNETES_CLUSTER,
				Family:  []string{"k8s-workload", "k8s"},
			},
			Expected: "Test Deployment, Kubernetes Cluster",
		},
		{
			Platform: &Platform{
				Title:   "Test Deployment",
				Runtime: providers.RUNTIME_KUBERNETES_MANIFEST,
				Family:  []string{"k8s-workload", "k8s"},
			},
			Expected: "Test Deployment, Kubernetes Manifest File",
		},
		{
			Platform: &Platform{
				Title:   "k8s test cluster",
				Runtime: providers.RUNTIME_KUBERNETES,
				Family:  []string{"k8s"},
			},
			Expected: "k8s test cluster",
		},
	}

	for i := range test {
		assert.Equal(t, test[i].Expected, test[i].Platform.PrettyTitle(), i)
	}
}
