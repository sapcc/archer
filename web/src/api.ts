// SPDX-FileCopyrightText: 2026 SAP SE
// SPDX-License-Identifier: Apache-2.0

import { useQuery, useQueries, useMutation, useQueryClient } from "@tanstack/react-query";
import { useStore } from "./store";
import type {
  Service,
  ServiceCreate,
  ServiceUpdate,
  Endpoint,
  EndpointCreate,
  EndpointUpdate,
  EndpointConsumer,
  RBACPolicy,
  RBACPolicyCreate,
  RBACPolicyUpdate,
  Agent,
  ListResponse,
  Version,
  NetworksResponse,
  SubnetsResponse,
  PortsResponse,
} from "./types";

// Pending status helpers
export const isPendingService = (s: Service): boolean =>
  s.status === "PENDING_CREATE" || s.status === "PENDING_UPDATE" || s.status === "PENDING_DELETE";

export const isPendingEndpoint = (e: { status: string }): boolean =>
  e.status === "PENDING_APPROVAL" ||
  e.status === "PENDING_CREATE" ||
  e.status === "PENDING_UPDATE" ||
  e.status === "PENDING_REJECTED" ||
  e.status === "PENDING_DELETE";

// Transient pending = states the backend will resolve on its own.
// Excludes PENDING_APPROVAL and PENDING_REJECTED, which wait on human action and shouldn't drive polling.
const isTransientPending = (e: { status: string }): boolean =>
  e.status === "PENDING_CREATE" || e.status === "PENDING_UPDATE" || e.status === "PENDING_DELETE";

// Backoff schedule for queries that poll while something is transitioning.
// Avoids hammering when state stays pending longer than expected.
const pollDelay = (failureCount: number, dataUpdateCount: number): number => {
  // Use whichever counter is higher — covers both repeated failures and repeated pending observations.
  const n = Math.max(failureCount, dataUpdateCount);
  if (n < 5) return 3000;
  if (n < 15) return 10000;
  if (n < 30) return 30000;
  return 60000;
};

const getHeaders = (token: string, withContent = false) => ({
  Accept: "application/json",
  "X-Auth-Token": token,
  ...(withContent && { "Content-Type": "application/json" }),
});

async function fetchJson<T>(url: string, init: RequestInit): Promise<T | null> {
  const res = await fetch(url, init);
  if (!res.ok) {
    const text = await res.text();
    let message = text;
    try {
      const json = JSON.parse(text);
      message = json.message || JSON.stringify(json, null, 2);
    } catch {
      // Not JSON, use raw text
    }
    throw new Error(`${message} (${res.status})`);
  }
  if (res.status === 204 || res.status === 202) return null;
  return res.json();
}

// For endpoints we know never return null on success (GETs, normal POSTs).
async function fetchJsonRequired<T>(url: string, init: RequestInit): Promise<T> {
  const result = await fetchJson<T>(url, init);
  if (result === null) {
    throw new Error(`Expected response body from ${url}, got 202/204`);
  }
  return result;
}

// Version
export const useVersion = () => {
  const { endpoint, token } = useStore((s) => s.globalAPI);
  return useQuery({
    queryKey: ["version"],
    queryFn: () =>
      fetchJsonRequired<Version>(`${endpoint}/`, {
        headers: getHeaders(token),
      }),
    enabled: !!endpoint && !!token,
  });
};

// Services
export const useServices = () => {
  const { endpoint, token } = useStore((s) => s.globalAPI);
  return useQuery({
    queryKey: ["services"],
    queryFn: () =>
      fetchJsonRequired<ListResponse<Service>>(`${endpoint}/service`, {
        headers: getHeaders(token),
      }),
    enabled: !!endpoint && !!token,
    refetchInterval: (query) => {
      const items = query.state.data?.items ?? [];
      if (!items.some(isPendingService)) return false;
      return pollDelay(query.state.fetchFailureCount, query.state.dataUpdateCount);
    },
  });
};

