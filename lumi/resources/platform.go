package resources

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io/ioutil"
	"strings"

	"github.com/rs/zerolog/log"
	"go.mondoo.io/mondoo/lumi"
	"go.mondoo.io/mondoo/lumi/resources/platformid"
)

func (s *lumiPlatform) init(args *lumi.Args) (*lumi.Args, error) {
	platform, err := s.Runtime.Motor.Platform()
	if err == nil {
		(*args)["name"] = platform.Name
		(*args)["title"] = platform.Title
		(*args)["arch"] = platform.Arch
		(*args)["release"] = platform.Release

		families := []interface{}{}
		for _, f := range platform.Family {
			families = append(families, f)
		}
		(*args)["family"] = families

	} else {
		log.Error().Err(err).Msg("could not determine platform")
	}
	return args, nil
}

func (s *lumiPlatform) id() (string, error) {
	return "platform", nil
}

func (s *lumiPlatform) GetHostname() (string, error) {
	c, err := s.Runtime.Motor.Transport.RunCommand("hostname")
	if err != nil || c.ExitStatus != 0 {
		return "", errors.New("lumi[platform]> cannot determine hostname")
	}

	res, err := ioutil.ReadAll(c.Stdout)
	return strings.TrimSpace(string(res)), nil
}

// returns the OS native machine UUID/GUID
func (s *lumiPlatform) GetUuid() (string, error) {
	platform, err := s.Runtime.Motor.Platform()
	if err != nil {
		return "", errors.New("cannot determine platform uuid")
	}

	var uuidProvider platformid.UniquePlatformIDProvider
	for i := range platform.Family {
		if platform.Family[i] == "linux" {
			uuidProvider = &platformid.LinuxIdProvider{Motor: s.Runtime.Motor}
		}
	}

	if uuidProvider == nil && platform.Name == "mac_os_x" {
		uuidProvider = &platformid.MacOSIdProvider{Motor: s.Runtime.Motor}
	}

	if uuidProvider == nil {
		return "", errors.New("cannot determine platform uuid for " + platform.Name)
	}

	id, err := uuidProvider.ID()
	if err != nil {
		return "", errors.New("cannot determine platform uuid on known system " + platform.Name)
	}

	// TODO: we may want to inject that during compile time
	return HashedMachineID("3zXPqBRdu2zyspzplk7gxi1LEveYBrY0hdgCYv4M", id), nil
}

// We use a mechanism established by https://github.com/denisbrodbeck/machineid to
// derive the platform id in a reliable manner but we are not exposing the machine secret
func HashedMachineID(secret, id string) string {
	mac := hmac.New(sha256.New, []byte(id))
	mac.Write([]byte(secret))
	return hex.EncodeToString(mac.Sum(nil))
}
