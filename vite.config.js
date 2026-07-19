// PoC vite.config.js — mirrors the architecture of Gitea's real webpack->vite migration
// (go-gitea/gitea#37002): a single config, driven by `vite build` / `vite dev`, with a plugin
// that handles the classic (non-module) script entries — jQuery/fomantic bootstrap, standalone
// swagger/webcomponents pages, the notification SharedWorker — which can't be part of the main
// ES-module build (iife/UMD output doesn't support Rollup's code-splitting).
//
// Forgejo differs from Gitea here in one structural way: Gitea has exactly one classic entry
// (iife.ts). Forgejo's current (still-JS, not TS) app has five — the split from the old single
// `index` webpack entry (jquery+fomantic+bootstrap) plus four already-standalone classic pages —
// so `classicEntryPlugin` below is parameterized over a list instead of Gitea's single hardcoded
// `iifePlugin`. See modules/public/vitedev.go's `classicEntryVirtualPaths` map, which must stay in sync.
import {build, defineConfig} from 'vite';
import vuePlugin from '@vitejs/plugin-vue';
import {stringPlugin} from 'vite-string-plugin';
import licensePlugin from 'rollup-plugin-license';
import wrapAnsi from 'wrap-ansi';
import {readFileSync, writeFileSync, unlinkSync, globSync, mkdirSync} from 'node:fs';
import {join, parse} from 'node:path';
import {env} from 'node:process';
import tailwindcss from 'tailwindcss';
import tailwindConfig from './tailwind.config.js';
import tailwindcssNesting from 'tailwindcss/nesting/index.js';
import postcssNesting from 'postcss-nesting';

const isProduction = env.NODE_ENV !== 'development';
const root = import.meta.dirname;
const outDir = join(root, 'public/assets');

const webComponents = new Set([
  'i18n', 'overflow-menu', 'origin-url', 'absolute-date', 'relative-time',
  'markdown-toolbar', 'text-expander',
]);

// Classic (non-module) entries: name -> {entry, extraCss?}. `extraCss` covers old webpack
// array-entries that concatenated an unimported stylesheet alongside the JS (swagger.js /
// forgejoswagger.js both used to pull in css/standalone/swagger.css this way).
const CLASSIC_ENTRIES = {
  iife: {entry: 'web_src/js/iife.js'},
  webcomponents: {entry: 'web_src/js/webcomponents/index.js'},
  swagger: {entry: 'web_src/js/standalone/swagger.js', extraCss: 'web_src/css/standalone/swagger.css'},
  forgejoswagger: {entry: 'web_src/js/standalone/forgejo-swagger.js'},
  'eventsource.sharedworker': {entry: 'web_src/js/features/eventsource.sharedworker.js'},
};

const sharedPlugins = () => [
  stringPlugin(),
  vuePlugin({template: {compilerOptions: {isCustomElement: (tag) => webComponents.has(tag)}}}),
];

function commonViteOpts(buildExtra, {write} = {}) {
  return {
    root,
    base: './',
    configFile: false,
    publicDir: false,
    logLevel: 'warn',
    css: {
      postcss: {
        plugins: [
          tailwindcssNesting(postcssNesting({edition: '2024-02'})),
          tailwindcss(tailwindConfig),
        ],
      },
    },
    define: {
      __VUE_OPTIONS_API__: true,
      __VUE_PROD_DEVTOOLS__: false,
      __VUE_PROD_HYDRATION_MISMATCH_DETAILS__: false,
      // webpack's DefinePlugin/NodeJS shims used to replace this in third-party deps (e.g. tippy.js).
      'process.env.NODE_ENV': JSON.stringify(isProduction ? 'production' : 'development'),
    },
    plugins: sharedPlugins(),
    build: {
      outDir,
      emptyOutDir: false,
      manifest: true,
      target: 'es2020',
      // 'oxc' is Rolldown's own built-in minifier — no separate `esbuild` package dependency needed.
      minify: isProduction ? 'oxc' : false,
      cssMinify: isProduction,
      chunkSizeWarningLimit: Infinity,
      reportCompressedSize: false,
      ...(write === false && {write: false}),
      ...buildExtra,
    },
  };
}

// NOTE: iife/UMD output cannot code-split, which Rolldown enforces by rejecting multiple inputs
// for that format — so an entry's extra unimported CSS (old webpack array-entries used to
// concatenate one in, e.g. swagger.js + standalone/swagger.css) must be its own single-input
// build, never combined into the same classicBuildOpts() call as the JS entry.
function classicBuildOpts(name, entry, {write} = {}) {
  return commonViteOpts({
    rollupOptions: {
      input: {[name]: join(root, entry)},
      output: {
        format: 'iife',
        inlineDynamicImports: true, // fine for these low-traffic standalone entries
        entryFileNames: `js/${name}-[hash].js`,
        assetFileNames: `css/${name}-[hash].[ext]`,
      },
    },
  }, {write});
}

