// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package inventory

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/cockroachdb/errors"
	"github.com/segmentio/ksuid"
	"go.mondoo.com/cnquery/providers-sdk/v1/vault"
	"google.golang.org/protobuf/proto"
	"sigs.k8s.io/yaml"
)

//go:generate protoc --proto_path=../../../:. --go_out=. --go_opt=paths=source_relative --rangerrpc_out=. inventory.proto

const (
	InventoryFilePath = "mondoo.app/source-file"
)

var ErrProviderTypeDoesNotMatch = errors.New("provider type does not match")

type Option func(*Inventory)

// passes a list of asset into the Inventory Manager
func WithAssets(assetList ...*Asset) Option {
	return func(inventory *Inventory) {
		inventory.AddAssets(assetList...)
	}
}

func New(opts ...Option) *Inventory {
	inventory := &Inventory{
		Metadata: &ObjectMeta{},
		Spec:     &InventorySpec{},
	}

	for _, option := range opts {
		option(inventory)
	}

	return inventory
}

// InventoryFromYAML create an inventory from yaml contents
func InventoryFromYAML(data []byte) (*Inventory, error) {
	res := New()
	err := yaml.Unmarshal(data, res)

	// FIXME: DEPRECATED, remove in v10.0 (or later) vv
	// This is only used to migrate the old "backend" field.
	if err == nil && res.Spec != nil {
		for _, asset := range res.Spec.Assets {
			for _, conn := range asset.Connections {
				if conn.Backend != 0 && conn.Type == "" {
					conn.Type = connBackendToType(conn.Backend)
				}
			}
		}
	}
	// ^^

	return res, err
}

func connBackendToType(backend int32) string {
	// ProviderType_LOCAL_OS                      ProviderType = 0
	// ProviderType_DOCKER_ENGINE_IMAGE           ProviderType = 1
	// ProviderType_DOCKER_ENGINE_CONTAINER       ProviderType = 2
	// ProviderType_SSH                           ProviderType = 3
	// ProviderType_WINRM                         ProviderType = 4
	// ProviderType_AWS_SSM_RUN_COMMAND           ProviderType = 5
	// ProviderType_CONTAINER_REGISTRY            ProviderType = 6
	// ProviderType_TAR                           ProviderType = 7
	// ProviderType_MOCK                          ProviderType = 8
	// ProviderType_VSPHERE                       ProviderType = 9
	// ProviderType_ARISTAEOS                     ProviderType = 10
	// ProviderType_AWS                           ProviderType = 12
	// ProviderType_GCP                           ProviderType = 13
	// ProviderType_AZURE                         ProviderType = 14
	// ProviderType_MS365                         ProviderType = 15
	// ProviderType_IPMI                          ProviderType = 16
	// ProviderType_VSPHERE_VM                    ProviderType = 17
	// ProviderType_FS                            ProviderType = 18
	// ProviderType_K8S                           ProviderType = 19
	// ProviderType_EQUINIX_METAL                 ProviderType = 20
	// ProviderType_DOCKER                        ProviderType = 21 // unspecified if this is a container or image
	// ProviderType_GITHUB                        ProviderType = 22
	// ProviderType_VAGRANT                       ProviderType = 23
	// ProviderType_AWS_EC2_EBS                   ProviderType = 24
	// ProviderType_GITLAB                        ProviderType = 25
	// ProviderType_TERRAFORM                     ProviderType = 26
	// ProviderType_HOST                          ProviderType = 27
	// ProviderType_UNKNOWN                       ProviderType = 28
	// ProviderType_OKTA                          ProviderType = 29
	// ProviderType_GOOGLE_WORKSPACE              ProviderType = 30
	// ProviderType_SLACK                         ProviderType = 31
	// ProviderType_VCD                           ProviderType = 32
	// ProviderType_OCI                           ProviderType = 33
	// ProviderType_OPCUA                         ProviderType = 34
	// ProviderType_GCP_COMPUTE_INSTANCE_SNAPSHOT ProviderType = 35
	switch backend {
	case 0:
		return "os"
	case 1:
		return "docker-image"
	case 2:
		return "docker-container"
	case 3:
		return "ssh"
	case 4:
		return "winrm"
	case 5:
		return "aws-ssm-run-command"
	case 6:
		return "container-registry"
	case 7:
		return "tar"
	case 8:
		return "mock"
	case 9:
		return "vsphere"
	case 10:
		return "arista-eos"
	case 12:
		return "aws"
	case 13:
		return "gcp"
	case 14:
		return "azure"
	case 15:
		return "ms365"
	case 16:
		return "ipmi"
	case 17:
		return "vsphere-vm"
	case 18:
		return "fs"
	case 19:
		return "k8s"
	case 20:
		return "equinix-metal"
	case 21:
		return "docker"
	case 22:
		return "github"
	case 23:
		return "vagrant"
	case 24:
		return "aws-ec2-ebs"
	case 25:
		return "gitlab"
	case 26:
		return "terraform"
	case 27:
		return "host"
	case 28:
		return "unknown"
	case 29:
		return "okta"
	case 30:
		return "google-workspace"
	case 31:
		return "slack"
	case 32:
		return "vcd"
	case 33:
		return "oci"
	case 34:
		return "opcua"
	case 35:
		return "gcp-compute-instance-snapshot"
	default:
		return ""
	}
}

