// SPDX-FileCopyrightText: 2026 SAP SE
// SPDX-License-Identifier: Apache-2.0

import { useRef, useEffect } from "react";
import { useStore } from "../store";
import type { ServiceStatus, EndpointStatus } from "../types";

type Status = ServiceStatus | EndpointStatus;
type ItemType = "service" | "endpoint";

interface TrackedItem {
  id: string;
  name: string | null;
  status: Status;
  created_at: string;
  updated_at: string;
}

interface TrackedState {
  name: string | null;
  status: Status;
  pendingSince: number; // timestamp when item entered pending state
}

const PENDING_STATUSES: Status[] = [
  "PENDING_CREATE",
  "PENDING_UPDATE",
  "PENDING_DELETE",
  "PENDING_APPROVAL",
  "PENDING_REJECTED",
];

const isPending = (status: Status): boolean => PENDING_STATUSES.includes(status);

const getToastVariant = (status: Status | "DELETED"): "success" | "warning" | "danger" => {
  switch (status) {
    case "AVAILABLE":
    case "DELETED":
      return "success";
    case "UNAVAILABLE":
      return "warning";
    case "REJECTED":
    case "FAILED":
    case "ERROR_QUOTA":
      return "danger";
    default:
      return "success";
  }
};

const formatDuration = (ms: number): string => {
  const seconds = Math.floor(ms / 1000);
  if (seconds < 60) return `${seconds}s`;
  const minutes = Math.floor(seconds / 60);
  if (minutes < 60) return `${minutes}m ${seconds % 60}s`;
  const hours = Math.floor(minutes / 60);
  return `${hours}h ${minutes % 60}m`;
};

const getToastMessage = (
  type: ItemType,
  name: string | null,
  status: Status | "DELETED",
  durationMs: number | null
): string => {
  const typeLabel = type.charAt(0).toUpperCase() + type.slice(1);
  const displayName = name ? `${typeLabel} "${name}"` : typeLabel;
  const duration = durationMs !== null ? ` (${formatDuration(durationMs)})` : "";

  switch (status) {
    case "AVAILABLE":
      return `${displayName} is now available${duration}`;
    case "DELETED":
      return `${displayName} was deleted${duration}`;
    case "REJECTED":
      return `${displayName} was rejected${duration}`;
    case "FAILED":
      return `${displayName} failed${duration}`;
    case "ERROR_QUOTA":
      return `${displayName} hit quota limit${duration}`;
    case "UNAVAILABLE":
      return `${displayName} is unavailable${duration}`;
    default:
      return `${displayName} status changed to ${status}${duration}`;
  }
};

export const useStatusChangeToast = <T extends TrackedItem>(items: T[], type: ItemType) => {
  const trackedRef = useRef<Map<string, TrackedState>>(new Map());
  const addToast = useStore((s) => s.addToast);

  useEffect(() => {
    const tracked = trackedRef.current;
    const currentIds = new Set(items.map((i) => i.id));

    // Check for items that disappeared while in PENDING_DELETE
    for (const [id, state] of tracked) {
      if (!currentIds.has(id) && state.status === "PENDING_DELETE") {
        const durationMs = state.pendingSince > 0 ? Date.now() - state.pendingSince : null;
        addToast({
          variant: getToastVariant("DELETED"),
          message: getToastMessage(type, state.name, "DELETED", durationMs),
        });
        tracked.delete(id);
      } else if (!currentIds.has(id)) {
        // Item disappeared but wasn't in PENDING_DELETE - just clean up
        tracked.delete(id);
      }
    }

    for (const item of items) {
      const prev = tracked.get(item.id);
      const curr = item.status;
      const currIsPending = isPending(curr);

      if (!prev) {
        // First time seeing this item
        tracked.set(item.id, {
          name: item.name,
          status: curr,
          pendingSince: currIsPending ? new Date(item.updated_at).getTime() : 0,
        });
        continue;
      }

      const prevIsPending = isPending(prev.status);

      // Transition: non-pending -> pending (start tracking duration)
      if (!prevIsPending && currIsPending) {
        tracked.set(item.id, {
          name: item.name,
          status: curr,
          pendingSince: new Date(item.updated_at).getTime(),
        });
        continue;
      }

      // Transition: pending -> non-pending (show toast with duration)
      if (prevIsPending && !currIsPending) {
        const durationMs = prev.pendingSince > 0 ? Date.now() - prev.pendingSince : null;
        addToast({
          variant: getToastVariant(curr),
          message: getToastMessage(type, item.name, curr, durationMs),
        });
        tracked.set(item.id, {
          name: item.name,
          status: curr,
          pendingSince: 0,
        });
        continue;
      }

      // Status changed but still in same category (pending or non-pending)
      if (prev.status !== curr) {
        tracked.set(item.id, {
          name: item.name,
          status: curr,
          pendingSince: currIsPending ? prev.pendingSince : 0,
        });
      }
    }
  }, [items, type, addToast]);
};
