# Native Web UI Design

**Status:** Approved design for implementation planning

## Summary

Roborev will own and serve its browser application. The first milestone moves
the existing review interface into this repository with behavior parity. Later
milestones extract reusable review presentation components, reduce the existing
host integration to a small contextual view, and add a first-class analytics
workspace backed directly by Roborev's SQLite database.

This is a code movement project before it is a redesign. The existing review
experience remains the reference implementation until the native application
passes its parity gate. The new shell and analytics workspace follow that
cutover instead of competing with it.

Public documentation and project history must describe integrations and
deployment generically. They must not identify private repositories, machines,
hostnames, dashboards, or infrastructure.

## Goals

- Make Roborev the owner of its full browser application.
- Serve the compiled single-page application from the Roborev daemon.
- Preserve the existing Reviews behavior before changing its visual design.
- Provide native cost, latency, failure-rate, and review-outcome analytics from
    SQLite, grouped by the project names users already see in Roborev.
- Support secure access through a loopback reverse proxy, including private
    network overlays, without exposing the existing CLI API listener.
- Publish a supported, network-free Svelte presentation package that another
    application can use to render inline review content without copying Roborev
    code.
- Retain live updates, streaming output, mutations, keyboard navigation, deep
    links, and responsive behavior in the native application.
- Keep release builds reproducible and prevent stale or missing web assets from
    entering a Roborev binary.

## Non-goals

- Rewriting the review interface from scratch during the migration.
- Publishing the presentation package to the npm registry in the first release.
- Giving the presentation package network access, stores, or mutation logic.
- Keeping a second complete review client in the existing host application.
- Moving analytics into PostgreSQL or changing SQLite into a metrics warehouse.
- Adding CSV export, cross-machine analytics federation, or a general charting
    framework in the first release.
- Making deep links portable between independently running daemons.
- Persisting browser sessions across daemon restarts.

## Ownership and repository layout

The repository becomes a Bun workspace with the following frontend units:

```text
web/                         Full Svelte application
packages/roborev-ui/         Reusable review presentation package
internal/daemon/             Browser listener, API, and embedded assets
internal/storage/            Analytics queries over SQLite
```

### `web/`: the complete application

The application owns all stateful and networked behavior:

- routing and deep-link resolution;
- the daemon client and generated OpenAPI types;
- review, job, log, prompt, queue, and analytics stores;
- event and output streams;
- filters, selections, preferences, and retry behavior;
- mutations such as close, reopen, cancel, rerun, and comment creation;
- authentication, session bootstrap, and cross-site request forgery (CSRF)
    handling;
- keyboard dispatch, modal stacking, and the application shell.

The application retains Effect for the transplanted streaming and concurrency
machinery. It copies the small generic streaming-fetch, retry-policy,
ordered-command-queue, error, and CSRF primitives into a narrow Roborev runtime.
It does not adopt the broad runtime type of the existing host.

### `packages/roborev-ui/`: supported presentation API

`@kenn-io/roborev-ui` is a network-free, read-only Svelte package. Its public
components accept typed values and callbacks. They do not create clients,
subscribe to streams, instantiate application stores, or depend on a host
runtime.

The package owns the complete rendering pipeline for review output:

- sanitized Markdown;
- syntax highlighting;
- Mermaid diagrams;
- verdict and status presentation;
- findings and review metadata;
- panel member and synthesis attribution;
- existing comments and responses;
- neutral host actions such as selecting a finding or opening a link.

The minimum supported export surface is `ReviewProjectionView` for the complete
read-only projection, plus composable `ReviewContent`, `ResponseList`,
`ReviewMetadata`, `PanelAttribution`, `VerdictBadge`, and `StatusBadge`
components. It also exports the projection and callback types used by those
components. Application-only job tables, filters, drawers, stores, and clients
remain private to `web/`. This gives contextual hosts both a safe high-level
entry point and enough smaller pieces to compose inline review layouts without
making the whole native application public API.

