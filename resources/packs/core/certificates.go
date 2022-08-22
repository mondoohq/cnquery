package core

import (
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
	"go.mondoo.io/mondoo/checksums"
	"go.mondoo.io/mondoo/llx"
	"go.mondoo.io/mondoo/resources"
	"go.mondoo.io/mondoo/resources/packs/core/certificates"
)

func (s *mqlParseCertificates) init(args *resources.Args) (*resources.Args, ParseCertificates, error) {
	// resolve path to file
	if x, ok := (*args)["path"]; ok {
		path, ok := x.(string)
		if !ok {
			return nil, nil, errors.New("Wrong type for 'path' in authorizedkeys initialization, it must be a string")
		}

		f, err := s.MotorRuntime.CreateResource("file", "path", path)
		if err != nil {
			return nil, nil, err
		}
		(*args)["file"] = f
	} else if x, ok := (*args)["content"]; ok {
		content := x.(string)
		virtualPath := "in-memory://" + checksums.New.Add(content).String()
		f, err := s.MotorRuntime.CreateResource("file", "path", virtualPath, "content", content, "exists", true)
		if err != nil {
			return nil, nil, err
		}
		(*args)["file"] = f
		(*args)["path"] = virtualPath
	} else {
		return nil, nil, errors.New("missing 'path' or 'content' for parse.json initialization")
	}

	return args, nil, nil
}

func certificatesid(path string) string {
	return "certificates:" + path
}

func (a *mqlParseCertificates) id() (string, error) {
	r, err := a.File()
	if err != nil {
		return "", err
	}
	path, err := r.Path()
	if err != nil {
		return "", err
	}

	return certificatesid(path), nil
}

func (a *mqlParseCertificates) GetFile() (File, error) {
	path, err := a.Path()
	if err != nil {
		return nil, err
	}

	f, err := a.MotorRuntime.CreateResource("file", "path", path)
	if err != nil {
		return nil, err
	}
	return f.(File), nil
}

func (a *mqlParseCertificates) GetContent(file File) (string, error) {
	// TODO: this can be heavily improved once we do it right, since this is constantly
	// re-registered as the file changes
	err := a.MotorRuntime.WatchAndCompute(file, "content", a, "content")
	if err != nil {
		return "", err
	}

	return file.Content()
}

func pkixnameToMql(runtime *resources.Runtime, name pkix.Name, id string) (PkixName, error) {
	names := map[string]interface{}{}
	for i := range name.Names {
		key := name.Names[i].Type.String()
		names[key] = fmt.Sprintf("%v", name.Names[i].Value)
	}

	extraNames := map[string]interface{}{}
	for i := range name.ExtraNames {
		key := name.ExtraNames[i].Type.String()
		extraNames[key] = fmt.Sprintf("%v", name.ExtraNames[i].Value)
	}

	mqlPkixName, err := runtime.CreateResource("pkix.name",
		"id", id,
		"dn", name.String(),
		"serialNumber", name.SerialNumber,
		"commonName", name.CommonName,
		"country", StrSliceToInterface(name.Country),
		"organization", StrSliceToInterface(name.Organization),
		"organizationalUnit", StrSliceToInterface(name.OrganizationalUnit),
		"locality", StrSliceToInterface(name.Locality),
		"province", StrSliceToInterface(name.Province),
		"streetAddress", StrSliceToInterface(name.StreetAddress),
		"postalCode", StrSliceToInterface(name.PostalCode),
		"names", names,
		"extraNames", extraNames,
	)
	if err != nil {
		return nil, err
	}
	return mqlPkixName.(PkixName), nil
}

func pkixextensionToMql(runtime *resources.Runtime, ext pkix.Extension, id string) (PkixExtension, error) {
	mqlPkixExt, err := runtime.CreateResource("pkix.extension",
		"identifier", id,
		"critical", ext.Critical,
		"value", string(ext.Value),
	)
	if err != nil {
		return nil, err
	}
	return mqlPkixExt.(PkixExtension), nil
}

func (p *mqlParseCertificates) GetList(content string, path string) ([]interface{}, error) {
	certs, err := certificates.ParseCertFromPEM(strings.NewReader(content))
	if err != nil {
		return nil, err
	}

	return CertificatesToMqlCertificates(p.MotorRuntime, certs)
}

