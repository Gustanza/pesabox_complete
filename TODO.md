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

## 0. Where we are today (updated 2026-09-25)

| Area | State |
|---|---|
| Language switch, Swahili default (web + mobile) | ✅ done — all screens translated (L4: client to review wording) |
| Report download: PDF + Excel + CSV | ✅ done — 19 datasets, organisation → partner → cluster → group |
| SMS templates + on/off switches (web) | ✅ done |
| SMS provider | ✅ **switched SMTZ → Beem Africa**, sender ID `TUKIIO` (`server/main.go`) |
| Live dashboard | ✅ scoped to the role, KPIs (PAR 30, attendance, active groups, gov loans), roll-ups by partner/cluster/group |
| Profile completion after OTP login (web + mobile) | ✅ done |
| Corrections | ✅ reversal route + app screen; financial records can't be deleted (GraphQL guard) |
| Roles | ✅ 7 levels + staff presets + assignments (`server/access.go`), every REST + GraphQL call scoped |
| Partner, Cluster, Government loans, Audit log | ✅ done (web + app) |
| Group Members logging in (self-service) | ⏳ phase 3 (U21–U23) |

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

- [x] **D10 Loan interest.** _Decided 2026-09-26 (user): flat interest on principal, added at issue; per-group rate; new loans only._ Loans store `interestRate` (10%) but repayments and balances use principal only.
  *Our recommendation:* flat interest on principal, added to the amount due at disbursement
  (total due = principal × (1 + rate)); PAR and balances use total due. Reports already show "Interest %".
  **Needs GATA to confirm before we change balances.**

- [x] **D11 Money recorded outside a meeting.** _Decided 2026-09-25 (user): yes — it counts toward that day's meeting (current behaviour kept)._ The "collections per meeting" report credits a payment
  with no meeting to that day's meeting. *Our recommendation:* only money-in recorded in the app during the
  meeting counts; mobile-money pay-ins outside it show as "between meetings". Needs GATA to confirm.

- [ ] **D12 Loan limit (our default, GATA to confirm).** A member may borrow up to (savings + shares − withdrawals) × the group's multiplier, **minus what they still owe**; only Active members can borrow. Multiplier 0 = no limit (the default).

