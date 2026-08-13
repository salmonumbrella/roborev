# Web Foundation and Browser Security Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use
> superpowers:subagent-driven-development (recommended) or
> superpowers:executing-plans to implement this plan task-by-task. Steps use
> checkbox (`- [ ]`) syntax for tracking.

**Goal:** Establish Roborev's Bun/Svelte workspace and make the daemon serve a
minimal embedded application through a separate, authenticated browser listener.

**Architecture:** A focused `internal/web` package owns validated embedded
assets and hardened static serving. `internal/daemon` owns browser endpoint
resolution, exact Host/Origin policy, process-local sessions, the browser API
allowlist, listener lifecycle, and runtime advertisement. The existing CLI API
listener remains unchanged. A minimal Svelte shell exercises the real login and
fresh-tab bootstrap contract before review code is transplanted.

**Tech Stack:** Go 1.26, Huma/OpenAPI, Svelte 5, TypeScript 5.9, Vite 8, Vitest
4, Bun 1.3.14, SQLite test fixtures, Cobra, and Go `embed`.

## Global Constraints

- Keep the existing CLI listener loopback/Unix-only and do not add browser
    routes, static fallback, authentication, or shutdown changes to it.
- The browser listener is separate. It defaults to enabled on `127.0.0.1:0` and
    may be disabled with `web.enabled = false`.
- A non-loopback browser bind requires a non-empty authentication token and an
    HTTPS `web.public_origin`.
- Host validation runs before authentication. `Forwarded`, `X-Forwarded-*`, and
    `X-Real-IP` never establish authority or local trust.
- Tokenless local sessions require a direct loopback peer, an accepted loopback
    Host and exact Origin, no forwarding headers, and no configured remote
    token.
- The remote token appears only in the login request. Do not place it in URLs,
    runtime metadata, logs, cookies, local storage, or session storage.
- Store the tab token and CSRF token only in browser `sessionStorage`.
- Browser sessions are process-local and intentionally expire on daemon restart.
    Do not add persistence.
- Use fetch-based streaming in all later consumers. Do not introduce native
    `EventSource`.
- Use this Content Security Policy exactly:
    `default-src 'self'; base-uri 'none'; object-src 'none'; frame-ancestors 'none'; form-action 'self'; script-src 'self' 'wasm-unsafe-eval'; style-src 'self'; style-src-attr 'unsafe-inline'; style-src-elem 'self' 'unsafe-inline'; img-src 'self' data:; font-src 'self'; connect-src 'self'`.
- Keep `internal/web/dist/index.html` as the only tracked distribution stub.
    Never commit generated `web/dist` or staged production assets.
- Release builds remain `CGO_ENABLED=0` and must fail when the embedded assets
    are the compilation stub or the Vite manifest is incomplete.
- The development target must use a temporary `ROBOREV_DATA_DIR`, temporary
    database, explicit loopback Vite origin, and synthetic data. It must not
    read or mutate the normal daemon database or runtime files.
- Use `testify` for Go assertions and the repository's isolated test helpers.
- Public docs, fixtures, screenshots, commits, and pull-request text must use
    generic deployment language and synthetic identifiers. Do not identify any
    external private project or infrastructure.
- Do not run `make install`, `go install ./cmd/roborev`, or a live release
    publication command while executing this plan.

______________________________________________________________________

## File map

### Frontend workspace

- `package.json` and `bun.lock`: root workspace and pinned package manager.
- `web/package.json`: application scripts and pinned frontend dependencies.
- `web/index.html`: production distribution marker and Vite entry point.
- `web/src/main.ts`: Svelte mount only.
- `web/src/App.svelte`: minimal authentication/bootstrap shell.
- `web/src/app.css`: initial Roborev tokens and foundation layout.
- `web/src/lib/api/generated.ts`: generated OpenAPI types.
- `web/src/lib/api/session.ts`: cookie-based login/bootstrap client and
    `sessionStorage` tab credentials.
- `web/scripts/generate-api.ts`: deterministic TypeScript generation and drift
    check.
- `web/scripts/validate-assets.ts`: Vite manifest and path validation.
- `web/scripts/embed-assets.ts`: transactional staging into `internal/web/dist`
    and compilation-stub restoration.
- `web/scripts/dev.ts`: isolated disposable daemon plus Vite orchestration.
- `web/vite.config.ts`: strict loopback dev server and `/api` proxy.
- `web/vitest.config.ts`, `web/tsconfig.json`, `web/svelte.config.js`, and
    `web/eslint.config.js`: checks used locally and in continuous integration.
- `packages/roborev-ui/`: source-shipping package skeleton and archive contract.

### Embedded web package

- `internal/web/dist/index.html`: harmless tracked compilation stub.
- `internal/web/embed.go`: `embed.FS` and release validation entry points.
- `internal/web/assets.go`: safe Vite manifest catalog and production marker.
- `internal/web/handler.go`: static assets, navigation fallback, cache headers,
    MIME types, and CSP.
- `internal/web/*_test.go`: in-memory distribution and handler behavior.

### Browser daemon track

- `internal/config/config.go`: `[web]` configuration and defaults.
- `internal/config/config_test.go`: normalization, masking, and unsafe exposure
    rejection.
- `internal/daemon/browser_endpoint.go`: bind address and canonical origin.
- `internal/daemon/browser_policy.go`: exact Host/Origin and local-trust policy.
- `internal/daemon/browser_session.go`: ambient and tab session state.
- `internal/daemon/browser_routes.go`: browser-only Huma routes and handlers.
- `internal/daemon/browser_handler.go`: allowlist, auth/CSRF middleware, and
    static fallback composition.
- `internal/daemon/browser_server.go`: listener start, readiness, and shutdown.
- `internal/daemon/routes.go`: split core/browser route registration while
    preserving one canonical OpenAPI document.
- `internal/daemon/server.go`: attach the browser lifecycle to the daemon.
- `internal/daemon/runtime.go`: advertise the non-secret browser origin and
    capabilities.
- `cmd/roborev/ui_cmd.go`: safe browser launch command.
- `cmd/roborev/open_browser.go`: platform opener selected behind a test seam.
- `cmd/roborev/main.go`: register `roborev ui`.
- `cmd/roborev/daemon_cmd.go`: hidden explicit development-origin flag.

### Build, release, and docs

- `Makefile`: frontend checks, build/embed/restore, isolated dev, and release
    verification targets.
- `.gitignore`: ignore generated frontend and staged embed output while keeping
    the compilation stub.
- `.github/workflows/ci.yml`: frontend path detection and checks.
- `.github/workflows/release.yml`: install pinned Bun dependencies before
    GoReleaser.
- `.goreleaser.yaml`: validate embedded production assets before compiling.
- `docs/configuration.md`, `docs/commands.md`, and `docs/development.md`:
    first-release operation and development contract.

______________________________________________________________________

### Task 1: Create the Bun/Svelte workspace and package archive boundary

**Files:**

- Create: `package.json`
- Create: `bun.lock` (generated by Bun)
- Create: `web/package.json`
- Create: `web/tsconfig.json`
- Create: `web/svelte.config.js`
- Create: `web/vite.config.ts`
- Create: `web/vitest.config.ts`
- Create: `web/eslint.config.js`
- Create: `web/index.html`
- Create: `web/src/main.ts`
- Create: `web/src/App.svelte`
- Create: `web/src/app.d.ts`
- Create: `web/src/app.css`
- Create: `web/src/App.test.ts`
- Create: `web/src/test/setup.ts`
- Create: `packages/roborev-ui/package.json`
- Create: `packages/roborev-ui/tsconfig.json`
- Create: `packages/roborev-ui/README.md`
- Create: `packages/roborev-ui/src/index.ts`
- Create: `packages/roborev-ui/pack.test.ts`
- Modify: `.gitignore`

**Interfaces:**

