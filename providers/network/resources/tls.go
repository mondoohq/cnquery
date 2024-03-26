// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"crypto/tls"
	"crypto/x509"
	"net"
	"regexp"
	"strconv"
	"sync"
	"time"

	"github.com/cockroachdb/errors"
	"go.mondoo.com/cnquery/v10/llx"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v10/providers/core/resources/regex"
	"go.mondoo.com/cnquery/v10/providers/network/connection"
	"go.mondoo.com/cnquery/v10/providers/network/resources/certificates"
	"go.mondoo.com/cnquery/v10/providers/network/resources/tlsshake"
	"go.mondoo.com/cnquery/v10/types"
)

var reTarget = regexp.MustCompile("([^/:]+?)(:\\d+)?$")

var rexUrlDomain = regexp.MustCompile(regex.UrlDomain)

// Returns the connection's port adjusted for TLS.
// If no port is set, we estimate what it might be from the scheme.
// If that doesn't help, we set it to 443.
func connTlsPort(conn *connection.HostConnection) int64 {
	if conn.Conf.Port != 0 {
		return int64(conn.Conf.Port)
	}

	if conn.Conf.Runtime == "" {
		return 443
	}

	port := CommonPorts[conn.Conf.Runtime]
	if port == 0 {
		return 443
	}
	return int64(port)
}

func initTls(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	// if the socket is set already, we have nothing else to do
	if _, ok := args["socket"]; ok {
		return args, nil, nil
	}

	conn := runtime.Connection.(*connection.HostConnection)
	port := connTlsPort(conn)

	if target, ok := args["target"]; ok {
		m := reTarget.FindStringSubmatch(target.Value.(string))
		if len(m) == 0 {
			return nil, nil, errors.New("target must be provided in the form of: tcp://target:port, udp://target:port, or target:port (defaults to tcp)")
		}

		proto := "tcp"
		// If the port is set as part of the target string, try to parse it
		// from here.
		if len(m[2]) != 0 {
			rawPort, err := strconv.ParseUint(m[2][1:], 10, 64)
			if err != nil {
				return nil, nil, errors.New("failed to parse port: " + m[2])
			}
			port = int64(rawPort)
		}

		address := m[1]
		domainName := ""
		if rexUrlDomain.MatchString(address) {
			domainName = address
		}

		socket, err := CreateResource(runtime, "socket", map[string]*llx.RawData{
			"protocol": llx.StringData(proto),
			"port":     llx.IntData(port),
			"address":  llx.StringData(address),
		})
		if err != nil {
			return nil, nil, err
		}

		args["socket"] = llx.ResourceData(socket, "socket")
		args["domainName"] = llx.StringData(domainName)
		delete(args, "target")

	} else {
		socket, err := CreateResource(runtime, "socket", map[string]*llx.RawData{
			"protocol": llx.StringData("tcp"),
			"port":     llx.IntData(port),
			"address":  llx.StringData(conn.Conf.Host),
		})
		if err != nil {
			return nil, nil, err
		}

		args["socket"] = llx.ResourceData(socket, "socket")
		args["domainName"] = llx.StringData(conn.Conf.Host)
	}

	return args, nil, nil
}

type mqlTlsInternal struct {
	lock sync.Mutex
}

func (s *mqlTls) id() (string, error) {
	return "tls+" + s.Socket.Data.__id, nil
}

func parseCertificates(runtime *plugin.Runtime, domainName string, certificateList []*x509.Certificate, revocations map[string]*tlsshake.Revocation) ([]interface{}, []string, error) {
	res := make([]interface{}, len(certificateList))
	errors := []string{}

	verified := false
	if len(certificateList) != 0 {
		intermediates := x509.NewCertPool()
		for i := 1; i < len(certificateList); i++ {
			intermediates.AddCert(certificateList[i])
		}

		verifyCerts, err := certificateList[0].Verify(x509.VerifyOptions{
			DNSName:       domainName,
			Intermediates: intermediates,
		})
		if err != nil {
			errors = append(errors, "Failed to verify certificate chain for "+certificateList[0].Subject.String())
		}

		if len(verifyCerts) != 0 {
			verified = verifyCerts[0][0].Equal(certificateList[0])
		}
	}

	for i := range certificateList {
		cert := certificateList[i]

		var isRevoked bool
		var revokedAt time.Time
		revocation, ok := revocations[string(cert.Signature)]
		if ok {
			if revocation == nil {
				isRevoked = false
				revokedAt = llx.NeverFutureTime
			} else {
				isRevoked = true
				revokedAt = revocation.At
			}
		}

		pem, err := certificates.EncodeCertAsPEM(cert)

		if err != nil {
			return nil, nil, err
		}

		raw, err := CreateResource(runtime, "certificate", map[string]*llx.RawData{
			"pem": llx.StringData(string(pem)),
			// NOTE: if we do not set the hash here, it will generate the cache content before we can store it
			// we are using the hashs for the id, therefore it is required during creation
			"fingerprints": llx.MapData(certificates.Fingerprints(cert), types.String),
			"isRevoked":    llx.BoolData(isRevoked),
			"revokedAt":    llx.TimeData(revokedAt),
			"isVerified":   llx.BoolData(verified),
		})
		if err != nil {
			return nil, nil, err
		}

		// store parsed object with resource
		mqlCert := raw.(*mqlCertificate)
		mqlCert.cert = plugin.TValue[*x509.Certificate]{Data: cert, State: plugin.StateIsSet}

		res[i] = mqlCert
	}

	return res, errors, nil
}

