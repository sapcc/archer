// SPDX-FileCopyrightText: 2026 SAP SE
// SPDX-License-Identifier: Apache-2.0

import { http, HttpResponse } from "msw";
import type { Service, Endpoint, EndpointConsumer, RBACPolicy, Network, Subnet, Port, Agent } from "../types";

// Project IDs for different tenants (32-char hex, no dashes - OpenStack format)
const OWN_PROJECT = "e9141fb24eee4b3e9f25ae69cda31132";
const FINANCE_PROJECT = "d0e1f2a3b4c54d6e8f8a9b0c1d2e3f4a";
const ANALYTICS_PROJECT = "e1f2a3b4c5d64e7f8a9b0c1d2e3f4a5b";
const DEVOPS_PROJECT = "f2a3b4c5d6e74f8a9b0c1d2e3f4a5b6c";

// Network IDs (UUIDv4)
const NET_PRODUCTION = "11110001-aaaa-4bbb-8ccc-ddddeeee0001";
const NET_STAGING = "22220002-bbbb-4ccc-8ddd-eeeeffff0002";
const NET_DEVELOPMENT = "33330003-cccc-4ddd-8eee-ffffaaaa0003";
const NET_SHARED = "44440004-dddd-4eee-8fff-aaaabbbb0004";
const NET_ANALYTICS = "55550005-eeee-4fff-8aaa-bbbbcccc0005";

// Service IDs (UUIDv4)
const SVC_POSTGRES = "a1b2c3d4-e5f6-4890-abcd-ef1234567890";
const SVC_REDIS = "b2c3d4e5-f6a7-4901-bcde-f12345678901";
const SVC_KAFKA = "c3d4e5f6-a7b8-4012-cdef-123456789012";
const SVC_ELASTICSEARCH = "d4e5f6a7-b8c9-4123-def0-234567890123";
const SVC_GRAFANA = "e5f6a7b8-c9d0-4234-ef01-345678901234";
const SVC_CP_APIGW = "f6a7b8c9-d0e1-4345-f012-456789012345";
const SVC_CP_IDENTITY = "a7b8c9d0-e1f2-4456-0123-567890123456";
const SVC_CP_LOGGING = "b8c9d0e1-f2a3-4567-1234-678901234567";
const SVC_FINANCE_DB = "c9d0e1f2-a3b4-4678-2345-789012345678";
const SVC_FINANCE_API = "d0e1f2a3-b4c5-4789-3456-890123456789";
const SVC_SPARK = "e1f2a3b4-c5d6-4890-4567-901234567890";
const SVC_AIRFLOW = "f2a3b4c5-d6e7-4901-5678-012345678901";
const SVC_ML = "a3b4c5d6-e7f8-4012-6789-123456789012";
const SVC_JENKINS = "b4c5d6e7-f8a9-4123-7890-234567890123";
const SVC_REGISTRY = "c5d6e7f8-a9b0-4234-8901-345678901234";
const SVC_VAULT = "d6e7f8a9-b0c1-4345-9012-456789012345";
const SVC_EDGE = "e7f8a9b0-c1d2-4456-0123-567890123456";

// Endpoint IDs (UUIDv4)
const EP_OWN_POSTGRES_1 = "11111111-1111-4111-8111-111111111111";
const EP_OWN_POSTGRES_2 = "22222222-2222-4222-8222-222222222222";
const EP_OWN_POSTGRES_3 = "33333333-3333-4333-8333-333333333333";
const EP_OWN_REDIS_1 = "44444444-4444-4444-8444-444444444444";
const EP_OWN_REDIS_2 = "55555555-5555-4555-8555-555555555555";
const EP_OWN_KAFKA_1 = "66666666-6666-4666-8666-666666666666";
const EP_OWN_APIGW_1 = "77777777-7777-4777-8777-777777777777";
const EP_OWN_IDENTITY_1 = "88888888-8888-4888-8888-888888888888";
const EP_OWN_LOGGING_1 = "99999999-9999-4999-8999-999999999999";
const EP_OWN_FINANCE_1 = "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa";
const EP_OWN_SPARK_1 = "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb";
const EP_OWN_VAULT_1 = "cccccccc-cccc-4ccc-8ccc-cccccccccccc";
const EP_OWN_REGISTRY_1 = "dddddddd-dddd-4ddd-8ddd-dddddddddddd";
const EP_FINANCE_POSTGRES_1 = "eeeeeeee-eeee-4eee-8eee-eeeeeeeeeeee";
const EP_ANALYTICS_KAFKA_1 = "ffffffff-ffff-4fff-8fff-ffffffffffff";
const EP_DEVOPS_ES_1 = "00000001-0001-4001-8001-000000000001";
const EP_DEVOPS_GRAFANA_1 = "00000002-0002-4002-8002-000000000002";

// Subnet IDs (UUIDv4)
const SUBNET_PRODUCTION = "aaa00001-0001-4001-8001-000000000001";
const SUBNET_STAGING = "aaa00002-0002-4002-8002-000000000002";
const SUBNET_DEVELOPMENT = "aaa00003-0003-4003-8003-000000000003";
const SUBNET_SHARED = "aaa00004-0004-4004-8004-000000000004";
const SUBNET_ANALYTICS = "aaa00005-0005-4005-8005-000000000005";
const SUBNET_BACKUP = "aaa00006-0006-4006-8006-000000000006";

// Additional Network IDs (UUIDv4)
const NET_BACKUP = "66660006-ffff-4aaa-8bbb-ccccdddd0006";
const NET_DMZ = "77770007-aaaa-4bbb-8ccc-ddddeeee0007";
const NET_INTERNAL = "88880008-bbbb-4ccc-8ddd-eeeeffff0008";

// Port IDs (UUIDv4)
const PORT_PROD_WEB = "bbb00001-0001-4001-8001-000000000001";
const PORT_PROD_DB = "bbb00002-0002-4002-8002-000000000002";
const PORT_STAGING_APP = "bbb00003-0003-4003-8003-000000000003";
const PORT_DEV_TEST = "bbb00004-0004-4004-8004-000000000004";
const PORT_SHARED_LB = "bbb00005-0005-4005-8005-000000000005";
const PORT_RESERVED = "bbb00006-0006-4006-8006-000000000006";

