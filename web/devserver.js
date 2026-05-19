// SPDX-FileCopyrightText: 2026 SAP SE
// SPDX-License-Identifier: Apache-2.0

// Dev server with proxy support for Neutron API (CORS bypass)
// Usage: node devserver.js

const http = require("node:http");
const https = require("node:https");
const fs = require("node:fs/promises");
const path = require("node:path");
const { spawn } = require("node:child_process");

const APP_PORT = parseInt(process.env.APP_PORT || "8000");
const ESBUILD_PORT = APP_PORT + 1;

// Read appProps to get neutronEndpoint
async function getNeutronEndpoint() {
  try {
    const content = await fs.readFile("./appProps.json", "utf-8");
    const props = JSON.parse(content);
    return props.neutronEndpoint || null;
  } catch {
    return null;
  }
}

// Proxy request to target URL
function proxyRequest(targetUrl, req, res) {
  const url = new URL(targetUrl);
  const isHttps = url.protocol === "https:";
  const client = isHttps ? https : http;

  const proxyReq = client.request(
    {
      hostname: url.hostname,
      port: url.port || (isHttps ? 443 : 80),
      path: url.pathname + url.search,
      method: req.method,
      headers: {
        ...req.headers,
        host: url.host,
      },
    },
    (proxyRes) => {
      // Add CORS headers
      res.writeHead(proxyRes.statusCode, {
        ...proxyRes.headers,
        "access-control-allow-origin": "*",
        "access-control-allow-methods": "GET, POST, PUT, DELETE, OPTIONS",
        "access-control-allow-headers": "X-Auth-Token, Content-Type, Accept",
      });
      proxyRes.pipe(res);
    }
  );

  proxyReq.on("error", (err) => {
    console.error("Proxy error:", err.message);
    res.writeHead(502);
    res.end(`Proxy error: ${err.message}`);
  });

  req.pipe(proxyReq);
}

// Forward request to esbuild dev server
function forwardToEsbuild(req, res) {
  const proxyReq = http.request(
    {
      hostname: "127.0.0.1",
      port: ESBUILD_PORT,
      path: req.url,
      method: req.method,
      headers: req.headers,
    },
    (proxyRes) => {
      res.writeHead(proxyRes.statusCode, proxyRes.headers);
      proxyRes.pipe(res);
    }
  );

  proxyReq.on("error", (err) => {
    res.writeHead(502);
    res.end(`esbuild server error: ${err.message}`);
  });

  req.pipe(proxyReq);
}

async function startServer() {
  const neutronEndpoint = await getNeutronEndpoint();

  if (neutronEndpoint) {
    console.log(`\x1b[36mNeutron proxy enabled: /proxy/neutron/* -> ${neutronEndpoint}/*\x1b[0m`);
  }

  // Start esbuild in background
  const esbuild = spawn("node", ["esbuild.config.js", "--serve", "--watch"], {
    env: { ...process.env, PORT: ESBUILD_PORT.toString(), NODE_ENV: "development" },
    stdio: "inherit",
  });

  esbuild.on("error", (err) => {
    console.error("Failed to start esbuild:", err);
    process.exit(1);
  });

  // Create proxy server
  const server = http.createServer(async (req, res) => {
    // Handle CORS preflight
    if (req.method === "OPTIONS") {
      res.writeHead(204, {
        "access-control-allow-origin": "*",
        "access-control-allow-methods": "GET, POST, PUT, DELETE, OPTIONS",
        "access-control-allow-headers": "X-Auth-Token, Content-Type, Accept",
        "access-control-max-age": "86400",
      });
      res.end();
      return;
    }

    // Proxy Neutron requests
    if (req.url.startsWith("/proxy/neutron/") && neutronEndpoint) {
      const targetPath = req.url.replace("/proxy/neutron", "");
      const targetUrl = neutronEndpoint + targetPath;
      console.log(`\x1b[35m[proxy]\x1b[0m ${req.method} ${targetUrl}`);
      proxyRequest(targetUrl, req, res);
      return;
    }

    // Forward everything else to esbuild
    forwardToEsbuild(req, res);
  });

  server.listen(APP_PORT, () => {
    console.log(`\x1b[32mDev server running at http://localhost:${APP_PORT}\x1b[0m`);
  });

  // Cleanup on exit
  process.on("SIGINT", () => {
    esbuild.kill();
    server.close();
    process.exit();
  });
}

startServer();