The package does not contain a comment composer. Mutations remain native-app
behavior. A host may place controls around package components, but the package
does not define a mutation transport.

The package ships Svelte and TypeScript source. Its exports point to source and
its package `files` allowlist defines the supported archive. Consumer Vite
builds compile the Svelte source. Continuous integration tests the packed source
archive, including its exports and allowlist, instead of assuming a compiled
package artifact.

The first distribution mechanism is an immutable repository tag or commit. This
allows the package and daemon to release from the same source while hosts pin an
exact compatible revision. The package may use the existing shared UI system,
but it owns Roborev-specific token defaults and documented override points so it
renders correctly outside the native application.

### Existing host application after cutover

The existing host deletes its top-level Reviews route, Roborev job and review
stores, log stores, stream management, and duplicated daemon client after native
parity and package extraction.

Its contextual sidebar becomes a narrow consumer:

1. A small host adapter identifies the active repository and branch.
2. It fetches the daemon's review projection endpoint.
3. It renders read-only review content through `@kenn-io/roborev-ui`.
4. It offers an **Open in Roborev** action when the daemon publishes a browser
    origin.

The sidebar refreshes on mount, workspace activation, browser focus, a modest
visible-only polling interval, and manual refresh. It shows stale and error
states explicitly. It does not retain a Roborev event-stream stack.

If the daemon does not advertise the projection capability, the adapter treats
an absent endpoint as an old daemon and shows an update affordance. Once the
capability is advertised, a missing review is a normal not-found state. An
unsupported projection `schema_version` also shows the update affordance.

If no browser origin is configured, the sidebar replaces **Open in Roborev**
with a setup hint. It never emits a dead link.

## Daemon browser surface

### Listener separation

The existing CLI, TUI, and automation listener remains unchanged. It retains its
loopback or Unix-socket transport and all existing routes, including shutdown
behavior.

A separate browser listener serves the single-page application and a strict
route allowlist. Its allowed API surface includes:

- browser session login, bootstrap, logout, and status;
- review and job routes needed by the application;
- the review projection and capability information;
- analytics and launch lookup;
- the event stream and streaming job output;
- static application assets and browser health.

Administrative routes such as daemon shutdown remain available only on the
existing listener. Unknown routes under `/api/` return not found; they do not
fall back to the single-page application.

The browser listener defaults to loopback. A configured listener address and
public origin support a same-host reverse proxy. Any non-loopback exposure
requires remote authentication and an HTTPS public origin. Runtime metadata
publishes the actual browser origin and capabilities but never publishes
credentials.

### Embedded application

Go embeds the validated frontend output. Only a small compilation stub is
tracked in source control; generated `web/dist` assets are not. Local builds,
continuous integration, and release packaging build the Bun workspace before the
Go binary and fail if the expected manifest or entry assets are missing.

Hashed assets receive immutable cache headers. The application shell receives
no-cache headers. History fallback applies only to browser navigation routes
such as `/reviews/123` and `/analytics`.

### Generated contracts and compatibility

Roborev's OpenAPI source is canonical. Browser types and the review projection
wire model are generated from it. A narrow projection route returns the data
needed to render one contextual review without exposing an application store:

```text
GET /api/ui/review-projection
```

The request accepts exactly one selector mode: a daemon-local job ID, or an
exact repository path with an optional exact branch. The contextual selector
returns the newest matching logical review. Repository paths are request-only
lookup keys and do not appear in the projection response. Invalid mixed or
incomplete selectors return a typed bad-request error.

The response carries an explicit `schema_version`. Changes within a supported
version are additive, and consumers ignore unknown fields. A breaking response
change increments the version. The package declares the versions it supports and
degrades to an update-required state for an incompatible response.

Daemon status or capability metadata advertises projection support before a
consumer requests an individual review. This distinguishes a pre-projection
daemon from a supported daemon on which a requested review does not exist.

The projection endpoint lands before package extraction. Its handler response
type comes from the canonical API definition. Roborev continuous integration
renders a response produced by the real daemon handler through the package. The
native application maps live store state into that same projection type; it does
not define a parallel component input shape.

