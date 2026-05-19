// SPDX-FileCopyrightText: 2026 SAP SE
// SPDX-License-Identifier: Apache-2.0

import { lazy, Suspense } from "react";
import {
  Container,
  TabNavigation,
  TabNavigationItem,
  Stack,
  Icon,
  Badge,
  LoadingIndicator,
} from "@cloudoperators/juno-ui-components";
import { HashRouter, Routes, Route, Navigate, useLocation, useNavigate, Outlet } from "react-router";
import { ServiceList } from "./components/ServiceList";
import { EndpointList } from "./components/EndpointList";
import { RBACList } from "./components/RBACList";
import { AgentList } from "./components/AgentList";
import { useVersion } from "./api";
import { useStore } from "./store";

const NetworkGraph = lazy(() => import("./components/NetworkGraph").then((m) => ({ default: m.NetworkGraph })));

const Nav = ({ embedded, cloudAdmin }: { embedded: boolean; cloudAdmin: boolean }) => {
  const loc = useLocation();
  const nav = useNavigate();
  const tab = loc.pathname.split("/")[1] || "endpoints";
  const { data: version } = useVersion();
  const { archerctlUrl, docsUrl } = useStore((s) => s.globalAPI);

  return (
    <Stack distribution="between" alignment="center">
      <TabNavigation activeItem={tab}>
        <TabNavigationItem label="Endpoints" value="endpoints" onClick={() => nav("/endpoints")} icon="place" />
        <TabNavigationItem label="Services" value="services" onClick={() => nav("/services")} icon="widgets" />
        <TabNavigationItem label="RBAC" value="rbac" onClick={() => nav("/rbac")} icon="manageAccounts" />
        <TabNavigationItem label="Topology" value="topology" onClick={() => nav("/topology")} icon="dns" />
        {cloudAdmin && (
          <TabNavigationItem
            label="Agents"
            value="agents"
            onClick={() => nav("/agents")}
            icon="autoAwesomeMotion"
            className="text-theme-danger"
          />
        )}
      </TabNavigation>
      {embedded && (
        <Stack gap="4" alignment="center">
          {docsUrl && (
            <a
              href={docsUrl}
              target="_blank"
              rel="noopener noreferrer"
              className="text-theme-accent hover:underline flex items-center gap-1"
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
              className="text-theme-accent hover:underline flex items-center gap-1"
            >
              <Icon icon="download" size="18" />
              <span>CLI</span>
            </a>
          )}
          {version && <Badge icon={false} text={version.version} />}
        </Stack>
      )}
    </Stack>
  );
};

const Layout = ({ canEdit, embedded, cloudAdmin }: { canEdit: boolean; embedded: boolean; cloudAdmin: boolean }) => (
  <div className="flex flex-col gap-4">
    <Nav embedded={embedded} cloudAdmin={cloudAdmin} />
    <Outlet context={{ canEdit }} />
  </div>
);

export const AppRoutes = ({
  canEdit,
  embedded,
  cloudAdmin,
}: {
  canEdit: boolean;
  embedded: boolean;
  cloudAdmin: boolean;
}) => (
  <Container px={false}>
    <HashRouter>
      <Routes>
        <Route element={<Layout canEdit={canEdit} embedded={embedded} cloudAdmin={cloudAdmin} />}>
          <Route index element={<Navigate to="/endpoints" replace />} />
          <Route path="/services/*" element={<ServiceList canEdit={canEdit} cloudAdmin={cloudAdmin} />} />
          <Route path="/endpoints/*" element={<EndpointList canEdit={canEdit} cloudAdmin={cloudAdmin} />} />
          <Route path="/rbac/*" element={<RBACList canEdit={canEdit} />} />
          <Route
            path="/topology"
            element={
              <Suspense fallback={<LoadingIndicator className="m-auto" />}>
                <NetworkGraph />
              </Suspense>
            }
          />
          {cloudAdmin && <Route path="/agents" element={<AgentList />} />}
        </Route>
      </Routes>
    </HashRouter>
  </Container>
);
