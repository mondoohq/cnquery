// shake that SSL

package tlsshake

import (
	"bufio"
	"bytes"
	"crypto/x509"
	"encoding/binary"
	"encoding/hex"
	"io"
	"math/rand"
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
	"golang.org/x/crypto/ocsp"
)

var TLS_VERSIONS = []string{"ssl3", "tls1.0", "tls1.1", "tls1.2", "tls1.3"}

// ScanConfig allows to tune the TLS scanner
type ScanConfig struct {
	Versions                  []string
	SNIsupported              bool
	FakeSNI                   bool
	SecureClientRenegotiation bool

	// internal scan fields that users don't configure
	version       string
	ciphersFilter func(string) bool
}

func DefaultScanConfig() ScanConfig {
	return ScanConfig{
		SNIsupported:              true,
		FakeSNI:                   true,
		SecureClientRenegotiation: true,
	}
}

// Tester is the test runner and results object for any findings done in a
// session of tests. We re-use it to avoid duplicate requests and optimize
// the overall test run.
type Tester struct {
	Findings   Findings
	sync       sync.Mutex
	proto      string
	target     string
	domainName string
}

// Findings tracks the current state of tested components and their findings
type Findings struct {
	Versions           map[string]bool
	Ciphers            map[string]bool
	Extensions         map[string]bool
	Errors             []string
	Certificates       []*x509.Certificate
	NonSNIcertificates []*x509.Certificate
	Revocations        map[string]*revocation
}

type revocation struct {
	At     time.Time
	Via    string
	Reason int
}

// New creates a new tester object for the given target (via proto, host, port)
//   - the proto, host, and port are used to construct the target for net.Dial
//     example: proto="tcp", host="mondoo.io", port=443
func New(proto string, domainName string, host string, port int) *Tester {
	target := host + ":" + strconv.Itoa(port)

	return &Tester{
		Findings: Findings{
			Versions:    map[string]bool{},
			Ciphers:     map[string]bool{},
			Extensions:  map[string]bool{},
			Revocations: map[string]*revocation{},
		},
		proto:      proto,
		target:     target,
		domainName: domainName,
	}
}

// Test runs the TLS/SSL probes for a given scan configuration
//   - versions may contain any supported pre-defined TLS/SSL versions
//     with a complete list found in TLS_VERSIONS. Leave empty to test all.
func (s *Tester) Test(conf ScanConfig) error {
	if len(conf.Versions) == 0 {
		conf.Versions = TLS_VERSIONS
	}

	workers := sync.WaitGroup{}
	var errs error

	remainingCiphers := func(cipher string) bool {
		s.sync.Lock()
		defer s.sync.Unlock()
		if v, ok := s.Findings.Ciphers[cipher]; ok && v {
			return false
		}
		return true
	}
	supportedCiphers := func(cipher string) bool {
		s.sync.Lock()
		defer s.sync.Unlock()
		if v, ok := s.Findings.Ciphers[cipher]; ok && v {
			return true
		}
		return false
	}

	for i := range conf.Versions {
		version := conf.Versions[i]

		workers.Add(1)
		go func() {
			defer workers.Done()

			// we don't activate any of the additioinal tests in the beginning
			// let's find out if we work on this version of TLS/SSL
			curConf := &ScanConfig{
				version:       version,
				ciphersFilter: remainingCiphers,
			}

			for {
				remaining, err := s.testTLS(s.proto, s.target, curConf)
				if err != nil {
					s.sync.Lock()
					errs = multierror.Append(errs, err)
					s.sync.Unlock()
					return
				}

				if remaining <= 0 {
					break
				}
			}

			if version == "tls1.2" || version == "tls1.3" {
				if conf.SNIsupported || conf.SecureClientRenegotiation {
					curConf = &ScanConfig{
						version:                   version,
						ciphersFilter:             supportedCiphers,
						SNIsupported:              conf.SNIsupported,
						SecureClientRenegotiation: conf.SecureClientRenegotiation,
					}
					s.testTLS(s.proto, s.target, curConf)
				}

				if conf.FakeSNI {
					curConf = &ScanConfig{
						version:       version,
						ciphersFilter: supportedCiphers,
						FakeSNI:       conf.FakeSNI,
					}
					s.testTLS(s.proto, s.target, curConf)
				}
			}
		}()
	}

	workers.Wait()

	return nil
}

