<!--
SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company

SPDX-License-Identifier: Apache-2.0
-->

# Changelog

All notable changes to archerctl will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Fixed

- API handlers that lock a service row with `FOR UPDATE` (service delete, service migrate, endpoint create) now bound the lock wait with a 5s `lock_timeout` instead of blocking until the 30s request deadline. When the lock is held by another operation (e.g. an agent mid-reconcile), the handler now returns HTTP 503 with a `Retry-After` header, and logs the blocking session — its PID, `application_name`, state, how long it has held the lock, and its running query — so the actual lock holder can be identified from the logs.
- Requests whose context is cancelled or times out (e.g. the client disconnects or the request deadline is hit while waiting on a `FOR UPDATE` row lock) no longer surface as HTTP 500 with an error-level stack trace. The recovery middleware now detects context-cancellation panics and returns HTTP 499 (Client Closed Request), logged at info level. Such panics are also filtered out in Sentry via `BeforeSend`, so client disconnects no longer create Sentry issues.

## [2.5.3] - 2026-07-10

### Added

- Audit: POST and PUT request bodies (JSON only, capped at 64 KiB) are now included in CADF audit events as a `request_body` attachment, so audit trails record what changed, not just that something changed.

### Changed

- Audit: 4xx responses (auth rejections, missing resources, validation failures) are no longer recorded as audit events. These represent requests that never took effect and were polluting the audit trail with non-state-changing noise. 2xx/3xx successes and 5xx server errors continue to be audited.
- Audit: `NewAuditController` now returns an error instead of panicking when the audit backend configuration is invalid, so the caller can decide how to handle audit-config problems at startup.

### Fixed

- archer-server: panic-induced HTTP 500 responses are now correctly labelled `code="500"` in the Prometheus `http_requests_total`, `http_request_duration_seconds`, and `http_response_size_bytes` metrics. Previously, the panic-recovery middleware ran outside the Prometheus instrumentation, so these requests were recorded with `code="0"` instead.
- Audit: fixed a nil-pointer panic in the audit response wrapper when the security principal was missing from the request context (e.g. requests rejected before the auth middleware attached a token). Failing to audit now logs an error instead of crashing the HTTP handler.
- Audit: `WriteHeader` is now idempotent — a second call (from an error renderer, or the stdlib's implicit call from `Write`) no longer produces a duplicate audit event.

### Removed

- archer-server: dropped the `github.com/dre1080/recovr` dependency in favor of a small in-tree panic recovery middleware. Panic responses are now a plain `500 Internal Server Error` with no panic details in the body (the previous recovr middleware leaked the panic message, file, and line number into the HTML/JSON response). Stack traces are still logged server-side.

## [2.5.2] - 2026-07-06

### Fixed

- F5 agent: `ProcessServices` now provisions a SelfIP on the service subnet again. The 2.5.1 fix removed the call under the assumption that only endpoints need SelfIPs, but the AS3 SnatPool addresses and pool members are pinned to the service's route domain and require the BIG-IP to have L3 presence on the service VLAN — otherwise ARP for SNAT IPs and health monitors against pool members fail. The nil-pointer hardening in `EnsureSelfIPs` from 2.5.1 is retained; the new call runs with `dryRun=false` so the Neutron port is materialized and the SelfIP is created on the device.

## [2.5.1] - 2026-07-01

### Fixed

- F5 agent: `ProcessServices` no longer panics with a nil-pointer dereference in `EnsureSelfIPs` when a subnet has no matching Neutron SelfIP ports yet. Services don't need per-device SelfIPs (only endpoints do, and `ProcessEndpoint` already ensures them), so the redundant call has been removed. `EnsureSelfIPs` now also skips (rather than dereferences) devices with no port when invoked in dry-run.

## [2.5.0] - 2026-06-22

### Changed

- Service: F5/tenant services now always allocate a dedicated Neutron SNAT port pool. The `snat_pool_size` field defaults to 1 and is no longer optional in storage. **Upgrade impact:** existing services that previously used per-device SelfIP addresses for SNAT will be reallocated dedicated SNAT ports on the next agent sync. In-flight connections through the old SNAT addresses will be reset.

### Fixed

- Server: immediate notification job no longer captures the HTTP request context, so the DB lookup and Campfire send no longer fail with "context canceled" once the handler returns
- Scheduler: leader election now recovers when its dedicated PostgreSQL connection dies (server restart, proxy cycling, idle timeout) — previously the elector logged `failed to deallocate cached statement(s): conn closed` on every tick and the instance could never become leader again until restart

### Added

- Service: `snat_pool_size` field on services to scale outbound SNAT capacity (f5 provider only, range 1-8, default 1). The controller allocates service-scoped Neutron ports synchronously on create/update.
- IPv6 address support for service `ip_addresses` and endpoint `ip_address` fields
- Network Injection agent: IPv6 support for proxy and HAProxy configuration
- F5 agent: IPv6 support for Proxy Protocol v2 iRule (dual-stack AF\_INET/AF\_INET6 detection)
- Network Injection agent: Proxy Protocol v2 support via HAProxy `send-proxy-v2`

## [2.4.1] - 2026-04-30

### Fixed

- Replace LICENSE symlink with regular file for compatibility with license detection tools

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

[Unreleased]: https://github.com/sapcc/archer/compare/v2.5.3...HEAD

[2.5.3]: https://github.com/sapcc/archer/compare/v2.5.2...v2.5.3

[2.5.2]: https://github.com/sapcc/archer/compare/v2.5.1...v2.5.2

[2.5.1]: https://github.com/sapcc/archer/compare/v2.5.0...v2.5.1

[2.5.0]: https://github.com/sapcc/archer/compare/v2.4.1...v2.5.0

[2.4.1]: https://github.com/sapcc/archer/compare/v2.4.0...v2.4.1

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
