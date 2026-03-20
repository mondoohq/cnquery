// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package connection

import (
	"crypto/tls"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/go-ldap/ldap/v3"
	"github.com/go-ldap/ldap/v3/gssapi"
	"github.com/jcmturner/gokrb5/v8/client"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/mql/v13/providers-sdk/v1/inventory"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/vault"
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
	OptionStartTLS = "starttls"
	OptionKerberos = "kerberos"
	OptionKeytab   = "keytab"
	OptionKrb5Conf = "krb5conf"
	OptionCCache   = "ccache"
)

// ActiveDirectoryConnection manages a single LDAP connection to an
// Active Directory Domain Services domain controller.
type ActiveDirectoryConnection struct {
	plugin.Connection
	Conf  *inventory.Config
	asset *inventory.Asset

	ldapConn *ldap.Conn
	dcHost   string

	baseDN               string
	configDN             string
	schemaDN             string
	rootDomainDN         string
	domainSID            string
	rootDomainSID        string
	domainDnsZonesDN     string
	forestDnsZonesDN     string
	domainNamingContexts []string

	domainFunctionalLevel string
	forestFunctionalLevel string
	cacheMu               sync.RWMutex
	cache                 map[string]interface{}
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

	// Read credentials: prefer Credentials slice (set by ParseCLI), fall back to Options.
	var user, password string
	if len(conf.Credentials) > 0 && conf.Credentials[0].Type == vault.CredentialType_password {
		user = conf.Credentials[0].User
		password = string(conf.Credentials[0].Secret)
	} else {
		user = conf.Options[OptionUser]
		password = conf.Options[OptionPassword]
	}

	backend := conf.Options[OptionBackend]
	if backend == "rsat" {
		return nil, errors.New("backend 'rsat' is not yet implemented; use --backend=ldap (the default)")
	}

	useTLS := strings.EqualFold(conf.Options[OptionLDAPS], "true")
	useStartTLS := strings.EqualFold(conf.Options[OptionStartTLS], "true")
	useKerberos := strings.EqualFold(conf.Options[OptionKerberos], "true")
	insecure := strings.EqualFold(conf.Options[OptionInsecure], "true")

	// Validate mutually exclusive TLS options.
	if useTLS && useStartTLS {
		return nil, errors.New("--ldaps and --starttls are mutually exclusive; use one or the other")
	}

	// Kerberos auth doesn't require a password (keytab or ccache can substitute),
	// but simple bind always does.
	if !useKerberos {
		if user == "" {
			return nil, errors.New("active directory provider requires option 'user'")
		}
		if password == "" {
			return nil, errors.New("active directory provider requires option 'password'")
		}
	}

	port := defaultPort(useTLS)
	if p := conf.Options[OptionPort]; p != "" {
		parsed, err := strconv.Atoi(p)
		if err != nil {
			return nil, fmt.Errorf("invalid port %q: %w", p, err)
		}
		port = parsed
	}

	addr := fmt.Sprintf("%s:%d", dcHost, port)

	// --- Dial ---
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

	// --- StartTLS: upgrade plaintext connection to TLS before bind ---
	if useStartTLS {
		if err := ldapConn.StartTLS(&tls.Config{
			ServerName:         dcHost,
			InsecureSkipVerify: insecure, //nolint:gosec // user-controlled flag for lab/test environments
		}); err != nil {
			ldapConn.Close()
			return nil, fmt.Errorf("StartTLS upgrade failed for %s: %w", addr, err)
		}
	}

	// --- Bind: Kerberos/GSSAPI or simple bind ---
	if useKerberos {
		if err := kerberosGSSAPIBind(ldapConn, dcHost, user, password, conf.Options); err != nil {
			ldapConn.Close()
			return nil, err
		}
	} else {
		if err := ldapConn.Bind(user, password); err != nil {
			ldapConn.Close()
			return nil, fmt.Errorf("LDAP bind failed for %s: %w", user, err)
		}
	}

	c := &ActiveDirectoryConnection{
		Connection: plugin.NewConnection(id, asset),
		Conf:       conf,
		asset:      asset,
		ldapConn:   ldapConn,
		dcHost:     dcHost,
		cache:      make(map[string]interface{}),
	}

	// Override baseDN: --base-dn takes precedence, then --domain, then RootDSE auto-detection.
	if explicitBase := conf.Options[OptionBaseDN]; explicitBase != "" {
		c.baseDN = explicitBase
	} else if domain := conf.Options[OptionDomain]; domain != "" {
		c.baseDN = domainToDN(domain)
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

	authMethod := "simple-bind"
	if useKerberos {
		authMethod = "kerberos/gssapi"
	}
	log.Info().
		Str("dc", dcHost).
		Str("baseDN", c.baseDN).
		Str("domainSID", c.domainSID).
		Str("forestRootSID", c.rootDomainSID).
		Str("domainLevel", c.domainFunctionalLevel).
		Str("forestLevel", c.forestFunctionalLevel).
		Str("auth", authMethod).
		Msg("Active Directory connection established")

	return c, nil
}

// domainToDN converts a DNS domain name to an LDAP distinguished name.
// Example: "mini.lab" → "DC=mini,DC=lab"
func domainToDN(domain string) string {
	parts := strings.Split(domain, ".")
	for i, p := range parts {
		parts[i] = "DC=" + p
	}
	return strings.Join(parts, ",")
}

// kerberosGSSAPIBind performs a Kerberos/GSSAPI SASL bind on the connection.
// It supports three credential sources, tried in order:
//  1. --keytab: service keytab file
//  2. --ccache: existing Kerberos credential cache (e.g. from kinit)
//  3. --user + --password: password-based Kerberos AS exchange
//
// The krb5.conf location is taken from --krb5conf, KRB5_CONFIG env, or
// the platform default /etc/krb5.conf.
func kerberosGSSAPIBind(conn *ldap.Conn, dcHost, user, password string, opts map[string]string) error {
	krb5confPath := resolveKrb5Conf(opts[OptionKrb5Conf])

	// LDAP service principal: ldap/<dc_hostname>
	servicePrincipal := "ldap/" + dcHost

	var gssClient *gssapi.Client
	var err error

	switch {
	case opts[OptionKeytab] != "":
		if user == "" {
			return errors.New("--user is required with --keytab to identify the principal")
		}
		// Extract realm from user@REALM or use empty string for default realm.
		principal, realm := splitPrincipal(user)
		gssClient, err = gssapi.NewClientWithKeytab(principal, realm, opts[OptionKeytab], krb5confPath, client.DisablePAFXFAST(true))
		if err != nil {
			return fmt.Errorf("kerberos keytab client: %w", err)
		}

	case opts[OptionCCache] != "":
		gssClient, err = gssapi.NewClientFromCCache(opts[OptionCCache], krb5confPath, client.DisablePAFXFAST(true))
		if err != nil {
			return fmt.Errorf("kerberos ccache client: %w", err)
		}

	default:
		if user == "" || password == "" {
			return errors.New("--kerberos requires either --keytab, --ccache, or both --user and --password")
		}
		principal, realm := splitPrincipal(user)
		gssClient, err = gssapi.NewClientWithPassword(principal, realm, password, krb5confPath, client.DisablePAFXFAST(true))
		if err != nil {
			return fmt.Errorf("kerberos password client: %w", err)
		}
	}

	log.Debug().
		Str("servicePrincipal", servicePrincipal).
		Str("krb5conf", krb5confPath).
		Msg("performing GSSAPI/Kerberos bind")

	if err := conn.GSSAPIBind(gssClient, servicePrincipal, ""); err != nil {
		gssClient.Close()
		// AD error 80090308 (SEC_E_INVALID_TOKEN) with data 57 typically means
		// the DC requires LDAP signing/sealing for SASL binds but the go-ldap
		// GSSAPI client does not negotiate SASL security layers (upstream
		// issue: https://github.com/go-ldap/ldap/issues/552). Advise the user
		// to use LDAPS/StartTLS or fall back to simple bind.
		if strings.Contains(err.Error(), "80090308") {
			return fmt.Errorf("GSSAPI bind to %s failed (the domain controller likely requires "+
				"LDAP signing for SASL binds; use --ldaps or --starttls, or fall back to "+
				"simple bind with --user/--password without --kerberos): %w", servicePrincipal, err)
		}
		return fmt.Errorf("GSSAPI bind to %s failed: %w", servicePrincipal, err)
	}

	return nil
}

// resolveKrb5Conf returns the krb5.conf path from the explicit option,
// the KRB5_CONFIG environment variable, or the platform default.
func resolveKrb5Conf(explicit string) string {
	if explicit != "" {
		return explicit
	}
	if env := os.Getenv("KRB5_CONFIG"); env != "" {
		return env
	}
	return "/etc/krb5.conf"
}

// splitPrincipal splits a Kerberos principal like "user@REALM" into
// ("user", "REALM"). If no '@' is present, realm is empty and the
// gokrb5 client uses the default realm from krb5.conf.
func splitPrincipal(upn string) (string, string) {
	if idx := strings.LastIndex(upn, "@"); idx >= 0 {
		return upn[:idx], upn[idx+1:]
	}
	return upn, ""
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

	// Detect domain naming contexts and DNS application partitions from RootDSE.

	namingContexts := GetStringSliceAttr(entry, "namingContexts")
	c.domainDnsZonesDN, c.forestDnsZonesDN, c.domainNamingContexts = classifyNamingContexts(namingContexts)

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

func (c *ActiveDirectoryConnection) Name() string             { return "activedirectory" }
func (c *ActiveDirectoryConnection) Asset() *inventory.Asset  { return c.asset }
func (c *ActiveDirectoryConnection) FQDN() string             { return c.dcHost }
func (c *ActiveDirectoryConnection) LDAPConn() *ldap.Conn     { return c.ldapConn }
func (c *ActiveDirectoryConnection) BaseDN() string           { return c.baseDN }
func (c *ActiveDirectoryConnection) ConfigDN() string         { return c.configDN }
func (c *ActiveDirectoryConnection) SchemaDN() string         { return c.schemaDN }
func (c *ActiveDirectoryConnection) RootDomainDN() string     { return c.rootDomainDN }
func (c *ActiveDirectoryConnection) DomainSID() string        { return c.domainSID }
func (c *ActiveDirectoryConnection) RootDomainSID() string    { return c.rootDomainSID }
func (c *ActiveDirectoryConnection) DomainDnsZonesDN() string { return c.domainDnsZonesDN }
func (c *ActiveDirectoryConnection) ForestDnsZonesDN() string { return c.forestDnsZonesDN }
func (c *ActiveDirectoryConnection) DomainNamingContexts() []string {
	res := make([]string, len(c.domainNamingContexts))
	copy(res, c.domainNamingContexts)
	return res
}
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