// Attempts to connect to an endpoint with a given version and records
// results in the Tester.
// Returns the number of remaining ciphers to test (if so desired)
// and any potential error
func (s *Tester) testTLS(proto string, target string, conf *ScanConfig) (int, error) {
	conn, err := net.Dial(proto, target)
	if err != nil {
		return 0, errors.Wrap(err, "failed to connect to target")
	}
	defer conn.Close()

	msg, cipherCount, err := s.helloTLSMsg(conf)
	if err != nil {
		return 0, err
	}

	_, err = conn.Write(msg)
	if err != nil {
		return 0, errors.Wrap(err, "failed to send TLS hello")
	}

	success, err := s.parseHello(conn, conf)
	if err != nil || !success {
		return 0, err
	}

	return cipherCount - 1, nil
}

func (s *Tester) addError(msg string) {
	s.sync.Lock()
	s.Findings.Errors = append(s.Findings.Errors, msg)
	s.sync.Unlock()
}

func (s *Tester) parseAlert(data []byte, conf *ScanConfig) error {
	var severity string
	switch data[0] {
	case '\x01':
		severity = "Warning"
	case '\x02':
		severity = "Fatal"
	default:
		severity = "Unknown"
	}

	description, ok := ALERT_DESCRIPTIONS[data[1]]
	if !ok {
		description = "cannot find description"
	}

	switch description {
	case "PROTOCOL_VERSION":
		// here we know the TLS version is not supported
		s.sync.Lock()
		s.Findings.Versions[conf.version] = false
		s.sync.Unlock()

	case "HANDSHAKE_FAILURE":
		if conf.version == "ssl3" {
			// Note: it's a little fuzzy here, since we don't know if the protocol
			// version is unsupported or just its ciphers. So we check if we found
			// it previously and if so, don't add it to the list of unsupported
			// versions.
			if _, ok := s.Findings.Versions["ssl3"]; !ok {
				s.sync.Lock()
				s.Findings.Versions["ssl3"] = false
				s.sync.Unlock()
			}
		}

		names := cipherNames(conf.version, conf.ciphersFilter)
		for i := range names {
			name := names[i]
			if _, ok := s.Findings.Ciphers[name]; !ok {
				s.sync.Lock()
				s.Findings.Ciphers[name] = false
				s.sync.Unlock()
			}
		}

	default:
		s.addError("failed to connect via " + conf.version + ": " + severity + " - " + description)
	}

	return nil
}

func (s *Tester) parseServerHello(data []byte, version string, conf *ScanConfig) error {
	idx := 0

	idx += 2 + 32
	// handshake tls version (2), which we don't need yet (we will look at it in the extension if necessary)
	// random (32), which we don't need

	// we don't need the session ID
	sessionIDlen := byte1int(data[idx])
	idx += 1
	idx += sessionIDlen

	cipher, cipherOK := ALL_CIPHERS[string(data[idx:idx+2])]
	idx += 2

	// TLS 1.3 pretends to be TLS 1.2 in the preceeding headers for
	// compatibility. To correctly identify it, we have to look at
	// any Supported Versions extensions that the server sent us.

	// compression method (which should be set to null)
	idx += 1

	// no extensions found
	var allExtLen int
	if len(data) >= idx+2 {
		allExtLen = bytes2int(data[idx : idx+2])
		idx += 2
	}

	for allExtLen > 0 && idx < len(data) {
		extType := string(data[idx : idx+2])
		extLen := bytes2int(data[idx+2 : idx+4])
		extData := string(data[idx+4 : idx+4+extLen])

		allExtLen -= 4 + extLen
		idx += 4 + extLen

		switch extType {
		case EXTENSION_SupportedVersions:
			if v, ok := VERSIONS_LOOKUP[extData]; ok {
				version = v
			} else {
				s.Findings.Errors = append(s.Findings.Errors, "Failed to parse supported_versions extension: '"+extData+"'")
			}
		case EXTENSION_RenegotiationInfo:
			s.Findings.Extensions["renegotiation_info"] = true
		}
	}

	// we have to wait for any changes to the detected version (in the extensions)
	// once done, let's lock it once and write all results
	s.sync.Lock()
	if !cipherOK {
		s.Findings.Ciphers["unknown"] = true
	} else {
		s.Findings.Ciphers[cipher] = true
	}
	s.Findings.Versions[version] = true
	s.sync.Unlock()

	return nil
}

