<!--
SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Archer contributors
SPDX-License-Identifier: Apache-2.0
-->

# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Archer is an OpenStack-compatible endpoint service that enables private connectivity between services across different OpenStack Networks. It implements a multi-tenant API service with two backends: F5 BigIP load balancers and a Network Injection agent using HAProxy with netlink namespaces.

**Core Concepts:**
- **Services**: Private or public services registered in Archer (manually configured)
- **Endpoints**: IP endpoints in local networks that provide transparent access to services in different networks
- **Providers**: Backend implementations (`f5` for F5 BigIP, `cp` for Network Injection/Control Plane agent)

## Build and Testing Commands

### Essential Commands
```bash
# Run all checks (linters, tests, builds) - USE THIS AFTER EVERY CHANGE
make check

# Build all binaries
make build-all

# Run tests for a specific package
go test -v ./internal/agent/ni/...

# Run a single test
go test -v ./internal/agent/ni -run TestAgent_EnableInjection

# Run tests with coverage
make build/cover.html

# Format code
make goimports

# Clean build artifacts
make clean
```

### Validation Workflow
**IMPORTANT**: Always run `make check` after making changes. It runs:
1. shellcheck (shell script validation)
2. golangci-lint (Go linters - includes errcheck, staticcheck, etc.)
3. go-licence-detector (dependency license scanning)
4. addlicense & reuse (license header checks)
5. Full test suite with coverage
6. All binary builds

## Architecture

### Binary Components

1. **archer-server** (`cmd/archer-server`): Main API server
   - REST API implementation using go-openapi/go-swagger
   - OpenStack Keystone integration for auth
   - Policy-based authorization (goslo.policy)
   - PostgreSQL for persistence

2. **archer-f5-agent** (`cmd/archer-f5-agent`): F5 BigIP backend agent
   - Manages F5 load balancer configurations
   - Handles endpoint provisioning on F5 devices
   - Supports vCMP (Virtual Clustered Multiprocessing)

3. **archer-ni-agent** (`cmd/archer-ni-agent`): Network Injection agent
   - Creates Linux network namespaces with netlink
   - Runs HAProxy instances in isolated namespaces
   - Integrates with OpenStack Neutron for port management

4. **archerctl** (`cmd/archerctl`): CLI client
   - OpenStack-style command-line interface
   - Uses OpenStack environment variables

5. **archer-migrate** (`cmd/archer-migrate`): Database migration tool

### Package Structure

```
internal/
├── agent/           # Shared agent code (common worker interface, DB notifications)
│   ├── f5/          # F5 BigIP agent implementation
│   │   ├── as3/     # AS3 (Application Services 3) API
│   │   ├── bigip/   # BigIP REST API client
│   │   └── f5os/    # F5 OS platform API
│   └── ni/          # Network Injection agent
│       ├── haproxy/ # HAProxy management
│       ├── netlink/ # Linux network namespace handling
│       ├── proxy/   # Unix socket proxy
│       └── models/  # NI-specific models
├── controller/      # API request handlers (services, endpoints, quotas, RBAC)
├── db/              # Database layer (pgx connection pool, migrations, helpers)
├── config/          # Configuration loading (INI files, environment variables)
├── neutron/         # OpenStack Neutron client wrapper
├── auth/            # Keystone authentication
├── policy/          # Policy engine integration
└── middlewares/     # HTTP middlewares (auth, audit, rate-limit, CORS)
```

### Agent Architecture

Both agents implement the `agent.Worker` interface:
```go
type Worker interface {
    ProcessServices(context.Context) error
    ProcessEndpoint(context.Context, strfmt.UUID) error
    GetPool() db.PgxIface
    GetScheduler() gocron.Scheduler
}
```

**Agent Workflow:**
1. Register agent with DB (host, availability zone, provider)
2. Start DB notification listener (LISTEN/NOTIFY pattern)
3. Start scheduled sync jobs (pending services/endpoints)
4. Process notifications:
   - Service changes → `ProcessServices()`
   - Endpoint changes → `ProcessEndpoint()`