const services: Service[] = [
  // === OWN PROJECT SERVICES ===
  {
    id: SVC_POSTGRES,
    name: "PostgreSQL Primary",
    description: "Primary PostgreSQL database cluster",
    enabled: true,
    ports: [5432],
    network_id: NET_PRODUCTION,
    ip_addresses: ["10.0.1.10"],
    status: "AVAILABLE",
    require_approval: true,
    visibility: "private",
    availability_zone: "eu-de-1a",
    proxy_protocol: true,
    connection_mirroring: false,
    protocol: "TCP",
    provider: "tenant",
    tags: ["database", "production"],
    health_status: "ONLINE",
    host: "db-primary.example.com",
    project_id: OWN_PROJECT,
    created_at: "2024-01-15T10:30:00Z",
    updated_at: "2024-01-20T14:45:00Z",
  },
  {
    id: SVC_REDIS,
    name: "Redis Cache",
    description: "Distributed Redis cache cluster",
    enabled: true,
    ports: [6379],
    network_id: NET_PRODUCTION,
    ip_addresses: ["10.0.1.20"],
    status: "AVAILABLE",
    require_approval: false,
    visibility: "private",
    availability_zone: "eu-de-1a",
    proxy_protocol: false,
    connection_mirroring: false,
    protocol: "TCP",
    provider: "tenant",
    tags: ["cache", "production"],
    health_status: "ONLINE",
    host: "redis.example.com",
    project_id: OWN_PROJECT,
    created_at: "2024-02-01T08:00:00Z",
    updated_at: "2024-02-10T12:30:00Z",
  },
  {
    id: SVC_KAFKA,
    name: "Kafka Broker",
    description: "Message streaming platform",
    enabled: true,
    ports: [9092, 9093],
    network_id: NET_PRODUCTION,
    ip_addresses: ["10.0.1.30"],
    status: "AVAILABLE",
    require_approval: true,
    visibility: "private",
    availability_zone: "eu-de-1b",
    proxy_protocol: false,
    connection_mirroring: false,
    protocol: "TCP",
    provider: "tenant",
    tags: ["messaging", "production"],
    health_status: "ONLINE",
    host: "kafka.example.com",
    project_id: OWN_PROJECT,
    created_at: "2024-03-01T09:00:00Z",
    updated_at: "2024-03-15T11:00:00Z",
  },
  {
    id: SVC_ELASTICSEARCH,
    name: "Elasticsearch",
    description: "Search and analytics engine",
    enabled: true,
    ports: [9200, 9300],
    network_id: NET_STAGING,
    ip_addresses: ["10.0.2.10"],
    status: "PENDING_UPDATE",
    require_approval: false,
    visibility: "private",
    availability_zone: "eu-de-1a",
    proxy_protocol: false,
    connection_mirroring: false,
    protocol: "HTTP",
    provider: "tenant",
    tags: ["search", "staging"],
    health_status: "DEGRADED",
    host: "es.example.com",
    project_id: OWN_PROJECT,
    created_at: "2024-04-01T10:00:00Z",
    updated_at: "2024-04-10T12:00:00Z",
  },
  {
    id: SVC_GRAFANA,
    name: "Grafana Monitoring",
    description: "Observability dashboards",
    enabled: true,
    ports: [3000],
    network_id: NET_SHARED,
    ip_addresses: ["10.0.3.10"],
    status: "AVAILABLE",
    require_approval: false,
    visibility: "public",
    availability_zone: "eu-de-1c",
    proxy_protocol: false,
    connection_mirroring: false,
    protocol: "HTTP",
    provider: "tenant",
    tags: ["monitoring", "observability"],
    health_status: "ONLINE",
    host: "grafana.example.com",
    project_id: OWN_PROJECT,
    created_at: "2024-05-01T08:00:00Z",
    updated_at: "2024-05-15T10:00:00Z",
  },

  // === MANAGED (CONTROL PLANE) SERVICES ===
  {
    id: SVC_CP_APIGW,
    name: "API Gateway",
    description: "Central API gateway managed by control plane",
    enabled: true,
    ports: [443, 8443],
    network_id: NET_SHARED,
    ip_addresses: ["10.100.0.10"],
    status: "AVAILABLE",
    require_approval: false,
    visibility: "public",
    availability_zone: "eu-de-1a",
    proxy_protocol: false,
    connection_mirroring: false,
    protocol: "HTTP",
    provider: "cp",
    tags: ["api", "managed"],
    health_status: "ONLINE",
    host: "api-gw.cp.example.com",
    project_id: OWN_PROJECT,
    created_at: "2024-01-01T00:00:00Z",
    updated_at: "2024-06-01T00:00:00Z",
  },
  {
    id: SVC_CP_IDENTITY,
    name: "Identity Service",
    description: "Centralized authentication and authorization",
    enabled: true,
    ports: [443],
    network_id: NET_SHARED,
    ip_addresses: ["10.100.0.20"],
    status: "AVAILABLE",
    require_approval: false,
    visibility: "public",
    availability_zone: "eu-de-1a",
    proxy_protocol: false,
    connection_mirroring: false,
    protocol: "HTTP",
    provider: "cp",
    tags: ["auth", "managed"],
    health_status: "ONLINE",
    host: "identity.cp.example.com",
    project_id: OWN_PROJECT,
    created_at: "2024-01-01T00:00:00Z",
    updated_at: "2024-06-01T00:00:00Z",
  },
  {
    id: SVC_CP_LOGGING,
    name: "Logging Service",
    description: "Centralized log aggregation",
    enabled: true,
    ports: [5044, 8080],
    network_id: NET_SHARED,
    ip_addresses: ["10.100.0.30"],
    status: "PENDING_UPDATE",
    require_approval: false,
    visibility: "private",
    availability_zone: "eu-de-1b",
    proxy_protocol: false,
    connection_mirroring: false,
    protocol: "TCP",
    provider: "cp",
    tags: ["logging", "managed"],
    health_status: "DEGRADED",
    host: "logging.cp.example.com",
    project_id: OWN_PROJECT,
    created_at: "2024-01-01T00:00:00Z",
    updated_at: "2024-06-10T00:00:00Z",
  },

  // === EXTERNAL PROJECT: FINANCE ===
  {
    id: SVC_FINANCE_DB,
    name: "Finance DB",
    description: "Financial data warehouse",
    enabled: true,
    ports: [5432],
    network_id: NET_ANALYTICS,
    ip_addresses: ["10.200.1.10"],
    status: "AVAILABLE",
    require_approval: true,
    visibility: "private",
    availability_zone: "eu-de-1a",
    proxy_protocol: true,
    connection_mirroring: false,
    protocol: "TCP",
    provider: "tenant",
    tags: ["finance", "database"],
    health_status: "ONLINE",
    host: "finance-db.example.com",
    project_id: FINANCE_PROJECT,
    created_at: "2024-02-01T00:00:00Z",
    updated_at: "2024-06-01T00:00:00Z",
  },
  {
    id: SVC_FINANCE_API,
    name: "Finance API",
    description: "Financial reporting API",
    enabled: true,
    ports: [443],
    network_id: NET_ANALYTICS,
    ip_addresses: ["10.200.1.20"],
    status: "AVAILABLE",
    require_approval: true,
    visibility: "private",
    availability_zone: "eu-de-1b",
    proxy_protocol: false,
    connection_mirroring: false,
    protocol: "HTTP",
    provider: "tenant",
    tags: ["finance", "api"],
    health_status: "ONLINE",
    host: "finance-api.example.com",
    project_id: FINANCE_PROJECT,
    created_at: "2024-02-15T00:00:00Z",
    updated_at: "2024-06-01T00:00:00Z",
  },

  // === EXTERNAL PROJECT: ANALYTICS ===
  {
    id: SVC_SPARK,
    name: "Spark Cluster",
    description: "Apache Spark processing cluster",
    enabled: true,
    ports: [7077, 8080, 4040],
    network_id: NET_ANALYTICS,
    ip_addresses: ["10.200.2.10", "10.200.2.11", "10.200.2.12"],
    status: "AVAILABLE",
    require_approval: false,
    visibility: "private",
    availability_zone: "eu-de-1a",
    proxy_protocol: false,
    connection_mirroring: false,
    protocol: "TCP",
    provider: "tenant",
    tags: ["analytics", "spark"],
    health_status: "ONLINE",
    host: "spark.analytics.example.com",
    project_id: ANALYTICS_PROJECT,
    created_at: "2024-03-01T00:00:00Z",
    updated_at: "2024-06-01T00:00:00Z",
  },
  {
    id: SVC_AIRFLOW,
    name: "Airflow",
    description: "Workflow orchestration",
    enabled: true,
    ports: [8080],
    network_id: NET_ANALYTICS,
    ip_addresses: ["10.200.2.20"],
    status: "PENDING_CREATE",
    require_approval: false,
    visibility: "private",
    availability_zone: "eu-de-1b",
    proxy_protocol: false,
    connection_mirroring: false,
    protocol: "HTTP",
    provider: "tenant",
    tags: ["analytics", "workflow"],
    health_status: "UNCHECKED",
    host: "airflow.analytics.example.com",
    project_id: ANALYTICS_PROJECT,
    created_at: "2024-06-01T00:00:00Z",
    updated_at: "2024-06-01T00:00:00Z",
  },
  {
    id: SVC_ML,
    name: "ML Platform",
    description: "Machine learning model serving",
    enabled: true,
    ports: [8501, 8500],
    network_id: NET_ANALYTICS,
    ip_addresses: ["10.200.2.30"],
    status: "AVAILABLE",
    require_approval: true,
    visibility: "private",
    availability_zone: "eu-de-1c",
    proxy_protocol: false,
    connection_mirroring: false,
    protocol: "HTTP",
    provider: "tenant",
    tags: ["analytics", "ml"],
    health_status: "ONLINE",
    host: "ml.analytics.example.com",
    project_id: ANALYTICS_PROJECT,
    created_at: "2024-04-01T00:00:00Z",
    updated_at: "2024-06-01T00:00:00Z",
  },

  // === EXTERNAL PROJECT: DEVOPS ===
  {
    id: SVC_JENKINS,
    name: "Jenkins CI",
    description: "Continuous integration server",
    enabled: true,
    ports: [8080, 50000],
    network_id: NET_DEVELOPMENT,
    ip_addresses: ["10.200.3.10"],
    status: "AVAILABLE",
    require_approval: false,
    visibility: "private",
    availability_zone: "eu-de-1a",
    proxy_protocol: false,
    connection_mirroring: false,
    protocol: "HTTP",
    provider: "tenant",
    tags: ["devops", "ci"],
    health_status: "ONLINE",
    host: "jenkins.devops.example.com",
    project_id: DEVOPS_PROJECT,
    created_at: "2024-01-01T00:00:00Z",
    updated_at: "2024-06-01T00:00:00Z",
  },
  {
    id: SVC_REGISTRY,
    name: "Container Registry",
    description: "Docker container registry",
    enabled: true,
    ports: [443, 5000],
    network_id: NET_DEVELOPMENT,
    ip_addresses: ["10.200.3.20"],
    status: "AVAILABLE",
    require_approval: false,
    visibility: "public",
    availability_zone: "eu-de-1b",
    proxy_protocol: false,
    connection_mirroring: false,
    protocol: "HTTP",
    provider: "tenant",
    tags: ["devops", "registry"],
    health_status: "ONLINE",
    host: "registry.devops.example.com",
    project_id: DEVOPS_PROJECT,
    created_at: "2024-01-15T00:00:00Z",
    updated_at: "2024-06-01T00:00:00Z",
  },
  {
    id: SVC_VAULT,
    name: "Vault Secrets",
    description: "HashiCorp Vault for secrets management",
    enabled: true,
    ports: [8200],
    network_id: NET_SHARED,
    ip_addresses: ["10.200.3.30"],
    status: "AVAILABLE",
    require_approval: true,
    visibility: "private",
    availability_zone: "eu-de-1a",
    proxy_protocol: false,
    connection_mirroring: false,
    protocol: "HTTP",
    provider: "tenant",
    tags: ["devops", "secrets"],
    health_status: "ONLINE",
    host: "vault.devops.example.com",
    project_id: DEVOPS_PROJECT,
    created_at: "2024-02-01T00:00:00Z",
    updated_at: "2024-06-01T00:00:00Z",
  },

  // === EDGE CASE SERVICE (keep for testing) ===
  {
    id: SVC_EDGE,
    name: "🚀 Über-Spëcial «Sèrvice» with <HTML> & \"quotes\" + 'apostrophes' — ™®© ½⅓ μs → ∞",
    description: `A very long description that tests edge cases: <script>alert("xss")</script> & special chars like äöüß 中文 日本語 한국어 العربية עברית emoji: 🎉🔥💯🤖

Line breaks and\ttabs and "nested 'quotes'" plus $variables \${interpolation} and \`backticks\`.

Mathematical symbols: ∑∏∫∂∇ ≤≥≠≈ ±×÷ √∛∜ ∈∉⊂⊃ ∩∪ ∀∃

Currency & units: €£¥₹₽ 100°C 45° 1024KB/s

Regex-like patterns: /^[a-z]+$/gi .*? (foo|bar) [^abc]

Path-like strings: /usr/local/bin/../lib C:\\Windows\\System32 file://localhost

URL fragments: https://example.com/path?query=value&foo=bar#anchor

Shell injection attempts: ; rm -rf / && echo "pwned" | sudo dd if=/dev/zero`,
    enabled: true,
    ports: [
      1, 22, 80, 443, 1024, 1433, 1521, 3306, 3389, 5432, 5900, 6379, 8080, 8443, 9000, 9090, 9200, 9300, 11211, 27017,
      50000, 65535,
    ],
    network_id: "99999999-9999-4999-8999-999999999999",
    ip_addresses: ["10.255.255.1", "10.255.255.2", "10.255.255.3", "192.168.100.100", "172.16.0.1"],
    status: "UNAVAILABLE",
    require_approval: true,
    visibility: "private",
    availability_zone: "eu-de-1c",
    proxy_protocol: true,
    connection_mirroring: true,
    protocol: "TCP",
    provider: "tenant",
    tags: ["edge-case", "test<tag>", "über", "🏷️emoji", "tag with spaces", "a".repeat(50)],
    health_status: "OFFLINE",
    host: "host-003.example.com",
    project_id: OWN_PROJECT,
    created_at: "2020-01-01T00:00:00Z",
    updated_at: "2026-12-31T23:59:59Z",
  },
];

