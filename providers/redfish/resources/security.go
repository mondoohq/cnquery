// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/stmcginnis/gofish/schemas"
	"go.mondoo.com/mql/llx"
	"go.mondoo.com/mql/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/providers/redfish/connection"
	"go.mondoo.com/mql/types"
)

// odataRef is a Redfish link to another resource.
type odataRef struct {
	ODataID string `json:"@odata.id"`
}

// protocolSettingJSON is one protocol block of a ManagerNetworkProtocol
// document. ProtocolEnabled and Port are pointers so an absent property stays
// distinguishable from a property the controller reports as false or zero.
type protocolSettingJSON struct {
	ProtocolEnabled *bool     `json:"ProtocolEnabled"`
	Port            *int64    `json:"Port"`
	Certificates    *odataRef `json:"Certificates"`
}

// networkProtocolJSON is the subset of a ManagerNetworkProtocol document the
// provider surfaces. Each protocol is a pointer so an absent block stays
// distinguishable from a block that reports the protocol as disabled.
type networkProtocolJSON struct {
	ODataID      string               `json:"@odata.id"`
	HostName     string               `json:"HostName"`
	FQDN         string               `json:"FQDN"`
	HTTP         *protocolSettingJSON `json:"HTTP"`
	HTTPS        *protocolSettingJSON `json:"HTTPS"`
	SSH          *protocolSettingJSON `json:"SSH"`
	Telnet       *protocolSettingJSON `json:"Telnet"`
	IPMI         *protocolSettingJSON `json:"IPMI"`
	SNMP         *protocolSettingJSON `json:"SNMP"`
	VirtualMedia *protocolSettingJSON `json:"VirtualMedia"`
	KVMIP        *protocolSettingJSON `json:"KVMIP"`
}

// parseNetworkProtocol decodes a ManagerNetworkProtocol document.
func parseNetworkProtocol(raw []byte) (*networkProtocolJSON, error) {
	var np networkProtocolJSON
	if err := json.Unmarshal(raw, &np); err != nil {
		return nil, err
	}
	return &np, nil
}

// protocolEnabled returns the enabled state of a protocol block, or nil when
// the controller does not report the protocol.
func protocolEnabled(p *protocolSettingJSON) *bool {
	if p == nil {
		return nil
	}
	return p.ProtocolEnabled
}

// protocolPort returns the port of a protocol block, or nil when the controller
// does not report the protocol or leaves the port out.
func protocolPort(p *protocolSettingJSON) *int64 {
	if p == nil {
		return nil
	}
	return p.Port
}

// parseManagerNetworkProtocolLink returns the NetworkProtocol URI of a manager
// document, or an empty string when the manager carries no such link.
func parseManagerNetworkProtocolLink(raw []byte) string {
	if len(raw) == 0 {
		return ""
	}
	var mgr struct {
		NetworkProtocol *odataRef `json:"NetworkProtocol"`
	}
	if err := json.Unmarshal(raw, &mgr); err != nil {
		log.Debug().Err(err).Msg("redfish: could not parse manager document")
		return ""
	}
	if mgr.NetworkProtocol == nil {
		return ""
	}
	return mgr.NetworkProtocol.ODataID
}

// parseCollectionMembers returns the member URIs of a Redfish collection.
func parseCollectionMembers(raw []byte) ([]string, error) {
	var collection struct {
		Members []odataRef `json:"Members"`
	}
	if err := json.Unmarshal(raw, &collection); err != nil {
		return nil, err
	}
	members := make([]string, 0, len(collection.Members))
	for _, m := range collection.Members {
		if m.ODataID != "" {
			members = append(members, m.ODataID)
		}
	}
	return members, nil
}

// certIdentifierJSON is the issuer or subject of a Redfish certificate.
type certIdentifierJSON struct {
	CommonName   string `json:"CommonName"`
	Organization string `json:"Organization"`
}

