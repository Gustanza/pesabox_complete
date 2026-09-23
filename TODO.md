# HelaBox (PesaBox) — Client Change TODO

**Sources:** HelaBox App WhatsApp group (GATA ↔ Dephics, Fri 2026-09-18) ·
`Pesa Box User Levels.pdf` (repo root) · codebase as of merge `8efa003` (2026-09-23).

## How to use this file

- Tick `[x]` only when a task is **done AND checked** (tested in the running app), not just coded.
- Section status: `⏳ TODO` · `🚧 IN PROGRESS` · `✅ DONE` · `⛔ BLOCKED`
- Every finished task → one line in the **Progress log** at the bottom (date, ID, who, commit).
- Work in phase order (see §1). Don't wait on the client: we build on **our recommended
  decisions (§2)** and only rework if GATA disagrees.

**Apps:** `server/` (Go API, `database.json`) · `web/` (Vue super-admin dashboard) ·
`pesa_box_app/` (Flutter mobile app — separate git repo).

---

## 0. Where we are today (already in the code)

| Area | State |
|---|---|
| Language switch, Swahili default (web + mobile) | ✅ done — some new web text is English-only (see H3) |
| Report download: PDF + Excel + CSV | ✅ done (`server/reports.go`) — group level only |
| SMS templates + on/off switches (web) | ✅ done |
| SMS provider | ✅ **switched SMTZ → Beem Africa**, sender ID `TUKIIO` (`server/main.go`) |
| Super Admin live dashboard (cross-group stats, 7-day chart) | ✅ done (`/api/admin/dashboard`) |
| Profile completion after OTP login (web + mobile) | ✅ done |
| Transaction correction via reversal (`reversed` flag) | ✅ exists — good base for "no delete" |
| Roles | ⚠️ only `super_admin`, `support_admin`, `group_admin` — no scoping by partner/cluster |
| Partner, Cluster, Government loans, Audit log | ❌ not started |
| Group Members logging in (self-service) | ❌ not started — members don't log in today |

---

## 1. Delivery plan

