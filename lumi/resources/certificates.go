package resources

import (
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
	"go.mondoo.io/mondoo/lumi"
	"go.mondoo.io/mondoo/lumi/resources/certificates"
	"go.mondoo.io/mondoo/motor/platform"
)

func (s *lumiParseCertificates) init(args *lumi.Args) (*lumi.Args, Authorizedkeys, error) {
	// resolve path to file
	if x, ok := (*args)["path"]; ok {
		path, ok := x.(string)
		if !ok {
			return nil, nil, errors.New("Wrong type for 'path' in authorizedkeys initialization, it must be a string")
		}

		f, err := s.Runtime.CreateResource("file", "path", path)
		if err != nil {
			return nil, nil, err
		}
		(*args)["file"] = f
	}

	return args, nil, nil
}

func certificatesid(path string) string {
	return "certificates:" + path
}

func (a *lumiParseCertificates) id() (string, error) {
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

func (a *lumiParseCertificates) GetFile() (File, error) {
	path, err := a.Path()
	if err != nil {
		return nil, err
	}

	f, err := a.Runtime.CreateResource("file", "path", path)
	if err != nil {
		return nil, err
	}
	return f.(File), nil
}

func (a *lumiParseCertificates) GetContent(file File) (string, error) {
	// TODO: this can be heavily improved once we do it right, since this is constantly
	// re-registered as the file changes
	err := a.Runtime.WatchAndCompute(file, "content", a, "content")
	if err != nil {
		return "", err
	}

	return file.Content()
}

func pkixnameToLumi(runtime *lumi.Runtime, name pkix.Name, id string) (PkixName, error) {
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

	lumiPkixName, err := runtime.CreateResource("pkix.name",
		"id", id,
		"dn", name.String(),
		"serialnumber", name.SerialNumber,
		"commonname", name.CommonName,
		"country", strSliceToInterface(name.Country),
		"organization", strSliceToInterface(name.Organization),
		"organizationalunit", strSliceToInterface(name.OrganizationalUnit),
		"locality", strSliceToInterface(name.Locality),
		"province", strSliceToInterface(name.Province),
		"streetaddress", strSliceToInterface(name.StreetAddress),
		"postalcode", strSliceToInterface(name.PostalCode),
		"names", names,
		"extranames", extraNames,
	)
	if err != nil {
		return nil, err
	}
	return lumiPkixName.(PkixName), nil
}

func pkixextensionToLumi(runtime *lumi.Runtime, ext pkix.Extension, id string) (PkixExtension, error) {
	lumiPkixExt, err := runtime.CreateResource("pkix.extension",
		"identifier", id,
		"critical", ext.Critical,
		"value", string(ext.Value),
	)
	if err != nil {
		return nil, err
	}
	return lumiPkixExt.(PkixExtension), nil
}

func (p *lumiParseCertificates) GetList(content string, path string) ([]interface{}, error) {
	certs, err := certificates.ParseCertFromPEM(strings.NewReader(content))
	if err != nil {
		return nil, err
	}

	return certificatesToLumiCertificates(p.Runtime, certs)
}

func certificatesToLumiCertificates(runtime *lumi.Runtime, certs []*x509.Certificate) ([]interface{}, error) {
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

		lumiCert, err := runtime.CreateResource("certificate",
			"pem", string(certdata),
			// NOTE: if we do not set the hash here, it will generate the cache content before we can store it
			// we are using the hashs for the id, therefore it is required during creation
			"hashs", certHashs(cert),
		)
		if err != nil {
			return nil, err
		}

		c := lumiCert.(Certificate)

		// store parsed object with resource
		c.LumiResource().Cache.Store("_cert", &lumi.CacheEntry{Data: cert})
		res = append(res, c)
	}
	return res, nil
}

func (r *lumiCertificate) id() (string, error) {
	fingerprints, err := r.Hashs()
	if err != nil {
		return "", err
	}
	return "certificate:" + fingerprints["sha256"].(string), nil
}