### Launching and deep links

`roborev ui` opens the configured browser origin. `roborev ui <job-id>` opens
`/reviews/<job-id>`. The initial route deliberately uses the daemon-local
numeric job identifier because it preserves the transplanted client and every
generated link targets that same daemon. These links are not promised to work
against a different daemon, where synchronization can assign a different local
identifier. Job UUID routing is a future compatible extension, not an implicit
first-release guarantee.

When no browser listener is available, the command returns an actionable setup
message rather than constructing a guessed URL.

## Browser security model

The static application shell contains no private data and may load before
authentication. Every data-bearing browser endpoint requires an authenticated
browser session, except for the narrowly defined login and bootstrap flows.

### Request authority

The daemon validates the HTTP `Host` before authentication. Accepted authorities
come from the listener address and configured public origin. Forwarded headers
may help construct diagnostics but never expand accepted hosts, establish a
local request, or confer authentication authority.

The daemon validates exact origins for browser requests. It does not use suffix
matching, wildcard origins, or reflected origins. Redirects are not part of the
session protocol.

### Local session path

A browser request may create a session without a configured token only when all
of these conditions hold:

- the direct peer is loopback;
- the `Host` is an accepted loopback authority;
- the `Origin` is the exact expected origin;
- forwarding headers are absent; and
- remote authentication is not configured.

Requiring every condition prevents a same-host reverse proxy from accidentally
turning remote traffic into trusted loopback traffic. When remote authentication
is configured, a local operator also logs in with the token.

### Remote token exchange

Remote access configures an exact public origin and a daemon authentication
token through secret-aware configuration. The browser sends the token only to
the login endpoint. The server does not place it in a URL, log it, return it, or
store it in browser persistence. The application discards it after exchange.

`POST /api/ui/session/login` requires the exact `Host` and `Origin`. It creates
an in-memory ambient session and sets a host-only, `HttpOnly`, `SameSite=Strict`
cookie with `Path=/`. Logout expires the cookie with the same path and security
attributes. The cookie has `Secure` when the configured public origin uses
HTTPS; an HTTP loopback-only origin omits it. The response returns a tab token
plus a CSRF token. The cookie name includes a random daemon instance identifier
because cookies do not distinguish localhost ports.

Sessions intentionally live only in daemon memory. A daemon restart changes the
instance identifier and invalidates all cookies and tab tokens. Remote users
must enter the daemon token again after a restart or upgrade. This recurring
login is an intentional first-release tradeoff; persistent sessions are not part
of this design.

### Fresh-tab bootstrap

A deep link opened in a new tab has the ambient cookie but no tab-scoped header.
`POST /api/ui/session/bootstrap` is the only cookie-only authenticated endpoint.
It requires:

- a valid instance-scoped ambient cookie;
- the exact accepted `Host` and `Origin`;
- `POST` with the expected JSON content type;
- same-origin Fetch Metadata values; and
- no redirect or forwarded-header authority.

The endpoint mints a new random tab token and CSRF token. The server-side token
record binds the tab token to the ambient session ID, daemon instance ID,
principal, and expiry. Bootstrap does not rotate the ambient cookie or
invalidate other tabs.

The application stores the tab and CSRF tokens only in `sessionStorage`. They
therefore disappear when the tab closes. A remote user without a valid ambient
cookie sees the login screen. When no remote token is configured,
`POST /api/ui/session/bootstrap` itself creates the ambient local session and
returns its first tab credentials, but only after every local trust condition
and Fetch Metadata check succeeds.

### Authenticated requests, CSRF, and streams

Every authenticated data request carries both the ambient cookie and the
tab-scoped session header. Mutation requests additionally carry the CSRF token
and pass exact-origin validation. The native CSRF implementation uses the
daemon's scheme; it does not retain a header or session convention from the old
host.