func (s *mqlTls) params(socket *mqlSocket, domainName string) (map[string]interface{}, error) {
	s.lock.Lock()
	defer s.lock.Unlock()

	host := socket.Address.Data
	port := socket.Port.Data
	proto := socket.Protocol.Data

	tester := tlsshake.New(proto, domainName, host, int(port))
	if err := tester.Test(tlsshake.DefaultScanConfig()); err != nil {
		if errors.Is(err, tlsshake.ErrFailedToConnect) || errors.Is(err, tlsshake.ErrFailedToTlsResponse) {
			s.Params.State = plugin.StateIsSet | plugin.StateIsNull
			return nil, nil
		}
		return nil, err
	}

	res := map[string]interface{}{}
	findings := tester.Findings

	lists := map[string][]string{
		"errors": findings.Errors,
	}
	for field, data := range lists {
		v := make([]interface{}, len(data))
		for i := range data {
			v[i] = data[i]
		}
		res[field] = v
	}

	maps := map[string]map[string]bool{
		"versions":   findings.Versions,
		"ciphers":    findings.Ciphers,
		"extensions": findings.Extensions,
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

func (s *mqlTls) versions(params interface{}) ([]interface{}, error) {
	paramsM, ok := params.(map[string]interface{})
	// only happens in case of unexpected errors or null
	if !ok {
		s.Versions.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}

	raw, ok := paramsM["versions"]
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

func (s *mqlTls) ciphers(params interface{}) ([]interface{}, error) {
	paramsM, ok := params.(map[string]interface{})
	// only happens in case of unexpected errors or null
	if !ok {
		s.Ciphers.State = plugin.StateIsSet | plugin.StateIsNull
		return []interface{}{}, nil
	}

	raw, ok := paramsM["ciphers"]
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

func (s *mqlTls) extensions(params interface{}) ([]interface{}, error) {
	paramsM, ok := params.(map[string]interface{})
	// only happens in case of unexpected errors or null
	if !ok {
		s.Extensions.State = plugin.StateIsSet | plugin.StateIsNull
		return []interface{}{}, nil
	}

	raw, ok := paramsM["extensions"]
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

func gatherTlsCertificates(host, port, domainName string) ([]*x509.Certificate, []*x509.Certificate, error) {
	isSNIcert := map[string]struct{}{}
	conn, err := tls.Dial("tcp", net.JoinHostPort(host, port), &tls.Config{
		InsecureSkipVerify: true,
		ServerName:         domainName,
	})
	if err != nil {
		return nil, nil, err
	}
	defer conn.Close()

	// Get the ConnectionState where we can find x509.Certificate(s)
	sniCerts := conn.ConnectionState().PeerCertificates
	for _, sniCerts := range sniCerts {
		isSNIcert[sniCerts.SerialNumber.String()] = struct{}{}
	}

	nonSniCerts := []*x509.Certificate{}
	nonSniConn, err := tls.Dial("tcp", net.JoinHostPort(host, port), &tls.Config{
		InsecureSkipVerify: true,
	})
	if err != nil {
		return nil, nil, err
	}
	defer nonSniConn.Close()
	potentialNonSniCerts := nonSniConn.ConnectionState()
	for _, nonSniCert := range potentialNonSniCerts.PeerCertificates {
		if _, ok := isSNIcert[nonSniCert.SerialNumber.String()]; !ok {
			nonSniCerts = append(nonSniCerts, nonSniCert)
		}
	}

	return sniCerts, nonSniCerts, nil
}

func (s *mqlTls) populateCertificates(socket *mqlSocket, domainName string) error {
	host := socket.Address.Data
	port := socket.Port.Data

	certs, nonSniCerts, err := gatherTlsCertificates(host, strconv.FormatInt(port, 10), domainName)
	if err != nil {
		s.Certificates = plugin.TValue[[]interface{}]{Error: err, State: plugin.StateIsSet}
		s.NonSniCertificates = plugin.TValue[[]interface{}]{Error: err, State: plugin.StateIsSet}
		return err
	}

	mqlCerts, _, err := parseCertificates(s.MqlRuntime, domainName, certs, map[string]*tlsshake.Revocation{})
	if err != nil {
		s.Certificates = plugin.TValue[[]interface{}]{Error: err, State: plugin.StateIsSet}
	} else {
		s.Certificates = plugin.TValue[[]interface{}]{Data: mqlCerts, State: plugin.StateIsSet}
	}

	mqlNonSniCerts, _, err := parseCertificates(s.MqlRuntime, domainName, nonSniCerts, map[string]*tlsshake.Revocation{})
	if err != nil {
		s.NonSniCertificates = plugin.TValue[[]interface{}]{Error: err, State: plugin.StateIsSet}
	} else {
		s.NonSniCertificates = plugin.TValue[[]interface{}]{Data: mqlNonSniCerts, State: plugin.StateIsSet}
	}
	return nil
}

func (s *mqlTls) certificates(socket *mqlSocket, domainName string) ([]interface{}, error) {
	return nil, s.populateCertificates(socket, domainName)
}

func (s *mqlTls) nonSniCertificates(socket *mqlSocket, domainName string) ([]interface{}, error) {
	return nil, s.populateCertificates(socket, domainName)
}
