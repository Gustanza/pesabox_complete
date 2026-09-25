# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this is

This repo holds **two things**:

1. **The product — HelaBox (formerly PesaBox)**, a savings-group (VSLA) platform for the client GATA, built by the Dephics team: `server/` (Go API), `web/` (Vue super-admin dashboard) and `pesa_box_app/` (Flutter mobile app — its **own nested git repo**, `Gustanza/pesabox`, git-ignored here). Most work happens here. See "HelaBox product" below.
2. **The Yekonga framework** it runs on: `github.com/robertkonga/yekonga-server-go` is a Go framework library (package `yekonga`) — `yekonga.ServerConfig(configFile, databaseFile)` spins up an HTTP server with REST + auto-generated GraphQL + WebSocket/Socket.IO + cloud functions, backed by MongoDB, MySQL, generic SQL, or a local JSON file store. `server/main.go` is the only `main` package that uses it.

**Work tracking:** `TODO.md` (repo root) is the task list agreed with the user — task IDs (U7, P2, …), a checkbox per task (tick only when done *and* verified), a status tag per section, recommended client decisions in §2, and a **Progress log** table at the bottom (add a row per finished chunk). Check it first when asked to "continue the TODO". The client's role spec is `Pesa Box User Levels.pdf` (repo root). Only commit/push when the user asks — the user commits themselves.

## Commands

There is no Makefile, linter config, or CI file in this repo — just plain tooling.

```bash
go build ./...                                    # build everything
go build ./yekonga/... ./config/... ./datatype/... ./helper/... ./gateway/... ./server/...   # first-party code only (skips vendored plugin examples that don't compile standalone)
go vet ./server/ ./yekonga/...                    # static checks (yekonga has a few old vet warnings — not ours to chase)
gofmt -l server/                                  # formatting check
go test ./server/                                 # server tests (reports, SMS templates)
(cd server && go build -o pesabox-server.exe .)   # build the API binary (git-ignored)
(cd web && npm install && npx vite build)         # web build check — run npm install after every pull
(cd pesa_box_app && flutter analyze && flutter test)   # app checks (10 old "info" hints in app_data.dart are pre-existing)
```

The framework itself still has no `_test.go` files outside `plugins/` (vendored third-party code). When adding tests for framework code, prefer `go test -race` — see the mutex note below.

`plugins/graphql/examples/httpdynamic` fails a plain `go build ./...` with "function main is undeclared" — this is a pre-existing vendored example, not a bug to fix.

To exercise the framework as a consumer would, a test app needs its own `config.json` and `database.json` (see `README.md` "Getting Started" for full field references, and `documentation/development/config.json` / `documentation/development/database.json` for minimal examples) and calls:

```go
app := yekonga.ServerConfig("./config.json", "./database.json")
app.Get("/", handler)
app.Start(":8080")
```

## Repo layout

- `server/` — the HelaBox API (`package main`): `main.go` (group routes `/api/main/*`, members, transactions, reversal), `access.go` (roles/Actor/migration), `guard.go` (GraphQL guard), `access_routes.go` (users, assignments, audit, officers), `structure_routes.go` (partners, clusters, safe deletes), `gov_loans.go`, `reports.go` + `report_kpis.go`, `sms.go`, `brand.go`; schema in `server/database.json`.
- `web/` — Vue 3 + Vite super-admin dashboard (`src/views`, `src/api`, `src/i18n`).
- `pesa_box_app/` — Flutter app, **its own git repo** (commit/pull there separately).
- `TODO.md` — the client-change task list and progress log (see "What this is").
- `yekonga/` — the framework itself (single package `yekonga`). Routing, request/response, middleware, models, query builder, GraphQL, cloud functions, cron, DB connections, sockets all live here as top-level files (`main.go`, `request.go`, `response.go`, `middleware.go`, `model.go`, `model_query.go`, `graphql.go`, `dbconnect*.go`, `cloud_functions.go`, `cronjob.go`, `socket.go`, ...). Treat this as one large package — grep across it rather than assuming a file's name fully scopes its contents.
- `config/` — parses `config.json` into `YekongaConfig`.
- `datatype/` — shared generic types (e.g. `DataMap`).
- `helper/` — large stdlib-style utility grab bag (type conversion/checking, string/date manipulation, data extraction, mail, images, serial numbers).
- `gateway/` — SMS/WhatsApp provider integrations (Beem, Infobip) behind a common `gateway/setting` interface.
- `plugins/` — **vendored/forked third-party source**, not app code: `mongo-driver`, `mysql`, `redigo`, `socketio`, `websocket`, `graphql` (forked graphql-go), `uuid`, and a custom local file-based DB engine (`plugins/database`). None of these have their own `go.mod` — they're imported directly under `github.com/robertkonga/yekonga-server-go/plugins/...`. Don't "fix" these to match upstream unless specifically asked; changes here are effectively patching a vendored fork.
- `documentation/` — a **separate nested git checkout** (its own `.git`), not a submodule of this repo. Contains longer-form internal docs (`documentation/documentations/INTERNAL_*.md`) plus example `config.json`/`database.json` and GraphQL schema dumps. Useful as reference but edits here don't belong to this repo's history.
- `README.md` / `README_GRAPHQL.md` — the consumer-facing docs; `README_GRAPHQL.md` documents the auto-generated GraphQL query/mutation shape for an example schema.