// certificateJSON is the subset of a Redfish Certificate document the provider
// surfaces.
type certificateJSON struct {
	ODataID                  string             `json:"@odata.id"`
	CertificateString        string             `json:"CertificateString"`
	CertificateType          string             `json:"CertificateType"`
	CertificateUsageTypes    []string           `json:"CertificateUsageTypes"`
	Fingerprint              string             `json:"Fingerprint"`
	FingerprintHashAlgorithm string             `json:"FingerprintHashAlgorithm"`
	Issuer                   certIdentifierJSON `json:"Issuer"`
	SerialNumber             string             `json:"SerialNumber"`
	SignatureAlgorithm       string             `json:"SignatureAlgorithm"`
	Subject                  certIdentifierJSON `json:"Subject"`
	ValidNotAfter            string             `json:"ValidNotAfter"`
	ValidNotBefore           string             `json:"ValidNotBefore"`
}

// parseCertificate decodes a Redfish Certificate document.
func parseCertificate(raw []byte) (*certificateJSON, error) {
	var cert certificateJSON
	if err := json.Unmarshal(raw, &cert); err != nil {
		return nil, err
	}
	return &cert, nil
}

// parsePEMCertificate decodes the PEM body of a Redfish certificate. It returns
// nil when the controller withholds the encoded certificate or the body does
// not decode, because several controllers return an empty CertificateString.
func parsePEMCertificate(certificateString string) *x509.Certificate {
	if certificateString == "" {
		return nil
	}
	block, _ := pem.Decode([]byte(certificateString))
	if block == nil || block.Type != "CERTIFICATE" {
		return nil
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		log.Debug().Err(err).Msg("redfish: could not parse certificate body")
		return nil
	}
	return cert
}

// certificateKeyInfo returns the public key size in bits and the key algorithm
// of a certificate. Both are nil and empty when the certificate is missing or
// the key type is unknown, so an audit does not read a missing value as a weak
// key.
func certificateKeyInfo(cert *x509.Certificate) (*int64, string) {
	if cert == nil {
		return nil, ""
	}
	switch key := cert.PublicKey.(type) {
	case *rsa.PublicKey:
		bits := int64(key.N.BitLen())
		return &bits, "RSA"
	case *ecdsa.PublicKey:
		bits := int64(key.Curve.Params().BitSize)
		return &bits, "ECDSA"
	case ed25519.PublicKey:
		bits := int64(len(key) * 8)
		return &bits, "Ed25519"
	default:
		return nil, ""
	}
}

// certificateSelfSigned reports whether the issuer and the subject of a
// certificate are the same entity. It prefers the encoded certificate and falls
// back to the issuer and subject that Redfish reports. The result is nil when
// neither source carries an identity, so an audit does not read missing data as
// a certificate signed by a certificate authority.
func certificateSelfSigned(cert *certificateJSON, parsed *x509.Certificate) *bool {
	if parsed != nil {
		selfSigned := bytes.Equal(parsed.RawIssuer, parsed.RawSubject)
		return &selfSigned
	}
	if cert == nil || cert.Issuer.CommonName == "" || cert.Subject.CommonName == "" {
		return nil
	}
	selfSigned := cert.Issuer.CommonName == cert.Subject.CommonName &&
		cert.Issuer.Organization == cert.Subject.Organization
	return &selfSigned
}

// redfishTimeLayouts lists the timestamp formats controllers return. The
// Redfish specification mandates ISO 8601, but several controllers drop the
// time zone or return a date alone.
var redfishTimeLayouts = []string{
	time.RFC3339,
	"2006-01-02T15:04:05",
	"2006-01-02",
}

// parseRedfishTime parses a Redfish timestamp. It returns nil when the
// controller reports no timestamp or a format the provider cannot read, so a
// date comparison resolves to null rather than to the zero time.
func parseRedfishTime(value string) *time.Time {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	for _, layout := range redfishTimeLayouts {
		if parsed, err := time.Parse(layout, value); err == nil {
			return &parsed
		}
	}
	log.Debug().Str("value", value).Msg("redfish: could not parse timestamp")
	return nil
}