All live streams use header-capable `fetch`. Native `EventSource` is prohibited
because it cannot send the tab credential. The existing fetch-based streaming
reader is retained for both server-sent-event-compatible notifications and
newline-delimited JSON job output. Reverse proxies must disable buffering for
these responses.

### Development origin

`make web-dev` starts a disposable daemon with a temporary data directory,
synthetic database, and explicit allowed Vite origin. The Make target passes
that origin through a daemon flag or config value. Production code never
recognizes a development hostname or relaxes origin checks automatically. The
Vite proxy exercises the real session and CSRF contract.

### Content Security Policy

The initial Content Security Policy uses self-hosted assets and no remote
origins. It includes:

```text
default-src 'self'
base-uri 'none'
object-src 'none'
frame-ancestors 'none'
form-action 'self'
script-src 'self' 'wasm-unsafe-eval'
style-src 'self'
style-src-attr 'unsafe-inline'
style-src-elem 'self' 'unsafe-inline'
img-src 'self' data:
font-src 'self'
connect-src 'self'
```

The transplanted Markdown pipeline currently uses Shiki's default
`getSingletonHighlighter` without an engine override. That selects the Oniguruma
WebAssembly engine, so the narrow `wasm-unsafe-eval` permission is an explicit
parity choice; ordinary JavaScript evaluation remains forbidden. A future switch
to Shiki's JavaScript regex engine can remove that permission only after
highlighting parity and performance tests.

Shiki emits style attributes, while Mermaid can embed a `<style>` element in its
rendered SVG. The two `unsafe-inline` style directives are therefore explicit
and limited to their respective style sinks. DOMPurify sanitizes Mermaid SVG
before insertion, and the existing Shiki hook admits only the expected generated
style values. Browser tests verify actual Markdown, highlighting, and Mermaid
rendering under the shipped policy.

## Migration and cutover plan

The migration proceeds in dependency order. The existing Reviews route remains
the reference and rollback path until the native parity gate passes.

### Stage 1: frontend workspace foundation

- Add the root Bun workspace with `web` and `packages/*` members.
- Establish Svelte, TypeScript, Vite, Vitest, and browser-test configuration.
- Add generated-client wiring from Roborev's canonical OpenAPI source.
- Define the source-shipping package exports and strict package allowlist.
- Add release checks that build and validate embedded frontend assets before Go
    compilation.

### Stage 2: daemon browser track

This Go track runs in parallel with the UI transplant and gates integration
testing:

- add the separate browser listener and route allowlist;
- add host and origin validation before authentication;
- implement local sessions, token login, tab bootstrap, logout, CSRF, expiry,
    and restart invalidation;
- add runtime browser-origin and capability metadata;
- serve and embed validated application assets;
- add `roborev ui` and its job deep link;
- provide the isolated Vite development proxy and daemon target.

### Stage 3: transplant the review application

Move the review view, components, stores, client patterns, styles, tests, and
supporting utilities before restructuring them. The copy list explicitly
includes the complete Markdown renderer and its sanitizer, syntax-highlighting,
and Mermaid support.

Some parity behavior requires small native shell infrastructure rather than a
file copy. Build a narrow router, keyboard dispatcher, modal stack, Roborev
Effect runtime, and daemon-specific CSRF integration. Adapt client base URLs
from the old proxy prefix to direct `/api/*` routes.

Port the complete browser-test infrastructure, not only component tests. This
includes the real-daemon Playwright suites, Go fixture seeder, Docker setup, and
test helpers. The new home is simpler because the tests and daemon share the
repository.

The parity gate covers:

- project and branch filters;
- project tree and job table behavior;
- standalone, compact, and panel reviews;
- review details, comments, and panel attribution;
- close, reopen, cancel, and rerun mutations;
- log and prompt views;
- keyboard shortcuts and help modal;
- direct job links and browser navigation;
- event and output-stream reconnect behavior;
- loading, empty, stale, and error states; and
- supported responsive layouts.

### Stage 4: projection contract and package extraction