- Produces: root workspaces `web` and `packages/*`.

- Produces: application commands `web:check`, `web:test`, and `web:build`.

- Produces: `@kenn-io/roborev-ui` as a source-only package with `src/index.ts`
    as its only initial runtime export.

- [ ] **Step 1: Add the package archive test first**

Create `packages/roborev-ui/pack.test.ts` with a real archive inspection:

```ts
import { execFileSync } from "node:child_process";
import { mkdtempSync, readdirSync, rmSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";

import { afterEach, expect, test } from "vitest";

const temporaryDirectories: string[] = [];

afterEach(() => {
  for (const directory of temporaryDirectories.splice(0)) {
    rmSync(directory, { recursive: true, force: true });
  }
});

test("the package ships only its supported source contract", () => {
  const destination = mkdtempSync(join(tmpdir(), "roborev-ui-pack-"));
  temporaryDirectories.push(destination);
  execFileSync("bun", ["pm", "pack", "--destination", destination], {
    cwd: import.meta.dirname,
    stdio: "pipe",
  });
  const archives = readdirSync(destination).filter((name) => name.endsWith(".tgz"));
  expect(archives).toHaveLength(1);

  const entries = execFileSync("tar", ["-tf", join(destination, archives[0]!)], {
    encoding: "utf8",
  }).trim().split("\n").filter(Boolean).sort();

  expect(entries).toEqual([
    "package/README.md",
    "package/package.json",
    "package/src/index.ts",
  ]);
});
```

- [ ] **Step 2: Run the archive test and verify the missing workspace fails**

Run: `bun x vitest run packages/roborev-ui/pack.test.ts`

Expected: FAIL because the root workspace and package manifest do not exist.

- [ ] **Step 3: Add the pinned workspace manifests**

Create the root `package.json`:

```json
{
  "private": true,
  "workspaces": ["web", "packages/*"],
  "packageManager": "bun@1.3.14",
  "scripts": {
    "web:check": "bun run --cwd web check",
    "web:test": "bun run --cwd web test && bun run --cwd packages/roborev-ui test",
    "web:build": "bun run --cwd web build"
  }
}
```

Create `packages/roborev-ui/package.json` with source exports and a strict
allowlist:

```json
{
  "name": "@kenn-io/roborev-ui",
  "version": "0.1.0",
  "type": "module",
  "license": "MIT",
  "svelte": "src/index.ts",
  "types": "src/index.ts",
  "exports": {
    ".": {
      "svelte": "./src/index.ts",
      "types": "./src/index.ts",
      "default": "./src/index.ts"
    }
  },
  "files": ["README.md", "src/index.ts"],
  "scripts": {
    "check": "tsc --noEmit",
    "test": "vitest run pack.test.ts"
  },
  "devDependencies": {
    "@types/node": "24.10.0",
    "typescript": "5.9.3",
    "vitest": "4.1.10"
  },
  "peerDependencies": {
    "svelte": "^5.56.4"
  },
  "packageManager": "bun@1.3.14"
}
```

Keep `packages/roborev-ui/src/index.ts` deliberately empty except for
`export {};`. The presentation API is introduced by the projection/package plan,
not invented during foundation work.

- [ ] **Step 4: Add the minimal Svelte application**

Use these exact runtime pins in `web/package.json`:

```json
{
  "name": "@kenn-io/roborev-web",
  "private": true,
  "version": "0.0.0",
  "type": "module",
  "packageManager": "bun@1.3.14",
  "scripts": {
    "typecheck": "tsc --noEmit",
    "check:svelte": "svelte-check --tsconfig ./tsconfig.json --fail-on-warnings",
    "lint": "eslint . --max-warnings 0",
    "format": "prettier --write .",
    "format:check": "prettier --check .",
    "check": "bun run typecheck && bun run check:svelte && bun run lint && bun run format:check",
    "test": "vitest run",
    "build": "vite build",
    "dev": "vite"
  },
  "dependencies": {
    "@kenn-io/roborev-ui": "workspace:*",
    "openapi-fetch": "0.17.0",
    "svelte": "5.56.4"
  },
  "devDependencies": {
    "@eslint/js": "9.39.2",
    "@sveltejs/vite-plugin-svelte": "7.1.2",
    "@testing-library/jest-dom": "6.9.1",
    "@testing-library/svelte": "5.4.2",
    "@types/node": "24.10.0",
    "eslint": "9.39.2",
    "eslint-plugin-svelte": "3.15.0",
    "globals": "16.5.0",
    "jsdom": "29.1.1",
    "openapi-typescript": "7.13.0",
    "prettier": "3.8.3",
    "prettier-plugin-svelte": "3.4.1",
    "svelte-check": "4.6.0",
    "typescript": "5.9.3",
    "typescript-eslint": "8.54.0",
    "vite": "8.1.3",
    "vitest": "4.1.10"
  }
}
```

Use this package compiler contract in `packages/roborev-ui/tsconfig.json`:

```json
{
  "compilerOptions": {
    "target": "ES2022",
    "module": "ESNext",
    "moduleResolution": "Bundler",
    "allowImportingTsExtensions": true,
    "noEmit": true,
    "strict": true,
    "skipLibCheck": true,
    "types": ["node"]
  },
  "include": ["src/**/*.ts", "pack.test.ts"]
}
```

The package README states that this is a source-shipping, network-free Svelte
presentation package and that consumers compile `src/index.ts`. Do not promise
review components before the extraction plan adds them.

`web/index.html` must contain the release marker:

```html
<!doctype html>
<html lang="en">
  <head>
    <meta charset="UTF-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1.0" />
    <meta name="roborev-web-distribution" content="production" />
    <title>Roborev</title>
  </head>
  <body>
    <div id="app"></div>
    <script type="module" src="/src/main.ts"></script>
  </body>
</html>
```

`App.svelte` initially renders a landmark, heading, and the text
`Browser foundation ready`. `App.test.ts` renders it and asserts the heading by
accessible role. Configure Vite to bind only `127.0.0.1`, use port 5173 with
`strictPort: true`, emit `.vite/manifest.json`, and proxy `/api` only when
`ROBOREV_WEB_DEV_BACKEND` is set, with `changeOrigin: false`.

Use a strict, no-emit `web/tsconfig.json` with `target: "ES2022"`,
`module: "ESNext"`, `moduleResolution: "Bundler"`, `strict: true`,
`isolatedModules: true`, `resolveJsonModule: true`, and types for `vite/client`
and `vitest/globals`. Include `src/**/*.ts`, `src/**/*.svelte`,
`scripts/**/*.ts`, and the TypeScript config files. `svelte.config.js` uses
`vitePreprocess()`. `vitest.config.ts` uses the Svelte plugin, `jsdom`, and
`setupFiles: ["./src/test/setup.ts"]`; that setup imports
`@testing-library/jest-dom/vitest` and calls Testing Library cleanup after each
test.

Build `web/eslint.config.js` from `@eslint/js` recommended, `typescript-eslint`
recommended, and `eslint-plugin-svelte` flat recommended configs. Apply browser
globals to `src`, Node globals to config/scripts, and ignore `dist`, `coverage`,
and `src/lib/api/generated.ts`. Do not disable individual rules merely to make
the empty shell pass.

- [ ] **Step 5: Install, format, and run the workspace checks**

Run:

```bash
bun install
bun run --cwd web format
bun run web:test
bun run web:check
bun run web:build
```

Expected: all commands PASS; `web/dist/.vite/manifest.json` exists; the archive
test lists exactly the three allowed files.

- [ ] **Step 6: Ignore generated frontend output**

Append these entries to `.gitignore`:

```gitignore
/node_modules/
/web/node_modules/
/web/dist/
/web/coverage/
/web/playwright-report/
/web/test-results/
/packages/*/node_modules/
```

- [ ] **Step 7: Commit the workspace foundation**

