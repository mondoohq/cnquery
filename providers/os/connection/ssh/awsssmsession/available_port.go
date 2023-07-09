package awsssmsession

import "net"

// GetAvailablePort get an open port that is ready to use from the kernel
func GetAvailablePort() (int, error) {
	// get a new address
	addr, err := net.ResolveTCPAddr("tcp", "localhost:0")
	if err != nil {
		return 0, err
	}

	// try to listen on port
	l, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return 0, err
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port, nil
}
