// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/firebase/connection"
	"go.mondoo.com/mql/v13/types"
)

func initFirebaseProject(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	conn := runtime.Connection.(*connection.FirebaseConnection)

	args["projectId"] = llx.StringData(conn.ProjectId())
	args["apiKey"] = llx.StringData(conn.ApiKey())
	args["authDomain"] = llx.StringData(conn.AuthDomain())
	args["domain"] = llx.StringData(conn.Domain())

	return args, nil, nil
}

func (c *mqlFirebaseProject) id() (string, error) {
	id := c.ProjectId.Data
	if id == "" {
		id = c.Domain.Data
	}
	return "firebase/project/" + id, nil
}

// realtimeDatabase creates the lazy-loaded Realtime Database check resource.
func (p *mqlFirebaseProject) realtimeDatabase() (*mqlFirebaseProjectRealtimeDatabase, error) {
	conn := p.MqlRuntime.Connection.(*connection.FirebaseConnection)
	projectId := conn.ProjectId()

	if projectId == "" {
		p.RealtimeDatabase.State = plugin.StateIsNull | plugin.StateIsSet
		return nil, nil
	}

	// Try both URL variants
	urls := []string{
		fmt.Sprintf("https://%s-default-rtdb.firebaseio.com", projectId),
		fmt.Sprintf("https://%s.firebaseio.com", projectId),
	}

	client := conn.HttpClient()
	publiclyReadable := false
	structureExposed := false
	testedURL := ""

	for _, baseURL := range urls {
		url := baseURL + "/.json"
		log.Debug().Str("url", url).Msg("checking Realtime Database readability")

		resp, err := client.Get(url)
		if err != nil {
			log.Debug().Err(err).Str("url", url).Msg("failed to reach Realtime Database")
			continue
		}
		// Drain and close body to allow connection reuse
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()

		// 404 means this URL variant doesn't exist, try the next one
		if resp.StatusCode == http.StatusNotFound {
			continue
		}

		testedURL = baseURL

		if resp.StatusCode == http.StatusOK {
			publiclyReadable = true
		}

		// Check shallow query for structure exposure
		shallowURL := baseURL + "/.json?shallow=true"
		shallowResp, err := client.Get(shallowURL)
		if err == nil {
			defer shallowResp.Body.Close()
			if shallowResp.StatusCode == http.StatusOK {
				body, _ := io.ReadAll(io.LimitReader(shallowResp.Body, 1<<16))
				// "null" means empty database, not structure exposure
				if string(body) != "null" {
					structureExposed = true
				}
			}
		}

		break // Successfully reached the database, no need to try the other URL
	}

	if testedURL == "" {
		testedURL = urls[0] // Default to first URL if neither responded
	}

	res, err := CreateResource(p.MqlRuntime, "firebase.project.realtimeDatabase", map[string]*llx.RawData{
		"url":              llx.StringData(testedURL),
		"publiclyReadable": llx.BoolData(publiclyReadable),
		"structureExposed": llx.BoolData(structureExposed),
	})
	if err != nil {
		return nil, err
	}

	return res.(*mqlFirebaseProjectRealtimeDatabase), nil
}

func (c *mqlFirebaseProjectRealtimeDatabase) id() (string, error) {
	return "firebase/realtimeDatabase/" + c.Url.Data, nil
}

// authConfig creates the lazy-loaded auth configuration check resource.
func (p *mqlFirebaseProject) authConfig() (*mqlFirebaseProjectAuthConfig, error) {
	conn := p.MqlRuntime.Connection.(*connection.FirebaseConnection)
	apiKey := conn.ApiKey()

	if apiKey == "" {
		p.AuthConfig.State = plugin.StateIsNull | plugin.StateIsSet
		return nil, nil
	}

	url := "https://identitytoolkit.googleapis.com/v1/projects?key=" + apiKey
	log.Debug().Str("url", url).Msg("checking Firebase auth config")

	client := conn.HttpClient()
	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to query Identity Toolkit API: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("failed to read Identity Toolkit response: %w", err)
	}

	var signInProviders []interface{}
	var anonymousAuthEnabled bool
	var authorizedDomains []interface{}

	if resp.StatusCode == http.StatusOK {
		var result map[string]interface{}
		if err := json.Unmarshal(body, &result); err != nil {
			return nil, fmt.Errorf("failed to parse Identity Toolkit response: %w", err)
		}

		// Try different response structures for providers
		if idps, ok := result["signInConfig"].(map[string]interface{}); ok {
			if methods, ok := idps["allowedMethods"].([]interface{}); ok {
				signInProviders = methods
			}
		}

		// Also check authorizedDomains
		if domains, ok := result["authorizedDomains"].([]interface{}); ok {
			authorizedDomains = domains
		}

		// Check for anonymous auth in the sign-in config
		for _, p := range signInProviders {
			if str, ok := p.(string); ok && strings.EqualFold(str, "anonymous") {
				anonymousAuthEnabled = true
				break
			}
		}
	}

	// If we didn't find providers in the new format, try the legacy format
	if len(signInProviders) == 0 && resp.StatusCode == http.StatusOK {
		var legacyResult map[string]interface{}
		if err := json.Unmarshal(body, &legacyResult); err == nil {
			// Legacy v3 format uses "idpConfig" and "signInOptions"
			if idps, ok := legacyResult["idpConfig"].([]interface{}); ok {
				for _, idp := range idps {
					if idpMap, ok := idp.(map[string]interface{}); ok {
						if provider, ok := idpMap["provider"].(string); ok {
							signInProviders = append(signInProviders, provider)
						}
					}
				}
			}
			if domains, ok := legacyResult["authorizedDomains"].([]interface{}); ok && len(authorizedDomains) == 0 {
				authorizedDomains = domains
			}
		}
	}

	res, err := CreateResource(p.MqlRuntime, "firebase.project.authConfig", map[string]*llx.RawData{
		"signInProviders":      llx.ArrayData(signInProviders, types.String),
		"anonymousAuthEnabled": llx.BoolData(anonymousAuthEnabled),
		"authorizedDomains":    llx.ArrayData(authorizedDomains, types.String),
	})
	if err != nil {
		return nil, err
	}

	return res.(*mqlFirebaseProjectAuthConfig), nil
}

