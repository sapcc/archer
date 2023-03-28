# Archer - CCloud Endpoint Service

Archer is an API service that can privatly connect services from one private [OpenStack Network](https://docs.openstack.org/neutron/latest/admin/intro-os-networking.html) to another. Consumers can select a *service* from a service catalog and **inject** it to their network, which means making this *service* available via a private ip address.

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
