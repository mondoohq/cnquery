package apk

import (
	apk_version "github.com/knqyf263/go-apk-version"
)

type Parser struct{}

func (p Parser) Compare(a, b string) (int, error) {
	v1, err := apk_version.NewVersion(a)
	if err != nil {
		return 0, err
	}

	v2, err := apk_version.NewVersion(b)
	if err != nil {
		return 0, err
	}

	// Quick check
	if v1 == v2 {
		return 0, nil
	}

	return v1.Compare(v2), nil
}
