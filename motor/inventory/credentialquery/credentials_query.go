package credentialquery

import (
	"strings"

	"go.mondoo.io/mondoo/motor/vault"

	"github.com/rs/zerolog/log"

	"github.com/cockroachdb/errors"
	"github.com/mitchellh/mapstructure"
	"go.mondoo.io/mondoo/llx"
	"go.mondoo.io/mondoo/motor/asset"
	"go.mondoo.io/mondoo/policy/executor"
	"go.mondoo.io/mondoo/types"
)

type CredentialQueryResponse struct {
	// maps to credentials
	Type     string `json:"type,omitempty"`
	User     string `json:"user,omitempty"`      // user associated with the secret
	SecretId string `json:"secret_id,omitempty"` // id to use to fetch the secret from the source vault
}

func NewCredentialQueryRunner(credentialQuery string) (*CredentialQueryRunner, error) {
	e, err := executor.NewEmbeddedExecutor()
	if err != nil {
		return nil, err
	}

	// just empty props to ensure we can compile
	props := map[string]*llx.Primitive{
		"mrn":      llx.StringPrimitive(""),
		"name":     llx.StringPrimitive(""),
		"labels":   llx.MapData(map[string]interface{}{}, types.String).Result().Data,
		"platform": llx.MapData(map[string]interface{}{}, types.String).Result().Data,
	}
	_, err = e.Compile(credentialQuery, props)
	if err != nil {
		return nil, errors.Wrap(err, "could not compile the secret metadata function")
	}
	return &CredentialQueryRunner{
		e:                   e,
		secretMetadataQuery: credentialQuery,
	}, nil
}

type CredentialQueryRunner struct {
	e                   *executor.EmbeddedExecutor
	secretMetadataQuery string
}

func (sq *CredentialQueryRunner) Run(a *asset.Asset) (*vault.Credential, error) {
	// map labels to props
	labelProps := map[string]interface{}{}
	labels := a.GetLabels()
	for k, v := range labels {
		labelProps[k] = v
	}

	// map platform to props
	var platformProps map[string]interface{}
	if a.Platform != nil {
		platformProps = map[string]interface{}{
			"name":    a.Platform.Name,
			"release": a.Platform.Release,
			"arch":    a.Platform.Arch,
		}
	} else {
		platformProps = map[string]interface{}{}
	}

	props := map[string]*llx.Primitive{
		"mrn":      llx.StringPrimitive(a.Mrn),
		"name":     llx.StringPrimitive(a.Name),
		"labels":   llx.MapData(labelProps, types.String).Result().Data,
		"platform": llx.MapData(platformProps, types.String).Result().Data,
	}

	value, err := sq.e.Run(sq.secretMetadataQuery, props)
	if err != nil {
		return nil, err
	}

	sMeta := &CredentialQueryResponse{}
	decoder, _ := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		Metadata: nil,
		Result:   sMeta,
		TagName:  "json",
	})
	err = decoder.Decode(value)

	code, ok := vault.CredentialType_value[strings.TrimSpace(sMeta.Type)]
	if !ok {
		log.Warn().Str("credential_type", sMeta.Type).Msg("unknown credential type used in credential query")
	}

	creds := &vault.Credential{
		Type:     vault.CredentialType(code),
		User:     sMeta.User,
		SecretId: sMeta.SecretId,
	}

	return creds, err
}