function cssOnlyBuildOpts(name, entry) {
  return commonViteOpts({
    rollupOptions: {
      input: {[name]: join(root, entry)},
      output: {assetFileNames: 'css/[name]-[hash].[ext]'},
    },
  });
}

// Builds every classic entry (+ its extra CSS, if any) once, merges their manifest.json fragments
// into the main build's manifest.json (each build() call below shares outDir, and each writes its
// own manifest.json — they never collide because manifest keys are the source file path relative
// to `root`).
function readManifestIfExists(manifestPath) {
  try {
    return JSON.parse(readFileSync(manifestPath, 'utf8'));
  } catch {
    return {}; // the main build's own manifest.json may not be written yet — plugin closeBundle
    // hooks across a Vite/Rolldown config aren't guaranteed to run strictly after the main
    // build's internal manifest-writing step just because this plugin appears later in the
    // `plugins` array, so this must not assume the file exists.
  }
}

async function buildAllClassicEntries() {
  const manifestPath = join(outDir, '.vite/manifest.json');
  const merged = readManifestIfExists(manifestPath);
  for (const [name, {entry, extraCss}] of Object.entries(CLASSIC_ENTRIES)) {
    await build(classicBuildOpts(name, entry));
    Object.assign(merged, readManifestIfExists(manifestPath));
    if (extraCss) {
      await build(cssOnlyBuildOpts(name, extraCss));
      Object.assign(merged, readManifestIfExists(manifestPath));
    }
  }
  // Merge on top of whatever is on disk *now* (the main build's manifest may have landed at any
  // point while the loop above was running), so it's never silently dropped from the final file.
  Object.assign(merged, readManifestIfExists(manifestPath));
  mkdirSync(join(outDir, '.vite'), {recursive: true});
  writeFileSync(manifestPath, JSON.stringify(merged, null, 2));
}

// Serves + rebuilds classic entries from memory in dev mode, and does the real (hashed, on-disk)
// build for them in production via closeBundle — mirrors Gitea's `iifePlugin`, generalized to N
// entries instead of Gitea's single hardcoded `iife.ts`.
function classicEntryPlugin() {
  const devCode = new Map(); // name -> {code, modules: Set<sourcePath>}
  let isBuilding = false;

  return {
    name: 'forgejo-classic-entries',
    async configureServer(server) {
      const buildAndCacheOne = async (name, {entry}) => {
        const result = await build(classicBuildOpts(name, entry, {write: false}));
        const output = (Array.isArray(result) ? result[0] : result).output;
        const chunk = output.find((o) => o.type === 'chunk' && o.isEntry);
        devCode.set(name, {
          code: chunk.code,
          modules: new Set(Object.keys(chunk.modules)),
        });
      };
      const buildAndCacheAll = async () => {
        for (const [name, opts] of Object.entries(CLASSIC_ENTRIES)) await buildAndCacheOne(name, opts);
      };
      await buildAndCacheAll();

      server.watcher.on('change', async (path) => {
        const affected = [...devCode.entries()].filter(([, v]) => v.modules.has(path)).map(([name]) => name);
        if (!affected.length || isBuilding) return;
        isBuilding = true;
        try {
          for (const name of affected) await buildAndCacheOne(name, CLASSIC_ENTRIES[name]);
          server.ws.send({type: 'full-reload'});
        } finally {
          isBuilding = false;
        }
      });

      server.middlewares.use((req, res, next) => {
        const pathname = req.url.split('?')[0];
        const match = pathname.match(/^\/web_src\/js\/__vite_classic_(.+)\.js$/);
        if (match && devCode.has(match[1])) {
          res.setHeader('Content-Type', 'application/javascript');
          res.setHeader('Cache-Control', 'no-store');
          res.end(devCode.get(match[1]).code);
        } else {
          next();
        }
      });
    },
    async closeBundle() {
      if (!isProduction) return; // production-only: `vite build`, not `vite dev`
      for (const file of globSync('js/{iife,webcomponents,swagger,forgejoswagger,eventsource.sharedworker}-*.js*', {cwd: outDir})) {
        unlinkSync(join(outDir, file));
      }
      await buildAllClassicEntries();
    },
  };
}

