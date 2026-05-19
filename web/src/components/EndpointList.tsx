// SPDX-FileCopyrightText: 2026 SAP SE
// SPDX-License-Identifier: Apache-2.0

import { useState, useMemo, useEffect, useRef } from "react";
import {
  DataGrid,
  DataGridRow,
  DataGridCell,
  DataGridHeadCell,
  Button,
  Stack,
  LoadingIndicator,
  Message,
  Icon,
  TextInput,
  Select,
  SelectOption,
  Badge,
  Tooltip,
  TooltipTrigger,
  TooltipContent,
} from "@cloudoperators/juno-ui-components";
import { Routes, Route, useNavigate, useLocation } from "react-router";
import {
  useEndpoints,
  useDeleteEndpoint,
  useServices,
  useServiceConsumers,
  useAcceptEndpoints,
  useRejectEndpoints,
} from "../api";
import { useStore } from "../store";
import { useStatusChangeToast } from "../hooks/useStatusChangeToast";
import { StatusBadge } from "./StatusBadge";
import { DeleteModal } from "./DeleteModal";
import { EndpointDetail } from "./EndpointDetail";
import { EndpointForm } from "./EndpointForm";
import type { Service, EndpointConsumer } from "../types";

type PendingWithService = EndpointConsumer & { service: Service };

type SortField = "name" | "status" | "ip_address" | "service_id";
type SortDir = "asc" | "desc";

const SortableHeader = ({
  label,
  field,
  sortField,
  sortDir,
  onSort,
}: {
  label: string;
  field: SortField;
  sortField: SortField | null;
  sortDir: SortDir;
  onSort: (field: SortField) => void;
}) => (
  <DataGridHeadCell>
    <button className="flex items-center gap-1 hover:text-theme-accent cursor-pointer" onClick={() => onSort(field)}>
      {label}
      {sortField === field && <Icon icon={sortDir === "asc" ? "expandLess" : "expandMore"} size="16" />}
    </button>
  </DataGridHeadCell>
);