- Add the versioned review projection handler and capability advertisement.
- Generate its wire type from OpenAPI.
- Test a real handler response through the presentation package.
- Make the native application the first package consumer.
- Extract read-only review rendering and the complete Markdown pipeline.
- Test the package archive as source, including imports from an independent
    fixture consumer.

The native live stores map their state to the projection type. This ensures the
same contract is continuously exercised before any other host depends on it.

### Stage 5: host cutover

- Pin an immutable `@kenn-io/roborev-ui` revision.
- Replace the old review sidebar stack with the projection adapter.
- Add capability, old-daemon, incompatible-schema, not-found, stale, and
    missing-browser-origin states.
- Remove the top-level Reviews route and duplicated Roborev network stack only
    after native parity and package rendering pass.
- Delete the old Roborev browser projects except for a narrow sidebar-adapter
    specification.

### Stage 6: shell and analytics

After cutover, add the two-workspace shell and analytics view. Review visual
cleanup remains separate from the move so it cannot disguise parity failures.

### Implementation-plan decomposition

This design is one end-to-end compatibility contract, but it is too broad for
one implementation checklist. After approval it becomes four dependency-ordered
plans: workspace and browser security; review transplant and parity; projection,
package, and host cutover; then analytics and shell refinement. Each plan has
its own focused verification and review checkpoint. A later plan may start only
after the contract it consumes has landed or has a tested branch artifact.

## Analytics contract

### Request

The native analytics API is:

```text
GET /api/ui/analytics
```

It accepts:

- RFC 3339 `since` and `until` bounds;
- repeated exact project-name filters;
- repeated stored source values;
- optional agent and model filters; and
- a requested or server-selected UTC time bucket.

Agent and model filters apply only to agent-attempt metrics. They do not remove
or duplicate logical reviews or alter review volume, failure rate, review
latency, or verdict mix. Project and source filters apply to both populations.

The default range is 30 days. The UI offers 24-hour, 7-day, 30-day, 90-day,
year, and all-time presets. Inputs convert at the boundary to the UTC timestamp
format used by SQLite and compare through `datetime()` so local and synchronized
rows remain compatible. Source parameters use stored machine values such as
`post_commit` and `auto_design`; display labels may use punctuation and spaces.

Time windows are half-open: `since` is inclusive and `until` is exclusive. An
all-time request omits `since`. Supported explicit buckets are `hour`, `day`,
`week`, and `month`; automatic selection chooses one from the requested span.
Unless a metric says otherwise, a terminal row belongs to the window containing
its `finished_at` value.

### Project identity

The project dimension is exactly `repos.name`, the same identity displayed in
the Roborev interface. Multiple repository records with the same displayed name
aggregate intentionally. The API does not expose filesystem paths to distinguish
them.

All sources are included by default. The source filter supports a focused CI
view without making CI the meaning of the headline metrics.

### Metric populations

A **logical review** represents one user-facing review:

- standalone review, range, and dirty jobs;
- a panel synthesis parent, excluding its member rows; and
- a legacy compact job, which also represents one review.

Task, fix, classifier, insights, and other non-review jobs are excluded from
headline review metrics. Future analytics may expose them as a separate scope.

An **agent attempt** is each cost-eligible invocation. It includes panel members
and an invoked synthesis job because each can consume time and money.

### Headline metrics

- **Volume:** terminal logical reviews in the selected range.
- **Failure rate:** failed logical reviews divided by done plus failed logical
    reviews. Canceled and skipped counts are reported separately and do not
    enter the denominator.
- **Review latency:** enqueue-to-finish duration for successful logical reviews.
    A panel synthesis parent therefore spans the complete panel lifecycle.
- **Agent execution:** start-to-finish duration for attempts, broken down by
    agent and model.
- **Cost:** recorded dollar cost for every eligible attempt whose `finished_at`
    falls in the range, including failed and canceled attempts and retained
    panel rows.
- **Cost coverage:** priced eligible attempts divided by all eligible attempts.
    A real zero-dollar attempt counts as priced; a null or unreadable cost does
    not.
