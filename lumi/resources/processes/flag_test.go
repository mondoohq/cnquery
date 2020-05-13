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
	// TODO: we need to parse this in the future
	assert.Equal(t, map[string]string{}, fs.actual)
}
