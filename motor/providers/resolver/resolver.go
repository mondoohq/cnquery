package resolver

import (
	"context"
	"fmt"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/motor"
	v1 "go.mondoo.com/cnquery/motor/inventory/v1"
	"go.mondoo.com/cnquery/motor/vault"
	"google.golang.org/protobuf/proto"
)

var providerDevelopmentStatus = map[string]string{
	"aws-ec2-ebs": "experimental",
}

func warnIncompleteFeature(backend string) {
	if providerDevelopmentStatus[backend] != "" {
		log.Warn().Str("feature", backend).Str("status", providerDevelopmentStatus[backend]).Msg("WARNING: you are using an early access feature")
	}
}

// NewMotorConnection establishes a motor connection by using the provided provider configuration
// By default, it uses the id detector mechanisms provided by the provider. User can overwrite that
// behaviour by optionally passing id detector identifier
func NewMotorConnection(ctx context.Context, tc *v1.Config, credsResolver vault.Resolver) (*motor.Motor, error) {
	log.Debug().Msg("establish motor connection")
	var m *motor.Motor

	warnIncompleteFeature(tc.Type)

	// we clone the config here, and replace all credential references with the real references
	// the clone is important so that credentials are not leaked outside of the function
	resolvedConfig := proto.Clone(tc).(*v1.Config)
	// cloning a proto object with an empty map will result in the copied map being nil. make sure to initialize it
	// to not break providers that check for nil.
	if resolvedConfig.Options == nil {
		resolvedConfig.Options = map[string]string{}
	}
	resolvedCredentials := []*vault.Credential{}
	for i := range resolvedConfig.Credentials {
		credential := resolvedConfig.Credentials[i]
		if credential.SecretId != "" && credsResolver != nil {
			resolvedCredential, err := credsResolver.GetCredential(credential)
			if err != nil {
				log.Debug().Str("secret-id", credential.SecretId).Err(err).Msg("could not fetch secret for motor connection")
				return nil, err
			}
			credential = resolvedCredential
		}
		resolvedCredentials = append(resolvedCredentials, credential)
	}
	resolvedConfig.Credentials = resolvedCredentials

	// establish connection
	switch resolvedConfig.Type {
	default:
		return nil, fmt.Errorf("connection> unsupported backend '%s'", resolvedConfig.Type)
	}

	return m, nil
}