const endpoints: Endpoint[] = [
  // === OWN PROJECT ENDPOINTS ===
  // Endpoint to own PostgreSQL
  {
    id: EP_OWN_POSTGRES_1,
    service_id: SVC_POSTGRES,
    name: "App DB Connection",
    description: "Main application database connection",
    target: { network: NET_PRODUCTION },
    ip_address: "192.168.1.100",
    status: "AVAILABLE",
    tags: ["prod", "database"],
    project_id: OWN_PROJECT,
    created_at: "2024-01-16T11:00:00Z",
    updated_at: "2024-01-16T11:00:00Z",
  },
  // Second endpoint to own PostgreSQL
  {
    id: EP_OWN_POSTGRES_2,
    service_id: SVC_POSTGRES,
    name: "Backup DB Connection",
    description: "Backup service database connection",
    target: { network: NET_PRODUCTION },
    ip_address: "192.168.1.105",
    status: "AVAILABLE",
    tags: ["prod", "backup"],
    project_id: OWN_PROJECT,
    created_at: "2024-01-20T11:00:00Z",
    updated_at: "2024-01-20T11:00:00Z",
  },
  // Third endpoint to own PostgreSQL
  {
    id: EP_OWN_POSTGRES_3,
    service_id: SVC_POSTGRES,
    name: "Analytics DB Reader",
    description: "Read-only connection for analytics",
    target: { network: NET_ANALYTICS },
    ip_address: "192.168.1.106",
    status: "AVAILABLE",
    tags: ["analytics", "readonly"],
    project_id: OWN_PROJECT,
    created_at: "2024-06-01T11:00:00Z",
    updated_at: "2024-06-01T11:00:00Z",
  },
  // Endpoint to own Redis
  {
    id: EP_OWN_REDIS_1,
    service_id: SVC_REDIS,
    name: "Cache Connection",
    description: "Application cache connection",
    target: { network: NET_PRODUCTION },
    ip_address: "192.168.1.101",
    status: "AVAILABLE",
    tags: ["prod", "cache"],
    project_id: OWN_PROJECT,
    created_at: "2024-02-02T09:00:00Z",
    updated_at: "2024-02-02T09:00:00Z",
  },
  // Second endpoint to own Redis
  {
    id: EP_OWN_REDIS_2,
    service_id: SVC_REDIS,
    name: "Session Cache",
    description: "Session storage cache connection",
    target: { network: NET_PRODUCTION },
    ip_address: "192.168.1.107",
    status: "AVAILABLE",
    tags: ["prod", "sessions"],
    project_id: OWN_PROJECT,
    created_at: "2024-02-05T09:00:00Z",
    updated_at: "2024-02-05T09:00:00Z",
  },
  // Endpoint to own Kafka
  {
    id: EP_OWN_KAFKA_1,
    service_id: SVC_KAFKA,
    name: "Event Stream",
    description: "Kafka event streaming endpoint",
    target: { network: NET_PRODUCTION },
    ip_address: "192.168.1.102",
    status: "AVAILABLE",
    tags: ["prod", "events"],
    project_id: OWN_PROJECT,
    created_at: "2024-03-05T10:00:00Z",
    updated_at: "2024-03-05T10:00:00Z",
  },
  // Endpoint to managed API Gateway
  {
    id: EP_OWN_APIGW_1,
    service_id: SVC_CP_APIGW,
    name: "API Gateway",
    description: "Connection to managed API gateway",
    target: { network: NET_SHARED },
    ip_address: "192.168.1.200",
    status: "AVAILABLE",
    tags: ["api", "managed"],
    project_id: OWN_PROJECT,
    created_at: "2024-01-20T08:00:00Z",
    updated_at: "2024-01-20T08:00:00Z",
  },
  // Endpoint to managed Identity
  {
    id: EP_OWN_IDENTITY_1,
    service_id: SVC_CP_IDENTITY,
    name: "Identity Access",
    description: "Authentication service connection",
    target: { network: NET_SHARED },
    ip_address: "192.168.1.201",
    status: "AVAILABLE",
    tags: ["auth", "managed"],
    project_id: OWN_PROJECT,
    created_at: "2024-06-01T08:00:00Z",
    updated_at: "2024-06-01T08:00:00Z",
  },
  // Endpoint to managed Logging
  {
    id: EP_OWN_LOGGING_1,
    service_id: SVC_CP_LOGGING,
    name: "Log Shipping",
    description: "Centralized logging connection",
    target: { network: NET_SHARED },
    ip_address: "192.168.1.202",
    status: "AVAILABLE",
    tags: ["logging", "managed"],
    project_id: OWN_PROJECT,
    created_at: "2024-02-01T08:00:00Z",
    updated_at: "2024-02-01T08:00:00Z",
  },
  // Endpoint to Finance DB (cross-project)
  {
    id: EP_OWN_FINANCE_1,
    service_id: SVC_FINANCE_DB,
    name: "Finance Data",
    description: "Read access to financial data warehouse",
    target: { network: NET_ANALYTICS },
    ip_address: "192.168.1.150",
    status: "AVAILABLE",
    tags: ["finance", "reporting"],
    project_id: OWN_PROJECT,
    created_at: "2024-03-01T08:00:00Z",
    updated_at: "2024-03-01T08:00:00Z",
  },
  // Endpoint to Analytics Spark
  {
    id: EP_OWN_SPARK_1,
    service_id: SVC_SPARK,
    name: "Spark Jobs",
    description: "Analytics processing cluster access",
    target: { network: NET_ANALYTICS },
    ip_address: "192.168.1.151",
    status: "AVAILABLE",
    tags: ["analytics", "spark"],
    project_id: OWN_PROJECT,
    created_at: "2024-05-01T08:00:00Z",
    updated_at: "2024-05-01T08:00:00Z",
  },
  // Endpoint to DevOps Vault
  {
    id: EP_OWN_VAULT_1,
    service_id: SVC_VAULT,
    name: "Secrets Access",
    description: "Vault secrets management",
    target: { network: NET_SHARED },
    ip_address: "192.168.1.160",
    status: "AVAILABLE",
    tags: ["secrets", "devops"],
    project_id: OWN_PROJECT,
    created_at: "2024-02-15T08:00:00Z",
    updated_at: "2024-02-15T08:00:00Z",
  },
  // Endpoint to DevOps Registry
  {
    id: EP_OWN_REGISTRY_1,
    service_id: SVC_REGISTRY,
    name: "Container Pull",
    description: "Docker image pull access",
    target: { network: NET_DEVELOPMENT },
    ip_address: "192.168.1.161",
    status: "AVAILABLE",
    tags: ["docker", "devops"],
    project_id: OWN_PROJECT,
    created_at: "2024-01-20T08:00:00Z",
    updated_at: "2024-01-20T08:00:00Z",
  },

  // === EXTERNAL PROJECT ENDPOINTS (other projects connecting to our services) ===
  // Finance project endpoint to our PostgreSQL
  {
    id: EP_FINANCE_POSTGRES_1,
    service_id: SVC_POSTGRES,
    name: "Reporting DB",
    description: "Finance reporting database access",
    target: { subnet: SUBNET_ANALYTICS },
    ip_address: "192.168.2.100",
    status: "AVAILABLE",
    tags: ["reporting"],
    project_id: FINANCE_PROJECT,
    created_at: "2024-02-01T10:00:00Z",
    updated_at: "2024-02-01T10:00:00Z",
  },
  // Analytics project endpoint to our Kafka
  {
    id: EP_ANALYTICS_KAFKA_1,
    service_id: SVC_KAFKA,
    name: "Event Consumer",
    description: "Analytics event stream consumer",
    target: { network: NET_ANALYTICS },
    ip_address: "192.168.3.100",
    status: "PENDING_APPROVAL",
    tags: ["events", "analytics"],
    project_id: ANALYTICS_PROJECT,
    created_at: "2024-04-01T10:00:00Z",
    updated_at: "2024-04-01T10:00:00Z",
  },
  // DevOps project endpoint to our Elasticsearch
  {
    id: EP_DEVOPS_ES_1,
    service_id: SVC_ELASTICSEARCH,
    name: "Log Search",
    description: "Centralized log search access",
    target: { network: NET_STAGING },
    ip_address: "192.168.4.100",
    status: "AVAILABLE",
    tags: ["logging", "search"],
    project_id: DEVOPS_PROJECT,
    created_at: "2024-04-15T10:00:00Z",
    updated_at: "2024-04-15T10:00:00Z",
  },
  // DevOps project endpoint to our Grafana
  {
    id: EP_DEVOPS_GRAFANA_1,
    service_id: SVC_GRAFANA,
    name: "Monitoring View",
    description: "Observability dashboard access",
    target: { network: NET_SHARED },
    ip_address: "192.168.4.101",
    status: "REJECTED",
    tags: ["monitoring"],
    project_id: DEVOPS_PROJECT,
    created_at: "2024-05-20T10:00:00Z",
    updated_at: "2024-05-25T10:00:00Z",
  },
];

