# HelaBox — Manual Test Cases

## Test logins — one per user level (local database)

Every login uses **OTP code `1234`** (dev mode). Web: `http://localhost:5173` — use a **new incognito window per
person**. App: log out first (Profile → Log out), then log in with the phone below.

Structure these accounts live in:

```
GATA (partner)
├── Arusha North (cluster) ── Mkusanyiko Group   (your group — Mwenyekiti: you, Katibu: Kassim)
└── Moshi (cluster) ──────── Upendo Vikoba       (Mwenyekiti: Upendo)
CARE Tanzania (partner)
└── Dodoma Central (cluster) ─ Tumaini Group      (Mwenyekiti: Tumaini — has a TZS 50,000 loan)
```

| Level | Who | Phone | Log in on | Should SEE | Should NOT see / do |
|---|---|---|---|---|---|
| 1 Super Admin | You | 0625689904 | 🌐 Web + 📱 App | 🌐 all 3 groups, both partners, every menu (Users, Audit Logs, Settings…) · 📱 Mkusanyiko as Mwenyekiti | — (can do everything) |
| 2 Staff — Operations | Olivia Operations | 0711000013 | 🌐 Web | all 3 groups; can create groups, edit group settings | Users & Roles, Audit Logs, Settings menus |
| 2 Staff — Support | Samuel Support | 0711000012 | 🌐 Web | all 3 groups; can create groups | Users, Audit, Settings; **no** Edit / Close / Delete on groups |
| 2 Staff — Viewer | Stella Viewer | 0711000011 | 🌐 Web | **2 groups**: Mkusanyiko + Upendo (GATA only) — read-only | Tumaini Group, CARE Tanzania; no Create Group, no edit buttons |
| 3 Partner User | Paul Partner | 0711000002 | 🌐 Web | **2 groups** (GATA) — dashboard, reports, downloads | Tumaini / CARE; Users, SMS, Audit, Settings; any edit button |
| 4 Cluster Manager | Clara Cluster | 0711000004 | 🌐 Web | **1 group**: Mkusanyiko (Arusha North); **Create Group** (only into Arusha North) | Upendo, Tumaini; Users, Audit, Settings; edit/close/delete group |
| 5 Group Admin (Mwenyekiti) | Upendo Mwenyekiti | 0711000005 | 📱 App | Upendo Vikoba — "Chairperson"; Group officers **with Add**; record money; reversals; close cycle | 🌐 Web shows "You don't have access" (group roles use the app); no other group's data |
| 5 Group Admin (Mwenyekiti) | Tumaini Mwenyekiti | 0711000007 | 📱 App | Tumaini Group, its loan of TZS 50,000 | Mkusanyiko / Upendo data |
| 6 Group Officer (Katibu) | Kassim Katibu | 0711000003 | 📱 App | Mkusanyiko — "Secretary"; record money, attendance, reversals | **no** Add officer / remove; **no** "Close cycle"; 🌐 Web → "You don't have access" |
| 7 Group Member | Mary Member | 0711000006 | 📱 App / 🌐 Web | nothing yet — 📱 "Waiting for your group", 🌐 "You don't have access" | everything (member self-service is phase 3) |

Two more accounts already existed before these were created — leave them as they are: 0624023920 (Super Admin)
and 0650980535 (Partner User, assigned to "Default Partner", which now has no groups).

---

## What the client asked for → what we built → which cases prove it

From the HelaBox App WhatsApp group (GATA, 18 Sep 2026) and `Pesa Box User Levels.pdf`.
✅ built · 🟡 partly built · ⏳ not built yet · 👤 needs you / the client (not code)
Where: 🌐 Web dashboard (`http://localhost:5173`) · 📱 App on the emulator · 🖥️ API console (the window running `pesabox-server.exe`)

