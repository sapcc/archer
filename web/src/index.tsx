// SPDX-FileCopyrightText: 2026 SAP SE
// SPDX-License-Identifier: Apache-2.0

import { createRoot, Root } from "react-dom/client";
import { createElement } from "react";
import type { AppProps } from "./types";

let root: Root | null = null;

export const mount = async (container: HTMLElement, options: { props?: AppProps } = {}) => {
  const props = options.props || {};

  // Enable mocking if requested
  if (props.mockAPI) {
    const { startMocking } = await import("./mocks/browser");
    const endpoint = props.endpoint || `https://${window.location.host}`;
    await startMocking(endpoint);
  }

  // Load and render app
  const { default: App } = await import("./App");
  root = createRoot(container);
  root.render(createElement(App, { ...props, endpoint: props.endpoint || `https://${window.location.host}` }));
};

export const unmount = () => root?.unmount();