```bash
git add package.json bun.lock web packages/roborev-ui .gitignore
git commit -m "build: establish the web workspace"
```

______________________________________________________________________

### Task 2: Generate browser API types from the canonical OpenAPI artifact

**Files:**

- Create: `web/scripts/generate-api.ts`
- Create: `web/src/lib/api/generated.ts` (generated)
- Create: `web/scripts/generate-api.test.ts`
- Modify: `package.json`
- Modify: `web/package.json`
- Modify: `Makefile`

**Interfaces:**

- Consumes: `pkg/client/openapi.yaml`, regenerated by `make api-generate`.

- Produces: `web/src/lib/api/generated.ts` with `paths`, `components`, and
    `operations` from Roborev's canonical API.

- Produces: `make api-check`, which detects Go-client and browser-type drift.

- [ ] **Step 1: Test check mode without touching the committed output**

In `generate-api.test.ts`, copy a tiny valid OpenAPI fixture and a deliberately
stale output into a temporary directory. Invoke the exported
`generateAPI({spec, output, check: true})`, assert that it rejects with
`generated browser API types are stale`, and assert that the stale file is
unchanged. Add a second case where generated output matches and check mode
resolves.

- [ ] **Step 2: Run the generator test and verify the missing module fails**

Run: `cd web && bun x vitest run scripts/generate-api.test.ts`

Expected: FAIL because `generateAPI` does not exist.

- [ ] **Step 3: Implement deterministic generation**

Implement this interface in `web/scripts/generate-api.ts`:

```ts
export interface GenerateAPIOptions {
  spec: string;
  output: string;
  check: boolean;
}

export async function generateAPI(options: GenerateAPIOptions): Promise<void>;
```

Resolve the local `openapi-typescript` executable from `web/node_modules/.bin`,
generate into a sibling temporary file, compare bytes in check mode, and rename
the temporary file over the output in write mode. Always remove the temporary
file in `finally`. The command-line entry uses `../pkg/client/openapi.yaml` and
`src/lib/api/generated.ts`; only `--check` is accepted. Reject any other
argument with exit code 2.

Add `generate` and `generate:check` scripts to `web/package.json`, prepend
`bun run generate:check` to its `check` script, and add
`"web:generate": "bun run --cwd web generate"` to the root scripts. This keeps
Task 1 independently green before the generator exists.

- [ ] **Step 4: Generate the committed type artifact**

Run:

```bash
make api-generate
bun run --cwd web generate
bun run --cwd web generate:check
```

Expected: all commands PASS and
`git diff --exit-code -- web/src/lib/api/generated.ts` is clean after the
check-only run.

- [ ] **Step 5: Extend the API drift target**

Add an `api-check` Make target:

```make
api-check:
	go test ./internal/daemon -run 'TestHumaOpenAPISpec$$'
	cd web && bun run generate:check
```

Keep `api-generate` as the only command that rewrites canonical/generated
artifacts.

- [ ] **Step 6: Commit canonical browser type generation**

```bash
git add package.json web/package.json web/scripts web/src/lib/api/generated.ts Makefile pkg/client
git commit -m "build: generate browser API types"
```

______________________________________________________________________

### Task 3: Serve a validated embedded distribution with the final CSP

**Files:**

- Create: `internal/web/dist/index.html`
- Create: `internal/web/embed.go`
- Create: `internal/web/assets.go`
- Create: `internal/web/assets_test.go`
- Create: `internal/web/handler.go`
- Create: `internal/web/handler_test.go`

**Interfaces:**

- Produces: `web.NewEmbeddedHandler() (http.Handler, error)`.

- Produces: `web.NewHandler(files fs.FS) (http.Handler, error)` for tests.

- Produces: `web.ValidateEmbeddedRelease() error`.

- Produces: `web.ContentSecurityPolicy` for an exact-header regression test.

- Distinguishes the one canonical compilation stub from malformed production
    assets: the handler may serve the harmless stub, while release validation
    must reject it.

- [ ] **Step 1: Write release-distribution tests**

Use `fstest.MapFS` fixtures to prove:

```go
func TestValidateReleaseDistributionRejectsCompilationStub(t *testing.T)
func TestLoadDistributionAcceptsCanonicalCompilationStub(t *testing.T)
func TestLoadDistributionRejectsUnknownIncompleteDistribution(t *testing.T)
func TestValidateReleaseDistributionRejectsMissingManifestAsset(t *testing.T)
func TestValidateReleaseDistributionAcceptsCompleteViteGraph(t *testing.T)
func TestLoadAssetCatalogRejectsTraversalAndSecretLikePaths(t *testing.T)
func TestEmbeddedReleaseDistribution(t *testing.T)
```

The accepted fixture contains a production-marked `index.html`, a
`.vite/manifest.json` referencing `assets/index-a1b2c3.js` and
`assets/index-a1b2c3.css`, and both files. Assert the immutable catalog contains
both assets.

`TestEmbeddedReleaseDistribution` is the release-only bridge to the real
`embed.FS`:

```go
func TestEmbeddedReleaseDistribution(t *testing.T) {
    if os.Getenv("ROBOREV_RUN_WEB_RELEASE_CHECK") != "1" {
        t.Skip("release asset validation is enabled by the release target")
    }
    require.NoError(t, ValidateEmbeddedRelease())
}
```

This test skips during ordinary Go development, where the tracked compilation
stub is expected, and runs only after `make web-release-check` has staged a
validated production distribution.

- [ ] **Step 2: Run the asset tests and verify they fail**

Run: `go test ./internal/web -run 'Test(Validate|LoadAsset)' -count=1`

Expected: FAIL because `internal/web` has no implementation.

- [ ] **Step 3: Implement the embedded asset catalog**

Use these constants and manifest shape:

```go
const viteManifestPath = ".vite/manifest.json"
const productionDistributionMarker = `<meta name="roborev-web-distribution" content="production"`

type viteManifestEntry struct {
    File   string   `json:"file"`
    CSS    []string `json:"css"`
    Assets []string `json:"assets"`
}

type assetCatalog struct {
    immutable map[string]struct{}
}
```

Require every manifest path to satisfy `fs.ValidPath`, equal `path.Clean`, use
forward slashes, contain no dot-prefixed segment, and avoid secret-like base
names (`credential`, `credentials`, `secret`, `secrets`, `token`, `tokens`,
`password`, `passwd`, `id_rsa`, `id_ed25519`) and extensions (`.key`, `.pem`,
`.p12`, `.pfx`). Require every referenced entry to be a regular file.

Embed with:

```go
//go:embed all:dist
var embeddedDistribution embed.FS
```

`ValidateEmbeddedRelease` calls the same `validateReleaseDistribution` used by
tests and rejects the tracked stub.

`NewEmbeddedHandler` first identifies the exact canonical compilation-stub
bytes. It may build a no-asset handler for that stub so ordinary `go test`,
`go run`, and the Vite-proxied development daemon can start. Any other missing
marker or manifest remains an error. Never let the stub exception enter
`ValidateEmbeddedRelease`.

- [ ] **Step 4: Write hardened handler tests**

Cover these observable cases with `httptest`:

- `/` redirects to `/reviews` with no-store caching.

- `/reviews/42` with `Accept: text/html` serves the application shell.

- `/analytics` serves the application shell.

- `/missing.js`, `/api/status`, `/openapi.json`, traversal, backslash, NUL, and
    dot-segment paths return 404 instead of the shell.

- GET and HEAD work; POST to static content returns 404.

- manifest assets receive `public, max-age=31536000, immutable`.

- the shell receives `no-store`; non-manifest files receive `no-cache`.

- content types are fixed from a small extension map rather than host sniffing.

- every response contains the exact CSP, `X-Content-Type-Options: nosniff`, and
    `Referrer-Policy: no-referrer`.

- [ ] **Step 5: Implement the static handler**

Export this exact CSP:

```go
const ContentSecurityPolicy = "default-src 'self'; base-uri 'none'; object-src 'none'; frame-ancestors 'none'; form-action 'self'; script-src 'self' 'wasm-unsafe-eval'; style-src 'self'; style-src-attr 'unsafe-inline'; style-src-elem 'self' 'unsafe-inline'; img-src 'self' data:; font-src 'self'; connect-src 'self'"
```

Only perform history fallback when the request accepts HTML, the path has no
dotted segment, and the path is not owned by the daemon. Write bytes from the
validated `fs.FS`; do not join request paths to the host filesystem.

- [ ] **Step 6: Add the harmless compilation stub**

`internal/web/dist/index.html` must omit the production marker and state that
web assets are not built. Verify:

```bash
go test ./internal/web -count=1
go build ./...
```

Expected: PASS. A normal Go development build compiles, while
`ValidateEmbeddedRelease` rejects that binary's stub.

- [ ] **Step 7: Commit embedded static serving**

```bash
git add internal/web
git commit -m "feat(web): serve hardened embedded assets"
```

______________________________________________________________________

### Task 4: Build, validate, stage, and restore frontend assets transactionally

**Files:**

- Create: `web/scripts/validate-assets.ts`
- Create: `web/scripts/validate-assets.test.ts`
- Create: `web/scripts/embed-assets.ts`
- Create: `web/scripts/embed-assets.test.ts`
- Modify: `web/package.json`
- Modify: `Makefile`
- Modify: `.gitignore`

**Interfaces:**

- Produces: `validateAssetGraph(root: string): Promise<void>`.

- Produces: `embedAssets(source: string): Promise<void>`.

- Produces: `restoreCompilationStub(): Promise<void>`.

- Produces: `make web-build`, `web-assets-check`, `web-embed`, and
    `web-release-check`.

- [ ] **Step 1: Write filesystem-behavior tests**

Create temporary distributions and assert validation rejects a missing marker,
missing manifest, empty manifest, missing referenced file, symlink, traversal,
and secret-like path. Assert it accepts a complete graph. For embedding, start
with a stub directory, stage a valid graph, verify all files changed together,
then call restore and verify the directory contains only the canonical stub.
Inject a failing rename/copy callback and verify the previous distribution is
restored.

- [ ] **Step 2: Run the script tests and verify they fail**

Run:

```bash
cd web
bun x vitest run scripts/validate-assets.test.ts scripts/embed-assets.test.ts
```

Expected: FAIL because the exported functions do not exist.

- [ ] **Step 3: Implement validation and transactional replacement**

Use `lstat` to reject symlinks. Validate the same path rules and production
marker as `internal/web/assets.go`. `embedAssets` must copy into
`internal/web/.dist-staging-<pid>`, validate the copy, rename the current `dist`
to `.dist-backup-<pid>`, rename staging to `dist`, and restore the backup if the
second rename fails. `finally` removes staging; success removes backup.

`restoreCompilationStub` uses the same replacement function and writes exactly
the tracked stub contents, rather than invoking Git or assuming a clean tree.

Ignore transient staging/backup directories and generated embedded files while
retaining the stub:

```gitignore
/internal/web/.dist-staging-*/
/internal/web/.dist-backup-*/
/internal/web/dist/*
!/internal/web/dist/index.html
```

- [ ] **Step 4: Add package and Make commands**

Add these web scripts:

```json
{
  "assets:check": "bun run scripts/validate-assets.ts dist",
  "assets:embed": "bun run scripts/embed-assets.ts dist",
  "assets:restore": "bun run scripts/embed-assets.ts --restore-stub"
}
```

Add Make targets patterned as follows:

```make
web-build:
	cd web && bun run build

web-assets-check: web-build
	cd web && bun run assets:check

web-embed: web-assets-check
	cd web && bun run assets:embed

web-release-check: web-embed
	@set -e; trap 'cd web && bun run assets:restore' EXIT; \
		ROBOREV_RUN_WEB_RELEASE_CHECK=1 CGO_ENABLED=0 \
		go test ./internal/web -run '^TestEmbeddedReleaseDistribution$$' -count=1
```

Make `build` depend on `web-embed` and restore the stub in an EXIT trap after
the Go binary is compiled. Do not change `install` in this plan.

- [ ] **Step 5: Verify staging leaves the worktree clean**

Run:

```bash
make web-release-check
make build
git diff --exit-code -- internal/web/dist
```

Expected: PASS; `bin/roborev` exists; the tracked compilation stub is restored.

- [ ] **Step 6: Commit the asset pipeline**

```bash
git add web/scripts web/package.json Makefile .gitignore
git commit -m "build(web): validate embedded release assets"
```

______________________________________________________________________

### Task 5: Add validated browser configuration and endpoint policy

**Files:**

- Modify: `internal/config/config.go`
- Modify: `internal/config/config_test.go`
- Create: `internal/daemon/browser_endpoint.go`
- Create: `internal/daemon/browser_endpoint_test.go`
- Create: `internal/daemon/browser_policy.go`
- Create: `internal/daemon/browser_policy_test.go`

**Interfaces:**

- Produces: `config.WebConfig`.

- Produces:
    `daemon.ResolveBrowserEndpoint(config.WebConfig) (BrowserEndpoint, error)`.

- Produces:
    `daemon.NewBrowserPolicy(BrowserEndpoint, string) (BrowserPolicy, error)`;
    the string is the optional explicit development origin.

- Produces: `BrowserPolicy.ValidateHost(*http.Request) error`,
    `ValidateOrigin(*http.Request) error`, and
    `AllowsLocalSession(*http.Request) bool`.

- [ ] **Step 1: Write configuration tests**

Add table tests for these values:

```go
type WebConfig struct {
    Enabled      bool   `toml:"enabled"`
    Listen       string `toml:"listen"`
    PublicOrigin string `toml:"public_origin"`
    AuthToken    string `toml:"auth_token" sensitive:"true"`
}
```

Assert defaults are enabled and `127.0.0.1:0`. Assert the token is a recognized
global sensitive key and is masked by config output. Reject origins with user
info, paths, queries, fragments, unsupported schemes, or non-loopback HTTP.
Reject non-loopback listens without both a token and HTTPS public origin. Accept
loopback HTTP and canonicalize scheme/host casing and default ports.

- [ ] **Step 2: Run the focused config tests and verify they fail**

Run: `go test ./internal/config -run 'Test(WebConfig|Sensitive.*Web)' -count=1`

Expected: FAIL because `WebConfig` is absent.

- [ ] **Step 3: Add config normalization**

Embed `Web WebConfig `toml:"web"\`\` in `Config`, initialize it in
`DefaultConfig`, and call `normalizeWebConfig` from `LoadGlobalFrom` after TOML
decode. Keep the token in memory only; do not log it in validation errors.

- [ ] **Step 4: Test endpoint resolution and exact authority policy**

Use an injected `listen(network, address)` function to test ephemeral binding
without racing for a port. Cover:

- disabled config returns no listener and no origin;

- loopback `:0` derives `http://<actual-address>`;

- public origin replaces the advertised origin but not the backend bind;

- accepted Hosts contain only the actual listener authority, public-origin
    authority, and optional explicit development-origin authority;

- Host is checked case-insensitively after strict authority parsing;

- suffixes, user info, empty hosts, invalid ports, and unlisted authorities
    fail;

- exact Origin matching succeeds and `null`, suffixes, paths, and absent Origin
    fail on session requests;

- forwarded headers never change Host/Origin outcomes;

- local trust requires direct loopback `RemoteAddr`, loopback Host/Origin,
    absent forwarding headers, and an empty auth token.

- [ ] **Step 5: Implement endpoint and policy types**

Use these exact public shapes:

