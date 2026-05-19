// SPDX-FileCopyrightText: 2026 SAP SE
// SPDX-License-Identifier: Apache-2.0

import { Badge } from "@cloudoperators/juno-ui-components";
import type { ServiceStatus, EndpointStatus, HealthStatus } from "../types";

type Status = ServiceStatus | EndpointStatus | HealthStatus;
type BadgeVariant = "success" | "info" | "warning" | "danger" | "default";

const COLORS: Record<string, BadgeVariant> = {
  AVAILABLE: "success",
  ONLINE: "success",
  PENDING_CREATE: "info",
  PENDING_UPDATE: "info",
  PENDING_APPROVAL: "warning",
  PENDING_DELETE: "info",
  PENDING_REJECTED: "info",
  DEGRADED: "warning",
  REJECTED: "danger",
  FAILED: "danger",
  OFFLINE: "danger",
  ERROR_QUOTA: "danger",
  UNAVAILABLE: "default",
  UNCHECKED: "default",
};

const PENDING_STATUSES = ["PENDING_CREATE", "PENDING_UPDATE", "PENDING_DELETE", "PENDING_REJECTED"];

export const StatusBadge = ({ status }: { status: Status | null | undefined }) => {
  if (!status) return null;
  const isPending = PENDING_STATUSES.includes(status);

  return (
    <Badge
      variant={COLORS[status] ?? "default"}
      style={isPending ? { animation: "pulse-pending-info 2s ease-in-out infinite" } : undefined}
    >
      {status}
    </Badge>
  );
};
