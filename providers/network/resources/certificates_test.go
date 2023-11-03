package resources

import (
	"crypto/x509/pkix"
	"encoding/asn1"
	"testing"
)

func TestExtensionValueToReadableFormat(t *testing.T) {
	// ASN.1-encoded DNS name "example.com" for the SAN extension.
	sanValue, err := asn1.Marshal([]asn1.RawValue{
		{
			Tag:   asn1.TagIA5String, // TagIA5String is commonly used for DNS names
			Bytes: []byte("example.com"),
		},
	})
	if err != nil {
		t.Fatalf("Error marshaling test SAN value: %v", err)
	}

	testCases := []struct {
		name        string
		extension   pkix.Extension
		want        string
		expectError bool
	}{
		{
			name: "SubjectKeyIdentifier",
			extension: pkix.Extension{
				Id:    asn1.ObjectIdentifier{2, 5, 29, 14},
				Value: []byte{0x04, 0x04, 0xDE, 0xAD, 0xBE, 0xEF},
			},
			want:        "DE:AD:BE:EF",
			expectError: false,
		},
		{
			name: "SubjectAlternativeName",
			extension: pkix.Extension{
				Id:    asn1.ObjectIdentifier{2, 5, 29, 17},
				Value: sanValue,
			},
			want:        "example.com",
			expectError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ExtensionValueToReadableFormat(tc.extension)
			if (err != nil) != tc.expectError {
				t.Errorf("ExtensionValueToReadableFormat() for test '%v' unexpected error = %v", tc.name, err)
				return
			}
			if got != tc.want {
				t.Errorf("ExtensionValueToReadableFormat() for test '%v' = %v, want %v", tc.name, got, tc.want)
			}
		})
	}
}
