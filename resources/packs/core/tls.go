package core

import (
	"crypto/x509"
	"regexp"
	"strconv"

	"errors"
	"go.mondoo.com/cnquery/llx"
	"go.mondoo.com/cnquery/motor/providers/network"
	"go.mondoo.com/cnquery/resources"
	"go.mondoo.com/cnquery/resources/packs/core/certificates"
	"go.mondoo.com/cnquery/resources/packs/core/tlsshake"
)

var reTarget = regexp.MustCompile("([^/:]+?)(:\\d+)?$")

func (s *mqlTls) init(args *resources.Args) (*resources.Args, Tls, error) {
	// if the socket is set already, we have nothing else to do
	if _, ok := (*args)["socket"]; ok {
		return args, nil, nil
	}

	var fqdn string
	var port int64

	if transport, ok := s.MotorRuntime.Motor.Provider.(*network.Provider); ok {
		fqdn = transport.FQDN
		port = int64(transport.Port)
		if port == 0 {
			port = 443
		}
	}

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

		address := m[1]
		domainName := ""
		if rexUrlDomain.MatchString(address) {
			domainName = address
		}

		socket, err := s.MotorRuntime.CreateResource("socket",
			"protocol", proto,
			"port", port,
			"address", address,
		)
		if err != nil {
			return nil, nil, err
		}

		(*args)["socket"] = socket
		(*args)["domainName"] = domainName
		delete(*args, "target")

	} else {
		socket, err := s.MotorRuntime.CreateResource("socket",
			"protocol", "tcp",
			"port", port,
			"address", fqdn,
		)
		if err != nil {
			return nil, nil, err
		}

		(*args)["socket"] = socket
		(*args)["domainName"] = fqdn
	}

	return args, nil, nil
}

func (s *mqlTls) id() (string, error) {
	socket, err := s.Socket()
	if err != nil {
		return "", err
	}

	return "tls+" + socket.MqlResource().Id, nil
}

func parseCertificates(runtime *resources.Runtime, domainName string, findings *tlsshake.Findings, certificateList []*x509.Certificate) ([]interface{}, error) {
	res := make([]interface{}, len(certificateList))

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
			findings.Errors = append(findings.Errors, "Failed to verify certificate chain for "+certificateList[0].Subject.String())
		}

		if len(verifyCerts) != 0 {
			verified = verifyCerts[0][0].Equal(certificateList[0])
		}
	}

	for i := range certificateList {
		cert := certificateList[i]

		var isRevoked interface{}
		var revokedAt interface{}
		revocation, ok := findings.Revocations[string(cert.Signature)]
		if ok {
			if revocation == nil {
				isRevoked = false
				revokedAt = &llx.NeverFutureTime
			} else {
				isRevoked = true
				revokedAt = &revocation.At
			}
		}

		certdata, err := certificates.EncodeCertAsPEM(cert)
		if err != nil {
			return nil, err
		}

		raw, err := runtime.CreateResource("certificate",
			"pem", string(certdata),
			// NOTE: if we do not set the hash here, it will generate the cache content before we can store it
			// we are using the hashs for the id, therefore it is required during creation
			"fingerprints", certFingerprints(cert),
			"isRevoked", isRevoked,
			"revokedAt", revokedAt,
			"isVerified", verified,
		)
		if err != nil {
			return nil, err
		}

		// store parsed object with resource
		mqlCert := raw.(Certificate)
		mqlCert.MqlResource().Cache.Store("_cert", &resources.CacheEntry{Data: cert})

		res[i] = mqlCert
	}

	return res, nil
}

func (s *mqlTls) GetParams(socket Socket, domainName string) (map[string]interface{}, error) {
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

	tester := tlsshake.New(proto, domainName, host, int(port))
	if err := tester.Test(tlsshake.DefaultScanConfig()); err != nil {
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

	// Create certificates
	res["certificates"], err = parseCertificates(s.MotorRuntime, domainName, &findings, findings.Certificates)
	if err != nil {
		return nil, err
	}

	res["non-sni-certificates"], err = parseCertificates(s.MotorRuntime, domainName, &findings, findings.NonSNIcertificates)
	if err != nil {
		return nil, err
	}

	return res, nil
}

func (s *mqlTls) GetVersions(params interface{}) ([]interface{}, error) {
	paramsM, ok := params.(map[string]interface{})
	if !ok {
		return []interface{}{}, nil
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

func (s *mqlTls) GetCiphers(params interface{}) ([]interface{}, error) {
	paramsM, ok := params.(map[string]interface{})
	if !ok {
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

func (s *mqlTls) GetExtensions(params interface{}) ([]interface{}, error) {
	paramsM, ok := params.(map[string]interface{})
	if !ok {
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

func (s *mqlTls) GetCertificates(params interface{}) ([]interface{}, error) {
	paramsM, ok := params.(map[string]interface{})
	if !ok {
		return []interface{}{}, nil
	}

	raw, ok := paramsM["certificates"]
	if !ok {
		return []interface{}{}, nil
	}

	return raw.([]interface{}), nil
}

func (s *mqlTls) GetNonSniCertificates(params interface{}) ([]interface{}, error) {
	paramsM, ok := params.(map[string]interface{})
	if !ok {
		return []interface{}{}, nil
	}

	raw, ok := paramsM["non-sni-certificates"]
	if !ok {
		return []interface{}{}, nil
	}

	return raw.([]interface{}), nil
}
