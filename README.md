<!--
SPDX-FileCopyrightText: Copyright 2025 SAP SE or an SAP affiliate company

SPDX-License-Identifier: Apache-2.0
-->

# Archer - CCloud Endpoint Service [![Go Reference](https://pkg.go.dev/badge/github.com/sapcc/archer.svg)](https://pkg.go.dev/github.com/sapcc/archer) [![Swagger](https://img.shields.io/badge/Swagger-UI-brightgreen)](https://sapcc.github.io/archer/) [![Go Report Card](https://goreportcard.com/badge/github.com/sapcc/archer)](https://goreportcard.com/report/github.com/sapcc/archer) [![Checks](https://github.com/sapcc/archer/actions/workflows/checks.yaml/badge.svg)](https://github.com/sapcc/archer/actions/workflows/checks.yaml) [![License](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)

Archer is an API service that can privately connect services from one private [OpenStack Network](https://docs.openstack.org/neutron/latest/admin/intro-os-networking.html) to another. Consumers can select a *service* from a service catalog and **inject** it to their network, which means making this *service* available via a private ip address.

Archer implements an *OpenStack* like API and integrates with *OpenStack Keystone* and *OpenStack Neutron*.

## Architecture
There are two types of resources: **services** and **endpoints**

* **Services** are private or public services that are manually configured in *Archer*. They can be accessed by creating an endpoint.
* **Service endpoints**, or short **endpoints**, are IP endpoints in a local network used to transparently access services residing in different private networks.

### Features
* Multi-tenant capable via OpenStack Identity service
* OpenStack `policy.json` access policy support
* Prometheus Exporter
* Rate limiting
* CORS
* CADF compatible audit tracing
* Sentry support
* CLI Client `archerctl`

### Supported Backends
* F5 BigIP
* Network Injection agent (together with `openvswitch-agent` or `linuxbridge-agent`)

### Requirements
* PostgreSQL Database
* OpenStack Keystone
* OpenStack Neutron

## CLI Client
`archerctl` provides a OpenStack-like CLI client for interacting with the Archer API Service. It supports common OpenStack environment setting as set by the OpenStack RC File.

```sh
# archerctl --help
Usage:
  archerctl [OPTIONS] <command>

Application Options:
      --debug                                  Show verbose debug information
      --os-endpoint=                           The endpoint that will always be used [$OS_ENDPOINT]
      --os-auth-url=                           Authentication URL [$OS_AUTH_URL]
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

Help Options:
  -h, --help                                   Show this help message

Available commands:
  endpoint  Endpoints
  quota     Quotas
  rbac      RBACs
  service   Services
  version   Version
```


#### Example
```sh
# archerctl service list
+--------------------------------------+------+------+---------+----------+-----------+-------------------+
| ID                                   | NAME | PORT | ENABLED | PROVIDER | STATUS    | AVAILABILITY_ZONE |
+--------------------------------------+------+------+---------+----------+-----------+-------------------+
| 3c8ab870-a409-46f2-b19a-f5672e793705 | test | 80   | true    | tenant   | AVAILABLE |                   |
+--------------------------------------+------+------+---------+----------+-----------+-------------------+
```

## API
This section describes properties of the Archer API. It uses a ReSTful HTTP API.

#### Request format
The Archer API only accepts requests with the JSON data serialization format. The Content-Type header for POST requests is always expected to be `application/json`.

#### Response format
The Archer API always response with JSON data serialization format. The Content-Type header is always `Content-Type: application/json`.

#### Authentication and authorization
The **Archer API** uses the OpenStack Identity service as the default authentication service. When Keystone is enabled, users that submit requests to the OpenStack Networking service must provide an authentication token in `X-Auth-Token` request header.
You obtain the token by authenticating to the Keystone endpoint.

When Keystone is enabled, the `project_id` attribute is not required in create requests because the project ID is derived from the authentication token.

#### Pagination
To reduce load on the service, list operations will return a maximum number of items at a time. To navigate the collection, the parameters limit, marker and page_reverse can be set in the URI. For example:

```
?limit=100&marker=1234&page_reverse=False
```

The `marker` parameter is the ID of the last item in the previous list. The `limit` parameter sets the page size. The `page_reverse` parameter sets the page direction.
These parameters are optional.
If the client requests a limit beyond the maximum limit configured by the deployment, the server returns the maximum limit number of items.

For convenience, list responses contain atom **next** links and **previous** links. The last page in the list requested with `page_reverse=False` will not contain **next** link, and the last page in the list requested with `page_reverse=True` will not contain **previous** link.

To determine if pagination is supported, a user can check whether the `pagination` capability is available through the Archer API detail endpoint.

#### Sorting
You can use the `sort` parameter to sort the results of list operations.
The sort parameter contains a comma-separated list of sort keys, in order of the sort priority. Each sort key can be optionally prepended with a minus **-** character to reverse default sort direction (ascending).

For example:

```
?sort=key1,-key2,key3
```

**key1** is the first key (ascending order), **key2** is the second key (descending order) and **key3** is the third key in ascending order.


To determine if sorting is supported, a user can check whether the `sort` capability is available through the Archer API detail endpoint.

#### Filtering by tags
Most resources (e.g. service and endpoint) support adding tags to the resource attributes. Archer supports advanced filtering using these tags for list operations. The following tag filters are supported by the Archer API:

* `tags` - Return the list of entities that have this tag or tags.
* `tags-any` - Return the list of entities that have one or more of the given tags.
* `not-tags` - Return the list of entities that do not have one or more of the given tags.
* `not-tags-any` - Return the list of entities that do not have at least one of the given tags.

Each tag supports a maximum amount of 64 characters.

For example to get a list of resources having both, **red** and **blue** tags:

```
?tags=red,blue
```

To get a list of resourcing having either, **red** or **blue** tags:

```
?tags-any=red,blue
```

Tag filters can also be combined in the same request:

```
?tags=red,blue&tags-any=green,orange
```

#### Response Codes (Faults)

| Code | Description                                                                                    |
|------|------------------------------------------------------------------------------------------------|
| 400  | Validation Error                                                                               |
| 401  | Unauthorized                                                                                   |
| 403  | Policy does not allow current user to do this <br> The project is over quota for the request   |
| 404  | Not Found <br> Resource not found                                                              |
| 409  | Conflict                                                                                       |
| 429  | You have reached maximum request limit                                                         |
| 500  | Internal server error                                                                          |

## Support, Feedback, Contributing

This project is open to feature requests/suggestions, bug reports etc. via [GitHub issues](https://docs.github.com/en/issues/tracking-your-work-with-issues/using-issues/creating-an-issue). Contribution and feedback are encouraged and always welcome. For more information about how to contribute, the project structure, as well as additional contribution information, see our [Contribution Guidelines](https://github.com/SAP-cloud-infrastructure/.github/blob/main/CONTRIBUTING.md).

## Security / Disclosure

If you find any bug that may be a security problem, please follow our instructions [in our security policy](https://github.com/SAP-cloud-infrastructure/.github/blob/main/SECURITY.md) on how to report it. Please do not create GitHub issues for security-related doubts or problems.

## Code of Conduct

We as members, contributors, and leaders pledge to make participation in our community a harassment-free experience for everyone. By participating in this project, you agree to abide by its [Code of Conduct](https://github.com/SAP-cloud-infrastructure/.github/blob/main/CODE_OF_CONDUCT.md) at all times.

## Licensing

Copyright 2023-2025 SAP SE or an SAP affiliate company and archer contributors. Please see our [LICENSE](./LICENSE) for copyright and license information. Detailed information including third-party components and their licensing/copyright information is available [via the REUSE tool](https://api.reuse.software/info/github.com/sapcc/archer).
