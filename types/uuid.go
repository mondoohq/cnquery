package types

import (
	"encoding/base64"

	"github.com/gofrs/uuid"
)

// UUID generates a new string encoded UUID
func UUID() string {
	b := uuid.Must(uuid.NewV4()).Bytes()
	return base64.StdEncoding.EncodeToString(b)
}