- **Verdict resolution:** pass, fail-open, and fail-closed. A closed passing
    review remains pass.

Latency summaries report p50, p90, and p99 using the storage package's existing
linear-interpolation percentile function. The headline card uses p50 while the
detail view exposes all three.

The closed flag is mutable. Historical fail-open versus fail-closed results are
therefore evaluated at query time and can change after users address reviews.
Job status, duration, and cost metrics retain their recorded event-time meaning.

### Cost semantics

Analytics reuses the existing `costEligible` and `hasCost` storage predicates.
It must not rederive eligibility from a looser flag or JSON check. In
particular:

- `has_cost` without a non-null readable `cost_usd` is unpriced;
- a stored zero value is a valid priced attempt;
- a synthesis passthrough that never invoked an agent is not eligible; and
- usage evidence remains the fallback for rows that predate the invocation
    marker.

The analytics time cut deliberately differs from the existing cost aggregate:
analytics uses `finished_at` because it accounts for attempts completed in the
selected period, while the existing aggregate applies its lower bound to
`enqueued_at`. Tests and comments preserve this distinction.

Cost remains best effort. Requeueing clears a job's token-usage record, so an
overwritten earlier retry cost cannot be recovered from the current schema.
Coverage describes recoverable attempt records, not external billing
completeness. The UI always displays coverage beside cost.

### Query implementation and response

All subqueries run inside one SQLite read transaction to provide a coherent
snapshot. SQL computes counts and sums. It returns only narrow duration values
for exact percentile calculation in Go; it never loads review output, prompts,
diffs, or logs for analytics.

The response contains:

- request metadata and normalized filters;
- headline summary metrics;
- UTC time-series buckets;
- project rows;
- source, agent, and model breakdowns;
- filter option values; and
- explicit cost coverage and terminal-state counts.

Errors and empty results use typed envelopes. A single request either returns a
coherent snapshot or an error; it does not mix independently refreshed panels.

No schema migration or index is added initially. Representative query plans and
benchmarks determine whether an index is necessary. Any later index must have a
measured query benefit and write-cost justification.

## Application experience

The application has two primary workspaces:

- `/reviews` and `/reviews/:job-id` contain the transplanted review application;
- `/analytics` contains the analytical view; and
- `/` redirects to Reviews.

Navigation, theme, daemon status, authentication, and reconnect state belong to
the shared shell. The Reviews workspace stays visually faithful during the
migration. The shell is intentionally modest until parity is established.

The analytics workspace uses the shared design system for cards, filters,
tables, menus, loading states, and empty states. Small app-owned SVG components
render time series and distributions. This avoids a chart-framework dependency
until requirements justify one. Each chart has an accessible text summary or
table representation.

The page contains:

- headline cards for logical reviews, failure rate, review latency, attempt
    cost, and cost coverage;
- volume, failure-rate, latency, and cost time series;
- a project table keyed by exact displayed project name;
- source, agent, and model breakdowns; and
- explicit canceled, skipped, pass, fail-open, and fail-closed values.

Analytics filters live in the URL: range, projects, sources, agent, model, and
bucket. Links are shareable and browser history is predictable. Local storage
may retain presentation preferences, but not analytical identity.

Filter changes abort obsolete requests. A late response may update the page only
when its normalized query still matches the active query. The application never
shows old data under new filters. A failure while refreshing the same query may
retain its prior snapshot with a clear stale badge and retry action; a failed
new query shows an error instead of unrelated old data.

Relevant daemon events trigger a debounced refresh only while Analytics is
visible. Manual refresh and browser-focus refresh remain available. This keeps
recent results current without running an independent analytics event model.

## Verification strategy

### Storage tests

Synthetic SQLite fixtures cover:

- standalone, panel, and compact logical-review classification;
- exclusion of panel members and non-review job types;
- duplicate displayed project-name aggregation;
- source, agent, model, and time-bound filters;
- done, failed, canceled, and skipped denominator behavior;
- mutable fail-open and fail-closed state;
- enqueue-to-finish and start-to-finish percentiles;
- cost eligibility, valid zero cost, malformed or null cost, and passthrough
    synthesis rows;