// CertificatesToMqlCertificates takes a collection of x509 certs
// and converts it into MQL certificate objects
func CertificatesToMqlCertificates(runtime *resources.Runtime, certs []*x509.Certificate) ([]interface{}, error) {
	res := []interface{}{}
	// to create certificate resources
	for i := range certs {
		cert := certs[i]

		if cert == nil {
			continue
		}

		certdata, err := certificates.EncodeCertAsPEM(cert)
		if err != nil {
			return nil, err
		}

		mqlCert, err := runtime.CreateResource("certificate",
			"pem", string(certdata),
			// NOTE: if we do not set the hash here, it will generate the cache content before we can store it
			// we are using the hashs for the id, therefore it is required during creation
			"fingerprints", certFingerprints(cert),
		)
		if err != nil {
			return nil, err
		}

		c := mqlCert.(Certificate)

		// store parsed object with resource
		c.MqlResource().Cache.Store("_cert", &resources.CacheEntry{Data: cert})
		res = append(res, c)
	}
	return res, nil
}

func (r *mqlCertificate) id() (string, error) {
	fingerprints, err := r.Fingerprints()
	if err != nil {
		return "", err
	}
	return "certificate:" + fingerprints["sha256"].(string), nil
}

func (s *mqlCertificate) getGoCert() *x509.Certificate {
	entry, ok := s.MqlResource().Cache.Load("_cert")
	if ok {
		return entry.Data.(*x509.Certificate)
	}

	log.Warn().Msg("restore cache object")
	data, err := s.Pem()
	if err != nil {
		log.Error().Err(err).Msg("certificate is missing pem data")
		return nil
	}

	cert, err := certificates.ParseCertFromPEM(strings.NewReader(data))
	if err != nil {
		log.Error().Err(err).Msg("certificate is has invalid pem data")
		return nil
	}

	if len(cert) > 1 {
		log.Error().Msg("pem for cert contains more than one certificate, ignore additional certificates")
	}

	s.MqlResource().Cache.Store("_cert", &resources.CacheEntry{Data: cert[0]})
	return cert[0]
}

func certFingerprints(cert *x509.Certificate) map[string]interface{} {
	return map[string]interface{}{
		"sha1":   hex.EncodeToString(certificates.Sha1Hash(cert)),
		"sha256": hex.EncodeToString(certificates.Sha256Hash(cert)),
		"md5":    hex.EncodeToString(certificates.Md5Hash(cert)),
	}
}

func (s *mqlCertificate) GetFingerprints() (map[string]interface{}, error) {
	cert := s.getGoCert()
	return certFingerprints(cert), nil
}

func (s *mqlCertificate) GetSerial() (string, error) {
	cert := s.getGoCert()
	// TODO: we may want return bytes and leave the printing to runtime
	return certificates.HexEncodeToHumanString(cert.SerialNumber.Bytes()), nil
}

func (s *mqlCertificate) GetSubjectKeyID() (string, error) {
	cert := s.getGoCert()
	// TODO: we may want return bytes and leave the printing to runtime
	return certificates.HexEncodeToHumanString(cert.SubjectKeyId), nil
}

func (s *mqlCertificate) GetAuthorityKeyID() (string, error) {
	cert := s.getGoCert()
	// TODO: we may want return bytes and leave the printing to runtime
	return certificates.HexEncodeToHumanString(cert.AuthorityKeyId), nil
}

func (s *mqlCertificate) GetSubject() (interface{}, error) {
	cert := s.getGoCert()
	fingerprint := hex.EncodeToString(certificates.Sha256Hash(cert))
	mqlSubject, err := pkixnameToMql(s.MotorRuntime, cert.Subject, fingerprint+":subject")
	if err != nil {
		return nil, err
	}
	return mqlSubject, nil
}

func (s *mqlCertificate) GetIssuer() (interface{}, error) {
	cert := s.getGoCert()
	fingerprint := hex.EncodeToString(certificates.Sha256Hash(cert))
	mqlIssuer, err := pkixnameToMql(s.MotorRuntime, cert.Issuer, fingerprint+":issuer")
	if err != nil {
		return nil, err
	}
	return mqlIssuer, nil
}

func (s *mqlCertificate) GetVersion() (int64, error) {
	cert := s.getGoCert()
	return int64(cert.Version), nil
}

func (s *mqlCertificate) GetIsCA() (bool, error) {
	cert := s.getGoCert()
	return cert.IsCA, nil
}

func (s *mqlCertificate) GetNotBefore() (*time.Time, error) {
	cert := s.getGoCert()
	return &cert.NotBefore, nil
}

func (s *mqlCertificate) GetNotAfter() (*time.Time, error) {
	cert := s.getGoCert()
	return &cert.NotAfter, nil
}

func (s *mqlCertificate) GetExpiresIn() (*time.Time, error) {
	cert := s.getGoCert()
	diff := cert.NotAfter.Unix() - time.Now().Unix()
	ts := llx.DurationToTime(diff)
	return &ts, nil
}

