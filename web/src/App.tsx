// SPDX-FileCopyrightText: 2026 SAP SE
// SPDX-License-Identifier: Apache-2.0

import { useEffect, useState } from "react";
import {
  AppShell,
  AppShellProvider,
  Message,
  LoadingIndicator,
  PageHeader,
  Stack,
  Icon,
  Badge,
  ThemeToggle,
} from "@cloudoperators/juno-ui-components";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { useStore } from "./store";
import { AppRoutes } from "./Routes";
import { useVersion } from "./api";
import { NotificationBell } from "./components/NotificationBell";
import type { AppProps } from "./types";
// @ts-expect-error css import
import styles from "./styles.css";

const queryClient = new QueryClient({
  defaultOptions: {
    queries: { staleTime: 5 * 60 * 1000, retry: 1 },
  },
});

const HeaderContent = () => {
  const { data: version } = useVersion();
  const { archerctlUrl, docsUrl } = useStore((s) => s.globalAPI);

  return (
    <Stack gap="4" alignment="center">
      {docsUrl && (
        <a
          href={docsUrl}
          target="_blank"
          rel="noopener noreferrer"
          className="text-white hover:text-theme-accent flex items-center gap-1"
        >
          <Icon icon="openInNew" size="18" />
          <span>API</span>
        </a>
      )}
      {archerctlUrl && (
        <a
          href={archerctlUrl}
          target="_blank"
          rel="noopener noreferrer"
          className="text-white hover:text-theme-accent flex items-center gap-1"
        >
          <Icon icon="download" size="18" />
          <span>CLI</span>
        </a>
      )}
      {version && <Badge icon={false} text={version.version} />}
      <NotificationBell />
      <ThemeToggle />
    </Stack>
  );
};

const AppContent = ({
  canEdit,
  embedded,
  cloudAdmin,
}: {
  canEdit: boolean;
  embedded: boolean;
  cloudAdmin: boolean;
}) => {
  const apiReady = useStore((s) => s.globalAPI.apiReady);

  if (!apiReady) return <LoadingIndicator className="m-auto" />;
  return <AppRoutes canEdit={canEdit} embedded={embedded} cloudAdmin={cloudAdmin} />;
};

const App = (props: AppProps) => {
  const setGlobalAPI = useStore((s) => s.setGlobalAPI);
  const setToken = useStore((s) => s.setToken);
  const setApiReady = useStore((s) => s.setApiReady);
  const [tokenError, setTokenError] = useState(false);

  const canEdit = props.canEdit === true || props.canEdit === "true";
  const embedded = props.embedded === true || props.embedded === "true";
  const cloudAdmin = props.cloudAdmin === true || props.cloudAdmin === "true";

  useEffect(() => {
    setGlobalAPI({
      endpoint: props.endpoint || "",
      projectID: props.projectID || "",
      archerctlUrl: props.archerctlUrl || "https://github.com/sapcc/archer/releases/latest",
      docsUrl: props.docsUrl || "https://sapcc.github.io/archer/",
      theme: (props.theme as "theme-dark" | "theme-light") || "theme-dark",
      neutronEndpoint: props.neutronEndpoint || "",
      cloudAdmin,
    });
  }, [
    props.endpoint,
    props.projectID,
    props.archerctlUrl,
    props.docsUrl,
    props.theme,
    props.neutronEndpoint,
    cloudAdmin,
    setGlobalAPI,
  ]);

  useEffect(() => {
    const fetchToken = async () => {
      const name = props.getTokenFuncName;
      const fn = name ? (window as unknown as Record<string, unknown>)[name] : undefined;
      if (typeof fn === "function") {
        const result = await (fn as () => Promise<{ authToken: string }>)();
        setToken(result.authToken);
        setApiReady(true);
      } else if (props.token) {
        setToken(props.token);
        setApiReady(true);
      } else {
        setTokenError(true);
      }
    };
    fetchToken();
  }, [props.getTokenFuncName, props.token, setToken, setApiReady]);

  if (tokenError) {
    return <Message>No token provided.</Message>;
  }

  return (
    <QueryClientProvider client={queryClient}>
      <AppShell
        pageHeader={
          <PageHeader applicationName="Archer - Endpoint as a Service">
            <HeaderContent />
          </PageHeader>
        }
        embedded={embedded}
      >
        <AppContent canEdit={canEdit} embedded={embedded} cloudAdmin={cloudAdmin} />
      </AppShell>
    </QueryClientProvider>
  );
};

export const StyledApp = (props: AppProps) => (
  <AppShellProvider theme={(props.theme as "theme-dark" | "theme-light") || "theme-dark"}>
    <style>{styles.toString()}</style>
    <App {...props} />
  </AppShellProvider>
);

export default StyledApp;
