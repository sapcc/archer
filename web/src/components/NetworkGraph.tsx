// SPDX-FileCopyrightText: 2026 SAP SE
// SPDX-License-Identifier: Apache-2.0

import { useMemo, useState, useCallback, useEffect, type MouseEvent } from "react";
import {
  ReactFlow,
  Node,
  Edge,
  Background,
  BackgroundVariant,
  Controls,
  MiniMap,
  useNodesState,
  useEdgesState,
  MarkerType,
  Position,
  ReactFlowProvider,
  useReactFlow,
  Handle,
  ConnectionLineType,
} from "@xyflow/react";
import { LoadingIndicator, Message, Stack } from "@cloudoperators/juno-ui-components";
import { useServices, useEndpoints, useServiceConsumers } from "../api";
import { useStore } from "../store";
import type { Service, Endpoint, EndpointConsumer } from "../types";

// Use Juno CSS variables for status colors
const STATUS_COLORS: Record<string, string> = {
  AVAILABLE: "var(--color-success)",
  ONLINE: "var(--color-success)",
  PENDING_CREATE: "var(--color-warning)",
  PENDING_UPDATE: "var(--color-warning)",
  PENDING_DELETE: "var(--color-warning)",
  PENDING_APPROVAL: "var(--color-warning)",
  PENDING_REJECTED: "var(--color-danger)",
  REJECTED: "var(--color-danger)",
  FAILED: "var(--color-danger)",
  ERROR_QUOTA: "var(--color-danger)",
  UNAVAILABLE: "var(--color-text-light)",
  DEGRADED: "var(--color-warning)",
  OFFLINE: "var(--color-danger)",
  UNCHECKED: "var(--color-text-light)",
};

const getStatusColor = (status: string | null) => STATUS_COLORS[status || ""] || "var(--color-text-light)";

const isPending = (status: string | null) => status?.startsWith("PENDING") || status === "UNCHECKED";

// Group bubble node
const GroupBubbleNode = ({
  data,
}: {
  data: { label: string; sublabel?: string; type: "own" | "external" | "managed"; width: number; height: number };
}) => {
  const colors = {
    own: {
      bg: "color-mix(in srgb, var(--color-success) 10%, transparent)",
      border: "var(--color-success)",
      text: "var(--color-success)",
    },
    external: {
      bg: "var(--color-background-lvl-3)",
      border: "var(--color-text-light)",
      text: "var(--color-text-light)",
    },
    managed: { bg: "rgba(168, 85, 247, 0.1)", border: "rgb(168, 85, 247)", text: "rgb(168, 85, 247)" },
  };
  const c = colors[data.type];

  return (
    <div
      className="rounded-2xl border-2 border-dashed relative"
      style={{
        width: data.width,
        height: data.height,
        backgroundColor: c.bg,
        borderColor: c.border,
      }}
    >
      <div
        className="absolute -top-3 left-4 px-2 py-0.5 rounded text-xs font-semibold z-10"
        style={{
          backgroundColor: "var(--color-background-lvl-0)",
          borderColor: c.border,
          color: c.text,
          border: `1px solid ${c.border}`,
        }}
      >
        {data.label}
      </div>
      {data.sublabel && (
        <div className="absolute top-2 right-3 text-[10px] opacity-60" style={{ color: c.text }}>
          {data.sublabel}
        </div>
      )}
    </div>
  );
};