| # | Client request | What we built | Status | Where to test | Test cases |
|---|---|---|---|---|---|
| 1 | **Rename PesaBox → HelaBox**, "update this everywhere" | Name on web, app, reports, PDF/Excel headers, SMS prefix (`HELABOX:`), phone app name, H logo | ✅ — launcher icon image and domain 👤 | 🌐 Web (sidebar, tab title, report files) · 📱 App (welcome, login, phone app list) · 🖥️ API console (SMS text) | A1, A3, A4, A6, H6, F4 |
| 2 | **User levels** (7): Super Admin → Staff → Partner User → Cluster Manager → Group Admin → Group Officer → Group Member | Roles 1–6 with the permissions from the PDF; staff presets Viewer/Support/Operations; every screen and API call limited to what the role may see | ✅ levels 1–6 · ⏳ level 7 (members log in) = phase 3 | 🌐 Web → Users & Roles, then log in as each person in its own incognito window | C1–C8, D1–D7 |
| 3 | **Clusters and Partners** — "a collection of VSLAs", staff assigned to them "same as groups" | Partner → Cluster → Group structure, assign people to partners / clusters / groups, move groups between clusters | ✅ | 🌐 Web → Partners & Clusters, Users & Roles, group page | B1–B6, C4–C5, D4–D5 |
| 4 | Group Admin (Mwenyekiti) manages **Group Officers** (Katibu, Mweka Hazina…) | Officers screen in the app; officers run day-to-day work but can't change settings or officers | ✅ | 📱 App → Group tab → Group officers (as Mwenyekiti, then as Katibu) | F1–F3, G1–G6 |
| 5 | **Reports** at organisation, partner, cluster and group level, to confirm every tracking point is captured | Every report filters by partner / cluster / group; roll-ups per level; summaries, PAR 30, meetings, attendance, savings… (19 report types) | ✅ — final report list from GATA 👤 | 🌐 Web → Reports, Dashboard | H1–H5, H7 |
| 6 | **Download reports as PDF and Excel**; some reports viewed live on the dashboard | PDF + Excel + CSV export; live dashboard with KPIs and roll-up table | ✅ | 🌐 Web → Dashboard, Reports → Export | H1–H3, H6 |
| 7 | Focus on **group meetings and collections** | Meetings, attendance, savings/contributions reports; attendance rate and inactive groups (no meeting in 30 days) on the dashboard | ✅ | 🌐 Web → Dashboard, Reports | H1, H4 |
| 8 | **Groups that get government loans** | Government loans to the group (lender, programme, repayments, balance, overdue), kept separate from members' savings; in reports and dashboard | ✅ — not yet on the app's Group Statement PDF | 📱 App → Group → Government loans · 🌐 Web → group page, Dashboard, Reports | F9–F12, H1, H5 |
| 9 | **Data that must NOT be deleted** (all levels) | Money is never deleted — mistakes are reversed with a reason; people/groups are deactivated; closed meetings locked; delete only when nothing financial is attached | ✅ (our proposal — GATA to confirm 👤) | 🌐 Web → Groups, Users & Roles · 📱 App → Transactions, Meetings | E2–E3, C7, F5–F8, L1 |
| 10 | **Language switch**, Swahili default | Web + app fully in Swahili/English, including Raymond's new app design | ✅ — wording review by GATA 👤 | 🌐 Web → SW/EN in the top bar · 📱 App → Profile → Settings → Language | A2, A8 |
| 11 | **SMS automation** on transactions + other notifications; sender keyword | SMS on every member money action, templates editable, on/off switches; reminders ready (off by default) | ✅ — sender `TUKIIO` until `HelaBox` is registered 👤, reminders switch-on 👤 | 📱 App (record money) · 🖥️ API console (SMS text) · 🌐 Web → Settings, SMS | F4, J1–J3 |
| 12 | Super Admin can **see audit logs** (from the PDF) | Every change is logged (who, what, when); Audit Logs page | ✅ | 🌐 Web → Audit Logs | I1–I3 |

---

Work top to bottom — later sections reuse the data created earlier.
Tick `[x]` when a case passes. If it fails, leave it unticked and add a row to
the **Results log** at the bottom (case ID + what happened).

## 0. Before you start

- Running: MongoDB, API (`server/`, port 8090), web (`http://localhost:5173`), app on the emulator
  (`flutter run -d emulator-5554` or VS Code Run — debug runs use your local server automatically). See `CLAUDE.md` → "Run it locally".
- **Dev mode:** with no Beem keys in `server/.env`, every OTP is **`1234`** and SMS texts are printed in the API console
  instead of sent. Keep the API console visible — several cases check it.
