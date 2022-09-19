package providers

//go:generate protoc --proto_path=../..:. --go_out=. --go_opt=paths=source_relative --rangerrpc_out=. provider.proto

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

type Instance interface {
	PlatformIdDetectors() []PlatformIdDetector

	// returns if this is a static asset that does not allow run command
	Capabilities() Capabilities

	Kind() Kind
	Runtime() string

	// Close closes the transport
	Close()
}

type PlatformIdentifier interface {
	Identifier() (string, error)
}
