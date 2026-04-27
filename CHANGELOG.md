<!--
SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company

SPDX-License-Identifier: Apache-2.0
-->

# Changelog

All notable changes to archerctl will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [2.4.0] - 2026-04-27

### Added

- `--cascade` flag to `service delete` command to delete all associated endpoints along with the service

### Fixed

- Server: use request context for database queries to prevent connection pool exhaustion when requests time out
- F5 agent: optimize `populateEndpointPorts` with O(n+m) map-based lookups and handle missing ports gracefully
- F5 agent: retain Neutron ports on rejection to enable re-acceptance without port recreation
- NI agent: `DisableInjection` no longer fetches port from Neutron, allowing cleanup even if port was manually deleted
- Server: log error when notification host lookup fails in accept/reject handler
- Agent: ensure endpoints are deleted before their services by scheduling pending endpoint deletions first

## [2.3.1] - 2026-04-22

### Fixed

- Server: accept rejected endpoints as PENDING\_UPDATE (not PENDING\_CREATE) since the Neutron port already exists
- Server: restrict accept/reject to only valid endpoint statuses to prevent agent lock contention

## [2.3.0] - 2026-04-20

### Added

- `--no-proxy-protocol` flag to `service create` and `service set` commands

### Fixed

- F5 agent: tolerate missing Neutron ports when endpoints are pending deletion/rejection
- Agent: fix DB notification thread reconnection - properly re-acquire connection when lost
- Agent: clarify log field names (`job_id`, `endpoint_ids`) for better debugging
- `service create` no longer defaults `require_approval`, `enabled`, and `proxy_protocol` when flags are not specified; the server defaults now apply

## [2.2.0] - 2026-04-16

### Added

- `connection_mirroring` field to endpoint resources for BIG-IP HA failover support
- `--connection-mirroring` flag to `endpoint create` and `endpoint set` commands
- `--no-connection-mirroring` flag to `endpoint set` command

## [2.1.0] - 2026-04-14

### Added

- `service create` and `service set` now support `--no-require-approval` flag to explicitly disable require-approval
- Commands now accept names in addition to IDs for services, endpoints, and networks (errors if multiple entities share the same name)

## [2.0.0] - 2026-04-03

### Added

- Add `service migrate` command to migrate a service to another agent
- Add `agent list` and `agent show` commands for viewing registered agents

## [1.9.0] - 2026-04-01

### Added

- Health status monitoring for services (ONLINE, DEGRADED, OFFLINE, UNCHECKED)
- Terminal colorization for CLI output

## [1.8.0] - 2026-03-31

### Added

- Support token-based authentication via `--os-token`/`OS_TOKEN`

## [1.7.0] - 2026-03-23

### Changed

- Internal config option fix for service visibility

## [1.6.0] - 2026-03-23

### Added

- Add IN\_USE column for service list

## [1.5.0] - 2026-03-23

### Added

- Improve display of portrange (e.g. \[0, 4000-4031])

## [1.4.1] - 2025-11-17

### Changed

- Re-Release with goreleaser

[Unreleased]: https://github.com/sapcc/archer/compare/v2.4.0...HEAD

[2.4.0]: https://github.com/sapcc/archer/compare/v2.3.1...v2.4.0

[2.3.1]: https://github.com/sapcc/archer/compare/v2.3.0...v2.3.1

[2.3.0]: https://github.com/sapcc/archer/compare/v2.2.0...v2.3.0

[2.2.0]: https://github.com/sapcc/archer/compare/v2.1.0...v2.2.0

[2.1.0]: https://github.com/sapcc/archer/compare/v2.0.0...v2.1.0

[2.0.0]: https://github.com/sapcc/archer/compare/v1.9.0...v2.0.0

[1.9.0]: https://github.com/sapcc/archer/compare/v1.8.0...v1.9.0

[1.8.0]: https://github.com/sapcc/archer/compare/v1.7.0...v1.8.0

[1.7.0]: https://github.com/sapcc/archer/compare/v1.6.0...v1.7.0

[1.6.0]: https://github.com/sapcc/archer/compare/v1.5.0...v1.6.0

[1.5.0]: https://github.com/sapcc/archer/compare/v1.4.1...v1.5.0

[1.4.1]: https://github.com/sapcc/archer/releases/tag/v1.4.1
