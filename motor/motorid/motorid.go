package motorid

import (
	"fmt"
	"strings"

	"github.com/rs/zerolog/log"
	"go.mondoo.io/mondoo/stringx"

	"github.com/cockroachdb/errors"
	"go.mondoo.io/mondoo/motor/motorid/awsec2"
	"go.mondoo.io/mondoo/motor/motorid/clouddetect"
	"go.mondoo.io/mondoo/motor/motorid/hostname"
	"go.mondoo.io/mondoo/motor/motorid/machineid"
	"go.mondoo.io/mondoo/motor/platform"
	"go.mondoo.io/mondoo/motor/transports"
	"go.mondoo.io/mondoo/motor/transports/mock"
	"go.mondoo.io/mondoo/motor/transports/ssh"
	gossh "golang.org/x/crypto/ssh"
)

func GatherIDs(t transports.Transport, p *platform.Platform, idDetectors []string) ([]string, error) {
	// sanitize ids
	idDetectors = stringx.RemoveEmpty(idDetectors)

	var ids []string
	for i := range idDetectors {
		idDetector := idDetectors[i]
		id, err := GatherID(t, p, idDetector)
		if err != nil {
			// we only err if we found zero platform ids, if we try multiple, a fail of an individual one is okay
			log.Debug().Err(err).Str("detector", idDetector).Msg("could not determine platform id")
			continue
		}

		if len(id) > 0 {
			ids = append(ids, id)
		}
	}

	// if we found zero platform ids something went wrong
	if len(ids) == 0 {
		return nil, errors.New("could not determine a platform identifier")
	}

	log.Debug().Strs("id-detector", idDetectors).Strs("platform-ids", ids).Msg("detected platform ids")

	return ids, nil
}

func GatherID(t transports.Transport, p *platform.Platform, idDetector string) (string, error) {
	transport := t
	// helper for recoding transport to extract the original transport
	recT, ok := t.(*mock.RecordTransport)
	if ok {
		transport = recT.Watched()
	}

	var identifier string
	switch idDetector {
	case "hostname":
		// NOTE: we need to be careful with hostname's since they are not required to be unique
		hostname, hostErr := hostname.Hostname(t, p)
		if hostErr == nil && len(hostname) > 0 {
			identifier = "//platformid.api.mondoo.app/hostname/" + hostname
		}
		return identifier, hostErr
	case "machineid":
		guid, hostErr := machineid.MachineId(t, p)
		if hostErr == nil && len(guid) > 0 {
			identifier = "//platformid.api.mondoo.app/machineid/" + guid
		}
		return identifier, hostErr
	case "ssh-hostkey":
		sshTrans, ok := transport.(*ssh.SSHTransport)
		if !ok {
			return "", errors.New("ssh-hostkey id detector is not supported for the transport")
		}
		if sshTrans != nil {
			fingerprint := gossh.FingerprintSHA256(sshTrans.HostKey)
			fingerprint = strings.Replace(fingerprint, ":", "-", 1)
			identifier = "//platformid.api.mondoo.app/runtime/ssh/hostkey/" + fingerprint
		}
		return identifier, nil
	case "awsec2":
		metadata, err := awsec2.Resolve(transport, p)
		if err != nil {
			return "", err
		}
		return metadata.InstanceID()
	case "clouddetect":
		identifier := clouddetect.Detect(t, p)
		return identifier, nil
	default:
		return "", errors.New(fmt.Sprintf("the provided id-detector is not supported: %s", idDetector))
	}
}
