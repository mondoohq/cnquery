// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"encoding/hex"
	"strings"
	"time"

	"github.com/ProtonMail/go-crypto/openpgp"
	"github.com/ProtonMail/go-crypto/openpgp/packet"
	"go.mondoo.com/cnquery/v11/llx"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/plugin"
)

func (p *mqlOpenpgpEntities) list(content string) ([]interface{}, error) {
	entries, err := openpgp.ReadArmoredKeyRing(strings.NewReader(content))
	if err != nil {
		return nil, err
	}

	res := []interface{}{}
	// to create certificate resources
	for i := range entries {
		entity := entries[i]

		if entity == nil {
			continue
		}

		pubKey, err := newMqlOpenpgpPublicKey(p.MqlRuntime, entity.PrimaryKey)
		if err != nil {
			return nil, err
		}

		mqlCert, err := CreateResource(p.MqlRuntime, "openpgp.entity", map[string]*llx.RawData{
			"primaryPublicKey": llx.ResourceData(pubKey, "openpgp.publicKey"),
		})
		if err != nil {
			return nil, err
		}

		c := mqlCert.(*mqlOpenpgpEntity)
		c._identities = entity.Identities
		res = append(res, c)
	}
	return res, nil
}

func pgpAlgoString(algorithm packet.PublicKeyAlgorithm) string {
	var pubKeyAlgo string

	switch algorithm {
	case packet.PubKeyAlgoRSA:
		pubKeyAlgo = "rsa"
	case packet.PubKeyAlgoRSAEncryptOnly:
		pubKeyAlgo = "rsa_encrypt_only"
	case packet.PubKeyAlgoRSASignOnly:
		pubKeyAlgo = "rsa_sign_only"
	case packet.PubKeyAlgoElGamal:
		pubKeyAlgo = "elgamal"
	case packet.PubKeyAlgoDSA:
		pubKeyAlgo = "dsa"
	case packet.PubKeyAlgoECDH:
		pubKeyAlgo = "ecdh"
	case packet.PubKeyAlgoECDSA:
		pubKeyAlgo = "ecdsa"
	case packet.PubKeyAlgoEdDSA:
		pubKeyAlgo = "eddsa"
	}

	return pubKeyAlgo
}

func newMqlOpenpgpPublicKey(runtime *plugin.Runtime, publicKey *packet.PublicKey) (*mqlOpenpgpPublicKey, error) {
	pubKeyAlgo := pgpAlgoString(publicKey.PubKeyAlgo)
	// we ignore the error here since it happens only when no algorithm is found
	bitlength, _ := publicKey.BitLength()

	o, err := CreateResource(runtime, "openpgp.publicKey", map[string]*llx.RawData{
		"id":           llx.StringData(publicKey.KeyIdString()),
		"version":      llx.IntData(int64(publicKey.Version)),
		"fingerprint":  llx.StringData(hex.EncodeToString(publicKey.Fingerprint)),
		"keyAlgorithm": llx.StringData(pubKeyAlgo),
		"bitLength":    llx.IntData(int64(bitlength)),
		"creationTime": llx.TimeData(publicKey.CreationTime),
	})
	if err != nil {
		return nil, err
	}

	return o.(*mqlOpenpgpPublicKey), nil
}

type mqlOpenpgpEntityInternal struct {
	_identities map[string]*openpgp.Identity
}

func (r *mqlOpenpgpEntity) id() (string, error) {
	fp := r.PrimaryPublicKey.Data.GetFingerprint()
	if fp.Error != nil {
		return "", fp.Error
	}
	return "openpgp.entity/" + fp.Data, nil
}

func (r *mqlOpenpgpEntity) identities() ([]interface{}, error) {
	fp := r.PrimaryPublicKey.Data.GetFingerprint()
	if fp.Error != nil {
		return nil, fp.Error
	}

	res := []interface{}{}
	for k := range r._identities {
		identity := r._identities[k]
		o, err := CreateResource(r.MqlRuntime, "openpgp.identity", map[string]*llx.RawData{
			"fingerprint": llx.StringData(fp.Data),
			"id":          llx.StringData(identity.UserId.Id),
			"name":        llx.StringData(identity.UserId.Name),
			"email":       llx.StringData(identity.UserId.Email),
			"comment":     llx.StringData(identity.UserId.Comment),
		})
		if err != nil {
			return nil, err
		}
		cur := o.(*mqlOpenpgpIdentity)
		cur._signatures = identity.Signatures

		res = append(res, cur)
	}

	return res, nil
}