```go
type BrowserEndpoint struct {
    Listener net.Listener
    Address  string
    DialAddress string
    Origin   string
    Enabled  bool
}

type BrowserPolicy struct {
    origin            string
    acceptedHosts     map[string]struct{}
    acceptedOrigins   map[string]struct{}
    remoteAuthEnabled bool
}
```

`ResolveBrowserEndpoint` owns the listener and closes it on every validation
error after bind. `NewBrowserPolicy` accepts the canonical endpoint origin plus
an optional development origin. The development origin must itself be an exact
loopback HTTP(S) origin.

`Address` is the listener's resolved authority for runtime diagnostics.
`DialAddress` is a reachable loopback authority used only for readiness probes;
when the listener binds an unspecified address, derive the same port with an
address-family-compatible loopback host. Never use `DialAddress` as an accepted
browser Host or advertised public origin.

Check the direct peer with `net.SplitHostPort(r.RemoteAddr)` and
`net.ParseIP(...).IsLoopback()`. Treat any non-empty `Forwarded`,
`X-Forwarded-For`, `X-Forwarded-Host`, `X-Forwarded-Proto`, or `X-Real-IP` as
disqualifying local trust. Do not reject those headers for authenticated remote
sessions; simply ignore them for authority.

- [ ] **Step 6: Run the policy suites**

Run:

```bash
go test ./internal/config -run 'Test(WebConfig|Sensitive.*Web)' -count=1
go test ./internal/daemon -run 'Test(BrowserEndpoint|BrowserPolicy)' -count=1
```

Expected: PASS.

- [ ] **Step 7: Commit browser endpoint policy**

```bash
git add internal/config internal/daemon/browser_endpoint* internal/daemon/browser_policy*
git commit -m "feat(web): validate browser listener authority"
```

______________________________________________________________________

### Task 6: Implement process-local ambient and tab sessions

**Files:**

- Create: `internal/daemon/browser_session.go`
- Create: `internal/daemon/browser_session_test.go`

**Interfaces:**

- Produces:
    `NewBrowserSessionManager(BrowserSessionConfig) (*BrowserSessionManager, error)`.

- Produces: `Login(token string) (SessionCredentials, error)`.

- Produces: `NewLocalSession() (SessionCredentials, error)`.

- Produces: `Bootstrap(ambient string) (SessionCredentials, error)`.

- Produces: `Authenticate(ambient, tab string) (BrowserPrincipal, error)`.

- Produces: `CheckCSRF(tab, csrf string) error` and `Logout(ambient string)`.

- Produces headers `WebSessionHeader = "X-Roborev-Web-Session"` and
    `WebCSRFHeader = "X-Roborev-CSRF"`.

- [ ] **Step 1: Write deterministic session tests**

Inject `io.Reader` entropy and `func() time.Time`. Test:

- a wrong token and empty token fail in constant-time comparison path;

- login returns an ambient cookie value, tab token, CSRF token, and expiry;

- the token is not present in any returned value;

- bootstrap mints a distinct tab/CSRF pair bound to the same ambient session;

- bootstrap does not invalidate the first tab;

- an ambient value from one manager cannot authenticate against another;

- swapping ambient and tab values across sessions fails;

- expiry removes ambient and all bound tabs;

- logout invalidates every tab for that ambient session;

- CSRF is bound to its tab and compared by hash;

- local-session creation fails when `AllowLocal` is false;

- the cookie name includes the manager's random instance identifier;

- cookie flags are Host-only, HttpOnly, SameSite Strict, Path `/`, and Secure
    exactly when the public origin is HTTPS.

- [ ] **Step 2: Run the tests and verify they fail**

Run: `go test ./internal/daemon -run 'TestBrowserSession' -count=1`

Expected: FAIL because the manager does not exist.

- [ ] **Step 3: Implement the session state model**

Use SHA-256 digests as map keys; never retain presented bearer values:

```go
type BrowserSessionConfig struct {
    Origin     string
    AuthToken  string
    AllowLocal bool
    TTL        time.Duration
    Entropy    io.Reader
    Clock      func() time.Time
}

type BrowserPrincipal struct {
    Local bool
}

type SessionCredentials struct {
    Ambient string
    Tab     string
    CSRF    string
    Expires time.Time
}

type ambientSession struct {
    id        [32]byte
    principal BrowserPrincipal
    expiresAt time.Time
}

type tabSession struct {
    ambientID [32]byte
    csrfHash  [32]byte
    expiresAt time.Time
}
```

Generate 32 random bytes and encode with unpadded base64url. Hash the configured
auth token at construction and compare SHA-256 values with
`subtle.ConstantTimeCompare`. Use a 12-hour default TTL. Expiration cleanup runs
opportunistically under the manager mutex; no background goroutine is needed.

- [ ] **Step 4: Implement cookie construction**

Expose:

```go
func (m *BrowserSessionManager) Cookie(value string) *http.Cookie
func (m *BrowserSessionManager) ExpiredCookie() *http.Cookie
func (m *BrowserSessionManager) CookieName() string
```

Do not set `Domain`. Set `MaxAge: -1` and an epoch expiry on the expired cookie.

- [ ] **Step 5: Run race-enabled session tests**

Run:

```bash
go test -race ./internal/daemon -run 'TestBrowserSession' -count=1
```

Expected: PASS, including a concurrent bootstrap/authenticate/logout case.

- [ ] **Step 6: Commit process-local sessions**

```bash
git add internal/daemon/browser_session.go internal/daemon/browser_session_test.go
git commit -m "feat(web): add tab-scoped browser sessions"
```

______________________________________________________________________

### Task 7: Add browser-only session routes, allowlist, authentication, and CSRF

**Files:**

- Create: `internal/daemon/browser_routes.go`
- Create: `internal/daemon/browser_routes_test.go`
- Create: `internal/daemon/browser_handler.go`
- Create: `internal/daemon/browser_handler_test.go`
- Modify: `internal/daemon/routes.go`
- Modify: `internal/daemon/routes_test.go`
- Regenerate: `pkg/client/openapi.yaml`
- Regenerate: `pkg/client/generated/*`
- Regenerate: `web/src/lib/api/generated.ts`

**Interfaces:**

- Produces Huma operations `login-web-session`, `bootstrap-web-session`,
    `logout-web-session`, and `get-web-session-status`.

- Produces `POST /api/ui/session/login`, `POST /api/ui/session/bootstrap`,
    `DELETE /api/ui/session`, and `GET /api/ui/session` on the browser listener
    only.

- Produces:
    `Server.newBrowserHandler(static http.Handler, policy BrowserPolicy, sessions *BrowserSessionManager) (http.Handler, error)`.

- Produces context accessor
    `BrowserPrincipalFromContext(context.Context) (BrowserPrincipal, bool)` for
    later browser handlers.

- [ ] **Step 1: Split route registration without changing the CLI surface**

Refactor `routes.go` into these helpers:

```go
func newHumaAPI(mux *http.ServeMux) huma.API
func (s *Server) registerCoreRoutes(api huma.API)
func (s *Server) registerBrowserRoutes(api huma.API)
func (s *Server) registerHumaAPI(mux *http.ServeMux) huma.API
func (s *Server) registerBrowserHumaAPI(mux *http.ServeMux) huma.API
```

`registerHumaAPI` calls only `registerCoreRoutes`. The actual browser mux calls
only `registerBrowserRoutes` for browser-specific endpoints. OpenAPI generation
creates one throwaway API and calls both registration helpers so one canonical
document describes both listener surfaces.

Before adding browser routes, run:

```bash
go test ./internal/daemon -run 'TestHumaOpenAPISpec|Test.*Routes' -count=1
```

Expected: PASS with no route changes.

- [ ] **Step 2: Write session-route contract tests**

Use real `httptest` requests and assert:

- login requires POST JSON, exact Host/Origin, and the correct token;

- invalid login returns a generic 401 code without echoing the token;