// Service node
const ServiceNode = ({
  data,
}: {
  data: { label: string; service: Service; isExternal: boolean; isManaged: boolean };
}) => {
  const { service, isExternal, isManaged } = data;
  // Use darker green for own services to ensure white text contrast
  const bgColor = isManaged ? "rgb(124, 58, 237)" : isExternal ? "var(--color-background-lvl-4)" : "#16a34a";
  const textColor = isManaged || !isExternal ? "white" : "var(--color-text-default)";
  const pending = isPending(service.status);
  // Show colored border based on service status (not health_status)
  const isHealthy = service.status === "AVAILABLE";
  const borderColor = isHealthy ? "transparent" : getStatusColor(service.status);

  return (
    <div
      className="px-3 py-2 rounded-lg shadow-lg min-w-[150px] cursor-pointer hover:brightness-110 transition-all"
      style={{
        backgroundColor: bgColor,
        border: isHealthy ? "none" : `3px solid ${borderColor}`,
        animation: pending ? "pulse-glow 2s ease-in-out infinite" : undefined,
      }}
    >
      <Handle type="target" position={Position.Left} className="!bg-white !w-2.5 !h-2.5 !border-2 !border-gray-600" />
      <div className="flex items-center justify-between gap-2">
        <span className="text-sm font-bold truncate max-w-[120px]" style={{ color: textColor }}>
          {data.label}
        </span>
        <span
          className="w-3 h-3 rounded-full flex-shrink-0 border border-white"
          style={{ backgroundColor: getStatusColor(service.health_status) }}
          title={`Health: ${service.health_status || "Unknown"}`}
        />
      </div>
      <div className="text-xs mt-1" style={{ color: textColor, opacity: 0.7 }}>
        {service.visibility} · {service.protocol}
      </div>
    </div>
  );
};

// Endpoint node
const EndpointNode = ({ data }: { data: { label: string; endpoint: Endpoint } }) => {
  const { endpoint } = data;
  const pending = isPending(endpoint.status);
  const isHealthy = endpoint.status === "AVAILABLE";
  const borderColor = isHealthy ? "transparent" : getStatusColor(endpoint.status);

  return (
    <div
      className="px-3 py-2 rounded-full shadow-lg min-w-[130px] cursor-pointer hover:brightness-110 transition-all"
      style={{
        backgroundColor: "#0d9488",
        border: isHealthy ? "none" : `3px solid ${borderColor}`,
        animation: pending ? "pulse-glow 2s ease-in-out infinite" : undefined,
      }}
    >
      <Handle type="source" position={Position.Right} className="!bg-white !w-2.5 !h-2.5 !border-2 !border-gray-600" />
      <div className="flex items-center justify-between gap-2">
        <span className="text-white text-sm font-bold truncate max-w-[90px]">{data.label}</span>
        <span
          className="w-3 h-3 rounded-full flex-shrink-0 border border-white"
          style={{ backgroundColor: getStatusColor(endpoint.status) }}
          title={`Status: ${endpoint.status}`}
        />
      </div>
      <div className="text-white/60 text-xs mt-0.5 text-center truncate">{endpoint.ip_address}</div>
    </div>
  );
};

const nodeTypes = {
  service: ServiceNode,
  endpoint: EndpointNode,
  groupBubble: GroupBubbleNode,
};