func (s *lumiCertificate) getGoCert() *x509.Certificate {
	entry, ok := s.LumiResource().Cache.Load("_cert")
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

	s.LumiResource().Cache.Store("_cert", &lumi.CacheEntry{Data: cert[0]})
	return cert[0]
}

func certHashs(cert *x509.Certificate) map[string]interface{} {
	return map[string]interface{}{
		"sha1":   hex.EncodeToString(certificates.Sha1Hash(cert)),
		"sha256": hex.EncodeToString(certificates.Sha256Hash(cert)),
		"md5":    hex.EncodeToString(certificates.Md5Hash(cert)),
	}
}

func (s *lumiCertificate) GetHashs() (map[string]interface{}, error) {
	cert := s.getGoCert()
	return certHashs(cert), nil
}

func (s *lumiCertificate) GetSerial() (string, error) {
	cert := s.getGoCert()
	// TODO: we may want return bytes and leave the printing to runtime
	return certificates.HexEncodeToHumanString(cert.SerialNumber.Bytes()), nil
}

func (s *lumiCertificate) GetSubjectkeyid() (string, error) {
	cert := s.getGoCert()
	// TODO: we may want return bytes and leave the printing to runtime
	return certificates.HexEncodeToHumanString(cert.SubjectKeyId), nil
}

func (s *lumiCertificate) GetAuthoritykeyid() (string, error) {
	cert := s.getGoCert()
	// TODO: we may want return bytes and leave the printing to runtime
	return certificates.HexEncodeToHumanString(cert.AuthorityKeyId), nil
}

func (s *lumiCertificate) GetSubject() (interface{}, error) {
	cert := s.getGoCert()
	fingerprint := hex.EncodeToString(certificates.Sha256Hash(cert))
	lumiSubject, err := pkixnameToLumi(s.Runtime, cert.Subject, fingerprint+":subject")
	if err != nil {
		return nil, err
	}
	return lumiSubject, nil
}

func (s *lumiCertificate) GetIssuer() (interface{}, error) {
	cert := s.getGoCert()
	fingerprint := hex.EncodeToString(certificates.Sha256Hash(cert))
	lumiIssuer, err := pkixnameToLumi(s.Runtime, cert.Issuer, fingerprint+":issuer")
	if err != nil {
		return nil, err
	}
	return lumiIssuer, nil
}

func (s *lumiCertificate) GetVersion() (int64, error) {
	cert := s.getGoCert()
	return int64(cert.Version), nil
}

func (s *lumiCertificate) GetIsca() (bool, error) {
	cert := s.getGoCert()
	return cert.IsCA, nil
}

func (s *lumiCertificate) GetNotbefore() (*time.Time, error) {
	cert := s.getGoCert()
	return &cert.NotBefore, nil
}