export const EndpointList = ({ canEdit, cloudAdmin }: { canEdit: boolean; cloudAdmin: boolean }) => {
  const navigate = useNavigate();
  const location = useLocation();
  const { data, isLoading, error } = useEndpoints();
  const { data: servicesData, isLoading: servicesLoading } = useServices();
  const del = useDeleteEndpoint();
  const accept = useAcceptEndpoints();
  const reject = useRejectEndpoints();

  const showEndpointModal = useStore((s) => s.showEndpointModal);
  const showDeleteModal = useStore((s) => s.showDeleteModal);
  const deleteTarget = useStore((s) => s.deleteTarget);
  const openEndpointModal = useStore((s) => s.openEndpointModal);
  const openDeleteModal = useStore((s) => s.openDeleteModal);
  const closeModals = useStore((s) => s.closeModals);
  const projectID = useStore((s) => s.globalAPI.projectID);
  const addToast = useStore((s) => s.addToast);

  const [actionError, setActionError] = useState<string | null>(null);
  const [pendingExpanded, setPendingExpanded] = useState(false);

  // Create service name lookup and endpoint name lookup
  const services = useMemo(() => servicesData?.items ?? [], [servicesData]);
  const serviceNameMap = useMemo(() => {
    const map = new Map<string, string>();
    services.forEach((s) => {
      map.set(s.id, s.name || "Unnamed");
    });
    return map;
  }, [services]);

  const endpoints = data?.items ?? [];
  useStatusChangeToast(endpoints, "endpoint");
  const endpointNameMap = useMemo(() => {
    const map = new Map<string, string | null>();
    endpoints.forEach((e) => map.set(e.id, e.name));
    return map;
  }, [endpoints]);

  // Batch consumer fetches for all services (collapses N+1)
  const serviceIds = useMemo(() => services.map((s) => s.id), [services]);
  const consumerQueries = useServiceConsumers(serviceIds);

  // Stable signature so dependent memos/effects don't fire on every render.
  // useQueries returns a fresh array reference every render even when data is unchanged.
  const consumersSignature = consumerQueries
    .map((q) => (q.data?.items ?? []).map((c) => `${c.id}:${c.status}`).join(","))
    .join("|");

  const allPending = useMemo<PendingWithService[]>(() => {
    const out: PendingWithService[] = [];
    services.forEach((service, idx) => {
      const items = consumerQueries[idx]?.data?.items ?? [];
      for (const c of items) {
        if (c.status === "PENDING_APPROVAL") {
          out.push({ ...c, service });
        }
      }
    });
    return out;
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [services, consumersSignature]);

  // Track consumer status changes for toasts
  const consumerStatusRef = useRef<Map<string, string>>(new Map());
  useEffect(() => {
    const tracked = consumerStatusRef.current;
    services.forEach((_service, idx) => {
      const items = consumerQueries[idx]?.data?.items ?? [];
      for (const consumer of items) {
        const prev = tracked.get(consumer.id);
        const curr = consumer.status;
        if (prev && prev !== curr) {
          const name = endpointNameMap.get(consumer.id) || "Endpoint";
          if (prev === "PENDING_APPROVAL" && curr === "AVAILABLE") {
            addToast({ variant: "success", message: `${name} was accepted` });
          } else if (prev === "PENDING_APPROVAL" && curr === "REJECTED") {
            addToast({ variant: "warning", message: `${name} was rejected` });
          } else if (curr === "FAILED") {
            addToast({ variant: "danger", message: `${name} failed` });
          }
        }
        tracked.set(consumer.id, curr);
      }
    });
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [services, consumersSignature, endpointNameMap, addToast]);

  const handleRowClick = (id: string) => {
    const currentPath = location.pathname;
    if (currentPath === `/endpoints/${id}`) {
      navigate("/endpoints");
    } else {
      navigate(`/endpoints/${id}`);
    }
  };

  // Pending approval handlers
  const handleAccept = async (serviceId: string, endpointIds: string[]) => {
    try {
      setActionError(null);
      await accept.mutateAsync({ serviceId, endpointIds });
    } catch (e) {
      setActionError(e instanceof Error ? e.message : "Failed to accept");
    }
  };

  const handleReject = async (serviceId: string, endpointIds: string[]) => {
    try {
      setActionError(null);
      await reject.mutateAsync({ serviceId, endpointIds });
    } catch (e) {
      setActionError(e instanceof Error ? e.message : "Failed to reject");
    }
  };

  const handleAcceptAll = async () => {
    setActionError(null);
    const byService = new Map<string, string[]>();
    for (const p of allPending) {
      const ids = byService.get(p.service.id) ?? [];
      ids.push(p.id);
      byService.set(p.service.id, ids);
    }
    try {
      await Promise.all(
        Array.from(byService, ([serviceId, endpointIds]) => accept.mutateAsync({ serviceId, endpointIds }))
      );
    } catch (e) {
      setActionError(e instanceof Error ? e.message : "Failed to accept");
    }
  };

  const handleRejectAll = async () => {
    setActionError(null);
    const byService = new Map<string, string[]>();
    for (const p of allPending) {
      const ids = byService.get(p.service.id) ?? [];
      ids.push(p.id);
      byService.set(p.service.id, ids);
    }
    try {
      await Promise.all(
        Array.from(byService, ([serviceId, endpointIds]) => reject.mutateAsync({ serviceId, endpointIds }))
      );
    } catch (e) {
      setActionError(e instanceof Error ? e.message : "Failed to reject");
    }
  };

  // Filter state
  const [nameFilter, setNameFilter] = useState("");
  const [statusFilter, setStatusFilter] = useState<string>("");

  // Sort state
  const [sortField, setSortField] = useState<SortField | null>(null);
  const [sortDir, setSortDir] = useState<SortDir>("asc");

  const handleSort = (field: SortField) => {
    if (sortField === field) {
      setSortDir(sortDir === "asc" ? "desc" : "asc");
    } else {
      setSortField(field);
      setSortDir("asc");
    }
  };

  const filteredAndSortedEndpoints = useMemo(() => {
    let result = endpoints.filter((e) => {
      if (
        nameFilter &&
        !(
          e.name?.toLowerCase().includes(nameFilter.toLowerCase()) ||
          e.id.toLowerCase().includes(nameFilter.toLowerCase())
        )
      ) {
        return false;
      }
      if (statusFilter && e.status !== statusFilter) return false;
      return true;
    });

    if (sortField) {
      result = [...result].sort((a, b) => {
        const aVal = a[sortField] || "";
        const bVal = b[sortField] || "";
        const cmp = String(aVal).localeCompare(String(bVal));
        return sortDir === "asc" ? cmp : -cmp;
      });
    }

    return result;
  }, [endpoints, nameFilter, statusFilter, sortField, sortDir]);

  const handleDelete = async () => {
    if (deleteTarget?.type === "endpoint") {
      try {
        await del.mutateAsync(deleteTarget.item.id);
        closeModals();
      } catch {
        // Error is captured in del.error, displayed by DeleteModal
      }
    }
  };

  const clearFilters = () => {
    setNameFilter("");
    setStatusFilter("");
  };

  const hasFilters = nameFilter || statusFilter;

  if (isLoading || servicesLoading) return <LoadingIndicator className="m-auto" />;
  if (error) return <Message variant="danger">{error.message}</Message>;

  return (
    <>
      <Stack direction="vertical" gap="4">
        {/* Pending Approvals Section - shown at top when there are pending approvals */}
        {allPending.length > 0 && (
          <div className="bg-theme-background-lvl-2 p-4 rounded-lg">
            <Stack direction="vertical" gap="4">
              <div className="flex items-center justify-between">
                <button
                  className="flex items-center gap-2 cursor-pointer hover:opacity-80"
                  onClick={() => setPendingExpanded(!pendingExpanded)}
                >
                  <Icon icon={pendingExpanded ? "expandMore" : "chevronRight"} size="24" />
                  <Icon icon="accessTime" size="24" className="text-theme-warning" />
                  <span className="font-semibold text-lg">Pending Approvals ({allPending.length})</span>
                  <span className="text-theme-light text-sm">
                    Endpoints from other projects requesting access to your services
                  </span>
                </button>
                {canEdit && pendingExpanded && (
                  <Stack gap="2">
                    <Button size="small" onClick={handleAcceptAll} disabled={accept.isPending || reject.isPending}>
                      Accept All
                    </Button>
                    <Button
                      size="small"
                      variant="primary-danger"
                      onClick={handleRejectAll}
                      disabled={accept.isPending || reject.isPending}
                    >
                      Reject All
                    </Button>
                  </Stack>
                )}
              </div>

              {pendingExpanded && (
                <>
                  {actionError && <Message variant="danger">{actionError}</Message>}

                  <DataGrid columns={canEdit ? 5 : 4}>
                    <DataGridRow>
                      <DataGridHeadCell>Service</DataGridHeadCell>
                      <DataGridHeadCell>Endpoint</DataGridHeadCell>
                      <DataGridHeadCell>Requesting Project</DataGridHeadCell>
                      <DataGridHeadCell>Status</DataGridHeadCell>
                      {canEdit && <DataGridHeadCell>Actions</DataGridHeadCell>}
                    </DataGridRow>
                    {allPending.map((p) => (
                      <DataGridRow key={p.id}>
                        <DataGridCell>
                          <div className="flex flex-col">
                            <span className="font-semibold">{p.service.name || "Unnamed"}</span>
                            <span className="text-xs text-theme-light">{p.service.id}</span>
                          </div>
                        </DataGridCell>
                        <DataGridCell>
                          <div className="flex flex-col">
                            <span className="font-semibold">{endpointNameMap.get(p.id) || "Unnamed"}</span>
                            <span className="text-xs text-theme-light">{p.id}</span>
                          </div>
                        </DataGridCell>
                        <DataGridCell>
                          <span className="text-xs text-theme-light">{p.project_id}</span>
                        </DataGridCell>
                        <DataGridCell>
                          <StatusBadge status={p.status} />
                        </DataGridCell>
                        {canEdit && (
                          <DataGridCell>
                            <Stack gap="1">
                              <Button
                                size="small"
                                onClick={() => handleAccept(p.service.id, [p.id])}
                                disabled={accept.isPending || reject.isPending}
                              >
                                Accept
                              </Button>
                              <Button
                                size="small"
                                variant="primary-danger"
                                onClick={() => handleReject(p.service.id, [p.id])}
                                disabled={accept.isPending || reject.isPending}
                              >
                                Reject
                              </Button>
                            </Stack>
                          </DataGridCell>
                        )}
                      </DataGridRow>
                    ))}
                  </DataGrid>
                </>
              )}
            </Stack>
          </div>
        )}

        {/* Regular Endpoints Section */}
        <div
          style={{
            display: "flex",
            flexDirection: "row",
            alignItems: "center",
            justifyContent: "space-between",
            gap: "1rem",
            flexWrap: "wrap",
          }}
        >
          <div
            style={{ display: "flex", flexDirection: "row", alignItems: "center", gap: "0.75rem", flexWrap: "wrap" }}
          >
            <div style={{ width: "14rem" }}>
              <TextInput
                placeholder="Filter by name or ID..."
                value={nameFilter}
                onChange={(e) => setNameFilter(e.target.value)}
              />
            </div>
            <div style={{ width: "10rem" }}>
              <Select value={statusFilter} onChange={(v) => setStatusFilter(String(v ?? ""))} placeholder="Status">
                <SelectOption value="AVAILABLE">Available</SelectOption>
                <SelectOption value="PENDING_APPROVAL">Pending Approval</SelectOption>
                <SelectOption value="PENDING_CREATE">Pending Create</SelectOption>
                <SelectOption value="PENDING_UPDATE">Pending Update</SelectOption>
                <SelectOption value="PENDING_DELETE">Pending Delete</SelectOption>
                <SelectOption value="REJECTED">Rejected</SelectOption>
                <SelectOption value="FAILED">Failed</SelectOption>
              </Select>
            </div>
            {hasFilters && (
              <Button variant="subdued" onClick={clearFilters}>
                Clear Filters
              </Button>
            )}
          </div>
          {canEdit && !cloudAdmin && (
            <Button icon="addCircle" onClick={() => openEndpointModal()}>
              Create Endpoint
            </Button>
          )}
        </div>

        <DataGrid columns={canEdit ? 6 : 5}>
          <DataGridRow>
            <SortableHeader label="Name" field="name" sortField={sortField} sortDir={sortDir} onSort={handleSort} />
            <SortableHeader
              label="Service"
              field="service_id"
              sortField={sortField}
              sortDir={sortDir}
              onSort={handleSort}
            />
            <SortableHeader label="Status" field="status" sortField={sortField} sortDir={sortDir} onSort={handleSort} />
            <SortableHeader label="IP" field="ip_address" sortField={sortField} sortDir={sortDir} onSort={handleSort} />
            <DataGridHeadCell>Tags</DataGridHeadCell>
            {canEdit && <DataGridHeadCell>Actions</DataGridHeadCell>}
          </DataGridRow>
          {filteredAndSortedEndpoints.map((e) => {
            const isExternal = projectID && e.project_id !== projectID;
            return (
              <DataGridRow
                key={e.id}
                onClick={() => handleRowClick(e.id)}
                className={`cursor-pointer hover:bg-theme-background-lvl-2 ${isExternal ? "border-l-4 border-l-theme-warning" : ""}`}
              >
                <DataGridCell>
                  <div className="flex flex-col max-w-72">
                    <div className="flex items-center gap-1">
                      <span className="font-semibold truncate" title={e.name || "Unnamed"}>
                        {e.name || "Unnamed"}
                      </span>
                      {isExternal && (
                        <Tooltip>
                          <TooltipTrigger>
                            <Icon icon="place" size="16" className="text-theme-warning" />
                          </TooltipTrigger>
                          <TooltipContent>External project: {e.project_id}</TooltipContent>
                        </Tooltip>
                      )}
                    </div>
                    <span className="text-xs text-theme-light">{e.id}</span>
                  </div>
                </DataGridCell>
                <DataGridCell>
                  <div className="flex flex-col max-w-48">
                    <span className="font-semibold truncate" title={serviceNameMap.get(e.service_id) || "Unknown"}>
                      {serviceNameMap.get(e.service_id) || "Unknown"}
                    </span>
                    <span className="text-xs text-theme-light">{e.service_id.slice(0, 8)}...</span>
                  </div>
                </DataGridCell>
                <DataGridCell>
                  <StatusBadge status={e.status} />
                </DataGridCell>
                <DataGridCell>{e.ip_address || "-"}</DataGridCell>
                <DataGridCell>
                  <div className="flex flex-wrap gap-1">
                    {e.tags?.length ? (
                      e.tags.map((tag) => <Badge key={tag} text={tag} />)
                    ) : (
                      <span className="text-theme-light">-</span>
                    )}
                  </div>
                </DataGridCell>
                {canEdit && (
                  <DataGridCell>
                    <Stack gap="1">
                      <Button
                        size="small"
                        icon="edit"
                        onClick={(ev) => {
                          ev.stopPropagation();
                          openEndpointModal(e);
                        }}
                      />
                      <Button
                        size="small"
                        icon="deleteForever"
                        variant="primary-danger"
                        onClick={(ev) => {
                          ev.stopPropagation();
                          openDeleteModal({ type: "endpoint", item: e });
                        }}
                      />
                    </Stack>
                  </DataGridCell>
                )}
              </DataGridRow>
            );
          })}
        </DataGrid>

        {filteredAndSortedEndpoints.length === 0 && endpoints.length > 0 && (
          <Message variant="info">No endpoints match your filters.</Message>
        )}
        {endpoints.length === 0 && <Message variant="info">No endpoints found.</Message>}
      </Stack>

      <Routes>
        <Route path=":endpointId" element={<EndpointDetail />} />
      </Routes>

      {showEndpointModal && <EndpointForm />}
      {showDeleteModal && deleteTarget?.type === "endpoint" && (
        <DeleteModal onConfirm={handleDelete} isLoading={del.isPending} error={del.error} />
      )}
    </>
  );
};
