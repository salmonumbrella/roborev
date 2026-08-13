# Review Application Transplant and Parity Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use
> superpowers:subagent-driven-development (recommended) or
> superpowers:executing-plans to implement this plan task-by-task. Steps use
> checkbox (`- [ ]`) syntax for tracking.

**Goal:** Move the existing review application into Roborev and prove its core
review, mutation, navigation, and streaming workflows against a real disposable
daemon.

**Architecture:** Preserve the reference application's components, stores,
Effect workflows, Markdown pipeline, and interaction behavior before extracting
the presentation package. Replace only host-owned boundaries with a native
router, Roborev-only runtime, session-aware fetch transport, and local context.
The production browser listener remains the only HTTP surface exercised by the
browser tests.

**Tech Stack:** Svelte 5.56.4, TypeScript 5.9.3, Bun 1.3.14, Effect
4.0.0-beta.102, `@kenn-io/kit-ui`, openapi-fetch 0.17.0, Marked 18.0.5,
DOMPurify 3.4.13, Shiki 4.3.0, Mermaid 11.16.1, Vitest 4.1.10, Playwright
1.61.1, Go 1.25, SQLite.

## Global Constraints

- This is a code movement first. Do not redesign review workflows while moving
    them.
- The native app talks directly to `/api/*`; no proxy prefix is retained.
- All browser API and stream requests use the tab session header. Mutations
    additionally use the tab CSRF header.
- Streaming stays fetch-based. Do not use `EventSource`.
- Keep Effect and the existing latest-command, ordered-mutation, cancellation,
    and reconnect behavior.
- Move the complete review Markdown renderer, including DOMPurify hooks, Shiki,
    Mermaid, and their existing tests. Remove only provider-specific URL
    rewriting that has no Roborev meaning.
- Keep the review comment composer in the native app. Package extraction is a
    later plan.
- Use `web/src/lib/api/generated.ts` as the sole browser wire model.
- Browser tests use a scratch `ROBOREV_DATA_DIR`, scratch config, loopback
    listeners, and synthetic fixtures. They must never discover or contact a
    user daemon.
- Preserve the tracked compilation stub after every build or test.

---

### Task 1: Add the transplanted dependency and test surface

**Files:**

- Modify: `web/package.json`
- Modify: `web/vite.config.ts`
- Modify: `web/src/test/setup.ts`
- Modify: `bun.lock`
- Create: `web/src/transplant-contract.test.ts`

**Interfaces:**

- Produces: source-resolved kit-ui, Effect, Markdown, highlighting, diagrams,
    and icon dependencies.

- [x] **Step 1: Write the failing dependency contract**

Create a test that imports `Clipboard`, `Button`, `EmptyState`, `Modal`,
one Lucide icon, DOMPurify, Effect, Marked, Mermaid, and
`getSingletonHighlighter`, then asserts each public entry point exists.

- [x] **Step 2: Verify the dependency contract fails**

Run: `bun x vitest run web/src/transplant-contract.test.ts`

Expected: FAIL because the review dependencies are not installed.

- [x] **Step 3: Add exact reference-compatible dependencies**

Add these runtime dependencies:

```json
{
  "@effect/platform-browser": "4.0.0-beta.102",
  "@kenn-io/kit-ui": "github:kenn-io/kit-ui#97be355ef25a02ce96de6da4f3627ed5b922c4c3",
  "@lucide/svelte": "1.23.0",
  "dompurify": "3.4.13",
  "effect": "4.0.0-beta.102",
  "marked": "18.0.5",
  "mermaid": "11.16.1",
  "shiki": "4.3.0"
}
```

Add `@effect/vitest` 4.0.0-beta.102, `@playwright/test` 1.61.1, and
`playwright` 1.61.1 as development dependencies. Run `bun install` at the
repository root. Import kit-ui theme and Mermaid CSS before app styles and
exclude the source-shipping kit package from Vite dependency optimization.

- [x] **Step 4: Complete the test environment**

Add deterministic `ResizeObserver`, `scrollIntoView`, `matchMedia`,
clipboard, and crypto shims used by moved tests. Keep origin-scoped storage.

- [x] **Step 5: Verify and commit**

Run:

```bash
bun x vitest run web/src/transplant-contract.test.ts
bun run --cwd web check
```

Expected: PASS.

Commit: `build(web): add review application dependencies`

---

### Task 2: Move the session-aware transport and Roborev-only Effect runtime

**Files:**