- bootstrap is POST-only, JSON-only, and requires `Sec-Fetch-Site: same-origin`,
    `Sec-Fetch-Mode: cors`, and `Sec-Fetch-Dest: empty`;

- a valid remote ambient cookie bootstraps a new tab without rotating
    `Set-Cookie`;

- a missing remote cookie returns 401;

- strict local policy permits bootstrap without a cookie and sets one;

- logout requires cookie, tab header, exact Origin, and CSRF, then expires the
    cookie;

- session status reveals only `authentication: "local"|"token"`,
    `authenticated`, and `expires_at`, never tokens.

- [ ] **Step 3: Define typed session envelopes and Huma routes**

Use these wire bodies:

```go
type WebLoginRequest struct {
    Token string `json:"token" minLength:"1"`
}

type WebSessionCredentials struct {
    Session   string    `json:"session"`
    CSRF      string    `json:"csrf"`
    ExpiresAt time.Time `json:"expires_at"`
}

type WebSessionStatus struct {
    Authentication string     `json:"authentication" enum:"local,token"`
    Authenticated  bool       `json:"authenticated"`
    ExpiresAt      *time.Time `json:"expires_at,omitempty"`
}
```

The ambient value is only a `Set-Cookie` response header. The JSON credential
body contains the tab and CSRF values, not the cookie value.

`GET /api/ui/session` is safe to call without credentials and returns
`authenticated: false`; when cookie and tab credentials are present it
authenticates them and returns the non-secret mode and expiry. Login and
bootstrap return the credentials envelope. Logout returns 204 and expires the
ambient cookie.

- [ ] **Step 4: Write allowlist and middleware tests**

Define the first browser API allowlist exactly:

```go
var browserAPIRoutes = map[routeKey]routePolicy{
    {http.MethodGet,  "/api/ping"}:          publicRoute,
    {http.MethodGet,  "/api/status"}:        authenticatedRoute,
    {http.MethodGet,  "/api/jobs"}:          authenticatedRoute,
    {http.MethodGet,  "/api/review"}:        authenticatedRoute,
    {http.MethodGet,  "/api/comments"}:      authenticatedRoute,
    {http.MethodGet,  "/api/repos"}:         authenticatedRoute,
    {http.MethodGet,  "/api/branches"}:      authenticatedRoute,
    {http.MethodGet,  "/api/job/output"}:    authenticatedRoute,
    {http.MethodGet,  "/api/job/log"}:       authenticatedRoute,
    {http.MethodGet,  "/api/stream/events"}: authenticatedRoute,
    {http.MethodPost, "/api/job/cancel"}:    mutationRoute,
    {http.MethodPost, "/api/job/rerun"}:     mutationRoute,
    {http.MethodPost, "/api/review/close"}:  mutationRoute,
    {http.MethodPost, "/api/comment"}:       mutationRoute,
}
```

Session routes are dispatched by the browser-only Huma mux before this table.
Future projection and analytics routes extend the table in their owning plans.

Assert Host rejection happens before an authorization failure by wrapping a
handler that records whether auth ran. Assert unknown APIs, wrong methods,
`/api/shutdown`, `/debug/pprof/`, and OpenAPI endpoints return 404. Assert GET
routes need cookie plus tab header. Assert mutations additionally require exact
Origin and the tab-bound CSRF header. Assert static navigation remains public.

- [ ] **Step 5: Implement handler composition**

Compose in this order:

```text
exact Host guard
  -> browser session endpoints
  -> explicit core API allowlist
       -> cookie + tab authentication
       -> mutation Origin + CSRF
       -> existing core API handler
  -> embedded static handler
```

Set authenticated responses to `Cache-Control: private, no-store` and add
`Vary: Cookie` and `Vary: X-Roborev-Web-Session`. Use generic JSON errors with
stable codes `invalid_host`, `invalid_origin`, `web_session_required`, and
`csrf_invalid`. Do not log presented credentials.

- [ ] **Step 6: Regenerate and verify both API clients**

Run:

```bash
make api-generate
bun run --cwd web generate
make api-check
go test ./internal/daemon -run 'Test(BrowserRoutes|BrowserHandler|HumaOpenAPI)' -count=1
```

Expected: PASS. The canonical OpenAPI contains browser session operations, but
requests to those paths on `server.httpServer.Handler` return 404.

- [ ] **Step 7: Commit the browser API boundary**

```bash
git add internal/daemon pkg/client web/src/lib/api/generated.ts
git commit -m "feat(web): enforce browser session boundaries"
```

______________________________________________________________________

### Task 8: Start the separate browser server and publish non-secret runtime metadata

**Files:**

- Create: `internal/daemon/browser_server.go`
- Create: `internal/daemon/browser_server_test.go`
- Modify: `internal/daemon/server.go`
- Modify: `internal/daemon/server_test.go`
- Modify: `internal/daemon/runtime.go`
- Modify: `internal/daemon/runtime_test.go`
- Modify: `cmd/roborev/daemon_cmd.go`
- Modify: `cmd/roborev/daemon_integration_test.go`

**Interfaces:**

- Produces: `daemon.WithWebDevelopmentOrigin(origin string) ServerOption`.

- Produces runtime fields `WebOrigin string`, `WebAddress string`, and
    `WebCapabilities []string`.

- Changes: `WriteRuntime(primary, alternate, version, browser)` where browser is
    `*BrowserRuntimeInfo` and nil means disabled.

- [ ] **Step 1: Write runtime round-trip tests**

Use this metadata shape:

```go
type BrowserRuntimeInfo struct {
    Address      string
    Origin       string
    Capabilities []string
}
```

Persist metadata keys `web_address`, `web_origin`, and comma-separated
`web_capabilities`. `web_address` is the actual listener authority used by the
isolated Vite proxy; `web_origin` is the user-facing origin used by
`roborev ui`. Round-trip `web-ui-v1` and `web-session-v1`; reject empty,
whitespace-padded, or duplicate capability names. Assert runtime JSON does not
contain the auth token, cookie name, instance ID, tab token, or CSRF token.

- [ ] **Step 2: Write full listener lifecycle tests**

Start a real daemon with isolated data and `web.listen = "127.0.0.1:0"`. Assert:

- primary and browser addresses differ;

- `/api/shutdown` works on primary and returns 404 on browser;

- browser `/api/ping` is ready before runtime metadata is published;

- browser `/reviews/7` serves the public shell;

- browser `/api/status` returns 401 without a session;

- `Stop` drains and closes both listeners;

- a browser bind failure aborts startup and removes any partial runtime file;

- `web.enabled = false` starts only the primary listener and omits web metadata;

- a daemon restart produces a new cookie name and rejects old credentials.

- [ ] **Step 3: Implement browser server startup**

`startBrowserServer` performs this order:

1. Resolve and bind the browser endpoint.
2. Build the exact policy and session manager.
3. Construct `web.NewEmbeddedHandler` and `newBrowserHandler`.
4. Start a distinct `http.Server` on the already-bound listener.
5. Poll `/api/ping` through the listener address with the accepted Host until
    ready or two seconds elapse.
6. Return `BrowserRuntimeInfo` only after readiness succeeds.

On any failure, close the listener, shut down the partial server, and return an
error. Treat browser startup failure as fatal when `web.enabled` is true.

- [ ] **Step 4: Integrate lifecycle and options**

Add these fields to `Server`:

```go
browserServer    *http.Server
browserListener  net.Listener
browserRuntime   *BrowserRuntimeInfo
webDevOrigin     string
```

Create the browser listener after the core listener is ready and before
`WriteRuntime`. Publish core and browser metadata atomically in the one runtime
record. In `stopOnce0`, shut down the browser server with the same bounded
context before removing runtime metadata. Keep `/api/shutdown` core-only.

Add a variadic option without breaking existing callers:

```go
type ServerOption func(*Server)

func WithWebDevelopmentOrigin(origin string) ServerOption {
    return func(server *Server) { server.webDevOrigin = origin }
}
```

