package dnsshake

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"errors"
	"io/ioutil"
	"mime/quotedprintable"
	"strings"
)

// DkimPublicKeyRepresentation represents a parsed version of public key record
// see https://datatracker.ietf.org/doc/html/rfc6376
type DkimPublicKeyRepresentation struct {
	// Version of the DKIM key record (plain-text; RECOMMENDED, default is "DKIM1")
	Version string
	// Acceptable hash algorithms (plain-text; OPTIONAL, defaults to allowing all algorithms)
	HashAlgorithms []string
	// Key type (plain-text; OPTIONAL, default is "rsa")
	KeyType string
	// Notes that might be of interest to a human (qp-section; OPTIONAL, default is empty)
	Notes string
	// Public-key data (base64; REQUIRED)
	PublicKeyData string
	// Service Type (plain-text; OPTIONAL; default is "*")
	ServiceType []string
	// Flags, represented as a colon-separated list of names (plain-text; OPTIONAL, default is no flags set)
	Flags []string
}

func (pkr *DkimPublicKeyRepresentation) Valid() (bool, []string, []string) {
	errorMsg := []string{}
	warningMsg := []string{}
	if pkr.Version != "" && pkr.Version != "DKIM1" {
		errorMsg = append(errorMsg, "If version is specified, this tag MUST be set to \"DKIM1\"")
	}

	if pkr.KeyType != "" && pkr.KeyType != "rsa" {
		// according to RFC, wrong types should be ignored but since it would not match with the public key
		// we throw an error here
		errorMsg = append(errorMsg, "Unrecognized key types")
	}

	if pkr.PublicKeyData == "" {
		// NOTE: empty value represents a revoked key, we could argue that it is a warning
		errorMsg = append(errorMsg, "public key has been revoked")
	}

	if pkr.PublicKeyData != "" {
		_, err := pkr.PublicKey()
		if err != nil {
			errorMsg = append(errorMsg, "unable to parse public key")
		}
	}

	// TODO: we may want to add warning checks for service type and flags values
	return len(errorMsg) == 0, errorMsg, warningMsg
}

func (pkr *DkimPublicKeyRepresentation) PublicKey() (*rsa.PublicKey, error) {
	if pkr.PublicKeyData == "" {
		return nil, errors.New("public key has been revoked")
	}
	pem64, err := base64.StdEncoding.DecodeString(pkr.PublicKeyData)
	if err != nil {
		return nil, errors.New("could not parse public key data")
	}

	pk, _ := x509.ParsePKIXPublicKey(pem64)
	if pk, ok := pk.(*rsa.PublicKey); ok {
		return pk, nil
	}
	return nil, errors.New("invalid rsa key")
}

// NewDkimPublicKeyRepresentation parses DNS DKIM record
// https://datatracker.ietf.org/doc/html/rfc6376#section-3.6.1
func NewDkimPublicKeyRepresentation(dkimRecord string) (*DkimPublicKeyRepresentation, error) {
	pkr := &DkimPublicKeyRepresentation{}
	p := strings.Split(dkimRecord, ";")
	for i, data := range p {
		keyVal := strings.SplitN(data, "=", 2)
		key := keyVal[0]
		val := ""
		if len(keyVal) > 1 {
			val = strings.TrimSpace(keyVal[1])
		}
		switch strings.ToLower(strings.TrimSpace(key)) {
		case "v":
			// RFC: This tag MUST be the first tag in the record.
			if i != 0 {
				return nil, errors.New("invalid DKIM record")
			}
			pkr.Version = val
		case "h":
			p := strings.Split(strings.ToLower(val), ":")
			for i := range p {
				h := strings.TrimSpace(p[i])
				pkr.HashAlgorithms = append(pkr.HashAlgorithms, h)
			}
		case "k":
			pkr.KeyType = strings.ToLower(val)
		case "n":
			pkr.Notes = val
			// parse quote printable
			qp, err := ioutil.ReadAll(quotedprintable.NewReader(strings.NewReader(val)))
			if err == nil {
				pkr.Notes = string(qp)
			}
		case "p":
			pkr.PublicKeyData = val
		case "s":
			serviceTypes := strings.Split(strings.ToLower(val), ":")
			for i := range serviceTypes {
				pkr.ServiceType = append(pkr.ServiceType, strings.TrimSpace(serviceTypes[i]))
			}
		case "t":
			flags := strings.Split(strings.ToLower(val), ":")
			for i := range flags {
				pkr.Flags = append(pkr.Flags, strings.TrimSpace(flags[i]))
			}
		}
	}

	return pkr, nil
}
