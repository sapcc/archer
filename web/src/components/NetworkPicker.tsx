// SPDX-FileCopyrightText: 2026 SAP SE
// SPDX-License-Identifier: Apache-2.0

import { useState } from "react";
import { PopupPicker } from "./shared/PopupPicker";
import { useNetworks } from "../api";
import type { Network } from "../types";

interface NetworkPickerProps {
  onSelect: (network: Network) => void;
  disabled?: boolean;
}

export const NetworkPicker = ({ onSelect, disabled }: NetworkPickerProps) => {
  const [fetchEnabled, setFetchEnabled] = useState(false);
  const { data, isLoading, error } = useNetworks(fetchEnabled);

  return (
    <PopupPicker
      items={data?.networks ?? []}
      isLoading={isLoading}
      error={error}
      onSelect={onSelect}
      onOpen={() => setFetchEnabled(true)}
      disabled={disabled}
      title="Browse networks"
      filterPlaceholder="Filter by name or ID..."
      emptyMessage="No networks found"
      errorMessage="Failed to load networks"
    />
  );
};
