package k8s

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/suite"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ManifestParserSuite struct {
	suite.Suite
	manifestParser manifestParser
}

func (s *ManifestParserSuite) SetupSuite() {
	manifest, err := loadManifestFile("./resources/testdata/mixed.yaml")
	s.Require().NoError(err)
	manP, err := newManifestParser(manifest, "", "")
	s.Require().NoError(err)

	s.manifestParser = manP
}

func (s *ManifestParserSuite) TestNamespace() {
	ns, err := s.manifestParser.Namespace("default")
	s.Require().NoError(err)
	s.Equal("default", ns.Name)
	s.Equal("Namespace", ns.Kind)
}

func (s *ManifestParserSuite) TestNamespaces() {
	nss, err := s.manifestParser.Namespaces()
	s.Require().NoError(err)
	s.Len(nss, 2)

	nsNames := make([]string, 0, len(nss))
	for _, ns := range nss {
		nsNames = append(nsNames, ns.Name)
		s.Equal("Namespace", ns.Kind)
	}
	s.ElementsMatch([]string{"default", "custom"}, nsNames)
}

func (s *ManifestParserSuite) TestPod() {
	pod, err := s.manifestParser.Pod("default", "pod")
	s.Require().NoError(err)

	s.Equal("default", pod.Namespace)
	s.Equal("pod", pod.Name)
	s.Equal("Pod", pod.Kind)
}

func (s *ManifestParserSuite) TestPod_NotFound() {
	name := "notexist"
	_, err := s.manifestParser.Pod("default", name)
	s.Require().Error(err)
	s.Equal(fmt.Sprintf("pod %s not found", name), err.Error())
}

func (s *ManifestParserSuite) TestPods() {
	pods, err := s.manifestParser.Pods(v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "default"}})
	s.Require().NoError(err)

	s.Len(pods, 1)
	s.Equal("default", pods[0].Namespace)
	s.Equal("pod", pods[0].Name)
	s.Equal("Pod", pods[0].Kind)
}

func (s *ManifestParserSuite) TestDeployment() {
	dep, err := s.manifestParser.Deployment("default", "deployment")
	s.Require().NoError(err)

	s.Equal("default", dep.Namespace)
	s.Equal("deployment", dep.Name)
	s.Equal("Deployment", dep.Kind)
}

func (s *ManifestParserSuite) TestDeployment_NotFound() {
	name := "notexist"
	_, err := s.manifestParser.Deployment("default", name)
	s.Require().Error(err)
	s.Equal(fmt.Sprintf("deployment %s not found", name), err.Error())
}

func (s *ManifestParserSuite) TestDeployments() {
	deps, err := s.manifestParser.Deployments(v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "default"}})
	s.Require().NoError(err)

	s.Len(deps, 1)
	s.Equal("default", deps[0].Namespace)
	s.Equal("deployment", deps[0].Name)
	s.Equal("Deployment", deps[0].Kind)
}

func (s *ManifestParserSuite) TestCronJob() {
	dep, err := s.manifestParser.CronJob("default", "cronjob")
	s.Require().NoError(err)

	s.Equal("default", dep.Namespace)
	s.Equal("cronjob", dep.Name)
	s.Equal("CronJob", dep.Kind)
}

func (s *ManifestParserSuite) TestCronJob_NotFound() {
	name := "notexist"
	_, err := s.manifestParser.CronJob("default", name)
	s.Require().Error(err)
	s.Equal(fmt.Sprintf("cronjob %s not found", name), err.Error())
}

func (s *ManifestParserSuite) TestCronJobs() {
	deps, err := s.manifestParser.Deployments(v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "default"}})
	s.Require().NoError(err)

	s.Len(deps, 1)
	s.Equal("default", deps[0].Namespace)
	s.Equal("deployment", deps[0].Name)
	s.Equal("Deployment", deps[0].Kind)
}

func (s *ManifestParserSuite) TestStatefulSet() {
	dep, err := s.manifestParser.StatefulSet("default", "statefulset")
	s.Require().NoError(err)

	s.Equal("default", dep.Namespace)
	s.Equal("statefulset", dep.Name)
	s.Equal("StatefulSet", dep.Kind)
}

func (s *ManifestParserSuite) TestStatefulSet_NotFound() {
	name := "notexist"
	_, err := s.manifestParser.StatefulSet("default", name)
	s.Require().Error(err)
	s.Equal(fmt.Sprintf("statefulset %s not found", name), err.Error())
}