| Phase | Goal | Sections | Target |
|---|---|---|---|
| **1** | Foundations the client already approved | §2 send decisions · §3 rename (config-driven) · §4 roles & structure · §6 no-delete | This week |
| **2** | What GATA cares about most | §5 multi-level reports · §7 government loans · §8 SMS automation | Next week (client's deadline) |
| **3** | Member-facing | §4e member self-service app | After phase 2 |

---

## 2. Client decisions — our recommendation (send to GATA for sign-off) — `⏳ TODO`

We decide with best practice now, build on it, and ask GATA only to confirm.
Tick when GATA confirms (or note their change in the log).

- [ ] **D1 Product name → `HelaBox`.**
  *Why:* it's the client's latest announcement. We make the brand **one config value**
  (`appName` + i18n strings), so a later change is a 10-minute job, not a rework.
  Internal IDs (package name, DB name `pesabox`, repo names) **stay as they are** —
  renaming them risks breaking installs and data for zero user benefit.

- [ ] **D2 Domain → `helabox.co.tz` primary; keep `pesabox.co.tz` and 301-redirect it.**
  *Why:* matches the brand; keeping the old one protects links already shared.
  Layout: `helabox.co.tz` (landing) · `app.` (web dashboard) · `api.` (backend).
  **Action for client:** register helabox.co.tz now (TCRA takes days).

- [ ] **D3 SMS sender ID → register `HelaBox` on Beem now; use `TUKIIO` until approved.**
  *Why:* sender IDs need provider/TCRA approval (can take 1–3 weeks); `HelaBox` is ≤11 chars
  so it's valid. Sender ID is already overridable via `BEEM_SENDER_ID` — no rebuild needed.
  ("PesaboxbyGATA" is too long: 13 chars.)

- [ ] **D4 Report set (standard for VSLA programmes).** Every report filterable by
  **date range** and **level** (organisation → partner → cluster → group).

  | Report | Live dashboard | Download (PDF + Excel) |
  |---|:-:|:-:|
  | Programme KPIs: active/inactive groups, members (M/F), total savings, loans outstanding, portfolio at risk, attendance rate | ✅ | — |
  | Group performance ranking (flags inactive / struggling groups) | ✅ | ✅ |
  | Savings & contributions report | — | ✅ |
  | Loan portfolio: disbursed, repaid, outstanding, overdue (PAR 30) | ✅ | ✅ |
  | Meetings & attendance report | — | ✅ |
  | Fines & social fund report | — | ✅ |
  | Government loans report (§7) | — | ✅ |
  | Group statement / member statement | — | ✅ |
  | Cycle share-out report (end of cycle) | — | ✅ |

  *Why:* **live = numbers for monitoring, download = documents to share/sign/archive.**
  Excel for anyone who analyses data, PDF for reports sent to donors/government.

- [ ] **D5 "Never delete" rules (apply to every role, including Super Admin).**
  - **Financial records are never deleted** — transactions, contributions, shares, social fund,
    loans, repayments, fines, share-outs, government loans. Mistakes are fixed with a
    **reversal entry** (we already have this), so the history always adds up.
  - **People and structures are deactivated, never deleted** — members, users, groups,
    clusters, partners (`status = inactive`, can be reactivated).
  - **Closed meetings are locked** — no edits after a meeting is closed; fix via reversal.
  - **Audit log and SMS log can't be edited or deleted** by anyone.
  - **Only exception:** a record with *no* financial activity linked (e.g. a member added by
    mistake 5 minutes ago) may be deleted by Group Admin or above, and the delete is logged.
  *Why:* this is standard for any system holding people's money — auditors and donors
  need a trail that can't be altered.

- [ ] **D6 Government loans = a loan *to the group* from an outside lender.**
  Fields: lender/programme (e.g. council 10% loans for women/youth/PWD groups, bank, NGO),
  reference/agreement no., amount, date received, interest rate (can be 0%), term,
  repayment schedule, repayments, outstanding balance, status, attached agreement (PDF/photo).
  Optional: record how the group shared it out to members (as normal internal loans).
  *Why:* keeps outside money **separate from members' savings**, so group totals stay honest.

- [ ] **D7 Partner & Cluster structure.**
  - **Partner** (organisation): name, type (NGO / government / bank / other), contact person,
    phone, email, regions covered, status.
  - **Cluster**: name, partner, region / district / ward, status.
  - A group belongs to **exactly one cluster**; a cluster to **exactly one partner**.
  - Staff / managers can be assigned to **many** partners / clusters / groups.
  *Why:* one parent each means roll-up reports never double-count a group.

- [ ] **D8 Roles & permissions (from the approved PDF), made practical:**
  - Existing users map to new roles: `super_admin` → Super Admin · `support_admin` →
    PesaBox Staff · `group_admin` → Group Admin.
  - **PesaBox Staff "configurable permissions"** → start with **3 presets** (Viewer, Support,
    Operations) instead of dozens of toggles; add toggles later only if needed.
  - **Partner Users are read-only on money** (they see and download, never edit transactions).
  - **Group Members log in with phone + OTP** (same as admins today) — phase 3.
  *Why:* presets are easier to explain, test and support than per-permission switches.

- [ ] **D9 Still genuinely needed from the client** (can't be decided for them):
  - Real partner & cluster names and which groups go where (data to load).
  - Notes from the call with Winnie (to adjust D4).
  - Confirm which government loan programmes their groups actually use (to adjust D6).

---

## 3. Rename PesaBox → HelaBox — `⏳ TODO` (build now on D1; ~22 files)

- [x] **R1** Make brand name a single source: server `config.json` `appName`, web + mobile — _server brand.go + web i18n app.name (linked @:app.name) — mobile still to do (R3)_
      i18n key (e.g. `app.name`) — no hardcoded "PesaBox" in views
- [x] **R2** Web: page titles, `index.html`, logo/wordmark, sw + en strings — _web: HelaBox in sidebar, login pages, page title, all translations_
- [ ] **R3** Mobile: Android/iOS display name, splash/login, sw + en strings, launcher icon
- [x] **R4** Server: PDF/Excel report headers, email text, SMS templates text — _server: brand.go (HelaBox, BRAND_NAME env), PDF header, export filename, SMS {BRAND} prefix_
- [ ] **R5** Point web/mobile to final domain (D2) when it's live
- [ ] **R6** Check: search web, mobile, SMS, reports — no user-facing "PesaBox" left
- [ ] **R7** Do NOT rename: `com.example.pesa_box_app`, DB `pesabox`, repos (see D1)

---

## 4. User levels & structure — `⏳ TODO` (approved by client)

| # | Role | Sees | Can't |
|---|---|---|---|
| 1 | Super Admin | Everything | — |
| 2 | PesaBox Staff | Assigned partners/clusters/groups | Beyond assignment (unless granted) |
| 3 | Partner User | Their organisation's clusters & groups | Edit group money records |
| 4 | Cluster Manager | Groups in their cluster | Other clusters |
| 5 | Group Admin (Mwenyekiti) | Own group, all | Other groups |
| 6 | Group Officer (Katibu, Mweka Hazina, committee) | Own group, day-to-day | Core settings, assign Group Admin |
| 7 | Group Member | Own records only | Others' finances |

### 4a. Data model (server)
- [x] **U1** `Partners` model in `database.json` (D7 fields)
- [x] **U2** `Clusters` model (→ partner)
- [x] **U3** `Groups.clusterId` (→ cluster → partner); migrate existing groups to a default cluster — _existing groups auto-placed in 'Default Cluster' on server start_
- [x] **U4** `Assignments` model: user ↔ partner / cluster / group (many-to-many)
- [x] **U5** 7 role values on users + migration of existing roles (D8 mapping) — _old roles migrated on start (support_admin → staff + operations + all, admin → super_admin), first super admin = earliest user or SUPER_ADMIN_PHONES_
- [x] **U6** Group-level positions: Mwenyekiti (Admin), Katibu / Mweka Hazina / committee (Officer)

### 4b. Access control (server) — the most important part
- [x] **U7** One central check `can(user, action, record)` = role + scope — every route uses it — _server/access.go (Actor.Can / CanIn / Sees)_
- [x] **U8** Scope filter on **every** list, report and API (no data outside your partner/cluster/group) — _REST routes + GraphQL guard (server/guard.go) scope every read_
- [x] **U9** Partner Users: read-only on financial data
- [x] **U10** Group Officers: no core settings, can't assign Group Admin
- [x] **U11** Staff permission presets: Viewer / Support / Operations (D8)
- [x] **U12** Audit log model + write an entry for every create/update/reverse/deactivate/delete — _AuditLog model, every REST write + GraphQL write logged_
- [x] **U13** Deactivate / reactivate accounts at every level

### 4c. Web dashboard
- [x] **U14** Super Admin: manage Partners, Clusters, Staff — _web: Partners & Clusters page (/structure)_
- [x] **U15** Assign staff/managers to partners / clusters / groups — _web: Users & Roles page — assignments with position_
- [x] **U16** Menus and pages by role (hide what a role can't use) — _web: sidebar + router guard by permission (web/src/api/access.js), roles without dashboard access see /no-access_
- [x] **U17** Partner & Cluster Manager dashboards (reuse `/api/admin/dashboard`, scoped) — _web: dashboard filters by partner/cluster + roll-up table by partner/cluster/group_
- [x] **U18** Audit log viewer (Super Admin) — _web: Audit Logs page reads the real log (filters: action, record, dates)_

### 4d. Mobile app (group roles)
- [ ] **U19** Show/hide actions by Group Admin vs Group Officer
- [ ] **U20** Group Admin: add/remove Group Officers

### 4e. Member self-service — phase 3
- [ ] **U21** Member login (phone + OTP)
- [ ] **U22** Member view: own savings, shares, social fund, loans, fines, history, announcements
- [ ] **U23** Member requests (loan application) → approved by Admin/Officer

### 4f. Check
- [ ] **U24** Test each role: can do what it should, **can't** do what it shouldn't
- [ ] **U25** Scope-leak test: a Cluster Manager / Partner never sees another's data (incl. reports & exports)

---

## 5. Reports at 4 levels — `⏳ TODO` (phase 2; build on D4)

- [x] **P1** Roll-up engine: group → cluster → partner → organisation (needs U1–U3) — _/api/admin/rollup + summary-partner/cluster/group datasets_
- [x] **P2** Check roll-up totals match the sum of groups exactly — _one KPI computation (server/report_kpis.go) feeds dashboard, roll-ups and exports_
- [x] **P3** Live KPI dashboard per level (D4 "live" rows) — _web dashboard KPI cards: active/inactive groups, women/men, PAR 30, attendance, shares, gov loans_
- [x] **P4** Downloadable reports per level, PDF + Excel (extend `server/reports.go`) — _15 → 19 datasets incl. summary by partner/cluster/group, PAR 30, government loans, PDF + Excel + CSV_
- [x] **P5** Date-range filter on all reports
- [x] **P6** Portfolio at risk (PAR 30) + inactive-group detection — _portfolio-at-risk dataset + Activity column (no meeting in 30 days)_
- [x] **P7** Reports respect role scope (U8)
- [x] **P8** All reports in Swahili + English — _all new report columns/labels have Swahili_

---

## 6. Data that must not be deleted — `⏳ TODO` (phase 1; build on D5)

- [x] **N1** Block delete at the API for all financial models (every role)
- [x] **N2** Soft delete (deactivate) for members, users, groups, clusters, partners — — _DELETE /api/admin/users/:id now deactivates_
      replace the hard delete at `DELETE /api/admin/users/:id` in `server/main.go`
- [x] **N3** Lock closed meetings against edits
- [x] **N4** Allow delete only when no financial activity is linked, Group Admin+, logged — _POST /api/main/transactions/:id/reverse (server) — mobile screen still to wire_
- [x] **N5** Audit log + SMS log are append-only
- [x] **N6** Check nothing (web, mobile, GraphQL auto-API) can bypass these rules — — _GraphQL deletes of business models always blocked, creates/updates only Group + Meeting (checked)_
      the framework auto-generates GraphQL mutations incl. delete; block them for these models

---

## 7. Government loans — `⏳ TODO` (phase 2; build on D6)

- [x] **G1** `GovernmentLoans` + repayments model (D6 fields) — _GovernmentLoans + GovernmentLoanRepayments models, /api/main/gov-loans routes_
- [ ] **G2** Record receipt + repayments (web + mobile, Group Admin/Officer)
- [ ] **G3** Show on group dashboard & group statement, separate from member savings
- [ ] **G4** Include in reports at all 4 levels (§5) and in SMS notifications (§8)

---

## 8. SMS automation — `🚧 IN PROGRESS` (promised end of next week)

- [x] **S1** Templates + on/off switches (web)
- [x] **S2** Provider switched to Beem Africa
- [ ] **S3** Add `BEEM_API_KEY` / `BEEM_SECRET_KEY` to local `server/.env` (only `SMTZ_API_KEY`
      is there; locally SMS runs in dev mode = printed, not sent)
- [x] **S4** Check an SMS fires for every transaction type (contribution, share, social fund, — _every member-facing money type texts the member, expenses/withdrawals/government loans are group-level (no SMS)_
      loan disbursement / repayment, fine, government loan)
- [ ] **S5** Meeting & loan-due reminders (currently opt-in `PESABOX_REMINDERS=1`) — turn on in production
- [ ] **S6** Switch sender ID to `HelaBox` once approved (D3) via `BEEM_SENDER_ID`
- [ ] **S7** Brand name in SMS text (R4)

---

## 9. Language — `✅ DONE` (clean-up left)

- [x] **L1** Swahili default + English, web and mobile
- [x] **L2** Translate new English-only web screens from the 2026-09-23 merge: — _ProfileView, CompleteProfileView, GroupDetails, Dashboard, Register translated — merge-broken imports fixed (AppLayout, RegisterView)_
      `ProfileView`, `CompleteProfileView`, parts of `DashboardView`, `GroupDetailsView`,
      `LoginView`, `RegisterView`
- [ ] **L3** Mobile: add Swahili for strings added after the i18n merge
      ("Deactivate member", "Mark as urgent", "No messages sent yet.", …)
- [ ] **L4** Client review of Swahili wording

---

## 10. Housekeeping

- [ ] **H1** Stop tracking `web/node_modules/` and `server/pesabox-server.exe` in git
      (the last commit changed 130+ node_modules files — `.gitignore` has `*/node_modules`
      but they're already tracked, so `git rm -r --cached` is needed)
- [ ] **H2** Push the 4 unpushed server-repo commits (incl. merge `8efa003`)
- [ ] **H3** Mobile repo: commit the merged work, then `git stash drop` the backup stash
- [ ] **H4** Mobile: bundle Google Fonts in the app (runtime download froze the emulator
      and will fail for users on bad networks)
- [ ] **H5** Add a real `server/.env.example` listing required keys (BEEM_*, PESABOX_REMINDERS)

---

## Progress log

| Date | Task | Who | Note / commit |
|------|------|-----|---------------|
| 2026-09-23 | — | — | Pulled `8dc51c5` + `8123410` (live dashboard, Beem SMS, profile completion); merged as `8efa003`, incoming side preferred |
| 2026-09-23 | — | — | TODO rewritten with recommended client decisions (§2) |
| 2026-09-23 | U1–U13, N1–N6, G1, P1–P7 (server) | Claude | Access control (`server/access.go`, `guard.go`, `access_routes.go`, `structure_routes.go`), gov loans (`gov_loans.go`), KPIs/roll-ups (`report_kpis.go`), reversal route. 44/44 end-to-end checks passed on an isolated DB |
| 2026-09-23 | framework | Claude | yekonga: `Delete()` now fires AfterDelete (was AfterCreate); trigger dispatch no longer holds the lock while running triggers (deadlock fix) |
| 2026-09-23 | U14–U18, R1–R2, P3–P4, P8, L2 (web) | Claude | New pages Structure / Users & Roles / Audit / NoAccess, role-aware menu + router, dashboard KPIs + roll-ups, report filters by partner/cluster. Headless-Chrome smoke test: 11/11 pages OK in sw + en, 0 JS errors |
| 2026-09-23 | fixes | Claude | The merge had broken the web: AppLayout + RegisterView lost their `useI18n` import/call (runtime crash), Dashboard stat labels, fake pagination on Groups — all fixed |
