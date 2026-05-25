// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package connection

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
	"go.mondoo.com/mql/v13/providers-sdk/v1/inventory"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
)

const (
	TokenOption        = "token"
	OrganizationOption = "organization"
	ProjectOption      = "project"
	BaseURLOption      = "base-url"

	PlatformIdPrefix = "//platformid.api.mondoo.app/runtime/openai"
)

type OpenaiConnection struct {
	plugin.Connection
	Conf             *inventory.Config
	asset            *inventory.Asset
	client           *openai.Client
	adminClient      *openai.Client
	organization     string
	organizationName string
	project          string
	tokenHash        string
	isAdminKey       bool
}

func isAdminToken(token string) bool {
	return strings.HasPrefix(token, "sk-admin-")
}

func NewOpenaiConnection(id uint32, asset *inventory.Asset, conf *inventory.Config) (*OpenaiConnection, error) {
	token := conf.Options[TokenOption]
	if token == "" {
		token = os.Getenv("OPENAI_API_KEY")
	}

	org := conf.Options[OrganizationOption]
	if org == "" {
		org = os.Getenv("OPENAI_ORG_ID")
	}

	project := conf.Options[ProjectOption]
	if project == "" {
		project = os.Getenv("OPENAI_PROJECT_ID")
	}

	baseURL := conf.Options[BaseURLOption]
	if baseURL == "" {
		baseURL = os.Getenv("OPENAI_BASE_URL")
	}

	var sharedOpts []option.RequestOption
	if org != "" {
		sharedOpts = append(sharedOpts, option.WithHeader("OpenAI-Organization", org))
	}
	if project != "" {
		sharedOpts = append(sharedOpts, option.WithHeader("OpenAI-Project", project))
	}
	if baseURL != "" {
		sharedOpts = append(sharedOpts, option.WithBaseURL(baseURL))
	}

	var tokenHash string
	if token != "" {
		sum := sha256.Sum256([]byte(token))
		tokenHash = hex.EncodeToString(sum[:8])
	}

	adminKey := isAdminToken(token)

	var client *openai.Client
	var adminClient *openai.Client

	if token != "" {
		if adminKey {
			opts := make([]option.RequestOption, 0, len(sharedOpts)+1)
			opts = append(opts, option.WithAdminAPIKey(token))
			opts = append(opts, sharedOpts...)
			c := openai.NewClient(opts...)
			adminClient = &c
		} else {
			opts := make([]option.RequestOption, 0, len(sharedOpts)+1)
			opts = append(opts, option.WithAPIKey(token))
			opts = append(opts, sharedOpts...)
			c := openai.NewClient(opts...)
			client = &c
		}
	}

	conn := &OpenaiConnection{
		Connection:   plugin.NewConnection(id, asset),
		Conf:         conf,
		asset:        asset,
		client:       client,
		adminClient:  adminClient,
		organization: org,
		project:      project,
		tokenHash:    tokenHash,
		isAdminKey:   adminKey,
	}

	if conn.organization == "" && token != "" {
		apiBase := baseURL
		if apiBase == "" {
			apiBase = "https://api.openai.com"
		}
		if info, err := fetchAccountInfo(apiBase, token); err == nil {
			if info.OrgID != "" {
				conn.organization = info.OrgID
			}
			if info.OrgName != "" {
				conn.organizationName = info.OrgName
			}
		}
	}

	return conn, nil
}

type accountInfo struct {
	OrgID   string
	OrgName string
}

// fetchAccountInfo calls the undocumented /v1/me endpoint for best-effort org detection.
func fetchAccountInfo(baseURL string, token string) (*accountInfo, error) {
	req, err := http.NewRequest("GET", baseURL+"/v1/me", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)

	httpClient := &http.Client{Timeout: 10 * time.Second}
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result struct {
		Orgs struct {
			Data []struct {
				ID        string `json:"id"`
				Name      string `json:"name"`
				IsDefault bool   `json:"is_default"`
			} `json:"data"`
		} `json:"orgs"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	info := &accountInfo{}
	for _, org := range result.Orgs.Data {
		if info.OrgID == "" || org.IsDefault {
			info.OrgID = org.ID
			info.OrgName = org.Name
		}
	}

	return info, nil
}

func (c *OpenaiConnection) Name() string {
	return "openai"
}

func (c *OpenaiConnection) Asset() *inventory.Asset {
	return c.asset
}

func (c *OpenaiConnection) Client() *openai.Client {
	return c.client
}

func (c *OpenaiConnection) AdminClient() *openai.Client {
	return c.adminClient
}

func (c *OpenaiConnection) Organization() string {
	return c.organization
}

func (c *OpenaiConnection) OrganizationName() string {
	return c.organizationName
}

func (c *OpenaiConnection) Project() string {
	return c.project
}

func (c *OpenaiConnection) IsAdminKey() bool {
	return c.isAdminKey
}

func (c *OpenaiConnection) PlatformId() string {
	if c.project != "" {
		return PlatformIdPrefix + "/project/" + c.project
	}
	if c.organization != "" {
		return PlatformIdPrefix + "/org/" + c.organization
	}
	if c.tokenHash != "" {
		return PlatformIdPrefix + "/key/" + c.tokenHash
	}
	return PlatformIdPrefix
}

func (c *OpenaiConnection) Identifier() string {
	if c.project != "" {
		return c.project
	}
	if c.organization != "" {
		return c.organization
	}
	return c.tokenHash
}