var keyusageNames = map[x509.KeyUsage]string{
	x509.KeyUsageDigitalSignature:  "DigitalSignature",
	x509.KeyUsageContentCommitment: "ContentCommitment",
	x509.KeyUsageKeyEncipherment:   "KeyEncipherment",
	x509.KeyUsageDataEncipherment:  "DataEncipherment",
	x509.KeyUsageKeyAgreement:      "KeyAgreement",
	x509.KeyUsageCertSign:          "CertificateSign",
	x509.KeyUsageCRLSign:           "CRLSign",
	x509.KeyUsageEncipherOnly:      "EncipherOnly",
	x509.KeyUsageDecipherOnly:      "DecipherOnly",
}

func (s *mqlCertificate) GetKeyUsage() ([]interface{}, error) {
	res := []interface{}{}
	cert := s.getGoCert()

	for k := range keyusageNames {
		if cert.KeyUsage&k != 0 {
			res = append(res, keyusageNames[k])
		}
	}

	return res, nil
}

var extendendkeyusageNames = map[x509.ExtKeyUsage]string{
	x509.ExtKeyUsageAny:                            "Any",
	x509.ExtKeyUsageServerAuth:                     "ServerAuth",
	x509.ExtKeyUsageClientAuth:                     "ClientAuth",
	x509.ExtKeyUsageCodeSigning:                    "CodeSigning",
	x509.ExtKeyUsageEmailProtection:                "EmailProtection",
	x509.ExtKeyUsageIPSECEndSystem:                 "IPSECEndSystem",
	x509.ExtKeyUsageIPSECTunnel:                    "IPSECTunnel",
	x509.ExtKeyUsageIPSECUser:                      "IPSECUser",
	x509.ExtKeyUsageTimeStamping:                   "TimeStamping",
	x509.ExtKeyUsageOCSPSigning:                    "OCSPSigning",
	x509.ExtKeyUsageMicrosoftServerGatedCrypto:     "MicrosoftServerGatedCrypto",
	x509.ExtKeyUsageNetscapeServerGatedCrypto:      "NetscapeServerGatedCrypto",
	x509.ExtKeyUsageMicrosoftCommercialCodeSigning: "MicrosoftCommercialCodeSigning",
	x509.ExtKeyUsageMicrosoftKernelCodeSigning:     "MicrosoftKernelCodeSigning",
}

func (s *mqlCertificate) GetExtendedKeyUsage() ([]interface{}, error) {
	res := []interface{}{}
	cert := s.getGoCert()
	for i := range cert.ExtKeyUsage {
		entry := cert.ExtKeyUsage[i]
		val, ok := extendendkeyusageNames[entry]
		if !ok {
			return nil, fmt.Errorf("unknown extended key usage %d", cert.KeyUsage)
		}
		res = append(res, val)
	}
	return res, nil
}

func (s *mqlCertificate) GetExtensions() ([]interface{}, error) {
	res := []interface{}{}
	cert := s.getGoCert()
	fingerprint := hex.EncodeToString(certificates.Sha256Hash(cert))
	for i := range cert.Extensions {
		extension := cert.Extensions[i]
		ext, err := pkixextensionToMql(s.MotorRuntime, extension, fingerprint+":"+extension.Id.String())
		if err != nil {
			return nil, err
		}
		res = append(res, ext)
	}
	return res, nil
}

func (s *mqlCertificate) GetPolicyIdentifier() ([]interface{}, error) {
	res := []interface{}{}
	cert := s.getGoCert()
	for i := range cert.PolicyIdentifiers {
		res = append(res, cert.PolicyIdentifiers[i].String())
	}
	return res, nil
}

func (s *mqlCertificate) GetSigningAlgorithm() (string, error) {
	cert := s.getGoCert()
	return cert.SignatureAlgorithm.String(), nil
}

func (s *mqlCertificate) GetSignature() (string, error) {
	cert := s.getGoCert()
	// TODO: return bytes
	return hex.EncodeToString(cert.Signature), nil
}

func (s *mqlCertificate) GetCrlDistributionPoints() ([]interface{}, error) {
	cert := s.getGoCert()
	return StrSliceToInterface(cert.CRLDistributionPoints), nil
}

func (s *mqlCertificate) GetOcspServer() ([]interface{}, error) {
	cert := s.getGoCert()
	return StrSliceToInterface(cert.OCSPServer), nil
}

func (s *mqlCertificate) GetIssuingCertificateUrl() ([]interface{}, error) {
	cert := s.getGoCert()
	return StrSliceToInterface(cert.IssuingCertificateURL), nil
}

func (s *mqlCertificate) GetIsRevoked() (bool, error) {
	return false, errors.New("unknown revocation status")
}

func (s *mqlCertificate) GetRevokedAt() (*time.Time, error) {
	return nil, nil
}

func (s *mqlCertificate) GetIsVerified() (bool, error) {
	return false, nil
}

func (r *mqlPkixName) id() (string, error) {
	return r.Id()
}

func (r *mqlPkixExtension) id() (string, error) {
	return r.Identifier()
}