## Architecture

**Startup (`ServerConfig` in `yekonga/main.go`):** load `config.json` → `NewDatabaseStructure` parses `database.json` into model definitions → `NewSystemModels` builds `*DataModel`s → `NewDatabaseConnections` opens the DB backend selected by `config.database.kind` (`mongodb` / `mysql` / `sql` / `local`, strategy pattern across `dbconnect_mongodb.go` / `dbconnect_mysql.go` / `dbconnect_sql.go` / `dbconnect_local.go`) → GraphQL schema auto-built from the same `DataModel`s (`GraphqlAutoBuild`) → returns the `*YekongaData` singleton (`Server`).

**Models are schema-driven, not Go structs.** `database.json` (parsed as `yekonga.DatabaseStructureType`) is the source of truth for field names, types, and options; `NewSystemModels` turns it into `*DataModel`s used by both the REST layer and the GraphQL auto-build. There's no per-model Go type to define — extending the data model means editing `database.json`, not writing a struct.

**Request lifecycle** (`ServeHTTP` in `yekonga/main.go`) runs middleware in this fixed order — built-ins first, then the three user-registrable hook points:

1. `MasterKeyMiddleware`, `ApplicationIDMiddleware` (built-in, always run)
2. `y.preloadMiddlewares` — user hooks registered via `app.Middleware(fn, yekonga.PreloadMiddleware)`
3. `ClientMiddleware`, `TenantCatchMiddleware`, `TokenMiddleware`, `BillingMiddleware`, `UserInfoMiddleware` (built-in auth/tenant/billing pipeline)
4. `y.initMiddlewares` — user hooks via `app.Middleware(fn, yekonga.InitMiddleware)`
5. `y.middlewares` — user hooks via `app.Use(fn)` (global, runs last, right before route dispatch)
6. route handler dispatch; if no route matched, `y.catchMiddlewares` run as the 404 fallback chain

REST and GraphQL requests both terminate in the same data layer: `ModelQuery` (fluent builder — `Where/OrderBy/Take/Skip/Select` → `Find/FindOne/FindById/Create/Update/UpdateMany/Delete/Count`) dispatches to whichever `dbconnect_*.go` backend is active. GraphQL additionally wraps this in the auto-built schema/resolvers, so a new field in `database.json` shows up in both REST and GraphQL automatically without separate wiring.

**Extension points** (called on the `*YekongaData`/`app` instance):
- Routes: `app.Get/Post/Put/Patch/Delete(path, handler)`, `app.All(path, handler)`, `app.Static(StaticConfig{...})`
- Middleware: `app.Use(fn)` (global) or `app.Middleware(fn, yekonga.InitMiddleware|PreloadMiddleware)` (phase-specific, see lifecycle above)
- Cloud functions (callable/backend functions): `app.Define("name", fn)` — a `CloudFunction` with signature `func(interface{}, *RequestContext) (interface{}, error)`
- DB triggers: `app.BeforeCreate/AfterCreate/BeforeUpdate/...("ModelName", fn)` — `TriggerFunction` with signature `func(*RequestContext, *QueryContext) (interface{}, error)`, or the `*All` variants for cross-model triggers
- Custom GraphQL mutations: register `ActionCloudFunction`s (`func(*RequestContext, *QueryContext) (GraphqlActionResult, error)`) per model/action
- Public (unauthenticated) routes: `config.json`'s `public` array

## Conventions / gotchas