const consumers: EndpointConsumer[] = [
  // Own project endpoints
  { id: EP_OWN_POSTGRES_1, status: "AVAILABLE", project_id: OWN_PROJECT },
  { id: EP_OWN_POSTGRES_2, status: "AVAILABLE", project_id: OWN_PROJECT },
  { id: EP_OWN_POSTGRES_3, status: "AVAILABLE", project_id: OWN_PROJECT },
  { id: EP_OWN_REDIS_1, status: "AVAILABLE", project_id: OWN_PROJECT },
  { id: EP_OWN_REDIS_2, status: "AVAILABLE", project_id: OWN_PROJECT },
  { id: EP_OWN_KAFKA_1, status: "AVAILABLE", project_id: OWN_PROJECT },
  { id: EP_OWN_APIGW_1, status: "AVAILABLE", project_id: OWN_PROJECT },
  { id: EP_OWN_IDENTITY_1, status: "AVAILABLE", project_id: OWN_PROJECT },
  { id: EP_OWN_LOGGING_1, status: "AVAILABLE", project_id: OWN_PROJECT },
  { id: EP_OWN_FINANCE_1, status: "AVAILABLE", project_id: OWN_PROJECT },
  { id: EP_OWN_SPARK_1, status: "AVAILABLE", project_id: OWN_PROJECT },
  { id: EP_OWN_VAULT_1, status: "AVAILABLE", project_id: OWN_PROJECT },
  { id: EP_OWN_REGISTRY_1, status: "AVAILABLE", project_id: OWN_PROJECT },
  // External project endpoints requesting access to our services
  { id: EP_FINANCE_POSTGRES_1, status: "AVAILABLE", project_id: FINANCE_PROJECT },
  { id: EP_ANALYTICS_KAFKA_1, status: "PENDING_APPROVAL", project_id: ANALYTICS_PROJECT },
  { id: EP_DEVOPS_ES_1, status: "AVAILABLE", project_id: DEVOPS_PROJECT },
  { id: EP_DEVOPS_GRAFANA_1, status: "REJECTED", project_id: DEVOPS_PROJECT },
];