func (c *mqlFirebaseProjectAuthConfig) id() (string, error) {
	return "firebase/authConfig", nil
}

// hosting creates the lazy-loaded hosting check resource.
func (p *mqlFirebaseProject) hosting() (*mqlFirebaseProjectHosting, error) {
	conn := p.MqlRuntime.Connection.(*connection.FirebaseConnection)
	domain := conn.Domain()

	if domain == "" {
		p.Hosting.State = plugin.StateIsNull | plugin.StateIsSet
		return nil, nil
	}

	if !strings.Contains(domain, "://") {
		domain = "https://" + domain
	}
	domain = strings.TrimRight(domain, "/")

	client := conn.HttpClient()

	// Check Apple App Site Association
	var appleData interface{}
	appleURL := domain + "/.well-known/apple-app-site-association"
	log.Debug().Str("url", appleURL).Msg("checking Apple App Site Association")

	if resp, err := client.Get(appleURL); err == nil {
		if resp.StatusCode == http.StatusOK {
			body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
			var parsed interface{}
			if json.Unmarshal(body, &parsed) == nil {
				appleData = parsed
			}
		}
		resp.Body.Close()
	}

	// Check Android Asset Links
	var androidData interface{}
	androidURL := domain + "/.well-known/assetlinks.json"
	log.Debug().Str("url", androidURL).Msg("checking Android Asset Links")

	if resp, err := client.Get(androidURL); err == nil {
		if resp.StatusCode == http.StatusOK {
			body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
			var parsed interface{}
			if json.Unmarshal(body, &parsed) == nil {
				androidData = parsed
			}
		}
		resp.Body.Close()
	}

	// Use the original domain (without scheme) for display
	displayDomain := conn.Domain()

	res, err := CreateResource(p.MqlRuntime, "firebase.project.hosting", map[string]*llx.RawData{
		"domain":                  llx.StringData(displayDomain),
		"appleAppSiteAssociation": llx.DictData(appleData),
		"androidAssetLinks":       llx.DictData(androidData),
	})
	if err != nil {
		return nil, err
	}

	return res.(*mqlFirebaseProjectHosting), nil
}

func (c *mqlFirebaseProjectHosting) id() (string, error) {
	return "firebase/hosting/" + c.Domain.Data, nil
}

// storage creates the lazy-loaded storage check resource.
func (p *mqlFirebaseProject) storage() (*mqlFirebaseProjectStorage, error) {
	conn := p.MqlRuntime.Connection.(*connection.FirebaseConnection)
	projectId := conn.ProjectId()

	if projectId == "" {
		p.Storage.State = plugin.StateIsNull | plugin.StateIsSet
		return nil, nil
	}

	bucketURL := fmt.Sprintf("https://firebasestorage.googleapis.com/v0/b/%s.appspot.com/o", projectId)
	log.Debug().Str("url", bucketURL).Msg("checking Firebase Storage public listing")

	client := conn.HttpClient()
	publiclyListable := false

	resp, err := client.Get(bucketURL)
	if err == nil {
		defer resp.Body.Close()
		if resp.StatusCode == http.StatusOK {
			publiclyListable = true
		}
	}

	res, err := CreateResource(p.MqlRuntime, "firebase.project.storage", map[string]*llx.RawData{
		"bucketUrl":        llx.StringData(bucketURL),
		"publiclyListable": llx.BoolData(publiclyListable),
	})
	if err != nil {
		return nil, err
	}

	return res.(*mqlFirebaseProjectStorage), nil
}

func (c *mqlFirebaseProjectStorage) id() (string, error) {
	return "firebase/storage/" + c.BucketUrl.Data, nil
}
