package credentialquery

import (
	"strings"

	"errors"
	"github.com/mitchellh/mapstructure"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery"
	"go.mondoo.com/cnquery/llx"
	"go.mondoo.com/cnquery/motor/asset"
	"go.mondoo.com/cnquery/motor/vault"
	"go.mondoo.com/cnquery/mql"
	"go.mondoo.com/cnquery/types"
)

type CredentialQueryResponse struct {
	// maps to credentials
	Type     string `json:"type,omitempty"`
	User     string `json:"user,omitempty"`      // user associated with the secret
	SecretId string `json:"secret_id,omitempty"` // id to use to fetch the secret from the source vault
}

func NewCredentialQueryRunner(credentialQuery string) (*CredentialQueryRunner, error) {
	rt, err := mql.MockRuntime()
	if err != nil {
		return nil, err
	}

	mqlExecutor := mql.New(rt, cnquery.DefaultFeatures)

	// just empty props to ensure we can compile
	props := map[string]*llx.Primitive{
		"mrn":      llx.StringPrimitive(""),
		"name":     llx.StringPrimitive(""),
		"labels":   llx.MapData(map[string]interface{}{}, types.String).Result().Data,
		"platform": llx.MapData(map[string]interface{}{}, types.String).Result().Data,
	}

	// test query to see if it compiles well
	_, err = mql.Exec(credentialQuery, rt, nil, props)
	if err != nil {
		return nil, errors.Join(err, errors.New("could not compile the secret metadata function"))
	}
	return &CredentialQueryRunner{
		mqlExecutor:         mqlExecutor,
		secretMetadataQuery: credentialQuery,
	}, nil
}

type CredentialQueryRunner struct {
	mqlExecutor         *mql.Executor
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
			"release": a.Platform.Version,
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

	value, err := sq.mqlExecutor.Exec(sq.secretMetadataQuery, props)
	if err != nil {
		return nil, err
	}

	sMeta := &CredentialQueryResponse{}
	decoder, _ := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		Metadata: nil,
		Result:   sMeta,
		TagName:  "json",
	})
	err = decoder.Decode(value.Value)

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
