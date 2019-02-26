package filesystem

import (
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"io"

	"github.com/spf13/afero"
)

func HashMd5(f afero.File) (string, error) {
	h := md5.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}

	return hex.EncodeToString(h.Sum(nil)), nil

}
func HashSha256(f afero.File) (string, error) {
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}
