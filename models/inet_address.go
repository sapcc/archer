// SPDX-FileCopyrightText: Copyright 2025 SAP SE or an SAP affiliate company
//
// SPDX-License-Identifier: Apache-2.0

package models

import (
	"context"
	"fmt"
	"net/netip"

	"github.com/go-openapi/strfmt"
)

// InetAddress represents an IP address (IPv4 or IPv6) that can be scanned from PostgreSQL inet type.
type InetAddress string

// Validate implements runtime.Validatable for go-swagger validation.
func (a InetAddress) Validate(_ strfmt.Registry) error {
	if a == "" {
		return fmt.Errorf("ip address must not be empty")
	}
	if _, err := netip.ParseAddr(string(a)); err != nil {
		return fmt.Errorf("invalid IP address: %s", a)
	}
	return nil
}

// ContextValidate implements runtime.ContextValidatable for go-swagger.
func (a InetAddress) ContextValidate(_ context.Context, _ strfmt.Registry) error {
	return nil
}

// ScanNetipPrefix implements pgtype.NetipPrefixScanner for pgx inet scanning.
func (a *InetAddress) ScanNetipPrefix(v netip.Prefix) error {
	if !v.IsValid() {
		*a = ""
		return nil
	}
	*a = InetAddress(v.Addr().String())
	return nil
}

// NetipPrefixValue implements pgtype.NetipPrefixValuer for pgx inet encoding.
func (a InetAddress) NetipPrefixValue() (netip.Prefix, error) {
	if a == "" {
		return netip.Prefix{}, nil
	}
	addr, err := netip.ParseAddr(string(a))
	if err != nil {
		return netip.Prefix{}, fmt.Errorf("parsing inet address %q: %w", a, err)
	}
	return netip.PrefixFrom(addr, addr.BitLen()), nil
}
