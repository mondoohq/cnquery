package provider

import (
	"errors"

	"go.mondoo.com/cnquery/motor/asset"
	"go.mondoo.com/cnquery/providers/os/detector"
)

func (s *Service) detect(asset *asset.Asset) error {
	conn, err := s.connect(asset)
	if err != nil {
		return err
	}

	var ok bool
	asset.Platform, ok = detector.Detect(conn)
	if !ok {
		return errors.New("failed to detect OS")
	}

	return nil
}
