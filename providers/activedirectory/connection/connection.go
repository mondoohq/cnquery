// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package connection

import (
	"crypto/tls"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"

	"github.com/go-ldap/ldap/v3"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/mql/v13/providers-sdk/v1/inventory"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
)

const (
	// Option keys for inventory.Config.Options
	OptionDC       = "dc"
	OptionUser     = "user"
	OptionPassword = "password"
	OptionDomain   = "domain"
	OptionBaseDN   = "base-dn"
	OptionLDAPS    = "ldaps"
	OptionPort     = "port"
	OptionInsecure = "insecure"
	OptionBackend  = "backend"
)

// ActiveDirectoryConnection manages a single LDAP connection to an
// Active Directory Domain Services domain controller.
type ActiveDirectoryConnection struct {
	plugin.Connection
	Conf  *inventory.Config
	asset *inventory.Asset

	ldapConn *ldap.Conn
	dcHost   string

	baseDN           string
	configDN         string
	schemaDN         string
	rootDomainDN     string
	domainSID        string
	rootDomainSID    string
	domainDnsZonesDN string
	forestDnsZonesDN string

	domainFunctionalLevel string
	forestFunctionalLevel string

	cacheMu sync.RWMutex
	cache   map[string]interface{}
}

// NewActiveDirectoryConnection dials the domain controller, binds, queries
// RootDSE for naming contexts and functional levels, and retrieves the domain
// and forest root SIDs.
func NewActiveDirectoryConnection(id uint32, asset *inventory.Asset, conf *inventory.Config) (*ActiveDirectoryConnection, error) {
	if conf.Options == nil {
		return nil, errors.New("active directory provider requires connection options")
	}

	dcHost := conf.Options[OptionDC]
	if dcHost == "" {
		return nil, errors.New("active directory provider requires option 'dc' (domain controller hostname)")
	}

	user := conf.Options[OptionUser]
	if user == "" {
		return nil, errors.New("active directory provider requires option 'user'")
	}

	password := conf.Options[OptionPassword]
	if password == "" {
		return nil, errors.New("active directory provider requires option 'password'")
	}

	useTLS := strings.EqualFold(conf.Options[OptionLDAPS], "true")
	insecure := strings.EqualFold(conf.Options[OptionInsecure], "true")

	port := defaultPort(useTLS)
	if p := conf.Options[OptionPort]; p != "" {
		parsed, err := strconv.Atoi(p)
		if err != nil {
			return nil, fmt.Errorf("invalid port %q: %w", p, err)
		}
		port = parsed
	}

	addr := fmt.Sprintf("%s:%d", dcHost, port)

	// Dial LDAP
	var ldapConn *ldap.Conn
	var err error
	if useTLS {
		ldapConn, err = ldap.DialTLS("tcp", addr, &tls.Config{
			InsecureSkipVerify: insecure, //nolint:gosec // user-controlled flag for lab/test environments
		})
	} else {
		ldapConn, err = ldap.DialURL("ldap://" + addr)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to dial LDAP at %s: %w", addr, err)
	}

	// Simple bind
	if err := ldapConn.Bind(user, password); err != nil {
		ldapConn.Close()
		return nil, fmt.Errorf("LDAP bind failed for %s: %w", user, err)
	}

	c := &ActiveDirectoryConnection{
		Connection: plugin.NewConnection(id, asset),
		Conf:       conf,
		asset:      asset,
		ldapConn:   ldapConn,
		dcHost:     dcHost,
		cache:      make(map[string]interface{}),
	}

	// Override baseDN from option if provided
	if explicitBase := conf.Options[OptionBaseDN]; explicitBase != "" {
		c.baseDN = explicitBase
	}

	if err := c.discoverRootDSE(); err != nil {
		ldapConn.Close()
		return nil, fmt.Errorf("RootDSE discovery failed: %w", err)
	}

	if err := c.discoverDomainSID(); err != nil {
		ldapConn.Close()
		return nil, fmt.Errorf("domain SID discovery failed: %w", err)
	}

	if err := c.discoverRootDomainSID(); err != nil {
		ldapConn.Close()
		return nil, fmt.Errorf("forest root domain SID discovery failed: %w", err)
	}

	log.Info().
		Str("dc", dcHost).
		Str("baseDN", c.baseDN).
		Str("domainSID", c.domainSID).
		Str("forestRootSID", c.rootDomainSID).
		Str("domainLevel", c.domainFunctionalLevel).
		Str("forestLevel", c.forestFunctionalLevel).
		Msg("Active Directory connection established")

	return c, nil
}

