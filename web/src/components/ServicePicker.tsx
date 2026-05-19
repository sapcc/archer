// SPDX-FileCopyrightText: 2026 SAP SE
// SPDX-License-Identifier: Apache-2.0

import { PopupPicker } from "./shared/PopupPicker";
import { useServices } from "../api";
import type { Service } from "../types";

interface ServicePickerProps {
  onSelect: (service: Service) => void;
  disabled?: boolean;
}

export const ServicePicker = ({ onSelect, disabled }: ServicePickerProps) => {
  const { data, isLoading, error } = useServices();

  return (
    <PopupPicker
      items={data?.items ?? []}
      isLoading={isLoading}
      error={error}
      onSelect={onSelect}
      disabled={disabled}
      title="Browse services"
      filterPlaceholder="Filter by name or ID..."
      emptyMessage="No services found"
      errorMessage="Failed to load services"
    />
  );
};
