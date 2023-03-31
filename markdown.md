


# 🏹 Archer
Archer is an API service that can privately connect services from one private [OpenStack Network](https://docs.openstack.org/neutron/latest/admin/intro-os-networking.html) to another. Consumers can select a *service* from a service catalog and **inject** it to their network, which means making this *service* available via a private ip address.

Archer implements an *OpenStack* like API and integrates with *OpenStack Keystone* and *OpenStack Neutron*.

### Architecture
There are two types of resources: **services** and **endpoints**

* **Services** are private or public services that are manually configured in *Archer*. They can be accessed by creating an endpoint.
* **Service endpoints**, or short **endpoints**, are IP endpoints in a local network used to transparently access services residing in different private networks.

### Features
* Multi-tenant capable via OpenStack Identity service
* OpenStack `policy.json` access policy support
* Prometheus Exporter
* Rate limiting

### Supported Backends
* F5 BigIP

### Requirements
* PostgreSQL Database

### API
The **Archer API** uses the OpenStack Identity service as the default authentication service. When Keystone is enabled, users that submit requests to the OpenStack Networking service must provide an authentication token in `X-Auth-Token` request header. 
You obtain the token by authenticating to the Keystone endpoint.

When Keystone is enabled, the `project_id` attribute is not required in create requests because the project ID is derived from the authentication token.

  
> [GitHub](https://github.com/sapcc/archer)

## Informations

### Version

1.3.0

### License

[Apache 2.0](https://www.apache.org/licenses/LICENSE-2.0.html)

### Contact

SAP SE / Converged Cloud  https://sap.com

## Tags

  ### <span id="tag-version"></span>Version

### Version API
Lists information of enabled Archer capabilities.


  ### <span id="tag-service"></span>Service

### Services
Services are for publishing TCP/UDP services using internal IP addresses in your private network.


  ### <span id="tag-endpoint"></span>Endpoint

### Endpoints
Endpoints are for accessing existing Services using internal IP addresses in your private network.


  ### <span id="tag-r-b-a-c"></span>RBAC

### RBAC Policies
RBAC Policies are used to provide service visibility to specific project or domains.


  ### <span id="tag-quota"></span>Quota

### Quota Operations
Administrative API for listing and setting quotas for services and endpoints.


## Content negotiation

### URI Schemes
  * http
  * https

### Consumes
  * application/json

### Produces
  * application/json

## Access control

### Security Schemes

#### X-Auth-Token (header: X-Auth-Token)

The **Archer API** uses the OpenStack Identity service as the default authentication service. When Keystone is enabled, users that submit requests to the OpenStack Networking service must provide an authentication token in `X-Auth-Token` request header. 
You obtain the token by authenticating to the Keystone endpoint.


> **Type**: apikey

### Security Requirements
  * X-Auth-Token

## All endpoints

###  endpoint

| Method  | URI     | Name   | Summary |
|---------|---------|--------|---------|
| DELETE | /endpoint/{endpoint_id} | [delete endpoint endpoint ID](#delete-endpoint-endpoint-id) | Remove an existing endpoint |
| GET | /endpoint | [get endpoint](#get-endpoint) | List existing service endpoints |
| GET | /endpoint/{endpoint_id} | [get endpoint endpoint ID](#get-endpoint-endpoint-id) | Show existing service endpoint |
| POST | /endpoint | [post endpoint](#post-endpoint) | Create endpoint for accessing a service |
  


###  quota

| Method  | URI     | Name   | Summary |
|---------|---------|--------|---------|
| DELETE | /quotas/{project_id} | [delete quotas project ID](#delete-quotas-project-id) | Reset all Quota of a project |
| GET | /quotas | [get quotas](#get-quotas) | List Quotas |
| GET | /quotas/defaults | [get quotas defaults](#get-quotas-defaults) | Show Quota Defaults |
| GET | /quotas/{project_id} | [get quotas project ID](#get-quotas-project-id) | Show Quota detail |
| PUT | /quotas/{project_id} | [put quotas project ID](#put-quotas-project-id) | Update Quota |
  


###  rbac

| Method  | URI     | Name   | Summary |
|---------|---------|--------|---------|
| DELETE | /rbac-policies/{rbac_policy_id} | [delete rbac policies rbac policy ID](#delete-rbac-policies-rbac-policy-id) | Delete RBAC policy |
| GET | /rbac-policies | [get rbac policies](#get-rbac-policies) | List RBAC policies |
| GET | /rbac-policies/{rbac_policy_id} | [get rbac policies rbac policy ID](#get-rbac-policies-rbac-policy-id) | Show details of an RBAC policy |
| POST | /rbac-policies | [post rbac policies](#post-rbac-policies) | Create RBAC policy |
| PUT | /rbac-policies/{rbac_policy_id} | [put rbac policies rbac policy ID](#put-rbac-policies-rbac-policy-id) | Update an existing RBAC policy |
  


###  service

| Method  | URI     | Name   | Summary |
|---------|---------|--------|---------|
| DELETE | /service/{service_id} | [delete service service ID](#delete-service-service-id) | Remove service from catalog |
| GET | /service | [get service](#get-service) | List services |
| GET | /service/{service_id} | [get service service ID](#get-service-service-id) | Show details of an service |
| GET | /service/{service_id}/endpoints | [get service service ID endpoints](#get-service-service-id-endpoints) | List service endpoints consumers |
| POST | /service | [post service](#post-service) | Add a new service to the catalog |
| PUT | /service/{service_id} | [put service service ID](#put-service-service-id) | Update an existing service |
| PUT | /service/{service_id}/accept_endpoints | [put service service ID accept endpoints](#put-service-service-id-accept-endpoints) | Accept endpoints |
| PUT | /service/{service_id}/reject_endpoints | [put service service ID reject endpoints](#put-service-service-id-reject-endpoints) | Reject endpoints |
  


###  version

| Method  | URI     | Name   | Summary |
|---------|---------|--------|---------|
| GET | / | [get](#get) | Shows details for Archer API |
  


## Paths

### <span id="delete-endpoint-endpoint-id"></span> Remove an existing endpoint (*DeleteEndpointEndpointID*)

```
DELETE /endpoint/{endpoint_id}
```

#### Parameters

| Name | Source | Type | Go type | Separator | Required | Default | Description |
|------|--------|------|---------|-----------| :------: |---------|-------------|
| endpoint_id | `path` | uuid (formatted string) | `strfmt.UUID` |  | ✓ |  | The UUID of the endpoint |

#### All responses
| Code | Status | Description | Has headers | Schema |
|------|--------|-------------|:-----------:|--------|
| [204](#delete-endpoint-endpoint-id-204) | No Content | Resource successfully deleted. |  | [schema](#delete-endpoint-endpoint-id-204-schema) |
| [404](#delete-endpoint-endpoint-id-404) | Not Found | Not Found |  | [schema](#delete-endpoint-endpoint-id-404-schema) |

#### Responses


##### <span id="delete-endpoint-endpoint-id-204"></span> 204 - Resource successfully deleted.
Status: No Content

###### <span id="delete-endpoint-endpoint-id-204-schema"></span> Schema

##### <span id="delete-endpoint-endpoint-id-404"></span> 404 - Not Found
Status: Not Found

###### <span id="delete-endpoint-endpoint-id-404-schema"></span> Schema

### <span id="delete-quotas-project-id"></span> Reset all Quota of a project (*DeleteQuotasProjectID*)

```
DELETE /quotas/{project_id}
```

#### Parameters

| Name | Source | Type | Go type | Separator | Required | Default | Description |
|------|--------|------|---------|-----------| :------: |---------|-------------|
| project_id | `path` | string | `string` |  | ✓ |  | The ID of the project to query. |

#### All responses
| Code | Status | Description | Has headers | Schema |
|------|--------|-------------|:-----------:|--------|
| [204](#delete-quotas-project-id-204) | No Content | Resource successfully reset |  | [schema](#delete-quotas-project-id-204-schema) |
| [404](#delete-quotas-project-id-404) | Not Found | Not Found |  | [schema](#delete-quotas-project-id-404-schema) |

#### Responses


##### <span id="delete-quotas-project-id-204"></span> 204 - Resource successfully reset
Status: No Content

###### <span id="delete-quotas-project-id-204-schema"></span> Schema

##### <span id="delete-quotas-project-id-404"></span> 404 - Not Found
Status: Not Found

###### <span id="delete-quotas-project-id-404-schema"></span> Schema

### <span id="delete-rbac-policies-rbac-policy-id"></span> Delete RBAC policy (*DeleteRbacPoliciesRbacPolicyID*)

```
DELETE /rbac-policies/{rbac_policy_id}
```

#### Parameters

| Name | Source | Type | Go type | Separator | Required | Default | Description |
|------|--------|------|---------|-----------| :------: |---------|-------------|
| rbac_policy_id | `path` | uuid (formatted string) | `strfmt.UUID` |  | ✓ |  | The UUID of the RBAC policy. |

#### All responses
| Code | Status | Description | Has headers | Schema |
|------|--------|-------------|:-----------:|--------|
| [204](#delete-rbac-policies-rbac-policy-id-204) | No Content | Resource successfully deleted. |  | [schema](#delete-rbac-policies-rbac-policy-id-204-schema) |
| [404](#delete-rbac-policies-rbac-policy-id-404) | Not Found | Not Found |  | [schema](#delete-rbac-policies-rbac-policy-id-404-schema) |

#### Responses


##### <span id="delete-rbac-policies-rbac-policy-id-204"></span> 204 - Resource successfully deleted.
Status: No Content

###### <span id="delete-rbac-policies-rbac-policy-id-204-schema"></span> Schema

##### <span id="delete-rbac-policies-rbac-policy-id-404"></span> 404 - Not Found
Status: Not Found

###### <span id="delete-rbac-policies-rbac-policy-id-404-schema"></span> Schema

### <span id="delete-service-service-id"></span> Remove service from catalog (*DeleteServiceServiceID*)

```
DELETE /service/{service_id}
```

Deletes this service. There **must** be no active associated endpoint for successfully deleting the service. 
Active endpoints can be rejected by the service owner via the `/service/{service_id}/reject_endpoints` API.


#### Parameters

| Name | Source | Type | Go type | Separator | Required | Default | Description |
|------|--------|------|---------|-----------| :------: |---------|-------------|
| service_id | `path` | uuid (formatted string) | `strfmt.UUID` |  | ✓ |  | The UUID of the service |

#### All responses
| Code | Status | Description | Has headers | Schema |
|------|--------|-------------|:-----------:|--------|
| [204](#delete-service-service-id-204) | No Content | Resource successfully deleted. |  | [schema](#delete-service-service-id-204-schema) |
| [404](#delete-service-service-id-404) | Not Found | Not Found |  | [schema](#delete-service-service-id-404-schema) |
| [409](#delete-service-service-id-409) | Conflict | In use. |  | [schema](#delete-service-service-id-409-schema) |

#### Responses


##### <span id="delete-service-service-id-204"></span> 204 - Resource successfully deleted.
Status: No Content

###### <span id="delete-service-service-id-204-schema"></span> Schema

##### <span id="delete-service-service-id-404"></span> 404 - Not Found
Status: Not Found

###### <span id="delete-service-service-id-404-schema"></span> Schema

##### <span id="delete-service-service-id-409"></span> 409 - In use.
Status: Conflict

###### <span id="delete-service-service-id-409-schema"></span> Schema

### <span id="get"></span> Shows details for Archer API (*Get*)

```
GET /
```

#### All responses
| Code | Status | Description | Has headers | Schema |
|------|--------|-------------|:-----------:|--------|
| [200](#get-200) | OK | Version |  | [schema](#get-200-schema) |

#### Responses


##### <span id="get-200"></span> 200 - Version
Status: OK

###### <span id="get-200-schema"></span> Schema
   
  

[Version](#version)

### <span id="get-endpoint"></span> List existing service endpoints (*GetEndpoint*)

```
GET /endpoint
```

#### All responses
| Code | Status | Description | Has headers | Schema |
|------|--------|-------------|:-----------:|--------|
| [200](#get-endpoint-200) | OK | An array of endpoints. |  | [schema](#get-endpoint-200-schema) |

#### Responses


##### <span id="get-endpoint-200"></span> 200 - An array of endpoints.
Status: OK

###### <span id="get-endpoint-200-schema"></span> Schema
   
  

[][Endpoint](#endpoint)

### <span id="get-endpoint-endpoint-id"></span> Show existing service endpoint (*GetEndpointEndpointID*)

```
GET /endpoint/{endpoint_id}
```

#### Parameters

| Name | Source | Type | Go type | Separator | Required | Default | Description |
|------|--------|------|---------|-----------| :------: |---------|-------------|
| endpoint_id | `path` | uuid (formatted string) | `strfmt.UUID` |  | ✓ |  | The UUID of the endpoint |

#### All responses
| Code | Status | Description | Has headers | Schema |
|------|--------|-------------|:-----------:|--------|
| [200](#get-endpoint-endpoint-id-200) | OK | An endpoint detail. |  | [schema](#get-endpoint-endpoint-id-200-schema) |
| [404](#get-endpoint-endpoint-id-404) | Not Found | Not Found |  | [schema](#get-endpoint-endpoint-id-404-schema) |

#### Responses


##### <span id="get-endpoint-endpoint-id-200"></span> 200 - An endpoint detail.
Status: OK

###### <span id="get-endpoint-endpoint-id-200-schema"></span> Schema
   
  

[Endpoint](#endpoint)

##### <span id="get-endpoint-endpoint-id-404"></span> 404 - Not Found
Status: Not Found

###### <span id="get-endpoint-endpoint-id-404-schema"></span> Schema

### <span id="get-quotas"></span> List Quotas (*GetQuotas*)

```
GET /quotas
```

#### Parameters

| Name | Source | Type | Go type | Separator | Required | Default | Description |
|------|--------|------|---------|-----------| :------: |---------|-------------|
| project_id | `query` | string | `string` |  |  |  | The ID of the project to query. |

#### All responses
| Code | Status | Description | Has headers | Schema |
|------|--------|-------------|:-----------:|--------|
| [200](#get-quotas-200) | OK | A JSON array of quotas |  | [schema](#get-quotas-200-schema) |
| [404](#get-quotas-404) | Not Found | Not Found |  | [schema](#get-quotas-404-schema) |

#### Responses


##### <span id="get-quotas-200"></span> 200 - A JSON array of quotas
Status: OK

###### <span id="get-quotas-200-schema"></span> Schema
   
  

[GetQuotasOKBody](#get-quotas-o-k-body)

##### <span id="get-quotas-404"></span> 404 - Not Found
Status: Not Found

###### <span id="get-quotas-404-schema"></span> Schema

###### Inlined models

**<span id="get-quotas-o-k-body"></span> GetQuotasOKBody**


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| quotas | [][GetQuotasOKBodyQuotasItems0](#get-quotas-o-k-body-quotas-items0)| `[]*GetQuotasOKBodyQuotasItems0` |  | |  |  |



**<span id="get-quotas-o-k-body-quotas-items0"></span> GetQuotasOKBodyQuotasItems0**


  


* composed type [Quota](#quota)
* composed type [QuotaUsage](#quota-usage)
* inlined member (*AO2*)



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| project_id | [Project](#project)| `models.Project` |  | |  |  |



### <span id="get-quotas-defaults"></span> Show Quota Defaults (*GetQuotasDefaults*)

```
GET /quotas/defaults
```

#### All responses
| Code | Status | Description | Has headers | Schema |
|------|--------|-------------|:-----------:|--------|
| [200](#get-quotas-defaults-200) | OK | Show the quota defaults configured for new projects. |  | [schema](#get-quotas-defaults-200-schema) |

#### Responses


##### <span id="get-quotas-defaults-200"></span> 200 - Show the quota defaults configured for new projects.
Status: OK

###### <span id="get-quotas-defaults-200-schema"></span> Schema
   
  

[GetQuotasDefaultsOKBody](#get-quotas-defaults-o-k-body)

###### Inlined models

**<span id="get-quotas-defaults-o-k-body"></span> GetQuotasDefaultsOKBody**


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| quota | [Quota](#quota)| `models.Quota` |  | |  |  |



### <span id="get-quotas-project-id"></span> Show Quota detail (*GetQuotasProjectID*)

```
GET /quotas/{project_id}
```

#### Parameters

| Name | Source | Type | Go type | Separator | Required | Default | Description |
|------|--------|------|---------|-----------| :------: |---------|-------------|
| project_id | `path` | string | `string` |  | ✓ |  | The ID of the project to query. |

#### All responses
| Code | Status | Description | Has headers | Schema |
|------|--------|-------------|:-----------:|--------|
| [200](#get-quotas-project-id-200) | OK | Shows the details of a specific monitor. |  | [schema](#get-quotas-project-id-200-schema) |
| [404](#get-quotas-project-id-404) | Not Found | Not Found |  | [schema](#get-quotas-project-id-404-schema) |

#### Responses


##### <span id="get-quotas-project-id-200"></span> 200 - Shows the details of a specific monitor.
Status: OK

###### <span id="get-quotas-project-id-200-schema"></span> Schema
   
  

[GetQuotasProjectIDOKBody](#get-quotas-project-id-o-k-body)

##### <span id="get-quotas-project-id-404"></span> 404 - Not Found
Status: Not Found

###### <span id="get-quotas-project-id-404-schema"></span> Schema

###### Inlined models

**<span id="get-quotas-project-id-o-k-body"></span> GetQuotasProjectIDOKBody**


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| quota | [](#)| `` |  | |  |  |



### <span id="get-rbac-policies"></span> List RBAC policies (*GetRbacPolicies*)

```
GET /rbac-policies
```

#### All responses
| Code | Status | Description | Has headers | Schema |
|------|--------|-------------|:-----------:|--------|
| [200](#get-rbac-policies-200) | OK | A JSON array of rbac policies |  | [schema](#get-rbac-policies-200-schema) |
| [default](#get-rbac-policies-default) | | Unexpected Error |  | [schema](#get-rbac-policies-default-schema) |

#### Responses


##### <span id="get-rbac-policies-200"></span> 200 - A JSON array of rbac policies
Status: OK

###### <span id="get-rbac-policies-200-schema"></span> Schema
   
  

[][RBACPolicy](#r-b-a-c-policy)

##### <span id="get-rbac-policies-default"></span> Default Response
Unexpected Error

###### <span id="get-rbac-policies-default-schema"></span> Schema
empty schema

### <span id="get-rbac-policies-rbac-policy-id"></span> Show details of an RBAC policy (*GetRbacPoliciesRbacPolicyID*)

```
GET /rbac-policies/{rbac_policy_id}
```

#### Parameters

| Name | Source | Type | Go type | Separator | Required | Default | Description |
|------|--------|------|---------|-----------| :------: |---------|-------------|
| rbac_policy_id | `path` | uuid (formatted string) | `strfmt.UUID` |  | ✓ |  | The UUID of the RBAC policy. |

#### All responses
| Code | Status | Description | Has headers | Schema |
|------|--------|-------------|:-----------:|--------|
| [200](#get-rbac-policies-rbac-policy-id-200) | OK | RBAC Policy |  | [schema](#get-rbac-policies-rbac-policy-id-200-schema) |
| [404](#get-rbac-policies-rbac-policy-id-404) | Not Found | Not Found |  | [schema](#get-rbac-policies-rbac-policy-id-404-schema) |

#### Responses


##### <span id="get-rbac-policies-rbac-policy-id-200"></span> 200 - RBAC Policy
Status: OK

###### <span id="get-rbac-policies-rbac-policy-id-200-schema"></span> Schema
   
  

[RBACPolicy](#r-b-a-c-policy)

##### <span id="get-rbac-policies-rbac-policy-id-404"></span> 404 - Not Found
Status: Not Found

###### <span id="get-rbac-policies-rbac-policy-id-404-schema"></span> Schema

### <span id="get-service"></span> List services (*GetService*)

```
GET /service
```

#### All responses
| Code | Status | Description | Has headers | Schema |
|------|--------|-------------|:-----------:|--------|
| [200](#get-service-200) | OK | An array of services. |  | [schema](#get-service-200-schema) |

#### Responses


##### <span id="get-service-200"></span> 200 - An array of services.
Status: OK

###### <span id="get-service-200-schema"></span> Schema
   
  

[][Service](#service)

### <span id="get-service-service-id"></span> Show details of an service (*GetServiceServiceID*)

```
GET /service/{service_id}
```

#### Parameters

| Name | Source | Type | Go type | Separator | Required | Default | Description |
|------|--------|------|---------|-----------| :------: |---------|-------------|
| service_id | `path` | uuid (formatted string) | `strfmt.UUID` |  | ✓ |  | The UUID of the service |

#### All responses
| Code | Status | Description | Has headers | Schema |
|------|--------|-------------|:-----------:|--------|
| [200](#get-service-service-id-200) | OK | Service |  | [schema](#get-service-service-id-200-schema) |
| [400](#get-service-service-id-400) | Bad Request | Validation Error |  | [schema](#get-service-service-id-400-schema) |
| [404](#get-service-service-id-404) | Not Found | Not Found |  | [schema](#get-service-service-id-404-schema) |

#### Responses


##### <span id="get-service-service-id-200"></span> 200 - Service
Status: OK

###### <span id="get-service-service-id-200-schema"></span> Schema
   
  

[Service](#service)

##### <span id="get-service-service-id-400"></span> 400 - Validation Error
Status: Bad Request

###### <span id="get-service-service-id-400-schema"></span> Schema

##### <span id="get-service-service-id-404"></span> 404 - Not Found
Status: Not Found

###### <span id="get-service-service-id-404-schema"></span> Schema

### <span id="get-service-service-id-endpoints"></span> List service endpoints consumers (*GetServiceServiceIDEndpoints*)

```
GET /service/{service_id}/endpoints
```

Provides a list of service consumers (endpoints).

This list can be used to accept or reject requests, or disable active endpoints. 
Rejected endpoints will be cleaned up after a specific time.


#### Parameters

| Name | Source | Type | Go type | Separator | Required | Default | Description |
|------|--------|------|---------|-----------| :------: |---------|-------------|
| service_id | `path` | uuid (formatted string) | `strfmt.UUID` |  | ✓ |  | The UUID of the service |

#### All responses
| Code | Status | Description | Has headers | Schema |
|------|--------|-------------|:-----------:|--------|
| [200](#get-service-service-id-endpoints-200) | OK | An array of service endpoint consumers. |  | [schema](#get-service-service-id-endpoints-200-schema) |
| [404](#get-service-service-id-endpoints-404) | Not Found | Not Found |  | [schema](#get-service-service-id-endpoints-404-schema) |

#### Responses


##### <span id="get-service-service-id-endpoints-200"></span> 200 - An array of service endpoint consumers.
Status: OK

###### <span id="get-service-service-id-endpoints-200-schema"></span> Schema
   
  

[][EndpointConsumer](#endpoint-consumer)

##### <span id="get-service-service-id-endpoints-404"></span> 404 - Not Found
Status: Not Found

###### <span id="get-service-service-id-endpoints-404-schema"></span> Schema

### <span id="post-endpoint"></span> Create endpoint for accessing a service (*PostEndpoint*)

```
POST /endpoint
```

#### Parameters

| Name | Source | Type | Go type | Separator | Required | Default | Description |
|------|--------|------|---------|-----------| :------: |---------|-------------|
| body | `body` | [Endpoint](#endpoint) | `models.Endpoint` | | ✓ | | Service and target network to inject. Only one of `target_network`, `target_subnet` or `target_port` must be specified. |

#### All responses
| Code | Status | Description | Has headers | Schema |
|------|--------|-------------|:-----------:|--------|
| [200](#post-endpoint-200) | OK | Endpoint |  | [schema](#post-endpoint-200-schema) |
| [400](#post-endpoint-400) | Bad Request | Validation Error |  | [schema](#post-endpoint-400-schema) |

#### Responses


##### <span id="post-endpoint-200"></span> 200 - Endpoint
Status: OK

###### <span id="post-endpoint-200-schema"></span> Schema
   
  

[Endpoint](#endpoint)

##### <span id="post-endpoint-400"></span> 400 - Validation Error
Status: Bad Request

###### <span id="post-endpoint-400-schema"></span> Schema

### <span id="post-rbac-policies"></span> Create RBAC policy (*PostRbacPolicies*)

```
POST /rbac-policies
```

#### Parameters

| Name | Source | Type | Go type | Separator | Required | Default | Description |
|------|--------|------|---------|-----------| :------: |---------|-------------|
| body | `body` | [RBACPolicy](#r-b-a-c-policy) | `models.RBACPolicy` | | ✓ | | RBAC Policy |

#### All responses
| Code | Status | Description | Has headers | Schema |
|------|--------|-------------|:-----------:|--------|
| [200](#post-rbac-policies-200) | OK | RBAC policy |  | [schema](#post-rbac-policies-200-schema) |
| [400](#post-rbac-policies-400) | Bad Request | Validation Error |  | [schema](#post-rbac-policies-400-schema) |
| [409](#post-rbac-policies-409) | Conflict | Exists |  | [schema](#post-rbac-policies-409-schema) |

#### Responses


##### <span id="post-rbac-policies-200"></span> 200 - RBAC policy
Status: OK

###### <span id="post-rbac-policies-200-schema"></span> Schema
   
  

[RBACPolicy](#r-b-a-c-policy)

##### <span id="post-rbac-policies-400"></span> 400 - Validation Error
Status: Bad Request

###### <span id="post-rbac-policies-400-schema"></span> Schema

##### <span id="post-rbac-policies-409"></span> 409 - Exists
Status: Conflict

###### <span id="post-rbac-policies-409-schema"></span> Schema

### <span id="post-service"></span> Add a new service to the catalog (*PostService*)

```
POST /service
```

#### Parameters

| Name | Source | Type | Go type | Separator | Required | Default | Description |
|------|--------|------|---------|-----------| :------: |---------|-------------|
| body | `body` | [Service](#service) | `models.Service` | | ✓ | | Service object that needs to be added to the catalog |

#### All responses
| Code | Status | Description | Has headers | Schema |
|------|--------|-------------|:-----------:|--------|
| [200](#post-service-200) | OK | Service |  | [schema](#post-service-200-schema) |
| [400](#post-service-400) | Bad Request | Validation Error |  | [schema](#post-service-400-schema) |

#### Responses


##### <span id="post-service-200"></span> 200 - Service
Status: OK

###### <span id="post-service-200-schema"></span> Schema
   
  

[Service](#service)

##### <span id="post-service-400"></span> 400 - Validation Error
Status: Bad Request

###### <span id="post-service-400-schema"></span> Schema

### <span id="put-quotas-project-id"></span> Update Quota (*PutQuotasProjectID*)

```
PUT /quotas/{project_id}
```

#### Parameters

| Name | Source | Type | Go type | Separator | Required | Default | Description |
|------|--------|------|---------|-----------| :------: |---------|-------------|
| project_id | `path` | string | `string` |  | ✓ |  | The ID of the project to query. |
| quota | `body` | [PutQuotasProjectIDBody](#put-quotas-project-id-body) | `PutQuotasProjectIDBody` | | ✓ | |  |

#### All responses
| Code | Status | Description | Has headers | Schema |
|------|--------|-------------|:-----------:|--------|
| [202](#put-quotas-project-id-202) | Accepted | Updated quota for a project. |  | [schema](#put-quotas-project-id-202-schema) |
| [400](#put-quotas-project-id-400) | Bad Request | Validation Error |  | [schema](#put-quotas-project-id-400-schema) |
| [404](#put-quotas-project-id-404) | Not Found | Not found |  | [schema](#put-quotas-project-id-404-schema) |

#### Responses


##### <span id="put-quotas-project-id-202"></span> 202 - Updated quota for a project.
Status: Accepted

###### <span id="put-quotas-project-id-202-schema"></span> Schema
   
  

[PutQuotasProjectIDAcceptedBody](#put-quotas-project-id-accepted-body)

##### <span id="put-quotas-project-id-400"></span> 400 - Validation Error
Status: Bad Request

###### <span id="put-quotas-project-id-400-schema"></span> Schema

##### <span id="put-quotas-project-id-404"></span> 404 - Not found
Status: Not Found

###### <span id="put-quotas-project-id-404-schema"></span> Schema

###### Inlined models

**<span id="put-quotas-project-id-accepted-body"></span> PutQuotasProjectIDAcceptedBody**


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| quota | [Quota](#quota)| `models.Quota` |  | |  |  |



**<span id="put-quotas-project-id-body"></span> PutQuotasProjectIDBody**


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| quota | [Quota](#quota)| `models.Quota` | ✓ | |  |  |



### <span id="put-rbac-policies-rbac-policy-id"></span> Update an existing RBAC policy (*PutRbacPoliciesRbacPolicyID*)

```
PUT /rbac-policies/{rbac_policy_id}
```

#### Parameters

| Name | Source | Type | Go type | Separator | Required | Default | Description |
|------|--------|------|---------|-----------| :------: |---------|-------------|
| rbac_policy_id | `path` | uuid (formatted string) | `strfmt.UUID` |  | ✓ |  | The UUID of the RBAC policy. |
| body | `body` | [RBACPolicyCommon](#r-b-a-c-policy-common) | `models.RBACPolicyCommon` | | ✓ | | RBAC policy resource that needs to be updated |

#### All responses
| Code | Status | Description | Has headers | Schema |
|------|--------|-------------|:-----------:|--------|
| [200](#put-rbac-policies-rbac-policy-id-200) | OK | RBAC Policy |  | [schema](#put-rbac-policies-rbac-policy-id-200-schema) |
| [400](#put-rbac-policies-rbac-policy-id-400) | Bad Request | Validation Error |  | [schema](#put-rbac-policies-rbac-policy-id-400-schema) |
| [404](#put-rbac-policies-rbac-policy-id-404) | Not Found | Not Found |  | [schema](#put-rbac-policies-rbac-policy-id-404-schema) |

#### Responses


##### <span id="put-rbac-policies-rbac-policy-id-200"></span> 200 - RBAC Policy
Status: OK

###### <span id="put-rbac-policies-rbac-policy-id-200-schema"></span> Schema
   
  

[RBACPolicyCommon](#r-b-a-c-policy-common)

##### <span id="put-rbac-policies-rbac-policy-id-400"></span> 400 - Validation Error
Status: Bad Request

###### <span id="put-rbac-policies-rbac-policy-id-400-schema"></span> Schema

##### <span id="put-rbac-policies-rbac-policy-id-404"></span> 404 - Not Found
Status: Not Found

###### <span id="put-rbac-policies-rbac-policy-id-404-schema"></span> Schema

### <span id="put-service-service-id"></span> Update an existing service (*PutServiceServiceID*)

```
PUT /service/{service_id}
```

#### Parameters

| Name | Source | Type | Go type | Separator | Required | Default | Description |
|------|--------|------|---------|-----------| :------: |---------|-------------|
| service_id | `path` | uuid (formatted string) | `strfmt.UUID` |  | ✓ |  | The UUID of the service |
| body | `body` | [Service](#service) | `models.Service` | | ✓ | | Service object that needs to be updated |

#### All responses
| Code | Status | Description | Has headers | Schema |
|------|--------|-------------|:-----------:|--------|
| [200](#put-service-service-id-200) | OK | Service |  | [schema](#put-service-service-id-200-schema) |
| [400](#put-service-service-id-400) | Bad Request | Validation Error |  | [schema](#put-service-service-id-400-schema) |
| [404](#put-service-service-id-404) | Not Found | Not Found |  | [schema](#put-service-service-id-404-schema) |

#### Responses


##### <span id="put-service-service-id-200"></span> 200 - Service
Status: OK

###### <span id="put-service-service-id-200-schema"></span> Schema
   
  

[Service](#service)

##### <span id="put-service-service-id-400"></span> 400 - Validation Error
Status: Bad Request

###### <span id="put-service-service-id-400-schema"></span> Schema

##### <span id="put-service-service-id-404"></span> 404 - Not Found
Status: Not Found

###### <span id="put-service-service-id-404-schema"></span> Schema

### <span id="put-service-service-id-accept-endpoints"></span> Accept endpoints (*PutServiceServiceIDAcceptEndpoints*)

```
PUT /service/{service_id}/accept_endpoints
```

Specify a list of endpoint consumers (`endpoint_ids` and/or `project_ids`) whose endpoints should be accepted.
* Existing active endpoints will be untouched.
* Rejected endpoints will be accepted.
* Pending endpoints will be accepted.


#### Parameters

| Name | Source | Type | Go type | Separator | Required | Default | Description |
|------|--------|------|---------|-----------| :------: |---------|-------------|
| service_id | `path` | uuid (formatted string) | `strfmt.UUID` |  | ✓ |  | The UUID of the service |
| body | `body` | [EndpointConsumerList](#endpoint-consumer-list) | `models.EndpointConsumerList` | | ✓ | | Service object that needs to be updated |

#### All responses
| Code | Status | Description | Has headers | Schema |
|------|--------|-------------|:-----------:|--------|
| [200](#put-service-service-id-accept-endpoints-200) | OK | Ok |  | [schema](#put-service-service-id-accept-endpoints-200-schema) |
| [400](#put-service-service-id-accept-endpoints-400) | Bad Request | Validation Error |  | [schema](#put-service-service-id-accept-endpoints-400-schema) |
| [404](#put-service-service-id-accept-endpoints-404) | Not Found | Not Found |  | [schema](#put-service-service-id-accept-endpoints-404-schema) |

#### Responses


##### <span id="put-service-service-id-accept-endpoints-200"></span> 200 - Ok
Status: OK

###### <span id="put-service-service-id-accept-endpoints-200-schema"></span> Schema
   
  

[][EndpointConsumer](#endpoint-consumer)

##### <span id="put-service-service-id-accept-endpoints-400"></span> 400 - Validation Error
Status: Bad Request

###### <span id="put-service-service-id-accept-endpoints-400-schema"></span> Schema

##### <span id="put-service-service-id-accept-endpoints-404"></span> 404 - Not Found
Status: Not Found

###### <span id="put-service-service-id-accept-endpoints-404-schema"></span> Schema

### <span id="put-service-service-id-reject-endpoints"></span> Reject endpoints (*PutServiceServiceIDRejectEndpoints*)

```
PUT /service/{service_id}/reject_endpoints
```

Specify a list of consumers (`endpoint_ids` and/or `project_ids`) whose endpoints should be rejected.
* Existing active endpoints will be rejected.
* Rejected endpoints will be untouched.
* Pending endpoints will be rejected.


#### Parameters

| Name | Source | Type | Go type | Separator | Required | Default | Description |
|------|--------|------|---------|-----------| :------: |---------|-------------|
| service_id | `path` | uuid (formatted string) | `strfmt.UUID` |  | ✓ |  | The UUID of the service |
| body | `body` | [EndpointConsumerList](#endpoint-consumer-list) | `models.EndpointConsumerList` | | ✓ | | Service object that needs to be updated |

#### All responses
| Code | Status | Description | Has headers | Schema |
|------|--------|-------------|:-----------:|--------|
| [200](#put-service-service-id-reject-endpoints-200) | OK | Ok |  | [schema](#put-service-service-id-reject-endpoints-200-schema) |
| [400](#put-service-service-id-reject-endpoints-400) | Bad Request | Validation Error |  | [schema](#put-service-service-id-reject-endpoints-400-schema) |
| [404](#put-service-service-id-reject-endpoints-404) | Not Found | Not Found |  | [schema](#put-service-service-id-reject-endpoints-404-schema) |

#### Responses


##### <span id="put-service-service-id-reject-endpoints-200"></span> 200 - Ok
Status: OK

###### <span id="put-service-service-id-reject-endpoints-200-schema"></span> Schema
   
  

[][EndpointConsumer](#endpoint-consumer)

##### <span id="put-service-service-id-reject-endpoints-400"></span> 400 - Validation Error
Status: Bad Request

###### <span id="put-service-service-id-reject-endpoints-400-schema"></span> Schema

##### <span id="put-service-service-id-reject-endpoints-404"></span> 404 - Not Found
Status: Not Found

###### <span id="put-service-service-id-reject-endpoints-404-schema"></span> Schema

## Models

### <span id="endpoint"></span> Endpoint


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| id | uuid (formatted string)| `strfmt.UUID` |  | | The ID of the resource. |  |
| project_id | [Project](#project)| `Project` |  | |  |  |
| proxy_protocol | boolean| `bool` |  | | Proxy protocol enabled for this endpoint. |  |
| service_id | uuid (formatted string)| `strfmt.UUID` |  | | The ID of the service. |  |
| service_name | string| `string` |  | | The name of the service. | `Example Service` |
| status | [EndpointStatus](#endpoint-status)| `EndpointStatus` |  | |  |  |
| target_network | uuid (formatted string)| `strfmt.UUID` |  | | Endpoint network target. One of `target_network`, `target_subnet` or `target_port` must be specified. |  |
| target_port | uuid (formatted string)| `strfmt.UUID` |  | | Endpoint port target. One of `target_network`, `target_subnet` or `target_port` must be specified. | `b2accf1a-1c99-4b54-9eeb-22be53f177f5` |
| target_subnet | uuid (formatted string)| `strfmt.UUID` |  | | Endpoint subnet target. One of `target_network`, `target_subnet` or `target_port` must be specified. |  |



### <span id="endpoint-consumer"></span> EndpointConsumer


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| id | uuid (formatted string)| `strfmt.UUID` |  | | The ID of the resource. |  |
| status | [EndpointStatus](#endpoint-status)| `EndpointStatus` |  | |  |  |



### <span id="endpoint-consumer-list"></span> EndpointConsumerList


> list of consumer ids.
  





**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| endpoint_ids | []uuid (formatted string)| `[]strfmt.UUID` |  | |  |  |
| project_ids | []uuid (formatted string)| `[]strfmt.UUID` |  | |  |  |



### <span id="endpoint-status"></span> EndpointStatus


> Status of the endpoint

### Status can be one of
| Status             | Description                           |
| ------------------ | ------------------------------------- |
| AVAILABLE          | Endpoint is active for consumption    |
| PENDING_APPROVAL   | Endpoint is waiting for approval      |
| PENDING_CREATE     | Endpoint is being set up              |
| PENDING_DELETE     | Endpoint is being deleted             |
| REJECTED           | Endpoint was rejected                 |
| FAILED             | Endpoint setup failed                 |

  



| Name | Type | Go type | Default | Description | Example |
|------|------|---------| ------- |-------------|---------|
| EndpointStatus | string| string | | Status of the endpoint

### Status can be one of
| Status             | Description                           |
| ------------------ | ------------------------------------- |
| AVAILABLE          | Endpoint is active for consumption    |
| PENDING_APPROVAL   | Endpoint is waiting for approval      |
| PENDING_CREATE     | Endpoint is being set up              |
| PENDING_DELETE     | Endpoint is being deleted             |
| REJECTED           | Endpoint was rejected                 |
| FAILED             | Endpoint setup failed                 | |  |



### <span id="project"></span> Project


> The ID of the project owning this resource.
  



| Name | Type | Go type | Default | Description | Example |
|------|------|---------| ------- |-------------|---------|
| Project | string| string | | The ID of the project owning this resource. | `fa84c217f361441986a220edf9b1e337` |



### <span id="quota"></span> Quota


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| endpoint | integer| `int64` |  | | The configured endpoint quota limit. A setting of null means it is using the deployment default quota. A setting of -1 means unlimited. | `5` |
| service | integer| `int64` |  | | The configured service quota limit. A setting of null means it is using the deployment default quota. A setting of -1 means unlimited. | `5` |



### <span id="quota-usage"></span> QuotaUsage


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| in_use_endpoint | integer| `int64` |  | | The current quota usage of endpoints. | `5` |
| in_use_service | integer| `int64` |  | | The current quota usage of services. | `5` |



### <span id="r-b-a-c-policy"></span> RBACPolicy


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| id | uuid (formatted string)| `strfmt.UUID` |  | | The ID of the resource. |  |
| project_id | [Project](#project)| `Project` |  | |  |  |
| service_id | uuid (formatted string)| `strfmt.UUID` | ✓ | | The ID of the service resource. |  |
| target | string| `string` |  | | The ID of the project to which the RBAC policy will be enforced. | `666da95112694b37b3efb0913de3f499` |
| target_type | string| `string` |  | |  |  |



### <span id="r-b-a-c-policy-common"></span> RBACPolicyCommon


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| id | uuid (formatted string)| `strfmt.UUID` |  | | The ID of the resource. |  |
| project_id | [Project](#project)| `Project` |  | |  |  |
| target | string| `string` | ✓ | | The ID of the project to which the RBAC policy will be enforced. | `666da95112694b37b3efb0913de3f499` |
| target_type | string| `string` |  | |  |  |



### <span id="service"></span> Service


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| availability_zone | string| `string` |  | | Availability zone of this service. | `AZ-A` |
| description | string| `string` |  | | Description of the service. | `An example of an Service.` |
| enabled | boolean| `bool` |  | | Enable/disable this service. Existing endpoints are not touched by this. |  |
| id | uuid (formatted string)| `strfmt.UUID` |  | | The ID of the resource. |  |
| ip_address | ipv4 (formatted string)| `strfmt.IPv4` |  | | IP Address of the providing service. | `1.2.3.4` |
| name | string| `string` |  | | Name of the service. | `ExampleService` |
| network_id | uuid (formatted string)| `strfmt.UUID` |  | | Network ID of the network that provides this service. |  |
| ports | []integer| `[]int64` |  | | Ports exposed by the service. | `[80,443]` |
| project_id | [Project](#project)| `Project` |  | |  |  |
| require_approval | boolean| `bool` |  | `true`| Require explicit project approval for the service owner. |  |
| status | string| `string` |  | | Status of the service.

### Status can be one of
| Status           | Description                            |
| ---------------- | -------------------------------------- |
| AVAILABLE        | Service is ready for consumption.      |
| PENDING_CREATE   | Service is being set up                |
| PENDING_UPDATE   | Service is being updated               |
| PENDING_DELETE   | Service is being deleted               |
| UNAVAILABLE      | Service is unavailable (e.g. disabled) | |  |
| visibility | string| `string` |  | `"private"`| Set global visibility of the service. For `private` visibility, RBAC policies can extend the visibility to specific projects. |  |



### <span id="version"></span> Version


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| capabilities | []string| `[]string` |  | |  | `["pagination","sort"]` |
| links | [][VersionLinksItems0](#version-links-items0)| `[]*VersionLinksItems0` |  | |  |  |
| updated | string| `string` |  | | Last update of the running version | `2018-09-30T00:00:00Z` |
| version | string| `string` |  | | Version of Archer | `1.3.0` |



#### Inlined models

**<span id="version-links-items0"></span> VersionLinksItems0**


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| href | string| `string` |  | |  | `https://example.com` |
| rel | string| `string` |  | |  | `self` |
| type | string| `string` |  | |  | `application/json` |


