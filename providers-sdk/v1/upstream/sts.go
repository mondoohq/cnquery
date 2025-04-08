// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package upstream

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"go.mondoo.com/cnquery/v11/providers/os/connection/ssh/signers"
	"go.mondoo.com/ranger-rpc"
	"golang.org/x/crypto/ssh"
)

func ExchangeSSHKey(apiEndpoint string, identityMrn string, resourceMrn string) (*ServiceAccountCredentials, error) {
	stsClient, err := NewSecureTokenServiceClient(apiEndpoint, ranger.DefaultHttpClient())
	if err != nil {
		return nil, err
	}

	claims := &Claims{
		Subject:  identityMrn,
		Resource: resourceMrn,
		Exp:      time.Now().Add(5 * time.Minute).Format(time.RFC3339),
		Iat:      time.Now().Format(time.RFC3339),
	}

	// fetch all signers from ssh
	sshSigners := signers.GetSignersFromSSHAgent()

	signatures, err := signClaims(claims, sshSigners...)
	if err != nil {
		return nil, err
	}

	resp, err := stsClient.ExchangeSSH(context.Background(), &ExchangeSSHKeyRequest{
		Claims:     claims,
		Signatures: signatures,
	})
	if err != nil {
		return nil, err
	}
	return &ServiceAccountCredentials{
		Mrn:         resp.Mrn,
		ParentMrn:   resp.ParentMrn,
		PrivateKey:  resp.PrivateKey,
		Certificate: resp.Certificate,
		ApiEndpoint: resp.ApiEndpoint,
	}, nil
}

func ExchangeExternalToken(apiEndpoint string, audience string, issuerURI string) (*ServiceAccountCredentials, error) {
	// TODO: This is just a testing function to fetch a GCP identity token
	// it should change to be a generic function.
	jsonToken, err := fetchGCPIdentityToken(audience)
	if err != nil {
		return nil, err
	}

	stsClient, err := NewSecureTokenServiceClient(apiEndpoint, ranger.DefaultHttpClient())
	if err != nil {
		return nil, err
	}

	resp, err := stsClient.ExchangeExternalToken(context.Background(), &ExchangeExternalTokenRequest{
		Audience:  audience,
		IssuerUri: issuerURI,
		JwtToken:  jsonToken,
	})
	if err != nil {
		return nil, err
	}

	// Decode the base64 credential string
	credBytes, err := base64.StdEncoding.DecodeString(resp.Base64Credential)
	if err != nil {
		return nil, err
	}

	// Unmarshal the JSON into ServiceAccountCredentials
	var creds ServiceAccountCredentials
	if err := json.Unmarshal(credBytes, &creds); err != nil {
		return nil, err
	}

	return &creds, nil
}

// TODO: This is just a testing function to fetch a GCP identity token
// it should change to be a generic function that checks the provider and fetches the token accordingly.
func fetchGCPIdentityToken(audience string) (string, error) {
	url := fmt.Sprintf("http://metadata.google.internal/computeMetadata/v1/instance/service-accounts/default/identity?audience=%s", audience)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Add("Metadata-Flavor", "Google")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	tokenBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(tokenBytes), nil
}

// signClaims implements claims signing with ssh.Signer
//
// To generate a new SSH key use:
// ssh-keygen -t ed25519 -C "your_email@example.com"
func signClaims(claims *Claims, signer ...ssh.Signer) ([]*SshSignature, error) {
	data, err := HashClaimsSha256(claims)
	if err != nil {
		return nil, err
	}

	signatures := make([]*SshSignature, 0, len(signer))
	for i := range signer {
		sig := signer[i]

		// sign content
		sshSign, err := sig.Sign(rand.Reader, data)
		if err != nil {
			return nil, err
		}

		signatures = append(signatures, &SshSignature{
			Alg: "x5t#S256",
			Kid: ssh.FingerprintSHA256(sig.PublicKey()),
			Sig: hex.EncodeToString(ssh.Marshal(sshSign)),
		})
	}
	return signatures, nil
}

// sha256hash returns a hash of the claims data
func sha256hash(data []byte) []byte {
	hash := sha256.New()
	hash.Write(data)
	return hash.Sum(nil)
}

// builds a canonical string from the claims to ensure that the hash is always the same and keys cannot be swapped
func buildCanonicalString(claims *Claims) string {
	params := url.Values{}
	params.Add("subject", claims.Subject)
	params.Add("resource", claims.Resource)
	params.Add("exp", claims.Exp)
	params.Add("iat", claims.Iat)
	return params.Encode() + "\n"
}

// HashClaims returns a hash of the claims data
func HashClaimsSha256(claims *Claims) ([]byte, error) {
	strToHash := buildCanonicalString(claims)
	return []byte(hex.EncodeToString(sha256hash([]byte(strToHash)))), nil
}
