package db

import (
	"encoding/base64"

	"golang.org/x/crypto/blake2b"
)

func (c *Query) checksum() string {
	originalID := c.Id
	c.Id = ""

	data, err := c.Marshal()
	if err != nil {
		panic("Failed to marshal Query for checksum calculation. Critical failure.")
	}

	c.Id = originalID

	hash := blake2b.Sum512(data)
	return base64.StdEncoding.EncodeToString(hash[:])
}

// UpdateID sets a new computed ID
func (c *Query) UpdateID() {
	c.Id = c.checksum()
}
