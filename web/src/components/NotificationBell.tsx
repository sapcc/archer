// SPDX-FileCopyrightText: 2026 SAP SE
// SPDX-License-Identifier: Apache-2.0

import { useState, useEffect } from "react";
import { Icon, Button } from "@cloudoperators/juno-ui-components";
import { useStore } from "../store";

const formatDuration = (timestamp: number): string => {
  const seconds = Math.floor((Date.now() - timestamp) / 1000);
  if (seconds < 60) return "just now";
  const minutes = Math.floor(seconds / 60);
  if (minutes < 60) return `${minutes}m ago`;
  const hours = Math.floor(minutes / 60);
  if (hours < 24) return `${hours}h ago`;
  const days = Math.floor(hours / 24);
  return `${days}d ago`;
};

const variantStyles = {
  success: "border-l-green-500 bg-green-500/10",
  warning: "border-l-yellow-500 bg-yellow-500/10",
  danger: "border-l-red-500 bg-red-500/10",
};

const variantIcons = {
  success: "checkCircle",
  warning: "warning",
  danger: "dangerous",
} as const;

export const NotificationBell = () => {
  const toasts = useStore((s) => s.toasts);
  const dismissToast = useStore((s) => s.dismissToast);
  const clearToasts = useStore((s) => s.clearToasts);
  const [isOpen, setIsOpen] = useState(false);
  const [, setTick] = useState(0);

  // Update durations every 30 seconds
  useEffect(() => {
    const interval = setInterval(() => setTick((t) => t + 1), 30000);
    return () => clearInterval(interval);
  }, []);

  // Auto-open when new toast arrives
  useEffect(() => {
    if (toasts.length > 0) {
      setIsOpen(true);
    }
  }, [toasts.length]);

  const unreadCount = toasts.length;

  return (
    <div className="relative">
      <button
        onClick={() => setIsOpen(!isOpen)}
        className="relative p-2 rounded hover:bg-theme-background-lvl-2 cursor-pointer"
        title="Notifications"
      >
        <Icon icon="comment" size="24" />
        {unreadCount > 0 && (
          <span className="absolute -top-1 -right-1 bg-red-500 text-white text-xs rounded-full w-5 h-5 flex items-center justify-center">
            {unreadCount > 9 ? "9+" : unreadCount}
          </span>
        )}
      </button>

      {isOpen && (
        <>
          {/* Backdrop */}
          <div className="fixed inset-0 z-40" onClick={() => setIsOpen(false)} />

          {/* Dropdown */}
          <div
            className="absolute right-0 top-full mt-2 w-80 max-h-96 overflow-y-auto rounded-lg shadow-lg z-50"
            style={{
              backgroundColor: "var(--color-background-lvl-1)",
              border: "1px solid var(--color-background-lvl-3)",
              color: "var(--color-text-default)",
            }}
          >
            <div
              className="sticky top-0 p-3 flex items-center justify-between"
              style={{
                backgroundColor: "var(--color-background-lvl-1)",
                borderBottom: "1px solid var(--color-background-lvl-3)",
              }}
            >
              <span className="font-semibold">Notifications</span>
              {toasts.length > 0 && (
                <Button size="small" variant="subdued" onClick={clearToasts}>
                  Clear all
                </Button>
              )}
            </div>

            {toasts.length === 0 ? (
              <div className="p-4 text-center" style={{ color: "var(--color-text-light)" }}>
                No notifications
              </div>
            ) : (
              <div>
                {toasts.map((t) => (
                  <div
                    key={t.id}
                    className={`p-3 border-l-4 flex items-start gap-3 ${variantStyles[t.variant]}`}
                    style={{ borderBottom: "1px solid var(--color-background-lvl-3)" }}
                  >
                    <Icon
                      icon={variantIcons[t.variant]}
                      size="20"
                      className={
                        t.variant === "success"
                          ? "text-green-500"
                          : t.variant === "warning"
                            ? "text-yellow-500"
                            : "text-red-500"
                      }
                    />
                    <div className="flex-1 min-w-0">
                      <p className="text-sm" style={{ color: "var(--color-text-default)" }}>
                        {t.message}
                      </p>
                      <p className="text-xs mt-1" style={{ color: "var(--color-text-light)" }}>
                        {formatDuration(t.timestamp)}
                      </p>
                    </div>
                    <button
                      onClick={() => dismissToast(t.id)}
                      className="cursor-pointer"
                      style={{ color: "var(--color-text-light)" }}
                    >
                      <Icon icon="close" size="16" />
                    </button>
                  </div>
                ))}
              </div>
            )}
          </div>
        </>
      )}
    </div>
  );
};