- `YekongaData`, `RequestContext`, and `GraphqlAutoBuild` are protected by `sync.RWMutex` and shared across concurrent requests. When touching server-wide state (routes, models, middleware slices), preserve existing lock discipline and prefer `go test -race` for anything exercising them.
- Trigger dispatch (`triggerCallback` / `triggerAllCallback` / `authTriggerCallback` in `cloud_functions.go`) looks the function up under `y.mut` and **calls it without holding the lock** — triggers run queries that re-enter dispatch, and `findRoute` takes the write lock on every request, so holding the read lock across the call deadlocks under load. Keep it that way.
- `DataModelQuery.Delete()` fires the **AfterDelete** triggers (it used to fire AfterCreate by mistake — fixed 2026-09-23).
- Per-model triggers are keyed by `accessRole_route`; a GraphQL call carrying a key with no registered trigger gets `false` back, i.e. the operation is silently dropped. Prefer the global `Before*All` / `After*All` triggers (what `server/guard.go` uses) for cross-cutting rules.
- Middleware and handlers pass data to each other via `req.SetContext`/`req.GetContext`, not package-level globals.
- Two config files drive everything: `config.json` (server/db/auth/graphql/mail settings — see README "Configuration" for the full field list) and `database.json` (data schema, also drives GraphQL types). Both are required by `ServerConfig`.
- `config.database.kind = "local"` uses a file-based JSON store (in `plugins/database`) — useful for local dev without standing up a real database.

## HelaBox product (server/, web/, pesa_box_app/)

### Run it locally

1. **MongoDB** (no Windows service): `C:\Users\Administrator\mongodb\mongodb-win32-x86_64-windows-8.0.4\bin\mongod.exe --dbpath C:\Users\Administrator\mongodb\data\db --bind_ip 127.0.0.1 --port 27017`. DB name `pesabox`.
2. **API** on :8090 — `cd server && go build -o pesabox-server.exe . && ./pesabox-server.exe` (run from `server/` so it finds `config.json`, `database.json`, `.env`). The port comes from `PORT`, not `config.json`. Settings: `server/.env.example`.
3. **Web** on :5173 — `cd web && npm install && npm run dev` (Vite proxies `/api`, `/graphql`, `/me`, `/logout`, `/refresh` to :8090).
4. **App** — `cd pesa_box_app && flutter run -d emulator-5554` (or VS Code Run). **Debug builds use the local server automatically** (`10.0.2.2:8090` on the Android emulator, `localhost:8090` elsewhere — `lib/services/api_base.dart`); **release builds use the live server** `161.97.99.40:8090`. `--dart-define=API_BASE_URL=...` overrides both (VS Code has a "LIVE server" launch config for that). Switching servers invalidates the saved login ("Domain mismatch") and the app asks you to sign in again. Start the emulator with `-dns-server 8.8.8.8` if it has no internet.
- **Dev login:** without `BEEM_API_KEY`/`BEEM_SECRET_KEY` the server is in dev mode — SMS/OTPs are printed to the console and the OTP is **1234**.
- First Super Admin: `SUPER_ADMIN_PHONES`, else the earliest account is promoted on start when none exists (`migrateAccess` in `server/access.go`).

### Access model (the core — read before touching any route)

