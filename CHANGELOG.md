<!--
SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company

SPDX-License-Identifier: Apache-2.0
-->

# Changelog

All notable changes to archerctl will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

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

- Add IN_USE column for service list

## [1.5.0] - 2026-03-23

### Added

- Improve display of portrange (e.g. [0, 4000-4031])

## [1.4.1] - 2025-11-17

### Changed

- Re-Release with goreleaser

[Unreleased]: https://github.com/sapcc/archer/compare/v1.9.0...HEAD
[1.9.0]: https://github.com/sapcc/archer/compare/v1.8.0...v1.9.0
[1.8.0]: https://github.com/sapcc/archer/compare/v1.7.0...v1.8.0
[1.7.0]: https://github.com/sapcc/archer/compare/v1.6.0...v1.7.0
[1.6.0]: https://github.com/sapcc/archer/compare/v1.5.0...v1.6.0
[1.5.0]: https://github.com/sapcc/archer/compare/v1.4.1...v1.5.0
[1.4.1]: https://github.com/sapcc/archer/releases/tag/v1.4.1