export const useService = (id: string | undefined) => {
  const { endpoint, token } = useStore((s) => s.globalAPI);
  return useQuery({
    queryKey: ["service", id],
    queryFn: () =>
      fetchJsonRequired<Service>(`${endpoint}/service/${id}`, {
        headers: getHeaders(token),
      }),
    enabled: !!endpoint && !!token && !!id,
    refetchInterval: (query) => {
      const service = query.state.data;
      if (!service || !isPendingService(service)) return false;
      return pollDelay(query.state.fetchFailureCount, query.state.dataUpdateCount);
    },
  });
};

export const useServiceEndpoints = (serviceId: string | undefined) => {
  const { endpoint, token } = useStore((s) => s.globalAPI);
  return useQuery({
    queryKey: ["serviceEndpoints", serviceId],
    queryFn: () =>
      fetchJsonRequired<ListResponse<EndpointConsumer>>(`${endpoint}/service/${serviceId}/endpoints`, {
        headers: getHeaders(token),
      }),
    enabled: !!endpoint && !!token && !!serviceId,
    refetchInterval: (query) => {
      const items = query.state.data?.items ?? [];
      if (!items.some(isTransientPending)) return false;
      return pollDelay(query.state.fetchFailureCount, query.state.dataUpdateCount);
    },
  });
};

// Batch consumer fetch for N services. Polls per-service while any consumer is pending.
export const useServiceConsumers = (serviceIds: string[]) => {
  const { endpoint, token } = useStore((s) => s.globalAPI);
  return useQueries({
    queries: serviceIds.map((id) => ({
      queryKey: ["serviceEndpoints", id],
      queryFn: () =>
        fetchJsonRequired<ListResponse<EndpointConsumer>>(`${endpoint}/service/${id}/endpoints`, {
          headers: getHeaders(token),
        }),
      enabled: !!endpoint && !!token && !!id,
      refetchInterval: (query: {
        state: { data?: ListResponse<EndpointConsumer>; fetchFailureCount: number; dataUpdateCount: number };
      }) => {
        const items = query.state.data?.items ?? [];
        if (!items.some(isTransientPending)) return false;
        return pollDelay(query.state.fetchFailureCount, query.state.dataUpdateCount);
      },
    })),
  });
};

export const useCreateService = () => {
  const { endpoint, token } = useStore((s) => s.globalAPI);
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (data: ServiceCreate) =>
      fetchJsonRequired<Service>(`${endpoint}/service`, {
        method: "POST",
        headers: getHeaders(token, true),
        body: JSON.stringify(data),
      }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["services"] }),
  });
};

export const useUpdateService = () => {
  const { endpoint, token } = useStore((s) => s.globalAPI);
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, data }: { id: string; data: ServiceUpdate }) =>
      fetchJsonRequired<Service>(`${endpoint}/service/${id}`, {
        method: "PUT",
        headers: getHeaders(token, true),
        body: JSON.stringify(data),
      }),
    onSuccess: (_, { id }) => {
      qc.invalidateQueries({ queryKey: ["services"] });
      qc.invalidateQueries({ queryKey: ["service", id] });
    },
  });
};

export const useDeleteService = () => {
  const { endpoint, token } = useStore((s) => s.globalAPI);
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, cascade = false }: { id: string; cascade?: boolean }) =>
      fetchJson<void>(`${endpoint}/service/${id}${cascade ? "?cascade=true" : ""}`, {
        method: "DELETE",
        headers: getHeaders(token),
      }),
    onSuccess: (_, { id }) => {
      qc.invalidateQueries({ queryKey: ["services"] });
      qc.removeQueries({ queryKey: ["service", id] });
      qc.removeQueries({ queryKey: ["serviceEndpoints", id] });
    },
  });
};

export const useMigrateService = () => {
  const { endpoint, token } = useStore((s) => s.globalAPI);
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, targetHost }: { id: string; targetHost?: string }) =>
      fetchJsonRequired<Service>(`${endpoint}/service/${id}/migrate`, {
        method: "POST",
        headers: getHeaders(token, true),
        body: JSON.stringify(targetHost ? { target_host: targetHost } : {}),
      }),
    onSuccess: (_, { id }) => {
      qc.invalidateQueries({ queryKey: ["services"] });
      qc.invalidateQueries({ queryKey: ["service", id] });
    },
  });
};

