package resources

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"io"
	"math/rand"
	"net"
	"regexp"
	"strconv"

	"github.com/cockroachdb/errors"
	"go.mondoo.io/mondoo/lumi"
)

var reTarget = regexp.MustCompile("([^/:]+?)(:\\d+)?$")

func (s *lumiSsl) init(args *lumi.Args) (*lumi.Args, Ssl, error) {
	if _target, ok := (*args)["target"]; ok {
		target := _target.(string)
		m := reTarget.FindStringSubmatch(target)
		if len(m) == 0 {
			return nil, nil, errors.New("target must be provided in the form of: tcp://target:port, udp://target:port, or target:port (defaults to tcp)")
		}

		proto := "tcp"

		var port int64 = 443
		if len(m[2]) != 0 {
			rawPort, err := strconv.ParseUint(m[2][1:], 10, 64)
			if err != nil {
				return nil, nil, errors.New("failed to parse port: " + m[2])
			}
			port = int64(rawPort)
		}

		socket, err := s.Runtime.CreateResource("socket",
			"protocol", proto,
			"port", port,
			"address", m[1],
		)
		if err != nil {
			return nil, nil, err
		}

		(*args)["socket"] = socket
		delete(*args, "target")
	}

	return args, nil, nil
}

func (s *lumiSsl) id() (string, error) {
	socket, err := s.Socket()
	if err != nil {
		return "", err
	}

	return "ssl+" + socket.LumiResource().Id, nil
}

func (s *lumiSsl) GetParams(socket Socket) (map[string]interface{}, error) {
	host, err := socket.Address()
	if err != nil {
		return nil, err
	}

	port, err := socket.Port()
	if err != nil {
		return nil, err
	}

	proto, err := socket.Protocol()
	if err != nil {
		return nil, err
	}

	capabilities, err := probeSSL(proto, host, int(port), []string{})
	if err != nil {
		return nil, err
	}

	res := map[string]interface{}{}

	lists := map[string][]string{
		"errors": capabilities.errors,
	}
	for field, data := range lists {
		v := make([]interface{}, len(data))
		for i := range data {
			v[i] = data[i]
		}
		res[field] = v
	}

	maps := map[string]map[string]bool{
		"versions": capabilities.versions,
		"ciphers":  capabilities.ciphers,
	}
	for field, data := range maps {
		v := make(map[string]interface{}, len(data))
		for k, vv := range data {
			v[k] = vv
		}
		res[field] = v
	}

	return res, nil
}

func (s *lumiSsl) GetVersions(params map[string]interface{}) ([]interface{}, error) {
	raw, ok := params["versions"]
	if !ok {
		return []interface{}{}, nil
	}

	data := raw.(map[string]interface{})
	res := []interface{}{}
	for k, v := range data {
		if v.(bool) {
			res = append(res, k)
		}
	}

	return res, nil
}

func (s *lumiSsl) GetCiphers(params map[string]interface{}) ([]interface{}, error) {
	raw, ok := params["ciphers"]
	if !ok {
		return []interface{}{}, nil
	}

	data := raw.(map[string]interface{})
	res := []interface{}{}
	for k, v := range data {
		if v.(bool) {
			res = append(res, k)
		}
	}

	return res, nil
}

// shake that SSL

type sslCapabilities struct {
	versions map[string]bool
	ciphers  map[string]bool
	errors   []string
}

func probeSSL(proto string, host string, port int, versions []string) (*sslCapabilities, error) {
	res := sslCapabilities{
		versions: map[string]bool{},
		ciphers:  map[string]bool{},
	}
	target := host + ":" + strconv.Itoa(port)

	if len(versions) == 0 {
		versions = []string{"ssl3", "tls1.0", "tls1.1", "tls1.2", "tls1.3"}
	}

	for i := range versions {
		version := versions[i]
		if err := helloSSL(proto, target, version, &res); err != nil {
			return nil, err
		}
	}

	return &res, nil
}

func helloSSL(proto string, target string, version string, res *sslCapabilities) error {
	conn, err := net.Dial(proto, target)
	if err != nil {
		return errors.Wrap(err, "failed to connect to target")
	}
	defer conn.Close()

	msg, err := helloSSLMsg(version, func(cipher string) bool {
		return true
	})
	if err != nil {
		return err
	}

	_, err = conn.Write(msg)
	if err != nil {
		return errors.Wrap(err, "failed to send SSL hello")
	}

	return parseHello(conn, res)
}

