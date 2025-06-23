// SPDX-FileCopyrightText: Copyright 2025 SAP SE or an SAP affiliate company
//
// SPDX-License-Identifier: Apache-2.0

package errors

import (
	"errors"
)

var (
	ErrBadRequest       = errors.New("bad request")
	ErrProjectMismatch  = errors.New("project mismatch")
	ErrMissingIPAddress = errors.New("missing ip address")
	ErrMissingSubnets   = errors.New("network has no subnets")
	ErrNoSelfIP         = errors.New("no self ip found")
	ErrNoVCMPFound      = errors.New("no VCMP guest found")
	ErrQuotaExceeded    = errors.New("quota has been met")
	ErrNoPhysNetFound   = errors.New("no physical network found")
	ErrNoSubnetFound    = errors.New("no subnet(s) found")
	ErrNoIPsAvailable   = errors.New("no IPs left")
)
