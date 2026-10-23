# HelaBox — Manual Test Cases (A → Z)

📱 **Part 1 — App (A–L)** first, then 🌐 **Part 2 — Web (M–Z)**. Work top to bottom: later cases reuse data made earlier.
Tick `[x]` when a case passes. If it fails, leave it unticked and add a row to the **Results log** at the bottom.

Covers: the client's role spec (`Pesa Box User Levels.pdf`), the concept note (`PesaBox-Concept.pdf`) and **GATA's monitoring
feedback** (the 11 indicators — see the map below). Indicator definitions: `server/MONITORING_API.md`.

---

## 0. Before you start (local only — don't test on the live server)

1. **MongoDB:** `C:\Users\Administrator\mongodb\mongodb-win32-x86_64-windows-8.0.4\bin\mongod.exe --dbpath C:\Users\Administrator\mongodb\data\db --bind_ip 127.0.0.1 --port 27017`
2. **API (port 8090):** `cd server && go build -o pesabox-server.exe . && ./pesabox-server.exe` — keep this console visible (SMS texts print here).
3. **Web:** `cd web && npm install && npm run dev` → `http://localhost:5173`
4. **App on the LOCAL server — important:** `pesa_box_app/lib/services/api_base.dart` currently has
   `kUseLiveServer = true`, so a normal debug run talks to the **live** server (157.245.46.244). For these tests run:
   `cd pesa_box_app && flutter run -d emulator-5554 --dart-define=API_BASE_URL=http://10.0.2.2:8090`
   (or set `kUseLiveServer = false` while testing — put it back before you build a release).
   Switching servers asks you to log in again ("Domain mismatch") — that's expected.
- **Dev mode:** with no Beem keys in `server/.env`, every OTP is **`1234`** and SMS are printed in the API console, not sent.
- **One browser window per person** on the web (a login belongs to the browser it was made in) — use incognito windows.

### Test logins (local database, OTP 1234)

| Level | Who | Phone | Use on | Home group / scope |
|---|---|---|---|---|
| 1 Super Admin | You | 0625689904 | 🌐 + 📱 | everything (in the app your home group is the one you hold a position in — check the Group tab) |
| 2 Staff — Operations | Olivia | 0711000013 | 🌐 | all groups; create/edit groups |
| 2 Staff — Support | Samuel | 0711000012 | 🌐 | all groups; create only |
| 2 Staff — Viewer | Stella | 0711000011 | 🌐 | GATA groups, read-only |
| 3 Partner User | Paul | 0711000002 | 🌐 | GATA only |
| 4 Cluster Manager | Clara | 0711000004 | 🌐 | Arusha North (Mkusanyiko) only |
| 5 Group Admin (Mwenyekiti) | Upendo | 0711000005 | 📱 | Upendo Vikoba |
| 5 Group Admin (Mwenyekiti) | Tumaini | 0711000007 | 📱 | Tumaini Group |
| 6 Group Officer (Katibu) | Kassim | 0711000003 | 📱 | Mkusanyiko |
| 7 Group Member | Mary | 0711000006 | 📱 | nothing yet (member self-service = phase 3) |
| new — QA group admin | QA Mwenyekiti | 0711000021 | 📱 | QA Monitoring Group (created in M7) |

### GATA's indicators → where to test them

| # | GATA indicator | Built as | Test cases |
|---|---|---|---|
| 1 | Savings — total, per member, growth | Monitoring → Savings; reports *Member Savings*, *Monthly Growth* | D2–D4, T2, U3 |
| 2 | Loans — disbursed, # borrowers, outstanding | Monitoring → Loans; *Loans* report | E1–E4, T3 |
| 3 | Repayments — rate, overdue, defaults | Monitoring → Loan Repayments (default = 90+ days overdue); *PAR 30* | E5–E7, T4 |
| 4 | Social fund — contributions, disbursements, balance | Social fund payout (new); Monitoring → Social Fund | G1–G4, T5 |
| 5 | Membership — active, new, retention, attendance by **gender & age** | Date of birth on members (new); Monitoring → Membership | C1–C5, D1, T6 |
| 6 | Digital adoption — groups using the app, % recorded digitally | Monitoring → Digital Adoption (👤 GATA to confirm the definition) | T7 |
| 7 | Financial performance — interest, expenses, shared out, returns | Share-out (new, real); Monitoring → Financial Performance | E5, F3, J1–J6, T8 |
| 8 | Accountability — accuracy, reconciliation, transparency | Reversals, audit log, totals drift, expected cash; Monitoring → Accountability | H1–H5, T9, X1–X3 |
| 9 | Growth over time | Monthly series + chart; *Monthly Growth* report | T10, U3 |
| 10 | Access — % saving regularly, % borrowing, avg per member | Monitoring → Access (regular = saved at ≥75% of meetings) | T11 |
| 11 | Economic benefits — IGA loans, enterprise types, returns | Loan purpose + enterprise type (new); Monitoring → Economic Benefits | E1–E2, T12, U4 |

