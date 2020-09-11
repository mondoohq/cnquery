package motorid

import (
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws/external"
	"github.com/cockroachdb/errors"
	"go.mondoo.io/mondoo/motor/motorid/awsec2"
	"go.mondoo.io/mondoo/motor/motorid/hostname"
	"go.mondoo.io/mondoo/motor/motorid/machineid"
	"go.mondoo.io/mondoo/motor/platform"
	"go.mondoo.io/mondoo/motor/transports"
	"go.mondoo.io/mondoo/motor/transports/local"
	"go.mondoo.io/mondoo/motor/transports/mock"
	"go.mondoo.io/mondoo/motor/transports/ssh"
	gossh "golang.org/x/crypto/ssh"
)

func GatherIDs(t transports.Transport, p *platform.Platform, idDetectors []string) ([]string, error) {
	var ids []string
	for i := range idDetectors {
		if len(idDetectors[i]) == 0 {
			continue
		}
		id, err := GatherID(t, p, idDetectors[i])
		if err != nil {
			return nil, err
		}

		if len(id) > 0 {
			ids = append(ids, id)
		}
	}

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
	case "machineid":
		guid, hostErr := machineid.MachineId(t, p)
		if hostErr == nil && len(guid) > 0 {
			identifier = "//platformid.api.mondoo.app/machineid/" + guid
		}
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
	case "awsec2":
		_, ok := transport.(*local.LocalTransport)
		if ok {
			cfg, err := external.LoadDefaultAWSConfig()
			if err != nil {
				return "", errors.Wrap(err, "cannot not determine aws environment")
			}
			metadata := awsec2.NewLocal(cfg)
			mrn, err := metadata.InstanceID()
			if err != nil {
				return "", errors.Wrap(err, "cannot not determine aws ec2 instance id")
			}
			identifier = mrn
		} else {
			if p.IsFamily(platform.FAMILY_LINUX) {
				metadata := awsec2.NewUnix(t, p)
				mrn, err := metadata.InstanceID()
				if err != nil {
					return "", errors.Wrap(err, "cannot not determine aws ec2 instance id")
				}
				identifier = mrn
			} else {
				return "", errors.New(fmt.Sprintf("awsec2 id detector is not supported for your asset: %s %s", p.Name, p.Release))
			}
		}
	default:
		return "", errors.New(fmt.Sprintf("the provided id-detector is not supported: %s", idDetector))
	}
	return identifier, nil
}
