# HelaBox — Quick Test (about 1 hour)

## Before you start (5 min)

1. Start MongoDB, the API (`cd server && ./pesabox-server.exe`) and the web (`cd web && npm run dev`).
2. Start the app on the **local** server: `cd pesa_box_app && flutter run -d emulator-5554 --dart-define=API_BASE_URL=http://10.0.2.2:8090`
3. Every OTP code is **1234**.
4. 🌐 On the web (`http://localhost:5173`) log in as **0625689904** → Groups → **Create Group**: name **QA Test Group**, cluster **Arusha North**, admin phone **0711000021**. Open it → Financial → **Edit rules**: share value 1,000 · max 5 shares · mandatory savings 5,000 · social fund 2,000 · interest 10% · max loan 3× → Save.

---

# PART 1 — 📱 APP

## A. Log in and add members

- [ ] **Q1 Log in as the new group's chairperson** — App → Get started → 0711000021 → 1234.
  *Expect:* the app opens **QA Test Group** and you are the Mwenyekiti.
- [ ] **Q2 Add 4 members** — Neema (female, born 10 May 2000) · Asha (female, born 15 Jan 1980) · John (male, born 1 Mar 2005) · Peter (male, leave date of birth empty).
  *Expect:* all saved; opening Neema shows her age (26).

## B. Run a meeting

- [ ] **Q3 Overdue meeting** — Meetings → create a meeting dated **yesterday**.
  *Expect:* it shows a red **Imechelewa / Overdue** label, not "Unakuja". Home → **Anza mkutano** opens that meeting and tells you it is the scheduled meeting from yesterday. Start it.
- [ ] **Q4 Attendance** — Neema present · Asha late · John absent · Peter excused.
  *Expect:* the four counts are right.
- [ ] **Q5 Record money** — Neema: savings 5,000, **3 shares**, social fund 2,000 · Asha: savings 5,000, social fund 2,000.
  *Expect:* Home shows shares 3,000 and social fund 4,000. Trying 8 shares is refused (max 5).
- [ ] **Q6 Loan with a purpose** — Loans → Record loan → Asha, 10,000, purpose **Biashara**, business type **Mama lishe**.
  *Expect:* interest 1,000, total to repay **11,000**.
- [ ] **Q7 Repayment** — Asha repays **5,500**. Then try to repay 6,000.
  *Expect:* balance **5,500**; the 6,000 is refused (more than she owes).
- [ ] **Q8 Social fund payout** — Funds → Social fund payout → John, 1,000, reason "Hospitali". Then try 5,000.
  *Expect:* fund goes to **3,000**; the 5,000 is refused (the fund doesn't hold that much).
- [ ] **Q9 Fix a mistake** — Record a 5,000 savings for John, then open it in Activity → Correct / reverse → reason "wrong member".
  *Expect:* it shows as reversed and the savings total is back to what it was. Without a reason it refuses.
- [ ] **Q10 Close the meeting** — Review & close. Then try to record anything in it.
  *Expect:* refused — "this meeting is closed".
- [ ] **Q11 Switch user** — Log out, log in as the Katibu **0711000003**.
  *Expect:* you see **Mkusanyiko** only — nothing from QA Test Group. Log out again.

# PART 2 — 🌐 WEB

## C. The group on the web

- [ ] **Q12 Phone layout** — Make the browser narrow (phone size).
  *Expect:* a menu button appears in the top bar and opens the sidebar; nothing scrolls sideways.
- [ ] **Q13 Members and loans** — Groups → QA Test Group → Members tab, then Financial tab.
  *Expect:* Members show **date of birth and age** (Peter: "—"). The loan shows purpose **Biashara · Mama lishe**. Transactions tab: John's 5,000 is struck through with the reason.

## D. Monitoring page (GATA's indicators)

- [ ] **Q14 Numbers are right** — Sidebar → **Monitoring** → Group = **QA Test Group**.
  *Expect:* the numbers in this table.

| Section | Number to check |
|---|---|
| Savings | recorded **13,000** (savings 10,000 + shares 3,000) · 2 savers |
| Loans | disbursed **10,000** · borrowers **1** · outstanding **5,500** |
| Social fund | in **4,000** · paid out **1,000** · balance **3,000** |
| Membership | **4** members (2 women, 2 men) · ages 15-24: 1 · 25-35: 1 · 36-59: 1 · unknown: 1 · youth **2** · attendance **50%** |
| Digital adoption | meetings fully recorded **1 of 1 (100%)** |
| Financial performance | interest earned **500** |
| Accountability | reversals **1** · expected cash **11,500** |
| Access | saving regularly **50%** · borrowing **25%** |
| Economic benefits | business loans **1** · business type **Mama lishe** |
- [ ] **Q15 Download** — In the Savings section click a report button → Reports opens with that report → Export **PDF** and **Excel**.
  *Expect:* both files open and show the same numbers.

## E. Who can see what

- [ ] **Q16 Partner user** — New incognito window → log in as **0711000002** (GATA partner).
  *Expect:* Monitoring and Reports show only GATA's groups; no Users, Audit or Settings in the menu.
- [ ] **Q17 Group role on the web** — Incognito → log in as **0711000021**.
  *Expect:* "You don't have access" — group roles use the app.
- [ ] **Q18 Audit log** — As 0625689904 → Audit Logs.
  *Expect:* the reversal with reason "wrong member", the social fund payout and the loan are listed.

# PART 3 — 📱 APP + 🌐 WEB: share-out (10 min)

## F. End of cycle

- [ ] **Q19 Share-out** — App as 0711000021: Asha repays the last **5,500**. Meetings → **Mgao** (share-out).
  *Expect:* suggested Neema **8,615** and Asha **5,384** (savings + shares + their part of the 1,000 interest). Change Asha to 6,000 → a red warning; paying it is refused. Put it back and pay.
- [ ] **Q20 Share-out on the web** — Monitoring → QA Test Group.
  *Expect:* Financial performance: shared out **13,999**, interest earned **1,000**, returns to members **999**.

---

## If something fails

Write the check number and what you saw (a screenshot helps) and send it over.

| Check | What happened |
|---|---|
| | |
