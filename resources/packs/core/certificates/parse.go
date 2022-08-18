package certificates

import (
	"bytes"
	"crypto/x509"
	"encoding/pem"
	"io"
	"io/ioutil"

	"k8s.io/client-go/util/cert"
)

func ParseCertFromPEM(r io.Reader) ([]*x509.Certificate, error) {
	data, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}

	certs, err := cert.ParseCertsPEM(data)
	if err != nil {
		return nil, err
	}

	return certs, nil
}

func EncodeCertAsPEM(cert *x509.Certificate) ([]byte, error) {
	certBuffer := bytes.Buffer{}
	if err := pem.Encode(&certBuffer, &pem.Block{Type: CertificateBlockType, Bytes: cert.Raw}); err != nil {
		return nil, err
	}
	return certBuffer.Bytes(), nil
}

const (
	// CertificateBlockType is a possible value for pem.Block.Type.
	CertificateBlockType = "CERTIFICATE"
)
