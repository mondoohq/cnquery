package providers

//go:generate protoc --proto_path=../..:. --go_out=. --go_opt=paths=source_relative --rangerrpc_out=. --iam-actions_out=. provider.proto

import (
	"regexp"

	"github.com/spf13/afero"
)

type PlatformIdDetector string

const (
	HostnameDetector  PlatformIdDetector = "hostname"
	MachineIdDetector PlatformIdDetector = "machine-id"
	CloudDetector     PlatformIdDetector = "cloud-detect"
	AWSEc2Detector    PlatformIdDetector = "aws-ec2"
	SshHostKey        PlatformIdDetector = "ssh-host-key"
	// TransportPlatformIdentifierDetector is a detector that gets the platform id
	// from the transports Identifier() method. This requires the
	// TransportIdentifier interface be implemented for the transport
	TransportPlatformIdentifierDetector PlatformIdDetector = "transport-platform-id"
)

func AvailablePlatformIdDetector() []string {
	return []string{HostnameDetector.String(), MachineIdDetector.String(), AWSEc2Detector.String(), CloudDetector.String(), SshHostKey.String(), TransportPlatformIdentifierDetector.String()}
}

var platformIdAliases = map[string]PlatformIdDetector{
	"awsec2":    AWSEc2Detector,
	"machineid": MachineIdDetector,
}

func (id PlatformIdDetector) String() string {
	return string(id)
}

func ToPlatformIdDetectors(idDetectors []string) []PlatformIdDetector {
	idDetectorsCopy := make([]PlatformIdDetector, len(idDetectors))
	for i, v := range idDetectors {
		if detector, ok := platformIdAliases[v]; ok {
			idDetectorsCopy[i] = detector
		} else {
			idDetectorsCopy[i] = PlatformIdDetector(v)
		}
	}
	return idDetectorsCopy
}

type Transport interface {
	// RunCommand executes a command on the target system
	RunCommand(command string) (*Command, error)
	// returns file permissions and ownership
	FileInfo(path string) (FileInfoDetails, error)
	// FS provides access to the file system of the target system
	FS() afero.Fs
	// Close closes the transport
	Close()
	// returns if this is a static asset that does not allow run command
	Capabilities() Capabilities

	Kind() Kind
	Runtime() string

	PlatformIdDetectors() []PlatformIdDetector
}

type TransportPlatformIdentifier interface {
	Identifier() (string, error)
}

type FileSearch interface {
	Find(from string, r *regexp.Regexp, typ string) ([]string, error)
}
