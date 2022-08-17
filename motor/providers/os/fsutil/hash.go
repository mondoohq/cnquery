package fsutil

import (
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"io"

	"github.com/spf13/afero"
)

func Md5(f afero.File) (string, error) {
	h := md5.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}

func Sha256(f afero.File) (string, error) {
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}

// LocalFileSha256 determines the hashsum for a local file
func LocalFileSha256(filename string) (string, error) {
	osFs := afero.NewOsFs()
	f, err := osFs.Open(filename)
	if err != nil {
		return "", err
	}

	defer f.Close()
	hash, err := Sha256(f)
	if err != nil {
		return "", err
	}
	return hash, nil
}