func (s *Tester) parseCertificate(data []byte, conf *ScanConfig) error {
	certsLen := bytes3int(data[0:3])
	if len(data) < certsLen+3 {
		return errors.New("malformed certificate response, too little data read from stream to parse certificate")
	}

	certs := []*x509.Certificate{}
	i := 3
	for i < 3+certsLen {
		certLen := bytes3int(data[i : i+3])
		i += 3

		rawCert := data[i : i+certLen]
		i += certLen

		cert, err := x509.ParseCertificate(rawCert)
		if err != nil {
			s.addError(
				errors.Wrap(err, "failed to parse certificate (x509 parser error)").Error(),
			)
		} else {
			certs = append(certs, cert)
		}
	}

	// TODO: we are currently overwriting any certs that may have been tested already
	// The assumption is that the same endpoint will always return the same
	// certificates no matter what version/configuration is used.
	// This may not be true and in case it isn't improve this code to carefully
	// write new certificates and manage separate certificate chains
	s.sync.Lock()
	if conf.SNIsupported {
		// by default we collect with SNI enabled. If the test is set to test SNI,
		// we actually test the exact opposite, ie what do we get without SNI.
		// Thus, we collect the non-sni certificates here
		s.Findings.NonSNIcertificates = certs

		if len(certs) != 0 && len(s.Findings.Certificates) != 0 {
			if !bytes.Equal(certs[0].Raw, s.Findings.Certificates[0].Raw) {
				s.Findings.Extensions["server_name"] = true
			}
		}
	} else if conf.FakeSNI {
		if len(certs) != 0 && len(s.Findings.NonSNIcertificates) != 0 {
			if bytes.Equal(certs[0].Raw, s.Findings.NonSNIcertificates[0].Raw) {
				s.Findings.Extensions["fake_server_name"] = true
			}
		}
	} else {
		s.Findings.Certificates = certs
	}
	s.sync.Unlock()

	for i := 0; i+1 < len(certs); i++ {
		err := s.ocspRequest(certs[i], certs[i+1])
		if err != nil {
			s.addError(err.Error())
		}
	}

	return nil
}

// returns true if we are done parsing through handshake responses.
//   - If i'ts a ServerHello, it will check if we have certificates.
//     If we don't, we should read more handshake responses...
//     If we do, we might as well be done at this stage, no need to read more
//   - There are a few other responses that also signal that we are done
//     processing handshake responses, like ServerHelloDone or Finished
func (s *Tester) parseHandshake(data []byte, version string, conf *ScanConfig) (bool, error) {
	handshakeType := data[0]
	handshakeLen := bytes3int(data[1:4])

	switch handshakeType {
	case HANDSHAKE_TYPE_ServerHello:
		err := s.parseServerHello(data[4:4+handshakeLen], version, conf)
		return false, err
	case HANDSHAKE_TYPE_Certificate:
		return true, s.parseCertificate(data[4:4+handshakeLen], conf)
	case HANDSHAKE_TYPE_ServerKeyExchange:
		return false, nil
	case HANDSHAKE_TYPE_ServerHelloDone:
		return true, nil
	case HANDSHAKE_TYPE_Finished:
		return true, nil
	default:
		typ := "0x" + hex.EncodeToString([]byte{handshakeType})
		s.addError("Unhandled TLS/SSL handshake: '" + typ + "'")
		return false, nil
	}
}