func (r *mqlOpenpgpPublicKey) id() (string, error) {
	return "openpgp.publickey/" + r.Fingerprint.Data, nil
}

type mqlOpenpgpIdentityInternal struct {
	_signatures []*packet.Signature
}

func (r *mqlOpenpgpIdentity) id() (string, error) {
	return "openpgp.identity/" + r.Fingerprint.Data + "/" + r.Name.Data, nil
}

func (r *mqlOpenpgpIdentity) signatures() ([]interface{}, error) {
	res := []interface{}{}
	for k := range r._signatures {
		signature := r._signatures[k]

		var signatureType string
		switch signature.SigType {
		case packet.SigTypeBinary:
			signatureType = "binary"
		case packet.SigTypeText:
			signatureType = "text"
		case packet.SigTypeGenericCert:
			signatureType = "generic_cert"
		case packet.SigTypePersonaCert:
			signatureType = "persona_cert"
		case packet.SigTypeCasualCert:
			signatureType = "casual_cert"
		case packet.SigTypePositiveCert:
			signatureType = "positive_cert"
		case packet.SigTypeSubkeyBinding:
			signatureType = "subkey_binding"
		case packet.SigTypePrimaryKeyBinding:
			signatureType = "primary_key_binding"
		case packet.SigTypeDirectSignature:
			signatureType = "direct_signature"
		case packet.SigTypeKeyRevocation:
			signatureType = "key_revocation"
		case packet.SigTypeSubkeyRevocation:
			signatureType = "subkey_revocation"
		case packet.SigTypeCertificationRevocation:
			signatureType = "cert_revocation"
		}

		lifetime := int64(-1)
		var expirationTime *llx.RawData
		if signature.SigLifetimeSecs != nil {
			// NOTE: this can potentially overflow
			lifetime = int64(*signature.SigLifetimeSecs)

			expiry := signature.CreationTime.Add(time.Duration(*signature.SigLifetimeSecs) * time.Second)
			diff := expiry.Unix() - time.Now().Unix()
			ts := llx.DurationToTime(diff)
			expirationTime = llx.TimeData(ts)
		} else {
			expirationTime = llx.NilData
		}

		keyLifetime := int64(-1)
		var keyExpirationTime *llx.RawData
		if signature.KeyLifetimeSecs != nil {
			// NOTE: this can potentially overflow
			keyLifetime = int64(*signature.KeyLifetimeSecs)

			expiry := signature.CreationTime.Add(time.Duration(*signature.KeyLifetimeSecs) * time.Second)
			diff := expiry.Unix() - time.Now().Unix()
			ts := llx.DurationToTime(diff)
			keyExpirationTime = llx.TimeData(ts)
		} else {
			expirationTime = llx.NilData
		}

		o, err := CreateResource(r.MqlRuntime, "openpgp.signature", map[string]*llx.RawData{
			"fingerprint":     llx.StringData(r.Fingerprint.Data),
			"identityName":    llx.StringData(r.Id.Data),
			"hash":            llx.StringData(signature.Hash.String()),
			"version":         llx.IntData(int64(signature.Version)),
			"signatureType":   llx.StringData(signatureType),
			"keyAlgorithm":    llx.StringData(pgpAlgoString(signature.PubKeyAlgo)),
			"creationTime":    llx.TimeData(signature.CreationTime),
			"lifetimeSecs":    llx.IntData(lifetime),
			"expiresIn":       expirationTime,
			"keyLifetimeSecs": llx.IntData(keyLifetime),
			"keyExpiresIn":    keyExpirationTime,
		})
		if err != nil {
			return nil, err
		}
		sig := o.(*mqlOpenpgpSignature)
		res = append(res, sig)
	}

	return res, nil
}

func (r *mqlOpenpgpSignature) id() (string, error) {
	return "openpgp.identity/" + r.Fingerprint.Data + "/" + r.IdentityName.Data + "/" + r.Hash.Data, nil
}
