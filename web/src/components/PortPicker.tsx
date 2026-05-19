// SPDX-FileCopyrightText: 2026 SAP SE
// SPDX-License-Identifier: Apache-2.0

import { useState } from "react";
import { PopupPicker } from "./shared/PopupPicker";
import { usePorts } from "../api";
import type { Port } from "../types";

interface PortPickerProps {
  onSelect: (port: Port) => void;
  disabled?: boolean;
}

export const PortPicker = ({ onSelect, disabled }: PortPickerProps) => {
  const [fetchEnabled, setFetchEnabled] = useState(false);
  const { data, isLoading, error } = usePorts(fetchEnabled);

  return (
    <PopupPicker
      items={data?.ports ?? []}
      isLoading={isLoading}
      error={error}
      onSelect={onSelect}
      onOpen={() => setFetchEnabled(true)}
      disabled={disabled}
      title="Browse ports"
      filterPlaceholder="Filter by name, ID, or IP..."
      emptyMessage="No unbound ports found"
      errorMessage="Failed to load ports"
      renderItem={(port) => (
        <div className="flex flex-col items-start gap-0.5">
          <span className="font-medium">{port.name || "(unnamed)"}</span>
          <span className="text-xs text-theme-light">
            {port.fixed_ips.map((ip) => ip.ip_address).join(", ") || "No IPs"}
          </span>
          <span className="text-xs text-theme-light font-mono">{port.id}</span>
        </div>
      )}
      filterFn={(port, search) =>
        (port.name?.toLowerCase().includes(search.toLowerCase()) ?? false) ||
        port.id.toLowerCase().includes(search.toLowerCase()) ||
        port.fixed_ips.some((ip) => ip.ip_address.includes(search))
      }
    />
  );
};
