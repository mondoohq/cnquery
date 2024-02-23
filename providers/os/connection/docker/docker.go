// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package docker

// The docker connection package provides a connection to a docker engine and handles:
//
// - docker containers
// - docker images
// - docker snapshots
//
// Each of these types of connections is implemented as a separate connection type, since the data format is different.
// All of these connections are based on the tar connection, which is a generic connection type that can handle tar
// files. All docker connections are implemented as a wrapper around the tar connection and prepare the
// data in the correct format for the tar connection.
