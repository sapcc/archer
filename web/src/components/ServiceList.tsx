// SPDX-FileCopyrightText: 2026 SAP SE
// SPDX-License-Identifier: Apache-2.0

import { useState, useMemo } from "react";
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
  Tooltip,
  TooltipTrigger,
  TooltipContent,
  Badge,
  TextInput,
  Select,
  SelectOption,
} from "@cloudoperators/juno-ui-components";
import { Routes, Route, useNavigate, useLocation } from "react-router";
import { useServices, useDeleteService } from "../api";
import { useStore } from "../store";
import { useStatusChangeToast } from "../hooks/useStatusChangeToast";
import { StatusBadge } from "./StatusBadge";
import { DeleteModal } from "./DeleteModal";
import { ServiceDetail } from "./ServiceDetail";
import { ServiceForm } from "./ServiceForm";
import { EndpointForm } from "./EndpointForm";
import { MigrateModal } from "./MigrateModal";
import type { Service, Provider } from "../types";

const ProviderLabel = ({ provider }: { provider: Provider | null }) => {
  const isCP = provider === "cp";
  return <span>{isCP ? "Control Plane" : "Tenant"}</span>;
};

const EnabledIcon = ({ enabled }: { enabled: boolean }) => (
  <Tooltip>
    <TooltipTrigger>
      <Icon
        icon={enabled ? "checkCircle" : "cancel"}
        size="18"
        className={enabled ? "text-green-500" : "text-red-500"}
      />
    </TooltipTrigger>
    <TooltipContent>{enabled ? "Enabled" : "Disabled"}</TooltipContent>
  </Tooltip>
);

const formatPorts = (ports: number[] | null | undefined): string => {
  if (!ports || ports.length === 0) return "-";

  const sorted = [...ports].sort((a, b) => a - b);
  const ranges: string[] = [];
  let start = sorted[0];
  let end = sorted[0];

  for (let i = 1; i <= sorted.length; i++) {
    if (i < sorted.length && sorted[i] === end + 1) {
      end = sorted[i];
    } else {
      ranges.push(start === end ? String(start) : `${start}-${end}`);
      if (i < sorted.length) {
        start = sorted[i];
        end = sorted[i];
      }
    }
  }

  return ranges.join(", ");
};

type SortField = "name" | "status" | "health_status" | "visibility" | "enabled" | "provider";
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

