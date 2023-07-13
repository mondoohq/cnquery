package core

import (
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"github.com/ProtonMail/go-crypto/openpgp"
	"github.com/ProtonMail/go-crypto/openpgp/packet"
	"go.mondoo.com/cnquery/checksums"
	"go.mondoo.com/cnquery/llx"
	"go.mondoo.com/cnquery/resources"
)

func (s *mqlParseOpenpgp) init(args *resources.Args) (*resources.Args, ParseCertificates, error) {
	// resolve path to file
	if x, ok := (*args)["path"]; ok {
		path, ok := x.(string)
		if !ok {
			return nil, nil, errors.New("Wrong type for 'path' in openpgp initialization, it must be a string")
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

func openpgpid(path string) string {
	return "certificates:" + path
}

func (a *mqlParseOpenpgp) id() (string, error) {
	r, err := a.File()
	if err != nil {
		return "", err
	}
	path, err := r.Path()
	if err != nil {
		return "", err
	}

	return openpgpid(path), nil
}

func (a *mqlParseOpenpgp) GetFile() (File, error) {
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

func (a *mqlParseOpenpgp) GetContent(file File) (string, error) {
	// TODO: this can be heavily improved once we do it right, since this is constantly
	// re-registered as the file changes
	err := a.MotorRuntime.WatchAndCompute(file, "content", a, "content")
	if err != nil {
		return "", err
	}

	return file.Content()
}

func (p *mqlParseOpenpgp) GetList(content string, path string) ([]interface{}, error) {
	entries, err := openpgp.ReadArmoredKeyRing(strings.NewReader(content))
	if err != nil {
		return nil, err
	}

	return newMqlOpenpgpEntries(p.MotorRuntime, entries)
}

// mqlOpenPgpEntries takes a collection of open pgp entries
// and converts it into MQL open pgp objects
func newMqlOpenpgpEntries(runtime *resources.Runtime, entities []*openpgp.Entity) ([]interface{}, error) {
	res := []interface{}{}
	// to create certificate resources
	for i := range entities {
		entity := entities[i]

		if entity == nil {
			continue
		}

		pubKey, err := newMqlOpenpgpPublicKey(runtime, entity.PrimaryKey)
		if err != nil {
			return nil, err
		}

		mqlCert, err := runtime.CreateResource("openpgp.entity",
			"primaryPublicKey", pubKey,
		)
		if err != nil {
			return nil, err
		}

		c := mqlCert.(OpenpgpEntity)
		c.MqlResource().Cache.Store("_identities", &resources.CacheEntry{Data: entity.Identities})
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

func newMqlOpenpgpPublicKey(runtime *resources.Runtime, publicKey *packet.PublicKey) (OpenpgpPublicKey, error) {
	pubKeyAlgo := pgpAlgoString(publicKey.PubKeyAlgo)
	// we ignore the error here since it happens only when no algorithm is found
	bitlength, _ := publicKey.BitLength()

	entry, err := runtime.CreateResource("openpgp.publicKey",
		"id", publicKey.KeyIdString(),
		"version", int64(publicKey.Version),
		"fingerprint", hex.EncodeToString(publicKey.Fingerprint),
		"keyAlgorithm", pubKeyAlgo,
		"bitLength", int64(bitlength),
		"creationTime", &publicKey.CreationTime,
	)
	if err != nil {
		return nil, err
	}

	return entry.(OpenpgpPublicKey), nil
}

func (r *mqlOpenpgpEntity) id() (string, error) {
	pubKey, err := r.PrimaryPublicKey()
	if err != nil {
		return "", err
	}
	fingerprint, err := pubKey.Fingerprint()
	if err != nil {
		return "", err
	}
	return "openpgp.entity/" + fingerprint, nil
}

func (r *mqlOpenpgpEntity) GetIdentities() ([]interface{}, error) {
	entry, ok := r.MqlResource().Cache.Load("_identities")
	if !ok {
		return nil, errors.New("identities not found in cache")
	}

	pubKey, err := r.PrimaryPublicKey()
	if err != nil {
		return nil, err
	}

	fingerprint, err := pubKey.Fingerprint()
	if err != nil {
		return nil, err
	}

	res := []interface{}{}
	identities := entry.Data.(map[string]*openpgp.Identity)
	for k := range identities {
		identity := identities[k]
		mqlOpenpgpIdentity, err := r.MotorRuntime.CreateResource("openpgp.identity",
			"fingerprint", fingerprint,
			"id", identity.UserId.Id,
			"name", identity.UserId.Name,
			"email", identity.UserId.Email,
			"comment", identity.UserId.Comment,
		)
		if err != nil {
			return nil, err
		}
		mqlOpenpgpIdentityResource := mqlOpenpgpIdentity.(OpenpgpIdentity)
		mqlOpenpgpIdentityResource.MqlResource().Cache.Store("_signatures", &resources.CacheEntry{Data: identity.Signatures})

		res = append(res, mqlOpenpgpIdentityResource)
	}

	return res, nil
}

func (r *mqlOpenpgpPublicKey) id() (string, error) {
	fingerprint, err := r.Fingerprint()
	if err != nil {
		return "", err
	}
	return "openpgp.publickey/" + fingerprint, nil
}

func (r *mqlOpenpgpIdentity) id() (string, error) {
	fingerprint, err := r.Fingerprint()
	if err != nil {
		return "", err
	}

	name, err := r.Name()
	if err != nil {
		return "", err
	}

	return "openpgp.identity/" + fingerprint + "/" + name, nil
}

func (r *mqlOpenpgpIdentity) GetSignatures() ([]interface{}, error) {
	entry, ok := r.MqlResource().Cache.Load("_signatures")
	if !ok {
		return nil, errors.New("signatures not found in cache")
	}

	fingerprint, err := r.Fingerprint()
	if err != nil {
		return nil, err
	}

	id, err := r.Id()
	if err != nil {
		return nil, err
	}

	res := []interface{}{}
	signatures := entry.Data.([]*packet.Signature)
	for k := range signatures {
		signature := signatures[k]

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
		var expirationTime *time.Time
		if signature.SigLifetimeSecs != nil {
			// NOTE: this can potentially overflow
			lifetime = int64(*signature.SigLifetimeSecs)

			expiry := signature.CreationTime.Add(time.Duration(*signature.SigLifetimeSecs) * time.Second)
			diff := expiry.Unix() - time.Now().Unix()
			ts := llx.DurationToTime(diff)
			expirationTime = &ts
		}

		keyLifetime := int64(-1)
		var keyExpirationTime *time.Time
		if signature.KeyLifetimeSecs != nil {
			// NOTE: this can potentially overflow
			keyLifetime = int64(*signature.KeyLifetimeSecs)

			expiry := signature.CreationTime.Add(time.Duration(*signature.KeyLifetimeSecs) * time.Second)
			diff := expiry.Unix() - time.Now().Unix()
			ts := llx.DurationToTime(diff)
			keyExpirationTime = &ts
		}

		mqlOpenpgpSignature, err := r.MotorRuntime.CreateResource("openpgp.signature",
			"fingerprint", fingerprint,
			"identityName", id,
			"hash", signature.Hash.String(),
			"version", int64(signature.Version),
			"signatureType", signatureType,
			"keyAlgorithm", pgpAlgoString(signature.PubKeyAlgo),
			"creationTime", &signature.CreationTime,
			"lifetimeSecs", lifetime,
			"expiresIn", expirationTime,
			"keyLifetimeSecs", keyLifetime,
			"keyExpiresIn", keyExpirationTime,
		)
		if err != nil {
			return nil, err
		}
		mqlOpenpgpSignatureResource := mqlOpenpgpSignature.(OpenpgpSignature)
		res = append(res, mqlOpenpgpSignatureResource)
	}

	return res, nil
}

func (r *mqlOpenpgpSignature) id() (string, error) {
	fingerprint, err := r.Fingerprint()
	if err != nil {
		return "", err
	}

	identityName, err := r.IdentityName()
	if err != nil {
		return "", err
	}

	hash, err := r.Hash()
	if err != nil {
		return "", err
	}

	return "openpgp.identity/" + fingerprint + "/" + identityName + "/" + hash, nil
}
