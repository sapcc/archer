// SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company
// SPDX-License-Identifier: Apache-2.0

module.exports = {
  presets: [require("@cloudoperators/juno-ui-components/build/lib/tailwind.config")],
  content: ["./src/**/*.{js,jsx,ts,tsx}", "./public/index.html"],
  corePlugins: {
    preflight: false,
  },
  prefix: "",
};
