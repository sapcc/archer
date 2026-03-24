// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company
//
// SPDX-License-Identifier: Apache-2.0

//go:build linux

package netlink

import (
	"testing"

	"github.com/sapcc/go-bits/osext"
	"github.com/stretchr/testify/suite"
)

// NetlinkSuite is a test suite for testing real Linux netlink operations.
// This suite runs inside a privileged container where netlink syscalls work.
type NetlinkSuite struct {
	suite.Suite
}

func TestNetlinkSuite(t *testing.T) {
	// This test suite should only run inside the privileged container
	if !osext.GetenvBool("NETLINK_TEST_PRIVILEGED") {
		t.Skip("Skipping NetlinkSuite - must run inside privileged container (NETLINK_TEST_PRIVILEGED=true)")
	}
	suite.Run(t, new(NetlinkSuite))
}