// InventoryFromFile loads an inventory from file system
func InventoryFromFile(path string) (*Inventory, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}

	inventoryData, err := os.ReadFile(absPath)
	if err != nil {
		return nil, err
	}

	inventory, err := InventoryFromYAML(inventoryData)
	if err != nil {
		return nil, err
	}

	inventory.ensureRequireMetadataStructs()
	inventory.Metadata.Labels[InventoryFilePath] = absPath

	return inventory, nil
}

func (p *Inventory) ensureRequireMetadataStructs() {
	if p.Metadata == nil {
		p.Metadata = &ObjectMeta{}
	}

	if p.Metadata.Labels == nil {
		p.Metadata.Labels = map[string]string{}
	}
}

// ToYAML returns the inventory as yaml
func (p *Inventory) ToYAML() ([]byte, error) {
	return yaml.Marshal(p)
}

// PreProcess extracts all the embedded credentials from the assets and migrates those to in the
// dedicated credentials section. The pre-processed content is optimized for runtime access.
// Re-generating yaml, results into a different yaml output. While the results are identical,
// the yaml file is not.
func (p *Inventory) PreProcess() error {
	if p.Spec == nil {
		p.Spec = &InventorySpec{}
	}

	if p.Spec.Credentials == nil {
		p.Spec.Credentials = map[string]*vault.Credential{}
	}

	// we are going to use the labels in metadata, ensure the structs are in place
	p.ensureRequireMetadataStructs()

	// extract embedded credentials from assets into dedicated section
	for i := range p.Spec.Assets {
		asset := p.Spec.Assets[i]

		for j := range asset.Connections {
			c := asset.Connections[j]
			for k := range c.Credentials {
				cred := c.Credentials[k]
				if cred != nil && cred.SecretId != "" {
					// clean credentials
					// if a secret id with content is provided, we discard the content and always prefer the secret id
					cleanSecrets(cred)
				} else {
					// create secret id and add id to the credential
					secretId := ksuid.New().String()
					cred.SecretId = secretId
					// add a cloned credential to the map
					copy := cloneCred(cred)
					p.Spec.Credentials[secretId] = copy

					// replace current credential the secret id, essentially we just remove all the content
					cleanCred(cred)
				}
			}
		}
	}

	// iterate over all credentials and load private keys references
	for k := range p.Spec.Credentials {
		cred := p.Spec.Credentials[k]

		// ensure the secret id is correct
		cred.SecretId = k
		cred.PreProcess()

		// TODO: we may want to load it but we probably need
		// a local file watcher to detect changes
		if cred.PrivateKeyPath != "" {
			path := cred.PrivateKeyPath

			// special handling for relative filenames, instead of loading
			// private keys from relative to the work directory, we want to
			// load the files relative to the source inventory
			if !filepath.IsAbs(cred.PrivateKeyPath) {
				// we handle credentials relative to the inventory file
				fileLoc, ok := p.Metadata.Labels[InventoryFilePath]
				if ok {
					path = filepath.Join(filepath.Dir(fileLoc), path)
				} else {
					absPath, err := filepath.Abs(path)
					if err != nil {
						return err
					}
					path = absPath
				}
			}

			data, err := os.ReadFile(path)
			if err != nil {
				return errors.New("cannot read credential: " + path)
			}
			cred.Secret = data

			// only set the credential type if it is not set, pkcs12 also uses the private key path
			if cred.Type == vault.CredentialType_undefined {
				cred.Type = vault.CredentialType_private_key
			}
		}
	}
	return nil
}

func (p *Inventory) MarkConnectionsInsecure() {
	for i := range p.Spec.Assets {
		asset := p.Spec.Assets[i]
		for j := range asset.Connections {
			asset.Connections[j].Insecure = true
		}
	}
}

func cleanCred(c *vault.Credential) {
	c.User = ""
	c.Type = vault.CredentialType_undefined
	cleanSecrets(c)
}

func cleanSecrets(c *vault.Credential) {
	c.Secret = []byte{}
	c.PrivateKey = ""
	c.PrivateKeyPath = ""
	c.Password = ""
}

func cloneCred(c *vault.Credential) *vault.Credential {
	m := proto.Clone(c)
	return m.(*vault.Credential)
}

