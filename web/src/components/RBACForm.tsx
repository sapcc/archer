// SPDX-FileCopyrightText: 2026 SAP SE
// SPDX-License-Identifier: Apache-2.0

import { useState, useEffect } from "react";
import { Modal, TextInput, Stack, FormRow, Message } from "@cloudoperators/juno-ui-components";
import { useStore } from "../store";
import { useCreateRBAC, useUpdateRBAC } from "../api";
import { ServicePicker } from "./ServicePicker";
import type { RBACPolicyCreate, RBACPolicyUpdate, Service } from "../types";

export const RBACForm = () => {
  const { editRBAC, closeModals } = useStore();
  const isEdit = !!editRBAC;
  const create = useCreateRBAC();
  const update = useUpdateRBAC();

  const [form, setForm] = useState({ service_id: "", service_name: "", target: "" });
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (editRBAC) {
      setForm({ service_id: editRBAC.service_id, service_name: "", target: editRBAC.target });
    }
  }, [editRBAC]);

  const set = <K extends keyof typeof form>(k: K, v: string) => {
    setForm((f) => ({ ...f, [k]: v }));
    setError(null);
  };

  const handleServiceSelect = (service: Service) => {
    set("service_id", service.id);
    setForm((f) => ({ ...f, service_name: service.name || service.id }));
  };

  const submit = async () => {
    if (!isEdit && !form.service_id) return setError("Service required");
    if (!form.target.trim()) return setError("Target project ID required");

    try {
      if (isEdit) {
        const data: RBACPolicyUpdate = { target: form.target.trim() };
        await update.mutateAsync({ id: editRBAC!.id, data });
      } else {
        const data: RBACPolicyCreate = {
          service_id: form.service_id,
          target: form.target.trim(),
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
      title={isEdit ? "Edit RBAC Policy" : "Create RBAC Policy"}
      open
      onCancel={closeModals}
      onConfirm={submit}
      confirmButtonLabel={pending ? "Saving..." : isEdit ? "Update" : "Create"}
      confirmButtonIcon={isEdit ? "edit" : "addCircle"}
      disableConfirmButton={pending}
    >
      <Stack direction="vertical" gap="4">
        {error && <Message variant="danger">{error}</Message>}

        {!isEdit && (
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
        )}

        <FormRow>
          <TextInput
            label="Target Project ID"
            value={form.target}
            onChange={(e) => set("target", e.target.value)}
            placeholder="32-char project ID or * for all"
            required
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