func (s *lumiCertificate) GetNotafter() (*time.Time, error) {
	cert := s.getGoCert()
	return &cert.NotAfter, nil
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

func (s *lumiCertificate) GetKeyusage() ([]interface{}, error) {
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

func (s *lumiCertificate) GetExtendedkeyusage() ([]interface{}, error) {
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

func (s *lumiCertificate) GetExtensions() ([]interface{}, error) {
	res := []interface{}{}
	cert := s.getGoCert()
	fingerprint := hex.EncodeToString(certificates.Sha256Hash(cert))
	for i := range cert.Extensions {
		extension := cert.Extensions[i]
		ext, err := pkixextensionToLumi(s.Runtime, extension, fingerprint+":"+extension.Id.String())
		if err != nil {
			return nil, err
		}
		res = append(res, ext)
	}
	return res, nil
}

func (s *lumiCertificate) GetPolicyidentifier() ([]interface{}, error) {
	res := []interface{}{}
	cert := s.getGoCert()
	for i := range cert.PolicyIdentifiers {
		res = append(res, cert.PolicyIdentifiers[i].String())
	}
	return res, nil
}

func (s *lumiCertificate) GetSigningalgorithm() (string, error) {
	cert := s.getGoCert()
	return cert.SignatureAlgorithm.String(), nil
}

func (s *lumiCertificate) GetSignature() (string, error) {
	cert := s.getGoCert()
	// TODO: return bytes
	return hex.EncodeToString(cert.Signature), nil
}

func (s *lumiCertificate) GetCrldistributionpoints() ([]interface{}, error) {
	cert := s.getGoCert()
	return strSliceToInterface(cert.CRLDistributionPoints), nil
}

func (s *lumiCertificate) GetOcspserver() ([]interface{}, error) {
	cert := s.getGoCert()
	return strSliceToInterface(cert.OCSPServer), nil
}

func (s *lumiCertificate) GetIssuingcertificateurl() ([]interface{}, error) {
	cert := s.getGoCert()
	return strSliceToInterface(cert.IssuingCertificateURL), nil
}

func (r *lumiPkixName) id() (string, error) {
	return r.Id()
}

func (r *lumiPkixExtension) id() (string, error) {
	return r.Identifier()
}

func (s *lumiOsRootcertificates) id() (string, error) {
	return "osrootcertificates", nil
}

func (s *lumiOsRootcertificates) init(args *lumi.Args) (*lumi.Args, OsRootcertificates, error) {
	pi, err := s.Runtime.Motor.Platform()
	if err != nil {
		return nil, nil, err
	}

	var files []string
	if pi.IsFamily(platform.FAMILY_LINUX) {
		files = certificates.LinuxCertFiles
	} else if pi.IsFamily(platform.FAMILY_BSD) {
		files = certificates.BsdCertFiles
	} else {
		return nil, nil, errors.New("root certificates are not unsupported on this platform: " + pi.Name + " " + pi.Release)
	}

	// search the first file that exists, it mimics the behavior go is doing
	lumiFiles := []interface{}{}
	for i := range files {
		log.Trace().Str("path", files[i]).Msg("os.rootcertificates> check root certificate path")
		fileInfo, err := s.Runtime.Motor.Transport.FS().Stat(files[i])
		if err != nil {
			log.Trace().Err(err).Str("path", files[i]).Msg("os.rootcertificates> file does not exist")
			continue
		}
		log.Debug().Str("path", files[i]).Msg("os.rootcertificates> found root certificate bundle path")
		if !fileInfo.IsDir() {
			f, err := s.Runtime.CreateResource("file", "path", files[i])
			if err != nil {
				return nil, nil, err
			}
			lumiFiles = append(lumiFiles, f.(File))
			break
		}
	}

	(*args)["files"] = lumiFiles
	return args, nil, nil
}

func (s *lumiOsRootcertificates) GetFiles() ([]interface{}, error) {
	return nil, errors.New("not implemented")
}

func (s *lumiOsRootcertificates) GetContent(files []interface{}) ([]interface{}, error) {
	contents := []interface{}{}

	for i := range files {
		file := files[i].(File)

		// TODO: this can be heavily improved once we do it right, since this is constantly
		// re-registered as the file changes
		err := s.Runtime.WatchAndCompute(file, "content", s, "content")
		if err != nil {
			return nil, err
		}

		content, err := file.Content()
		if err != nil {
			return nil, err
		}
		contents = append(contents, content)
	}

	return contents, nil
}

func (s *lumiOsRootcertificates) GetList(content []interface{}) ([]interface{}, error) {
	certificateList := []*x509.Certificate{}
	for i := range content {
		certs, err := certificates.ParseCertFromPEM(strings.NewReader(content[i].(string)))
		if err != nil {
			return nil, err
		}
		certificateList = append(certificateList, certs...)
	}
	return certificatesToLumiCertificates(s.Runtime, certificateList)
}
