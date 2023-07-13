package lsof

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
)

func init() {
	for i := range allFileTypes {
		ft := allFileTypes[i]
		fileTypeMap[string(ft)] = ft
	}
}

// FileType is the type of the node associated with the file
type FileType string

const (
	FileTypeUnknown          FileType = ""
	FileTypeIPv4             FileType = "IPv4"
	FileTypeIPv6             FileType = "IPv6"
	FileTypeAX25             FileType = "ax25"
	FileTypeInternetDomain   FileType = "inet"
	FileTypeHPLinkLevel      FileType = "lla"
	FileTypeAFRouteSocket    FileType = "rte"
	FileTypeSocket           FileType = "sock"
	FileTypeUnixDomainSocket FileType = "unix"
	FileTypeHPUXX25          FileType = "x.25"
	FileTypeBlockSpecial     FileType = "BLK"
	FileTypeCharacterSpecial FileType = "CHR"
	FileTypeLinuxMapFile     FileType = "DEL"
	FileTypeDir              FileType = "DIR"
	FileTypeDOOR             FileType = "DOOR"
	FileTypeFIFO             FileType = "FIFO"
	FileTypeKQUEUE           FileType = "KQUEUE"
	FileTypeSymbolicLink     FileType = "LINK"
	FileTypeRegularFile      FileType = "REG"
	FileTypeStreamSocket     FileType = "STSO"
	FileTypeUnnamedType      FileType = "UNNM"
)

var allFileTypes = []FileType{
	FileTypeUnknown,
	FileTypeIPv4,
	FileTypeIPv6,
	FileTypeAX25,
	FileTypeInternetDomain,
	FileTypeHPLinkLevel,
	FileTypeAFRouteSocket,
	FileTypeSocket,
	FileTypeUnixDomainSocket,
	FileTypeHPUXX25,
	FileTypeBlockSpecial,
	FileTypeCharacterSpecial,
	FileTypeLinuxMapFile,
	FileTypeDir,
	FileTypeDOOR,
	FileTypeFIFO,
	FileTypeKQUEUE,
	FileTypeSymbolicLink,
	FileTypeRegularFile,
	FileTypeStreamSocket,
	FileTypeUnnamedType,
}

var fileTypeMap = map[string]FileType{}

func fileTypeFromString(s string) FileType {
	ft, ok := fileTypeMap[s]
	if !ok {
		return FileTypeUnknown
	}
	return ft
}

// FileDescriptor defines a file in use by a process
type FileDescriptor struct {
	FileDescriptor string
	Type           FileType
	// file name, comment, Internet address
	Name                string
	AccessMode          string
	LockStatus          string
	Flags               string
	DeviceCharacterCode string
	Offset              string
	Protocol            string
	// QR
	TcpReadQueueSize string
	// QS
	TcpSendQueueSize string
	// SO
	TcpSocketOptions string
	// SS
	TcpSocketStates string
	// ST
	TcpConnectionState string
	// TF
	TcpFlags string
	// WR
	TcpWindowReadSize string
	// WW
	TcpWindowWriteSize string
}

// n127.0.0.1:3000->127.0.0.1:54335
var networkNameRegexp = regexp.MustCompile(`(.*):(.*)->(.*):(.*)`)
var networkLoopbackRegexp = regexp.MustCompile(`(.*):(.*)`)

// local and remote Internet addresses of a network file
func (f *FileDescriptor) NetworkFile() (string, int64, string, int64, error) {
	if f.Name == "no PCB" { // socket files that do not have a protocol block
		return "", 0, "", 0, nil
	}
	if f.Name == "*:*" {
		return "*", 0, "*", 0, nil
	}

	if strings.Contains(f.Name, "->") {
		matches := networkNameRegexp.FindStringSubmatch(f.Name)
		if len(matches) != 5 {
			return "", 0, "", 0, errors.New("network name not supported: " + f.Name)
		}

		localPort, err := strconv.Atoi(matches[2])
		if err != nil {
			return "", 0, "", 0, errors.New("network name not supported: " + f.Name)
		}

		remotePort, err := strconv.Atoi(matches[4])
		if err != nil {
			return "", 0, "", 0, errors.New("network name not supported: " + f.Name)
		}

		return matches[1], int64(localPort), matches[3], int64(remotePort), nil
	}

	// loop-back address [::1]:17223 or *:56863
	address := networkLoopbackRegexp.FindStringSubmatch(f.Name)
	if len(address) < 3 {
		return "", 0, "", 0, errors.New("network name not supported: " + f.Name)
	}
	localPort := 0
	var err error
	if address[2] != "*" {
		localPort, err = strconv.Atoi(address[2])
		if err != nil {
			return "", 0, "", 0, errors.New("network name not supported: " + f.Name)
		}
	}

	return address[1], int64(localPort), "", 0, nil
}