// Mock agents
const agents: Agent[] = [
  {
    host: "archer-agent-eu-de-1a-001.cloud.sap",
    availability_zone: "eu-de-1a",
    provider: "tenant",
    enabled: true,
    physnet: "physnet1",
    created_at: "2025-01-15T10:00:00Z",
    updated_at: "2026-04-21T08:30:00Z",
    heartbeat_at: new Date(Date.now() - 10 * 1000).toISOString(), // 10 seconds ago
    services: 12,
  },
  {
    host: "archer-agent-eu-de-1a-002.cloud.sap",
    availability_zone: "eu-de-1a",
    provider: "tenant",
    enabled: true,
    physnet: "physnet1",
    created_at: "2025-01-15T10:05:00Z",
    updated_at: "2026-04-21T08:30:00Z",
    heartbeat_at: new Date(Date.now() - 5 * 1000).toISOString(), // 5 seconds ago
    services: 8,
  },
  {
    host: "archer-agent-eu-de-1b-001.cloud.sap",
    availability_zone: "eu-de-1b",
    provider: "tenant",
    enabled: true,
    physnet: "physnet2",
    created_at: "2025-02-01T14:00:00Z",
    updated_at: "2026-04-21T08:25:00Z",
    heartbeat_at: new Date(Date.now() - 45 * 1000).toISOString(), // 45 seconds ago (yellow)
    services: 15,
  },
  {
    host: "archer-agent-eu-de-1b-002.cloud.sap",
    availability_zone: "eu-de-1b",
    provider: "tenant",
    enabled: false,
    physnet: "physnet2",
    created_at: "2025-02-01T14:05:00Z",
    updated_at: "2026-04-20T16:00:00Z",
    heartbeat_at: new Date(Date.now() - 10 * 60 * 1000).toISOString(), // 10 minutes ago (red)
    services: 0,
  },
  {
    host: "archer-cp-agent-eu-de-1a-001.cloud.sap",
    availability_zone: "eu-de-1a",
    provider: "cp",
    enabled: true,
    physnet: "cp-physnet",
    created_at: "2025-03-01T09:00:00Z",
    updated_at: "2026-04-21T08:30:00Z",
    heartbeat_at: new Date(Date.now() - 3 * 1000).toISOString(), // 3 seconds ago
    services: 5,
  },
  {
    host: "archer-cp-agent-eu-de-1b-001.cloud.sap",
    availability_zone: "eu-de-1b",
    provider: "cp",
    enabled: true,
    physnet: "cp-physnet",
    created_at: "2025-03-01T09:05:00Z",
    updated_at: "2026-04-21T08:28:00Z",
    heartbeat_at: new Date(Date.now() - 20 * 1000).toISOString(), // 20 seconds ago
    services: 3,
  },
];