- overwritten retry usage and coverage limitations; and
- a coherent transaction snapshot.

### Daemon and contract tests

Handler tests cover input normalization, errors, empty results, response schema
versions, generated wire types, capability advertisement, launch lookup, and
projection not-found behavior.

Security tests cover Host-before-auth, exact Origin checks, forwarded-header
rejection, local-session gates, token login, fresh-tab bootstrap, cookie naming
and flags, tab binding, CSRF, expiry, logout, restart invalidation, and browser
route isolation.

### Frontend and package tests

Unit and component tests cover stores, filter URL state, cancellation races,
stale data, review projection rendering, Markdown sanitization, Shiki style
filtering, Mermaid sanitization, and incompatible schemas.

The package test packs the real source archive and installs it into an
independent fixture application. A contract test passes output from the real
projection handler through the package renderer.

### Real-browser tests

The ported Playwright suite runs against a real disposable daemon and synthetic
SQLite database. It covers:

- the complete Reviews parity checklist;
- local session creation and remote token login;
- a fresh-tab deep link bootstrapped from an ambient cookie;
- login after daemon restart;
- fetch-based streams carrying the required header;
- CSRF rejection and accepted mutation flows;
- shipped CSP behavior for Markdown, Shiki WebAssembly, and Mermaid styles;
- analytics filters, URL history, race cancellation, empty/error/stale states,
    and incomplete cost coverage; and
- supported responsive layouts and basic keyboard accessibility.

Fixture values use reserved domains, synthetic project names, and documentation
addresses. Tests and screenshots contain no real review data or infrastructure
identifiers.

## Documentation and delivery

The cutover includes end-user documentation, not a later cleanup. Public docs
cover:

- `roborev ui` and job deep links;
- enabling and locating the browser listener;
- public-origin and token configuration;
- local behavior when remote authentication is enabled;
- restart-invalidated sessions;
- a generic private-network reverse-proxy recipe, including stream buffering;
- analytics definitions, populations, filters, mutable closed state, and cost
    coverage limitations; and
- frontend development with the disposable Vite/daemon target.

Release validation is local and non-publishing. It proves asset creation,
embedding, cache behavior, package archive contents, and Go builds with the same
steps used by release automation. Deployment configuration outside this
repository moves traffic only after native parity and security checks pass.

## Completion criteria

The migration is complete when:

- the daemon securely serves the native application on its browser listener;
- Reviews passes the real-daemon parity suite;
- `@kenn-io/roborev-ui` renders the versioned projection as a supported
    source-shipping package;
- the existing host uses only the narrow sidebar adapter and package;
- Analytics reports the approved SQLite metrics and their limitations;
- deep links, fresh tabs, streams, CSP, and restart authentication behave as
    specified;
- release and package validation prevent incomplete artifacts; and
- public docs describe operation and deployment without disclosing private
    project or infrastructure details.

## Implementation checkpoint: native Reviews

As of 2026-08-13, the daemon browser listener, browser session boundary,
release-asset pipeline, and code-moved Reviews application are implemented. The
native application owns the review list, project/status/ref filters, sorting,
pagination, panel expansion, deep-linked detail, Markdown rendering, comments,
logs, prompts, mutations, keyboard controls, and reconnect/recovery behavior.

The parity lane builds the production SPA, embeds it in a scratch binary, seeds
the production SQLite schema with synthetic data, starts the real authenticated
browser listener, and runs 24 Chromium scenarios. The lane also checks fresh-tab
bootstrap, session and CSRF headers, forbidden browser routes, shipped content
security policy behavior, stale-data handling, and narrow-layout scrolling.

This checkpoint does not complete the migration. The projection endpoint,
source-shipping presentation package, contextual host adapter, native analytics,
and deployment cutover remain separate follow-on plans.