export const useAcceptEndpoints = () => {
  const { endpoint, token } = useStore((s) => s.globalAPI);
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ serviceId, endpointIds }: { serviceId: string; endpointIds: string[] }) =>
      fetchJsonRequired<EndpointConsumer[]>(`${endpoint}/service/${serviceId}/accept_endpoints`, {
        method: "PUT",
        headers: getHeaders(token, true),
        body: JSON.stringify({ endpoint_ids: endpointIds }),
      }),
    onSuccess: (_, { serviceId }) => {
      qc.invalidateQueries({ queryKey: ["serviceEndpoints", serviceId] });
      qc.invalidateQueries({ queryKey: ["endpoints"] });
    },
  });
};

export const useRejectEndpoints = () => {
  const { endpoint, token } = useStore((s) => s.globalAPI);
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ serviceId, endpointIds }: { serviceId: string; endpointIds: string[] }) =>
      fetchJsonRequired<EndpointConsumer[]>(`${endpoint}/service/${serviceId}/reject_endpoints`, {
        method: "PUT",
        headers: getHeaders(token, true),
        body: JSON.stringify({ endpoint_ids: endpointIds }),
      }),
    onSuccess: (_, { serviceId }) => {
      qc.invalidateQueries({ queryKey: ["serviceEndpoints", serviceId] });
      qc.invalidateQueries({ queryKey: ["endpoints"] });
    },
  });
};

// Endpoints
export const useEndpoints = () => {
  const { endpoint, token } = useStore((s) => s.globalAPI);
  return useQuery({
    queryKey: ["endpoints"],
    queryFn: () =>
      fetchJsonRequired<ListResponse<Endpoint>>(`${endpoint}/endpoint`, {
        headers: getHeaders(token),
      }),
    enabled: !!endpoint && !!token,
    refetchInterval: (query) => {
      const items = query.state.data?.items ?? [];
      if (!items.some(isTransientPending)) return false;
      return pollDelay(query.state.fetchFailureCount, query.state.dataUpdateCount);
    },
  });
};

export const useEndpoint = (id: string | undefined) => {
  const { endpoint, token } = useStore((s) => s.globalAPI);
  return useQuery({
    queryKey: ["endpoint", id],
    queryFn: () =>
      fetchJsonRequired<Endpoint>(`${endpoint}/endpoint/${id}`, {
        headers: getHeaders(token),
      }),
    enabled: !!endpoint && !!token && !!id,
    refetchInterval: (query) => {
      const ep = query.state.data;
      if (!ep || !isTransientPending(ep)) return false;
      return pollDelay(query.state.fetchFailureCount, query.state.dataUpdateCount);
    },
  });
};

export const useCreateEndpoint = () => {
  const { endpoint, token } = useStore((s) => s.globalAPI);
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (data: EndpointCreate) =>
      fetchJsonRequired<Endpoint>(`${endpoint}/endpoint`, {
        method: "POST",
        headers: getHeaders(token, true),
        body: JSON.stringify(data),
      }),
    onSuccess: (created) => {
      qc.invalidateQueries({ queryKey: ["endpoints"] });
      qc.invalidateQueries({ queryKey: ["serviceEndpoints", created.service_id] });
    },
  });
};

export const useUpdateEndpoint = () => {
  const { endpoint, token } = useStore((s) => s.globalAPI);
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, data }: { id: string; data: EndpointUpdate }) =>
      fetchJsonRequired<Endpoint>(`${endpoint}/endpoint/${id}`, {
        method: "PUT",
        headers: getHeaders(token, true),
        body: JSON.stringify(data),
      }),
    onSuccess: (updated, { id }) => {
      qc.invalidateQueries({ queryKey: ["endpoints"] });
      qc.invalidateQueries({ queryKey: ["endpoint", id] });
      qc.invalidateQueries({ queryKey: ["serviceEndpoints", updated.service_id] });
    },
  });
};

export const useDeleteEndpoint = () => {
  const { endpoint, token } = useStore((s) => s.globalAPI);
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) =>
      fetchJson<void>(`${endpoint}/endpoint/${id}`, {
        method: "DELETE",
        headers: getHeaders(token),
      }),
    onSuccess: (_, id) => {
      qc.invalidateQueries({ queryKey: ["endpoints"] });
      qc.removeQueries({ queryKey: ["endpoint", id] });
      // We don't know which service this endpoint belonged to once it's deleted,
      // so invalidate all serviceEndpoints queries.
      qc.invalidateQueries({ queryKey: ["serviceEndpoints"] });
    },
  });
};