const viteDevServerPort = Number(env.FRONTEND_DEV_SERVER_PORT) || 3001;
const viteDevPortFilePath = join(outDir, '.vite/dev-port');

// Writes the Vite dev server's actual port to a file so the Go server can discover it and proxy
// requests to it (modules/public/vitedev.go). Mirrors Gitea's viteDevServerPortPlugin.
function viteDevServerPortPlugin() {
  return {
    name: 'forgejo-vite-dev-server-port',
    apply: 'serve',
    configureServer(server) {
      server.httpServer.once('listening', () => {
        const addr = server.httpServer.address();
        if (typeof addr === 'object' && addr) {
          mkdirSync(join(outDir, '.vite'), {recursive: true});
          writeFileSync(viteDevPortFilePath, String(addr.port));
        }
      });
    },
  };
}

function formatLicenseText(licenseText) {
  return wrapAnsi(licenseText || '', 80).trim();
}

// Generates public/assets/licenses.txt from the JS dependency tree (production) plus the Go module
// licenses already collected into assets/go-licenses.json by `make generate-license`. In dev mode,
// skip the (slow) license scan and write a stub instead — mirrors the old webpack.config.js split.
function licenseTextPlugin() {
  if (!isProduction) {
    return {
      name: 'forgejo-dev-licenses-stub',
      closeBundle() {
        writeFileSync(join(outDir, 'licenses.txt'), 'Licenses are disabled during development');
      },
    };
  }
  return licensePlugin({
    thirdParty: {
      output: {
        file: join(outDir, 'licenses.txt'),
        template(deps) {
          const line = '-'.repeat(80);
          const goJson = readFileSync(join(root, 'assets/go-licenses.json'), 'utf8');
          const goModules = JSON.parse(goJson).map(({name, licenseText}) => ({name, body: formatLicenseText(licenseText)}));
          const jsModules = deps.map((dep) => ({name: dep.name, version: dep.version, body: formatLicenseText(dep.licenseText ?? '')}));
          const modules = [...goModules, ...jsModules].sort((a, b) => a.name.localeCompare(b.name));
          return modules.map(({name, version, body}) => {
            const title = version ? `${name}@${version}` : name;
            return `${line}\n${title}\n${line}\n${body}`;
          }).join('\n');
        },
      },
      allow(dependency) {
        // argparse@2.0.1 - Python-2.0. It's used in the CLI file of markdown-it and js-yaml and not in the library code.
        // idiomorph@0.3.0. See https://github.com/bigskysoftware/idiomorph/pull/37
        if (['argparse', 'idiomorph'].includes(dependency.name)) return true;
        return /(Apache-2\.0|0BSD|BSD-2-Clause|BSD-3-Clause|BlueOak-1\.0\.0|MIT|ISC|Unlicense|CC-BY-4\.0)/.test(dependency.license ?? '');
      },
    },
  });
}

const themes = {};
for (const path of globSync('web_src/css/themes/*.css', {cwd: root})) {
  themes[parse(path).name] = join(root, path);
}

const mainOpts = commonViteOpts({
  modulePreload: false,
  rollupOptions: {
    input: {
      index: join(root, 'web_src/js/index.js'),
      ...themes,
    },
    output: {
      entryFileNames: 'js/[name]-[hash].js',
      chunkFileNames: 'js/[name]-[hash].js',
      assetFileNames: (info) => {
        const name = info.names?.[0] ?? info.name ?? '';
        if (name.endsWith('.css')) return 'css/[name]-[hash].[ext]';
        if (/\.(ttf|woff2?)$/.test(name)) return 'fonts/[name]-[hash].[ext]';
        return '[name]-[hash].[ext]';
      },
    },
  },
});

export default defineConfig({
  ...mainOpts,
  appType: 'custom', // Go serves all HTML, disable Vite's HTML handling
  clearScreen: false,
  server: {
    port: viteDevServerPort,
    open: false,
    host: '0.0.0.0',
    strictPort: false,
    fs: {
      // VITE-DEV-SERVER-SECURITY: the dev server is exposed to the public by Forgejo's web server,
      // so access must be strictly limited. Otherwise `/@fs/*` could serve any file (including
      // app.ini, which contains INTERNAL_TOKEN).
      strict: true,
      allow: ['assets', 'node_modules', 'public', 'web_src'],
    },
    headers: {
      'Cache-Control': 'no-store',
    },
  },
  plugins: [
    classicEntryPlugin(),
    viteDevServerPortPlugin(),
    licenseTextPlugin(),
    ...sharedPlugins(),
  ],
});