const NetworkGraphInner = () => {
  const { data: servicesData, isLoading: servicesLoading, error: servicesError } = useServices();
  const { data: endpointsData, isLoading: endpointsLoading, error: endpointsError } = useEndpoints();
  const projectID = useStore((s) => s.globalAPI.projectID);
  const theme = useStore((s) => s.globalAPI.theme);
  const { fitView } = useReactFlow();
  const isDark = theme === "theme-dark";

  const [selectedNode, setSelectedNode] = useState<Node | null>(null);
  const [isUnlocked, setIsUnlocked] = useState(false);

  const services = useMemo(() => servicesData?.items ?? [], [servicesData]);
  const endpoints = useMemo(() => endpointsData?.items ?? [], [endpointsData]);

  // Own services that we need to fetch consumers for
  const ownServices = useMemo(
    () => services.filter((s) => s.project_id === projectID && s.provider !== "cp"),
    [services, projectID]
  );

  const ownServiceIds = useMemo(() => ownServices.map((s) => s.id), [ownServices]);
  const consumerQueries = useServiceConsumers(ownServiceIds);

  // Stable signature of all consumer data so memos downstream don't fire when nothing changed.
  // useQueries returns a new array reference every render, so we can't depend on it directly.
  const consumersSignature = consumerQueries
    .map((q) => (q.data?.items ?? []).map((c) => `${c.id}:${c.status}:${c.project_id}`).join(","))
    .join("|");

  // Aggregate all foreign consumers (from other projects, attached to our services)
  const allForeignConsumers = useMemo(() => {
    const result: (EndpointConsumer & { service_id: string })[] = [];
    ownServices.forEach((service, idx) => {
      const items = consumerQueries[idx]?.data?.items ?? [];
      for (const c of items) {
        if (c.project_id !== projectID) {
          result.push({ ...c, service_id: service.id });
        }
      }
    });
    return result;
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [ownServices, consumersSignature, projectID]);

  const { initialNodes, initialEdges } = useMemo(() => {
    const nodes: Node[] = [];
    const edges: Edge[] = [];

    // Categorize services
    const managedServices = services.filter((s) => s.provider === "cp");
    const ownServicesFiltered = services.filter((s) => s.project_id === projectID && s.provider !== "cp");
    const externalServices = services.filter((s) => s.project_id !== projectID && s.provider !== "cp");

    // Group external services by project
    const externalByProject = new Map<string, Service[]>();
    externalServices.forEach((s) => {
      if (!externalByProject.has(s.project_id)) externalByProject.set(s.project_id, []);
      externalByProject.get(s.project_id)!.push(s);
    });

    // Group foreign consumers by project
    const foreignByProject = new Map<string, (EndpointConsumer & { service_id: string })[]>();
    allForeignConsumers.forEach((c) => {
      if (!foreignByProject.has(c.project_id)) foreignByProject.set(c.project_id, []);
      foreignByProject.get(c.project_id)!.push(c);
    });

    // Get own endpoints
    const ownEndpoints = endpoints.filter((e) => e.project_id === projectID);

    const nodeWidth = 160;
    const nodeHeight = 55;
    const nodePadding = 25;
    const groupPadding = 50;
    const sectionGap = 60;
    const verticalGap = 30;
    const startY = 60;

    // === Calculate foreign consumer groups width first (to position them on the left) ===
    let foreignGroupMaxWidth = 0;
    if (foreignByProject.size > 0) {
      Array.from(foreignByProject.values()).forEach((foreignConsumers) => {
        const cols = Math.min(foreignConsumers.length, 2);
        const groupWidth = cols * (nodeWidth + nodePadding) + groupPadding;
        foreignGroupMaxWidth = Math.max(foreignGroupMaxWidth, groupWidth);
      });
    }

    let currentX = 50;

    // === Foreign Consumers (endpoints from other projects using our services) ===
    // Position them on the left, before the current project
    let foreignY = startY;
    Array.from(foreignByProject.entries()).forEach(([foreignProjectId, foreignConsumers]) => {
      const cols = Math.min(foreignConsumers.length, 2);
      const rows = Math.ceil(foreignConsumers.length / cols);
      const groupWidth = cols * (nodeWidth + nodePadding) + groupPadding;
      const groupHeight = rows * (nodeHeight + nodePadding) + groupPadding + 20;

      nodes.push({
        id: `group-foreign-${foreignProjectId || "unknown"}`,
        type: "groupBubble",
        position: { x: currentX, y: foreignY },
        data: {
          label: "External Consumers",
          sublabel: foreignProjectId || "unknown project",
          type: "external",
          width: groupWidth,
          height: groupHeight,
        },
        draggable: isUnlocked,
        selectable: false,
        zIndex: -10,
      });

      foreignConsumers.forEach((c, i) => {
        const col = i % cols;
        const row = Math.floor(i / cols);
        // Create a minimal endpoint-like object for display
        const truncatedLabel = `${c.id.slice(0, 8)}…`;
        const consumerAsEndpoint: Endpoint = {
          id: c.id,
          service_id: c.service_id,
          name: truncatedLabel,
          description: null,
          status: c.status,
          project_id: c.project_id,
          target: {},
          ip_address: "",
          tags: [],
          created_at: "",
          updated_at: "",
        };
        nodes.push({
          id: `consumer-${c.id}`,
          type: "endpoint",
          position: {
            x: groupPadding / 2 + col * (nodeWidth + nodePadding),
            y: groupPadding + row * (nodeHeight + nodePadding),
          },
          parentId: `group-foreign-${foreignProjectId || "unknown"}`,
          data: { label: truncatedLabel, endpoint: consumerAsEndpoint },
        });
      });

      foreignY += groupHeight + verticalGap;
    });

    // Move currentX past the foreign consumer groups
    if (foreignGroupMaxWidth > 0) {
      currentX += foreignGroupMaxWidth + sectionGap;
    }

    // === Own Project (Endpoints on left, Services on right) ===
    if (ownServicesFiltered.length > 0 || ownEndpoints.length > 0) {
      const endpointCols = 2;
      const endpointRows = Math.ceil(ownEndpoints.length / endpointCols);
      const serviceCols = 2;
      const serviceRows = Math.ceil(ownServicesFiltered.length / serviceCols);

      const endpointsWidth = ownEndpoints.length > 0 ? endpointCols * (nodeWidth + nodePadding) : 0;
      const servicesWidth = ownServicesFiltered.length > 0 ? serviceCols * (nodeWidth + nodePadding) : 0;
      const groupWidth =
        endpointsWidth +
        servicesWidth +
        groupPadding +
        (ownEndpoints.length > 0 && ownServicesFiltered.length > 0 ? 20 : 0);

      const endpointsHeight = ownEndpoints.length > 0 ? endpointRows * (nodeHeight + nodePadding) : 0;
      const servicesHeight = ownServicesFiltered.length > 0 ? serviceRows * (nodeHeight + nodePadding) : 0;
      const groupHeight = Math.max(endpointsHeight, servicesHeight) + groupPadding + 20;

      nodes.push({
        id: "group-own",
        type: "groupBubble",
        position: { x: currentX, y: startY },
        data: { label: "Current Project", sublabel: projectID, type: "own", width: groupWidth, height: groupHeight },
        draggable: isUnlocked,
        selectable: false,
        zIndex: -10,
      });

      // Add own endpoints on the left
      ownEndpoints.forEach((e, i) => {
        const col = i % endpointCols;
        const row = Math.floor(i / endpointCols);
        nodes.push({
          id: `endpoint-${e.id}`,
          type: "endpoint",
          position: {
            x: groupPadding / 2 + col * (nodeWidth + nodePadding),
            y: groupPadding + row * (nodeHeight + nodePadding),
          },
          parentId: "group-own",
          data: { label: e.name || e.id.slice(0, 8), endpoint: e },
        });
      });

      // Add own services on the right
      const servicesStartX = endpointsWidth + (ownEndpoints.length > 0 ? 20 : 0);
      ownServicesFiltered.forEach((s, i) => {
        const col = i % serviceCols;
        const row = Math.floor(i / serviceCols);
        nodes.push({
          id: `service-${s.id}`,
          type: "service",
          position: {
            x: groupPadding / 2 + servicesStartX + col * (nodeWidth + nodePadding),
            y: groupPadding + row * (nodeHeight + nodePadding),
          },
          parentId: "group-own",
          data: { label: s.name || s.id.slice(0, 8), service: s, isExternal: false, isManaged: false },
        });
      });

      currentX += groupWidth + sectionGap;
    }

    // === Managed Services (Control Plane) ===
    const managedX = currentX;
    if (managedServices.length > 0) {
      const cols = Math.min(managedServices.length, 2);
      const rows = Math.ceil(managedServices.length / cols);
      const groupWidth = cols * (nodeWidth + nodePadding) + groupPadding;
      const groupHeight = rows * (nodeHeight + nodePadding) + groupPadding + 20;

      nodes.push({
        id: "group-managed",
        type: "groupBubble",
        position: { x: currentX, y: startY },
        data: {
          label: "Managed Services",
          sublabel: "Control Plane",
          type: "managed",
          width: groupWidth,
          height: groupHeight,
        },
        draggable: isUnlocked,
        selectable: false,
        zIndex: -10,
      });

      managedServices.forEach((s, i) => {
        const col = i % cols;
        const row = Math.floor(i / cols);
        nodes.push({
          id: `service-${s.id}`,
          type: "service",
          position: {
            x: groupPadding / 2 + col * (nodeWidth + nodePadding),
            y: groupPadding + row * (nodeHeight + nodePadding),
          },
          parentId: "group-managed",
          data: { label: s.name || s.id.slice(0, 8), service: s, isExternal: false, isManaged: true },
        });
      });

      currentX += groupWidth + sectionGap;
    }

    // === External Services (grouped by project, stacked vertically) ===
    const externalX = currentX;
    let externalY = startY;

    Array.from(externalByProject.entries()).forEach(([extProjectId, extServices]) => {
      const cols = Math.min(extServices.length, 2);
      const rows = Math.ceil(extServices.length / cols);
      const groupWidth = cols * (nodeWidth + nodePadding) + groupPadding;
      const groupHeight = rows * (nodeHeight + nodePadding) + groupPadding + 20;

      nodes.push({
        id: `group-external-${extProjectId}`,
        type: "groupBubble",
        position: { x: externalX, y: externalY },
        data: {
          label: "External Services",
          sublabel: extProjectId,
          type: "external",
          width: groupWidth,
          height: groupHeight,
        },
        draggable: isUnlocked,
        selectable: false,
        zIndex: -10,
      });

      extServices.forEach((s, i) => {
        const col = i % cols;
        const row = Math.floor(i / cols);
        nodes.push({
          id: `service-${s.id}`,
          type: "service",
          position: {
            x: groupPadding / 2 + col * (nodeWidth + nodePadding),
            y: groupPadding + row * (nodeHeight + nodePadding),
          },
          parentId: `group-external-${extProjectId}`,
          data: { label: s.name || s.id.slice(0, 8), service: s, isExternal: true, isManaged: false },
        });
      });

      externalY += groupHeight + verticalGap;
    });

    // === Create edges from endpoints to services ===
    endpoints.forEach((e) => {
      const edgeColor = getStatusColor(e.status);

      edges.push({
        id: `edge-${e.id}`,
        source: `endpoint-${e.id}`,
        target: `service-${e.service_id}`,
        type: "default",
        animated: true,
        zIndex: 1000,
        style: {
          stroke: edgeColor,
          strokeWidth: 2,
        },
        markerEnd: { type: MarkerType.ArrowClosed, color: edgeColor, width: 20, height: 20 },
      });
    });

    // === Create edges from foreign consumers to own services ===
    allForeignConsumers.forEach((c) => {
      const edgeColor = getStatusColor(c.status);

      edges.push({
        id: `edge-consumer-${c.id}`,
        source: `consumer-${c.id}`,
        target: `service-${c.service_id}`,
        type: "default",
        animated: true,
        zIndex: 1000,
        style: {
          stroke: edgeColor,
          strokeWidth: 2,
        },
        markerEnd: { type: MarkerType.ArrowClosed, color: edgeColor, width: 20, height: 20 },
      });
    });

    return { initialNodes: nodes, initialEdges: edges };
  }, [services, endpoints, projectID, isUnlocked, allForeignConsumers]);

  const [nodes, setNodes, onNodesChange] = useNodesState(initialNodes);
  const [edges, setEdges, onEdgesChange] = useEdgesState(initialEdges);

  useEffect(() => {
    setNodes(initialNodes);
    setEdges(initialEdges);
    if (!isUnlocked) {
      const t = setTimeout(() => fitView({ padding: 0.15 }), 100);
      return () => clearTimeout(t);
    }
  }, [initialNodes, initialEdges, setNodes, setEdges, fitView, isUnlocked]);

  const onNodeClick = useCallback((_: MouseEvent, node: Node) => {
    if (node.type === "groupBubble" || node.type === "networkSubgroup") return;
    setSelectedNode((prev) => (prev?.id === node.id ? null : node));
  }, []);

  const getNodeDetails = (node: Node) => {
    if (node.type === "service") {
      const s = node.data.service as Service;
      return (
        <div className="text-xs space-y-1">
          <div className="font-semibold text-sm mb-2">{s.name || "Unnamed Service"}</div>
          <div>
            <span className="text-theme-light">ID:</span> <span className="font-mono text-[10px]">{s.id}</span>
          </div>
          <div>
            <span className="text-theme-light">Status:</span> {s.status}
          </div>
          <div>
            <span className="text-theme-light">Health:</span> {s.health_status}
          </div>
          <div>
            <span className="text-theme-light">Visibility:</span> {s.visibility}
          </div>
          <div>
            <span className="text-theme-light">IPs:</span>{" "}
            {s.ip_addresses?.map((ip) => ip.replace(/\/32$/, "")).join(", ")}
          </div>
          <div>
            <span className="text-theme-light">Ports:</span> {s.ports?.join(", ")}
          </div>
          <div>
            <span className="text-theme-light">Protocol:</span> {s.protocol}
          </div>
          <div>
            <span className="text-theme-light">Network:</span>{" "}
            <span className="font-mono text-[10px]">{s.network_id}</span>
          </div>
        </div>
      );
    } else if (node.type === "endpoint") {
      const e = node.data.endpoint as Endpoint;
      const isForeign = e.project_id && e.project_id !== projectID;
      return (
        <div className="text-xs space-y-1">
          <div className="font-semibold text-sm mb-2">{e.name || "Unnamed Endpoint"}</div>
          <div>
            <span className="text-theme-light">ID:</span> <span className="font-mono text-[10px]">{e.id}</span>
          </div>
          <div>
            <span className="text-theme-light">Status:</span> {e.status}
          </div>
          {isForeign && (
            <div>
              <span className="text-theme-light">Project:</span>{" "}
              <span className="font-mono text-[10px]">{e.project_id}</span>
            </div>
          )}
          {e.ip_address && (
            <div>
              <span className="text-theme-light">IP:</span> {e.ip_address}
            </div>
          )}
          {(e.target.network || e.target.subnet || e.target.port) && (
            <div>
              <span className="text-theme-light">Target:</span>{" "}
              <span className="font-mono text-[10px]">{e.target.network || e.target.subnet || e.target.port}</span>
            </div>
          )}
          <div>
            <span className="text-theme-light">Service:</span>{" "}
            <span className="font-mono text-[10px]">{e.service_id}</span>
          </div>
        </div>
      );
    }
    return null;
  };

  if (servicesLoading || endpointsLoading) {
    return <LoadingIndicator className="m-auto" />;
  }

  if (servicesError || endpointsError) {
    return <Message variant="danger">{servicesError?.message || endpointsError?.message}</Message>;
  }

  if (services.length === 0 && endpoints.length === 0) {
    return <Message variant="info">No services or endpoints to display.</Message>;
  }

  return (
    <Stack direction="vertical" gap="4" className="h-full">
      <div className="flex items-center gap-4 text-sm flex-wrap">
        <div className="flex items-center gap-2">
          <div
            className="w-5 h-5 rounded border-2 border-dashed"
            style={{ backgroundColor: "rgba(168, 85, 247, 0.15)", borderColor: "rgb(168, 85, 247)" }}
          />
          <span>Managed</span>
        </div>
        <div className="flex items-center gap-2">
          <div
            className="w-5 h-5 rounded border-2 border-dashed"
            style={{
              backgroundColor: "color-mix(in srgb, var(--color-success) 10%, transparent)",
              borderColor: "var(--color-success)",
            }}
          />
          <span>Own Project</span>
        </div>
        <div className="flex items-center gap-2">
          <div
            className="w-5 h-5 rounded border-2 border-dashed"
            style={{ backgroundColor: "var(--color-background-lvl-3)", borderColor: "var(--color-text-light)" }}
          />
          <span>External</span>
        </div>
        <div className="border-l border-theme-background-lvl-4 h-4" />
        <div className="flex items-center gap-2">
          <div className="w-4 h-4 rounded" style={{ backgroundColor: "#16a34a" }} />
          <span>Service</span>
        </div>
        <div className="flex items-center gap-2">
          <div className="w-4 h-4 rounded-full" style={{ backgroundColor: "#0d9488" }} />
          <span>Endpoint</span>
        </div>
        <div className="border-l border-theme-background-lvl-4 h-4" />
        <div className="flex items-center gap-1">
          <div className="w-2.5 h-2.5 rounded-full" style={{ backgroundColor: "var(--color-success)" }} />
          <span className="text-xs">OK</span>
        </div>
        <div className="flex items-center gap-1">
          <div className="w-2.5 h-2.5 rounded-full" style={{ backgroundColor: "var(--color-warning)" }} />
          <span className="text-xs">Pending</span>
        </div>
        <div className="flex items-center gap-1">
          <div className="w-2.5 h-2.5 rounded-full" style={{ backgroundColor: "var(--color-danger)" }} />
          <span className="text-xs">Error</span>
        </div>
      </div>

      <div className="flex gap-4 flex-1" style={{ minHeight: "500px" }}>
        <div
          className="flex-1 border border-theme-background-lvl-4 rounded overflow-hidden"
          style={{
            background: isDark
              ? "radial-gradient(ellipse at center, #172033 0%, #0f172a 50%, #030712 100%)"
              : undefined,
          }}
        >
          <ReactFlow
            nodes={nodes}
            edges={edges}
            onNodesChange={onNodesChange}
            onEdgesChange={onEdgesChange}
            onNodeClick={onNodeClick}
            nodeTypes={nodeTypes}
            colorMode={isDark ? "dark" : "light"}
            style={{ background: isDark ? "transparent" : "#f9fafb" }}
            fitView
            fitViewOptions={{ padding: 0.15 }}
            minZoom={0.2}
            maxZoom={2}
            connectionLineType={ConnectionLineType.SmoothStep}
            defaultEdgeOptions={{ type: "smoothstep" }}
            nodesDraggable={isUnlocked}
            nodesConnectable={false}
            elementsSelectable={false}
            panOnDrag
            zoomOnScroll
          >
            <Background
              variant={BackgroundVariant.Dots}
              color={isDark ? "#374151" : "#9ca3af"}
              gap={24}
              size={2}
              bgColor={isDark ? "transparent" : undefined}
            />
            <Controls showInteractive={true} onInteractiveChange={setIsUnlocked} />
            <MiniMap
              nodeColor={(node) => {
                if (node.type === "groupBubble" || node.type === "networkSubgroup") return "transparent";
                if (node.type === "service") {
                  if (node.data.isManaged) return "#7c3aed";
                  return node.data.isExternal ? (isDark ? "#6b7280" : "#9ca3af") : "#16a34a";
                }
                if (node.type === "endpoint") return "#0d9488";
                return isDark ? "#374151" : "#9ca3af";
              }}
              maskColor={isDark ? "rgba(0, 0, 0, 0.8)" : "rgba(255, 255, 255, 0.8)"}
              style={{ backgroundColor: isDark ? "#1f2937" : "#f3f4f6" }}
            />
          </ReactFlow>
        </div>

        {selectedNode && (
          <div className="w-72 p-4 border border-theme-background-lvl-4 rounded bg-theme-background-lvl-2 flex-shrink-0 overflow-auto">
            <div className="flex items-center justify-between mb-3">
              <span className="font-semibold text-sm">
                {selectedNode.type === "service" ? "Service" : "Endpoint"} Details
              </span>
              <button
                onClick={() => setSelectedNode(null)}
                className="text-theme-light hover:text-white text-lg leading-none"
              >
                ×
              </button>
            </div>
            {getNodeDetails(selectedNode)}
          </div>
        )}
      </div>
    </Stack>
  );
};

export const NetworkGraph = () => {
  return (
    <ReactFlowProvider>
      <NetworkGraphInner />
    </ReactFlowProvider>
  );
};