// returns:
//
//	true if the handshake was successful, false otherwise
func (s *Tester) parseHello(conn net.Conn, conf *ScanConfig) (bool, error) {
	reader := bufio.NewReader(conn)
	header := make([]byte, 5)
	var success bool
	var done bool

	for !done {
		_, err := io.ReadFull(reader, header)
		if err != nil {
			if err == io.EOF {
				break
			}
			return false, err
		}

		typ := "0x" + hex.EncodeToString(header[0:1])
		headerVersion := VERSIONS_LOOKUP[string(header[1:3])]

		msgLen := bytes2int(header[3:5])
		if msgLen == 0 {
			return false, errors.New("No body in TLS/SSL response (type: '" + typ + "')")
		}
		if msgLen > 1<<20 {
			return false, errors.New("TLS/SSL response body is too larget (type: '" + typ + "')")
		}

		msg := make([]byte, msgLen)
		_, err = io.ReadFull(reader, msg)
		if err != nil {
			return false, errors.Wrap(err, "Failed to read full TLS/SSL response body (type: '"+typ+"')")
		}

		switch header[0] {
		case CONTENT_TYPE_Alert:
			// Do not grab the version here, instead use the pre-provided
			// There is a nice edge-case in TLS1.3 which is handled further down,
			// but not required here since we are dealing with an error
			if err := s.parseAlert(msg, conf); err != nil {
				return false, err
			}

		case CONTENT_TYPE_Handshake:
			handshakeDone, err := s.parseHandshake(msg, headerVersion, conf)
			if err != nil {
				return false, err
			}
			success = true
			done = handshakeDone

		case CONTENT_TYPE_ChangeCipherSpec:
			// This also means we are done with this stream, since it signals that we
			// are no longer looking at a handshake.
			done = true

		case CONTENT_TYPE_Application:
			// Definitely don't care about anything past the handshake.
			done = true

		default:
			s.addError("Unhandled TLS/SSL response (received '" + typ + "')")
		}
	}

	return success, nil
}

func filterCipherMsg(org map[string]string, f func(cipher string) bool) ([]byte, int) {
	var res bytes.Buffer
	var n int
	for k, v := range org {
		if f(v) {
			res.WriteString(k)
			n++
		}
	}
	return res.Bytes(), n
}

func filterCipherNames(org map[string]string, f func(cipher string) bool) []string {
	var res []string
	for _, v := range org {
		if f(v) {
			res = append(res, v)
		}
	}
	return res
}

func cipherNames(version string, filter func(cipher string) bool) []string {
	switch version {
	case "ssl3":
		regular := filterCipherNames(SSL3_CIPHERS, filter)
		fips := filterCipherNames(SSL_FIPS_CIPHERS, filter)
		return append(regular, fips...)
	case "tls1.0", "tls1.1", "tls1.2":
		return filterCipherNames(TLS_CIPHERS, filter)
	case "tls1.3":
		return filterCipherNames(TLS13_CIPHERS, filter)
	default:
		return []string{}
	}
}

func writeExtension(buf *bytes.Buffer, typ string, body []byte) {
	buf.WriteString(typ)
	buf.Write(int2bytes(len(body)))
	buf.Write(body)
}

func sniMsg(domainName string) []byte {
	l := len(domainName)
	var res bytes.Buffer

	res.Write(int2bytes(l + 3)) // server name list length
	res.WriteByte('\x00')       // name type: host name
	res.Write(int2bytes(l))     // name length
	res.WriteString(domainName) // name

	return res.Bytes()
}

