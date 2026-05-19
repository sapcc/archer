// SPDX-FileCopyrightText: 2026 SAP SE
// SPDX-License-Identifier: Apache-2.0

import { create } from "zustand";
import { devtools } from "zustand/middleware";
import type { Service, Endpoint, RBACPolicy, Provider } from "./types";

interface GlobalAPI {
  apiReady: boolean;
  endpoint: string;
  token: string;
  projectID: string;
  archerctlUrl: string;
  docsUrl: string;
  theme: "theme-dark" | "theme-light";
  neutronEndpoint: string;
  cloudAdmin: boolean;
}

interface DeleteTarget {
  type: "service" | "endpoint" | "rbac";
  item: Service | Endpoint | RBACPolicy;
}

export interface ToastMessage {
  id: string;
  variant: "success" | "warning" | "danger";
  message: string;
  timestamp: number;
}

interface AppState {
  // API state
  globalAPI: GlobalAPI;
  setGlobalAPI: (api: Partial<GlobalAPI>) => void;
  setToken: (token: string) => void;
  setApiReady: (ready: boolean) => void;

  // Toast state
  toasts: ToastMessage[];
  addToast: (toast: Omit<ToastMessage, "id" | "timestamp">) => void;
  dismissToast: (id: string) => void;
  clearToasts: () => void;

  // UI state
  showServiceModal: boolean;
  showEndpointModal: boolean;
  showRBACModal: boolean;
  showDeleteModal: boolean;
  showMigrateModal: boolean;
  editService: Service | null;
  editEndpoint: Endpoint | null;
  editRBAC: RBACPolicy | null;
  deleteTarget: DeleteTarget | null;
  preselectedServiceId: string | null;
  createServiceProvider: Provider | null;
  migrateService: Service | null;

  // UI actions
  openServiceModal: (service?: Service | null, provider?: Provider | null) => void;
  openEndpointModal: (endpoint?: Endpoint | null, preselectedServiceId?: string | null) => void;
  openRBACModal: (policy?: RBACPolicy | null) => void;
  openDeleteModal: (target: DeleteTarget) => void;
  openMigrateModal: (service: Service) => void;
  closeModals: () => void;
}

export const useStore = create<AppState>()(
  devtools((set) => ({
    // API state
    globalAPI: {
      apiReady: false,
      endpoint: "",
      token: "",
      projectID: "",
      archerctlUrl: "",
      docsUrl: "",
      theme: "theme-dark",
      neutronEndpoint: "",
      cloudAdmin: false,
    },
    setGlobalAPI: (api) => set((state) => ({ globalAPI: { ...state.globalAPI, ...api } })),
    setToken: (token) => set((state) => ({ globalAPI: { ...state.globalAPI, token } })),
    setApiReady: (apiReady) => set((state) => ({ globalAPI: { ...state.globalAPI, apiReady } })),

    // Toast state
    toasts: [],
    addToast: (toast) =>
      set((state) => ({
        toasts: [{ ...toast, id: `${Date.now()}-${Math.random()}`, timestamp: Date.now() }, ...state.toasts],
      })),
    dismissToast: (id) => set((state) => ({ toasts: state.toasts.filter((t) => t.id !== id) })),
    clearToasts: () => set({ toasts: [] }),

    // UI state
    showServiceModal: false,
    showEndpointModal: false,
    showRBACModal: false,
    showDeleteModal: false,
    showMigrateModal: false,
    editService: null,
    editEndpoint: null,
    editRBAC: null,
    deleteTarget: null,
    preselectedServiceId: null,
    createServiceProvider: null,
    migrateService: null,

    // UI actions
    openServiceModal: (service = null, provider = null) =>
      set({ showServiceModal: true, editService: service, createServiceProvider: provider }),
    openEndpointModal: (endpoint = null, preselectedServiceId = null) =>
      set({ showEndpointModal: true, editEndpoint: endpoint, preselectedServiceId }),
    openRBACModal: (policy = null) => set({ showRBACModal: true, editRBAC: policy }),
    openDeleteModal: (target) => set({ showDeleteModal: true, deleteTarget: target }),
    openMigrateModal: (service) => set({ showMigrateModal: true, migrateService: service }),
    closeModals: () =>
      set({
        showServiceModal: false,
        showEndpointModal: false,
        showRBACModal: false,
        showDeleteModal: false,
        showMigrateModal: false,
        editService: null,
        editEndpoint: null,
        editRBAC: null,
        deleteTarget: null,
        preselectedServiceId: null,
        createServiceProvider: null,
        migrateService: null,
      }),
  }))
);
