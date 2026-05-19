// SPDX-FileCopyrightText: 2026 SAP SE
// SPDX-License-Identifier: Apache-2.0

import { useState } from "react";
import { Modal, Stack, Message, Select, SelectOption, FormRow } from "@cloudoperators/juno-ui-components";
import { useAgents, useMigrateService } from "../api";
import type { Service } from "../types";

interface MigrateModalProps {
  service: Service;
  onClose: () => void;
}

export const MigrateModal = ({ service, onClose }: MigrateModalProps) => {
  const [targetHost, setTargetHost] = useState<string>("");
  const [error, setError] = useState<string | null>(null);
  const { data: agentsData, isLoading: agentsLoading } = useAgents();
  const migrate = useMigrateService();

  const agents = agentsData?.items ?? [];
  // Filter agents: enabled, not current host, same provider type
  const enabledAgents = agents.filter((a) => a.enabled && a.host !== service.host && a.provider === service.provider);

  const handleMigrate = async () => {
    setError(null);
    try {
      await migrate.mutateAsync({
        id: service.id,
        targetHost: targetHost || undefined,
      });
      onClose();
    } catch (e) {
      setError(e instanceof Error ? e.message : "Migration failed");
    }
  };

  return (
    <Modal
      title="Migrate Service"
      open
      onCancel={onClose}
      onConfirm={handleMigrate}
      confirmButtonLabel={migrate.isPending ? "Migrating..." : "Migrate"}
      confirmButtonIcon="upload"
      disableConfirmButton={migrate.isPending}
    >
      <Stack direction="vertical" gap="4">
        {error && <Message variant="danger">{error}</Message>}

        <div className="text-sm">
          <p>
            Migrate service <strong>{service.name || service.id}</strong> to another agent.
          </p>
          {service.host && (
            <p className="text-theme-light mt-2">
              Current host: <code>{service.host}</code>
            </p>
          )}
        </div>

        <FormRow>
          <Select
            label="Target Host"
            value={targetHost}
            onChange={(v) => setTargetHost(String(v ?? ""))}
            placeholder="Auto-select (least loaded)"
            loading={agentsLoading}
          >
            {enabledAgents.map((agent) => (
              <SelectOption key={agent.host} value={agent.host}>
                {`${agent.host} (${agent.availability_zone || "cross-AZ"}, ${agent.services} services)`}
              </SelectOption>
            ))}
          </Select>
          <p className="text-xs text-theme-light mt-1">
            Leave empty to auto-select the least loaded agent in the same availability zone.
          </p>
        </FormRow>
      </Stack>
    </Modal>
  );
};
