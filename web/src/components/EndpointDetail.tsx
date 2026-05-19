// SPDX-FileCopyrightText: 2026 SAP SE
// SPDX-License-Identifier: Apache-2.0

import {
  Panel,
  PanelBody,
  PanelFooter,
  Button,
  DataGrid,
  DataGridRow,
  DataGridCell,
  LoadingIndicator,
  Message,
  Spinner,
} from "@cloudoperators/juno-ui-components";
import { useParams, useNavigate } from "react-router";
import { useEndpoint, useService, isPendingEndpoint } from "../api";
import { StatusBadge } from "./StatusBadge";

export const EndpointDetail = () => {
  const { endpointId } = useParams<{ endpointId: string }>();
  const navigate = useNavigate();
  const { data: ep, isLoading, error } = useEndpoint(endpointId);
  const { data: service } = useService(ep?.service_id);

  const close = () => navigate("/endpoints");
  const fmt = (d: string | undefined) => (d ? new Date(d).toLocaleString() : "-");

  const targetType = ep?.target.network ? "Network" : ep?.target.subnet ? "Subnet" : ep?.target.port ? "Port" : "-";
  const targetValue = ep?.target.network || ep?.target.subnet || ep?.target.port || "-";

  if (isLoading) {
    return (
      <Panel heading="Endpoint Details" opened onClose={close}>
        <PanelBody>
          <LoadingIndicator className="m-auto" />
        </PanelBody>
      </Panel>
    );
  }

  if (error || !ep) {
    return (
      <Panel heading="Endpoint Details" opened onClose={close}>
        <PanelBody>
          <Message variant="danger">{error?.message || "Not found"}</Message>
        </PanelBody>
      </Panel>
    );
  }

  const serviceLink = (
    <div className="flex flex-col">
      <button
        type="button"
        onClick={() => navigate(`/services/${ep.service_id}`)}
        className="text-theme-accent hover:underline text-left cursor-pointer"
      >
        {service?.name || "Unnamed"}
      </button>
      <span className="text-xs text-theme-light">{ep.service_id}</span>
    </div>
  );

  const panelHeading = (
    <span className="flex items-center gap-2">
      {ep.name || "Endpoint Details"}
      {isPendingEndpoint(ep) && <Spinner size="small" />}
    </span>
  );

  return (
    <Panel heading={panelHeading} opened onClose={close}>
      <PanelBody>
        <DataGrid columns={2}>
          {[
            ["ID", ep.id],
            ["Name", ep.name || "-"],
            ["Description", ep.description || "-"],
            ["Status", <StatusBadge key="s" status={ep.status} />],
            ["Service", serviceLink],
            ["IP Address", ep.ip_address || "-"],
            ["Target Type", targetType],
            ["Target ID", targetValue],
            ["Tags", ep.tags?.join(", ") || "-"],
            ["Project", ep.project_id],
            ["Created", fmt(ep.created_at)],
            ["Updated", fmt(ep.updated_at)],
          ].map(([label, value], i) => (
            <DataGridRow key={i}>
              <DataGridCell className="font-semibold">{label}</DataGridCell>
              <DataGridCell>{value}</DataGridCell>
            </DataGridRow>
          ))}
        </DataGrid>
      </PanelBody>
      <PanelFooter>
        <Button onClick={close}>Close</Button>
      </PanelFooter>
    </Panel>
  );
};
