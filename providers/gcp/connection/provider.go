// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package connection

import (
	"errors"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/motor/providers"
	"go.mondoo.com/cnquery/motor/vault"
)

var (
	_ providers.Instance           = (*Provider)(nil)
	_ providers.PlatformIdentifier = (*Provider)(nil)
)

type ResourceType int

const (
	Unknown ResourceType = iota
	Project
	Organization
	Workspace
	Folder
)

func New(pCfg *providers.Config) (*Provider, error) {
	var cred *vault.Credential
	if len(pCfg.Credentials) != 0 {
		cred = pCfg.Credentials[0]
	}
	if pCfg.Backend == providers.ProviderType_GCP {
		// FIXME: DEPRECATED, update in v8.0 vv
		// The options "project" and "organization" have been deprecated in favor of project-id and organization-id
		if pCfg.Options == nil || (pCfg.Options["project-id"] == "" && pCfg.Options["project"] == "" && pCfg.Options["organization-id"] == "" && pCfg.Options["organization"] == "" && pCfg.Options["folder-id"] == "") {
			// ^^
			return nil, errors.New("google provider requires a gcp organization id, gcp project id or google workspace customer id. please set option `project-id` or `organization-id` or `customer-id` or `folder-id`")
		}
	} else if pCfg.Backend == providers.ProviderType_GOOGLE_WORKSPACE {
		if pCfg.Options == nil || pCfg.Options["customer-id"] == "" {
			return nil, errors.New("google workspace provider requires an customer id. please set option `customer-id`")
		}

		if pCfg.Options == nil || pCfg.Options["impersonated-user-email"] == "" {
			return nil, errors.New("google workspace provider requires an impersonated user email. please set option `impersonated_user_email`")
		}
	} else {
		return nil, providers.ErrProviderTypeDoesNotMatch
	}

	var resourceType ResourceType
	var id string
	requireServiceAccount := false
	if pCfg.Options["project-id"] != "" {
		resourceType = Project
		id = pCfg.Options["project-id"]

		// FIXME: DEPRECATED, remove in v8.0 vv
		// The options "project" and "organization" have been deprecated in favor of project-id and organization-id
	} else if pCfg.Options["project"] != "" {
		resourceType = Project
		id = pCfg.Options["project"]
		// ^^

	} else if pCfg.Options["organization-id"] != "" {
		resourceType = Organization
		id = pCfg.Options["organization-id"]

		// FIXME: DEPRECATED, remove in v8.0 vv
		// The options "project" and "organization" have been deprecated in favor of project-id and organization-id
	} else if pCfg.Options["organization"] != "" {
		resourceType = Organization
		id = pCfg.Options["organization"]
		// ^^

	} else if pCfg.Options["customer-id"] != "" {
		resourceType = Workspace
		id = pCfg.Options["customer-id"]
		requireServiceAccount = true
	} else if pCfg.Options["folder-id"] != "" {
		resourceType = Folder
		id = pCfg.Options["folder-id"]
	}

	var override string
	if pCfg.Options != nil {
		override = pCfg.Options["platform-override"]
	}

	t := &Provider{
		resourceType:     resourceType,
		id:               id,
		opts:             pCfg.Options,
		cred:             cred,
		platformOverride: override,
	}

	if cred == nil && requireServiceAccount {
		return nil, errors.New("google workspace provider requires a service account")
	}

	// verify that we have access to the organization or project
	switch resourceType {
	case Organization:
		_, err := t.GetOrganization(id)
		if err != nil {
			log.Error().Err(err).Msgf("could not find or have no access to organization %s", id)
			return nil, err
		}
	case Project:
		_, err := t.GetProject(id)
		if err != nil {
			log.Error().Err(err).Msgf("could not find or have no access to project %s", id)
			return nil, err
		}
	case Workspace:
		_, err := t.GetWorkspaceCustomer(id)
		if err != nil {
			log.Error().Err(err).Msgf("could not find or have no access to workspace %s", id)
			return nil, err
		}
		t.serviceAccountSubject = pCfg.Options["impersonated-user-email"]

	}
	return t, nil
}

type Provider struct {
	resourceType ResourceType
	id           string
	opts         map[string]string
	// serviceAccountSubject subject is used to impersonate a subject
	serviceAccountSubject string
	cred                  *vault.Credential
	platformOverride      string
}

func (p *Provider) Close() {}

func (p *Provider) Capabilities() providers.Capabilities {
	return providers.Capabilities{
		providers.Capability_Google,
	}
}

func (p *Provider) Options() map[string]string {
	return p.opts
}

func (p *Provider) Kind() providers.Kind {
	return providers.Kind_KIND_API
}

func (p *Provider) Runtime() string {
	if p.resourceType == Workspace {
		return providers.RUNTIME_GOOGLE_WORKSPACE
	}
	return providers.RUNTIME_GCP
}

func (p *Provider) PlatformIdDetectors() []providers.PlatformIdDetector {
	return []providers.PlatformIdDetector{
		providers.TransportPlatformIdentifierDetector,
	}
}
