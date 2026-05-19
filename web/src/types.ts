// SPDX-FileCopyrightText: 2026 SAP SE
// SPDX-License-Identifier: Apache-2.0

// Archer API Types - derived from swagger.yaml

export type ServiceStatus =
  | "AVAILABLE"
  | "PENDING_CREATE"
  | "PENDING_UPDATE"
  | "PENDING_DELETE"
  | "UNAVAILABLE"
  | "ERROR_QUOTA";

export type EndpointStatus =
  | "AVAILABLE"
  | "PENDING_APPROVAL"
  | "PENDING_CREATE"
  | "PENDING_UPDATE"
  | "PENDING_REJECTED"
  | "PENDING_DELETE"
  | "REJECTED"
  | "FAILED";

export type HealthStatus = "ONLINE" | "DEGRADED" | "OFFLINE" | "UNCHECKED";
export type Visibility = "private" | "public";
export type Protocol = "HTTP" | "TCP";
export type Provider = "tenant" | "cp";

export interface Service {
  id: string;
  name: string | null;
  description: string | null;
  enabled: boolean;
  ports: number[];
  network_id: string;
  ip_addresses: string[];
  status: ServiceStatus;
  require_approval: boolean;
  visibility: Visibility;
  availability_zone: string | null;
  proxy_protocol: boolean;
  connection_mirroring: boolean;
  protocol: Protocol;
  provider: Provider | null;
  tags: string[] | null;
  health_status: HealthStatus | null;
  host: string | null;
  project_id: string;
  created_at: string;
  updated_at: string;
}

export interface ServiceCreate {
  name?: string;
  description?: string;
  ports: number[];
  network_id: string;
  ip_addresses: string[];
  enabled?: boolean;
  require_approval?: boolean;
  visibility?: Visibility;
  proxy_protocol?: boolean;
  connection_mirroring?: boolean;
  protocol?: Protocol;
  tags?: string[];
  provider?: Provider;
}

export interface ServiceUpdate {
  name?: string | null;
  description?: string | null;
  enabled?: boolean;
  ip_addresses?: string[];
  ports?: number[];
  require_approval?: boolean;
  visibility?: Visibility;
  proxy_protocol?: boolean;
  connection_mirroring?: boolean;
  protocol?: Protocol;
  tags?: string[];
}

export interface EndpointTarget {
  network?: string | null;
  subnet?: string | null;
  port?: string | null;
}

export interface Endpoint {
  id: string;
  service_id: string;
  name: string | null;
  description: string | null;
  target: EndpointTarget;
  ip_address: string;
  status: EndpointStatus;
  tags: string[] | null;
  project_id: string;
  created_at: string;
  updated_at: string;
}

export interface EndpointCreate {
  service_id: string;
  name?: string;
  description?: string;
  target: EndpointTarget;
  tags?: string[];
}

export interface EndpointUpdate {
  name?: string | null;
  description?: string | null;
  tags?: string[] | null;
}

export interface EndpointConsumer {
  id: string;
  status: EndpointStatus;
  project_id: string;
}

export interface RBACPolicy {
  id: string;
  service_id: string;
  target_type: "project";
  target: string;
  project_id: string;
  created_at: string;
  updated_at: string;
}

export interface RBACPolicyCreate {
  service_id: string;
  target_type?: "project";
  target: string;
}

export interface RBACPolicyUpdate {
  target: string;
}

export interface Agent {
  host: string;
  availability_zone: string;
  provider: Provider;
  enabled: boolean;
  physnet: string;
  created_at: string;
  updated_at: string;
  heartbeat_at: string;
  services: number;
}

export interface ListResponse<T> {
  items: T[];
  links?: { href: string; rel: string }[];
}

export interface Version {
  version: string;
  updated: string;
  links?: { href: string; rel: string }[];
}

// OpenStack Network (from Neutron API)
export interface Network {
  id: string;
  name: string;
  status: string;
  subnets: string[];
  project_id: string;
}

export interface NetworksResponse {
  networks: Network[];
}

// OpenStack Subnet (from Neutron API)
export interface Subnet {
  id: string;
  name: string;
  network_id: string;
  cidr: string;
  ip_version: number;
  project_id: string;
}

export interface SubnetsResponse {
  subnets: Subnet[];
}

// OpenStack Port (from Neutron API)
export interface Port {
  id: string;
  name: string;
  network_id: string;
  status: string;
  device_owner: string;
  device_id: string;
  fixed_ips: { subnet_id: string; ip_address: string }[];
  project_id: string;
}

export interface PortsResponse {
  ports: Port[];
}

// App props passed from host
export interface AppProps {
  endpoint?: string;
  projectID?: string;
  token?: string;
  canEdit?: boolean | string;
  mockAPI?: boolean;
  theme?: string;
  embedded?: boolean | string;
  getTokenFuncName?: string;
  archerctlUrl?: string;
  docsUrl?: string;
  neutronEndpoint?: string;
  cloudAdmin?: boolean | string;
}