- **One browser window per person.** A login belongs to the browser it was made in. For each new person, use a new
  **incognito/private window** (or another browser), and log out of incognito windows when you're done.

Test people used below (all created during the tests):

| Who | Phone | Role |
|---|---|---|
| You | 0625689904 | Super Admin (+ Mwenyekiti of Mkusanyiko Group) |
| Vera | 0711000001 | HelaBox Staff — Viewer |
| Paul | 0711000002 | Partner User |
| Katibu Test | 0711000003 | Group Officer (Katibu) |
| Clara | 0711000004 | Cluster Manager |

---

## A. Login, branding, language — 🌐 Web + 📱 App

- [ ] **A1 Web login** — Open `http://localhost:5173` → enter 0625689904 → code 1234.
  **Where:** 🌐 Web → `http://localhost:5173` → login page
  *Expect:* the dashboard opens and the sidebar says **HelaBox Admin** with your role under it.
- [ ] **A2 Language switch (web)** — Click **SW** / **EN** in the top bar.
  **Where:** 🌐 Web → top bar → **SW / EN** buttons (any page)
  *Expect:* every menu item, heading and button changes language; no text looks like `kpi.members` or `struct.title`.
- [ ] **A3 Browser tab title** — *Expect:* **HelaBox — Admin**.
  **Where:** 🌐 Web → the browser tab at the top of the window
- [ ] **A4 App welcome screen** — Log out in the app (Profile → Log out) or start it fresh.
  **Where:** 📱 App → Profile tab → **Log out** → welcome screen → **Get started**
  *Expect:* the **H** logo, the name **HelaBox**, and "Karibu HelaBox" / "Welcome to HelaBox" on the login screen.
- [ ] **A5 App login** — Log in with 0625689904 / 1234.
  **Where:** 📱 App → welcome screen → Get started → phone → code → Home tab
  *Expect:* the Home tab shows **Mkusanyiko Group** and its balances (not "Waiting for your group").
- [ ] **A6 App name on the phone** — Leave the app and open the phone's app list.
  **Where:** 📱 App → leave the app (Home button) → phone app list
  *Expect:* the app is called **HelaBox**.
- [ ] **A7 Offline fonts** — Turn off the emulator's Wi-Fi/data and restart the app.
  **Where:** 📱 App → emulator Settings (turn off Wi-Fi/data) → reopen the app
  *Expect:* it opens normally with proper fonts and doesn't freeze.

- [ ] **A8 App language** — In the app: Profile → Settings → **Kiswahili**.
  **Where:** 📱 App → Profile tab → **Settings** → Language → **Kiswahili**
  *Expect:* every screen switches — bottom bar (Mwanzo, Kikundi, Mikutano…), cards (Akiba, Hisa), pills ("Unakuja"),
  the Group menu ("Mikopo ya serikali", "Viongozi wa kikundi") and your position ("Mwenyekiti"). Switch back afterwards.

## B. Partners & Clusters — 🌐 Web (as Super Admin)

- [ ] **B1 Create partner** — Partners & Clusters → **Add partner** → name "GATA", type NGO, contact person "Winnie" → Save.
  **Where:** 🌐 Web → sidebar **Partners & Clusters** → Partners tab → **Add partner**
  *Expect:* GATA appears in the list with 0 clusters and 0 groups.
- [ ] **B2 Partner name is required** — Add partner with an empty name → Save.
  **Where:** 🌐 Web → Partners & Clusters → **Add partner**
  *Expect:* an error message; nothing is created.
- [ ] **B3 Create clusters** — Clusters tab → **Add cluster** "Arusha North" (partner GATA, region Arusha); add a second one, "Moshi".
  **Where:** 🌐 Web → Partners & Clusters → **Clusters** tab → **Add cluster**
  *Expect:* both are listed under GATA.
- [ ] **B4 Move a group** — Groups → Mkusanyiko Group → Overview → Cluster dropdown → "Arusha North — GATA".
  **Where:** 🌐 Web → sidebar **Groups** → Mkusanyiko Group → Overview tab → **Cluster** dropdown
  *Expect:* the header reads **GATA › Arusha North**, and on Partners & Clusters, Arusha North shows 1 group.