export const ServiceList = ({ canEdit, cloudAdmin }: { canEdit: boolean; cloudAdmin: boolean }) => {
  const navigate = useNavigate();
  const location = useLocation();
  const { data, isLoading, error } = useServices();
  const deleteService = useDeleteService();
  const showServiceModal = useStore((s) => s.showServiceModal);
  const showEndpointModal = useStore((s) => s.showEndpointModal);
  const showDeleteModal = useStore((s) => s.showDeleteModal);
  const showMigrateModal = useStore((s) => s.showMigrateModal);
  const deleteTarget = useStore((s) => s.deleteTarget);
  const migrateService = useStore((s) => s.migrateService);
  const openServiceModal = useStore((s) => s.openServiceModal);
  const openEndpointModal = useStore((s) => s.openEndpointModal);
  const openDeleteModal = useStore((s) => s.openDeleteModal);
  const openMigrateModal = useStore((s) => s.openMigrateModal);
  const closeModals = useStore((s) => s.closeModals);
  const projectID = useStore((s) => s.globalAPI.projectID);

  const handleRowClick = (id: string) => {
    const currentPath = location.pathname;
    if (currentPath === `/services/${id}`) {
      navigate("/services");
    } else {
      navigate(`/services/${id}`);
    }
  };

  // Filter state
  const [nameFilter, setNameFilter] = useState("");
  const [visibilityFilter, setVisibilityFilter] = useState<string>("");
  const [providerFilter, setProviderFilter] = useState<string>("");
  const [enabledFilter, setEnabledFilter] = useState<string>("");
  const [cascadeDelete, setCascadeDelete] = useState(false);

  // Sort state
  const [sortField, setSortField] = useState<SortField | null>(null);
  const [sortDir, setSortDir] = useState<SortDir>("asc");

  const services = data?.items ?? [];

  useStatusChangeToast(services, "service");

  const handleSort = (field: SortField) => {
    if (sortField === field) {
      setSortDir(sortDir === "asc" ? "desc" : "asc");
    } else {
      setSortField(field);
      setSortDir("asc");
    }
  };

  const filteredAndSortedServices = useMemo(() => {
    let result = services.filter((s) => {
      if (
        nameFilter &&
        !(
          s.name?.toLowerCase().includes(nameFilter.toLowerCase()) ||
          s.id.toLowerCase().includes(nameFilter.toLowerCase())
        )
      ) {
        return false;
      }
      if (visibilityFilter && s.visibility !== visibilityFilter) return false;
      if (providerFilter && s.provider !== providerFilter) return false;
      if (enabledFilter && (s.enabled ? "yes" : "no") !== enabledFilter) return false;
      return true;
    });

    if (sortField) {
      result = [...result].sort((a, b) => {
        let aVal: string | boolean | null = "";
        let bVal: string | boolean | null = "";

        switch (sortField) {
          case "name":
            aVal = a.name || "";
            bVal = b.name || "";
            break;
          case "status":
            aVal = a.status;
            bVal = b.status;
            break;
          case "health_status":
            aVal = a.health_status;
            bVal = b.health_status;
            break;
          case "visibility":
            aVal = a.visibility;
            bVal = b.visibility;
            break;
          case "provider":
            aVal = a.provider;
            bVal = b.provider;
            break;
          case "enabled":
            aVal = a.enabled;
            bVal = b.enabled;
            break;
        }

        if (typeof aVal === "boolean") {
          return sortDir === "asc" ? (aVal === bVal ? 0 : aVal ? -1 : 1) : aVal === bVal ? 0 : aVal ? 1 : -1;
        }

        const cmp = String(aVal || "").localeCompare(String(bVal || ""));
        return sortDir === "asc" ? cmp : -cmp;
      });
    }

    return result;
  }, [services, nameFilter, visibilityFilter, providerFilter, enabledFilter, sortField, sortDir]);

  const handleDelete = async (cascade?: boolean) => {
    if (deleteTarget?.type === "service") {
      try {
        await deleteService.mutateAsync({ id: deleteTarget.item.id, cascade });
        closeModals();
        setCascadeDelete(false);
      } catch {
        // Error is captured in deleteService.error, displayed by DeleteModal
      }
    }
  };

  const clearFilters = () => {
    setNameFilter("");
    setVisibilityFilter("");
    setProviderFilter("");
    setEnabledFilter("");
  };

  const hasFilters = nameFilter || visibilityFilter || providerFilter || enabledFilter;

  if (isLoading) return <LoadingIndicator className="m-auto" />;
  if (error) return <Message variant="danger">{error.message}</Message>;

  return (
    <>
      <Stack direction="vertical" gap="4">
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
            <div style={{ width: "8rem" }}>
              <Select
                value={visibilityFilter}
                onChange={(v) => setVisibilityFilter(String(v ?? ""))}
                placeholder="Visibility"
              >
                <SelectOption value="private">Private</SelectOption>
                <SelectOption value="public">Public</SelectOption>
              </Select>
            </div>
            <div style={{ width: "9rem" }}>
              <Select
                value={providerFilter}
                onChange={(v) => setProviderFilter(String(v ?? ""))}
                placeholder="Provider"
              >
                <SelectOption value="tenant">Tenant</SelectOption>
                <SelectOption value="cp">Control Plane</SelectOption>
              </Select>
            </div>
            <div style={{ width: "8rem" }}>
              <Select value={enabledFilter} onChange={(v) => setEnabledFilter(String(v ?? ""))} placeholder="Enabled">
                <SelectOption value="yes">Yes</SelectOption>
                <SelectOption value="no">No</SelectOption>
              </Select>
            </div>
            {hasFilters && (
              <Button variant="subdued" onClick={clearFilters}>
                Clear Filters
              </Button>
            )}
          </div>
          {canEdit && (
            <Stack gap="2">
              {cloudAdmin ? (
                <Button icon="addCircle" variant="primary-danger" onClick={() => openServiceModal(null, "cp")}>
                  Create CP Service
                </Button>
              ) : (
                <Button icon="addCircle" onClick={() => openServiceModal()}>
                  Create Service
                </Button>
              )}
            </Stack>
          )}
        </div>

        <DataGrid columns={canEdit ? 8 : 7}>
          <DataGridRow>
            <SortableHeader label="Name" field="name" sortField={sortField} sortDir={sortDir} onSort={handleSort} />
            <SortableHeader
              label="Provider"
              field="provider"
              sortField={sortField}
              sortDir={sortDir}
              onSort={handleSort}
            />
            <SortableHeader label="Status" field="status" sortField={sortField} sortDir={sortDir} onSort={handleSort} />
            <SortableHeader
              label="Health"
              field="health_status"
              sortField={sortField}
              sortDir={sortDir}
              onSort={handleSort}
            />
            <SortableHeader
              label="Visibility"
              field="visibility"
              sortField={sortField}
              sortDir={sortDir}
              onSort={handleSort}
            />
            <DataGridHeadCell>Ports</DataGridHeadCell>
            <SortableHeader
              label="Enabled"
              field="enabled"
              sortField={sortField}
              sortDir={sortDir}
              onSort={handleSort}
            />
            {canEdit && <DataGridHeadCell>Actions</DataGridHeadCell>}
          </DataGridRow>
          {filteredAndSortedServices.map((s) => {
            const isExternal = projectID && s.project_id !== projectID;
            return (
              <DataGridRow
                key={s.id}
                onClick={() => handleRowClick(s.id)}
                className={`cursor-pointer hover:bg-theme-background-lvl-2 ${!s.enabled ? "bg-theme-background-lvl-3 opacity-60" : ""} ${isExternal ? "border-l-4 border-l-theme-warning" : ""}`}
              >
                <DataGridCell>
                  <div className="flex flex-col max-w-72">
                    <div className="flex items-center gap-2">
                      <span className="font-semibold truncate" title={s.name || "Unnamed"}>
                        {s.name || "Unnamed"}
                      </span>
                      {isExternal && (
                        <Tooltip>
                          <TooltipTrigger asChild>
                            <Badge text="External" />
                          </TooltipTrigger>
                          <TooltipContent>Service belongs to project {s.project_id}</TooltipContent>
                        </Tooltip>
                      )}
                    </div>
                    <span className="text-xs text-theme-light">{s.id}</span>
                  </div>
                </DataGridCell>
                <DataGridCell>
                  <ProviderLabel provider={s.provider} />
                </DataGridCell>
                <DataGridCell>
                  <StatusBadge status={s.status} />
                </DataGridCell>
                <DataGridCell>
                  <StatusBadge status={s.health_status} />
                </DataGridCell>
                <DataGridCell>{s.visibility}</DataGridCell>
                <DataGridCell>
                  {(() => {
                    const portsStr = formatPorts(s.ports);
                    if (portsStr.length > 30) {
                      return (
                        <Tooltip>
                          <TooltipTrigger>
                            <span className="truncate max-w-32 block">{portsStr}</span>
                          </TooltipTrigger>
                          <TooltipContent>{portsStr}</TooltipContent>
                        </Tooltip>
                      );
                    }
                    return portsStr;
                  })()}
                </DataGridCell>
                <DataGridCell>
                  <EnabledIcon enabled={s.enabled} />
                </DataGridCell>
                {canEdit && (
                  <DataGridCell>
                    <Stack gap="1">
                      {!cloudAdmin && (
                        <Button
                          size="small"
                          icon="place"
                          onClick={(e) => {
                            e.stopPropagation();
                            openEndpointModal(null, s.id);
                          }}
                          title="Create Endpoint"
                        />
                      )}
                      {(!isExternal || cloudAdmin) && (
                        <>
                          <Button
                            size="small"
                            icon="edit"
                            onClick={(e) => {
                              e.stopPropagation();
                              openServiceModal(s);
                            }}
                          />
                          <Button
                            size="small"
                            icon="deleteForever"
                            variant="primary-danger"
                            onClick={(e) => {
                              e.stopPropagation();
                              openDeleteModal({ type: "service", item: s });
                            }}
                          />
                        </>
                      )}
                      {cloudAdmin && (
                        <Button
                          size="small"
                          icon="upload"
                          variant="primary-danger"
                          onClick={(e) => {
                            e.stopPropagation();
                            openMigrateModal(s);
                          }}
                          title="Migrate Service"
                        />
                      )}
                    </Stack>
                  </DataGridCell>
                )}
              </DataGridRow>
            );
          })}
        </DataGrid>

        {filteredAndSortedServices.length === 0 && services.length > 0 && (
          <Message variant="info">No services match your filters.</Message>
        )}
        {services.length === 0 && <Message variant="info">No services found.</Message>}
      </Stack>

      <Routes>
        <Route path=":serviceId/*" element={<ServiceDetail canEdit={canEdit} />} />
      </Routes>

      {showServiceModal && <ServiceForm />}
      {showEndpointModal && <EndpointForm />}
      {showDeleteModal && deleteTarget?.type === "service" && (
        <DeleteModal
          onConfirm={handleDelete}
          isLoading={deleteService.isPending}
          error={deleteService.error}
          cascade={cascadeDelete}
          onCascadeChange={setCascadeDelete}
        />
      )}
      {showMigrateModal && migrateService && <MigrateModal service={migrateService} onClose={closeModals} />}
    </>
  );
};