- [ ] **D9 Still genuinely needed from the client** (can't be decided for them):
  - Real partner & cluster names and which groups go where (data to load).
  - Notes from the call with Winnie (to adjust D4).
  - Confirm which government loan programmes their groups actually use (to adjust D6).

---

## 3. Rename PesaBox → HelaBox — `✅ DONE` (R5 waits for the domain)

- [x] **R1** Make brand name a single source: server `config.json` `appName`, web + mobile — _server brand.go + web i18n app.name (linked @:app.name) — mobile still to do (R3)_
      i18n key (e.g. `app.name`) — no hardcoded "PesaBox" in views
- [x] **R2** Web: page titles, `index.html`, logo/wordmark, sw + en strings — _web: HelaBox in sidebar, login pages, page title, all translations_
- [x] **R3** Mobile: Android/iOS display name, splash/login, sw + en strings, launcher icon — _app/brand.dart (kBrandName), Android label + iOS display name HelaBox, H logo mark — launcher icon image still the default (needs a HelaBox logo file)_
- [x] **R4** Server: PDF/Excel report headers, email text, SMS templates text — _server: brand.go (HelaBox, BRAND_NAME env), PDF header, export filename, SMS {BRAND} prefix_
- [ ] **R5** Point web/mobile to final domain (D2) when it's live
- [x] **R6** Check: search web, mobile, SMS, reports — no user-facing "PesaBox" left — _swept 2026-09-25: remaining hits are internal ids only (PesaBoxApp class, PESABOX_REMINDERS, DB name). Web + mobile Settings now show the real sender ID from the server_
- [x] **R7** Do NOT rename: `com.example.pesa_box_app`, DB `pesabox`, repos (see D1) — _kept: package com.example.pesa_box_app, DB pesabox, config appName PesaBox (it names the data dir)_

---

## 4. User levels & structure — `✅ DONE` (4e member app = phase 3)

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
- [x] **U19** Show/hide actions by Group Admin vs Group Officer — _app reads position + permissions from GET /api/main/group, officer-only screens hide admin actions (officers, close cycle, reversal gated by finance.write)_
- [x] **U20** Group Admin: add/remove Group Officers — _app: Group → Group officers (add by phone + position, remove) — POST/DELETE /api/main/officers_

### 4e. Member self-service — phase 3
- [ ] **U21** Member login (phone + OTP)
- [ ] **U22** Member view: own savings, shares, social fund, loans, fines, history, announcements
- [ ] **U23** Member requests (loan application) → approved by Admin/Officer

### 4f. Check
- [x] **U24** Test each role: can do what it should, **can't** do what it shouldn't — _server e2e: super admin, staff viewer, partner user, katibu — 44/44_
- [x] **U25** Scope-leak test: a Cluster Manager / Partner never sees another's data (incl. reports & exports) — _server e2e: viewer on another cluster sees 0 groups / 0 transactions over REST and GraphQL_

---

## 4g. Group rules (constitution) — `✅ DONE` (2026-09-26)

- [x] **K1** Edit rules: web Financial tab (Super Admin / Operations) + app Rules & Constitution (Mwenyekiti) — `/api/admin/groups/:id/rules`, `/api/main/group/rules`, audit-logged, validated, locked out of GraphQL
- [x] **K2** Rules take effect: flat interest (`Loan.interestAmount` / `totalDue`), repayment up to total due, loan limit (D12), social fund amount, services on/off, fine reasons only (old fine fields unused), voluntary vs mandatory savings (`Transaction.savingsType`)
- [x] **K3** Balances everywhere use total due (reports, PAR, dashboard, app, SMS, stored `totalLoans`, `/totals-drift`)
- [x] **K4** Security fixes found on the way: GraphQL could overwrite Group totals, a group admin could move a meeting to another group, `/transactions` accepted any type — all closed
- [ ] **K5 Rollout:** release the new app **before or with** the new server on the live machine — the installed app caps repayments at the principal, so interest could never be repaid from it

## 5. Reports at 4 levels — `✅ DONE`

- [x] **P1** Roll-up engine: group → cluster → partner → organisation (needs U1–U3) — _/api/admin/rollup + summary-partner/cluster/group datasets_
- [x] **P2** Check roll-up totals match the sum of groups exactly — _one KPI computation (server/report_kpis.go) feeds dashboard, roll-ups and exports_
- [x] **P3** Live KPI dashboard per level (D4 "live" rows) — _web dashboard KPI cards: active/inactive groups, women/men, PAR 30, attendance, shares, gov loans_
- [x] **P4** Downloadable reports per level, PDF + Excel (extend `server/reports.go`) — _15 → 19 datasets incl. summary by partner/cluster/group, PAR 30, government loans, PDF + Excel + CSV_
- [x] **P5** Date-range filter on all reports
- [x] **P6** Portfolio at risk (PAR 30) + inactive-group detection — _portfolio-at-risk dataset + Activity column (no meeting in 30 days)_
- [x] **P7** Reports respect role scope (U8)
- [x] **P8** All reports in Swahili + English — _all new report columns/labels have Swahili_
- [x] **P9** Reports hardening (review 2026-09-25, R1–R27 + V1–V8) — _date range on every dataset, EAT time zone, balances computed from non-reversed transactions (not stored totals), cancelled loans = 0, SMS reports need SMS permission, 400 on bad dates, 403 for out-of-scope filters, TOTAL rows, real Excel dates/number formats, PDF columns no longer cut, new datasets meeting-collections / fines-outstanding / expenses / gov-loan-repayments, dashboard drill-down, app statement + reports aligned with server. `GET /api/admin/totals-drift` (super admin) shows stored-vs-real total differences_
- [ ] **P10** DB indexes on `groupId` + `createdAt` (Transactions, SmsLogs, MeetingAttendance) — before large data

---

## 6. Data that must not be deleted — `✅ DONE`

- [x] **N1** Block delete at the API for all financial models (every role)
- [x] **N2** Soft delete (deactivate) for members, users, groups, clusters, partners — — _DELETE /api/admin/users/:id now deactivates_
      replace the hard delete at `DELETE /api/admin/users/:id` in `server/main.go`
- [x] **N3** Lock closed meetings against edits
- [x] **N4** Allow delete only when no financial activity is linked, Group Admin+, logged — _POST /api/main/transactions/:id/reverse (server) — mobile screen still to wire_
- [x] **N5** Audit log + SMS log are append-only
- [x] **N6** Check nothing (web, mobile, GraphQL auto-API) can bypass these rules — — _GraphQL deletes of business models always blocked, creates/updates only Group + Meeting (checked)_
      the framework auto-generates GraphQL mutations incl. delete; block them for these models

---

## 7. Government loans — `✅ DONE`

- [x] **G1** `GovernmentLoans` + repayments model (D6 fields) — _GovernmentLoans + GovernmentLoanRepayments models, /api/main/gov-loans routes_
- [x] **G2** Record receipt + repayments (web + mobile, Group Admin/Officer) — _app: Group → Government loans (record + repayments), web shows them read-only on the group page_
- [x] **G3** Show on group dashboard & group statement, separate from member savings — _web group page + app screen + app Group Statement (screen + PDF)_
- [x] **G4** Include in reports at all 4 levels (§5) and in SMS notifications (§8) — _all 4 report levels + government-loans dataset. No member SMS by design (the loan is to the group)_

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
- [x] **S7** Brand name in SMS text (R4) — _SMS templates use {BRAND} → HELABOX, saved templates migrated on start_

---

## 9. Language — `✅ DONE` (L4 = client review)

- [x] **L1** Swahili default + English, web and mobile
- [x] **L2** Translate new English-only web screens from the 2026-09-23 merge: — _ProfileView, CompleteProfileView, GroupDetails, Dashboard, Register translated — merge-broken imports fixed (AppLayout, RegisterView)_
      `ProfileView`, `CompleteProfileView`, parts of `DashboardView`, `GroupDetailsView`,
      `LoginView`, `RegisterView`
- [x] **L3** Mobile: add Swahili for strings added after the i18n merge — _79 strings added to lib/i18n/sw.dart — only demo text on unused onboarding screens left_
      ("Deactivate member", "Mark as urgent", "No messages sent yet.", …)
- [ ] **L4** Client review of Swahili wording

---

## 10. Housekeeping — `🚧` (H3 stash drop is yours)

- [x] **H1** Stop tracking `web/node_modules/` and `server/pesabox-server.exe` in git — _git rm --cached web/node_modules, both server .exe files, server/server.log + .gitignore rules — commit to apply_
      (the last commit changed 130+ node_modules files — `.gitignore` has `*/node_modules`
      but they're already tracked, so `git rm -r --cached` is needed)
- [x] **H2** Push the 4 unpushed server-repo commits (incl. merge `8efa003`) — _pushed by you (569b4c8)_
- [ ] **H3** Mobile repo: commit the merged work, then `git stash drop` the backup stash — _committed by you (a6eb6db) — the backup stash is still there, drop it when happy_
- [x] **H4** Mobile: bundle Google Fonts in the app (runtime download froze the emulator — _pesa_box_app/google_fonts/*.ttf + GoogleFonts.config.allowRuntimeFetching=false_
      and will fail for users on bad networks)
- [x] **H5** Add a real `server/.env.example` listing required keys (BEEM_*, PESABOX_REMINDERS) — _server/.env.example_

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
| 2026-09-25 | R3, R6, U19–U20, G2, L3, H4 (mobile) | Claude | HelaBox brand, role-aware app (/api/main/group), officers + government loans screens, real reversal screen, 79 Swahili strings, bundled fonts, expired session → login instead of 'waiting for group'. Checked on the emulator + flutter test 29/29 |
| 2026-09-25 | settings, H1, H5 | Claude | Web Settings showed fake data (NextSMS, PESABOX, fake admins, dead 2FA toggles) — now real provider/sender/health/admins. Untracked node_modules + exe + log, added server/.env.example |
| 2026-09-25 | P9, G3 (reports review) | Claude (reviewer + coder agents) | Reviewer found 27 report issues (3 high: date range ignored by 11 datasets, UTC dates, app counted cancelled loans) — all fixed; re-review found 8 small ones — fixed. go test 28/28, isolated e2e 131/131, web build + i18n parity 633/633, flutter test 35/35. D10 (interest) + D11 (meeting rule) added for GATA |
| 2026-09-26 | K1–K4, D10, D11 | Claude (coder + reviewer agents) | Group rules editable (web + app) and enforced; flat interest; loan limit; voluntary savings; 2 old bugs + 1 security hole fixed. Review round W1–W10 fixed. go test 44/44, e2e 204/204, web build + i18n 666/666, flutter test 43/43. TESTING.md: H8–H14, M1–M10 |
| 2026-09-26 | login gate | Claude | Login is invite-only: an unknown number gets no OTP (no SMS, no account) — only existing active accounts, a group's admin phone, SUPER_ADMIN_PHONES, or first setup. Deactivated accounts refused. Framework Google sign-in refused when no client ID is set (it accepted any Google token). Web + app show the reason in sw/en. go test + flutter test 43/43 + web build OK; checked live on the isolated :8091 server. TESTING.md A9–A11 |
