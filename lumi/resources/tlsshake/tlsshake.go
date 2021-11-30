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
	"strconv"
	"sync"

	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
)

var TLS_VERSIONS = []string{"ssl3", "tls1.0", "tls1.1", "tls1.2", "tls1.3"}

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
	Versions     map[string]bool
	Ciphers      map[string]bool
	Errors       []string
	Certificates []*x509.Certificate
}

// New creates a new tester object for the given target (via proto, host, port)
// - the proto, host, and port are used to construct the target for net.Dial
//   example: proto="tcp", host="mondoo.io", port=443
func New(proto string, domainName string, host string, port int) *Tester {
	target := host + ":" + strconv.Itoa(port)

	return &Tester{
		Findings: Findings{
			Versions: map[string]bool{},
			Ciphers:  map[string]bool{},
		},
		proto:      proto,
		target:     target,
		domainName: domainName,
	}
}

// Test runs the TLS/SSL probes for all given versions
// - versions may contain any supported pre-defined TLS/SSL versions
//   with a complete list found in TLS_VERSIONS. Leave empty to test all.
func (s *Tester) Test(versions ...string) error {
	if len(versions) == 0 {
		versions = TLS_VERSIONS
	}

	workers := sync.WaitGroup{}
	var errs error

	for i := range versions {
		version := versions[i]

		workers.Add(1)
		go func() {
			defer workers.Done()

			for {
				remaining, err := s.testTLS(s.proto, s.target, version)
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
		}()
	}

	workers.Wait()

	return nil
}

// Attempts to connect to an endpoint with a given version and records
// results in the Tester.
// Returns the number of remaining ciphers to test (if so desired)
// and any potential error
func (s *Tester) testTLS(proto string, target string, version string) (int, error) {
	conn, err := net.Dial(proto, target)
	if err != nil {
		return 0, errors.Wrap(err, "failed to connect to target")
	}
	defer conn.Close()

	ciphersFilter := func(cipher string) bool {
		s.sync.Lock()
		defer s.sync.Unlock()
		if v, ok := s.Findings.Ciphers[cipher]; ok && v {
			return false
		}
		return true
	}

	msg, cipherCount, err := s.helloTLSMsg(version, ciphersFilter)
	if err != nil {
		return 0, err
	}

	_, err = conn.Write(msg)
	if err != nil {
		return 0, errors.Wrap(err, "failed to send TLS hello")
	}

	success, err := s.parseHello(conn, version, ciphersFilter)
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

func (s *Tester) parseAlert(data []byte, version string, ciphersFilter func(cipher string) bool) error {
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
		s.Findings.Versions[version] = false
		s.sync.Unlock()

	case "HANDSHAKE_FAILURE":
		if version == "ssl3" {
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

		names := cipherNames(version, ciphersFilter)
		for i := range names {
			name := names[i]
			if _, ok := s.Findings.Ciphers[name]; !ok {
				s.sync.Lock()
				s.Findings.Ciphers[name] = false
				s.sync.Unlock()
			}
		}

	default:
		s.addError("failed to connect via " + version + ": " + severity + " - " + description)
	}

	return nil
}

func (s *Tester) parseServerHello(data []byte, version string, ciphersFilter func(cipher string) bool) error {
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

	allExtLen := bytes2int(data[idx : idx+2])
	idx += 2

	for allExtLen > 0 && idx < len(data) {
		extType := string(data[idx : idx+2])
		extLen := bytes2int(data[idx+2 : idx+4])
		extData := string(data[idx+4 : idx+4+extLen])

		allExtLen -= 4 + extLen
		idx += 4 + extLen

		if extType == "\x00\x2b" && extData == "\x03\x04" {
			version = "tls1.3"
			break
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

func (s *Tester) parseCertificate(data []byte) error {
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
	s.Findings.Certificates = certs
	s.sync.Unlock()

	return nil
}

// returns true if we are done parsing through handshake responses.
// - If i'ts a ServerHello, it will check if we have certificates.
//   If we don't, we should read more handshake responses...
//   If we do, we might as well be done at this stage, no need to read more
// - There are a few other responses that also signal that we are done
//   processing handshake responses, like ServerHelloDone or Finished
func (s *Tester) parseHandshake(data []byte, version string, ciphersFilter func(cipher string) bool) (bool, error) {
	handshakeType := data[0]
	handshakeLen := bytes3int(data[1:4])

	switch handshakeType {
	case HANDSHAKE_TYPE_ServerHello:
		err := s.parseServerHello(data[4:4+handshakeLen], version, ciphersFilter)
		done := len(s.Findings.Certificates) != 0
		return done, err
	case HANDSHAKE_TYPE_Certificate:
		return true, s.parseCertificate(data[4 : 4+handshakeLen])
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
//   true if the handshake was successful, false otherwise
func (s *Tester) parseHello(conn net.Conn, version string, ciphersFilter func(cipher string) bool) (bool, error) {
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
			if err := s.parseAlert(msg, version, ciphersFilter); err != nil {
				return false, err
			}

		case CONTENT_TYPE_Handshake:
			handshakeDone, err := s.parseHandshake(msg, headerVersion, ciphersFilter)
			if err != nil {
				return false, err
			}
			success = true
			done = handshakeDone

		case CONTENT_TYPE_ChangeCipherSpec:
			// We don't care about other cipher strategies. We get what we need from
			// the handshake for now.
			//
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

func (s *Tester) helloTLSMsg(version string, ciphersFilter func(cipher string) bool) ([]byte, int, error) {
	var ciphers []byte
	var cipherCount int

	var extensions bytes.Buffer

	if version != "ssl3" {
		// SNI
		if s.domainName != "" {
			l := len(s.domainName)
			extensions.WriteString("\x00\x00")   // type of this extension
			extensions.Write(int2bytes(l + 5))   // length of extension
			extensions.Write(int2bytes(l + 3))   // server name list length
			extensions.WriteByte('\x00')         // name type: host name
			extensions.Write(int2bytes(l))       // name length
			extensions.WriteString(s.domainName) // name
		}

		// add signature_algorithms
		extensions.WriteString("\x00\x0d\x00\x14\x00\x12\x04\x03\x08\x04\x04\x01\x05\x03\x08\x05\x05\x01\x08\x06\x06\x01\x02\x01")
	}

	switch version {
	case "ssl3":
		regular, n1 := filterCipherMsg(SSL3_CIPHERS, ciphersFilter)
		fips, n2 := filterCipherMsg(SSL_FIPS_CIPHERS, ciphersFilter)
		ciphers = append(regular, fips...)
		cipherCount = n1 + n2

	case "tls1.0", "tls1.1", "tls1.2":
		org, n1 := filterCipherMsg(TLS10_CIPHERS, ciphersFilter)
		tls, n2 := filterCipherMsg(TLS_CIPHERS, ciphersFilter)
		ciphers = append(org, tls...)
		cipherCount = n1 + n2

		// add heartbeat
		extensions.WriteString("\x00\x0f\x00\x01\x01")
		// add ec_points_format
		extensions.WriteString("\x00\x0b\x00\x02\x01\x00")
		// add elliptic_curve
		extensions.WriteString("\x00\x0a\x00\x0a\x00\x08\xfa\xfa\x00\x1d\x00\x17\x00\x18")

	case "tls1.3":
		org, n1 := filterCipherMsg(TLS10_CIPHERS, ciphersFilter)
		tls, n2 := filterCipherMsg(TLS_CIPHERS, ciphersFilter)
		tls13, n3 := filterCipherMsg(TLS13_CIPHERS, ciphersFilter)
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
		return nil, 0, errors.New("unsupported TLS/SSL version: " + version)
	}

	return constructTLSHello(version, ciphers, extensions.Bytes()), cipherCount, nil
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
	// '\x2D': "CERTIFICATE_UNKNOWN",
	'\x2E': "ILLEGAL_PARAMETER",
	'\x2F': "UNKNOWN_CA",
	'\x30': "ACCESS_DENIED",
	'\x31': "DECODE_ERROR",
	'\x32': "DECRYPT_ERROR",
	'\x3C': "EXPORT_RESTRICTION_RESERVED",
	'\x46': "PROTOCOL_VERSION",
	'\x47': "INSUFFICIENT_SECURITY",
	'\x50': "INTERNAL_ERROR",
	'\x5A': "USER_CANCELED",
	'\x64': "NO_RENEGOTIATION",
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

var SSL2_CIPHERS = map[string]string{
	"\x01\x00\x80": "SSL_CK_RC4_128_WITH_MD5",
	"\x02\x00\x80": "SSL_CK_RC4_128_EXPORT40_WITH_MD5",
	"\x03\x00\x80": "SSL_CK_RC2_128_CBC_WITH_MD5",
	"\x04\x00\x80": "SSL_CK_RC2_128_CBC_EXPORT40_WITH_MD5",
	"\x05\x00\x80": "SSL_CK_IDEA_128_CBC_WITH_MD5",
	"\x06\x00\x40": "SSL_CK_DES_64_CBC_WITH_MD5",
	"\x07\x00\xC0": "SSL_CK_DES_192_EDE3_CBC_WITH_MD5",
	"\x08\x00\x80": "SSL_CK_RC4_64_WITH_MD5",
}

var SSL_FIPS_CIPHERS = map[string]string{
	"\xFE\xFE": "SSL_RSA_FIPS_WITH_DES_CBC_SHA",
	"\xFE\xFF": "SSL_RSA_FIPS_WITH_3DES_EDE_CBC_SHA",
	"\xFF\xE0": "SSL_RSA_FIPS_WITH_3DES_EDE_CBC_SHA",
	"\xFF\xE1": "SSL_RSA_FIPS_WITH_DES_CBC_SHA",
}

// https://datatracker.ietf.org/doc/html/rfc6101#appendix-A.6
var SSL3_CIPHERS = map[string]string{
	"\x00\x00": "SSL_NULL_WITH_NULL_NULL",
	"\x00\x01": "SSL_RSA_WITH_NULL_MD5",
	"\x00\x02": "SSL_RSA_WITH_NULL_SHA",
	"\x00\x03": "SSL_RSA_EXPORT_WITH_RC4_40_MD5",
	"\x00\x04": "SSL_RSA_WITH_RC4_128_MD5",
	"\x00\x05": "SSL_RSA_WITH_RC4_128_SHA",
	"\x00\x06": "SSL_RSA_EXPORT_WITH_RC2_CBC_40_MD5",
	"\x00\x07": "SSL_RSA_WITH_IDEA_CBC_SHA",
	"\x00\x08": "SSL_RSA_EXPORT_WITH_DES40_CBC_SHA",
	"\x00\x09": "SSL_RSA_WITH_DES_CBC_SHA",
	"\x00\x0A": "SSL_RSA_WITH_3DES_EDE_CBC_SHA",
	"\x00\x0B": "SSL_DH_DSS_EXPORT_WITH_DES40_CBC_SHA",
	"\x00\x0C": "SSL_DH_DSS_WITH_DES_CBC_SHA",
	"\x00\x0D": "SSL_DH_DSS_WITH_3DES_EDE_CBC_SHA",
	"\x00\x0E": "SSL_DH_RSA_EXPORT_WITH_DES40_CBC_SHA",
	"\x00\x0F": "SSL_DH_RSA_WITH_DES_CBC_SHA",
	"\x00\x10": "SSL_DH_RSA_WITH_3DES_EDE_CBC_SHA",
	"\x00\x11": "SSL_DHE_DSS_EXPORT_WITH_DES40_CBC_SHA",
	"\x00\x12": "SSL_DHE_DSS_WITH_DES_CBC_SHA",
	"\x00\x13": "SSL_DHE_DSS_WITH_3DES_EDE_CBC_SHA",
	"\x00\x14": "SSL_DHE_RSA_EXPORT_WITH_DES40_CBC_SHA",
	"\x00\x15": "SSL_DHE_RSA_WITH_DES_CBC_SHA",
	"\x00\x16": "SSL_DHE_RSA_WITH_3DES_EDE_CBC_SHA",
	"\x00\x17": "SSL_DH_anon_EXPORT_WITH_RC4_40_MD5",
	"\x00\x18": "SSL_DH_anon_WITH_RC4_128_MD5",
	"\x00\x19": "SSL_DH_anon_EXPORT_WITH_DES40_CBC_SHA",
	"\x00\x1A": "SSL_DH_anon_WITH_DES_CBC_SHA",
	"\x00\x1B": "SSL_DH_anon_WITH_3DES_EDE_CBC_SHA",
	// TODO: we may want to add back the FORTEZZA ciphers?
}

// TLS 1.2 https://datatracker.ietf.org/doc/html/rfc5246

var TLS10_CIPHERS = map[string]string{
	"\x00\x00": "TLS_NULL_WITH_NULL_NULL",
	"\x00\x01": "TLS_RSA_WITH_NULL_MD5",
	"\x00\x02": "TLS_RSA_WITH_NULL_SHA",
	"\x00\x03": "TLS_RSA_EXPORT_WITH_RC4_40_MD5",
	"\x00\x04": "TLS_RSA_WITH_RC4_128_MD5",
	"\x00\x05": "TLS_RSA_WITH_RC4_128_SHA",
	"\x00\x06": "TLS_RSA_EXPORT_WITH_RC2_CBC_40_MD5",
	"\x00\x07": "TLS_RSA_WITH_IDEA_CBC_SHA",
	"\x00\x08": "TLS_RSA_EXPORT_WITH_DES40_CBC_SHA",
	"\x00\x09": "TLS_RSA_WITH_DES_CBC_SHA",
	"\x00\x0A": "TLS_RSA_WITH_3DES_EDE_CBC_SHA",
	"\x00\x0B": "TLS_DH_DSS_EXPORT_WITH_DES40_CBC_SHA",
	"\x00\x0C": "TLS_DH_DSS_WITH_DES_CBC_SHA",
	"\x00\x0D": "TLS_DH_DSS_WITH_3DES_EDE_CBC_SHA",
	"\x00\x0E": "TLS_DH_RSA_EXPORT_WITH_DES40_CBC_SHA",
	"\x00\x0F": "TLS_DH_RSA_WITH_DES_CBC_SHA",
	"\x00\x10": "TLS_DH_RSA_WITH_3DES_EDE_CBC_SHA",
	"\x00\x11": "TLS_DHE_DSS_EXPORT_WITH_DES40_CBC_SHA",
	"\x00\x12": "TLS_DHE_DSS_WITH_DES_CBC_SHA",
	"\x00\x13": "TLS_DHE_DSS_WITH_3DES_EDE_CBC_SHA",
	"\x00\x14": "TLS_DHE_RSA_EXPORT_WITH_DES40_CBC_SHA",
	"\x00\x15": "TLS_DHE_RSA_WITH_DES_CBC_SHA",
	"\x00\x16": "TLS_DHE_RSA_WITH_3DES_EDE_CBC_SHA",
	"\x00\x17": "TLS_DH_anon_EXPORT_WITH_RC4_40_MD5",
	"\x00\x18": "TLS_DH_anon_WITH_RC4_128_MD5",
	"\x00\x19": "TLS_DH_anon_EXPORT_WITH_DES40_CBC_SHA",
	"\x00\x1A": "TLS_DH_anon_WITH_DES_CBC_SHA",
	"\x00\x1B": "TLS_DH_anon_WITH_3DES_EDE_CBC_SHA",
	"\x00\x1E": "TLS_KRB5_WITH_DES_CBC_SHA",
	"\x00\x1F": "TLS_KRB5_WITH_3DES_EDE_CBC_SHA",
	"\x00\x20": "TLS_KRB5_WITH_RC4_128_SHA",
	"\x00\x21": "TLS_KRB5_WITH_IDEA_CBC_SHA",
	"\x00\x22": "TLS_KRB5_WITH_DES_CBC_MD5",
	"\x00\x23": "TLS_KRB5_WITH_3DES_EDE_CBC_MD5",
	"\x00\x24": "TLS_KRB5_WITH_RC4_128_MD5",
	"\x00\x25": "TLS_KRB5_WITH_IDEA_CBC_MD5",
	"\x00\x26": "TLS_KRB5_EXPORT_WITH_DES_CBC_40_SHA",
	"\x00\x27": "TLS_KRB5_EXPORT_WITH_RC2_CBC_40_SHA",
	"\x00\x28": "TLS_KRB5_EXPORT_WITH_RC4_40_SHA",
	"\x00\x29": "TLS_KRB5_EXPORT_WITH_DES_CBC_40_MD5",
	"\x00\x2A": "TLS_KRB5_EXPORT_WITH_RC2_CBC_40_MD5",
	"\x00\x2B": "TLS_KRB5_EXPORT_WITH_RC4_40_MD5",
	"\x00\x2C": "TLS_PSK_WITH_NULL_SHA",
	"\x00\x2D": "TLS_DHE_PSK_WITH_NULL_SHA",
	"\x00\x2E": "TLS_RSA_PSK_WITH_NULL_SHA",
	"\x00\x2F": "TLS_RSA_WITH_AES_128_CBC_SHA",
	"\x00\x30": "TLS_DH_DSS_WITH_AES_128_CBC_SHA",
	"\x00\x31": "TLS_DH_RSA_WITH_AES_128_CBC_SHA",
	"\x00\x32": "TLS_DHE_DSS_WITH_AES_128_CBC_SHA",
	"\x00\x33": "TLS_DHE_RSA_WITH_AES_128_CBC_SHA",
	"\x00\x34": "TLS_DH_anon_WITH_AES_128_CBC_SHA",
	"\x00\x35": "TLS_RSA_WITH_AES_256_CBC_SHA",
	"\x00\x36": "TLS_DH_DSS_WITH_AES_256_CBC_SHA",
	"\x00\x37": "TLS_DH_RSA_WITH_AES_256_CBC_SHA",
	"\x00\x38": "TLS_DHE_DSS_WITH_AES_256_CBC_SHA",
	"\x00\x39": "TLS_DHE_RSA_WITH_AES_256_CBC_SHA",
	"\x00\x3A": "TLS_DH_anon_WITH_AES_256_CBC_SHA",
	"\x00\x3B": "TLS_RSA_WITH_NULL_SHA256",
	"\x00\x3C": "TLS_RSA_WITH_AES_128_CBC_SHA256",
	"\x00\x3D": "TLS_RSA_WITH_AES_256_CBC_SHA256",
	"\x00\x3E": "TLS_DH_DSS_WITH_AES_128_CBC_SHA256",
	"\x00\x3F": "TLS_DH_RSA_WITH_AES_128_CBC_SHA256",
	"\x00\x40": "TLS_DHE_DSS_WITH_AES_128_CBC_SHA256",
	"\x00\x41": "TLS_RSA_WITH_CAMELLIA_128_CBC_SHA",
	"\x00\x42": "TLS_DH_DSS_WITH_CAMELLIA_128_CBC_SHA",
	"\x00\x43": "TLS_DH_RSA_WITH_CAMELLIA_128_CBC_SHA",
	"\x00\x44": "TLS_DHE_DSS_WITH_CAMELLIA_128_CBC_SHA",
	"\x00\x45": "TLS_DHE_RSA_WITH_CAMELLIA_128_CBC_SHA",
	"\x00\x46": "TLS_DH_anon_WITH_CAMELLIA_128_CBC_SHA",
	"\x00\x60": "TLS_RSA_EXPORT1024_WITH_RC4_56_MD5",
	"\x00\x61": "TLS_RSA_EXPORT1024_WITH_RC2_56_MD5",
	"\x00\x62": "TLS_RSA_EXPORT1024_WITH_DES_CBC_SHA",
	"\x00\x63": "TLS_DHE_DSS_EXPORT1024_WITH_DES_CBC_SHA",
	"\x00\x64": "TLS_RSA_EXPORT1024_WITH_RC4_56_SHA",
	"\x00\x65": "TLS_DHE_DSS_EXPORT1024_WITH_RC4_56_SHA",
	"\x00\x66": "TLS_DHE_DSS_WITH_RC4_128_SHA",
	"\x00\x67": "TLS_DHE_RSA_WITH_AES_128_CBC_SHA256",
	"\x00\x68": "TLS_DH_DSS_WITH_AES_256_CBC_SHA256",
	"\x00\x69": "TLS_DH_RSA_WITH_AES_256_CBC_SHA256",
	"\x00\x6A": "TLS_DHE_DSS_WITH_AES_256_CBC_SHA256",
	"\x00\x6B": "TLS_DHE_RSA_WITH_AES_256_CBC_SHA256",
	"\x00\x6C": "TLS_DH_anon_WITH_AES_128_CBC_SHA256",
	"\x00\x6D": "TLS_DH_anon_WITH_AES_256_CBC_SHA256",
	"\x00\x80": "TLS_GOSTR341094_WITH_28147_CNT_IMIT",
	"\x00\x81": "TLS_GOSTR341001_WITH_28147_CNT_IMIT",
	"\x00\x82": "TLS_GOSTR341094_WITH_NULL_GOSTR3411",
	"\x00\x83": "TLS_GOSTR341001_WITH_NULL_GOSTR3411",
	"\x00\x84": "TLS_RSA_WITH_CAMELLIA_256_CBC_SHA",
	"\x00\x85": "TLS_DH_DSS_WITH_CAMELLIA_256_CBC_SHA",
	"\x00\x86": "TLS_DH_RSA_WITH_CAMELLIA_256_CBC_SHA",
	"\x00\x87": "TLS_DHE_DSS_WITH_CAMELLIA_256_CBC_SHA",
	"\x00\x88": "TLS_DHE_RSA_WITH_CAMELLIA_256_CBC_SHA",
	"\x00\x89": "TLS_DH_anon_WITH_CAMELLIA_256_CBC_SHA",
	"\x00\x8A": "TLS_PSK_WITH_RC4_128_SHA",
	"\x00\x8B": "TLS_PSK_WITH_3DES_EDE_CBC_SHA",
	"\x00\x8C": "TLS_PSK_WITH_AES_128_CBC_SHA",
	"\x00\x8D": "TLS_PSK_WITH_AES_256_CBC_SHA",
	"\x00\x8E": "TLS_DHE_PSK_WITH_RC4_128_SHA",
	"\x00\x8F": "TLS_DHE_PSK_WITH_3DES_EDE_CBC_SHA",
	"\x00\x90": "TLS_DHE_PSK_WITH_AES_128_CBC_SHA",
	"\x00\x91": "TLS_DHE_PSK_WITH_AES_256_CBC_SHA",
	"\x00\x92": "TLS_RSA_PSK_WITH_RC4_128_SHA",
	"\x00\x93": "TLS_RSA_PSK_WITH_3DES_EDE_CBC_SHA",
	"\x00\x94": "TLS_RSA_PSK_WITH_AES_128_CBC_SHA",
	"\x00\x95": "TLS_RSA_PSK_WITH_AES_256_CBC_SHA",
	"\x00\x96": "TLS_RSA_WITH_SEED_CBC_SHA",
	"\x00\x97": "TLS_DH_DSS_WITH_SEED_CBC_SHA",
	"\x00\x98": "TLS_DH_RSA_WITH_SEED_CBC_SHA",
	"\x00\x99": "TLS_DHE_DSS_WITH_SEED_CBC_SHA",
	"\x00\x9A": "TLS_DHE_RSA_WITH_SEED_CBC_SHA",
	"\x00\x9B": "TLS_DH_anon_WITH_SEED_CBC_SHA",
	"\x00\x9C": "TLS_RSA_WITH_AES_128_GCM_SHA256",
	"\x00\x9D": "TLS_RSA_WITH_AES_256_GCM_SHA384",
	"\x00\x9E": "TLS_DHE_RSA_WITH_AES_128_GCM_SHA256",
	"\x00\x9F": "TLS_DHE_RSA_WITH_AES_256_GCM_SHA384",
	"\x00\xA0": "TLS_DH_RSA_WITH_AES_128_GCM_SHA256",
	"\x00\xA1": "TLS_DH_RSA_WITH_AES_256_GCM_SHA384",
	"\x00\xA2": "TLS_DHE_DSS_WITH_AES_128_GCM_SHA256",
	"\x00\xA3": "TLS_DHE_DSS_WITH_AES_256_GCM_SHA384",
	"\x00\xA4": "TLS_DH_DSS_WITH_AES_128_GCM_SHA256",
	"\x00\xA5": "TLS_DH_DSS_WITH_AES_256_GCM_SHA384",
	"\x00\xA6": "TLS_DH_anon_WITH_AES_128_GCM_SHA256",
	"\x00\xA7": "TLS_DH_anon_WITH_AES_256_GCM_SHA384",
	"\x00\xA8": "TLS_PSK_WITH_AES_128_GCM_SHA256",
	"\x00\xA9": "TLS_PSK_WITH_AES_256_GCM_SHA384",
	"\x00\xAA": "TLS_DHE_PSK_WITH_AES_128_GCM_SHA256",
	"\x00\xAB": "TLS_DHE_PSK_WITH_AES_256_GCM_SHA384",
	"\x00\xAC": "TLS_RSA_PSK_WITH_AES_128_GCM_SHA256",
	"\x00\xAD": "TLS_RSA_PSK_WITH_AES_256_GCM_SHA384",
	"\x00\xAE": "TLS_PSK_WITH_AES_128_CBC_SHA256",
	"\x00\xAF": "TLS_PSK_WITH_AES_256_CBC_SHA384",
	"\x00\xB0": "TLS_PSK_WITH_NULL_SHA256",
	"\x00\xB1": "TLS_PSK_WITH_NULL_SHA384",
	"\x00\xB2": "TLS_DHE_PSK_WITH_AES_128_CBC_SHA256",
	"\x00\xB3": "TLS_DHE_PSK_WITH_AES_256_CBC_SHA384",
	"\x00\xB4": "TLS_DHE_PSK_WITH_NULL_SHA256",
	"\x00\xB5": "TLS_DHE_PSK_WITH_NULL_SHA384",
	"\x00\xB6": "TLS_RSA_PSK_WITH_AES_128_CBC_SHA256",
	"\x00\xB7": "TLS_RSA_PSK_WITH_AES_256_CBC_SHA384",
	"\x00\xB8": "TLS_RSA_PSK_WITH_NULL_SHA256",
	"\x00\xB9": "TLS_RSA_PSK_WITH_NULL_SHA384",
	"\x00\xBA": "TLS_RSA_WITH_CAMELLIA_128_CBC_SHA256",
	"\x00\xBB": "TLS_DH_DSS_WITH_CAMELLIA_128_CBC_SHA256",
	"\x00\xBC": "TLS_DH_RSA_WITH_CAMELLIA_128_CBC_SHA256",
	"\x00\xBD": "TLS_DHE_DSS_WITH_CAMELLIA_128_CBC_SHA256",
	"\x00\xBE": "TLS_DHE_RSA_WITH_CAMELLIA_128_CBC_SHA256",
	"\x00\xBF": "TLS_DH_anon_WITH_CAMELLIA_128_CBC_SHA256",
	"\x00\xC0": "TLS_RSA_WITH_CAMELLIA_256_CBC_SHA256",
	"\x00\xC1": "TLS_DH_DSS_WITH_CAMELLIA_256_CBC_SHA256",
	"\x00\xC2": "TLS_DH_RSA_WITH_CAMELLIA_256_CBC_SHA256",
	"\x00\xC3": "TLS_DHE_DSS_WITH_CAMELLIA_256_CBC_SHA256",
	"\x00\xC4": "TLS_DHE_RSA_WITH_CAMELLIA_256_CBC_SHA256",
	"\x00\xC5": "TLS_DH_anon_WITH_CAMELLIA_256_CBC_SHA256",
	"\x00\xFF": "TLS_EMPTY_RENEGOTIATION_INFO_SCSV",
}

var TLS13_CIPHERS = map[string]string{
	// See https://tools.ietf.org/html/rfc8446#appendix-B.4
	"\x13\x01": "TLS_AES_128_GCM_SHA256",
	"\x13\x02": "TLS_AES_256_GCM_SHA384",
	"\x13\x03": "TLS_CHACHA20_POLY1305_SHA256",
	"\x13\x04": "TLS_AES_128_CCM_SHA256",
	"\x13\x05": "TLS_AES_128_CCM_8_SHA256",
}

var TLS_CIPHERS = map[string]string{
	"\xC0\x01": "TLS_ECDH_ECDSA_WITH_NULL_SHA",
	"\xC0\x02": "TLS_ECDH_ECDSA_WITH_RC4_128_SHA",
	"\xC0\x03": "TLS_ECDH_ECDSA_WITH_3DES_EDE_CBC_SHA",
	"\xC0\x04": "TLS_ECDH_ECDSA_WITH_AES_128_CBC_SHA",
	"\xC0\x05": "TLS_ECDH_ECDSA_WITH_AES_256_CBC_SHA",
	"\xC0\x06": "TLS_ECDHE_ECDSA_WITH_NULL_SHA",
	"\xC0\x07": "TLS_ECDHE_ECDSA_WITH_RC4_128_SHA",
	"\xC0\x08": "TLS_ECDHE_ECDSA_WITH_3DES_EDE_CBC_SHA",
	"\xC0\x09": "TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA",
	"\xC0\x0A": "TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA",
	"\xC0\x0B": "TLS_ECDH_RSA_WITH_NULL_SHA",
	"\xC0\x0C": "TLS_ECDH_RSA_WITH_RC4_128_SHA",
	"\xC0\x0D": "TLS_ECDH_RSA_WITH_3DES_EDE_CBC_SHA",
	"\xC0\x0E": "TLS_ECDH_RSA_WITH_AES_128_CBC_SHA",
	"\xC0\x0F": "TLS_ECDH_RSA_WITH_AES_256_CBC_SHA",
	"\xC0\x10": "TLS_ECDHE_RSA_WITH_NULL_SHA",
	"\xC0\x11": "TLS_ECDHE_RSA_WITH_RC4_128_SHA",
	"\xC0\x12": "TLS_ECDHE_RSA_WITH_3DES_EDE_CBC_SHA",
	"\xC0\x13": "TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA",
	"\xC0\x14": "TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA",
	"\xC0\x15": "TLS_ECDH_anon_WITH_NULL_SHA",
	"\xC0\x16": "TLS_ECDH_anon_WITH_RC4_128_SHA",
	"\xC0\x17": "TLS_ECDH_anon_WITH_3DES_EDE_CBC_SHA",
	"\xC0\x18": "TLS_ECDH_anon_WITH_AES_128_CBC_SHA",
	"\xC0\x19": "TLS_ECDH_anon_WITH_AES_256_CBC_SHA",
	"\xC0\x1A": "TLS_SRP_SHA_WITH_3DES_EDE_CBC_SHA",
	"\xC0\x1B": "TLS_SRP_SHA_RSA_WITH_3DES_EDE_CBC_SHA",
	"\xC0\x1C": "TLS_SRP_SHA_DSS_WITH_3DES_EDE_CBC_SHA",
	"\xC0\x1D": "TLS_SRP_SHA_WITH_AES_128_CBC_SHA",
	"\xC0\x1E": "TLS_SRP_SHA_RSA_WITH_AES_128_CBC_SHA",
	"\xC0\x1F": "TLS_SRP_SHA_DSS_WITH_AES_128_CBC_SHA",
	"\xC0\x20": "TLS_SRP_SHA_WITH_AES_256_CBC_SHA",
	"\xC0\x21": "TLS_SRP_SHA_RSA_WITH_AES_256_CBC_SHA",
	"\xC0\x22": "TLS_SRP_SHA_DSS_WITH_AES_256_CBC_SHA",
	"\xC0\x23": "TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA256",
	"\xC0\x24": "TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA384",
	"\xC0\x25": "TLS_ECDH_ECDSA_WITH_AES_128_CBC_SHA256",
	"\xC0\x26": "TLS_ECDH_ECDSA_WITH_AES_256_CBC_SHA384",
	"\xC0\x27": "TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA256",
	"\xC0\x28": "TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA384",
	"\xC0\x29": "TLS_ECDH_RSA_WITH_AES_128_CBC_SHA256",
	"\xC0\x2A": "TLS_ECDH_RSA_WITH_AES_256_CBC_SHA384",
	"\xC0\x2B": "TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256",
	"\xC0\x2C": "TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384",
	"\xC0\x2D": "TLS_ECDH_ECDSA_WITH_AES_128_GCM_SHA256",
	"\xC0\x2E": "TLS_ECDH_ECDSA_WITH_AES_256_GCM_SHA384",
	"\xC0\x2F": "TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256",
	"\xC0\x30": "TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384",
	"\xC0\x31": "TLS_ECDH_RSA_WITH_AES_128_GCM_SHA256",
	"\xC0\x32": "TLS_ECDH_RSA_WITH_AES_256_GCM_SHA384",
	"\xC0\x33": "TLS_ECDHE_PSK_WITH_RC4_128_SHA",
	"\xC0\x34": "TLS_ECDHE_PSK_WITH_3DES_EDE_CBC_SHA",
	"\xC0\x35": "TLS_ECDHE_PSK_WITH_AES_128_CBC_SHA",
	"\xC0\x36": "TLS_ECDHE_PSK_WITH_AES_256_CBC_SHA",
	"\xC0\x37": "TLS_ECDHE_PSK_WITH_AES_128_CBC_SHA256",
	"\xC0\x38": "TLS_ECDHE_PSK_WITH_AES_256_CBC_SHA384",
	"\xC0\x39": "TLS_ECDHE_PSK_WITH_NULL_SHA",
	"\xC0\x3A": "TLS_ECDHE_PSK_WITH_NULL_SHA256",
	"\xC0\x3B": "TLS_ECDHE_PSK_WITH_NULL_SHA384",
	"\xC0\x3C": "TLS_RSA_WITH_ARIA_128_CBC_SHA256",
	"\xC0\x3D": "TLS_RSA_WITH_ARIA_256_CBC_SHA384",
	"\xC0\x3E": "TLS_DH_DSS_WITH_ARIA_128_CBC_SHA256",
	"\xC0\x3F": "TLS_DH_DSS_WITH_ARIA_256_CBC_SHA384",
	"\xC0\x40": "TLS_DH_RSA_WITH_ARIA_128_CBC_SHA256",
	"\xC0\x41": "TLS_DH_RSA_WITH_ARIA_256_CBC_SHA384",
	"\xC0\x42": "TLS_DHE_DSS_WITH_ARIA_128_CBC_SHA256",
	"\xC0\x43": "TLS_DHE_DSS_WITH_ARIA_256_CBC_SHA384",
	"\xC0\x44": "TLS_DHE_RSA_WITH_ARIA_128_CBC_SHA256",
	"\xC0\x45": "TLS_DHE_RSA_WITH_ARIA_256_CBC_SHA384",
	"\xC0\x46": "TLS_DH_anon_WITH_ARIA_128_CBC_SHA256",
	"\xC0\x47": "TLS_DH_anon_WITH_ARIA_256_CBC_SHA384",
	"\xC0\x48": "TLS_ECDHE_ECDSA_WITH_ARIA_128_CBC_SHA256",
	"\xC0\x49": "TLS_ECDHE_ECDSA_WITH_ARIA_256_CBC_SHA384",
	"\xC0\x4A": "TLS_ECDH_ECDSA_WITH_ARIA_128_CBC_SHA256",
	"\xC0\x4B": "TLS_ECDH_ECDSA_WITH_ARIA_256_CBC_SHA384",
	"\xC0\x4C": "TLS_ECDHE_RSA_WITH_ARIA_128_CBC_SHA256",
	"\xC0\x4D": "TLS_ECDHE_RSA_WITH_ARIA_256_CBC_SHA384",
	"\xC0\x4E": "TLS_ECDH_RSA_WITH_ARIA_128_CBC_SHA256",
	"\xC0\x4F": "TLS_ECDH_RSA_WITH_ARIA_256_CBC_SHA384",
	"\xC0\x50": "TLS_RSA_WITH_ARIA_128_GCM_SHA256",
	"\xC0\x51": "TLS_RSA_WITH_ARIA_256_GCM_SHA384",
	"\xC0\x52": "TLS_DHE_RSA_WITH_ARIA_128_GCM_SHA256",
	"\xC0\x53": "TLS_DHE_RSA_WITH_ARIA_256_GCM_SHA384",
	"\xC0\x54": "TLS_DH_RSA_WITH_ARIA_128_GCM_SHA256",
	"\xC0\x55": "TLS_DH_RSA_WITH_ARIA_256_GCM_SHA384",
	"\xC0\x56": "TLS_DHE_DSS_WITH_ARIA_128_GCM_SHA256",
	"\xC0\x57": "TLS_DHE_DSS_WITH_ARIA_256_GCM_SHA384",
	"\xC0\x58": "TLS_DH_DSS_WITH_ARIA_128_GCM_SHA256",
	"\xC0\x59": "TLS_DH_DSS_WITH_ARIA_256_GCM_SHA384",
	"\xC0\x5A": "TLS_DH_anon_WITH_ARIA_128_GCM_SHA256",
	"\xC0\x5B": "TLS_DH_anon_WITH_ARIA_256_GCM_SHA384",
	"\xC0\x5C": "TLS_ECDHE_ECDSA_WITH_ARIA_128_GCM_SHA256",
	"\xC0\x5D": "TLS_ECDHE_ECDSA_WITH_ARIA_256_GCM_SHA384",
	"\xC0\x5E": "TLS_ECDH_ECDSA_WITH_ARIA_128_GCM_SHA256",
	"\xC0\x5F": "TLS_ECDH_ECDSA_WITH_ARIA_256_GCM_SHA384",
	"\xC0\x60": "TLS_ECDHE_RSA_WITH_ARIA_128_GCM_SHA256",
	"\xC0\x61": "TLS_ECDHE_RSA_WITH_ARIA_256_GCM_SHA384",
	"\xC0\x62": "TLS_ECDH_RSA_WITH_ARIA_128_GCM_SHA256",
	"\xC0\x63": "TLS_ECDH_RSA_WITH_ARIA_256_GCM_SHA384",
	"\xC0\x64": "TLS_PSK_WITH_ARIA_128_CBC_SHA256",
	"\xC0\x65": "TLS_PSK_WITH_ARIA_256_CBC_SHA384",
	"\xC0\x66": "TLS_DHE_PSK_WITH_ARIA_128_CBC_SHA256",
	"\xC0\x67": "TLS_DHE_PSK_WITH_ARIA_256_CBC_SHA384",
	"\xC0\x68": "TLS_RSA_PSK_WITH_ARIA_128_CBC_SHA256",
	"\xC0\x69": "TLS_RSA_PSK_WITH_ARIA_256_CBC_SHA384",
	"\xC0\x6A": "TLS_PSK_WITH_ARIA_128_GCM_SHA256",
	"\xC0\x6B": "TLS_PSK_WITH_ARIA_256_GCM_SHA384",
	"\xC0\x6C": "TLS_DHE_PSK_WITH_ARIA_128_GCM_SHA256",
	"\xC0\x6D": "TLS_DHE_PSK_WITH_ARIA_256_GCM_SHA384",
	"\xC0\x6E": "TLS_RSA_PSK_WITH_ARIA_128_GCM_SHA256",
	"\xC0\x6F": "TLS_RSA_PSK_WITH_ARIA_256_GCM_SHA384",
	"\xC0\x70": "TLS_ECDHE_PSK_WITH_ARIA_128_CBC_SHA256",
	"\xC0\x71": "TLS_ECDHE_PSK_WITH_ARIA_256_CBC_SHA384",
	"\xC0\x72": "TLS_ECDHE_ECDSA_WITH_CAMELLIA_128_CBC_SHA256",
	"\xC0\x73": "TLS_ECDHE_ECDSA_WITH_CAMELLIA_256_CBC_SHA384",
	"\xC0\x74": "TLS_ECDH_ECDSA_WITH_CAMELLIA_128_CBC_SHA256",
	"\xC0\x75": "TLS_ECDH_ECDSA_WITH_CAMELLIA_256_CBC_SHA384",
	"\xC0\x76": "TLS_ECDHE_RSA_WITH_CAMELLIA_128_CBC_SHA256",
	"\xC0\x77": "TLS_ECDHE_RSA_WITH_CAMELLIA_256_CBC_SHA384",
	"\xC0\x78": "TLS_ECDH_RSA_WITH_CAMELLIA_128_CBC_SHA256",
	"\xC0\x79": "TLS_ECDH_RSA_WITH_CAMELLIA_256_CBC_SHA384",
	"\xC0\x7A": "TLS_RSA_WITH_CAMELLIA_128_GCM_SHA256",
	"\xC0\x7B": "TLS_RSA_WITH_CAMELLIA_256_GCM_SHA384",
	"\xC0\x7C": "TLS_DHE_RSA_WITH_CAMELLIA_128_GCM_SHA256",
	"\xC0\x7D": "TLS_DHE_RSA_WITH_CAMELLIA_256_GCM_SHA384",
	"\xC0\x7E": "TLS_DH_RSA_WITH_CAMELLIA_128_GCM_SHA256",
	"\xC0\x7F": "TLS_DH_RSA_WITH_CAMELLIA_256_GCM_SHA384",
	"\xC0\x80": "TLS_DHE_DSS_WITH_CAMELLIA_128_GCM_SHA256",
	"\xC0\x81": "TLS_DHE_DSS_WITH_CAMELLIA_256_GCM_SHA384",
	"\xC0\x82": "TLS_DH_DSS_WITH_CAMELLIA_128_GCM_SHA256",
	"\xC0\x83": "TLS_DH_DSS_WITH_CAMELLIA_256_GCM_SHA384",
	"\xC0\x84": "TLS_DH_anon_WITH_CAMELLIA_128_GCM_SHA256",
	"\xC0\x85": "TLS_DH_anon_WITH_CAMELLIA_256_GCM_SHA384",
	"\xC0\x86": "TLS_ECDHE_ECDSA_WITH_CAMELLIA_128_GCM_SHA256",
	"\xC0\x87": "TLS_ECDHE_ECDSA_WITH_CAMELLIA_256_GCM_SHA384",
	"\xC0\x88": "TLS_ECDH_ECDSA_WITH_CAMELLIA_128_GCM_SHA256",
	"\xC0\x89": "TLS_ECDH_ECDSA_WITH_CAMELLIA_256_GCM_SHA384",
	"\xC0\x8A": "TLS_ECDHE_RSA_WITH_CAMELLIA_128_GCM_SHA256",
	"\xC0\x8B": "TLS_ECDHE_RSA_WITH_CAMELLIA_256_GCM_SHA384",
	"\xC0\x8C": "TLS_ECDH_RSA_WITH_CAMELLIA_128_GCM_SHA256",
	"\xC0\x8D": "TLS_ECDH_RSA_WITH_CAMELLIA_256_GCM_SHA384",
	"\xC0\x8E": "TLS_PSK_WITH_CAMELLIA_128_GCM_SHA256",
	"\xC0\x8F": "TLS_PSK_WITH_CAMELLIA_256_GCM_SHA384",
	"\xC0\x90": "TLS_DHE_PSK_WITH_CAMELLIA_128_GCM_SHA256",
	"\xC0\x91": "TLS_DHE_PSK_WITH_CAMELLIA_256_GCM_SHA384",
	"\xC0\x92": "TLS_RSA_PSK_WITH_CAMELLIA_128_GCM_SHA256",
	"\xC0\x93": "TLS_RSA_PSK_WITH_CAMELLIA_256_GCM_SHA384",
	"\xC0\x94": "TLS_PSK_WITH_CAMELLIA_128_CBC_SHA256",
	"\xC0\x95": "TLS_PSK_WITH_CAMELLIA_256_CBC_SHA384",
	"\xC0\x96": "TLS_DHE_PSK_WITH_CAMELLIA_128_CBC_SHA256",
	"\xC0\x97": "TLS_DHE_PSK_WITH_CAMELLIA_256_CBC_SHA384",
	"\xC0\x98": "TLS_RSA_PSK_WITH_CAMELLIA_128_CBC_SHA256",
	"\xC0\x99": "TLS_RSA_PSK_WITH_CAMELLIA_256_CBC_SHA384",
	"\xC0\x9A": "TLS_ECDHE_PSK_WITH_CAMELLIA_128_CBC_SHA256",
	"\xC0\x9B": "TLS_ECDHE_PSK_WITH_CAMELLIA_256_CBC_SHA384",
	"\xC0\x9C": "TLS_RSA_WITH_AES_128_CCM",
	"\xC0\x9D": "TLS_RSA_WITH_AES_256_CCM",
	"\xC0\x9E": "TLS_DHE_RSA_WITH_AES_128_CCM",
	"\xC0\x9F": "TLS_DHE_RSA_WITH_AES_256_CCM",
	"\xC0\xA0": "TLS_RSA_WITH_AES_128_CCM_8",
	"\xC0\xA1": "TLS_RSA_WITH_AES_256_CCM_8",
	"\xC0\xA2": "TLS_DHE_RSA_WITH_AES_128_CCM_8",
	"\xC0\xA3": "TLS_DHE_RSA_WITH_AES_256_CCM_8",
	"\xC0\xA4": "TLS_PSK_WITH_AES_128_CCM",
	"\xC0\xA5": "TLS_PSK_WITH_AES_256_CCM",
	"\xC0\xA6": "TLS_DHE_PSK_WITH_AES_128_CCM",
	"\xC0\xA7": "TLS_DHE_PSK_WITH_AES_256_CCM",
	"\xC0\xA8": "TLS_PSK_WITH_AES_128_CCM_8",
	"\xC0\xA9": "TLS_PSK_WITH_AES_256_CCM_8",
	"\xC0\xAA": "TLS_PSK_DHE_WITH_AES_128_CCM_8",
	"\xC0\xAB": "TLS_PSK_DHE_WITH_AES_256_CCM_8",
	"\xC0\xAC": "TLS_ECDHE_ECDSA_WITH_AES_128_CCM",
	"\xC0\xAD": "TLS_ECDHE_ECDSA_WITH_AES_256_CCM",
	"\xC0\xAE": "TLS_ECDHE_ECDSA_WITH_AES_128_CCM_8",
	"\xC0\xAF": "TLS_ECDHE_ECDSA_WITH_AES_256_CCM_8",
	"\xCC\xA8": "TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305",
	"\xCC\xA9": "TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305",
	"\xCC\xAA": "TLS_DHE_RSA_WITH_CHACHA20_POLY1305",
	"\xCC\xAB": "TLS_PSK_WITH_CHACHA20_POLY1305",
	"\xCC\xAC": "TLS_ECDHE_PSK_WITH_CHACHA20_POLY1305",
	"\xCC\xAD": "TLS_DHE_PSK_WITH_CHACHA20_POLY1305",
	"\xCC\xAE": "TLS_RSA_PSK_WITH_CHACHA20_POLY1305",
	"\xCC\x13": "OLD_TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305_SHA256",
	"\xCC\x14": "OLD_TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305_SHA256",
	"\xCC\x15": "OLD_TLS_DHE_RSA_WITH_CHACHA20_POLY1305_SHA256",
}
