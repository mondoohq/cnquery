package processes

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseFlags(t *testing.T) {
	cmd := "etcd --advertise-client-urls=https://192.168.99.101:2379 --cert-file=/var/lib/minikube/certs/etcd/server.crt --client-cert-auth=true --data-dir=/var/lib/minikube/etcd --initial-advertise-peer-urls=https://192.168.99.101:2380 --initial-cluster=m01=https://192.168.99.101:2380 --key-file=/var/lib/minikube/certs/etcd/server.key --listen-client-urls=https://127.0.0.1:2379,https://192.168.99.101:2379 --listen-metrics-urls=http://127.0.0.1:2381 --listen-peer-urls=https://192.168.99.101:2380 --name=m01 --peer-cert-file=/var/lib/minikube/certs/etcd/peer.crt --peer-client-cert-auth=true --peer-key-file=/var/lib/minikube/certs/etcd/peer.key --peer-trusted-ca-file=/var/lib/minikube/certs/etcd/ca.crt --snapshot-count=10000 --trusted-ca-file=/var/lib/minikube/certs/etcd/ca.crt"

	fs := FlagSet{}
	err := fs.ParseCommand(cmd)
	require.NoError(t, err)
	assert.Equal(t, map[string]string{
		"advertise-client-urls":       "https://192.168.99.101:2379",
		"cert-file":                   "/var/lib/minikube/certs/etcd/server.crt",
		"client-cert-auth":            "true",
		"data-dir":                    "/var/lib/minikube/etcd",
		"initial-advertise-peer-urls": "https://192.168.99.101:2380",
		"initial-cluster":             "m01=https://192.168.99.101:2380",
		"key-file":                    "/var/lib/minikube/certs/etcd/server.key",
		"listen-client-urls":          "https://127.0.0.1:2379,https://192.168.99.101:2379",
		"listen-metrics-urls":         "http://127.0.0.1:2381",
		"listen-peer-urls":            "https://192.168.99.101:2380",
		"name":                        "m01", "peer-cert-file": "/var/lib/minikube/certs/etcd/peer.crt",
		"peer-client-cert-auth": "true",
		"peer-key-file":         "/var/lib/minikube/certs/etcd/peer.key",
		"peer-trusted-ca-file":  "/var/lib/minikube/certs/etcd/ca.crt",
		"snapshot-count":        "10000",
		"trusted-ca-file":       "/var/lib/minikube/certs/etcd/ca.crt",
	}, fs.actual)

}

func TestParseShorthand(t *testing.T) {
	cmd := "go run apps/mondoo/mondoo.go shell -t ssh://docker@1.1.1.1 -i ~/.minikube/machines/minikube/id_rsa"
	fs := FlagSet{}
	err := fs.ParseCommand(cmd)
	require.NoError(t, err)
	// TODO: implement shorthand parser
	//assert.Equal(t, map[string]string{
	//	"t" : "ssh://docker@1.1.1.1",
	//	"i" : "~/.minikube/machines/minikube/id_rsa",
	//}, fs.actual)
}

type testSet struct {
	cmd string
	flags map[string]string
}

