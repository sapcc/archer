// SPDX-FileCopyrightText: 2026 SAP SE
// SPDX-License-Identifier: Apache-2.0

import { useState, useMemo } from "react";
import {
  Panel,
  PanelBody,
  PanelFooter,
  Button,
  DataGrid,
  DataGridRow,
  DataGridCell,
  DataGridHeadCell,
  LoadingIndicator,
  Message,
  Stack,
  Tabs,
  Tab,
  TabList,
  TabPanel,
} from "@cloudoperators/juno-ui-components";
import { useParams, useNavigate } from "react-router";
import { useService, useServiceEndpoints, useEndpoints, useAcceptEndpoints, useRejectEndpoints } from "../api";
import { StatusBadge } from "./StatusBadge";

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

export const ServiceDetail = ({ canEdit }: { canEdit: boolean }) => {
  const { serviceId } = useParams<{ serviceId: string }>();
  const navigate = useNavigate();
  const [activeTab, setActiveTab] = useState(0);

  const { data: service, isLoading, error } = useService(serviceId);
  const { data: consumersData, isLoading: consumersLoading } = useServiceEndpoints(serviceId);
  const { data: endpointsData } = useEndpoints();
  const accept = useAcceptEndpoints();
  const reject = useRejectEndpoints();

  const consumers = consumersData?.items ?? [];
  const endpoints = endpointsData?.items ?? [];
  const close = () => navigate("/services");

  // Create a map of endpoint ID to endpoint name
  const endpointNameMap = useMemo(() => {
    const map = new Map<string, string | null>();
    endpoints.forEach((e) => map.set(e.id, e.name));
    return map;
  }, [endpoints]);

  const handleAcceptIds = async (ids: string[]) => {
    if (serviceId) {
      await accept.mutateAsync({ serviceId, endpointIds: ids });
    }
  };

  const handleRejectIds = async (ids: string[]) => {
    if (serviceId) {
      await reject.mutateAsync({ serviceId, endpointIds: ids });
    }
  };

  if (isLoading) {
    return (
      <Panel heading="Service Details" opened onClose={close}>
        <PanelBody>
          <LoadingIndicator className="m-auto" />
        </PanelBody>
      </Panel>
    );
  }

  if (error || !service) {
    return (
      <Panel heading="Service Details" opened onClose={close}>
        <PanelBody>
          <Message variant="danger">{error?.message || "Not found"}</Message>
        </PanelBody>
      </Panel>
    );
  }

  const fmt = (d: string | undefined) => (d ? new Date(d).toLocaleString() : "-");

  const panelHeading = service.name || "Service Details";

  return (
    <Panel heading={panelHeading} opened onClose={close}>
      <PanelBody>
        <Tabs selectedIndex={activeTab} onSelect={setActiveTab}>
          <TabList>
            <Tab>Details</Tab>
            <Tab>Consumers ({consumers.length})</Tab>
          </TabList>

          <TabPanel>
            <DataGrid columns={2} className="mt-4">
              {[
                ["ID", service.id],
                ["Name", service.name || "-"],
                ["Description", service.description || "-"],
                ["Provider", service.provider === "cp" ? "Control Plane" : "Tenant"],
                ["Host", service.host || "-"],
                ["Status", <StatusBadge key="s" status={service.status} />],
                ["Health", <StatusBadge key="h" status={service.health_status} />],
                ["Enabled", service.enabled ? "Yes" : "No"],
                ["Visibility", service.visibility],
                ["Require Approval", service.require_approval ? "Yes" : "No"],
                ["Protocol", service.protocol],
                ["Ports", formatPorts(service.ports)],
                ["IP Addresses", service.ip_addresses?.map((ip) => ip.replace(/\/32$/, "")).join(", ") || "-"],
                ["Network ID", service.network_id],
                ["Availability Zone", service.availability_zone || "-"],
                ["Proxy Protocol", service.proxy_protocol ? "Yes" : "No"],
                ["Connection Mirroring", service.connection_mirroring ? "Yes" : "No"],
                ["Tags", service.tags?.join(", ") || "-"],
                ["Project", service.project_id],
                ["Created", fmt(service.created_at)],
                ["Updated", fmt(service.updated_at)],
              ].map(([label, value], i) => (
                <DataGridRow key={i}>
                  <DataGridCell className="font-semibold">{label}</DataGridCell>
                  <DataGridCell>{value}</DataGridCell>
                </DataGridRow>
              ))}
            </DataGrid>
          </TabPanel>

          <TabPanel>
            {consumersLoading ? (
              <LoadingIndicator className="m-auto mt-4" />
            ) : (
              <Stack direction="vertical" gap="4" className="mt-4">
                {consumers.length > 0 && (
                  <DataGrid columns={canEdit ? 5 : 3}>
                    <DataGridRow>
                      <DataGridHeadCell>Endpoint</DataGridHeadCell>
                      <DataGridHeadCell>Project ID</DataGridHeadCell>
                      <DataGridHeadCell>Status</DataGridHeadCell>
                      {canEdit && <DataGridHeadCell>Accept</DataGridHeadCell>}
                      {canEdit && <DataGridHeadCell>Reject</DataGridHeadCell>}
                    </DataGridRow>
                    {consumers.map((c) => (
                      <DataGridRow key={c.id}>
                        <DataGridCell>
                          <div className="flex flex-col">
                            <div className="flex items-center gap-1">
                              <span className="font-semibold">{endpointNameMap.get(c.id) || "Unnamed"}</span>
                            </div>
                            <span className="text-xs text-theme-light">{c.id}</span>
                          </div>
                        </DataGridCell>
                        <DataGridCell className="text-xs">{c.project_id}</DataGridCell>
                        <DataGridCell>
                          <StatusBadge status={c.status} />
                        </DataGridCell>
                        {canEdit && (
                          <DataGridCell>
                            {(c.status === "REJECTED" || c.status === "PENDING_APPROVAL") && (
                              <Button
                                size="small"
                                onClick={() => handleAcceptIds([c.id])}
                                disabled={accept.isPending || reject.isPending}
                              >
                                Accept
                              </Button>
                            )}
                          </DataGridCell>
                        )}
                        {canEdit && (
                          <DataGridCell>
                            {(c.status === "AVAILABLE" || c.status === "PENDING_APPROVAL") && (
                              <Button
                                size="small"
                                variant="primary-danger"
                                onClick={() => handleRejectIds([c.id])}
                                disabled={accept.isPending || reject.isPending}
                              >
                                Reject
                              </Button>
                            )}
                          </DataGridCell>
                        )}
                      </DataGridRow>
                    ))}
                  </DataGrid>
                )}

                {consumers.length === 0 && <Message variant="info">No consumers.</Message>}
              </Stack>
            )}
          </TabPanel>
        </Tabs>
      </PanelBody>
      <PanelFooter>
        <Button onClick={close}>Close</Button>
      </PanelFooter>
    </Panel>
  );
};
