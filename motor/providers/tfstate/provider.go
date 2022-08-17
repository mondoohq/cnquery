package tfstate

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"

	"go.mondoo.io/mondoo/motor/providers"
)

var (
	_ providers.Transport                   = (*Provider)(nil)
	_ providers.TransportPlatformIdentifier = (*Provider)(nil)
)

func New(pCfg *providers.Config) (*Provider, error) {
	path := pCfg.Options["path"]
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var state State
	err = json.Unmarshal(data, &state)
	if err != nil {
		return nil, err
	}

	absPath, _ := filepath.Abs(path)
	h := sha256.New()
	h.Write([]byte(absPath))
	hash := hex.EncodeToString(h.Sum(nil))

	platformID := "//platformid.api.mondoo.app/runtime/tfstate/hash/" + hash

	return &Provider{
		platformID: platformID,
		path:       path,
		state:      &state,
	}, nil
}

type Provider struct {
	platformID string
	path       string
	state      *State
}

func (p *Provider) Close() {}

func (p *Provider) Capabilities() providers.Capabilities {
	return providers.Capabilities{}
}

func (p *Provider) Kind() providers.Kind {
	return providers.Kind_KIND_CODE
}

func (p *Provider) Runtime() string {
	return ""
}

func (p *Provider) Identifier() (string, error) {
	return p.platformID, nil
}

func (p *Provider) PlatformIdDetectors() []providers.PlatformIdDetector {
	return []providers.PlatformIdDetector{
		providers.TransportPlatformIdentifierDetector,
	}
}

func (p *Provider) State() (*State, error) {
	return p.state, nil
}
