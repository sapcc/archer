// SPDX-FileCopyrightText: 2026 SAP SE
// SPDX-License-Identifier: Apache-2.0

import { useState } from "react";
import { PopupPicker } from "./shared/PopupPicker";
import { useSubnets } from "../api";
import type { Subnet } from "../types";

interface SubnetPickerProps {
  onSelect: (subnet: Subnet) => void;
  disabled?: boolean;
}

export const SubnetPicker = ({ onSelect, disabled }: SubnetPickerProps) => {
  const [fetchEnabled, setFetchEnabled] = useState(false);
  const { data, isLoading, error } = useSubnets(fetchEnabled);

  return (
    <PopupPicker
      items={data?.subnets ?? []}
      isLoading={isLoading}
      error={error}
      onSelect={onSelect}
      onOpen={() => setFetchEnabled(true)}
      disabled={disabled}
      title="Browse subnets"
      filterPlaceholder="Filter by name, ID, or CIDR..."
      emptyMessage="No subnets found"
      errorMessage="Failed to load subnets"
      renderItem={(subnet) => (
        <div className="flex flex-col items-start gap-0.5">
          <span className="font-medium">{subnet.name || "(unnamed)"}</span>
          <span className="text-xs text-theme-light font-mono">{subnet.cidr}</span>
          <span className="text-xs text-theme-light font-mono">{subnet.id}</span>
        </div>
      )}
      filterFn={(subnet, search) =>
        (subnet.name?.toLowerCase().includes(search.toLowerCase()) ?? false) ||
        subnet.id.toLowerCase().includes(search.toLowerCase()) ||
        subnet.cidr.toLowerCase().includes(search.toLowerCase())
      }
    />
  );
};
