package dnsshake

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

const (
	pubKey = `MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAx3E9IavfvGHiENM/bFBTJfRLBUE1PV9f2q2mbYOHu2d1zZ3VB22sXnpGN6TV1m8Tq8zUWlXPgkApOaSF/+zRqBuyF6ci1rmcfvFCAHdERXy37bFgi0/EkoslaqEZel4eddqqWt93KuwydPL2jEhd01M+PGbfFfCu65iZFW107u0PhlXWZG0iJbFsBNdp4mKXI4CxWNlVb0xPr0kcYaE0eAi+EcnG5QHONv5cQrQJ6ncUNehV0caUKWibIKTKPmwttPTyTYbF6sWY7olT9FAgbGz5flHHqBVWPXsf5Jivv5HbsJLTdejAvQwm7e+w0S//OFafffZUXgF/yNB4HczZiQIDAQAB`
)

func TestDkimPublicKeyRepresentation(t *testing.T) {
	type test struct {
		Title        string
		DnsTxtRecord string
		Expected     *DkimPublicKeyRepresentation
		ParseErr     error
		IsValid      bool
	}

	// test cases for https://datatracker.ietf.org/doc/html/rfc6376#section-3.6.1
	testCases := []test{
		{
			Title:        "minimal valid dkim",
			DnsTxtRecord: "p=" + pubKey,
			Expected: &DkimPublicKeyRepresentation{
				PublicKeyData: pubKey,
			},
			IsValid: true,
		},
		{
			Title:        "empty dkim record",
			DnsTxtRecord: "",
			Expected:     &DkimPublicKeyRepresentation{},
			IsValid:      false,
		},
		//  v= Version of the DKIM key record (plain-text; RECOMMENDED, default is "DKIM1")
		{
			Title:        "v tag MUST be the first tag",
			DnsTxtRecord: "p=" + pubKey + "; v=DKIM1",
			Expected:     nil,
			ParseErr:     errors.New("invalid DKIM record"),
			IsValid:      false,
		},
		{
			Title:        "invalid DKIM version",
			DnsTxtRecord: "v=DKIM1.0; p=" + pubKey,
			Expected: &DkimPublicKeyRepresentation{
				Version:       "DKIM1.0",
				PublicKeyData: pubKey,
			},
			IsValid: false,
		},
		// h= Acceptable hash algorithms (plain-text; OPTIONAL, defaults to allowing all algorithms)
		{
			Title:        "valid hash algorithms",
			DnsTxtRecord: "v=DKIM1; h=sha1:sha256; p=" + pubKey,
			Expected: &DkimPublicKeyRepresentation{
				Version:        "DKIM1",
				HashAlgorithms: []string{"sha1", "sha256"},
				PublicKeyData:  pubKey,
			},
			IsValid: true,
		},
		{
			Title:        "single hash algorithm",
			DnsTxtRecord: "v=DKIM1; h=sha256; p=" + pubKey,
			Expected: &DkimPublicKeyRepresentation{
				Version:        "DKIM1",
				HashAlgorithms: []string{"sha256"},
				PublicKeyData:  pubKey,
			},
			IsValid: true,
		},
		{
			Title:        "unsupported hash algorithm",
			DnsTxtRecord: "v=DKIM1; h=sha512; p=" + pubKey,
			Expected: &DkimPublicKeyRepresentation{
				Version:        "DKIM1",
				HashAlgorithms: []string{"sha512"},
				PublicKeyData:  pubKey,
			},
			IsValid: true, // still valid according to RFC: Unrecognized algorithms MUST be ignored
		},
		{
			Title:        "empty hash list",
			DnsTxtRecord: "v=DKIM1; h=; p=" + pubKey,
			Expected: &DkimPublicKeyRepresentation{
				Version:        "DKIM1",
				HashAlgorithms: []string{""},
				PublicKeyData:  pubKey,
			},
			IsValid: true, // still valid according to RFC: defaults to allowing all algorithms
		},
		// k= Key type (plain-text; OPTIONAL, default is "rsa")
		{
			Title:        "valid DKIM with rsa key type and public key",
			DnsTxtRecord: "v=DKIM1; k=rsa; p=" + pubKey,
			Expected: &DkimPublicKeyRepresentation{
				Version:       "DKIM1",
				KeyType:       "rsa",
				PublicKeyData: pubKey,
			},
			IsValid: true,
		},
		{
			Title:        "unsupported public key type",
			DnsTxtRecord: "v=DKIM1; k=dsa; p=" + pubKey,
			Expected: &DkimPublicKeyRepresentation{
				Version:       "DKIM1",
				KeyType:       "dsa",
				PublicKeyData: pubKey,
			},
			IsValid: false,
		},
		{
			Title:        "empty_key_type",
			DnsTxtRecord: "v=DKIM1; k=; p=" + pubKey,
			Expected: &DkimPublicKeyRepresentation{
				Version:       "DKIM1",
				PublicKeyData: pubKey,
			},
			IsValid: true, // valid since it defaults to rsa
		},
		// n= Notes that might be of interest to a human
		{
			Title:        "empty note",
			DnsTxtRecord: "v=DKIM1; n=; p=" + pubKey,
			Expected: &DkimPublicKeyRepresentation{
				Version:       "DKIM1",
				PublicKeyData: pubKey,
			},
			IsValid: true,
		},
		{
			Title:        "with simple note",
			DnsTxtRecord: "v=DKIM1; n=a note; p=" + pubKey,
			Expected: &DkimPublicKeyRepresentation{
				Version:       "DKIM1",
				Notes:         "a note",
				PublicKeyData: pubKey,
			},
			IsValid: true,
		},
		{
			// see https://en.wikipedia.org/wiki/Quoted-printable
			Title:        "quoted printable note",
			DnsTxtRecord: "v=DKIM1; n=H=E4tten H=FCte ein =DF im Namen, w=E4ren sie m=F6glicherweise keine H=FCte= mehr,sondern H=FC=DFe.; p=" + pubKey,
			Expected: &DkimPublicKeyRepresentation{
				Version:       "DKIM1",
				Notes:         "H\xe4tten H\xfcte ein \xdf im Namen, w\xe4ren sie m\xf6glicherweise keine H\xfcte= mehr,sondern H\xfc\xdfe.",
				PublicKeyData: pubKey,
			},
			IsValid: true,
		},
		{
			Title:        "uninterpreted note",
			DnsTxtRecord: "v=DKIM1; n=Hätten; p=" + pubKey,
			Expected: &DkimPublicKeyRepresentation{
				Version:       "DKIM1",
				Notes:         "Hätten",
				PublicKeyData: pubKey,
			},
			IsValid: true,
		},
		// p= Public-key data (base64; REQUIRED)
		{
			Title:        "missing public key",
			DnsTxtRecord: "v=DKIM1",
			Expected: &DkimPublicKeyRepresentation{
				Version: "DKIM1",
			},
			IsValid: false,
		},
		{
			Title:        "revoked public key",
			DnsTxtRecord: "v=DKIM1; p=",
			Expected: &DkimPublicKeyRepresentation{
				Version: "DKIM1",
			},
			IsValid: false,
		},
		{
			Title:        "invalid base64 public key",
			DnsTxtRecord: "v=DKIM1; p=invalidBase64key",
			Expected: &DkimPublicKeyRepresentation{
				Version:       "DKIM1",
				PublicKeyData: "invalidBase64key",
			},
			IsValid: false,
		},
		// s= Service Type (plain-text; OPTIONAL; default is "*")
		{
			Title:        "matches all service types",
			DnsTxtRecord: "v=DKIM1; s=*; p=" + pubKey,
			Expected: &DkimPublicKeyRepresentation{
				Version:       "DKIM1",
				ServiceType:   []string{"*"},
				PublicKeyData: pubKey,
			},
			IsValid: true,
		},
		{
			Title:        "email service type",
			DnsTxtRecord: "v=DKIM1; s=email; p=" + pubKey,
			Expected: &DkimPublicKeyRepresentation{
				Version:       "DKIM1",
				ServiceType:   []string{"email"},
				PublicKeyData: pubKey,
			},
			IsValid: true,
		},
		{
			Title:        "unsupported service type",
			DnsTxtRecord: "v=DKIM1; s=unsupported; p=" + pubKey,
			Expected: &DkimPublicKeyRepresentation{
				Version:       "DKIM1",
				ServiceType:   []string{"unsupported"},
				PublicKeyData: pubKey,
			},
			IsValid: true,
		},
		{
			Title:        "colon seperated service type list with supported and unsupported entries",
			DnsTxtRecord: "v=DKIM1; s=email:unsupported; p=" + pubKey,
			Expected: &DkimPublicKeyRepresentation{
				Version:       "DKIM1",
				ServiceType:   []string{"email", "unsupported"},
				PublicKeyData: pubKey,
			},
			IsValid: true,
		},
		{
			Title:        "empty services types",
			DnsTxtRecord: "v=DKIM1; s=; p=" + pubKey,
			Expected: &DkimPublicKeyRepresentation{
				Version:       "DKIM1",
				ServiceType:   []string{""},
				PublicKeyData: pubKey,
			},
			IsValid: true,
		},

		// t= Flags, represented as a colon-separated list of name
		{
			Title:        "testing mode flag",
			DnsTxtRecord: "v=DKIM1; t=y; p=" + pubKey,
			Expected: &DkimPublicKeyRepresentation{
				Version:       "DKIM1",
				PublicKeyData: pubKey,
				Flags:         []string{"y"},
			},
			IsValid: true,
		},
		{
			Title:        "include invalid test flag",
			DnsTxtRecord: "v=DKIM1; t=y:s:?; p=" + pubKey,
			Expected: &DkimPublicKeyRepresentation{
				Version:        "DKIM1",
				HashAlgorithms: nil,
				KeyType:        "",
				ServiceType:    nil,
				PublicKeyData:  pubKey,
				Flags:          []string{"y", "s", "?"},
			},
			IsValid: true,
		},
		{
			Title:        "no flags",
			DnsTxtRecord: "v=DKIM1; t=; p=" + pubKey,
			Expected: &DkimPublicKeyRepresentation{
				Version:       "DKIM1",
				PublicKeyData: pubKey,
				Flags:         []string{""},
			},
			IsValid: true,
		},
	}

	for i := range testCases {
		tc := testCases[i]
		t.Run(tc.Title, func(t *testing.T) {
			// parse dns entry
			pubKeyRep, err := NewDkimPublicKeyRepresentation(tc.DnsTxtRecord)
			if tc.ParseErr != nil {
				assert.EqualError(t, err, tc.ParseErr.Error())
			} else {
				assert.NoError(t, err)
			}

			// check that the data was parsed as expected
			assert.EqualValues(t, tc.Expected, pubKeyRep)

			// check if validation is successful
			if pubKeyRep != nil {
				valid, _, _ := pubKeyRep.Valid()
				assert.Equal(t, tc.IsValid, valid)
			}
		})
	}
}
