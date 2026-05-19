// SPDX-FileCopyrightText: 2026 SAP SE
// SPDX-License-Identifier: Apache-2.0

import { useState, useEffect } from "react";
import {
  Modal,
  TextInput,
  Textarea,
  Select,
  SelectOption,
  Stack,
  FormRow,
  Message,
} from "@cloudoperators/juno-ui-components";
import { useStore } from "../store";
import { useCreateEndpoint, useUpdateEndpoint } from "../api";
import { ServicePicker } from "./ServicePicker";
import { NetworkPicker } from "./NetworkPicker";
import { SubnetPicker } from "./SubnetPicker";
import { PortPicker } from "./PortPicker";
import type { EndpointCreate, EndpointUpdate, Service, Network, Subnet, Port } from "../types";

type TargetType = "network" | "subnet" | "port";

export const EndpointForm = () => {
  const { editEndpoint, preselectedServiceId, closeModals } = useStore();
  const isEdit = !!editEndpoint;
  const create = useCreateEndpoint();
  const update = useUpdateEndpoint();

  const [form, setForm] = useState({
    name: "",
    description: "",
    service_id: preselectedServiceId ?? "",
    service_name: "",
    target_type: "network" as TargetType,
    target_value: "",
    tags: "",
  });
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (editEndpoint) {
      const t = editEndpoint.target;
      const type: TargetType = t.network ? "network" : t.subnet ? "subnet" : "port";
      const value = t.network || t.subnet || t.port || "";
      setForm({
        name: editEndpoint.name ?? "",
        description: editEndpoint.description ?? "",
        service_id: editEndpoint.service_id,
        service_name: "",
        target_type: type,
        target_value: value,
        tags: editEndpoint.tags?.join(", ") ?? "",
      });
    }
  }, [editEndpoint]);

  const set = <K extends keyof typeof form>(k: K, v: (typeof form)[K]) => {
    setForm((f) => ({ ...f, [k]: v }));
    setError(null);
  };

  const handleServiceSelect = (service: Service) => {
    set("service_id", service.id);
    setForm((f) => ({ ...f, service_name: service.name || service.id }));
  };

  const handleTargetSelect = (item: Network | Subnet | Port) => {
    set("target_value", item.id);
  };

  const neutronEndpoint = useStore((s) => s.globalAPI.neutronEndpoint);

  const renderTargetPicker = () => {
    if (!neutronEndpoint) return null;
    switch (form.target_type) {
      case "network":
        return <NetworkPicker onSelect={handleTargetSelect} />;
      case "subnet":
        return <SubnetPicker onSelect={handleTargetSelect} />;
      case "port":
        return <PortPicker onSelect={handleTargetSelect} />;
    }
  };

  const parseList = (s: string) =>
    s
      .split(",")
      .map((x) => x.trim())
      .filter(Boolean);

  const submit = async () => {
    if (!isEdit && !form.service_id) return setError("Service required");
    if (!isEdit && !form.target_value.trim()) return setError("Target required");

    try {
      if (isEdit) {
        const data: EndpointUpdate = {
          name: form.name || null,
          description: form.description || null,
          tags: parseList(form.tags),
        };
        await update.mutateAsync({ id: editEndpoint!.id, data });
      } else {
        const data: EndpointCreate = {
          service_id: form.service_id,
          name: form.name || undefined,
          description: form.description || undefined,
          target: { [form.target_type]: form.target_value.trim() },
          tags: parseList(form.tags),
        };
        await create.mutateAsync(data);
      }
      closeModals();
    } catch (e) {
      setError(e instanceof Error ? e.message : "Unknown error");
    }
  };

  const pending = create.isPending || update.isPending;

  return (
    <Modal
      title={isEdit ? "Edit Endpoint" : "Create Endpoint"}
      open
      onCancel={closeModals}
      onConfirm={submit}
      confirmButtonLabel={pending ? "Saving..." : isEdit ? "Update" : "Create"}
      confirmButtonIcon={isEdit ? "edit" : "addCircle"}
      disableConfirmButton={pending}
    >
      <Stack direction="vertical" gap="4">
        {error && <Message variant="danger">{error}</Message>}

        <FormRow>
          <TextInput label="Name" value={form.name} onChange={(e) => set("name", e.target.value)} />
        </FormRow>
        <FormRow>
          <Textarea label="Description" value={form.description} onChange={(e) => set("description", e.target.value)} />
        </FormRow>

        {!isEdit && (
          <>
            <FormRow>
              <div className="flex gap-2 items-end">
                <div className="flex-1">
                  <TextInput
                    label="Service"
                    value={form.service_name || form.service_id}
                    onChange={(e) => {
                      set("service_id", e.target.value);
                      setForm((f) => ({ ...f, service_name: "" }));
                    }}
                    placeholder="Service ID"
                    required
                  />
                </div>
                <ServicePicker onSelect={handleServiceSelect} />
              </div>
            </FormRow>

            <fieldset className="border border-theme-background-lvl-4 rounded p-4">
              <legend className="px-2 text-sm font-semibold text-theme-light">Target Configuration</legend>
              <Stack direction="vertical" gap="4">
                <FormRow>
                  <Select
                    label="Target Type"
                    value={form.target_type}
                    onChange={(v) => {
                      set("target_type", v as TargetType);
                      set("target_value", "");
                    }}
                  >
                    <SelectOption value="network">Network</SelectOption>
                    <SelectOption value="subnet">Subnet</SelectOption>
                    <SelectOption value="port">Port</SelectOption>
                  </Select>
                </FormRow>
                <FormRow>
                  <div className="flex gap-2 items-end">
                    <div className="flex-1">
                      <TextInput
                        label="Target ID"
                        value={form.target_value}
                        onChange={(e) => set("target_value", e.target.value)}
                        placeholder="UUID"
                        required
                      />
                    </div>
                    {renderTargetPicker()}
                  </div>
                </FormRow>
              </Stack>
            </fieldset>
          </>
        )}

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
