// SPDX-FileCopyrightText: 2026 SAP SE
// SPDX-License-Identifier: Apache-2.0

import { useState, useEffect } from "react";
import {
  Modal,
  TextInput,
  Textarea,
  Select,
  SelectOption,
  Switch,
  Stack,
  FormRow,
  Message,
} from "@cloudoperators/juno-ui-components";
import { useStore } from "../store";
import { useCreateService, useUpdateService } from "../api";
import { NetworkPicker } from "./NetworkPicker";
import type { ServiceCreate, ServiceUpdate, Visibility, Protocol, Network, Provider } from "../types";

export const ServiceForm = () => {
  const { editService, createServiceProvider, closeModals } = useStore();
  const neutronEndpoint = useStore((s) => s.globalAPI.neutronEndpoint);
  const isEdit = !!editService;
  const create = useCreateService();
  const update = useUpdateService();

  const [form, setForm] = useState({
    name: "",
    description: "",
    network_id: "",
    ip_addresses: "",
    ports: "",
    enabled: true,
    visibility: "private" as Visibility,
    require_approval: false,
    proxy_protocol: false,
    connection_mirroring: false,
    protocol: "TCP" as Protocol,
    tags: "",
  });
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (editService) {
      setForm({
        name: editService.name ?? "",
        description: editService.description ?? "",
        network_id: editService.network_id,
        ip_addresses: formatIPs(editService.ip_addresses),
        ports: editService.ports?.join(", ") ?? "",
        enabled: editService.enabled,
        visibility: editService.visibility,
        require_approval: editService.require_approval,
        proxy_protocol: editService.proxy_protocol,
        connection_mirroring: editService.connection_mirroring,
        protocol: editService.protocol,
        tags: editService.tags?.join(", ") ?? "",
      });
    }
  }, [editService]);

  const set = <K extends keyof typeof form>(k: K, v: (typeof form)[K]) => {
    setForm((f) => ({ ...f, [k]: v }));
    setError(null);
  };

  const parseList = (s: string) =>
    s
      .split(",")
      .map((x) => x.trim())
      .filter(Boolean);
  const parseIPs = (s: string) => parseList(s).map((ip) => ip.replace(/\/32$/, ""));
  const parsePorts = (s: string) =>
    parseList(s)
      .map((x) => parseInt(x, 10))
      .filter((n) => !isNaN(n));
  const formatIPs = (ips: string[] | undefined) => ips?.map((ip) => ip.replace(/\/32$/, "")).join(", ") ?? "";

  const submit = async () => {
    const isNI = createServiceProvider === "cp";
    if (!isNI && !form.network_id.trim()) return setError("Network ID required");
    if (!form.ip_addresses.trim()) return setError("IP addresses required");
    if (!form.ports.trim()) return setError("Ports required");

    try {
      if (isEdit) {
        const newIPs = parseIPs(form.ip_addresses);
        const newPorts = parsePorts(form.ports);
        const oldIPs = editService!.ip_addresses?.map((ip) => ip.replace(/\/32$/, "")) ?? [];
        const oldPorts = editService!.ports ?? [];

        // Only include ip_addresses/ports if they changed to avoid 409 conflict
        const ipsChanged = JSON.stringify(newIPs.sort()) !== JSON.stringify(oldIPs.sort());
        const portsChanged = JSON.stringify(newPorts.sort()) !== JSON.stringify(oldPorts.sort());

        const data: ServiceUpdate = {
          name: form.name || null,
          description: form.description || null,
          enabled: form.enabled,
          visibility: form.visibility,
          require_approval: form.require_approval,
          proxy_protocol: form.proxy_protocol,
          connection_mirroring: form.connection_mirroring,
          protocol: form.protocol,
          tags: parseList(form.tags),
        };
        if (ipsChanged) data.ip_addresses = newIPs;
        if (portsChanged) data.ports = newPorts;

        await update.mutateAsync({ id: editService!.id, data });
      } else {
        const data: ServiceCreate = {
          name: form.name || undefined,
          description: form.description || undefined,
          network_id: isNI ? "00000000-0000-0000-0000-000000000000" : form.network_id.trim(),
          ip_addresses: parseIPs(form.ip_addresses),
          ports: parsePorts(form.ports),
          enabled: form.enabled,
          visibility: form.visibility,
          require_approval: form.require_approval,
          proxy_protocol: isNI ? undefined : form.proxy_protocol,
          connection_mirroring: isNI ? undefined : form.connection_mirroring,
          protocol: form.protocol,
          tags: parseList(form.tags),
          provider: createServiceProvider ?? undefined,
        };
        await create.mutateAsync(data);
      }
      closeModals();
    } catch (e) {
      setError(e instanceof Error ? e.message : "Unknown error");
    }
  };

  const handleNetworkSelect = (network: Network) => {
    set("network_id", network.id);
  };

  const pending = create.isPending || update.isPending;

  const getTitle = () => {
    if (isEdit) return "Edit Service";
    if (createServiceProvider === "cp") return "Create Control Plane Service";
    return "Create Service";
  };

  return (
    <Modal
      title={getTitle()}
      open
      onCancel={closeModals}
      onConfirm={submit}
      confirmButtonLabel={pending ? "Saving..." : isEdit ? "Update" : "Create"}
      confirmButtonIcon={isEdit ? "edit" : "addCircle"}
      disableConfirmButton={pending}
    >
      <Stack direction="vertical" gap="4">
        {error && <Message variant="danger">{error}</Message>}

        <div className="flex items-center justify-between bg-theme-background-lvl-2 rounded p-3">
          <span className="font-semibold">Enabled</span>
          <Switch on={form.enabled} onClick={() => set("enabled", !form.enabled)} />
        </div>

        <FormRow>
          <TextInput label="Name" value={form.name} onChange={(e) => set("name", e.target.value)} />
        </FormRow>
        <FormRow>
          <Textarea label="Description" value={form.description} onChange={(e) => set("description", e.target.value)} />
        </FormRow>

        <fieldset className="border border-theme-background-lvl-4 rounded p-4">
          <legend className="px-2 text-sm font-semibold text-theme-light">Network Configuration</legend>
          <Stack direction="vertical" gap="4">
            {createServiceProvider !== "cp" && (
              <FormRow>
                <div className="flex gap-2 items-end">
                  <div className="flex-1">
                    <TextInput
                      label="Network ID"
                      value={form.network_id}
                      onChange={(e) => set("network_id", e.target.value)}
                      disabled={isEdit}
                      required
                    />
                  </div>
                  {neutronEndpoint && !isEdit && <NetworkPicker onSelect={handleNetworkSelect} />}
                </div>
              </FormRow>
            )}
            <FormRow>
              <TextInput
                label="IP Addresses"
                value={form.ip_addresses}
                onChange={(e) => set("ip_addresses", e.target.value)}
                placeholder="10.0.1.10, 10.0.1.11"
                helptext="Backend IP addresses (multiple IPs are round-robin load balanced)"
                required
              />
            </FormRow>
            <div className="flex gap-4">
              <div className="flex-1">
                <FormRow>
                  <TextInput
                    label="Ports"
                    value={form.ports}
                    onChange={(e) => set("ports", e.target.value)}
                    placeholder="80, 443"
                    helptext="Backend ports to expose"
                    required
                  />
                </FormRow>
              </div>
              <div className="flex-1">
                <FormRow>
                  <Select label="Protocol" value={form.protocol} onChange={(v) => set("protocol", v as Protocol)}>
                    <SelectOption value="TCP">TCP</SelectOption>
                    <SelectOption value="HTTP">HTTP</SelectOption>
                  </Select>
                </FormRow>
              </div>
            </div>
          </Stack>
        </fieldset>

        <div className="flex items-center justify-between">
          <div>
            <span>Public Visibility</span>
            <p className="text-xs text-theme-light">Make this service visible to all projects</p>
          </div>
          <Switch
            on={form.visibility === "public"}
            onClick={() => set("visibility", form.visibility === "public" ? "private" : "public")}
          />
        </div>
        <Stack direction="vertical" gap="4">
          <div className="flex items-center justify-between">
            <div>
              <span>Require Approval</span>
              <p className="text-xs text-theme-light">Endpoint requests must be approved before activation</p>
            </div>
            <Switch on={form.require_approval} onClick={() => set("require_approval", !form.require_approval)} />
          </div>
          {createServiceProvider !== "cp" && (
            <>
              <div className="flex items-center justify-between">
                <div>
                  <span>Proxy Protocol v2</span>
                  <p className="text-xs text-theme-light">Prepend proxy protocol header to TCP payload</p>
                </div>
                <Switch on={form.proxy_protocol} onClick={() => set("proxy_protocol", !form.proxy_protocol)} />
              </div>
              <div className="flex items-center justify-between">
                <div>
                  <span>Connection Mirroring</span>
                  <p className="text-xs text-theme-light">
                    Enable BIG-IP connection mirroring for HA failover (may increase latency)
                  </p>
                </div>
                <Switch
                  on={form.connection_mirroring}
                  onClick={() => set("connection_mirroring", !form.connection_mirroring)}
                />
              </div>
            </>
          )}
        </Stack>

        <FormRow>
          <TextInput
            label="Tags"
            value={form.tags}
            onChange={(e) => set("tags", e.target.value)}
            placeholder="production, team:network"
            helptext="Comma-separated tags for organization"
          />
        </FormRow>

        <p className="text-xs text-theme-light flex items-center gap-1">
          <span className="w-2 h-2 rounded-full jn:bg-theme-required" />
          Required fields
        </p>
      </Stack>
    </Modal>
  );
};
