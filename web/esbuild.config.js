// SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company
// SPDX-License-Identifier: Apache-2.0

const esbuild = require("esbuild");
const fs = require("node:fs/promises");
const pkg = require("./package.json");
const postcss = require("postcss");
const sass = require("sass");
const { transform } = require("@svgr/core");
const tailwindcss = require("@tailwindcss/postcss");
const autoprefixer = require("autoprefixer");

if (!/.+\/.+\.js/.test(pkg.module)) throw new Error("module value is incorrect, use DIR/FILE.js like build/index.js");

const isProduction = process.env.NODE_ENV === "production";
const IGNORE_EXTERNALS = process.env.IGNORE_EXTERNALS === "true";
let outfile = `${isProduction ? "" : "public/"}${pkg.main || pkg.module}`;
let outdir = outfile.slice(0, outfile.lastIndexOf("/"));
const args = process.argv.slice(2);
const watch = args.indexOf("--watch") >= 0;
const serve = args.indexOf("--serve") >= 0;

const green = "\x1b[32m%s\x1b[0m";
const yellow = "\x1b[33m%s\x1b[0m";
const clear = "\033c";

const build = async () => {
  await fs.rm(outdir, { recursive: true, force: true });
  await fs.mkdir(outdir, { recursive: true });

  let ctx = await esbuild.context({
    bundle: true,
    minify: isProduction,
    treeShaking: true,
    legalComments: isProduction ? "none" : "inline",
    drop: isProduction ? ["console", "debugger"] : [],
    target: ["es2022"],
    format: "esm",
    platform: "browser",
    loader: { ".ts": "ts", ".tsx": "tsx" },
    sourcemap: !isProduction,
    external: isProduction && !IGNORE_EXTERNALS ? Object.keys(pkg.peerDependencies || {}) : [],
    entryPoints: [pkg.source],
    outdir,
    splitting: true,
    metafile: isProduction,
    plugins: [
      {
        name: "start/end",
        setup(build) {
          build.onStart(() => {
            console.log(clear);
            console.log(yellow, "Compiling...");
          });
          build.onEnd(() => console.log(green, "Done!"));
        },
      },

      {
        name: "svg-loader",
        setup(build) {
          build.onLoad(
            { filter: /.\.(svg)$/, namespace: "file" },
            async (args) => {
              let contents = await fs.readFile(args.path);
              let loader = "text";
              if (args.suffix === "?url") {
                const maxSize = 10240;
                loader = contents.length <= maxSize ? "dataurl" : "file";
              } else {
                loader = "jsx";
                contents = await transform(contents, {
                  plugins: ["@svgr/plugin-jsx"],
                });
              }
              return { contents, loader };
            }
          );
        },
      },

      {
        name: "image-loader",
        setup(build) {
          build.onLoad(
            { filter: /.\.(png|jpg|jpeg|gif)$/, namespace: "file" },
            async (args) => {
              let contents = await fs.readFile(args.path);
              const maxSize = 10240;
              const loader = contents.length <= maxSize ? "dataurl" : "file";
              return { contents, loader };
            }
          );
        },
      },

      {
        name: "parse-styles",
        setup(build) {
          const postcssProcessor = postcss([tailwindcss, autoprefixer]);

          build.onLoad(
            { filter: /.\.(css|scss)$/, namespace: "file" },
            async (args) => {
              let content;
              if (args.path.endsWith(".scss")) {
                const result = await sass.compileAsync(args.path);
                content = result.css;
              } else {
                content = await fs.readFile(args.path);
              }

              const { css } = await postcssProcessor.process(content, {
                from: args.path,
                to: outdir,
              });
              return { contents: css, loader: "text" };
            }
          );
        },
      },
    ],
  });

  if (watch || serve) {
    if (watch) await ctx.watch();
    if (serve) {
      await fs.copyFile(`./appProps.json`, `./public/build/appProps.json`);

      let { host, port } = await ctx.serve({
        host: "0.0.0.0",
        port: parseInt(process.env.PORT),
        servedir: "public",
      });
      console.log("serve on", `${host}:${port}`);
    }
  } else {
    const result = await ctx.rebuild();
    if (isProduction && result.metafile) {
      await fs.writeFile(`${outdir}/meta.json`, JSON.stringify(result.metafile));
      const analysis = await esbuild.analyzeMetafile(result.metafile, {
        verbose: false,
      });
      console.log(analysis);
    }
    await ctx.dispose();
  }
};

build();
