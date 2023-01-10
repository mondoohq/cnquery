package upstream

import (
	"time"

	unverified_jwt "github.com/golang-jwt/jwt"
	"gopkg.in/square/go-jose.v2/jwt"
)

type CustomTokenClaims struct {
	Space          string            `json:"space"`
	Description    string            `json:"desc"`
	ApiEndpoint    string            `json:"api_endpoint"`
	Labels         map[string]string `json:"labels"`
	Owner          string            `json:"owner"`
	CertValidUntil time.Time         `json:"cert_valid_until"`
}

type VerifyClaim struct {
	jwt.Claims
	CustomTokenClaims
}

func (a *VerifyClaim) IsExpired() bool {
	if a.Expiry != nil && time.Now().After(a.Expiry.Time()) {
		return true
	}
	return false
}

type extractTokenClaims struct {
	// TODO: workaround for https://github.com/dgrijalva/jwt-go/pull/308
	Aud []string `json:"aud"`
	unverified_jwt.StandardClaims
	CustomTokenClaims
}

// ExtractTokenClaims is just reading the jwt token and extracts the claims
// This is especially useful for the client that has no access to the certificate
// to verify the token but still want to display information like expiry time and description
func ExtractTokenClaims(token string) (*VerifyClaim, error) {
	unverifiedClaims := &extractTokenClaims{}
	p := unverified_jwt.Parser{}
	_, _, err := p.ParseUnverified(token, unverifiedClaims)
	if err != nil {
		return nil, err
	}

	// convert to AmsVerifyClaim
	var expiry *jwt.NumericDate
	if unverifiedClaims.ExpiresAt > 0 {
		nd := jwt.NumericDate(unverifiedClaims.ExpiresAt)
		expiry = &nd
	}

	var notBefore *jwt.NumericDate
	if unverifiedClaims.NotBefore > 0 {
		nd := jwt.NumericDate(unverifiedClaims.NotBefore)
		notBefore = &nd
	}

	var issuedAt *jwt.NumericDate
	if unverifiedClaims.IssuedAt > 0 {
		nd := jwt.NumericDate(unverifiedClaims.IssuedAt)
		notBefore = &nd
	}

	out := VerifyClaim{
		Claims: jwt.Claims{
			ID:        unverifiedClaims.Id,
			Issuer:    unverifiedClaims.Issuer,
			Subject:   unverifiedClaims.Subject,
			Audience:  jwt.Audience([]string{unverifiedClaims.Audience}),
			Expiry:    expiry,
			NotBefore: notBefore,
			IssuedAt:  issuedAt,
		},
		CustomTokenClaims: unverifiedClaims.CustomTokenClaims,
	}

	return &out, nil
}
