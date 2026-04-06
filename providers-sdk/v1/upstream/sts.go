// Copyright Mondoo, Inc. 2024, 2026
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
	"os"
	"strings"
	"time"

	signerv4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"go.mondoo.com/mql/v13/providers/os/connection/ssh/signers"
	"go.mondoo.com/mql/v13/utils/sortx"
	"go.mondoo.com/ranger-rpc"
	"golang.org/x/crypto/ssh"
)

const (
	// we use global STS endpoint and it maps to us-east-1
	awsStsURL     = "https://sts.amazonaws.com/?Action=GetCallerIdentity&Version=2011-06-15"
	awsStsRegion  = "us-east-1"
	awsStsService = "sts"
	awsStsBody    = "Action=GetCallerIdentity&Version=2011-06-15"
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

func ExchangeExternalToken(apiEndpoint, audience, issuerURI, jwtToken string) (*ServiceAccountCredentials, error) {
	if jwtToken == "" {
		// Try to fetch the token from an environment variable
		jwtToken = os.Getenv("JWT_TOKEN")
	}

	if jwtToken == "" {
		// Try to fetch the identity token from the cloud provider
		jsonToken, err := fetchIdentityToken(audience)
		if err != nil {
			return nil, err
		}
		jwtToken = jsonToken
	}

	if jwtToken == "" {
		return nil, fmt.Errorf("no identity token to use for an external exchange")
	}

	stsClient, err := NewSecureTokenServiceClient(apiEndpoint, ranger.DefaultHttpClient())
	if err != nil {
		return nil, err
	}

	request := &ExchangeExternalTokenRequest{
		Audience:  audience,
		IssuerUri: issuerURI,
		JwtToken:  jwtToken,
	}
	resp, err := stsClient.ExchangeExternalToken(context.Background(), request)
	if err != nil {
		return nil, err
	}

	// Decode the base64 credential string
	credBytes, err := base64.StdEncoding.DecodeString(resp.Base64Credential)
	if err != nil {
		return nil, err
	}

	// First unmarshal to a temporary structure to handle the field name mismatch
	var tempCreds struct {
		Mrn         string `json:"mrn"`
		ParentMrn   string `json:"parent_mrn"`
		SpaceMrn    string `json:"space_mrn"`
		PrivateKey  string `json:"private_key"`
		Certificate string `json:"certificate"`
		ApiEndpoint string `json:"api_endpoint"`
	}

	if err := json.Unmarshal(credBytes, &tempCreds); err != nil {
		return nil, err
	}

	// Create the ServiceAccountCredentials with the correct field mapping
	creds := ServiceAccountCredentials{
		Mrn:         tempCreds.Mrn,
		ParentMrn:   tempCreds.SpaceMrn,
		ScopeMrn:    tempCreds.SpaceMrn, // Map SpaceMrn to ScopeMrn
		PrivateKey:  tempCreds.PrivateKey,
		Certificate: tempCreds.Certificate,
		ApiEndpoint: tempCreds.ApiEndpoint,
	}

	return &creds, nil
}

// fetchIdentityToken fetches an identity token from the current cloud environment
// It supports GCP, Azure, AWS and GitHub Actions
func fetchIdentityToken(audience string) (string, error) {
	// Try GCP
	if token, err := fetchGCPIdentityToken(audience); err == nil {
		return token, nil
	}

	// Try Azure
	if token, err := fetchAzureIdentityToken(audience); err == nil {
		return token, nil
	}

	// Try GitHub Actions
	if token, err := fetchGitHubActionsIdentityToken(audience); err == nil {
		return token, nil
	}

	// Try AWS
	if token, err := fetchAWSIdentityToken(); err == nil {
		return token, nil
	}

	return "", fmt.Errorf("failed to fetch identity token from any supported cloud provider")
}

// fetchGCPIdentityToken fetches an identity token from GCP metadata service
func fetchGCPIdentityToken(audience string) (string, error) {
	url := fmt.Sprintf("http://metadata.google.internal/computeMetadata/v1/instance/service-accounts/default/identity?audience=%s", audience)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Add("Metadata-Flavor", "Google")

	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("gcp metadata service returned non-OK status: %d", resp.StatusCode)
	}

	tokenBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(tokenBytes), nil
}

// fetchAzureIdentityToken fetches an identity token from Azure IMDS
func fetchAzureIdentityToken(audience string) (string, error) {
	reqUrl := "http://localhost:50342/oauth2/token"
	data := make(url.Values)
	data.Set("resource", audience)

	req, err := http.NewRequest("POST", reqUrl, strings.NewReader(data.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Add("Metadata", "true")
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("azure IMDS returned non-OK status: %d", resp.StatusCode)
	}

	var result struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	return result.AccessToken, nil
}

// fetchGitHubActionsIdentityToken fetches an identity token from GitHub Actions
func fetchGitHubActionsIdentityToken(audience string) (string, error) {
	tokenRequestToken := os.Getenv("ACTIONS_ID_TOKEN_REQUEST_TOKEN")
	tokenRequestURL := os.Getenv("ACTIONS_ID_TOKEN_REQUEST_URL")

	if tokenRequestToken == "" || tokenRequestURL == "" {
		return "", fmt.Errorf("github Actions environment variables not set")
	}

	url := fmt.Sprintf("%s&audience=%s", tokenRequestURL, audience)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Add("Authorization", "bearer "+tokenRequestToken)

	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("github Actions token service returned non-OK status: %d", resp.StatusCode)
	}

	var result struct {
		Value string `json:"value"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	return result.Value, nil
}

// fetchAWSIdentityToken generates a pre-signed STS GetCallerIdentity POST request
// and returns it as a JSON object (token) for the AWS Workload Identity Federation
// token exchange flow. an audience is not necessary for AWS because the token is
// not a JWT, rather a pre-signed request as a JSON object.
func fetchAWSIdentityToken() (string, error) {
	ctx := context.Background()
	cfg, err := awsconfig.LoadDefaultConfig(ctx)
	if err != nil {
		return "", err
	}

	creds, err := cfg.Credentials.Retrieve(ctx)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", awsStsURL, strings.NewReader(awsStsBody))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	bodyHash := sha256.Sum256([]byte(awsStsBody))
	err = signerv4.NewSigner().SignHTTP(
		ctx,
		creds,
		req,
		hex.EncodeToString(bodyHash[:]),
		awsStsService,
		awsStsRegion,
		time.Now(),
	)
	if err != nil {
		return "", err
	}

	// SignHTTP attaches the following headers to the request:
	// - Authorization: SigV4 signature (algorithm, credential scope, signed headers, signature)
	// - X-Amz-Date: timestamp used during signing; AWS rejects requests older than ~15 minutes
	// - X-Amz-Security-Token: session token for temporary credentials (SSO, assumed roles, IMDS)
	headers := []map[string]string{}

	// Attach the host header as well, in case it is needed to reconstruct the signed request on the server side.
	if req.Host != "" {
		headers = append(headers, map[string]string{"key": "host", "value": req.Host})
	}

	for _, key := range sortx.Keys(req.Header) {
		values := req.Header.Values(key)
		if len(values) == 0 {
			continue
		}
		headers = append(headers, map[string]string{"key": key, "value": values[0]})
	}

	token := map[string]any{
		"method":  "POST",
		"url":     awsStsURL,
		"headers": headers,
	}
	tokenBytes, err := json.Marshal(token)
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