---

# PART 1 — 📱 APP

## A. Login, language, session

- [ ] **A1 Unknown number refused** — Get started → phone 0799 123 456.
  *Expect:* "Namba hii ya simu haijasajiliwa…"; no code in the API console.
- [ ] **A2 Login** — Log in as Upendo (0711000005 / 1234).
  *Expect:* Home shows **Upendo Vikoba**, "Habari, …", balances (Salio la kikundi, Akiba, Hisa, Mfuko wa Jamii, Mikopo).
- [ ] **A3 Branding** — Welcome screen and phone app list. *Expect:* **H** logo and the name **HelaBox**.
- [ ] **A4 Language** — Profile → Settings → Language → English, then back to Kiswahili.
  *Expect:* every screen switches, including meeting titles ("Mkutano #1"), month names (Okt, Mac…), fine reasons and "N shares/hisa".
- [ ] **A5 Deactivated account refused** — (after P5 on the web) try to log in as a deactivated person. *Expect:* "Akaunti hii imezimwa…".
- [ ] **A6 Dead session → login** — Leave the app idle 15+ minutes and reopen it (or restart the API with the app open and pull to refresh).
  *Expect:* it continues silently or returns to the welcome/login screen — **never** "Waiting for your group" for a real group admin.
- [ ] **A7 Switching users clears data** — Log out, log in as Kassim (0711000003).
  *Expect:* only **Mkusanyiko** data — no Upendo transactions, loans or meetings left over from the previous user.

## B. Home & meetings (log in as Upendo)

- [ ] **B1 Overdue meeting** — Meetings → schedule a meeting dated **yesterday**.
  *Expect:* red **Imechelewa / Overdue** pill on Home, in the meetings list and in the meeting details (not "Unakuja").
- [ ] **B2 Start screen explains itself** — Home → **Anza mkutano**.
  *Expect:* it opens the **earliest scheduled** meeting and says "Unaanza Mkutano #N uliopangwa kufanyika …" with its date, a warning that it's overdue, and a link to schedule a different meeting instead.
- [ ] **B3 Quick action order** — Start that meeting, go back Home.
  *Expect:* the quick action now says **Endelea na mkutano** (continue). After closing it, it goes to the next scheduled meeting, or to **Unda mkutano** if none — never to a closed meeting.
- [ ] **B4 Status pills** — *Expect:* Unakuja (future) · Imechelewa (past, not started) · Inaendelea (in progress) · Imefungwa (closed). In-progress meetings are **not** listed under History.
- [ ] **B5 Closed meeting is locked** — Close a meeting (Review & Close), then try attendance or money on it.
  *Expect:* "this meeting is closed and can no longer be changed" (translated in Swahili).

## C. Members & date of birth

- [ ] **C1 Add member with DOB** — Members → Add → "QA Neema", female, DOB 10 May 2000.
  *Expect:* saved; member details show "Born 10 May 2000 · 26 years" (Swahili: "Alizaliwa …").
- [ ] **C2 DOB optional** — Add "QA Peter", male, no DOB. *Expect:* saved; details show no age.
- [ ] **C3 Bad DOB refused** — Edit a member, pick a future date (if the picker allows) or an age under 10.
  *Expect:* refused with a clear message.
- [ ] **C4 Clear DOB** — Edit QA Neema → clear DOB → save → set it back. *Expect:* both saves work.
- [ ] **C5 Deactivate** — Deactivate a QA member. *Expect:* status shows **Inactive** (not Active); they can't borrow.

