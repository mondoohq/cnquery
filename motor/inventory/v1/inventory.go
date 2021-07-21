package v1

import (
	"io/ioutil"
	"path/filepath"

	"github.com/cockroachdb/errors"

	"github.com/segmentio/ksuid"
	"go.mondoo.io/mondoo/motor/transports"
	"google.golang.org/protobuf/proto"
	"sigs.k8s.io/yaml"
)

//go:generate protoc --proto_path=$PWD:. --go_out=. --go_opt=paths=source_relative --falcon_out=. --iam-actions_out=. inventory.proto

const (
	InventoryFilePath = "mondoo.app/source-file"
)

// InventoryFromYAML create an inventory from yaml contents
func InventoryFromYAML(data []byte) (*Inventory, error) {
	var res Inventory
	err := yaml.Unmarshal(data, &res)
	return &res, err
}

// InventoryFromFile loads an inventory from file system
func InventoryFromFile(path string) (*Inventory, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}

	inventoryData, err := ioutil.ReadFile(absPath)
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
	if p.Spec.Credentials == nil {
		p.Spec.Credentials = map[string]*transports.Credential{}
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
				if cred.SecretId != "" {
					// clean credentials
					// if a secret id with content is provided, we discard the content and always prefer the secret id
					cleanCred(cred)
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

		// load private key pem into secret
		if cred.PrivateKey != "" {
			cred.Secret = []byte(cred.PrivateKey)
		}

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

			data, err := ioutil.ReadFile(path)
			if err != nil {
				return errors.New("cannot read credential: " + path)
			}
			cred.Secret = data
		}
	}
	return nil
}

func cleanCred(c *transports.Credential) {
	c.User = ""
	c.Secret = []byte{}
	c.Type = transports.CredentialType_undefined
	c.PrivateKey = ""
	c.PrivateKeyPath = ""
	c.Password = ""
}

func cloneCred(c *transports.Credential) *transports.Credential {
	m := proto.Clone(c)
	return m.(*transports.Credential)
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

// isValidCredentialRef ensures an asset credential is defined properly
// The implementation assumes the credentials have been offloaded to the
// credential map before via PreProcess
func isValidCredentialRef(cred *transports.Credential) error {
	if cred.SecretId == "" {
		return errors.New("credential is missing the secret_id")
	}

	// credential references have no type defined
	if cred.Type != transports.CredentialType_undefined {
		return errors.New("credential reference has a wrong type defined")
	}

	return nil
}