- Modify: `web/src/lib/api/session.ts`
- Modify: `web/src/lib/api/session.test.ts`
- Create: `web/src/lib/api/client.ts`
- Create: `web/src/lib/api/client.test.ts`
- Create: `web/src/lib/api/effect-errors.ts`
- Create: `web/src/lib/api/retry-policy.ts`
- Create: `web/src/lib/api/schemas.ts`
- Create: `web/src/lib/browser/microtask.ts`
- Create: `web/src/lib/browser/streaming-fetch.ts`
- Create: `web/src/lib/effect/latest-command-by-key.ts`
- Create: `web/src/lib/effect/ordered-command-queue.ts`
- Create: `web/src/lib/runtime/layer.ts`
- Create: `web/src/lib/runtime/runtime.ts`
- Create: `web/src/lib/runtime/context.ts`
- Create: `web/src/lib/runtime/runtime.test.ts`

**Interfaces:**

- Produces: `authenticatedFetch(inner?: Fetch): Fetch`.
- Produces:
    `createRoborevClient(baseUrl?: string, fetchFn?: Fetch): RoborevClient`.
- Produces: fetch-based `roborevEventStream`, `loadRoborevJobOutput`, and
    `roborevJobOutputStream`.
- Produces: `makeAppRuntime(): OwnedAppRuntime` containing only Clipboard,
    streaming fetch, and Roborev workflow services.

- [x] **Step 1: Write failing transport tests**

Prove the wrapper adds the tab header to reads, adds the tab and CSRF headers to
mutations, uses same-origin credentials, preserves caller headers and abort
signals, defaults JSON content type only for JSON-shaped mutations, and clears
tab credentials after a 401.

- [x] **Step 2: Verify the tests fail**

Run:
`bun x vitest run web/src/lib/api/session.test.ts web/src/lib/api/client.test.ts`

Expected: FAIL because the authenticated transport and client do not exist.

- [x] **Step 3: Move and adapt the transport**

Move the NDJSON parser, body release, stream decoding, reconnect errors, retry
schedule, ordered queue, latest-command coordinator, and runtime boundary
without algorithm changes. Use `/` as the client base. Build stream URLs as
`/api/stream/events` and `/api/job/output?job_id=<id>&stream=1`.
`authenticatedFetch` reads session storage at request time.

- [x] **Step 4: Verify runtime behavior**

Test typed failures, interruption-only exits, disposal, malformed NDJSON,
skippable output records, and abort propagation.

Run:
`bun x vitest run web/src/lib/runtime web/src/lib/api web/src/lib/browser`

Expected: PASS.

- [x] **Step 5: Commit**

Commit: `feat(web): add the native review runtime`

---

### Task 3: Move review state and its unit tests

**Files:**

- Create: `web/src/lib/stores/roborev/daemon.svelte.ts`
- Create: `web/src/lib/stores/roborev/jobs.svelte.ts`
- Create: `web/src/lib/stores/roborev/review.svelte.ts`
- Create: `web/src/lib/stores/roborev/log.svelte.ts`
- Create: `web/src/lib/stores/roborev/workflow.ts`
- Create: `web/src/lib/stores/roborev/*.test.ts`
- Create: `web/src/lib/stores/context.ts`
- Create: `web/src/lib/stores/composition.svelte.ts`
- Create: `web/src/lib/utils/roborev-panel.ts`
- Create: `web/src/lib/utils/roborev-panel.test.ts`
- Create: `web/src/lib/utils/roborev-usage.ts`
- Create: `web/src/lib/utils/roborev-usage.test.ts`

**Interfaces:**

- Produces: `createReviewStores({ runtime, client, navigate, onError })`.
- Produces: required daemon, jobs, review, and log stores in one native Svelte
    context.
- Produces: owner-scoped latest-command and ordered-mutation coordination.

- [ ] **Step 1: Move the store tests before implementations**

Copy the daemon, job, cancellation, mutation, workflow, and log tests. Change
only imports, native test layers, canonical generated types, and direct URL
expectations. Preserve assertions for filtering, sorting, panels, stale
responses, cancellation, reconciliation, reconnect, and ownership.

- [ ] **Step 2: Verify representative tests fail**

Run:

```bash
bun x vitest run web/src/lib/stores/roborev/jobs.svelte.test.ts
bun x vitest run web/src/lib/stores/roborev/review.svelte.test.ts
bun x vitest run web/src/lib/stores/roborev/log.svelte.test.ts
```

Expected: FAIL because the implementations are absent.

- [ ] **Step 3: Move stores and utilities**

Move store bodies without changing transitions. Adapt type imports to the
canonical generated module, route selection to injected
`navigate(jobId?: number)`, and daemon health to direct `/api/status`. Keep
the 30-second available poll and 1-second recovery poll. Compose all stores with
one client, runtime, and workflow owner.

- [ ] **Step 4: Run the moved suite**

Run:
`bun x vitest run web/src/lib/stores web/src/lib/utils/roborev-*.test.ts`

Expected: PASS.

- [ ] **Step 5: Commit**

Commit: `feat(web): move review state and workflows`

---