// defaultVendorAccountNames holds the login names that management controllers
// ship with from the factory. Names are compared without case, because vendors
// differ in how they capitalize them.
var defaultVendorAccountNames = map[string]struct{}{
	"root":          {}, // Dell iDRAC, HPE iLO on some generations
	"administrator": {}, // HPE iLO
	"admin":         {}, // Cisco CIMC, Supermicro on newer firmware
	"userid":        {}, // Lenovo XClarity, IBM IMM
	"sysadmin":      {}, // Quanta and several OpenBMC builds
	"anonymous":     {}, // Fujitsu iRMC
}

// isDefaultVendorAccountName reports whether a login name is a known vendor
// default.
func isDefaultVendorAccountName(userName string) bool {
	_, ok := defaultVendorAccountNames[strings.ToLower(strings.TrimSpace(userName))]
	return ok
}

// isPersistentBootOverride reports whether a boot source override stays active
// across resets. It requires the Continuous state and a target other than None,
// because a Continuous state with no target changes nothing.
func isPersistentBootOverride(overrideEnabled, overrideTarget string) bool {
	if !strings.EqualFold(overrideEnabled, "Continuous") {
		return false
	}
	target := strings.TrimSpace(overrideTarget)
	return target != "" && !strings.EqualFold(target, "None")
}

// mqlRedfishInternal caches the raw ManagerNetworkProtocol documents. Both the
// network protocol settings and the HTTPS certificates come from the same
// document, so it is fetched once per scan.
type mqlRedfishInternal struct {
	networkProtocolOnce sync.Once
	networkProtocolData []*networkProtocolJSON
	networkProtocolErr  error
}

// loadNetworkProtocols fetches and parses the ManagerNetworkProtocol document
// of every controller that links one.
func (r *mqlRedfish) loadNetworkProtocols() ([]*networkProtocolJSON, error) {
	r.networkProtocolOnce.Do(func() {
		conn := redfishConn(r.MqlRuntime)
		managers, err := conn.Managers()
		if err != nil {
			r.networkProtocolErr = err
			return
		}

		res := make([]*networkProtocolJSON, 0, len(managers))
		for _, m := range managers {
			link := parseManagerNetworkProtocolLink(m.RawData)
			if link == "" {
				continue
			}
			raw, err := conn.GetRaw(link)
			if errors.Is(err, connection.ErrNotFound) {
				// The manager advertises the resource but does not serve it.
				log.Debug().Err(err).Msg("redfish: manager serves no network protocol resource")
				continue
			}
			if err != nil {
				r.networkProtocolErr = err
				return
			}
			np, err := parseNetworkProtocol(raw)
			if err != nil {
				r.networkProtocolErr = err
				return
			}
			if np.ODataID == "" {
				np.ODataID = link
			}
			res = append(res, np)
		}
		r.networkProtocolData = res
	})
	return r.networkProtocolData, r.networkProtocolErr
}

