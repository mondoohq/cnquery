package provider

import (
	"errors"

	"go.mondoo.com/cnquery/motor/asset"
	"go.mondoo.com/cnquery/providers/os/detector"
	"go.mondoo.com/cnquery/providers/os/id/hostname"
	"go.mondoo.com/cnquery/providers/plugin"
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

	asset.PlatformIds = plugin.IdentifyPlatform(
		conn, asset.Platform,
		hostname.Hostname,
	)

	return nil
}
