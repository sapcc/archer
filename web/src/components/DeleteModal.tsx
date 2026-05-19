// SPDX-FileCopyrightText: 2026 SAP SE
// SPDX-License-Identifier: Apache-2.0

import { Modal, ModalFooter, Button, Stack, Message, Switch, Box } from "@cloudoperators/juno-ui-components";
import { useStore } from "../store";

interface Props {
  onConfirm: (cascade?: boolean) => void;
  isLoading?: boolean;
  error?: Error | null;
  cascade?: boolean;
  onCascadeChange?: (cascade: boolean) => void;
}

export const DeleteModal = ({ onConfirm, isLoading, error, cascade, onCascadeChange }: Props) => {
  const { deleteTarget, closeModals } = useStore();
  if (!deleteTarget) return null;

  const { type, item } = deleteTarget;
  const name = "name" in item ? item.name : item.id;
  const isService = type === "service";

  return (
    <Modal
      title={`Delete ${type}`}
      open
      onCancel={closeModals}
      modalFooter={
        <ModalFooter className="justify-end gap-2">
          <Button variant="subdued" onClick={closeModals} disabled={isLoading}>
            Cancel
          </Button>
          <Button variant="primary-danger" icon="deleteForever" onClick={() => onConfirm(cascade)} disabled={isLoading}>
            {isLoading ? "Deleting..." : "Delete"}
          </Button>
        </ModalFooter>
      }
    >
      <Stack direction="vertical" gap="4">
        <p>Are you sure you want to delete this {type}?</p>
        <Box>
          <strong>{name || item.id}</strong>
          <br />
          <span className="text-xs text-theme-light">{item.id}</span>
        </Box>
        {isService && onCascadeChange && (
          <Switch
            on={cascade}
            onChange={(e) => onCascadeChange((e.target as HTMLButtonElement).getAttribute("aria-checked") === "true")}
            label="Cascade"
            helptext="Force delete all associated endpoints"
          />
        )}
        {error && <Message variant="danger">{error.message}</Message>}
      </Stack>
    </Modal>
  );
};
