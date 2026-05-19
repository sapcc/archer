// SPDX-FileCopyrightText: 2026 SAP SE
// SPDX-License-Identifier: Apache-2.0

import { useState, ReactNode } from "react";
import {
  PopupMenu,
  PopupMenuToggle,
  PopupMenuOptions,
  PopupMenuItem,
  Button,
  TextInput,
  Spinner,
  Icon,
} from "@cloudoperators/juno-ui-components";

interface PopupPickerItem {
  id: string;
  name?: string | null;
}

interface PopupPickerProps<T extends PopupPickerItem> {
  items: T[];
  isLoading?: boolean;
  error?: Error | null;
  onSelect: (item: T) => void;
  onOpen?: () => void;
  disabled?: boolean;
  title?: string;
  filterPlaceholder?: string;
  emptyMessage?: string;
  errorMessage?: string;
  renderItem?: (item: T) => ReactNode;
  filterFn?: (item: T, search: string) => boolean;
}

export function PopupPicker<T extends PopupPickerItem>({
  items,
  isLoading = false,
  error = null,
  onSelect,
  onOpen,
  disabled = false,
  title = "Browse",
  filterPlaceholder = "Filter by name or ID...",
  emptyMessage = "No items found",
  errorMessage = "Failed to load items",
  renderItem,
  filterFn,
}: PopupPickerProps<T>) {
  const [search, setSearch] = useState("");

  const defaultFilter = (item: T, s: string) =>
    (item.name?.toLowerCase().includes(s.toLowerCase()) ?? false) || item.id.toLowerCase().includes(s.toLowerCase());

  const filtered = items.filter((item) => (filterFn ? filterFn(item, search) : defaultFilter(item, search)));

  const handleClose = () => {
    setSearch("");
  };

  const handleSelect = (item: T) => {
    onSelect(item);
    setSearch("");
  };

  const defaultRenderItem = (item: T) => (
    <div className="flex flex-col items-start gap-0.5">
      <span className="font-medium">{item.name || "(unnamed)"}</span>
      <span className="text-xs text-theme-light font-mono">{item.id}</span>
    </div>
  );

  return (
    <PopupMenu onOpen={onOpen} onClose={handleClose} disabled={disabled}>
      <PopupMenuToggle>
        <Button variant="subdued" title={title} disabled={disabled} className="h-[2.375rem]">
          <Icon icon="search" size="18" />
        </Button>
      </PopupMenuToggle>
      <PopupMenuOptions className="w-96">
        <div className="p-2 border-b border-theme-background-lvl-3">
          <TextInput
            placeholder={filterPlaceholder}
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            autoFocus
          />
        </div>

        <div className="max-h-64 overflow-auto">
          {isLoading && (
            <div className="flex justify-center py-4">
              <Spinner />
            </div>
          )}

          {error && <div className="p-3 text-sm text-theme-danger">{errorMessage}</div>}

          {!isLoading && !error && filtered.length === 0 && (
            <div className="p-3 text-sm text-theme-light">{emptyMessage}</div>
          )}

          {!isLoading &&
            !error &&
            filtered.map((item) => (
              <PopupMenuItem key={item.id} onClick={() => handleSelect(item)}>
                {renderItem ? renderItem(item) : defaultRenderItem(item)}
              </PopupMenuItem>
            ))}
        </div>
      </PopupMenuOptions>
    </PopupMenu>
  );
}
