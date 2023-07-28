package semver

import (
	mastermind "github.com/Masterminds/semver"
)

type Parser struct{}

func (p Parser) Compare(a, b string) (int, error) {
	va, err := mastermind.NewVersion(a)
	if err != nil {
		return 0, err
	}
	vb, err := mastermind.NewVersion(b)
	if err != nil {
		return 0, err
	}

	return va.Compare(vb), nil
}