// Mock Neutron networks
const networks: Network[] = [
  {
    id: NET_PRODUCTION,
    name: "Production Network",
    status: "ACTIVE",
    subnets: [SUBNET_PRODUCTION],
    project_id: OWN_PROJECT,
  },
  {
    id: NET_STAGING,
    name: "Staging Network",
    status: "ACTIVE",
    subnets: [SUBNET_STAGING],
    project_id: OWN_PROJECT,
  },
  {
    id: NET_DEVELOPMENT,
    name: "Development Network",
    status: "ACTIVE",
    subnets: [SUBNET_DEVELOPMENT],
    project_id: OWN_PROJECT,
  },
  {
    id: NET_SHARED,
    name: "Shared Services",
    status: "ACTIVE",
    subnets: [SUBNET_SHARED],
    project_id: OWN_PROJECT,
  },
  {
    id: NET_ANALYTICS,
    name: "Analytics Network",
    status: "ACTIVE",
    subnets: [SUBNET_ANALYTICS],
    project_id: OWN_PROJECT,
  },
  {
    id: NET_BACKUP,
    name: "Backup Network",
    status: "ACTIVE",
    subnets: [SUBNET_BACKUP],
    project_id: OWN_PROJECT,
  },
  {
    id: NET_DMZ,
    name: "DMZ Network",
    status: "ACTIVE",
    subnets: [],
    project_id: OWN_PROJECT,
  },
  {
    id: NET_INTERNAL,
    name: "Internal Services",
    status: "ACTIVE",
    subnets: [],
    project_id: OWN_PROJECT,
  },
];