### Task 4: Move Markdown and review components

**Files:**

- Create: `web/src/lib/markdown/render.ts`
- Create: `web/src/lib/markdown/render.test.ts`
- Create: `web/src/lib/markdown/render.browser.test.ts`
- Create: `web/src/lib/components/reviews/*.svelte`
- Create: `web/src/lib/components/reviews/*.test.ts`
- Create: `web/src/lib/components/reviews/test/*`
- Create: `web/src/lib/keyboard/modal-stack.svelte.ts`
- Create: `web/src/lib/keyboard/modal-stack.test.ts`
- Modify: `web/src/app.css`

**Interfaces:**

- Produces: `renderMarkdownSync(source: string): string` and
    `renderMarkdownEffect(source: string): Effect.Effect<string>`.
- Produces: the moved filter, picker, status, table, row, drawer, review,
    comments, log, prompt, badge, and shortcut components.

- [ ] **Step 1: Move tests first**

Move sanitizer, task-list, Shiki-theme, badge, filter, picker, row, drawer,
daemon-status, help-modal, footer-layout, and composer-inset tests. Preserve
assertions; change only runner imports, generated types, and context harnesses.

- [ ] **Step 2: Verify the tests fail**

Run:
`bun x vitest run web/src/lib/markdown web/src/lib/components/reviews`

Expected: FAIL because the renderer and components are absent.

- [ ] **Step 3: Move the renderer**

Move the Marked extensions, code-fence planning, Shiki singleton, nonce-bound
style sanitizer, image sanitizer, Mermaid fences, DOMPurify configuration,
synchronous fallback, and asynchronous highlighting path. Rename the generated
marker to `data-roborev-shiki`. Remove only provider item-reference and
image-proxy rewriting. Keep safe absolute HTTP(S), same-origin, anchor, and
data-image forms after sanitization.

- [ ] **Step 4: Move components and styles**

Move component markup and scoped styles without visual cleanup. Replace only
host context/runtime imports. Keep native comment mutations. Move the global
theme/reset, review status/verdict tokens, and Markdown rules required by the
components; omit unrelated tokens.

- [ ] **Step 5: Verify and commit**

Run:

```bash
bun x vitest run web/src/lib/markdown web/src/lib/components/reviews
bun run --cwd web check
bun run --cwd web build
```

Expected: PASS.

Commit: `feat(web): move review presentation and markdown`

---

### Task 5: Route and compose the native review application

**Files:**

- Create: `web/src/lib/router/router.svelte.ts`
- Create: `web/src/lib/router/router.test.ts`
- Create: `web/src/lib/views/ReviewsView.svelte`
- Create: `web/src/lib/views/ReviewsView.test.ts`
- Create: `web/src/lib/views/AnalyticsPlaceholder.svelte`
- Create: `web/src/lib/components/AppShell.svelte`
- Modify: `web/src/App.svelte`
- Modify: `web/src/App.test.ts`
- Modify: `web/src/app.css`

**Interfaces:**

- Produces: `/reviews`, `/reviews/:job-id`, and `/analytics`.
- Produces:
    `navigateToReview(jobId?: number, options?: { replace?: boolean })`.
- Consumes: one authenticated runtime and store composition.

- [ ] **Step 1: Write failing route/composition tests**

Cover initial deep links, invalid IDs, push/replace history, popstate, selection
sync, runtime disposal, polling/event cleanup, and the analytics placeholder.
Change the App success assertion from the foundation card to the job table.

- [ ] **Step 2: Verify tests fail**

Run:
`bun x vitest run web/src/lib/router web/src/lib/views web/src/App.test.ts`

Expected: FAIL because routing and composition do not exist.

- [ ] **Step 3: Move the view and keyboard behavior**

Move the review view. Preserve `j`, `k`, arrow panel expansion, Enter,
Escape, `x`, `r`, `a`, `c`, `l`, `p`, `y`, `h`, `/`, and
`?`. The modal stack owns Escape while shortcut help is open. App creates the
runtime/stores once after bootstrap and disposes them on logout or unmount.
`/analytics` remains an explicit placeholder.

- [ ] **Step 4: Verify and commit**

Run:

```bash
bun x vitest run web/src/lib/router web/src/lib/views web/src/App.test.ts
bun run --cwd web check
```

Expected: PASS.

Commit: `feat(web): route the transplanted review workspace`

---

### Task 6: Seed a real disposable daemon database

**Files:**

- Create: `internal/testutil/webfixture/seed.go`
- Create: `internal/testutil/webfixture/seed_test.go`
- Create: `internal/testutil/cmd/seed-web/main.go`

**Interfaces:**

- Produces: `webfixture.Seed(path string) error`.
- Produces: `go run ./internal/testutil/cmd/seed-web -out <scratch.db>`.