Add hidden daemon flag `--web-dev-origin`; pass the option only when non-empty.
Endpoint validation restricts it to an exact loopback origin.

- [ ] **Step 5: Run lifecycle tests on the supported transport paths**

Run:

```bash
go test ./internal/daemon -run 'Test(BrowserServer|Server.*Browser|Runtime.*Web)' -count=1
go test ./cmd/roborev -run 'TestDaemon.*Web' -count=1
CGO_ENABLED=0 go build ./...
```

Expected: PASS on the current platform. Existing Unix/TCP alternate-listener
tests remain unchanged.

- [ ] **Step 6: Commit listener lifecycle and runtime metadata**

```bash
git add internal/daemon cmd/roborev/daemon_cmd.go cmd/roborev/daemon_integration_test.go
git commit -m "feat(web): run the authenticated browser listener"
```

______________________________________________________________________

### Task 9: Exercise login and fresh-tab bootstrap from the minimal Svelte shell

**Files:**

- Create: `web/src/lib/api/session.ts`
- Create: `web/src/lib/api/session.test.ts`
- Modify: `web/src/App.svelte`
- Modify: `web/src/App.test.ts`
- Modify: `web/src/app.css`
- Create: `web/scripts/dev.ts`
- Create: `web/scripts/dev.test.ts`
- Modify: `web/vite.config.ts`
- Modify: `web/package.json`
- Modify: `Makefile`

**Interfaces:**

- Produces: `bootstrapSession`, `login`, `logout`, `sessionHeaders`, and
    `clearTabSession` in `web/src/lib/api/session.ts`.

- Produces session-storage keys `roborev.web.session` and `roborev.web.csrf`.

- Produces: `make web-dev` as the only supported full-stack frontend dev entry.

- [ ] **Step 1: Write client credential-lifetime tests**

Mock `fetch` and `sessionStorage`. Assert:

- bootstrap sends POST JSON with `credentials: "same-origin"` and no tab header;

- login sends the token only in the JSON body and the caller clears its input;

- successful responses write only tab and CSRF values to `sessionStorage`;

- no token, cookie, or expiry is written to local storage;

- `sessionHeaders()` returns both custom headers for authenticated API calls;

- 401 clears stale tab credentials and returns `login-required`;

- logout sends tab plus CSRF headers and clears storage even when fetch fails.

- [ ] **Step 2: Run tests and verify the missing client fails**

Run: `cd web && bun x vitest run src/lib/api/session.test.ts`

Expected: FAIL because `session.ts` does not exist.

- [ ] **Step 3: Implement the session client**

Use a discriminated result:

```ts
export type SessionResult =
  | { state: "authenticated"; expiresAt: string }
  | { state: "login-required" }
  | { state: "error"; message: string };
```

All requests use relative `/api/ui/session/*` URLs and `redirect: "error"`.
Parse only the documented credential body. Never accept a token in a return URL
or query string.

- [ ] **Step 4: Replace the placeholder with the authentication shell**

On mount, call bootstrap. Render one of four states: checking session, token
login form, authenticated foundation, or retryable error. The password input
uses `autocomplete="current-password"`; clear its bound value in a `finally`
block after login. The authenticated state renders links to `/reviews` and
`/analytics` but does not implement either workspace in this plan.

- [ ] **Step 5: Test the isolated development orchestrator**

Inject process spawning and port allocation into `dev.ts`. Assert it:

- creates a temporary root;
- sets `ROBOREV_DATA_DIR` to `<root>/data`;
- passes `--db <root>/reviews.db`, `--addr 127.0.0.1:0`, and
    `--web-dev-origin http://127.0.0.1:<vite-port>` to
    `go run ./cmd/roborev daemon run`;
- sets `ROBOREV_WEB_DEV_BACKEND` for Vite from runtime `web_address`, while
    runtime `web_origin` remains the explicit Vite origin;
- does not copy the normal config or inherit a normal data-directory override;
- forwards termination to both children and removes the temporary root.

The script waits for runtime metadata instead of parsing log text. Use synthetic
database setup only; an empty temporary SQLite database is sufficient for this
foundation shell.

- [ ] **Step 6: Implement and verify full-stack development**

Add `"dev:full": "bun run scripts/dev.ts"` and:

```make
web-dev:
	cd web && bun run dev:full
```

Run the script test, then start `make web-dev` and manually verify in the
browser-network inspector that bootstrap succeeds, the custom credentials live
only in session storage, and API requests go through Vite to the disposable
daemon. Stop it and verify no process or temporary data remains.

- [ ] **Step 7: Run frontend checks**

Run:

```bash
bun run --cwd web test
bun run --cwd web check
bun run --cwd web build
```

Expected: PASS.

- [ ] **Step 8: Commit the authenticated foundation shell**

```bash
git add web Makefile
git commit -m "feat(web): bootstrap browser sessions in the shell"
```

______________________________________________________________________

### Task 10: Add `roborev ui` without moving credentials into the launch URL

**Files:**

- Create: `cmd/roborev/ui_cmd.go`
- Create: `cmd/roborev/ui_cmd_test.go`
- Create: `cmd/roborev/open_browser.go`
- Create: `cmd/roborev/open_browser_test.go`
- Modify: `cmd/roborev/main.go`

**Interfaces:**

- Produces: `roborev ui [job-id]`.

- Produces: `openBrowserURL(url string) error` as an injectable package
    variable.

- Consumes: live `RuntimeInfo.WebOrigin` after `ensureDaemon()`.

- [ ] **Step 1: Write command behavior tests**

Inject daemon discovery and the opener. Assert:

- no argument opens `<origin>/reviews`;

- positive numeric `42` opens `<origin>/reviews/42`;

- zero, negative, non-numeric, and multiple arguments fail as usage errors;

- missing browser metadata returns `the daemon browser listener is disabled`;

- discovery access denial is preserved;

- the opened URL never includes configured token material;

- opener errors include the validated public URL so the user can open it
    manually.

- [ ] **Step 2: Run the command tests and verify they fail**

Run: `go test ./cmd/roborev -run 'TestUICmd|TestOpenBrowser' -count=1`

Expected: FAIL because `uiCmd` is absent.

- [ ] **Step 3: Implement URL resolution and command registration**

Parse `RuntimeInfo.WebOrigin` with `net/url`; require an HTTP(S) origin without
credentials, path, query, or fragment. Resolve only `/reviews` or
`/reviews/<positive-id>`. Call `ensureDaemon`, rediscover runtime metadata after
any start/restart, and then call the injected opener. Add
`rootCmd.AddCommand(uiCmd())` next to `tuiCmd()`.

- [ ] **Step 4: Implement platform opening without a shell**

Select an argv vector and call `exec.Command` directly:

```go
switch runtime.GOOS {
case "darwin":
    return exec.Command("open", target).Start()
case "windows":
    return exec.Command("rundll32", "url.dll,FileProtocolHandler", target).Start()
default:
    return exec.Command("xdg-open", target).Start()
}
```

Do not use `sh -c`, `cmd /c start`, or string concatenation. Tests inject the
command starter and assert each argv vector.

- [ ] **Step 5: Run command and cross-platform build checks**

Run:

```bash
go test ./cmd/roborev -run 'TestUICmd|TestOpenBrowser' -count=1
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go test -c -o /tmp/roborev-ui-windows.test.exe ./cmd/roborev
CGO_ENABLED=0 go build ./...
```

Expected: PASS. Remove the temporary test binary afterward.

- [ ] **Step 6: Commit browser launch support**

```bash
git add cmd/roborev
git commit -m "feat(cli): open the native browser UI"
```

______________________________________________________________________

### Task 11: Wire frontend verification into CI and release packaging

**Files:**

