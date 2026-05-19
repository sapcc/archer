# ArcherUI

React UI for **Archer** - an "Endpoint as a Service" API that privately connects services across OpenStack networks.

## Features

- **Services**: Create, view, edit, and delete private/public services
- **Endpoints**: Manage endpoints that connect to services
- **RBAC Policies**: Control service visibility across projects
- **Accept/Reject**: Approve or deny endpoint connection requests

## Quick Start

1. Install dependencies:

```bash
pnpm install
```

2. Create `appProps.json` in the root directory (copy from example):

```bash
cp appProps.example.json appProps.json
```

Then edit `appProps.json` with your configuration:

```json
{
  "endpoint": "https://archer-api.example.com/v1",
  "projectID": "your-project-id",
  "token": "your-keystone-token",
  "canEdit": true,
  "mockAPI": true,
  "theme": "theme-dark"
}
```

Set `mockAPI: true` to run with mock data (no real Archer backend needed).

3. Run the dev server:

```bash
APP_PORT=8000 pnpm start
```

4. Open http://localhost:8000

## Configuration

| Field       | Type    | Description                                   |
| ----------- | ------- | --------------------------------------------- |
| `endpoint`  | string  | Archer API base URL                           |
| `projectID` | string  | OpenStack project ID                          |
| `token`     | string  | Keystone auth token                           |
| `canEdit`   | boolean | Enable create/edit/delete actions             |
| `mockAPI`   | boolean | Use mock data (no real backend needed)        |
| `theme`     | string  | `theme-dark` or `theme-light`                 |
| `embedded`  | boolean | Hide page header when embedded in another app |

## Scripts

```bash
pnpm start          # Dev server with hot reload
pnpm build          # Production build to build/
pnpm typecheck      # TypeScript validation
pnpm format         # Prettier formatting
```

## Architecture

```
src/
├── index.tsx      # Mount function (entry point)
├── App.tsx        # Providers setup (QueryClient, AppShell)
├── Routes.tsx     # Tab navigation + React Router
├── store.ts       # Zustand state (API config, UI modals)
├── api.ts         # React Query hooks for all endpoints
├── types.ts       # TypeScript types from Archer API spec
├── components/    # UI components
└── mocks/         # MSW handlers for development
```

### API Endpoints Implemented

**Services**: `GET/POST /service`, `GET/PUT/DELETE /service/{id}`, `GET /service/{id}/endpoints`, `PUT /service/{id}/accept_endpoints`, `PUT /service/{id}/reject_endpoints`

**Endpoints**: `GET/POST /endpoint`, `GET/PUT/DELETE /endpoint/{id}`

**RBAC**: `GET/POST /rbac-policies`, `GET/PUT/DELETE /rbac-policies/{id}`

## Tech Stack

- React 19 + TypeScript
- Zustand (state management)
- TanStack Query (data fetching)
- React Router 7
- Juno UI Components
- Tailwind CSS
- esbuild (bundler)
- MSW (API mocking)
