// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

//go:build debugtest
// +build debugtest

package eos

import (
	"fmt"
	"testing"

	"github.com/aristanetworks/goeapi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAristaConnection(t *testing.T) {
	// connect to our device
	node, err := goeapi.Connect("https", "192.168.178.154", "admin", "password1!", 443)
	if err != nil {
		t.Fatal(err)
	}

	// TODO: support enable password
	// node.EnableAuthentication("password1!")
	// TODO: read errors from node.err

	eos := Eos{node: node}

	config := eos.RunningConfig()
	assert.True(t, len(config) > 0)

	systemConfig := eos.SystemConfig()
	assert.Equal(t, 2, len(systemConfig))
	assert.Equal(t, "localhost", systemConfig["hostname"])

	ifaces := eos.IPInterfaces()
	assert.Equal(t, 2, len(ifaces))

	res, err := eos.Stp()
	require.NoError(t, err)
	fmt.Printf("%v", res)

	res2, err := eos.StpInterfaceDetails("0", "Ethernet1")
	require.NoError(t, err)
	fmt.Printf("%v", res2)
}