func (s *ManifestParserSuite) TestStatefulSets() {
	sss, err := s.manifestParser.StatefulSets(v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "default"}})
	s.Require().NoError(err)

	s.Len(sss, 1)
	s.Equal("default", sss[0].Namespace)
	s.Equal("statefulset", sss[0].Name)
	s.Equal("StatefulSet", sss[0].Kind)
}

func (s *ManifestParserSuite) TestJob() {
	job, err := s.manifestParser.Job("default", "job")
	s.Require().NoError(err)

	s.Equal("default", job.Namespace)
	s.Equal("job", job.Name)
	s.Equal("Job", job.Kind)
}

func (s *ManifestParserSuite) TestJob_NotFound() {
	name := "notexist"
	_, err := s.manifestParser.Job("default", name)
	s.Require().Error(err)
	s.Equal(fmt.Sprintf("job %s not found", name), err.Error())
}

func (s *ManifestParserSuite) TestJobs() {
	jobs, err := s.manifestParser.Jobs(v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "default"}})
	s.Require().NoError(err)

	s.Len(jobs, 1)
	s.Equal("default", jobs[0].Namespace)
	s.Equal("job", jobs[0].Name)
	s.Equal("Job", jobs[0].Kind)
}

func (s *ManifestParserSuite) TestReplicaSet() {
	rs, err := s.manifestParser.ReplicaSet("default", "replicaset")
	s.Require().NoError(err)

	s.Equal("default", rs.Namespace)
	s.Equal("replicaset", rs.Name)
	s.Equal("ReplicaSet", rs.Kind)
}

func (s *ManifestParserSuite) TestReplicaSet_NotFound() {
	name := "notexist"
	_, err := s.manifestParser.ReplicaSet("default", name)
	s.Require().Error(err)
	s.Equal(fmt.Sprintf("replicaset %s not found", name), err.Error())
}

func (s *ManifestParserSuite) TestReplicaSets() {
	rss, err := s.manifestParser.ReplicaSets(v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "default"}})
	s.Require().NoError(err)

	s.Len(rss, 1)
	s.Equal("default", rss[0].Namespace)
	s.Equal("replicaset", rss[0].Name)
	s.Equal("ReplicaSet", rss[0].Kind)
}

func (s *ManifestParserSuite) TestDaemonSet() {
	ds, err := s.manifestParser.DaemonSet("custom", "daemonset")
	s.Require().NoError(err)

	s.Equal("custom", ds.Namespace)
	s.Equal("daemonset", ds.Name)
	s.Equal("DaemonSet", ds.Kind)
}

func (s *ManifestParserSuite) TestDaemonSet_NotFound() {
	name := "notexist"
	_, err := s.manifestParser.DaemonSet("default", name)
	s.Require().Error(err)
	s.Equal(fmt.Sprintf("daemonset %s not found", name), err.Error())
}

func (s *ManifestParserSuite) TestDaemonSets() {
	dss, err := s.manifestParser.DaemonSets(v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "custom"}})
	s.Require().NoError(err)

	s.Len(dss, 1)
	s.Equal("custom", dss[0].Namespace)
	s.Equal("daemonset", dss[0].Name)
	s.Equal("DaemonSet", dss[0].Kind)
}

func (s *ManifestParserSuite) TestIngress() {
	i, err := s.manifestParser.Ingress("default", "ingress")
	s.Require().NoError(err)

	s.Equal("default", i.Namespace)
	s.Equal("ingress", i.Name)
	s.Equal("Ingress", i.Kind)
}

func (s *ManifestParserSuite) TestIngress_NotFound() {
	name := "notexist"
	_, err := s.manifestParser.Ingress("default", name)
	s.Require().Error(err)
	s.Equal(fmt.Sprintf("ingress %s not found", name), err.Error())
}

func (s *ManifestParserSuite) TestIngresses() {
	is, err := s.manifestParser.Ingresses(v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "default"}})
	s.Require().NoError(err)

	s.Len(is, 1)
	s.Equal("default", is[0].Namespace)
	s.Equal("ingress", is[0].Name)
	s.Equal("Ingress", is[0].Kind)
}

func (s *ManifestParserSuite) TestSecret() {
	i, err := s.manifestParser.Secret("default", "secret")
	s.Require().NoError(err)

	s.Equal("default", i.Namespace)
	s.Equal("secret", i.Name)
	s.Equal("Secret", i.Kind)
}

func (s *ManifestParserSuite) TestSecret_NotFound() {
	name := "notexist"
	_, err := s.manifestParser.Secret("default", name)
	s.Require().Error(err)
	s.Equal(fmt.Sprintf("secret %s not found", name), err.Error())
}

func TestManifestParserSuite(t *testing.T) {
	suite.Run(t, new(ManifestParserSuite))
}
