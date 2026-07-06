<!--
SPDX-FileCopyrightText: Copyright 2025 SAP SE or an SAP affiliate company

SPDX-License-Identifier: Apache-2.0
-->

# Archer — SCI Endpoint Service

[![Go Reference](https://pkg.go.dev/badge/github.com/sapcc/archer/v2.svg)](https://pkg.go.dev/github.com/sapcc/archer/v2)
[![Swagger](https://img.shields.io/badge/Swagger-UI-brightgreen)](https://sapcc.github.io/archer/)
[![Checks](https://github.com/sapcc/archer/actions/workflows/checks.yaml/badge.svg)](https://github.com/sapcc/archer/actions/workflows/checks.yaml)
[![License](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)

Archer is an OpenStack-style API service that privately connects services across [OpenStack Networks](https://docs.openstack.org/neutron/latest/admin/intro-os-networking.html). Consumers pick a *service* from a catalog and **inject** it into their own network — the service becomes reachable through a private IP, without exposing either network to the other.

Archer integrates with *OpenStack Keystone* for authentication and *OpenStack Neutron* for network and port management.

## Concepts

Archer exposes two resource types:

- **Services** — private or public services registered in Archer. They are reached by creating an endpoint.
- **Endpoints** — IP endpoints in a local network that transparently forward to a service living in another private network.

## Features

- Multi-tenant via OpenStack Identity
- OpenStack `policy.json` access policies
- Prometheus exporter
- Rate limiting and CORS
- CADF-compatible audit trail
- Sentry error reporting
- OpenStack-style CLI client (`archerctl`)

## Supported Backends

- **F5 BigIP** — provisioned via the `archer-f5-agent`
- **Network Injection** — the `archer-ni-agent`, using HAProxy inside Linux network namespaces; works alongside `openvswitch-agent` or `linuxbridge-agent`

## Requirements

- PostgreSQL
- OpenStack Keystone
- OpenStack Neutron

## Components

| Binary | Description |
|--------|-------------|
| `archer-server` | REST API server |
| `archer-f5-agent` | Backend agent for F5 BigIP |
| `archer-ni-agent` | Network Injection agent (HAProxy in netns) |
| `archer-migrate` | Database schema migration tool |
| `archerctl` | CLI client |

Build all of them with `make build-all`. Run `make check` to lint, test and build.

## CLI Client

`archerctl` is an OpenStack-style CLI for the Archer API. It honours the standard OpenStack environment variables set by an OpenStack RC file.

```sh
# archerctl --help
Usage:
  archerctl [OPTIONS] <command>

Application Options:
      --debug                                  Show verbose debug information
      --os-endpoint=                           The endpoint that will always be used [$OS_ENDPOINT]
      --os-auth-url=                           Authentication URL [$OS_AUTH_URL]
      --os-token=                              Authentication token [$OS_TOKEN]
      --os-password=                           User's password to use with [$OS_PASSWORD]
      --os-username=                           User's username to use with [$OS_USERNAME]
      --os-project-domain-name=                Domain name containing project [$OS_PROJECT_DOMAIN_NAME]
      --os-project-name=                       Project name to scope to [$OS_PROJECT_NAME]
      --os-region-name=                        Authentication region name [$OS_REGION_NAME]
      --os-user-domain-name=                   User's domain name [$OS_USER_DOMAIN_NAME]
      --os-pw-cmd=                             Derive user's password from command [$OS_PW_CMD]

Output formatters:
  -f, --format=[table|csv|markdown|html|value] The output format, defaults to table (default: table)
  -c, --column=                                specify the column(s) to include, can be repeated to show multiple columns
      --sort-column=                           specify the column(s) to sort the data (columns specified first have a priority, non-existing columns are ignored), can be repeated
      --long                                   Show all columns in output
      --no-color                               Disable colorized output for tables [$NO_COLOR]

Help Options:
  -h, --help                                   Show this help message

Available commands:
  endpoint  Endpoints
  quota     Quotas
  rbac      RBACs
  service   Services
  version   Version
```

### Example

```sh
# archerctl service list
+--------------------------------------+------+------+---------+----------+-----------+-------------------+
| ID                                   | NAME | PORT | ENABLED | PROVIDER | STATUS    | AVAILABILITY_ZONE |
+--------------------------------------+------+------+---------+----------+-----------+-------------------+
| 3c8ab870-a409-46f2-b19a-f5672e793705 | test | 80   | true    | tenant   | AVAILABLE |                   |
+--------------------------------------+------+------+---------+----------+-----------+-------------------+
```

### Colorized Output

Table output is colorized by default on interactive terminals to make states easier to scan:

| Color | Values |
|-------|--------|
| Green | `true`, `enabled`, `active`, `available`, `success`, `accepted` |
| Red | `false`, `disabled`, `inactive`, `error`, `failed`, `rejected` |
| Yellow | `pending`, `processing`, `waiting` |
| Cyan | Headers, status values, names |
| Gray | IDs, project IDs, null values |

Colors are automatically disabled when:

- Output is piped or redirected (e.g. `archerctl service list | less`)
- `NO_COLOR` is set in the environment
- `TERM` is `dumb`
- A non-table format is used (`--format=csv`, `--format=markdown`, …)

To force-disable colors, use `--no-color` or `NO_COLOR=1`.

## API

Archer exposes a RESTful HTTP API. The full spec is browsable via [Swagger UI](https://sapcc.github.io/archer/).

### Request / Response Format

Requests and responses use JSON. `POST` requests must set `Content-Type: application/json`; responses always come back with `Content-Type: application/json`.

### Authentication

Archer uses OpenStack Keystone. Requests must include a Keystone token in the `X-Auth-Token` header. Because the project ID is derived from the token, `project_id` is not required on create requests.

### Pagination

List operations return a bounded number of items. Navigate the collection with URI parameters:

```
?limit=100&marker=1234&page_reverse=False
```

- `marker` — the ID of the last item from the previous page
- `limit` — page size (clamped to the deployment maximum)
- `page_reverse` — reverse pagination direction

Responses include atom `next` and `previous` links. The final forward page has no `next`; the final reverse page has no `previous`. Deployments advertise pagination support through the `pagination` capability on the API detail endpoint.

### Sorting

Use `sort` with a comma-separated list of keys, in priority order. Prefix a key with `-` to sort descending:

```
?sort=key1,-key2,key3
```

Sort support is advertised through the `sort` capability on the API detail endpoint.

### Filtering by Tags

Most resources (services, endpoints, …) accept tags. Archer supports four tag filters on list operations:

- `tags` — entities that have **all** the given tags
- `tags-any` — entities that have **any** of the given tags
- `not-tags` — entities that do **not** have all of the given tags
- `not-tags-any` — entities that do **not** have any of the given tags

Each tag is limited to 64 characters. Filters can be combined:

```
?tags=red,blue&tags-any=green,orange
```

### Response Codes

| Code | Meaning |
|------|---------|
| 400 | Validation error |
| 401 | Unauthorized |
| 403 | Policy denies the action, or the project is over quota |
| 404 | Resource not found |
| 409 | Conflict |
| 429 | Rate limit exceeded |
| 500 | Internal server error |

## Support, Feedback, Contributing

This project is open to feature requests, suggestions, and bug reports via [GitHub issues](https://docs.github.com/en/issues/tracking-your-work-with-issues/using-issues/creating-an-issue). Contributions and feedback are welcome — see our [Contribution Guidelines](https://github.com/SAP-cloud-infrastructure/.github/blob/main/CONTRIBUTING.md) for project structure and how to get involved.

## Security / Disclosure

If you find a bug that may be a security problem, please follow our [security policy](https://github.com/SAP-cloud-infrastructure/.github/blob/main/SECURITY.md). Do not open GitHub issues for security-related concerns.

## Code of Conduct

We pledge to make participation in our community a harassment-free experience for everyone. By participating in this project, you agree to abide by our [Code of Conduct](https://github.com/SAP-cloud-infrastructure/.github/blob/main/CODE_OF_CONDUCT.md).

## Licensing

Copyright 2023-2025 SAP SE or an SAP affiliate company and Archer contributors. See [LICENSE](./LICENSE) for copyright and license information. Detailed third-party component information is available via the [REUSE tool](https://api.reuse.software/info/github.com/sapcc/archer).
