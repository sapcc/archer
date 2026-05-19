# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

ArcherUI is a TypeScript React application for managing Archer "Endpoint as a Service" resources. It allows users to create services, endpoints, and RBAC policies that privately connect OpenStack networks.

## Commands

```bash
pnpm install                  # Install dependencies
APP_PORT=8000 pnpm start      # Dev server at localhost:8000
pnpm build                    # Production build
pnpm typecheck                # TypeScript validation
pnpm format                   # Prettier formatting
```

## Architecture

### Entry Point

- `src/index.tsx` exports `mount(container, { props })` for embedding
- Enables MSW mocking when `props.mockAPI` is true
- Lazy-loads `App.tsx` for code splitting

### State Management

- Single Zustand store in `src/store.ts`
- `globalAPI`: endpoint, token, projectID, apiReady flag
- UI state: modal visibility, edit targets, delete targets
- Access via `useStore()` hook directly (no context needed)

### API Layer

- All React Query hooks in `src/api.ts`
- Hooks read endpoint/token from Zustand store
- Pattern: `useServices()`, `useCreateService()`, `useDeleteService()`, etc.
- Automatic cache invalidation on mutations

### Routing

- HashRouter in `src/Routes.tsx`
- Three tabs: Services, Endpoints, RBAC
- Detail panels open as nested routes (e.g., `/services/:id`)

### Types

- All API types in `src/types.ts`, derived from `swagger.yaml`
- Key types: `Service`, `Endpoint`, `RBACPolicy`, `AppProps`

## Key Patterns

### Adding a new API endpoint

1. Add types to `src/types.ts`
2. Add query/mutation hook to `src/api.ts`
3. Use the hook in components

### Adding a new view

1. Create component in `src/components/`
2. Add route in `src/Routes.tsx`
3. Add tab in the `Nav` component

### Modal pattern

```tsx
// Open modal
openServiceModal(serviceToEdit); // or null for create

// In component, check store
const { showServiceModal, editService, closeModals } = useStore();
if (showServiceModal) return <ServiceForm />;
```

## Testing with Mock API

Set `mockAPI: true` in `appProps.json`. Mock handlers are in `src/mocks/handlers.ts` with fixture data inline.

**Note:** Changes to `appProps.json` require restarting the dev server—they are not auto-detected.

## File Structure

```
src/
├── index.tsx          # Mount function
├── App.tsx            # QueryClient + AppShell providers
├── Routes.tsx         # Navigation + routing
├── store.ts           # Zustand store
├── api.ts             # React Query hooks
├── types.ts           # TypeScript types
├── styles.css         # Tailwind import
├── components/
│   ├── ServiceList.tsx, ServiceDetail.tsx, ServiceForm.tsx
│   ├── EndpointList.tsx, EndpointDetail.tsx, EndpointForm.tsx
│   ├── RBACList.tsx, RBACForm.tsx
│   ├── StatusBadge.tsx
│   └── DeleteModal.tsx
└── mocks/
    ├── browser.ts     # MSW setup
    └── handlers.ts    # Mock API responses
```