func parseHello(conn net.Conn, res *sslCapabilities) error {
	reader := bufio.NewReader(conn)
	b := make([]byte, 5)
	_, err := io.ReadFull(reader, b)
	if err != nil {
		return err
	}

	if b[0] == CONTENT_TYPE_Alert {
		version := VERSIONS_LOOKUP[string(b[1:3])]
		msgLen := bytes2int(b[3:5])

		if msgLen == 0 {
			return errors.New("SSL alert (without message body)")
		}
		if msgLen > 1<<20 {
			return errors.New("SSL alert (with too large message body)")
		}

		b = make([]byte, msgLen)
		_, err = io.ReadFull(reader, b)
		if err != nil {
			return errors.Wrap(err, "failed to read SSL alert body")
		}

		var severity string
		switch b[0] {
		case '\x01':
			severity = "Warning"
		case '\x02':
			severity = "Fatal"
		default:
			severity = "Unknown"
		}

		description, ok := ALERT_DESCRIPTIONS[b[1]]
		if !ok {
			description = "cannot find description"
		}

		if version == "ssl3" && description == "HANDSHAKE_FAILURE" {
			// Note: it's a little fuzzy here, since we don't know if the protocol
			// version is unsupported or just its ciphers. So we check if we found
			// it previously and if so, don't add it to the list of unsupported
			// versions.
			if _, ok := res.versions["ssl3"]; !ok {
				res.versions["ssl3"] = false
			}
		} else if version != "ssl3" && description == "PROTOCOL_VERSION" {
			// here we know the TLS version is not supported
			res.versions[version] = false
		} else {
			res.errors = append(res.errors, "failed to connect via "+version+": "+severity+" - "+description)
		}

		return nil
	}

	if b[0] != CONTENT_TYPE_Handshake {
		return errors.New("unhandled SSL response (was expecting a handshake, got '0x" + hex.EncodeToString(b[0:1]) + "' )")
	}

	version := VERSIONS_LOOKUP[string(b[1:3])]
	msgLen := bytes2int(b[3:5])

	b = make([]byte, msgLen)
	_, err = io.ReadFull(reader, b)
	if err != nil {
		return errors.Wrap(err, "failed to read SSL handshake body")
	}

	idx := 0

	idx += 1 + 3 + 2 + 32
	// handshake type (1)
	// len (3)
	// handshake tls version (2)
	// random (32)

	sessionIDlen := byte1int(b[idx])
	idx += 1
	idx += sessionIDlen

	cipherID := string(b[idx : idx+2])
	idx += 2
	cipher, ok := ALL_CIPHERS[cipherID]
	if !ok {
		res.ciphers["unknown"] = true
	} else {
		res.ciphers[cipher] = true
	}

	// TLS 1.3 pretends to be TLS 1.2 in the preceeding headers for
	// compatibility. To correctly identify it, we have to look at
	// any Supported Versions extensions that the server sent us.

	// compression method (which should be set to null)
	idx += 1

	allExtLen := bytes2int(b[idx : idx+2])
	idx += 2

	for allExtLen > 0 && idx < len(b) {
		extType := string(b[idx : idx+2])
		extLen := bytes2int(b[idx+2 : idx+4])
		extData := string(b[idx+4 : idx+4+extLen])

		allExtLen -= 4 + extLen
		idx += 4 + extLen

		if extType == "\x00\x2b" && extData == "\x03\x04" {
			version = "tls1.3"
			break
		}
	}

	res.versions[version] = true

	return nil
}

func filterCiphers(org map[string]string, f func(cipher string) bool) []byte {
	var res bytes.Buffer
	for k, v := range org {
		if f(v) {
			res.WriteString(k)
		}
	}
	return res.Bytes()
}

func helloSSLMsg(version string, ciphersFilter func(cipher string) bool) ([]byte, error) {
	var ciphers []byte
	var extensions []byte

	switch version {
	case "ssl3":
		regular := filterCiphers(SSL3_CIPHERS, ciphersFilter)
		fips := filterCiphers(SSL_FIPS_CIPHERS, ciphersFilter)
		ciphers = append(regular, fips...)

	case "tls1.0", "tls1.1", "tls1.2":
		ciphers = filterCiphers(TLS_CIPHERS, ciphersFilter)

		var ext bytes.Buffer
		// add heartbeat
		ext.WriteString("\x00\x0f\x00\x01\x01")
		// add signature_algorithms
		ext.WriteString("\x00\x0d\x00\x14\x00\x12\x04\x03\x08\x04\x04\x01\x05\x03\x08\x05\x05\x01\x08\x06\x06\x01\x02\x01")
		// add ec_points_format
		ext.WriteString("\x00\x0b\x00\x02\x01\x00")
		// add elliptic_curve
		ext.WriteString("\x00\x0a\x00\x0a\x00\x08\xfa\xfa\x00\x1d\x00\x17\x00\x18")

		extensions = ext.Bytes()

	case "tls1.3":
		ciphers = filterCiphers(TLS13_CIPHERS, ciphersFilter)

		var ext bytes.Buffer
		// TLSv1.3 Supported Versions extension
		ext.WriteString("\x00\x2b\x00\x03\x02\x03\x04")
		// add signature_algorithms
		ext.WriteString("\x00\x0d\x00\x14\x00\x12\x04\x03\x08\x04\x04\x01\x05\x03\x08\x05\x05\x01\x08\x06\x06\x01\x02\x01")
		// add supported groups extension
		ext.WriteString("\x00\x0a\x00\x08\x00\x06\x00\x1d\x00\x17\x00\x18")

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
		ext.WriteString("\x00\x33\x00\x26\x00\x24\x00\x1d\x00\x20")
		ext.WriteString(publicKey)
		extensions = ext.Bytes()

	default:
		return nil, errors.New("unsupported SSL version: " + version)
	}

	return constructSSLHello(version, ciphers, extensions), nil
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

func constructSSLHello(version string, ciphers []byte, extensions []byte) []byte {
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

	return constructSSLMsg(CONTENT_TYPE_Handshake, core, []byte(VERSIONS[version]))
}

func constructSSLMsg(contentType byte, content []byte, version []byte) []byte {
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

	for k, v := range SSL2_CIPHERS {
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
	HANDSHAKE_TYPE_Certificate        byte = '\x11'
	HANDSHAKE_TYPE_ServerKeyExchange  byte = '\x12'
	HANDSHAKE_TYPE_CertificateRequest byte = '\x13'
	HANDSHAKE_TYPE_ServerHelloDone    byte = '\x14'
	HANDSHAKE_TYPE_CertificateVerify  byte = '\x15'
	HANDSHAKE_TYPE_ClientKeyExchange  byte = '\x16'
	HANDSHAKE_TYPE_Finished           byte = '\x20'
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