func TestFlagParser(t *testing.T) {
	tests := []testSet{
		{
			cmd: "/usr/local/bin/kube-proxy --config=/var/lib/kube-proxy/config.conf --hostname-override=minikube",
			flags: map[string]string{
				"config": "/var/lib/kube-proxy/config.conf",
				"hostname-override": "minikube",
			},
		},
		//{
		//	cmd: "/usr/bin/containerd-shim-runc-v2 -namespace moby -id 233b77262f559db23e3f663c21174dc346aaa025f39ef9643ba068fee0b87912 -address /var/run/docker/containerd/containerd.sock",
		//	flags: map[string]string{
		//		"namespace":"moby",
		//		"id": "233b77262f559db23e3f663c21174dc346aaa025f39ef9643ba068fee0b87912",
		//		"address": "/var/run/docker/containerd/containerd.sock",
		//	},
		//},
		{
			cmd: "kube-apiserver --advertise-address=192.168.99.103 --allow-privileged=true --authorization-mode=Node,RBAC --client-ca-file=/var/lib/minikube/certs/ca.crt --enable-admission-plugins=NamespaceLifecycle,LimitRanger,ServiceAccount,DefaultStorageClass,DefaultTolerationSeconds,NodeRestriction,MutatingAdmissionWebhook,ValidatingAdmissionWebhook,ResourceQuota --enable-bootstrap-token-auth=true --etcd-cafile=/var/lib/minikube/certs/etcd/ca.crt --etcd-certfile=/var/lib/minikube/certs/apiserver-etcd-client.crt --etcd-keyfile=/var/lib/minikube/certs/apiserver-etcd-client.key --etcd-servers=https://127.0.0.1:2379 --insecure-port=0 --kubelet-client-certificate=/var/lib/minikube/certs/apiserver-kubelet-client.crt --kubelet-client-key=/var/lib/minikube/certs/apiserver-kubelet-client.key --kubelet-preferred-address-types=InternalIP,ExternalIP,Hostname --proxy-client-cert-file=/var/lib/minikube/certs/front-proxy-client.crt --proxy-client-key-file=/var/lib/minikube/certs/front-proxy-client.key --requestheader-allowed-names=front-proxy-client --requestheader-client-ca-file=/var/lib/minikube/certs/front-proxy-ca.crt --requestheader-extra-headers-prefix=X-Remote-Extra- --requestheader-group-headers=X-Remote-Group --requestheader-username-headers=X-Remote-User --secure-port=8443 --service-account-issuer=https://kubernetes.default.svc.cluster.local --service-account-key-file=/var/lib/minikube/certs/sa.pub --service-account-signing-key-file=/var/lib/minikube/certs/sa.key --service-cluster-ip-range=10.96.0.0/12 --tls-cert-file=/var/lib/minikube/certs/apiserver.crt --tls-private-key-file=/var/lib/minikube/certs/apiserver.key",
			flags: map[string]string{
				"advertise-address": "192.168.99.103",
				"allow-privileged": "true",
				"authorization-mode": "Node,RBAC",
				"client-ca-file": "/var/lib/minikube/certs/ca.crt",
				"enable-admission-plugins": "NamespaceLifecycle,LimitRanger,ServiceAccount,DefaultStorageClass,DefaultTolerationSeconds,NodeRestriction,MutatingAdmissionWebhook,ValidatingAdmissionWebhook,ResourceQuota",
				"enable-bootstrap-token-auth": "true",
				"etcd-cafile": "/var/lib/minikube/certs/etcd/ca.crt",
				"etcd-certfile": "/var/lib/minikube/certs/apiserver-etcd-client.crt",
				"etcd-keyfile": "/var/lib/minikube/certs/apiserver-etcd-client.key",
				"etcd-servers": "https://127.0.0.1:2379",
				"insecure-port": "0",
				"kubelet-client-certificate": "/var/lib/minikube/certs/apiserver-kubelet-client.crt",
				"kubelet-client-key": "/var/lib/minikube/certs/apiserver-kubelet-client.key",
				"kubelet-preferred-address-types": "InternalIP,ExternalIP,Hostname",
				"proxy-client-cert-file": "/var/lib/minikube/certs/front-proxy-client.crt",
				"proxy-client-key-file": "/var/lib/minikube/certs/front-proxy-client.key",
				"requestheader-allowed-names": "front-proxy-client",
				"requestheader-client-ca-file": "/var/lib/minikube/certs/front-proxy-ca.crt",
				"requestheader-extra-headers-prefix": "X-Remote-Extra-",
				"requestheader-group-headers": "X-Remote-Group",
				"requestheader-username-headers": "X-Remote-User",
				"secure-port": "8443",
				"service-account-issuer": "https://kubernetes.default.svc.cluster.local",
				"service-account-key-file": "/var/lib/minikube/certs/sa.pub",
				"service-account-signing-key-file": "/var/lib/minikube/certs/sa.key",
				"service-cluster-ip-range": "10.96.0.0/12",
				"tls-cert-file": "/var/lib/minikube/certs/apiserver.crt",
				"tls-private-key-file": "/var/lib/minikube/certs/apiserver.key",
				},
		},
	}

	for i := range tests {
		test := tests[i]
		fs := FlagSet{}
		err := fs.ParseCommand(test.cmd)
		require.NoError(t, err)
		assert.Equal(t,test.flags,fs.actual, test.cmd)
	}
}