// networkProtocols returns the management network protocol settings of every
// controller. The provider parses the Redfish document itself so an absent
// protocol block resolves to null rather than to a disabled protocol.
func (r *mqlRedfish) networkProtocols() ([]any, error) {
	protocols, err := r.loadNetworkProtocols()
	if err != nil {
		return nil, err
	}

	res := make([]any, 0, len(protocols))
	for _, np := range protocols {
		o, err := CreateResource(r.MqlRuntime, "redfish.networkProtocol", map[string]*llx.RawData{
			"__id":                llx.StringData(np.ODataID),
			"hostName":            llx.StringData(np.HostName),
			"fqdn":                llx.StringData(np.FQDN),
			"httpEnabled":         llx.BoolDataPtr(protocolEnabled(np.HTTP)),
			"httpPort":            llx.IntDataPtr(protocolPort(np.HTTP)),
			"httpsEnabled":        llx.BoolDataPtr(protocolEnabled(np.HTTPS)),
			"httpsPort":           llx.IntDataPtr(protocolPort(np.HTTPS)),
			"sshEnabled":          llx.BoolDataPtr(protocolEnabled(np.SSH)),
			"sshPort":             llx.IntDataPtr(protocolPort(np.SSH)),
			"telnetEnabled":       llx.BoolDataPtr(protocolEnabled(np.Telnet)),
			"telnetPort":          llx.IntDataPtr(protocolPort(np.Telnet)),
			"ipmiEnabled":         llx.BoolDataPtr(protocolEnabled(np.IPMI)),
			"ipmiPort":            llx.IntDataPtr(protocolPort(np.IPMI)),
			"snmpEnabled":         llx.BoolDataPtr(protocolEnabled(np.SNMP)),
			"snmpPort":            llx.IntDataPtr(protocolPort(np.SNMP)),
			"virtualMediaEnabled": llx.BoolDataPtr(protocolEnabled(np.VirtualMedia)),
			"virtualMediaPort":    llx.IntDataPtr(protocolPort(np.VirtualMedia)),
			"kvmipEnabled":        llx.BoolDataPtr(protocolEnabled(np.KVMIP)),
			"kvmipPort":           llx.IntDataPtr(protocolPort(np.KVMIP)),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, o)
	}
	return res, nil
}

// certificates returns the TLS certificates that the controllers present on
// their HTTPS endpoint.
func (r *mqlRedfish) certificates() ([]any, error) {
	conn := redfishConn(r.MqlRuntime)
	protocols, err := r.loadNetworkProtocols()
	if err != nil {
		return nil, err
	}

	res := make([]any, 0, len(protocols))
	seen := map[string]struct{}{}
	for _, np := range protocols {
		if np.HTTPS == nil || np.HTTPS.Certificates == nil || np.HTTPS.Certificates.ODataID == "" {
			continue
		}

		collectionRaw, err := conn.GetRaw(np.HTTPS.Certificates.ODataID)
		if errors.Is(err, connection.ErrNotFound) {
			// The controller links a certificate collection it does not serve.
			log.Debug().Err(err).Msg("redfish: controller serves no certificate collection")
			continue
		}
		if err != nil {
			return nil, err
		}
		members, err := parseCollectionMembers(collectionRaw)
		if err != nil {
			return nil, err
		}

		for _, member := range members {
			if _, dup := seen[member]; dup {
				continue
			}
			seen[member] = struct{}{}

			certRaw, err := conn.GetRaw(member)
			if errors.Is(err, connection.ErrNotFound) {
				// The collection lists a certificate the controller removed.
				log.Debug().Err(err).Msg("redfish: certificate member is gone")
				continue
			}
			if err != nil {
				return nil, err
			}
			cert, err := parseCertificate(certRaw)
			if err != nil {
				return nil, err
			}

			parsed := parsePEMCertificate(cert.CertificateString)
			keySize, keyAlgorithm := certificateKeyInfo(parsed)

			usageTypes := make([]any, 0, len(cert.CertificateUsageTypes))
			for _, t := range cert.CertificateUsageTypes {
				usageTypes = append(usageTypes, t)
			}

			id := cert.ODataID
			if id == "" {
				id = member
			}
			o, err := CreateResource(r.MqlRuntime, "redfish.certificate", map[string]*llx.RawData{
				"__id":                     llx.StringData(id),
				"issuerCommonName":         llx.StringData(cert.Issuer.CommonName),
				"issuerOrganization":       llx.StringData(cert.Issuer.Organization),
				"subjectCommonName":        llx.StringData(cert.Subject.CommonName),
				"subjectOrganization":      llx.StringData(cert.Subject.Organization),
				"validNotBefore":           llx.TimeDataPtr(parseRedfishTime(cert.ValidNotBefore)),
				"validNotAfter":            llx.TimeDataPtr(parseRedfishTime(cert.ValidNotAfter)),
				"serialNumber":             llx.StringData(cert.SerialNumber),
				"signatureAlgorithm":       llx.StringData(cert.SignatureAlgorithm),
				"fingerprint":              llx.StringData(cert.Fingerprint),
				"fingerprintHashAlgorithm": llx.StringData(cert.FingerprintHashAlgorithm),
				"certificateType":          llx.StringData(cert.CertificateType),
				"certificateUsageTypes":    llx.ArrayData(usageTypes, types.String),
				"keySizeBits":              llx.IntDataPtr(keySize),
				"keyAlgorithm":             llx.StringData(keyAlgorithm),
				"selfSigned":               llx.BoolDataPtr(certificateSelfSigned(cert, parsed)),
			})
			if err != nil {
				return nil, err
			}
			res = append(res, o)
		}
	}
	return res, nil
}

// mqlRedfishSessionServiceInternal holds the session service document, which
// every field of the resource reads. The document itself is cached on the
// connection, because redfish.sessions reads it too.
type mqlRedfishSessionServiceInternal struct {
	once   sync.Once
	loaded bool
	svc    *schemas.SessionService
}

func (r *mqlRedfishSessionService) id() (string, error) {
	return "redfish.sessionService", nil
}

// load fetches the session service once. loaded stays false when the controller
// exposes no session service or the fetch fails, so every field resolves to
// null instead of to a disabled service.
func (r *mqlRedfishSessionService) load() {
	r.once.Do(func() {
		sessionService, err := redfishConn(r.MqlRuntime).SessionService()
		if err != nil {
			log.Warn().Err(err).Msg("redfish: could not read the session service")
			return
		}
		if sessionService == nil {
			return
		}
		r.svc = sessionService
		r.loaded = true
	})
}

func (r *mqlRedfishSessionService) serviceEnabled() (bool, error) {
	r.load()
	if !r.loaded {
		r.ServiceEnabled.State = plugin.StateIsNull | plugin.StateIsSet
		return false, nil
	}
	return r.svc.ServiceEnabled, nil
}

func (r *mqlRedfishSessionService) sessionTimeout() (int64, error) {
	r.load()
	if !r.loaded {
		r.SessionTimeout.State = plugin.StateIsNull | plugin.StateIsSet
		return 0, nil
	}
	return int64(r.svc.SessionTimeout), nil
}

func (r *mqlRedfishSessionService) absoluteSessionTimeout() (int64, error) {
	r.load()
	if !r.loaded {
		r.AbsoluteSessionTimeout.State = plugin.StateIsNull | plugin.StateIsSet
		return 0, nil
	}
	return int64(r.svc.AbsoluteSessionTimeout), nil
}

func (r *mqlRedfishSessionService) absoluteSessionTimeoutEnabled() (bool, error) {
	r.load()
	if !r.loaded {
		r.AbsoluteSessionTimeoutEnabled.State = plugin.StateIsNull | plugin.StateIsSet
		return false, nil
	}
	return r.svc.AbsoluteSessionTimeoutEnabled, nil
}

// sessions returns the sessions that are currently open on the management
// service.
func (r *mqlRedfish) sessions() ([]any, error) {
	sessionService, err := redfishConn(r.MqlRuntime).SessionService()
	if err != nil {
		return nil, err
	}
	if sessionService == nil {
		return []any{}, nil
	}
	sessions, err := sessionService.Sessions()
	if err != nil {
		return nil, err
	}

	res := make([]any, 0, len(sessions))
	for _, s := range sessions {
		roles := make([]any, 0, len(s.Roles))
		for _, role := range s.Roles {
			roles = append(roles, role)
		}

		o, err := CreateResource(r.MqlRuntime, "redfish.session", map[string]*llx.RawData{
			"__id":                  llx.StringData(s.ODataID),
			"userName":              llx.StringData(s.UserName),
			"clientOriginIPAddress": llx.StringData(s.ClientOriginIPAddress),
			"sessionType":           llx.StringData(string(s.SessionType)),
			"createdTime":           llx.TimeDataPtr(parseRedfishTime(s.CreatedTime)),
			"expirationTime":        llx.TimeDataPtr(parseRedfishTime(s.ExpirationTime)),
			"roles":                 llx.ArrayData(roles, types.String),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, o)
	}
	return res, nil
}

// serviceRootUnauthenticated reports whether the controller returns its service
// root without credentials. It resolves to null when the probe cannot reach the
// controller, so an audit does not read a network failure as a closed service
// root.
func (r *mqlRedfish) serviceRootUnauthenticated() (bool, error) {
	answers, err := redfishConn(r.MqlRuntime).ServiceRootUnauthenticated()
	if err != nil {
		log.Debug().Err(err).Msg("redfish: unauthenticated service root probe failed")
		r.ServiceRootUnauthenticated.State = plugin.StateIsNull | plugin.StateIsSet
		return false, nil
	}
	return answers, nil
}