// discoverRootDSE queries the RootDSE (base scope, empty baseDN) to populate
// naming contexts, functional levels, and DNS zone partitions.
func (c *ActiveDirectoryConnection) discoverRootDSE() error {
	req := ldap.NewSearchRequest(
		"",
		ldap.ScopeBaseObject,
		ldap.NeverDerefAliases,
		0, 0, false,
		"(objectClass=*)",
		[]string{
			"defaultNamingContext",
			"configurationNamingContext",
			"schemaNamingContext",
			"rootDomainNamingContext",
			"namingContexts",
			"domainFunctionality",
			"forestFunctionality",
		},
		nil,
	)

	resp, err := c.ldapConn.Search(req)
	if err != nil {
		return fmt.Errorf("RootDSE search: %w", err)
	}
	if len(resp.Entries) == 0 {
		return errors.New("RootDSE returned no entries")
	}

	entry := resp.Entries[0]

	// Only set baseDN from RootDSE if not explicitly overridden via options.
	if c.baseDN == "" {
		c.baseDN = GetStringAttr(entry, "defaultNamingContext")
	}
	c.configDN = GetStringAttr(entry, "configurationNamingContext")
	c.schemaDN = GetStringAttr(entry, "schemaNamingContext")
	c.rootDomainDN = GetStringAttr(entry, "rootDomainNamingContext")

	domainLevel := GetStringAttr(entry, "domainFunctionality")
	forestLevel := GetStringAttr(entry, "forestFunctionality")
	c.domainFunctionalLevel = FunctionalLevelName(domainLevel)
	c.forestFunctionalLevel = FunctionalLevelName(forestLevel)

	// Detect DomainDnsZones and ForestDnsZones application partitions.
	for _, nc := range GetStringSliceAttr(entry, "namingContexts") {
		upper := strings.ToUpper(nc)
		if strings.HasPrefix(upper, "DC=DOMAINDNSZONES,") {
			c.domainDnsZonesDN = nc
		} else if strings.HasPrefix(upper, "DC=FORESTDNSZONES,") {
			c.forestDnsZonesDN = nc
		}
	}

	return nil
}

// discoverDomainSID retrieves the objectSid of the current domain by
// searching the baseDN at base scope.
func (c *ActiveDirectoryConnection) discoverDomainSID() error {
	sid, err := c.fetchObjectSID(c.baseDN)
	if err != nil {
		return fmt.Errorf("reading domain objectSid from %s: %w", c.baseDN, err)
	}
	c.domainSID = sid
	return nil
}

// discoverRootDomainSID retrieves the objectSid of the forest root domain.
// In a single-domain forest, rootDomainDN == baseDN, so this may be identical
// to domainSID. In a child domain it differs and is needed for resolving
// Enterprise Admins / Schema Admins well-known SIDs.
func (c *ActiveDirectoryConnection) discoverRootDomainSID() error {
	if c.rootDomainDN == "" || c.rootDomainDN == c.baseDN {
		// Same domain — reuse the already-fetched SID.
		c.rootDomainSID = c.domainSID
		return nil
	}
	sid, err := c.fetchObjectSID(c.rootDomainDN)
	if err != nil {
		return fmt.Errorf("reading forest root objectSid from %s: %w", c.rootDomainDN, err)
	}
	c.rootDomainSID = sid
	return nil
}