func (s *Tester) helloTLSMsg(conf *ScanConfig) ([]byte, int, error) {
	var ciphers []byte
	var cipherCount int

	var extensions bytes.Buffer

	if conf.version != "ssl3" {
		domainName := s.domainName
		if conf.SNIsupported {
			// don't write an SNI and see if we get a different certificates
			domainName = ""
		} else if conf.FakeSNI {
			// give it a fake name
			domainName = strconv.FormatUint(rand.Uint64(), 10) + ".com"
		}

		if domainName != "" {
			writeExtension(&extensions, EXTENSION_ServerName, sniMsg(domainName))
		}

		// add signature_algorithms
		extensions.WriteString("\x00\x0d\x00\x14\x00\x12\x04\x03\x08\x04\x04\x01\x05\x03\x08\x05\x05\x01\x08\x06\x06\x01\x02\x01")
	}

	if conf.version == "tls1.2" || conf.version == "tls1.3" {
		// Renegotiation info
		// https://datatracker.ietf.org/doc/html/rfc5746
		// - we leave the body empty, we only need a response to the request
		// - the body has 1 byte containing the length of the extension (which is 0)
		if conf.SecureClientRenegotiation {
			writeExtension(&extensions, EXTENSION_RenegotiationInfo, []byte("\x00"))
		}
	}

	switch conf.version {
	case "ssl3":
		regular, n1 := filterCipherMsg(SSL3_CIPHERS, conf.ciphersFilter)
		fips, n2 := filterCipherMsg(SSL_FIPS_CIPHERS, conf.ciphersFilter)
		ciphers = append(regular, fips...)
		cipherCount = n1 + n2

	case "tls1.0", "tls1.1", "tls1.2":
		org, n1 := filterCipherMsg(TLS10_CIPHERS, conf.ciphersFilter)
		tls, n2 := filterCipherMsg(TLS_CIPHERS, conf.ciphersFilter)
		ciphers = append(org, tls...)
		cipherCount = n1 + n2

		// add heartbeat
		extensions.WriteString("\x00\x0f\x00\x01\x01")
		// add ec_points_format
		extensions.WriteString("\x00\x0b\x00\x02\x01\x00")
		// add elliptic_curve
		extensions.WriteString("\x00\x0a\x00\x0a\x00\x08\xfa\xfa\x00\x1d\x00\x17\x00\x18")

	case "tls1.3":
		org, n1 := filterCipherMsg(TLS10_CIPHERS, conf.ciphersFilter)
		tls, n2 := filterCipherMsg(TLS_CIPHERS, conf.ciphersFilter)
		tls13, n3 := filterCipherMsg(TLS13_CIPHERS, conf.ciphersFilter)
		ciphers = append(org, tls...)
		ciphers = append(ciphers, tls13...)
		cipherCount = n1 + n2 + n3

		// TLSv1.3 Supported Versions extension
		extensions.WriteString("\x00\x2b\x00\x03\x02\x03\x04")
		// add supported groups extension
		extensions.WriteString("\x00\x0a\x00\x08\x00\x06\x00\x1d\x00\x17\x00\x18")

		// This is a pre-generated public/private key pair using the x25519 curve:
		// It was generated from the command line with:
		//
		// > openssl-1.1.1e/apps/openssl genpkey -algorithm x25519 > pkey
		// > openssl-1.1.1e/apps/openssl pkey -noout -text < pkey
		// priv:
		//     30:90:f3:89:f4:9e:52:59:3c:ba:e9:f4:78:84:a0:
		//     23:86:73:5e:f5:c9:46:6c:3a:c3:4e:ec:56:57:81:
		//     5d:62
		// pub:
		//     e7:08:71:36:d0:81:e0:16:19:3a:cb:67:ca:b8:28:
		//     d9:45:92:16:ff:36:63:0d:0d:5a:3d:9d:47:ce:3e:
		//     cd:7e

		publicKey := "\xe7\x08\x71\x36\xd0\x81\xe0\x16\x19\x3a\xcb\x67\xca\xb8\x28\xd9\x45\x92\x16\xff\x36\x63\x0d\x0d\x5a\x3d\x9d\x47\xce\x3e\xcd\x7e"
		extensions.WriteString("\x00\x33\x00\x26\x00\x24\x00\x1d\x00\x20")
		extensions.WriteString(publicKey)

	default:
		return nil, 0, errors.New("unsupported TLS/SSL version: " + conf.version)
	}

	return constructTLSHello(conf.version, ciphers, extensions.Bytes()), cipherCount, nil
}

// OCSP:
// https://datatracker.ietf.org/doc/html/rfc6960
// https://datatracker.ietf.org/doc/html/rfc2560

func (s *Tester) ocspRequest(cert *x509.Certificate, issuer *x509.Certificate) error {
	if len(cert.OCSPServer) == 0 {
		return errors.New("no OCSP server specified for revocation check, skipping it")
	}

	server := cert.OCSPServer[0]

	req, err := ocsp.CreateRequest(cert, issuer, &ocsp.RequestOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to create OCSP request")
	}

	reqBody := bytes.NewBuffer(req)
	res, err := http.Post(server, "application/ocsp-request", reqBody)
	if err != nil {
		return errors.Wrap(err, "failed to post OCSP request")
	}

	if res.StatusCode != 200 {
		return errors.New("OCSP request returned " + res.Status)
	}
	resp, err := io.ReadAll(res.Body)
	if err != nil {
		return errors.Wrap(err, "failed to read OCSP response")
	}
	ocspRes, err := ocsp.ParseResponseForCert(resp, cert, issuer)
	if err != nil {
		return errors.Wrap(err, "failed to parse OCSP response")
	}

	s.sync.Lock()
	if ocspRes.RevokedAt.IsZero() {
		s.Findings.Revocations[string(cert.Signature)] = nil
	} else {
		s.Findings.Revocations[string(cert.Signature)] = &revocation{
			At:     ocspRes.RevokedAt,
			Via:    server,
			Reason: ocspRes.RevocationReason,
		}
	}
	s.sync.Unlock()

	return nil
}