// maps lsof state to tcp states
var lsofTcpStateMapping = map[string]int64{
	"ESTABLISHED": 1,  // "established"
	"SYN_SENT":    2,  //  "syn sent"
	"SYN_RCDV":    3,  //  "syn recv"
	"FIN_WAIT1":   4,  //  "fin wait1"
	"FIN_WAIT_2":  5,  //   "fin wait2",
	"TIME_WAIT":   6,  //  "time wait",
	"CLOSED":      7,  // "close"
	"CLOSE_WAIT":  8,  //   "close wait"
	"LAST_ACK":    9,  //  "last ack"
	"LISTEN":      10, // "listen"
	"CLOSING":     11, // "closing"
	// "SYN_RCDV":    12, // "new syn recv"
}

func (f *FileDescriptor) TcpState() int64 {
	return lsofTcpStateMapping[f.TcpConnectionState]
}

var tcpInfoRegex = regexp.MustCompile(`(.*)=(.*)`)

// https://man7.org/linux/man-pages/man8/lsof.8.html#OUTPUT_FOR_OTHER_PROGRAMS
func (f *FileDescriptor) parseField(s string) error {
	key := s[0]
	value := s[1:]
	switch key {
	case 'f':
		f.FileDescriptor = value
	case 'a':
		f.AccessMode = value
	case 'l':
		f.LockStatus = value
	case 't':
		f.Type = fileTypeFromString(value)
	case 'G':
		f.Flags = value
	case 'd':
		f.DeviceCharacterCode = value
	case 'o':
		f.Offset = value
	case 'P':
		f.Protocol = value
	case 'n':
		f.Name = value
	case 'T':
		// TCP/TPI information
		// we need to parse it separately
		keyPair := tcpInfoRegex.FindStringSubmatch(value)
		switch keyPair[1] {
		case "QR":
			f.TcpReadQueueSize = keyPair[2]
		case "QS":
			f.TcpSendQueueSize = keyPair[2]
		case "SO":
			f.TcpSocketOptions = keyPair[2]
		case "SS":
			f.TcpSocketStates = keyPair[2]
		case "ST":
			f.TcpConnectionState = keyPair[2]
		case "TF":
			f.TcpFlags = keyPair[2]
		case "WR":
			f.TcpWindowReadSize = keyPair[2]
		case "WW":
			f.TcpWindowWriteSize = keyPair[2]
		}
	default:
		// nothing do to, skip all unsupported fields
	}

	return nil
}

type Process struct {
	PID             string
	UID             string
	GID             string
	Command         string
	ParentPID       string
	SocketInode     string
	FileDescriptors []FileDescriptor
}

// https://man7.org/linux/man-pages/man8/lsof.8.html#OUTPUT_FOR_OTHER_PROGRAMS
func (p *Process) parseField(s string) error {
	if s == "" {
		return fmt.Errorf("Empty field")
	}
	key := s[0]
	value := s[1:]
	switch key {
	case 'p':
		p.PID = value
	case 'R':
		p.ParentPID = value
	case 'g':
		p.GID = value
	case 'd':
		p.SocketInode = value
	case 'c':
		p.Command = value
	case 'u':
		p.UID = value
	default:
		// nothing do to, skip all unsupported fields
	}
	return nil
}

// parseFileLines parses all attributes until the next file is defined
func (p *Process) parseFileLines(lines []string) error {
	file := FileDescriptor{}
	for _, line := range lines {
		if strings.HasPrefix(line, "f") && file.FileDescriptor != "" {
			// New file
			p.FileDescriptors = append(p.FileDescriptors, file)
			file = FileDescriptor{}
		}
		err := file.parseField(line)
		if err != nil {
			return err
		}
	}
	if file.FileDescriptor != "" {
		p.FileDescriptors = append(p.FileDescriptors, file)
	}
	return nil
}

// parseProcessLines parses all entries for one process. All
// files reported by lsof are centered around a process. One
// process can include multiple file descriptors. An open port
// is a file as well.
func parseProcessLines(lines []string) (Process, error) {
	p := Process{}
	for index, line := range lines {
		if strings.HasPrefix(line, "f") {
			err := p.parseFileLines(lines[index:])
			if err != nil {
				return p, err
			}
		} else if strings.HasPrefix(line, "o") {
			break
		} else {
			err := p.parseField(line)
			if err != nil {
				return p, err
			}
		}
	}
	return p, nil
}

func Parse(r io.Reader) ([]Process, error) {
	processes := []Process{}
	scanner := bufio.NewScanner(r)
	processData := []string{}
	for scanner.Scan() {
		line := scanner.Text()

		// ignore empty lines
		if strings.TrimSpace(line) == "" {
			continue
		}

		// we collect all the data until a new process occurs
		// for the first process we have no data, therefore we skip that
		if strings.HasPrefix(line, "p") && len(processData) > 0 {
			process, err := parseProcessLines(processData)
			if err != nil {
				return nil, err
			}
			processes = append(processes, process)
			processData = []string{}
		}
		processData = append(processData, line)
	}

	// handle the last process because there is no additional 'p' that tiggers it
	if len(processData) > 0 {
		process, err := parseProcessLines(processData)
		if err != nil {
			return nil, err
		}
		processes = append(processes, process)
	}
	return processes, nil
}
