// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package inventory

import (
	"os"
	"os/user"
	"path/filepath"
	"strings"

	"github.com/cockroachdb/errors"
	"github.com/rs/zerolog/log"
	"github.com/segmentio/ksuid"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/vault"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/vault/config"
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
				if conn.Type == "" {
					log.Warn().Msg("no connection `type` provided in inventory, falling back to deprecated `backend` field")
					conn.Type = ConnBackendToType(conn.Backend)
				}
			}
		}
	}
	// ^^

	return res, err
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

func (p *Inventory) GetVault() (vault.Vault, error) {
	// instantiate with full vault config
	v, err := config.New(p.Spec.Vault)
	if err != nil {
		return nil, err
	}
	return v, nil
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

			if strings.HasPrefix(path, "~/") {
				// special handling for ~
				usr, err := user.Current()
				if err != nil {
					return err
				}
				path = filepath.Join(usr.HomeDir, path[2:])
			} else if !filepath.IsAbs(cred.PrivateKeyPath) {
				// special handling for relative filenames, instead of loading
				// private keys from relative to the work directory, we want to
				// load the files relative to the source inventory

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
	noDiscovery        bool
	parentConnectionId *uint32
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

// WithoutDiscovery removes the discovery flags in the opts to ensure the same discovery does not run again
func WithParentConnectionId(parentId uint32) CloneOption {
	return withParentConnectionId{parentId: parentId}
}

type withParentConnectionId struct {
	parentId uint32
}

func (w withParentConnectionId) Apply(o *cloneSettings) { o.parentConnectionId = &w.parentId }

func (cfg *Config) Clone(opts ...CloneOption) *Config {
	if cfg == nil {
		return nil
	}

	cloneSettings := &cloneSettings{}
	for _, option := range opts {
		option.Apply(cloneSettings)
	}

	clonedObject := proto.Clone(cfg).(*Config)
	clonedObject.Id = 0
	if cloneSettings.noDiscovery {
		clonedObject.Discover = &Discovery{}
	}
	if cloneSettings.parentConnectionId != nil {
		clonedObject.ParentConnectionId = *cloneSettings.parentConnectionId
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