func int1byte(i int) []byte {
	res := make([]byte, 2)
	binary.BigEndian.PutUint16(res, uint16(i))
	return res[1:]
}

func int2bytes(i int) []byte {
	res := make([]byte, 2)
	binary.BigEndian.PutUint16(res, uint16(i))
	return res
}

func int3bytes(i int) []byte {
	res := make([]byte, 4)
	binary.BigEndian.PutUint32(res, uint32(i))
	return res[1:]
}

func byte1int(b byte) int {
	return int(binary.BigEndian.Uint16([]byte{0x00, b}))
}

func bytes2int(b []byte) int {
	return int(binary.BigEndian.Uint16(b))
}

func bytes3int(b []byte) int {
	return int(binary.BigEndian.Uint32(append([]byte{0x00}, b...)))
}

func constructTLSHello(version string, ciphers []byte, extensions []byte) []byte {
	sessionID := ""
	compressions := "\x00"

	var content bytes.Buffer
	content.WriteString(VERSIONS[version])

	rnd := make([]byte, 8)
	binary.BigEndian.PutUint64(rnd, rand.Uint64())
	content.Write(rnd)
	binary.BigEndian.PutUint64(rnd, rand.Uint64())
	content.Write(rnd)
	binary.BigEndian.PutUint64(rnd, rand.Uint64())
	content.Write(rnd)
	binary.BigEndian.PutUint64(rnd, rand.Uint64())
	content.Write(rnd)

	content.Write(int1byte(len(sessionID)))
	content.WriteString(sessionID)

	content.Write(int2bytes(len(ciphers)))
	content.Write(ciphers)

	content.Write(int1byte(len(compressions)))
	content.WriteString(compressions)

	content.Write(int2bytes(len(extensions)))
	content.Write(extensions)

	var c = content.Bytes()

	var core = []byte{HANDSHAKE_TYPE_ClientHello}
	core = append(core, int3bytes(len(c))...)
	core = append(core, c...)

	return constructTLSMsg(CONTENT_TYPE_Handshake, core, []byte(VERSIONS[version]))
}

func constructTLSMsg(contentType byte, content []byte, version []byte) []byte {
	var res bytes.Buffer
	res.WriteByte(contentType)
	res.Write(version)
	res.Write(int2bytes(len(content)))
	res.Write(content)
	return res.Bytes()
}

var VERSIONS = map[string]string{
	"ssl3":   "\x03\x00",
	"tls1.0": "\x03\x01",
	"tls1.1": "\x03\x02",
	"tls1.2": "\x03\x03",
	// RFC 8446 4.1.2:
	// In TLS 1.3, the client indicates its version preferences in the
	// "supported_versions" extension (Section 4.2.1) and the
	// legacy_version field MUST be set to 0x0303, which is the version
	// number for TLS 1.2.  TLS 1.3 ClientHellos are identified as having
	// a legacy_version of 0x0303 and a supported_versions extension
	// present with 0x0304 as the highest version indicated therein.
	"tls1.3": "\x03\x04",
}

var VERSIONS_LOOKUP map[string]string
var ALL_CIPHERS map[string]string

func init() {
	VERSIONS_LOOKUP = make(map[string]string, len(VERSIONS))
	for k, v := range VERSIONS {
		VERSIONS_LOOKUP[v] = k
	}

	ALL_CIPHERS = make(map[string]string,
		len(SSL2_CIPHERS)+
			len(SSL_FIPS_CIPHERS)+
			len(TLS10_CIPHERS)+
			len(TLS13_CIPHERS)+
			len(TLS_CIPHERS))

	// Note: overlapping names will be overwritten
	for k, v := range SSL2_CIPHERS {
		ALL_CIPHERS[k] = v
	}
	for k, v := range SSL3_CIPHERS {
		ALL_CIPHERS[k] = v
	}
	for k, v := range SSL_FIPS_CIPHERS {
		ALL_CIPHERS[k] = v
	}
	for k, v := range TLS10_CIPHERS {
		ALL_CIPHERS[k] = v
	}
	for k, v := range TLS13_CIPHERS {
		ALL_CIPHERS[k] = v
	}
	for k, v := range TLS_CIPHERS {
		ALL_CIPHERS[k] = v
	}
}

