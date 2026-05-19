// SPDX-FileCopyrightText: 2026 SAP SE
// SPDX-License-Identifier: Apache-2.0

import { useState, useMemo } from "react";
import {
  DataGrid,
  DataGridRow,
  DataGridCell,
  DataGridHeadCell,
  Stack,
  LoadingIndicator,
  Message,
  Icon,
  Tooltip,
  TooltipTrigger,
  TooltipContent,
  TextInput,
  Select,
  SelectOption,
  Button,
} from "@cloudoperators/juno-ui-components";
import { useAgents } from "../api";
import type { Agent, Provider } from "../types";

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

const formatDate = (dateStr: string | null | undefined): string => {
  if (!dateStr) return "-";
  const date = new Date(dateStr);
  return date.toLocaleString();
};

const HeartbeatStatus = ({ heartbeatAt }: { heartbeatAt: string | null | undefined }) => {
  if (!heartbeatAt) return <span className="text-theme-light">-</span>;

  const heartbeat = new Date(heartbeatAt);
  const now = new Date();
  const diffMs = now.getTime() - heartbeat.getTime();
  const diffSeconds = diffMs / 1000;

  const isRecent = diffSeconds < 30;
  const isStale = diffSeconds >= 30 && diffSeconds < 5 * 60;
  const isOld = diffSeconds >= 5 * 60;

  const formatDiff = () => {
    if (diffSeconds < 60) return `${Math.floor(diffSeconds)} sec ago`;
    return `${Math.floor(diffSeconds / 60)} min ago`;
  };

  return (
    <Tooltip>
      <TooltipTrigger>
        <Icon
          icon={isRecent ? "checkCircle" : isOld ? "dangerous" : "warning"}
          size="18"
          className={isRecent ? "text-green-500" : isOld ? "text-red-500" : "text-yellow-500"}
        />
      </TooltipTrigger>
      <TooltipContent>
        Last heartbeat: {formatDate(heartbeatAt)}
        {!isRecent && ` (${formatDiff()})`}
      </TooltipContent>
    </Tooltip>
  );
};

type SortField = "host" | "availability_zone" | "provider" | "enabled" | "services";
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

export const AgentList = () => {
  const { data, isLoading, error } = useAgents();

  // Filter state
  const [hostFilter, setHostFilter] = useState("");
  const [providerFilter, setProviderFilter] = useState<string>("");
  const [enabledFilter, setEnabledFilter] = useState<string>("");

  // Sort state
  const [sortField, setSortField] = useState<SortField | null>(null);
  const [sortDir, setSortDir] = useState<SortDir>("asc");

  const agents = data?.items ?? [];

  const handleSort = (field: SortField) => {
    if (sortField === field) {
      setSortDir(sortDir === "asc" ? "desc" : "asc");
    } else {
      setSortField(field);
      setSortDir("asc");
    }
  };

  const filteredAndSortedAgents = useMemo(() => {
    let result = agents.filter((a) => {
      if (hostFilter && !a.host?.toLowerCase().includes(hostFilter.toLowerCase())) {
        return false;
      }
      if (providerFilter && a.provider !== providerFilter) return false;
      if (enabledFilter && (a.enabled ? "yes" : "no") !== enabledFilter) return false;
      return true;
    });

    if (sortField) {
      result = [...result].sort((a, b) => {
        let aVal: string | boolean | number | null = "";
        let bVal: string | boolean | number | null = "";

        switch (sortField) {
          case "host":
            aVal = a.host || "";
            bVal = b.host || "";
            break;
          case "availability_zone":
            aVal = a.availability_zone || "";
            bVal = b.availability_zone || "";
            break;
          case "provider":
            aVal = a.provider || "";
            bVal = b.provider || "";
            break;
          case "enabled":
            aVal = a.enabled;
            bVal = b.enabled;
            break;
          case "services":
            aVal = a.services ?? 0;
            bVal = b.services ?? 0;
            break;
        }

        if (typeof aVal === "boolean") {
          return sortDir === "asc" ? (aVal === bVal ? 0 : aVal ? -1 : 1) : aVal === bVal ? 0 : aVal ? 1 : -1;
        }

        if (typeof aVal === "number" && typeof bVal === "number") {
          return sortDir === "asc" ? aVal - bVal : bVal - aVal;
        }

        const cmp = String(aVal || "").localeCompare(String(bVal || ""));
        return sortDir === "asc" ? cmp : -cmp;
      });
    }

    return result;
  }, [agents, hostFilter, providerFilter, enabledFilter, sortField, sortDir]);

  const clearFilters = () => {
    setHostFilter("");
    setProviderFilter("");
    setEnabledFilter("");
  };

  const hasFilters = hostFilter || providerFilter || enabledFilter;

  if (isLoading) return <LoadingIndicator className="m-auto" />;
  if (error) return <Message variant="danger">{error.message}</Message>;

  return (
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
        <div style={{ display: "flex", flexDirection: "row", alignItems: "center", gap: "0.75rem", flexWrap: "wrap" }}>
          <div style={{ width: "14rem" }}>
            <TextInput
              placeholder="Filter by host..."
              value={hostFilter}
              onChange={(e) => setHostFilter(e.target.value)}
            />
          </div>
          <div style={{ width: "9rem" }}>
            <Select value={providerFilter} onChange={(v) => setProviderFilter(String(v ?? ""))} placeholder="Provider">
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
      </div>

      <DataGrid columns={7}>
        <DataGridRow>
          <SortableHeader label="Host" field="host" sortField={sortField} sortDir={sortDir} onSort={handleSort} />
          <SortableHeader
            label="Availability Zone"
            field="availability_zone"
            sortField={sortField}
            sortDir={sortDir}
            onSort={handleSort}
          />
          <SortableHeader
            label="Provider"
            field="provider"
            sortField={sortField}
            sortDir={sortDir}
            onSort={handleSort}
          />
          <DataGridHeadCell>Physnet</DataGridHeadCell>
          <SortableHeader
            label="Services"
            field="services"
            sortField={sortField}
            sortDir={sortDir}
            onSort={handleSort}
          />
          <DataGridHeadCell>Heartbeat</DataGridHeadCell>
          <SortableHeader label="Enabled" field="enabled" sortField={sortField} sortDir={sortDir} onSort={handleSort} />
        </DataGridRow>
        {filteredAndSortedAgents.map((a) => (
          <DataGridRow key={a.host} className={!a.enabled ? "bg-theme-background-lvl-3 opacity-60" : ""}>
            <DataGridCell>
              <span className="font-semibold">{a.host}</span>
            </DataGridCell>
            <DataGridCell>{a.availability_zone || "-"}</DataGridCell>
            <DataGridCell>
              <ProviderLabel provider={a.provider} />
            </DataGridCell>
            <DataGridCell>{a.physnet || "-"}</DataGridCell>
            <DataGridCell>{a.services ?? 0}</DataGridCell>
            <DataGridCell>
              <HeartbeatStatus heartbeatAt={a.heartbeat_at} />
            </DataGridCell>
            <DataGridCell>
              <EnabledIcon enabled={a.enabled} />
            </DataGridCell>
          </DataGridRow>
        ))}
      </DataGrid>

      {filteredAndSortedAgents.length === 0 && agents.length > 0 && (
        <Message variant="info">No agents match your filters.</Message>
      )}
      {agents.length === 0 && <Message variant="info">No agents found.</Message>}
    </Stack>
  );
};