// Mock Neutron subnets
const subnets: Subnet[] = [
  {
    id: SUBNET_PRODUCTION,
    name: "Production Subnet",
    network_id: NET_PRODUCTION,
    cidr: "10.0.1.0/24",
    ip_version: 4,
    project_id: OWN_PROJECT,
  },
  {
    id: SUBNET_STAGING,
    name: "Staging Subnet",
    network_id: NET_STAGING,
    cidr: "10.0.2.0/24",
    ip_version: 4,
    project_id: OWN_PROJECT,
  },
  {
    id: SUBNET_DEVELOPMENT,
    name: "Development Subnet",
    network_id: NET_DEVELOPMENT,
    cidr: "10.0.3.0/24",
    ip_version: 4,
    project_id: OWN_PROJECT,
  },
  {
    id: SUBNET_SHARED,
    name: "Shared Subnet",
    network_id: NET_SHARED,
    cidr: "10.0.4.0/24",
    ip_version: 4,
    project_id: OWN_PROJECT,
  },
  {
    id: SUBNET_ANALYTICS,
    name: "Analytics Subnet",
    network_id: NET_ANALYTICS,
    cidr: "10.200.0.0/24",
    ip_version: 4,
    project_id: OWN_PROJECT,
  },
  {
    id: SUBNET_BACKUP,
    name: "Backup Subnet",
    network_id: NET_BACKUP,
    cidr: "10.100.0.0/24",
    ip_version: 4,
    project_id: OWN_PROJECT,
  },
];

// Mock Neutron ports (mix of bound and unbound)
const ports: Port[] = [
  // Bound ports (attached to instances)
  {
    id: PORT_PROD_WEB,
    name: "Web Server Port",
    network_id: NET_PRODUCTION,
    status: "ACTIVE",
    device_owner: "compute:nova",
    device_id: "ccc00001-0001-4001-8001-000000000001",
    fixed_ips: [{ subnet_id: SUBNET_PRODUCTION, ip_address: "10.0.1.10" }],
    project_id: OWN_PROJECT,
  },
  {
    id: PORT_PROD_DB,
    name: "Database Port",
    network_id: NET_PRODUCTION,
    status: "ACTIVE",
    device_owner: "compute:nova",
    device_id: "ccc00002-0002-4002-8002-000000000002",
    fixed_ips: [{ subnet_id: SUBNET_PRODUCTION, ip_address: "10.0.1.20" }],
    project_id: OWN_PROJECT,
  },
  // Unbound ports (available for selection)
  {
    id: PORT_STAGING_APP,
    name: "App Server Port",
    network_id: NET_STAGING,
    status: "DOWN",
    device_owner: "",
    device_id: "",
    fixed_ips: [{ subnet_id: SUBNET_STAGING, ip_address: "10.0.2.10" }],
    project_id: OWN_PROJECT,
  },
  {
    id: PORT_DEV_TEST,
    name: "Test Instance",
    network_id: NET_DEVELOPMENT,
    status: "DOWN",
    device_owner: "",
    device_id: "",
    fixed_ips: [{ subnet_id: SUBNET_DEVELOPMENT, ip_address: "10.0.3.50" }],
    project_id: OWN_PROJECT,
  },
  {
    id: PORT_SHARED_LB,
    name: "Load Balancer VIP",
    network_id: NET_SHARED,
    status: "DOWN",
    device_owner: "",
    device_id: "",
    fixed_ips: [{ subnet_id: SUBNET_SHARED, ip_address: "10.0.4.100" }],
    project_id: OWN_PROJECT,
  },
  {
    id: PORT_RESERVED,
    name: "Reserved Port",
    network_id: NET_PRODUCTION,
    status: "DOWN",
    device_owner: "",
    device_id: "",
    fixed_ips: [{ subnet_id: SUBNET_PRODUCTION, ip_address: "10.0.1.50" }],
    project_id: OWN_PROJECT,
  },
];

const rbacPolicies: RBACPolicy[] = [
  // Allow Finance project to access our PostgreSQL
  {
    id: "01234567-0001-4001-8001-000000000001",
    service_id: SVC_POSTGRES,
    target_type: "project",
    target: FINANCE_PROJECT,
    project_id: OWN_PROJECT,
    created_at: "2024-01-17T10:00:00Z",
    updated_at: "2024-01-17T10:00:00Z",
  },
  // Allow Analytics project to access our Kafka
  {
    id: "01234567-0002-4002-8002-000000000002",
    service_id: SVC_KAFKA,
    target_type: "project",
    target: ANALYTICS_PROJECT,
    project_id: OWN_PROJECT,
    created_at: "2024-03-01T10:00:00Z",
    updated_at: "2024-03-01T10:00:00Z",
  },
  // Allow DevOps project to access our Elasticsearch
  {
    id: "01234567-0003-4003-8003-000000000003",
    service_id: SVC_ELASTICSEARCH,
    target_type: "project",
    target: DEVOPS_PROJECT,
    project_id: OWN_PROJECT,
    created_at: "2024-04-10T10:00:00Z",
    updated_at: "2024-04-10T10:00:00Z",
  },
  // Allow DevOps project to access our Grafana
  {
    id: "01234567-0004-4004-8004-000000000004",
    service_id: SVC_GRAFANA,
    target_type: "project",
    target: DEVOPS_PROJECT,
    project_id: OWN_PROJECT,
    created_at: "2024-05-01T10:00:00Z",
    updated_at: "2024-05-01T10:00:00Z",
  },
  // Wildcard RBAC for Redis (anyone can access)
  {
    id: "01234567-0005-4005-8005-000000000005",
    service_id: SVC_REDIS,
    target_type: "project",
    target: "*",
    project_id: OWN_PROJECT,
    created_at: "2024-02-01T10:00:00Z",
    updated_at: "2024-02-01T10:00:00Z",
  },
];