- [ ] **B5 Deactivate / reactivate** — Deactivate the "Moshi" cluster → confirm.
  **Where:** 🌐 Web → Partners & Clusters → Clusters tab → Moshi row → **Deactivate / Reactivate**
  *Expect:* its badge changes to Inactive and nothing is deleted. Then reactivate it.
- [ ] **B6 Cluster link** — Click the cluster name "Arusha North".
  **Where:** 🌐 Web → Partners & Clusters → Clusters tab → click the cluster name
  *Expect:* the Groups page opens, filtered to that cluster.

## C. Users & Roles — 🌐 Web (as Super Admin)

- [ ] **C1 Add staff** — Users & Roles → **Add person** → phone 0711000001, name Vera, role *HelaBox Staff*, preset *Viewer* → Save.
  **Where:** 🌐 Web → sidebar **Users** (Users & Roles) → **Add person**
  *Expect:* Vera appears with role Staff and preset Viewer.
- [ ] **C2 Add partner user** — Add 0711000002, Paul, role *Partner User*.
  **Where:** 🌐 Web → Users & Roles → **Add person**
- [ ] **C3 Add cluster manager** — Add 0711000004, Clara, role *Cluster Manager*.
  **Where:** 🌐 Web → Users & Roles → **Add person**
- [ ] **C4 Assign** — Vera: **+ Assign** → Cluster → **Moshi**. Paul: **+ Assign** → Partner → **GATA**. Clara: **+ Assign** → Cluster → **Arusha North**.
  **Where:** 🌐 Web → Users & Roles → People tab → the person's row → **+ Assign**
  *Expect:* each person shows a badge like "Cluster: Moshi".
- [ ] **C5 Remove an assignment** — Click the **×** on a badge → confirm, then add it back.
  **Where:** 🌐 Web → Users & Roles → People tab → **×** on an access badge
  *Expect:* the badge disappears, then returns.
- [ ] **C6 You can't demote yourself** — Try to change your own role.
  **Where:** 🌐 Web → Users & Roles → your own row ("you") → Role dropdown
  *Expect:* the dropdown is disabled for your row.
- [ ] **C7 Deactivate instead of delete** — Click Vera's **Active** badge → confirm.
  **Where:** 🌐 Web → Users & Roles → Vera's row → **Active** badge in the Status column
  *Expect:* the row fades and shows Inactive. Reactivate her afterwards (needed in section D).
- [ ] **C8 Pending admins tab** — Open the "Group admins without a group" tab.
  **Where:** 🌐 Web → Users & Roles → **Group admins without a group** tab
  *Expect:* it lists only Group Admins who don't run a group yet (you are **not** listed, because you run Mkusanyiko).

## D. What each role sees — 🌐 Web (one incognito window per person)

- [ ] **D1 Partner user** — Log in as Paul (0711000002 / 1234).
  **Where:** 🌐 Web (new incognito window) → `http://localhost:5173` → log in as Paul → Dashboard + sidebar
  *Expect:* the sidebar has Dashboard, Groups, Partners & Clusters and Reports — **no** Users, SMS, Audit Logs or Settings. The dashboard counts only GATA's groups (Mkusanyiko), and the partner filter only lists GATA.