## D. Attendance & contributions (inside an open meeting)

- [ ] **D1 Attendance** — Mark present / late / absent / excused (Ameomba udhuru).
  *Expect:* all four cards the same height; counts correct.
- [ ] **D2 Mandatory savings** — Record the configured mandatory amount for a member. *Expect:* Home Akiba goes up; API console prints an SMS starting **HELABOX:**.
- [ ] **D3 Shares** — Record 3 shares. *Expect:* "3 hisa", value = 3 × share value; a number above max shares is refused.
- [ ] **D4 Social fund** — Record the configured social fund amount. *Expect:* Mfuko wa Jamii goes up.
- [ ] **D5 Funds screen** — Group → Funds. *Expect:* Savings / Shares / Social fund rows show "in · out" in Swahili, reversed rows not counted.

## E. Loans (purpose, interest, limit, repayment)

- [ ] **E1 Loan with purpose** — Loans → Record loan → member, 10,000 → **Madhumuni: Biashara**, **Aina ya biashara: "Mama lishe"**.
  *Expect:* preview shows interest + total due; loan details show "Biashara · Mama lishe".
- [ ] **E2 Enterprise field only for IGA** — Pick purpose Elimu (education). *Expect:* the enterprise-type field disappears.
- [ ] **E3 No purpose** — Record a loan without purpose. *Expect:* details show **Haijabainishwa**.
- [ ] **E4 Loan limit** — With max loan multiplier set (Rules), ask for more than the member's limit; try a loan for an inactive member.
  *Expect:* refused with the reason.
- [ ] **E5 Partial repayment** — Repay half the total due. *Expect:* balance = total due − repaid; loan stays Active.
- [ ] **E6 Over-repayment blocked** — Repay more than the remaining balance. *Expect:* refused.
- [ ] **E7 Full repayment** — Repay the rest. *Expect:* loan becomes Repaid; loan duration shows the loan's own term ("miezi N").

## F. Fines & expenses