// fetchObjectSID performs a base-scope search for objectSid on the given DN
// and returns the decoded string SID.
func (c *ActiveDirectoryConnection) fetchObjectSID(dn string) (string, error) {
	req := ldap.NewSearchRequest(
		dn,
		ldap.ScopeBaseObject,
		ldap.NeverDerefAliases,
		0, 0, false,
		"(objectClass=*)",
		[]string{"objectSid"},
		nil,
	)
	resp, err := c.ldapConn.Search(req)
	if err != nil {
		return "", err
	}
	if len(resp.Entries) == 0 {
		return "", fmt.Errorf("no entry returned for DN %s", dn)
	}
	raw := GetBinaryAttr(resp.Entries[0], "objectSid")
	if len(raw) == 0 {
		return "", fmt.Errorf("objectSid attribute empty for DN %s", dn)
	}
	return DecodeSID(raw)
}

func defaultPort(useTLS bool) int {
	if useTLS {
		return 636
	}
	return 389
}

// ---------------------------------------------------------------------------
// Accessors
// ---------------------------------------------------------------------------

func (c *ActiveDirectoryConnection) Name() string              { return "activedirectory" }
func (c *ActiveDirectoryConnection) Asset() *inventory.Asset   { return c.asset }
func (c *ActiveDirectoryConnection) FQDN() string              { return c.dcHost }
func (c *ActiveDirectoryConnection) LDAPConn() *ldap.Conn      { return c.ldapConn }
func (c *ActiveDirectoryConnection) BaseDN() string            { return c.baseDN }
func (c *ActiveDirectoryConnection) ConfigDN() string          { return c.configDN }
func (c *ActiveDirectoryConnection) SchemaDN() string          { return c.schemaDN }
func (c *ActiveDirectoryConnection) RootDomainDN() string      { return c.rootDomainDN }
func (c *ActiveDirectoryConnection) DomainSID() string         { return c.domainSID }
func (c *ActiveDirectoryConnection) RootDomainSID() string     { return c.rootDomainSID }
func (c *ActiveDirectoryConnection) DomainDnsZonesDN() string  { return c.domainDnsZonesDN }
func (c *ActiveDirectoryConnection) ForestDnsZonesDN() string  { return c.forestDnsZonesDN }
func (c *ActiveDirectoryConnection) DomainFunctionalLevel() string { return c.domainFunctionalLevel }
func (c *ActiveDirectoryConnection) ForestFunctionalLevel() string { return c.forestFunctionalLevel }

// PlatformId returns a deterministic platform identifier for the connected domain.
func (c *ActiveDirectoryConnection) PlatformId() string {
	return "//platformid.api.mondoo.app/runtime/activedirectory/domain/" + c.baseDN
}

// Close terminates the LDAP connection.
func (c *ActiveDirectoryConnection) Close() error {
	if c.ldapConn != nil {
		c.ldapConn.Close()
	}
	return nil
}

// ---------------------------------------------------------------------------
// Thread-safe cache
// ---------------------------------------------------------------------------

// CachedFetch returns a cached value for the given key, computing it via fn
// on first access. Concurrent callers block on the write if the key is absent.
func (c *ActiveDirectoryConnection) CachedFetch(key string, fn func() (interface{}, error)) (interface{}, error) {
	c.cacheMu.RLock()
	if v, ok := c.cache[key]; ok {
		c.cacheMu.RUnlock()
		return v, nil
	}
	c.cacheMu.RUnlock()

	c.cacheMu.Lock()
	defer c.cacheMu.Unlock()

	// Double-check after acquiring write lock.
	if v, ok := c.cache[key]; ok {
		return v, nil
	}

	v, err := fn()
	if err != nil {
		return nil, err
	}
	c.cache[key] = v
	return v, nil
}