- [ ] **D2 Partner user can't change groups** — As Paul, open Mkusanyiko Group.
  **Where:** 🌐 Web (Paul's incognito window) → Groups → Mkusanyiko Group
  *Expect:* no Edit, Close or Delete buttons.
- [ ] **D3 Partner user blocked by URL** — As Paul, type `http://localhost:5173/users` in the address bar.
  **Where:** 🌐 Web (Paul's incognito window) → address bar → `http://localhost:5173/users`
  *Expect:* you're sent back to the dashboard.
- [ ] **D4 Staff viewer sees only their cluster** — Log in as Vera (assigned to Moshi, which has no groups).
  **Where:** 🌐 Web (new incognito window) → log in as Vera → Dashboard, then Groups
  *Expect:* the dashboard shows **0 groups**, Groups is empty, and Mkusanyiko is **not** listed.
- [ ] **D5 Cluster manager** — Log in as Clara.
  **Where:** 🌐 Web (new incognito window) → log in as Clara → Groups → **Create Group**
  *Expect:* she sees Mkusanyiko (it's in Arusha North) and a **Create Group** button. When creating a group, she must pick a cluster (only Arusha North is offered).
- [ ] **D6 No dashboard role** — Do this after F2 and before G6 (while Katibu is still an officer): log in on the web as 0711000003.
  **Where:** 🌐 Web (new incognito window) → log in as 0711000003
  *Expect:* a "You don't have access to this dashboard" page that tells them to use the mobile app for their group.
- [ ] **D7 Deactivated person is locked out** — Deactivate Vera (C7), then refresh her window.
  **Where:** 🌐 Web → your window: Users & Roles → deactivate Vera · then Vera's incognito window → refresh
  *Expect:* she's sent to the login page and can't get back in. Reactivate her afterwards.

## E. Groups — 🌐 Web (as Super Admin)

- [ ] **E1 Create with cluster** — Groups → Create Group → pick cluster "Moshi", name "Test Delete Group" → Create.
  **Where:** 🌐 Web → Groups → **Create Group**
  *Expect:* the group opens and its header shows GATA › Moshi.
- [ ] **E2 Safe delete (no money)** — On "Test Delete Group" → **Delete Group** → confirm.
  **Where:** 🌐 Web → Groups → Test Delete Group → **Delete Group** (top right)
  *Expect:* it's deleted.
- [ ] **E3 Delete blocked when there's money** — On Mkusanyiko Group → Delete Group → confirm.
  **Where:** 🌐 Web → Groups → Mkusanyiko Group → **Delete Group** (top right)
  *Expect:* an error saying it has financial records and should be closed instead. The group is still there.
- [ ] **E4 Close / reopen** — Mkusanyiko → **Close group**, then **Activate**.
  **Where:** 🌐 Web → Groups → Mkusanyiko Group → **Close group** / **Activate** (top right)
  *Expect:* the badge changes to Closed, then back to Active.
- [ ] **E5 Pagination** — With more than 8 groups, the page buttons really change which groups are listed.
  **Where:** 🌐 Web → Groups → page buttons under the list

## F. Mwenyekiti (you) — 📱 App (+ 🌐 Web for F7)

- [ ] **F1 Position shown** — Group tab. *Expect:* "**Chairperson** · …" under the group name. The Profile tab shows the same.
  **Where:** 📱 App → **Group** tab (under the group name) · then **Profile** tab
- [ ] **F2 Add an officer** — Group → **Group officers** → Add officer → phone 711000003, name "Katibu Test", position **Secretary** → save.
  **Where:** 📱 App → Group tab → **Group officers** → **Add officer**
  *Expect:* they appear in the list as "Secretary · 255711000003".
- [ ] **F3 Officer can't be added twice** — Add the same phone again.
  **Where:** 📱 App → Group tab → Group officers → **Add officer**
  *Expect:* "already an officer of a group".
- [ ] **F4 Record a contribution** — Record a mandatory savings contribution of the exact configured amount for a member.
  **Where:** 📱 App → **Meetings** tab → Meeting #1 → **Start meeting** → Meeting activity → **Record contribution** · then 🖥️ API console for the SMS
  *Expect:* the Home balance and Savings go up. The API console prints an SMS starting with **HELABOX:**.
- [ ] **F5 Reverse — reason required** — Transactions → open that contribution → **Correct / reverse transaction** → leave Reason empty → Reverse.
  **Where:** 📱 App → **Activity** tab (transactions) → tap the contribution → **Correct / reverse transaction**
  *Expect:* "Explain why…" error; nothing changes.
- [ ] **F6 Reverse** — Enter a reason → **Reverse transaction** → confirm.
  **Where:** 📱 App → Activity tab → the contribution → Correct / reverse transaction → **Reverse transaction**
  *Expect:* "Transaction reversed"; Savings drops back by that amount; the transaction shows as reversed and the reverse button is gone.
- [ ] **F7 Reversed shows on the web** — Web → Mkusanyiko → Transactions tab.
  **Where:** 🌐 Web → Groups → Mkusanyiko Group → **Transactions** tab
  *Expect:* the row is struck through, with a "reversed" badge and the reason.
- [ ] **F8 Closed meeting is locked** — Close a meeting (Review & Close), then try to record attendance or money against it.
  **Where:** 📱 App → **Meetings** tab → a meeting → Review & Close · then try Attendance / record money on it
  *Expect:* "this meeting is closed and can no longer be changed".
- [ ] **F9 Record a government loan** — Group → **Government loans** → Record → lender "Halmashauri ya Arusha", programme "10% women loan", amount 1,000,000, interest 0, term 12 → Save.
  **Where:** 📱 App → Group tab → **Government loans** → **Record government loan**
  *Expect:* Received TZS 1,000,000 and Still owed TZS 1,000,000; the card shows Active with a due date 12 months away.
- [ ] **F10 Repay part** — Record repayment 250,000.
  **Where:** 📱 App → Group tab → Government loans → the loan card → **Record repayment**
  *Expect:* Balance 750,000.
- [ ] **F11 Over-repayment blocked** — Record repayment 900,000.
  **Where:** 📱 App → Group tab → Government loans → the loan card → **Record repayment**
  *Expect:* "exceeds the remaining balance".
- [ ] **F12 Group savings unchanged** — After F9–F10, Home → Savings is **not** increased by the government loan.
  **Where:** 📱 App → **Home** tab → Savings card
- [ ] **F13 Settings sender ID** — Profile → Settings.
  **Where:** 📱 App → Profile tab → **Settings** → About → SMS sender ID
  *Expect:* SMS sender ID shows **TUKIIO** (not PESABOX).

## G. Group Officer (Katibu) — 📱 App

Log out in the app, then log in with 0711000003 / 1234.

- [ ] **G1 Home group** — *Expect:* Mkusanyiko Group opens and the Group tab shows "**Secretary** · …".
  **Where:** 📱 App → Profile → Log out → log in as 0711000003 → **Group** tab
- [ ] **G2 Officer can record money** — Record a contribution. *Expect:* it succeeds.
  **Where:** 📱 App (as Katibu) → Meetings tab → an open meeting → Meeting activity → **Record contribution**
- [ ] **G3 Officer can't manage officers** — Group → Group officers.
  **Where:** 📱 App (as Katibu) → Group tab → **Group officers**
  *Expect:* the list shows, but there's no Add officer button or remove icons, and the note says only the Mwenyekiti can change officers.
- [ ] **G4 No close cycle** — Meetings tab. *Expect:* no "Close cycle →" link.
  **Where:** 📱 App (as Katibu) → **Meetings** tab → cycle card at the top
- [ ] **G5 Officer can reverse** — Reverse the contribution from G2 (with a reason). *Expect:* it succeeds.
  **Where:** 📱 App (as Katibu) → Activity tab → the G2 contribution → Correct / reverse transaction
- [ ] **G6 Remove the officer** — Log back in as yourself → Group officers → remove Katibu Test.
  **Where:** 📱 App → Log out → log in as yourself → Group tab → Group officers → remove icon
  *Expect:* removed. When Katibu logs in again, they get "Waiting for your group".

## H. Dashboard & reports — 🌐 Web

- [ ] **H1 KPI cards** — Dashboard. *Expect:* Active groups, Members (women / men), PAR 30, Attendance, Shares and Government loans (**TZS 750,000** after F10) all show numbers.
  **Where:** 🌐 Web → sidebar **Dashboard** → KPI cards (second row)
- [ ] **H2 Filters** — Pick partner GATA, then cluster Moshi.
  **Where:** 🌐 Web → Dashboard → **All partners / All clusters** dropdowns (top right)
  *Expect:* the numbers change (Moshi → 0 groups).
- [ ] **H3 Roll-up** — Click By partner / By cluster / By group.
  **Where:** 🌐 Web → Dashboard → "Performance by level" table → **By partner / By cluster / By group**
  *Expect:* one row per partner / cluster / group. Clicking a group row opens that group.
- [ ] **H4 Report preview** — Reports → **Summary by Cluster** → Preview.
  **Where:** 🌐 Web → sidebar **Reports** → Programme → Summary by Cluster → **Preview →**
  *Expect:* rows per cluster with Swahili or English column headings, following the language switch.
- [ ] **H5 New datasets** — Preview **Government Loans** (shows the Halmashauri loan) and **Portfolio at Risk (PAR 30)** (only loans more than 30 days overdue).
  **Where:** 🌐 Web → Reports → Financial → Government Loans / Portfolio at Risk → **Preview →**
- [ ] **H6 Export** — Select 3 datasets, format Excel → Export; repeat as PDF.
  **Where:** 🌐 Web → Reports → tick datasets → Export card (right) → **Excel** / **PDF** → Export
  *Expect:* files named `helabox-report-….xlsx` / `.pdf`; the PDF header says **HelaBox Report** / **Ripoti ya HelaBox**.
- [ ] **H7 Report scope** — As Paul (partner), open Reports.
  **Where:** 🌐 Web (Paul's incognito window) → Reports
  *Expect:* only GATA / its clusters in the filters, and his exports contain only GATA's groups.

## I. Audit log — 🌐 Web (as Super Admin)

- [ ] **I1 Entries exist** — Audit Logs. *Expect:* entries for what you did above: partner/cluster created, user added, assignment, officer added, contribution recorded, **reversal (with the reason)**, government loan, deactivate/reactivate.
  **Where:** 🌐 Web → sidebar **Audit Logs**
- [ ] **I2 Filters** — Filter by action "Reversed" and by a date range. *Expect:* the list narrows correctly.
  **Where:** 🌐 Web → Audit Logs → filter dropdowns + dates at the top
- [ ] **I3 Hidden from others** — As Paul, open `/audit`. *Expect:* sent away (no access).
  **Where:** 🌐 Web (Paul's incognito window) → address bar → `http://localhost:5173/audit`

## J. Settings & SMS — 🌐 Web (as Super Admin)

- [ ] **J1 Real SMS info** — Settings. *Expect:* provider **Beem Africa**, sender **TUKIIO**, delivery "Dev mode (no Beem keys)", and the admin list shows real people (you, Vera).
  **Where:** 🌐 Web → sidebar **Settings** → SMS card, System Administrators, System Health
- [ ] **J2 Switches save immediately** — Toggle "Transactional SMS" off, refresh the page. *Expect:* it's still off. Turn it back on.
  **Where:** 🌐 Web → Settings → SMS card → **Transactional SMS** switch → refresh the page
- [ ] **J3 Template brand** — SMS → Templates. *Expect:* templates start with `{BRAND}:`, and `BRAND` is listed as a variable.
  **Where:** 🌐 Web → sidebar **SMS** → **Templates** tab

## K. Sessions — 📱 App + 🌐 Web

- [ ] **K1 Expired app session** — Leave the app idle longer than 15 minutes (or clear its session), then reopen it.
  **Where:** 📱 App → leave it idle 15+ minutes → reopen
  *Expect:* it either continues silently, or lands on the welcome/login screen — **never** "Waiting for your group" for your own account.
- [ ] **K2 Web logout** — Click Logout (Toka), then press the browser's Back button.
  **Where:** 🌐 Web → sidebar **Logout** (Toka) → browser Back button
  *Expect:* you stay on the login page.

- [ ] **K3 App reads the local server** — Run the app from VS Code with the default "HelaBox - LOCAL server" config.
  **Where:** VS Code → Run and Debug → **HelaBox - LOCAL server** → 📱 App → log in → compare with 🌐 Web Dashboard
  *Expect:* after login it shows the same group and balances as the web dashboard (not "Waiting for your group").

## L. Technical checks — 💻 Git Bash (optional, for developers)

Run in Git Bash with the API on 8090.

- [ ] **L1 Logged-out GraphQL sees nothing** — run:
  **Where:** 💻 Git Bash (API running on 8090)
  `curl -s -H 'Content-Type: application/json' localhost:8090/graphql -d '{"query":"{ groups { id } }"}'`
  *Expect:* `{"data":{"groups":[]}}`.
- [ ] **L2 Unit tests** — `go test ./server/` → ok, and `cd pesa_box_app && flutter test` → all pass.
  **Where:** 💻 Git Bash → repo folder, then `pesa_box_app` folder

---

## Results log

| Date | Case | Result | Notes |
|------|------|--------|-------|
| | | | |