- [ ] **F1 Fine** — Record a fine (reason from the group's list). *Expect:* reason shown in Swahili in the fines list.
- [ ] **F2 Pay fine** — Pay it. *Expect:* status Paid; Home updates.
- [ ] **F3 Expense** — Record an expense 1,500. *Expect:* it appears in activity and in Funds "out".

## G. Social fund payout (new)

- [ ] **G1 Who sees it** — As Upendo (Mwenyekiti) and as Kassim (Katibu): Funds → **Social fund payout** is visible (both have finance.write).
- [ ] **G2 Payout** — Pay 1,500 to a member, reason "Hospitali".
  *Expect:* recorded; social fund balance drops by 1,500; SMS printed in the API console; activity shows "Malipo ya mfuko wa jamii".
- [ ] **G3 Can't overdraw** — Pay more than the fund holds. *Expect:* refused before sending ("the social fund only holds …").
- [ ] **G4 Reverse** — Reverse the payout with a reason. *Expect:* the fund goes back up by exactly 1,500.

## H. Reversals (money is never deleted or edited)

- [ ] **H1 Reason required** — Activity → a contribution → Correct / reverse → empty reason. *Expect:* "Explain why…"; nothing changes.
- [ ] **H2 Reverse** — Enter a reason → reverse. *Expect:* balance drops back; row shows reversed; reverse button gone.
- [ ] **H3 Reverse a repayment** — Reverse the E5 repayment. *Expect:* the loan balance goes back up.
- [ ] **H4 No delete anywhere** — There is no delete/edit for any money record in the app.
- [ ] **H5 Shows on the web** — Web → that group → Transactions. *Expect:* struck through, "reversed" badge and the reason.

## I. Government loans (loan to the group)

- [ ] **I1 Record** — Group → Mikopo ya serikali → lender "QA Halmashauri", 1,000,000, interest 0, 12 months.
  *Expect:* Received 1,000,000; Still owed 1,000,000; Active.
- [ ] **I2 Repay part** — 250,000. *Expect:* balance 750,000.
- [ ] **I3 Over-repayment** — 900,000. *Expect:* translated error, refused.
- [ ] **I4 Savings unchanged** — Home Akiba is **not** increased by the government loan.

## J. Share-out & close cycle (new — was a mock before)

Use the **QA Monitoring Group** after section T has been checked (its numbers are worked out there), or any test group.

- [ ] **J1 Who sees it** — Mwenyekiti: Meetings → cycle card → **Mgao →** with a "Also close cycle" tick box. Katibu: **Mgao →** but **no** close-cycle tick box.
- [ ] **J2 Suggested amounts** — Open Mgao. *Expect:* each member = their savings + shares + a pro-rata share of the profit; a warning if loans are still outstanding; the total checked against cash available (cash − social fund).
- [ ] **J3 Over-entitlement warning** — Edit one member above their suggestion.
  *Expect:* red line "Zaidi ya stahili ya mwanachama huyu (TZS X)"; if you submit, the server refuses and its message is shown above the button.
- [ ] **J4 Pay out** — Pay the suggested amounts **without** closing the cycle.
  *Expect:* one `Mgawanyo wa akiba` transaction per member; SMS per member in the API console; Funds shows a **Share-out** row "… paid · … of it profit"; Savings/Shares "out" grow by the savings/shares returned.
- [ ] **J5 Reverse a share-out** — Reverse one member's payout. *Expect:* their savings/shares come back exactly.
- [ ] **J6 Close cycle** — Pay out again with **Also close cycle** ticked. *Expect:* cycle number +1. Reverse all those payouts → the cycle number goes back.

## K. Roles in the app

- [ ] **K1 Katibu** (0711000003) — Group tab shows "Katibu/Secretary"; can record money, attendance and reversals; **no** Add/remove officer, **no** Edit rules.
- [ ] **K2 Mwenyekiti adds an officer** — Group → Viongozi wa kikundi → add phone 0711000099 as Committee. *Expect:* listed; adding the same phone again → "already an officer". Remove it afterwards.
- [ ] **K3 Member** (0711000006) — *Expect:* "Waiting for your group" screen (member self-service is phase 3).
- [ ] **K4 Other group's data hidden** — As Tumaini (0711000007): only Tumaini Group and its TZS 50,000 loan.

## L. Rules, announcements, statements, profile

- [ ] **L1 Rules** — As Mwenyekiti: Group → Rules & Constitution → Edit → change social fund amount → save. *Expect:* saved; the "Huduma" row doesn't overflow; Katibu sees no Edit button.
- [ ] **L2 Announcement** — Add "QA Notice". *Expect:* listed.
- [ ] **L3 Group statement PDF** — Reports → Group statement. *Expect:* opens; totals skip reversed rows; share-out and payouts labelled.
- [ ] **L4 Member statement PDF** — A member → Statement. *Expect:* opens with that member's records only.
- [ ] **L5 Settings** — Profile → Settings. *Expect:* SMS sender ID **TUKIIO**.
- [ ] **L6 Offline** — Turn off emulator data, reopen the app. *Expect:* opens with proper fonts, no freeze.
- [ ] **L7 Logout** — Profile → Log out. *Expect:* welcome screen; Back doesn't return into the app.

---

# PART 2 — 🌐 WEB (`http://localhost:5173`)

## M. Login, language, layout, groups

- [ ] **M1 Login** — 0625689904 / 1234. *Expect:* dashboard; sidebar "HelaBox Admin" + your role. Tab title **HelaBox — Admin**.
- [ ] **M2 Language** — SW / EN in the top bar. *Expect:* no raw keys like `mon.title`; first visit opens in **English**.
- [ ] **M3 Phone width** — Narrow the window to ~390px (or DevTools phone view).
  *Expect:* a **menu button** in the top bar opens the sidebar as a drawer (it closes on navigation / tapping outside); no sideways scrolling on any page.
- [ ] **M4 Unknown number** — Log in with 0799 123 456. *Expect:* "not registered"; no code sent.
- [ ] **M5 Old finance page gone** — Open `http://localhost:5173/finance`. *Expect:* you land on Reports (it used to show made-up figures).
- [ ] **M6 Groups list** — Groups. *Expect:* real groups, working pagination.
- [ ] **M7 Create QA Monitoring Group** — Create Group → name **QA Monitoring Group**, cluster Arusha North, admin phone **0711000021**; then Financial → Edit rules:
  share value 1,000 · min 1 · max 5 shares · mandatory savings 5,000 · social fund 2,000 · interest 10% · max loan 3× · Save.
  *Expect:* created; rules saved; Audit has GroupRules.
- [ ] **M8 Members & loans columns** — Upendo Vikoba → Members tab: **Date of birth** + **Age** columns (QA Neema: 10 May 2000, 26). Financial tab → loans: **Loan purpose** column with enterprise type underneath ("Biashara · Mama lishe"; old loans "Haijabainishwa").
- [ ] **M9 Safe delete** — Create "QA Delete Me" → Delete Group → deleted. Delete Mkusanyiko → refused ("has financial records").
- [ ] **M10 Close / reopen group** — Close group → Activate. *Expect:* badge changes; nothing deleted.

## N. Partners & clusters (Super Admin)

- [ ] **N1 Add partner** — "QA Partner", type NGO. *Expect:* listed with 0 clusters. Empty name → error.
- [ ] **N2 Add cluster** — "QA Cluster" under QA Partner. *Expect:* listed.
- [ ] **N3 Move a group** — QA Delete Me (or a QA group) → Cluster dropdown → QA Cluster. *Expect:* header shows QA Partner › QA Cluster.
- [ ] **N4 Deactivate / reactivate** — QA Cluster. *Expect:* Inactive badge; nothing deleted.
- [ ] **N5 Cluster link** — Click a cluster name. *Expect:* Groups filtered to it.

## O. Users & assignments (Super Admin)

- [ ] **O1 Add staff** — Add 0711000031 "QA Viewer", Staff / Viewer. *Expect:* listed.
- [ ] **O2 Assign** — + Assign → Partner → GATA. *Expect:* badge "Partner: GATA". × removes it, then add it back.
- [ ] **O3 Own role locked** — Your own row's role dropdown is disabled.
- [ ] **O4 Deactivate** — Click QA Viewer's Active badge. *Expect:* Inactive; their open browser is sent to login; logging in → "deactivated". Reactivate.
- [ ] **O5 Pending group admins** — tab "Group admins without a group" lists only admins with no group.

## P. What each role sees (one incognito window each)

- [ ] **P1 Staff Operations** (0711000013) — menu: Dashboard, Monitoring, Groups, Partners & Clusters, Reports, SMS. `/users`, `/audit`, `/settings` send you back. Can Create Group and Edit rules.
- [ ] **P2 Staff Support** (0711000012) — all groups; Create Group; **no** Edit / Close / Delete / Edit rules.
- [ ] **P3 Staff Viewer** (0711000011) — only GATA's groups; no create or edit buttons.
- [ ] **P4 Partner User** (0711000002) — only GATA; no SMS / Users / Audit / Settings; partner filter lists only GATA; no SMS datasets in Reports.
- [ ] **P5 Cluster Manager** (0711000004) — only Arusha North groups (Mkusanyiko, QA Monitoring Group); Create Group offers only Arusha North.
- [ ] **P6 Group roles on the web** (0711000005, 0711000003, 0711000006) — every page → "You don't have access… use the mobile app".

## Q. Dashboard

- [ ] **Q1 KPI cards** — Active groups, members (women/men), savings, shares, social fund, loans outstanding, PAR 30, attendance, government loans — all numbers, no "NaN".
- [ ] **Q2 Matches the API** — Loans outstanding on Mkusanyiko's group page = Reports → Summary by Group.
- [ ] **Q3 Filters & roll-up** — Partner / cluster filters change the numbers; "Performance by level" by partner → click GATA → its clusters, breadcrumb back.
- [ ] **Q4 Activity feed** — After G2 and J4: entries read "Malipo ya mfuko wa jamii" / "Mgawanyo wa akiba" (not raw codes).

## R. Record the QA Monitoring Group scenario (📱 App, as 0711000021) — feeds section T

Log in to the app as **0711000021** (the QA group's admin — the first login opens the group as Mwenyekiti), then:

- [ ] **R1 Members** — QA Neema (F, DOB 2000-05-10) · QA Asha (F, DOB 1980-01-15) · QA John (M, DOB 2005-03-01) · QA Peter (M, no DOB).
- [ ] **R2 Meeting #1** — create for today, start; attendance: Neema present · Asha late · John absent · Peter excused.
- [ ] **R3 Money** — Neema: savings 5,000, 3 shares (3,000), social fund 2,000 · Asha: savings 5,000, social fund 2,000.
- [ ] **R4 Loan** — Asha 10,000, purpose **Biashara**, enterprise "Mama lishe" (interest 10% → total due **11,000**).
- [ ] **R5 Repayment** — Asha repays **5,500**.
- [ ] **R6 Payout** — Social fund payout 1,000 to John, reason "Hospitali".
- [ ] **R7 Close** — Review & close the meeting.

## T. Monitoring page — GATA's 11 indicators (🌐 Web → **Monitoring / Ufuatiliaji**)

Filter **Group = QA Monitoring Group**, leave dates empty (the page then says it shows the **last 12 months**).
Each section has a "how it's calculated" line and buttons that open the matching report. Expected numbers for the R-scenario:

- [ ] **T1 Page basics** — period line + "balances as at" date + scope "1 group"; 11 sections in order; works in SW and at phone width; a cluster with no groups shows the empty state.
- [ ] **T2 Savings** — savings recorded **13,000** (savings 10,000 + shares 3,000); contributions **10,000**; shares **3,000**; savers **2**; average per saver **6,500** (Neema 8,000 / Asha 5,000).
- [ ] **T3 Loans** — disbursed **10,000**; borrowers **1**; outstanding **5,500**.
- [ ] **T4 Loan repayments** — overdue **0**, defaults **0** (the loan isn't due yet); repayment rate counts only loans that fell due in the period (see the hint) — so it is 0 / "—" here. Re-check after a loan passes its due date.
- [ ] **T5 Social fund** — contributions **4,000**; payouts **1,000**; balance **3,000**.
- [ ] **T6 Membership** — active **4** (F 2 / M 2); new **4**; age bands 15-24: **1** (John) · 25-35: **1** (Neema) · 36-59: **1** (Asha) · unknown: **1** (Peter); youth (15–35) **2**; date-of-birth coverage **75%**; attendance: present 1, late 1, absent 1, excused 1 → rate **50%** (present + late).
- [ ] **T7 Digital adoption** — groups using the app **1 of 1**; meetings fully recorded (attendance + money) **1 of 1 = 100%**. 👤 GATA to confirm this definition.
- [ ] **T8 Financial performance** — interest earned **500** (5,500 × 1,000 / 11,000, pro-rata); expenses **0**; shared out **0**.
- [ ] **T9 Accountability** — reversals **0**; expected cash on hand **11,500** (in 10,000 + 3,000 + 4,000 + 5,500 − out 10,000 − 1,000); audit entries > 0 (Super Admin only).
- [ ] **T10 Growth** — this month's bar shows savings + shares 13,000 and loan portfolio 5,500; the *Monthly Growth* report has 12 rows.
- [ ] **T11 Access** — saving regularly **2 of 4 = 50%** (Neema, Asha saved at 100% of meetings); borrowing **1 of 4 = 25%**.
- [ ] **T12 Economic benefits** — loans by purpose: Biashara **1**; IGA borrowers **1** (Female, 36-59); enterprise types: "Mama lishe" **1**.
- [ ] **T13 Reversal moves the numbers** — In the app reverse R5 (Asha's 5,500). *Expect:* outstanding **11,000**, interest earned **0**, reversals **1**, expected cash **6,000**. Record the 5,500 again afterwards.
- [ ] **T14 Share-out numbers** — In the app Asha repays the remaining **5,500** (loan Repaid). Mgao suggests Neema **8,615** (8,000 + 615 profit) and Asha **5,384–5,385** (5,000 + her 384.6 share) — profit 1,000 = the interest. First try Asha **6,000** → refused: "TZS 6,000 is more than this member can receive (TZS 5,385 …)". Then pay out the suggestions (J4).
  *Expect:* Monitoring → Financial performance: shared out **13,999**, returns to members **999**, interest earned **1,000**; Accountability: expected cash **≈ 3,001** (the social fund 3,000 stays in the box).
- [ ] **T15 Scope** — As Clara (cluster manager): Monitoring shows only Arusha North groups. As Paul (partner): only GATA. Group roles: no Monitoring at all. SMS/audit counts are hidden from roles without SMS/audit access.
- [ ] **T16 Bad dates** — From after To → error (400), no numbers.

## U. Reports & exports

- [ ] **U1 Every dataset loads** — Reports → preview each dataset with no filter, a date range, a partner, a cluster, a group.
- [ ] **U2 New datasets** — *Monitoring Summary*, *Members by Age*, *Member Savings*, *Loans by Purpose*, *Loans by Enterprise*, *Monthly Growth*, *Social Fund Payouts*, *Share-outs*. *Expect:* rows match section T for QA Monitoring Group.
- [ ] **U3 Member savings & growth** — *Member Savings* lists each member's savings/shares; *Monthly Growth* 12 months.
- [ ] **U4 Loans report** — has **Purpose** and **Enterprise Type** columns; *Members* report has **Date of Birth** and **Age Band**.
- [ ] **U5 Export** — Tick several datasets → PDF, Excel, CSV. *Expect:* files `helabox-…`, non-empty, Swahili/English headers per language, TOTAL row bold.
- [ ] **U6 Link from Monitoring** — On Monitoring click a section's report button. *Expect:* Reports opens with the same filters and previews that dataset.
- [ ] **U7 Scope** — As Paul: only GATA in filters and exports; no SMS datasets.
- [ ] **U8 Dates** — From after To → error; a day with no data → 0 rows; dates shown in East Africa time.

## V. Audit log (Super Admin)

- [ ] **V1 Entries** — Audit Logs shows: partner/cluster/user/assignment changes, group rules, money recorded, **reversals with reason**, social fund payout, share-out, government loan.
- [ ] **V2 Filters** — by action and date range.
- [ ] **V3 Hidden** — As Paul, `/audit` → sent away.

## W. Settings, SMS, Mobile App page

- [ ] **W1 Settings** — provider Beem Africa, sender TUKIIO, "Dev mode (no Beem keys)", real admin list.
- [ ] **W2 SMS switch** — Toggle Transactional SMS off → refresh → still off → turn back on.
- [ ] **W3 SMS page** — KPI cards fit at phone width; templates start with `{BRAND}:`; payout/share-out SMS types have proper names.
- [ ] **W4 Mobile App page** — `/admin/app`: upload an APK → `GET /api/public/app` shows it available; landing page shows the download.

## X. Data rules & security (web + API)

- [ ] **X1 No money delete** — No delete/edit for transactions anywhere on the web.
- [ ] **X2 GraphQL can't delete or write money** — logged-out: `curl -s -H 'Content-Type: application/json' localhost:8090/graphql -d '{"query":"{ groups { id } }"}'` → `{"data":{"groups":[]}}`.
- [ ] **X3 Totals drift** — `/api/admin/totals-drift` (as Super Admin, in the browser) lists any group whose stored totals differ from the computed ones — expect none for QA Monitoring Group.

## Y. Sessions

- [ ] **Y1 Web logout** — Logout → browser Back → stays on login.
- [ ] **Y2 Deactivated person kicked out** — O4.

## Z. Technical checks (💻 Git Bash, optional)

- [ ] **Z1** `go build ./server/ && go vet ./server/ && go test ./server/` → ok
- [ ] **Z2** `cd web && npx vite build` → ok
- [ ] **Z3** `cd pesa_box_app && flutter analyze && flutter test` → only the 10 old info hints; all tests pass
- [ ] **Z4** Before a release build of the app: `kUseLiveServer` / API URL points where you intend.

---

## 👤 Open questions for GATA

1. **Digital adoption** — all HelaBox records are digital, so "% recorded digitally" is measured as % of meetings fully recorded (attendance + money). OK?
2. **Cash reconciliation** — the app has no physical cash count; we show *expected* cash on hand. Do they want a "cash counted" field at meeting close?
3. **Default** = more than 90 days overdue; **regular saver** = saved at ≥ 75% of meetings; **youth** = 15–35. Confirm.
4. **Share-out** doesn't require all loans repaid first (the app warns). Confirm the policy.

---

## Results log

| Date | Case | Result | Notes |
|------|------|--------|-------|
| | | | |