// Validate ensures consistency within the inventory.
// The implementation expects that PreProcess was executed before.
// - it checks that all secret ids are either part of the credential map or a vault is defined
// - it checks that all credentials have a secret id
func (p *Inventory) Validate() error {
	var err error
	for i := range p.Spec.Assets {
		a := p.Spec.Assets[i]
		for j := range a.Connections {
			conn := a.Connections[j]
			for k := range conn.Credentials {
				cred := conn.Credentials[k]
				err = isValidCredentialRef(cred)
				if err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func (p *Inventory) AddAssets(assetList ...*Asset) {
	if p.Spec == nil {
		p.Spec = &InventorySpec{}
	}
	for i := range assetList {
		p.Spec.Assets = append(p.Spec.Assets, assetList[i])
	}
}

func (p *Inventory) ApplyLabels(labels map[string]string) {
	for i := range p.Spec.Assets {
		a := p.Spec.Assets[i]

		if a.Labels == nil {
			a.Labels = map[string]string{}
		}

		for k := range labels {
			a.Labels[k] = labels[k]
		}
	}
}

func (p *Inventory) ApplyCategory(category AssetCategory) {
	for i := range p.Spec.Assets {
		a := p.Spec.Assets[i]
		a.Category = category
	}
}

// isValidCredentialRef ensures an asset credential is defined properly
// The implementation assumes the credentials have been offloaded to the
// credential map before via PreProcess
func isValidCredentialRef(cred *vault.Credential) error {
	if cred.SecretId == "" {
		return errors.New("credential is missing the secret_id")
	}

	// credential references have no type defined
	if cred.Type != vault.CredentialType_undefined {
		return errors.New("credential reference has a wrong type defined")
	}

	return nil
}

// often used family names
var (
	FAMILY_UNIX    = "unix"
	FAMILY_DARWIN  = "darwin"
	FAMILY_LINUX   = "linux"
	FAMILY_BSD     = "bsd"
	FAMILY_WINDOWS = "windows"
)

func (p *Platform) IsFamily(family string) bool {
	for i := range p.Family {
		if p.Family[i] == family {
			return true
		}
	}
	return false
}

func (p *Platform) PrettyTitle() string {
	prettyTitle := p.Title

	// extend the title only for OS and k8s objects
	if !(p.IsFamily("k8s-workload") || p.IsFamily("os")) {
		return prettyTitle
	}

	var runtimeNiceName string
	runtimeName := p.Runtime
	if runtimeName != "" {
		switch runtimeName {
		case "aws-ec2-instance":
			runtimeNiceName = "AWS EC2 Instance"
		case "azure-vm":
			runtimeNiceName = "Azure Virtual Machine"
		case "docker-container":
			runtimeNiceName = "Docker Container"
		case "docker-image":
			runtimeNiceName = "Docker Image"
		case "gcp-vm":
			runtimeNiceName = "GCP Virtual Machine"
		case "k8s-cluster":
			runtimeNiceName = "Kubernetes Cluster"
		case "k8s-manifest":
			runtimeNiceName = "Kubernetes Manifest File"
		case "vsphere-host":
			runtimeNiceName = "vSphere Host"
		case "vsphere-vm":
			runtimeNiceName = "vSphere Virtual Machine"
		}
	} else {
		runtimeKind := p.Kind
		switch runtimeKind {
		case "baremetal":
			runtimeNiceName = "bare metal"
		case "container":
			runtimeNiceName = "Container"
		case "container-image":
			runtimeNiceName = "Container Image"
		case "virtualmachine":
			runtimeNiceName = "Virtual Machine"
		case "virtualmachine-image":
			runtimeNiceName = "Virtual Machine Image"
		}
	}
	// e.g. ", Kubernetes Cluster" and also "Kubernetes, Kubernetes Cluster" do not look nice, so prevent them
	if prettyTitle == "" || strings.Contains(runtimeNiceName, prettyTitle) {
		return runtimeNiceName
	}

	// do not add runtime name when the title is already obvious, e.g. "Network API, Network"
	if !strings.Contains(prettyTitle, runtimeNiceName) {
		prettyTitle += ", " + runtimeNiceName
	}

	return prettyTitle
}

type cloneSettings struct {
	noDiscovery bool
}

type CloneOption interface {
	Apply(*cloneSettings)
}

// WithoutDiscovery removes the discovery flags in the opts to ensure the same discovery does not run again
func WithoutDiscovery() CloneOption {
	return withoutDiscovery{}
}

type withoutDiscovery struct{}

func (w withoutDiscovery) Apply(o *cloneSettings) { o.noDiscovery = true }

func (cfg *Config) Clone(opts ...CloneOption) *Config {
	if cfg == nil {
		return nil
	}

	cloneSettings := &cloneSettings{}
	for _, option := range opts {
		option.Apply(cloneSettings)
	}

	clonedObject := proto.Clone(cfg).(*Config)

	if cloneSettings.noDiscovery {
		clonedObject.Discover = &Discovery{}
	}

	return clonedObject
}

func (c *Config) ToUrl() string {
	schema := c.Type
	if _, ok := c.Options["tls"]; ok {
		schema = "tls"
	}

	host := c.Host
	if strings.HasPrefix(host, "sha256:") {
		host = strings.Replace(host, "sha256:", "", -1)
	}

	path := c.Path
	if path != "" {
		if path[0] != '/' {
			path = "/" + path
		}
	}

	return schema + "://" + host + path
}