// RBAC Policies
export const useRBACPolicies = () => {
  const { endpoint, token } = useStore((s) => s.globalAPI);
  return useQuery({
    queryKey: ["rbacPolicies"],
    queryFn: () =>
      fetchJsonRequired<ListResponse<RBACPolicy>>(`${endpoint}/rbac-policies`, {
        headers: getHeaders(token),
      }),
    enabled: !!endpoint && !!token,
  });
};

export const useCreateRBAC = () => {
  const { endpoint, token } = useStore((s) => s.globalAPI);
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (data: RBACPolicyCreate) =>
      fetchJsonRequired<RBACPolicy>(`${endpoint}/rbac-policies`, {
        method: "POST",
        headers: getHeaders(token, true),
        body: JSON.stringify(data),
      }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["rbacPolicies"] }),
  });
};

export const useUpdateRBAC = () => {
  const { endpoint, token } = useStore((s) => s.globalAPI);
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, data }: { id: string; data: RBACPolicyUpdate }) =>
      fetchJsonRequired<RBACPolicy>(`${endpoint}/rbac-policies/${id}`, {
        method: "PUT",
        headers: getHeaders(token, true),
        body: JSON.stringify(data),
      }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["rbacPolicies"] }),
  });
};

export const useDeleteRBAC = () => {
  const { endpoint, token } = useStore((s) => s.globalAPI);
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) =>
      fetchJson<void>(`${endpoint}/rbac-policies/${id}`, {
        method: "DELETE",
        headers: getHeaders(token),
      }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["rbacPolicies"] }),
  });
};

// Agents (admin only)
export const useAgents = () => {
  const { endpoint, token } = useStore((s) => s.globalAPI);
  return useQuery({
    queryKey: ["agents"],
    queryFn: () =>
      fetchJsonRequired<ListResponse<Agent>>(`${endpoint}/agents`, {
        headers: getHeaders(token),
      }),
    enabled: !!endpoint && !!token,
  });
};

// Networks (from Neutron API)
// In development, requests go through /proxy/neutron to bypass CORS
export const useNetworks = (enabled: boolean) => {
  const { neutronEndpoint, token } = useStore((s) => s.globalAPI);
  const isDev = typeof window !== "undefined" && window.location.hostname === "localhost";
  const baseUrl = isDev ? "/proxy/neutron" : neutronEndpoint;

  return useQuery({
    queryKey: ["networks"],
    queryFn: () =>
      fetchJsonRequired<NetworksResponse>(`${baseUrl}/v2.0/networks?limit=100`, {
        headers: getHeaders(token),
      }),
    enabled: enabled && !!neutronEndpoint && !!token,
    staleTime: 60 * 1000,
  });
};

// Subnets (from Neutron API)
export const useSubnets = (enabled: boolean) => {
  const { neutronEndpoint, token } = useStore((s) => s.globalAPI);
  const isDev = typeof window !== "undefined" && window.location.hostname === "localhost";
  const baseUrl = isDev ? "/proxy/neutron" : neutronEndpoint;

  return useQuery({
    queryKey: ["subnets"],
    queryFn: () =>
      fetchJsonRequired<SubnetsResponse>(`${baseUrl}/v2.0/subnets?limit=100`, {
        headers: getHeaders(token),
      }),
    enabled: enabled && !!neutronEndpoint && !!token,
    staleTime: 60 * 1000,
  });
};

// Ports (from Neutron API) - only unbound ports
export const usePorts = (enabled: boolean) => {
  const { neutronEndpoint, token } = useStore((s) => s.globalAPI);
  const isDev = typeof window !== "undefined" && window.location.hostname === "localhost";
  const baseUrl = isDev ? "/proxy/neutron" : neutronEndpoint;

  return useQuery({
    queryKey: ["ports"],
    queryFn: () =>
      fetchJsonRequired<PortsResponse>(`${baseUrl}/v2.0/ports?limit=100&device_owner=&device_id=`, {
        headers: getHeaders(token),
      }),
    enabled: enabled && !!neutronEndpoint && !!token,
    staleTime: 60 * 1000,
  });
};