### Network Injection Agent Details

**Critical netlink implementation notes** (internal/agent/ni/netlink/):
- Uses Linux network namespaces for isolation
- Thread locking is CRITICAL: namespace operations must lock OS thread (`runtime.LockOSThread()`)
- Helper methods `lockThread()` and `unlockThread()` manage thread locks and `isLocked` flag atomically
- All namespace handles must be properly closed to prevent leaks
- Veth pair validation: namespace can exist but veth pair can be broken - always validate both
- Cleanup on failure: use `cleanupFailedNamespace()` to clean up veth pairs and namespaces
- Never skip thread unlock on error paths

**HAProxy Management:**
- One HAProxy instance per network namespace
- Configuration generated dynamically
- Unix socket for stats/control

### Database Patterns

- Uses `pgx` v5 (not database/sql)
- Connection pooling via `pgxpool`
- Query builder: `Masterminds/squirrel` with PostgreSQL dollar placeholders
- Row scanning: `scany/v2/pgxscan`
- LISTEN/NOTIFY for real-time change propagation
- Transactions via `pgx.BeginFunc()`

Example:
```go
sql, args := db.Select("*").From("endpoint").Where("id = ?", id).MustSql()
err := pgxscan.Get(ctx, tx, &endpoint, sql, args...)
```

### Testing Conventions

- Table-driven tests for multiple scenarios
- Use `testify/assert` for assertions
- Test files: `*_test.go` in same package
- Fake implementations for testing:
  - `netlink.FakeNetlink` (no actual namespace creation)
  - `haproxy.FakeHaproxy` (no actual process spawning)
- HTTP mocking via `gophercloud/v2/testhelper`
- Test PostgreSQL database required for integration tests

### Error Handling

- Always wrap errors with context: `fmt.Errorf("context: %w", err)`
- Use `errcheck` linter - all errors must be handled
- Deferred cleanup: `defer func() { _ = res.Close() }()` (blank identifier for errcheck)
- Log errors with structured fields: `log.WithField("endpoint_id", id).Error(err)`

### Code Style

- `go-makefile-maker` manages the Makefile (don't edit directly)
- License headers: SPDX format (Apache-2.0)
- Imports: grouped (stdlib, external, internal) via `goimports`
- Linting: `golangci-lint` with strict rules
- Avoid over-engineering: simple solutions over abstractions

### OpenStack Integration

- Keystone: Authentication tokens in `X-Auth-Token` header
- Neutron: Port management, network lookups
- Project ID derived from token (not required in requests)
- Policy enforcement via `goslo.policy` (OpenStack policy.json format)

### Configuration

- INI files loaded via `go-flags`
- Environment variables override INI settings
- Example: `OS_AUTH_URL`, `OS_PASSWORD`, etc.
- Agent-specific config in `[agent]` section

### Common Pitfalls

1. **Netlink namespace management**: Always pair `Enable` with `Disable`, never leave threads locked
2. **Database context**: Use request context, not `context.Background()` in handlers
3. **Error returns**: Network Injection agent HAProxy errors must be returned (not swallowed)
4. **Thread safety**: Netlink operations are NOT thread-safe without proper locking
5. **Resource cleanup**: Use defer for cleanup, but check state before cleanup (idempotent)

### Development Workflow

1. Make code changes
2. Run `make check` (validates everything)
3. Review linter output and fix issues
4. **If changes affect `cmd/archerctl/`**: Update `CHANGELOG.md` in the `[Unreleased]` section
   - Follow [Keep a Changelog](https://keepachangelog.com/en/1.1.0/) format
   - Categorize under: Added, Changed, Deprecated, Removed, Fixed, or Security
   - Verify format: `release-info CHANGELOG.md <version>`
5. Commit with meaningful message
6. Run `make check` again before pushing

When working on netlink code:
- Test with both real and fake implementations
- Verify thread lock/unlock pairs
- Check for handle leaks (Close() calls)
- Validate error paths clean up resources
