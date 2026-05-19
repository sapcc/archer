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
  TextInput,
  Tooltip,
  TooltipTrigger,
  TooltipContent,
} from "@cloudoperators/juno-ui-components";
import { useRBACPolicies, useDeleteRBAC, useServices } from "../api";
import { useStore } from "../store";
import { DeleteModal } from "./DeleteModal";
import { RBACForm } from "./RBACForm";

type SortField = "id" | "service_id" | "target";
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

export const RBACList = ({ canEdit }: { canEdit: boolean }) => {
  const { data, isLoading, error } = useRBACPolicies();
  const { data: servicesData } = useServices();
  const del = useDeleteRBAC();
  const { showRBACModal, showDeleteModal, deleteTarget, openRBACModal, openDeleteModal, closeModals } = useStore();

  // Copy to clipboard state
  const [copiedId, setCopiedId] = useState<string | null>(null);

  const copyToClipboard = (id: string) => {
    navigator.clipboard.writeText(id);
    setCopiedId(id);
    setTimeout(() => setCopiedId(null), 2000);
  };

  // Create service name lookup
  const serviceNameMap = useMemo(() => {
    const map = new Map<string, string>();
    servicesData?.items?.forEach((s) => {
      map.set(s.id, s.name || "Unnamed");
    });
    return map;
  }, [servicesData]);

  // Filter state
  const [filter, setFilter] = useState("");

  // Sort state
  const [sortField, setSortField] = useState<SortField | null>(null);
  const [sortDir, setSortDir] = useState<SortDir>("asc");

  const policies = data?.items ?? [];

  const handleSort = (field: SortField) => {
    if (sortField === field) {
      setSortDir(sortDir === "asc" ? "desc" : "asc");
    } else {
      setSortField(field);
      setSortDir("asc");
    }
  };

  const filteredAndSortedPolicies = useMemo(() => {
    let result = policies.filter((p) => {
      if (
        filter &&
        !(
          p.id.toLowerCase().includes(filter.toLowerCase()) ||
          p.service_id.toLowerCase().includes(filter.toLowerCase()) ||
          p.target.toLowerCase().includes(filter.toLowerCase())
        )
      ) {
        return false;
      }
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
  }, [policies, filter, sortField, sortDir]);

  const handleDelete = async () => {
    if (deleteTarget?.type === "rbac") {
      try {
        await del.mutateAsync(deleteTarget.item.id);
        closeModals();
      } catch {
        // Error is captured in del.error, displayed by DeleteModal
      }
    }
  };

  const clearFilters = () => {
    setFilter("");
  };

  const hasFilters = filter;

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
                placeholder="Filter by ID, service, target..."
                value={filter}
                onChange={(e) => setFilter(e.target.value)}
              />
            </div>
            {hasFilters && (
              <Button variant="subdued" onClick={clearFilters}>
                Clear Filters
              </Button>
            )}
          </div>
          {canEdit && (
            <Button icon="addCircle" onClick={() => openRBACModal()}>
              Create RBAC Policy
            </Button>
          )}
        </div>

        <DataGrid columns={canEdit ? 5 : 4}>
          <DataGridRow>
            <SortableHeader label="ID" field="id" sortField={sortField} sortDir={sortDir} onSort={handleSort} />
            <SortableHeader
              label="Service"
              field="service_id"
              sortField={sortField}
              sortDir={sortDir}
              onSort={handleSort}
            />
            <DataGridHeadCell>Target Type</DataGridHeadCell>
            <SortableHeader label="Target" field="target" sortField={sortField} sortDir={sortDir} onSort={handleSort} />
            {canEdit && <DataGridHeadCell>Actions</DataGridHeadCell>}
          </DataGridRow>
          {filteredAndSortedPolicies.map((p) => (
            <DataGridRow key={p.id}>
              <DataGridCell>
                <div className="flex items-center gap-1">
                  <span>{p.id.slice(0, 16)}...</span>
                  <Tooltip>
                    <TooltipTrigger asChild>
                      <button onClick={() => copyToClipboard(p.id)} className="hover:text-theme-accent cursor-pointer">
                        <Icon icon={copiedId === p.id ? "checkCircle" : "contentCopy"} size="16" />
                      </button>
                    </TooltipTrigger>
                    <TooltipContent>{copiedId === p.id ? "Copied!" : p.id}</TooltipContent>
                  </Tooltip>
                </div>
              </DataGridCell>
              <DataGridCell>
                <div className="flex flex-col">
                  <span className="font-semibold">{serviceNameMap.get(p.service_id) || "Unknown"}</span>
                  <span className="text-xs text-theme-light">{p.service_id}</span>
                </div>
              </DataGridCell>
              <DataGridCell>{p.target_type}</DataGridCell>
              <DataGridCell>{p.target}</DataGridCell>
              {canEdit && (
                <DataGridCell>
                  <Stack gap="1">
                    <Button size="small" icon="edit" onClick={() => openRBACModal(p)} />
                    <Button
                      size="small"
                      icon="deleteForever"
                      variant="primary-danger"
                      onClick={() => openDeleteModal({ type: "rbac", item: p })}
                    />
                  </Stack>
                </DataGridCell>
              )}
            </DataGridRow>
          ))}
        </DataGrid>

        {filteredAndSortedPolicies.length === 0 && policies.length > 0 && (
          <Message variant="info">No RBAC policies match your filters.</Message>
        )}
        {policies.length === 0 && <Message variant="info">No RBAC policies.</Message>}
      </Stack>

      {showRBACModal && <RBACForm />}
      {showDeleteModal && deleteTarget?.type === "rbac" && (
        <DeleteModal onConfirm={handleDelete} isLoading={del.isPending} error={del.error} />
      )}
    </>
  );
};