- [ ] **Step 1: Write the failing fixture test**

Seed a temporary path, open it through `storage.Open`, and assert at least 50
rows across two synthetic projects, every terminal/running state, compact
review output, a synthesis parent with two members, priced/zero/unpriced usage,
open/closed reviews, comments, and stable mutation targets. Assert roots begin
with `/workspace/project-` and contain no home path or external hostname.

- [ ] **Step 2: Verify the fixture test fails**

Run: `go test ./internal/testutil/webfixture -count=1`

Expected: FAIL because the package is absent.

- [ ] **Step 3: Implement through the current schema**

Call `storage.Open(path)` first, then insert deterministic rows through its
embedded `*sql.DB`. Do not copy the production schema into the fixture. Use
UTC timestamps and current columns.

- [ ] **Step 4: Verify and commit**

Run:

```bash
go test ./internal/testutil/webfixture -count=1
scratch_dir="$(mktemp -d)"
go run ./internal/testutil/cmd/seed-web -out "$scratch_dir/reviews.db"
```

Expected: PASS.

Commit: `test(web): add deterministic review fixtures`

---

### Task 7: Port the real-daemon Playwright parity lane

**Files:**

- Create: `web/playwright.config.ts`
- Create: `web/scripts/e2e.ts`
- Create: `web/scripts/e2e.test.ts`
- Create: `web/tests/reviews.spec.ts`
- Create: `web/tests/security.spec.ts`
- Create: `web/tests/support/daemon.ts`
- Create: `web/tests/support/reviews.ts`
- Modify: `web/package.json`
- Modify: `package.json`
- Modify: `.github/workflows/ci.yml`
- Modify: `Makefile`

**Interfaces:**

- Produces: `bun run --cwd web test:e2e`.
- Produces: one scratch embedded daemon and Chromium session pointed at its
    published browser origin.

- [ ] **Step 1: Write the failing runner isolation test**

With injected spawn/temp functions, assert scratch HOME, data/config/database
paths, loopback addresses, cleared inherited endpoint/token variables, runtime
readiness, and child cleanup on success, failure, SIGINT, and SIGTERM.

- [ ] **Step 2: Verify the runner test fails**

Run: `bun x vitest run web/scripts/e2e.test.ts`

Expected: FAIL because the runner is absent.

- [ ] **Step 3: Implement the isolated runner**

Build and embed the SPA into a scratch binary without touching a user PATH.
Seed the database, write scratch config, launch the daemon, read runtime
metadata, and invoke Playwright with `ROBOREV_E2E_ORIGIN`. Restore the tracked
asset stub on every exit.

- [ ] **Step 4: Port parity scenarios**

Move reference scenarios and adapt only native-shell selectors and direct
`/api/*` URLs. Cover table data/pagination/sorting/filters; standalone,
compact, and panel detail; output/comments/log/prompt; close/reopen,
cancel/rerun; shortcuts; deep links/history/refresh; event and output reconnect;
unavailable/recovery; loading/empty/stale/error; and responsive overflow.

Add local and fresh-tab bootstrap, header enforcement, CSRF rejection and
acceptance, disallowed routes, restart login, and actual Markdown/Shiki/Mermaid
rendering under the shipped CSP.

- [ ] **Step 5: Run and wire the real browser lane**

Run: `bun run --cwd web test:e2e`

Expected: PASS against only the script-created daemon.

Add the lane to CI after release asset validation. Upload traces only on
failure.

- [ ] **Step 6: Commit**

Commit: `test(web): prove native review parity end to end`

---

### Task 8: Complete the transplant checkpoint

**Files:**

- Modify: `README.md`
- Modify: `docs/development.md`
- Modify: `docs/superpowers/specs/2026-08-12-native-web-ui-design.md`

**Interfaces:**

- Produces: a review-parity checkpoint ready for projection/package extraction.

- [ ] **Step 1: Update current-state documentation**

Replace foundation-only language with the available native review workflow and
real-browser command. Do not describe analytics or package extraction as done.

- [ ] **Step 2: Run every gate**

Run:

```bash
go test ./...
PATH="/tmp/roborev-web-tools.lg3fiD:$PATH" make lint-ci
bun run web:test
bun run web:check
bun run web:build
bun run --cwd web test:e2e
make api-check
make web-release-check
make release-snapshot-check
CGO_ENABLED=0 go build ./...
git diff --check
```

Expected: PASS with the compilation stub restored.

- [ ] **Step 3: Scrub and publish the checkpoint**

Scan every unpushed commit, current diff, fixture, test, trace, screenshot, and
pull-request update against the canonical private-term denylist and structural
heuristics. Require zero hits. Commit documentation as
`docs(web): document the native review workspace`, push this branch, and
update the existing pull request to state only behavior demonstrated by the
real-daemon lane.
