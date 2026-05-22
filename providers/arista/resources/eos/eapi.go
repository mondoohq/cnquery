// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package eos

// EapiServer describes one of the four eAPI listener variants
// (httpServer, httpsServer, localHttpServer, unixSocketServer).
// The Port field is unused for the Unix socket variant; everything else
// shares the same shape.
type EapiServer struct {
	Configured bool  `json:"configured"`
	Running    bool  `json:"running"`
	Port       int64 `json:"port"`
}

type showManagementApiHttpCommands struct {
	Enabled          bool       `json:"enabled"`
	HttpServer       EapiServer `json:"httpServer"`
	HttpsServer      EapiServer `json:"httpsServer"`
	LocalHttpServer  EapiServer `json:"localHttpServer"`
	UnixSocketServer EapiServer `json:"unixSocketServer"`
}

func (s *showManagementApiHttpCommands) GetCmd() string {
	return "show management api http-commands"
}

// EapiStatus returns the eAPI service configuration: whether eAPI is enabled
// and the configured/running state and port of each transport (HTTP, HTTPS,
// local HTTP, and Unix-domain socket).
func (eos *Eos) EapiStatus() (*showManagementApiHttpCommands, error) {
	shRsp := &showManagementApiHttpCommands{}

	handle, err := eos.node.GetHandle("json")
	if err != nil {
		return nil, err
	}
	if err := handle.AddCommand(shRsp); err != nil {
		return nil, err
	}
	if err := handle.Call(); err != nil {
		return nil, err
	}
	handle.Close()

	return shRsp, nil
}
