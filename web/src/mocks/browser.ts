// SPDX-FileCopyrightText: 2026 SAP SE
// SPDX-License-Identifier: Apache-2.0

import { setupWorker } from "msw/browser";
import { createHandlers } from "./handlers";

export const startMocking = async (endpoint: string) => {
  const worker = setupWorker(...createHandlers(endpoint));
  await worker.start({ onUnhandledRequest: "bypass" });
  return worker;
};