- Create: `cmd/roborev/verify_web_assets_cmd.go`
- Create: `cmd/roborev/verify_web_assets_cmd_test.go`
- Modify: `cmd/roborev/main.go`
- Modify: `.github/workflows/ci.yml`
- Modify: `.github/workflows/release.yml`
- Modify: `.goreleaser.yaml`
- Modify: `prek.toml`
- Modify: `Makefile`

**Interfaces:**

- Produces: a frontend CI job for generated types, checks, tests, package
    archive, build, and embedded-release validation.

- Produces: release builds that cannot compile before validated production
    assets are staged.

- [ ] **Step 1: Extend path detection**

Add a `web` filter covering:

```yaml
- 'package.json'
- 'bun.lock'
- 'web/**'
- 'packages/roborev-ui/**'
- 'internal/web/**'
- 'internal/daemon/browser_*.go'
- 'pkg/client/openapi.yaml'
- 'Makefile'
- '.goreleaser.yaml'
- '.github/workflows/ci.yml'
- '.github/workflows/release.yml'
```

Expose it as `needs.go-changes.outputs.web`.

- [ ] **Step 2: Add the frontend CI job**

Pin `oven-sh/setup-bun` by full commit SHA and configure Bun 1.3.14. The job
runs:

```bash
bun install --frozen-lockfile
make api-check
bun run web:check
bun run web:test
make web-release-check
git diff --exit-code
```

The final diff check proves generation and stub restoration are non-mutating.

- [ ] **Step 3: Gate release compilation on production assets**

In the release job, set up Bun 1.3.14 and run `bun install --frozen-lockfile`
before GoReleaser. Add this GoReleaser hook:

```yaml
before:
  hooks:
    - make web-release-check
```

Because the hook restores the stub after its verification binary exits, add a
second `make web-embed` hook immediately before GoReleaser builds. Add a final
workflow cleanup step with `if: always()` that runs
`bun run --cwd web assets:restore`. Verify the built release binary through a
hidden `verify-web-assets` Cobra command that calls
`web.ValidateEmbeddedRelease`; the command is unavailable from normal help.

Implement the command as a direct validator with no daemon startup or database
access:

```go
func verifyWebAssetsCmd() *cobra.Command {
    cmd := &cobra.Command{
        Use:    "verify-web-assets",
        Hidden: true,
        Args:   cobra.NoArgs,
        RunE: func(_ *cobra.Command, _ []string) error {
            return webassets.ValidateEmbeddedRelease()
        },
    }
    return cmd
}
```

Register it on the root command. Unit-test that it is hidden and rejects the
ordinary compilation-stub build. After staging assets, build the current host
target with `goreleaser build --snapshot --clean --single-target`, locate the
single executable under `dist/`, and run `verify-web-assets`; require exit 0.

Add a `release-snapshot-check` Make target that wraps the following operations
in an EXIT trap which always runs `cd web && bun run assets:restore`:

1. Run `goreleaser build --snapshot --clean --single-target`.
2. Require exactly one executable named `roborev` under `dist`, then run its
    `verify-web-assets` command.
3. Run `goreleaser release --snapshot --clean` to validate every configured
    archive and package format without publishing.
4. Require `git diff --exit-code -- internal/web/dist` after restoration.

- [ ] **Step 4: Add non-mutating frontend hooks**

Add local `prek` hooks for `bun run web:check` and `bun run web:test` restricted
to frontend/workspace paths. Do not run asset embedding in a commit hook.

- [ ] **Step 5: Validate workflows and a local release snapshot**

Run:

```bash
make check-actions
bun install --frozen-lockfile
bun run web:check
bun run web:test
make web-release-check
make release-snapshot-check
git diff --exit-code
```

Expected: PASS; the current-host snapshot binary's hidden verifier exits 0, all
snapshot packages are created locally, and no release, tag, upload, signature,
or external publication occurs.

- [ ] **Step 6: Commit CI and release enforcement**

```bash
git add .github/workflows .goreleaser.yaml prek.toml Makefile cmd/roborev
git commit -m "ci: require validated browser assets"
```

______________________________________________________________________

### Task 12: Document browser operation, security tradeoffs, and development

**Files:**

- Modify: `docs/configuration.md`
- Modify: `docs/commands.md`
- Modify: `docs/development.md`
- Modify: `docs/changelog.md`

**Interfaces:**

- Documents: `web.enabled`, `web.listen`, `web.public_origin`, and the sensitive
    `web.auth_token` setting.

- Documents: `roborev ui [job-id]`, restart logout, local-token behavior, exact
    reverse-proxy requirements, and `make web-dev` isolation.

- [ ] **Step 1: Add user-facing browser documentation**

Add a configuration example using only reserved values:

```toml
[web]
enabled = true
listen = "127.0.0.1:7374"
public_origin = "https://reviews.example.com"
auth_token = "replace-with-a-random-secret"
```

State explicitly:

- the backend remains loopback behind the proxy;

- the public origin is exact and HTTPS for remote access;

- the proxy preserves the public Host and does not buffer event/output streams;

- the daemon token is entered after opening the shell and is not retained;

- sessions expire on daemon restart, requiring token entry again;

- when a remote token is configured, local browser access also requires it;

- the public shell can load without a cookie so cross-site deep links work with
    SameSite Strict; the same-origin bootstrap fetch then receives the cookie;

- no browser listener means `roborev ui` returns a setup message.

- [ ] **Step 2: Document command and deep-link identity**

Add `roborev ui` and `roborev ui 42` to the command sheet. Explain that the
numeric job ID is local to the selected daemon and links are not portable to a
different synchronized machine.

- [ ] **Step 3: Document isolated frontend development**

Describe Bun 1.3.14, `bun install --frozen-lockfile`, workspace checks,
generation, build/embed validation, and `make web-dev`. State that
`make web-dev` creates a disposable data directory/database and supplies an
explicit loopback Vite origin; there is no automatic development origin bypass.

- [ ] **Step 4: Format and validate docs**

Run:

```bash
make markdown
make markdown-ci
make docs-check
```

Expected: PASS.

- [ ] **Step 5: Scrub the complete public change**

Run the private-data scrub workflow across the full branch diff, unpushed commit
messages, fixtures, and generated assets. Also scan for absolute home paths,
real hostnames, email addresses outside public Git identity, and infrastructure
names. Replace every hit with reserved examples and rerun until there are zero
hits.

- [ ] **Step 6: Run the complete foundation gate**

Run:

```bash
bun install --frozen-lockfile
make api-check
bun run web:check
bun run web:test
make web-release-check
go test ./internal/config ./internal/web ./internal/daemon ./cmd/roborev
CGO_ENABLED=0 go build ./...
make lint-ci
make markdown-ci
git diff --check
```

Expected: every command PASS and the worktree contains only intended docs
changes after the embed stub restores.

- [ ] **Step 7: Commit end-user and developer documentation**

```bash
git add docs
git commit -m "docs: explain native browser access"
```

______________________________________________________________________

## Plan completion checkpoint

Before starting the review-transplant plan, verify these behaviors against a
real disposable daemon:

1. `roborev ui` resolves a runtime-advertised origin without putting a token in
    the URL.
2. The public shell loads from the embedded daemon assets.
3. A fresh local tab bootstraps through strict loopback trust.
4. A token-configured daemon requires login for local and proxied access.
5. A second tab bootstraps from the ambient cookie and does not invalidate the
    first tab.
6. Restarting the daemon invalidates both tabs.
7. Host rejection precedes authentication, unknown browser APIs are 404, and
    `/api/shutdown` remains core-listener-only.
8. Authenticated fetch streams can carry `X-Roborev-Web-Session`; no code uses
    native `EventSource`.
9. Production assets pass the embedded validator under `CGO_ENABLED=0`; the
    compilation stub fails it.
10. `make web-dev` leaves normal Roborev runtime and SQLite data untouched.

Record any intentional contract change in the approved design before changing
these invariants in implementation.