const (
	CONTENT_TYPE_ChangeCipherSpec byte = '\x14'
	CONTENT_TYPE_Alert            byte = '\x15'
	CONTENT_TYPE_Handshake        byte = '\x16'
	CONTENT_TYPE_Application      byte = '\x17'
	CONTENT_TYPE_Heartbeat        byte = '\x18'

	HANDSHAKE_TYPE_HelloRequest       byte = '\x00'
	HANDSHAKE_TYPE_ClientHello        byte = '\x01'
	HANDSHAKE_TYPE_ServerHello        byte = '\x02'
	HANDSHAKE_TYPE_NewSessionTicket   byte = '\x04'
	HANDSHAKE_TYPE_Certificate        byte = '\x0b'
	HANDSHAKE_TYPE_ServerKeyExchange  byte = '\x0c'
	HANDSHAKE_TYPE_CertificateRequest byte = '\x0d'
	HANDSHAKE_TYPE_ServerHelloDone    byte = '\x0e'
	HANDSHAKE_TYPE_CertificateVerify  byte = '\x0f'
	HANDSHAKE_TYPE_ClientKeyExchange  byte = '\x10'
	HANDSHAKE_TYPE_Finished           byte = '\x14'

	EXTENSION_ServerName        string = "\x00\x00"
	EXTENSION_SupportedVersions string = "\x00\x2b"
	EXTENSION_RenegotiationInfo string = "\xff\x01"
)

// https://tools.ietf.org/html/rfc5246#appendix-A.3
// https://tools.ietf.org/html/rfc8446#appendix-B.2
var ALERT_DESCRIPTIONS = map[byte]string{
	'\x00': "CLOSE_NOTIFY",
	'\x0A': "UNEXPECTED_MESSAGE",
	'\x14': "BAD_RECORD_MAC",
	'\x15': "DECRYPTION_FAILED_RESERVED",
	'\x16': "RECORD_OVERFLOW",
	'\x1E': "DECOMPRESSION_FAILURE",
	'\x28': "HANDSHAKE_FAILURE",
	'\x29': "NO_CERTIFICATE_RESERVED",
	'\x2A': "BAD_CERTIFICATE",
	'\x2B': "UNSUPPORTED_CERTIFICATE",
	'\x2C': "CERTIFICATE_REVOKED",
	'\x2D': "CERTIFICATE_EXPIRED",
	'\x2E': "CERTIFICATE_UNKNOWN",
	'\x2F': "ILLEGAL_PARAMETER",
	'\x30': "UNKNOWN_CA",
	'\x31': "ACCESS_DENIED",
	'\x32': "DECODE_ERROR",
	'\x33': "DECRYPT_ERROR",
	'\x3C': "EXPORT_RESTRICTION_RESERVED",
	'\x46': "PROTOCOL_VERSION",
	'\x47': "INSUFFICIENT_SECURITY",
	'\x50': "INTERNAL_ERROR",
	'\x56': "INAPPROPRIATE_FALLBACK",
	'\x5A': "USER_CANCELED",
	'\x64': "NO_RENEGOTIATION_RESERVED",
	'\x6D': "MISSING_EXTENSION",
	'\x6E': "UNSUPPORTED_EXTENSION",
	'\x6F': "CERTIFICATE_UNOBTAINABLE_RESERVED",
	'\x70': "UNRECOGNIZED_NAME",
	'\x71': "BAD_CERTIFICATE_STATUS_RESPONSE",
	'\x72': "BAD_CERTIFICATE_HASH_VALUE_RESERVED",
	'\x73': "UNKNOWN_PSK_IDENTITY",
	'\x74': "CERTIFICATE_REQUIRED",
	'\x78': "NO_APPLICATION_PROTOCOL",
}