- **7 roles** on `User.role` (`server/access.go`): `super_admin`, `staff` (+ preset `viewer`/`support`/`operations` in the `AccessProfile` model), `partner_user`, `cluster_manager`, `group_admin`, `group_officer`, `group_member`. Old names are migrated on start (`support_admin` → staff/operations/all, `admin` → super_admin).
- **Structure:** Partner → Cluster → Group (`Groups.clusterId`). **Assignments** (`userId`, `scopeType` all/partner/cluster/group, `scopeId`, optional group `position`) decide what non-super-admins can see. A group `position` (mwenyekiti/katibu/mweka_hazina/committee) makes that group the user's **home group**; otherwise the home group is the one they created or whose `adminPhone` matches their login phone.
- Every request resolves an **`Actor`** (`actorFromRequest` / `actorFromContext`, cached 10 s — call `invalidateActors()` after changing roles, assignments, clusters or a group's cluster). Checks: `a.Can(perm)` (platform-wide), `a.CanIn(perm, groupId)` (scope + home-group perms), `a.Sees(groupId)`. Permissions are the `Perm*` constants.
- **REST routes:** use `requireActor` / `requirePerm`; group routes under `/api/main/*` use `requireGroup` / `requireGroupPerm` (the caller's home group). Every write calls `writeAudit(...)`.
- **GraphQL guard** (`server/guard.go`): the auto-generated API is open to any logged-in user by config, so global triggers scope every read of group data to what the caller may see, hide `AuditLog`/`Assignment`/`AccessProfile`/`MemberVerification`/settings from non-super-admins, **block every GraphQL delete of business data**, and only allow GraphQL writes to `Group` (web create/edit form) and `Meeting` (close meeting). Anything else needs a checked REST route.
- Internal server code uses `app.ModelQuery(...).SkipBeforeCommit()` (no request context) and is not guarded.

### Data rules (client-approved, TODO.md D5)

- **Money is never deleted or edited.** Fix mistakes with `POST /api/main/transactions/:id/reverse` (marks `reversed`, puts back group totals / loan / fine balances). Every balance and report must skip `reversed` rows.
- People and structure are **deactivated** (`status`), not deleted. A group/member may only be hard-deleted while it has no financial records (`DELETE /api/admin/groups/:id`, `DELETE /api/main/members/:id`).
- **Closed meetings are locked** (`status == completed`): no attendance/money against them.
- Audit log and SMS log are append-only.
- Government loans (`GovernmentLoans` + `GovernmentLoanRepayments`) are loans **to the group** — kept out of member savings totals.

### Reports & dashboard

One KPI computation (`server/report_kpis.go`: `computeGroupKPIs` + `rollup`) feeds `/api/admin/dashboard`, `/api/admin/rollup?level=partner|cluster|group` and the `summary-*` report datasets. `reportScope` (`server/reports.go`) turns `partnerId`/`clusterId`/`groupId` filters into a group set, always clipped to the caller's scope. A new report = one entry in `reportDatasets` (`server/reports.go`: label, column types, `Totals`/`pointInTime` flags) + one builder in `server/report_build.go` (uses the per-request `reportCtx` lookups) + Swahili labels in `columnSw`/`valueSw`. Exports live in `server/report_export.go`. Rules every report follows: dates parsed and printed in **EAT** (`to` inclusive), balances computed from non-reversed transactions and non-cancelled loans (never the stored `Group.total*` — `/api/admin/totals-drift` shows drift), SMS datasets need `PermSms` per group, bad dates → 400, out-of-scope filters from group roles → 403. Tests: `server/reports_test.go` (in-memory fixtures).

### Branding

The user-facing name is **HelaBox**: `server/brand.go` (`brandName`, env `BRAND_NAME`), web i18n key `app.name` (other messages reference it as `@:app.name`, or `@:{'app.name'}` when punctuation follows), app `lib/brand.dart` (`kBrandName`). SMS templates start with `{BRAND}:`. **Do not rename** `config.json` `appName` ("PesaBox" — it names the server's data directory), the `pesabox` DB, `com.example.pesa_box_app` or the `PesaBoxApp` class. SMS sender ID is `TUKIIO` (Beem), overridable with `BEEM_SENDER_ID`.

### Web (`web/`)

- `web/src/api/access.js` loads `/api/access/me`; the router (`meta.perm`) and sidebar (`AppLayout.vue`) show only what the role allows; roles without `dashboard.view` land on `/no-access`.
- Admin REST calls live in `web/src/api/admin.js`. Groups still use GraphQL for read/create/edit; delete goes through the REST safe-delete.
- **i18n:** `web/src/i18n/sw.js` and `en.js` must keep **identical keys** (Swahili is the default). Every visible string goes through `t()`.

### App (`pesa_box_app/`)

- `AppState` (`lib/services/app_data.dart`) is the single data/session layer. `checkGroupAssignment()` calls `GET /api/main/group` → group + `groupPosition` + `groupPermissions`; use `AppState.I.can('finance.write')` etc. to show/hide actions. A dead session (refresh tokens are single-use) is cleared locally and the app returns to login.
- Writes go through REST `/api/main/*` (`_restPost` / `_restGet` / `_restDelete`, Bearer token); a few reads still use GraphQL.
- **i18n:** `tr('English text', [args])`; Swahili lives in `lib/i18n/sw.dart` (a missing key shows English). Add a Swahili entry for every new string.
- Fonts are bundled (`google_fonts/`, runtime fetching off) — add the TTF if you use a new weight.

### Gotchas found the hard way

- **Sessions are bound to the device and domain** that logged in: a token minted by curl/Node is rejected when replayed from a browser (redirect to `/logout`), and tokens issued via `localhost` don't work via `10.0.2.2`. For browser tests, log in *inside* the browser.
- The auth middleware answers an **expired** token with `401 Token expired` only for JSON requests (Content-Type/Accept json); otherwise it 307-redirects to an `https://…/refresh` URL. An *invalid* token gets a 307 even for JSON.
- After `git pull`, run `npm install` in `web/` and `flutter pub get` in the app. Merges that prefer one side (`-X theirs`) have silently dropped `import` lines before — build and smoke-test both apps after a merge.
- The web's GraphQL `createGroup` sends `createdBy` = the group admin's user id (links the group to its Mwenyekiti); the guard keeps it when it's a real user.