export const createHandlers = (baseUrl: string) => [
  // Version
  http.get(`${baseUrl}/`, () =>
    HttpResponse.json({
      version: "v2.0.0",
      updated: "2026-04-01T00:00:00Z",
    })
  ),

  // Services
  http.get(`${baseUrl}/service`, () => HttpResponse.json({ items: services })),
  http.get(`${baseUrl}/service/:id`, ({ params }) => {
    const s = services.find((x) => x.id === params.id);
    return s ? HttpResponse.json(s) : HttpResponse.json({ message: "Not found" }, { status: 404 });
  }),
  http.post(`${baseUrl}/service`, async ({ request }) => {
    const body = (await request.json()) as Partial<Service>;
    const newService: Service = {
      ...body,
      id: crypto.randomUUID(),
      status: "PENDING_CREATE",
      health_status: "UNCHECKED",
      project_id: OWN_PROJECT,
      created_at: new Date().toISOString(),
      updated_at: new Date().toISOString(),
    } as Service;
    return HttpResponse.json(newService, { status: 201 });
  }),
  http.put(`${baseUrl}/service/:id`, async ({ params, request }) => {
    const s = services.find((x) => x.id === params.id);
    if (!s) return HttpResponse.json({ message: "Not found" }, { status: 404 });
    const body = (await request.json()) as Partial<Service>;
    return HttpResponse.json({ ...s, ...body, updated_at: new Date().toISOString() });
  }),
  http.delete(`${baseUrl}/service/:id`, ({ params }) => {
    const s = services.find((x) => x.id === params.id);
    return s ? new HttpResponse(null, { status: 202 }) : HttpResponse.json({ message: "Not found" }, { status: 404 });
  }),
  http.get(`${baseUrl}/service/:id/endpoints`, ({ params }) => {
    // Get endpoint IDs that belong to this service
    const serviceEndpointIds = new Set(endpoints.filter((e) => e.service_id === params.id).map((e) => e.id));
    // Return only consumers for those endpoints
    const serviceConsumers = consumers.filter((c) => serviceEndpointIds.has(c.id));
    return HttpResponse.json({ items: serviceConsumers });
  }),
  http.put(`${baseUrl}/service/:id/accept_endpoints`, async ({ request }) => {
    const { endpoint_ids } = (await request.json()) as { endpoint_ids: string[] };
    return HttpResponse.json(
      consumers.filter((c) => endpoint_ids.includes(c.id)).map((c) => ({ ...c, status: "AVAILABLE" }))
    );
  }),
  http.put(`${baseUrl}/service/:id/reject_endpoints`, async ({ request }) => {
    const { endpoint_ids } = (await request.json()) as { endpoint_ids: string[] };
    return HttpResponse.json(
      consumers.filter((c) => endpoint_ids.includes(c.id)).map((c) => ({ ...c, status: "REJECTED" }))
    );
  }),

  // Endpoints
  http.get(`${baseUrl}/endpoint`, () => HttpResponse.json({ items: endpoints })),
  http.get(`${baseUrl}/endpoint/:id`, ({ params }) => {
    const e = endpoints.find((x) => x.id === params.id);
    return e ? HttpResponse.json(e) : HttpResponse.json({ message: "Not found" }, { status: 404 });
  }),
  http.post(`${baseUrl}/endpoint`, async ({ request }) => {
    const body = (await request.json()) as Partial<Endpoint>;
    const newEndpoint: Endpoint = {
      ...body,
      id: crypto.randomUUID(),
      status: "PENDING_CREATE",
      ip_address: `192.168.${Math.floor(Math.random() * 255)}.${Math.floor(Math.random() * 255)}`,
      project_id: OWN_PROJECT,
      created_at: new Date().toISOString(),
      updated_at: new Date().toISOString(),
    } as Endpoint;
    return HttpResponse.json(newEndpoint, { status: 201 });
  }),
  http.put(`${baseUrl}/endpoint/:id`, async ({ params, request }) => {
    const e = endpoints.find((x) => x.id === params.id);
    if (!e) return HttpResponse.json({ message: "Not found" }, { status: 404 });
    const body = (await request.json()) as Partial<Endpoint>;
    return HttpResponse.json({ ...e, ...body, updated_at: new Date().toISOString() });
  }),
  http.delete(`${baseUrl}/endpoint/:id`, ({ params }) => {
    const e = endpoints.find((x) => x.id === params.id);
    return e ? new HttpResponse(null, { status: 202 }) : HttpResponse.json({ message: "Not found" }, { status: 404 });
  }),

  // RBAC
  http.get(`${baseUrl}/rbac-policies`, () => HttpResponse.json({ items: rbacPolicies })),
  http.post(`${baseUrl}/rbac-policies`, async ({ request }) => {
    const body = (await request.json()) as Partial<RBACPolicy>;
    const newPolicy: RBACPolicy = {
      ...body,
      id: `rbac-${crypto.randomUUID()}`,
      target_type: "project",
      project_id: OWN_PROJECT,
      created_at: new Date().toISOString(),
      updated_at: new Date().toISOString(),
    } as RBACPolicy;
    return HttpResponse.json(newPolicy, { status: 201 });
  }),
  http.put(`${baseUrl}/rbac-policies/:id`, async ({ params, request }) => {
    const p = rbacPolicies.find((x) => x.id === params.id);
    if (!p) return HttpResponse.json({ message: "Not found" }, { status: 404 });
    const body = (await request.json()) as Partial<RBACPolicy>;
    return HttpResponse.json({ ...p, ...body, updated_at: new Date().toISOString() });
  }),
  http.delete(`${baseUrl}/rbac-policies/:id`, ({ params }) => {
    const p = rbacPolicies.find((x) => x.id === params.id);
    return p ? new HttpResponse(null, { status: 204 }) : HttpResponse.json({ message: "Not found" }, { status: 404 });
  }),

  // Agents
  http.get(`${baseUrl}/agents`, () => HttpResponse.json({ items: agents })),

  // Neutron Networks (mock - matches any neutron endpoint)
  http.get(/.*\/v2\.0\/networks.*/, () => HttpResponse.json({ networks })),
  http.get(/.*\/v2\.0\/subnets.*/, () => HttpResponse.json({ subnets })),
  http.get(/.*\/v2\.0\/ports.*/, () => HttpResponse.json({ ports })),
];
