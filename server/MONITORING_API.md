# Monitoring indicators API (GATA)

GATA's monitoring indicators, computed by the server from the recorded data.
Code: `server/monitoring.go` (computation + route), `server/member_profile.go`
(date of birth, age bands, loan purpose), `server/payouts.go` (social fund
payouts, share-out), datasets in `server/reports.go` / `server/report_build.go`.
Tests: `server/monitoring_test.go`.

## Endpoint

```
GET /api/admin/monitoring?partnerId=&clusterId=&groupId=&from=YYYY-MM-DD&to=YYYY-MM-DD
```

- **Scope:** same as `/api/admin/dashboard`, `/api/admin/rollup` and the
  reports. It uses `reportScope`, so it needs `reports.view`: platform-wide for
  super admin / staff / partner / cluster manager, or in the home group for a
  group admin or officer. The filters narrow the scope, and the result is
  always limited to what the caller may see. A partner or cluster filter
  outside a platform role's scope gives an empty result (`scope.groups = 0`).
  For a group admin or officer, a filter that points at another group,
  cluster or partner gives **403**. A group member gets **403**.
- **Period:** days are EAT days and `to` is inclusive. `to` defaults to today.
  `from` defaults to the first day of the month 11 months before `to`, which
  gives 12 calendar months. A `from` after the resolved `to` (including a
  future `from` with no `to`) gives 400. **This differs from the other reports, where an
  empty range means all history:** the start of the period is needed for
  retention, growth and the start/end averages. A bad date gives 400.
  `period.asAt` is the day balances are taken: `to`, or today when `to` is in
  the future.
- **Rules every number follows:** reversed transactions never count.
  Cancelled loans never count. Balances are computed from the records, never
  from the Group's stored running totals. At "now", the balances (savings,
  shares, social fund, loans outstanding, active members) are exactly the
  dashboard's (`computeKPIs`); a unit test checks this.
- **Response:** one object per GATA section, plus `scope`, `period` and
  `definitions`. Each section is a flat object of numbers, plus a few small
  arrays for breakdowns. Money is TZS, rounded to 2 decimals. Every `…Pct` /
  `…Rate` is a percentage from 0 to 100 with 1 decimal (0 when its base is 0).
  Breakdown arrays always list every key in a fixed order: gender is
  `Female, Male, unknown`; age band is `under 15, 15-24, 25-35, 36-59, 60+,
  unknown`. So the web can draw them without filling gaps.

## Data the indicators need (new fields and routes)

| What | Contract |
|---|---|
| `Members.dateOfBirth` | Optional string `"YYYY-MM-DD"`. Accepted by `POST /api/members` (create) and `POST /api/main/members/:id` (edit; `""` clears it). It must be a real date, not in the future, giving an age of 10–110 today; anything else gives 400 with a message. A full ISO timestamp is cut to its date. It can be read over GraphQL (`members { dateOfBirth }`). There is no separate admin member route. |
| Age bands | `"under 15"`, `"15-24"`, `"25-35"`, `"36-59"`, `"60+"`, `"unknown"` (no DOB). The age is taken on `period.asAt`. **Youth = 15–35** (Tanzania National Youth Development Policy). |
| `Loans.purpose` | Optional enum: `business, agriculture, livestock, education, health, household, housing, emergency, other`. Accepted by `POST /api/main/loans` (case-insensitive; any other value gives 400). Loans with no purpose (including all older loans) are reported as `"unspecified"`. **Income-generating (IGA) = business, agriculture, livestock.** |
| `Loans.enterpriseType` | Optional free text, at most 80 characters, with spaces collapsed (e.g. "Mama lishe", "Poultry"). It is grouped case-insensitively in breakdowns. |
| Social fund payout | `POST /api/main/social-fund/payout` with body `{ "amount": 3000, "reason": "Funeral", "memberId"?: "...", "meetingId"?: "...", "method"?: "Cash" }`. Needs `finance.write` in the home group. `reason` is required. The amount cannot exceed the fund's balance (social_fund minus payouts, not counting reversed rows). A closed meeting refuses it. It writes a `Transaction` of type `social_fund_payout` (direction `out`, `description` = reason), lowers `Group.totalSocialFund`, is audited, and sends the member an SMS (template `social_fund_payout`). Undo it with `POST /api/main/transactions/:id/reverse`. |
| Share-out (cycle close) | `POST /api/main/share-out` with body `{ "payouts": [{ "memberId": "...", "amount": 2500 }], "meetingId"?: "...", "method"?: "Cash", "closeCycle"?: true }`. Needs `finance.write`; `closeCycle` also needs `group.settings` (the group admin). There must be 1–1000 payouts and no member twice. The total cannot exceed the cash available to share: the cash box from non-reversed transactions, less the social fund balance, which stays in the box. Government-loan money is not counted. Each payout becomes a `Transaction` of type `share_out` (direction `out`) that stores how it splits: `savingsReturned` (the member's own savings come back first), then `sharesReturned`, then `profit` (the member's return), plus `cycle` (`Group.cycleCurrent`, at least 1). `closeCycle: true` adds 1 to `Group.cycleCurrent`. One audit entry is written (model `ShareOut`) and each member gets an SMS (template `share_out`). The response is `{ success, cycle, total, transactionIds, cycleClosed }`. Each payout can be undone with the reverse route, which puts the returned savings and shares back. |
| Cash a share-out may use | **Exact formula** (`groupCash` in `payouts.go`; the app's `lib/services/share_out.dart` must mirror it):<br>`cash = Σ amount of non-reversed transactions of type contribution, share, social_fund, loan_repayment, fine − Σ amount of every other non-reversed transaction − max(0, Σ GovernmentLoanRepayment.amount − Σ GovernmentLoan.amount)`<br>`socialFund = Σ social_fund − Σ social_fund_payout` (non-reversed)<br>`available to share = cash − max(0, socialFund)`<br>Money borrowed from government is not the members' money, so it is left out. Once the group has repaid more than it received (the interest), that extra came out of the members' cash and is subtracted. |
| Share-out entitlement | Each member's **pool** is their savings (contributions − withdrawals − savings returned earlier) plus their shares (bought − shares returned), each at least 0. The **group profit** is `max(0, available − Σ every member's pool)`. A member may receive at most `pool + profit × pool ÷ Σ pools`, plus 1 TZS of rounding. The profit paid across the whole request may not exceed the group profit (+1 TZS per payout). Errors are plain sentences, e.g. "TZS 30,000 is more than this member can receive (TZS 8,000: their savings and shares TZS 8,000 plus their share of the group's profit)". If loans are still out, the box can hold less than the pools: then there is no profit, and members can only get their own money back up to the cash available. |
| Payout safety | Share-outs and social-fund payouts take a per-group lock around the check and the write, so two requests at once cannot both pass. The share-out writes every payment first and sends the SMS afterwards. If a write fails half way, the request answers 500 naming how many payments were recorded, and the `ShareOut` audit entry still lists the transaction ids that were created. |
| Reversing a share-out | Each `share_out` transaction stores `cycle` and `cycleClosed`. Reversing it puts the member's savings and shares back. When it was part of a share-out that closed the cycle and no other non-reversed payment of that close remains, `Group.cycleCurrent` goes back one, never below the cycle that was closed. That step is audited ("Cycle N reopened"). |
| Plain transactions | `POST /api/main/transactions` refuses `social_fund_payout` and `share_out` (400) and points to the routes above. |

After a share-out, a member's savings, shares and loan limit
(`/api/main/members/balances`, `maxLoanMultiplier`) leave out what came back
to them. `/api/main/members/balances` `savings` now also subtracts
withdrawals, as the loan limit already did. Its `savings`, `shares` and
`shareCount` never go below 0.

## Definitions (GATA should confirm the ones marked ⚑)

| Indicator | Definition |
|---|---|
| Savings recorded | Contributions (mandatory and voluntary) plus shares bought, dated in the period. `netSavings` subtracts withdrawals. |
| Savings growth | `totalBalance` (savings and shares balance at `asAt`) against `totalBalanceAtStart` (the balance just before `from`). |
| Individual contributions | `memberContribution{Min,Median,Max}` across the members who saved in the period. The full per-member list is the **member-savings** dataset. |
| Loans disbursed / borrowers | Loans **issued** in the period that are not cancelled; `borrowers` counts distinct members. |
| Outstanding balance | Total due (principal plus the flat interest fixed at issue) less repaid, at `asAt`. Older loans with no `totalDue` owe their principal. |
| Repayment rate ⚑ | Covers **loans that fell due within the period** (due date between `from` and `asAt`): repaid by `asAt` (capped at the total due) ÷ their total due. `repaymentRateAllTime` uses every loan that has fallen due by `asAt`. Loans not yet due are left out. |
| Overdue | Due date before `asAt` and a balance still owed at `asAt`. `par30*` covers loans more than 30 days late. |
| Default ⚑ | Owes money and is more than **90 days** past due (`loanDefaultDays`, a const in `monitoring.go`), or is marked `defaulted`. `defaultRate` = defaulted balance ÷ outstanding. |
| Social fund | `contributions` (social_fund) and `disbursements` (social_fund_payout) dated in the period; `availableBalance` at `asAt`, never below 0 per group (the dashboard's `socialFund` is clamped the same way). |
| Active member | `status` Active (or empty) and joined by that date. An inactive member counts as active until the time of their last "deactivate" audit entry (else the member's `updatedAt`). There is no full status history, so a member who was deactivated and then reactivated counts as active throughout. |
| Retention | Members active at both the start and the end ÷ members active at the start. |
| New members | `joinedAt` (else `createdAt`) in the period. |
| Attendance | (present + late) ÷ attendance rows, for meetings in the period that are not cancelled. Gender and age breakdowns use each member's own rows. |
| Digital adoption ⚑ | `groupsUsingApp`: groups that recorded a meeting (in progress or completed) or a non-reversed transaction in the **30 days** up to `asAt`; `adoptionRate` = ÷ groups in scope. "% recorded digitally": every HelaBox transaction is digital by definition, so this is `digitalRecordingRate` = completed meetings in the period that have attendance **and** at least one money record ÷ completed meetings. A transaction belongs to a meeting by its `meetingId`, else to the group's only non-cancelled meeting that EAT day (the same rule as the meeting-collections report). |
| Interest earned ⚑ | Each repayment is split **pro rata**: the interest part = amount × interestAmount ÷ totalDue of its loan (matched by loan number in the group). Legacy loans earn 0. `interestCharged` = interest on loans issued in the period. |
| Operational expenses / net income | `expense` transactions; `netIncome` = interest earned + fines collected − expenses. |
| Savings shared out / returns to members | `share_out` transactions in the period: `savingsSharedOut` = amounts paid, `savingsReturned` = the members' own savings and shares returned, `returnsToMembers` = the profit part. `returnOnSavingsPct` = profit ÷ savings returned. |
| Accuracy of records ⚑ | `groupsWithDrift`: groups whose stored running totals differ from the computed ones (the same check as `/api/admin/totals-drift`). `recordsAccuracyPct` = groups without drift ÷ groups. Also `reversals` / `reversalRate` / `reversedAmount` (transactions dated in the period), `overpaidLoans`, and `transactionsAtMeetingPct`. |
| Cash reconciliation ⚑ | `expectedCashOnHand`: the physical cash box computed at `asAt` from every non-reversed transaction (types money comes in on minus everything else), plus government-loan money received, minus government repayments. The amount available to share out is less than this: `expectedCashOnHand − max(0, govReceived − govRepaid) − socialFund`, which equals the `groupCash` formula above. HelaBox does **not** capture a physical cash count, so there is no counted-vs-expected difference yet. GATA must decide whether to add that. |
| Transparency | `auditEntries` / `auditByAction` (audit log in the period) appear **only for callers with `audit.view`** (super admin). For everyone else they are 0 / empty and `auditHidden: true`. `smsSent`, `smsFailed`, `smsDeliveryRate` and `membersNotified` (active members who got at least one SMS in the period) count only groups where the caller has `sms.view`. If any group in scope is left out, `smsHidden: true`. A partner user, for example, has no `sms.view`, so gets 0 with `smsHidden: true`. |
| Group financial assets | `groupAssets` = cash on hand + loan portfolio (gross; government loans owed are not subtracted). |
| Growth series | `growth.monthly`: one point per EAT calendar month from `from` to `asAt` (at most 60). Each point holds the balances at the end of the month (or at `asAt` for the last one) and that month's flows. The first month's flows start at `from`, so the monthly points add up to the period totals. |
| Saving regularly ⚑ | Among active members whose group held at least one completed meeting in the period (counting meetings on or after the member joined): a member saved or bought shares at **≥ 75 %** of those meetings (`regularSaverPct`). |
| Accessing loans | `borrowers` (issued in the period) ÷ active members; `membersWithOpenLoans` = active members owing at `asAt`. |
| Average savings / borrowing per member | Savings and shares balance (or loan balance) per active member, at the start and at the end; the `…ChangePct` fields give the % change. |
| Economic benefits | IGA loans, borrowers and amount for loans issued in the period, plus breakdowns by purpose and enterprise type (top 25 by amount). |

## Report datasets (Reports screen and exports, category "Monitoring")

All of these use the same computation, so with no `from` they also cover the
last 12 months. Each has Swahili column and value labels.

| key | rows |
|---|---|
| `monitoring-summary` | One row per group: members, female/male/youth, new members, retention %, attendance %, savings, balances, loans, repayment rate %, overdue/defaulted, interest earned, expenses, shared out, returns, regular savers %, meetings fully recorded %, using the app. The TOTAL row recomputes the rates. |
| `members-by-age` | One row per age band: active, female, male, new, retention %, attendance %, savers, regular savers %, borrowers, IGA borrowers. |
| `member-savings` | One row per member: gender, age band, savings and shares in the period, balance, loans outstanding, meetings saved / held, regular saver. |
| `loans-by-purpose` | Purpose, income-generating (yes/no), loans, borrowers, principal. |
| `loans-by-enterprise` | Enterprise type, loans, borrowers, principal. |
| `monthly-growth` | The `growth.monthly` series. |
| `social-fund-payouts` | Payout transactions (Financial category). |
| `share-outs` | Share-out transactions, with cycle, savings returned and return (Financial category). |

The existing `loans` dataset gains **Purpose** and **Enterprise Type**
columns, and `members` gains **Date of Birth** and **Age Band**.

New enum values in `valueSw` / `valueEn` (sent in the `enums` field of
`/api/admin/reports`) are the purposes, `unspecified`, `under 15`, `unknown`,
`yes`/`no`, `social_fund_payout` and `share_out`. For the dashboard's activity
feed, `recentActivity[].type` can now also be `social_fund_payout` or
`share_out`, so the web needs `dash2.act.social_fund_payout` and
`dash2.act.share_out`.

## Example response

This is real output from the isolated end-to-end run: one group with 4
members, 1 closed meeting, 3 loans (all repaid), 1 social fund payout and a
share-out (C got 2,160: 2,000 of her own savings plus 160 of profit). It
was requested as `GET /api/admin/monitoring?groupId=…` with no dates, so the
period defaulted to 12 months. `growth.monthly` is cut to its first and last
point here; the real response has 12.

```json
{
  "access": {
    "accessingLoansPct": 75,
    "activeMembers": 4,
    "avgBorrowingChangePct": 0,
    "avgBorrowingPerMemberEnd": 0,
    "avgBorrowingPerMemberStart": 0,
    "avgLoanPerBorrower": 11000,
    "avgSavingsChangePct": 0,
    "avgSavingsPerMemberEnd": 8250,
    "avgSavingsPerMemberStart": 0,
    "borrowers": 3,
    "byAgeBand": [
      {
        "activeMembers": 0,
        "ageBand": "under 15",
        "borrowers": 0,
        "borrowersPct": 0,
        "eligibleSavers": 0,
        "regularSavers": 0,
        "regularSaversPct": 0
      },
      {
        "activeMembers": 1,
        "ageBand": "15-24",
        "borrowers": 1,
        "borrowersPct": 100,
        "eligibleSavers": 1,
        "regularSavers": 1,
        "regularSaversPct": 100
      },
      {
        "activeMembers": 1,
        "ageBand": "25-35",
        "borrowers": 1,
        "borrowersPct": 100,
        "eligibleSavers": 1,
        "regularSavers": 1,
        "regularSaversPct": 100
      },
      {
        "activeMembers": 0,
        "ageBand": "36-59",
        "borrowers": 0,
        "borrowersPct": 0,
        "eligibleSavers": 0,
        "regularSavers": 0,
        "regularSaversPct": 0
      },
      {
        "activeMembers": 1,
        "ageBand": "60+",
        "borrowers": 1,
        "borrowersPct": 100,
        "eligibleSavers": 1,
        "regularSavers": 1,
        "regularSaversPct": 100
      },
      {
        "activeMembers": 1,
        "ageBand": "unknown",
        "borrowers": 0,
        "borrowersPct": 0,
        "eligibleSavers": 1,
        "regularSavers": 1,
        "regularSaversPct": 100
      }
    ],
    "byGender": [
      {
        "activeMembers": 2,
        "borrowers": 2,
        "borrowersPct": 100,
        "eligibleSavers": 2,
        "gender": "Female",
        "regularSavers": 2,
        "regularSaversPct": 100
      },
      {
        "activeMembers": 2,
        "borrowers": 1,
        "borrowersPct": 50,
        "eligibleSavers": 2,
        "gender": "Male",
        "regularSavers": 2,
        "regularSaversPct": 100
      },
      {
        "activeMembers": 0,
        "borrowers": 0,
        "borrowersPct": 0,
        "eligibleSavers": 0,
        "gender": "unknown",
        "regularSavers": 0,
        "regularSaversPct": 0
      }
    ],
    "eligibleMembers": 4,
    "membersWithOpenLoans": 0,
    "membersWithOpenLoansPct": 0,
    "regularSaverThresholdPct": 75,
    "regularSavers": 4,
    "regularSaversPct": 100,
    "savers": 4,
    "saversPct": 100
  },
  "accountability": {
    "auditByAction": [
      {
        "action": "create",
        "count": 34
      },
      {
        "action": "update",
        "count": 8
      },
      {
        "action": "assign",
        "count": 1
      },
      {
        "action": "reverse",
        "count": 5
      },
      {
        "action": "delete",
        "count": 1
      },
      {
        "action": "deactivate",
        "count": 1
      },
      {
        "action": "reactivate",
        "count": 1
      }
    ],
    "auditEntries": 51,
    "auditHidden": false,
    "cashOnHandAtStart": 0,
    "driftAmount": 0,
    "expectedCashOnHand": 126640,
    "groupsChecked": 1,
    "groupsWithDrift": 0,
    "membersNotified": 4,
    "membersNotifiedPct": 100,
    "overpaidLoans": 0,
    "recordsAccuracyPct": 100,
    "reversalRate": 21.7,
    "reversals": 5,
    "reversedAmount": 17660,
    "smsDeliveryRate": 100,
    "smsFailed": 0,
    "smsHidden": false,
    "smsSent": 27,
    "transactions": 23,
    "transactionsAtMeeting": 18,
    "transactionsAtMeetingPct": 100
  },
  "definitions": {
    "ageBands": [
      "under 15",
      "15-24",
      "25-35",
      "36-59",
      "60+",
      "unknown"
    ],
    "appActiveDays": 30,
    "defaultAfterDays": 90,
    "incomeGeneratingPurposes": [
      "business",
      "agriculture",
      "livestock"
    ],
    "loanPurposes": [
      "business",
      "agriculture",
      "livestock",
      "education",
      "health",
      "household",
      "housing",
      "emergency",
      "other"
    ],
    "parDays": 30,
    "regularSaverThresholdPct": 75,
    "youthAges": "15-35"
  },
  "digitalAdoption": {
    "activeDays": 30,
    "activeGroups": 1,
    "adoptionRate": 100,
    "digitalRecordingRate": 100,
    "groups": 1,
    "groupsUsingApp": 1,
    "groupsUsingAppInPeriod": 1,
    "meetingsCompleted": 1,
    "meetingsFullyRecorded": 1,
    "recordingUsers": 2,
    "transactionsRecorded": 18
  },
  "economicBenefits": {
    "avgReturnPerMember": 160,
    "byEnterpriseType": [
      {
        "amount": 20000,
        "borrowers": 1,
        "enterpriseType": "Mama lishe",
        "loans": 1
      },
      {
        "amount": 10000,
        "borrowers": 1,
        "enterpriseType": "Poultry",
        "loans": 1
      }
    ],
    "byPurpose": [
      {
        "amount": 20000,
        "borrowers": 1,
        "incomeGenerating": true,
        "loans": 1,
        "purpose": "business"
      },
      {
        "amount": 10000,
        "borrowers": 1,
        "incomeGenerating": true,
        "loans": 1,
        "purpose": "agriculture"
      },
      {
        "amount": 3000,
        "borrowers": 1,
        "incomeGenerating": false,
        "loans": 1,
        "purpose": "unspecified"
      }
    ],
    "igaAmount": 30000,
    "igaBorrowers": 2,
    "igaBorrowersByAgeBand": [
      {
        "ageBand": "under 15",
        "igaBorrowers": 0
      },
      {
        "ageBand": "15-24",
        "igaBorrowers": 0
      },
      {
        "ageBand": "25-35",
        "igaBorrowers": 1
      },
      {
        "ageBand": "36-59",
        "igaBorrowers": 0
      },
      {
        "ageBand": "60+",
        "igaBorrowers": 1
      },
      {
        "ageBand": "unknown",
        "igaBorrowers": 0
      }
    ],
    "igaBorrowersByGender": [
      {
        "gender": "Female",
        "igaBorrowers": 1
      },
      {
        "gender": "Male",
        "igaBorrowers": 1
      },
      {
        "gender": "unknown",
        "igaBorrowers": 0
      }
    ],
    "igaBorrowersPct": 50,
    "igaLoans": 2,
    "igaLoansPct": 66.7,
    "interestEarned": 3300,
    "loansWithPurpose": 2,
    "returnOnSavingsPct": 8,
    "returnsToMembers": 160,
    "unspecifiedLoans": 1
  },
  "financialPerformance": {
    "avgReturnPerMember": 160,
    "finesCollected": 1000,
    "interestCharged": 3300,
    "interestEarned": 3300,
    "membersPaidOut": 1,
    "netIncome": 2800,
    "operationalExpenses": 1500,
    "returnOnSavingsPct": 8,
    "returnsToMembers": 160,
    "savingsReturned": 2000,
    "savingsSharedOut": 2160,
    "shareOutPayments": 1
  },
  "growth": {
    "assetsEnd": 126640,
    "assetsGrowthPct": 0,
    "assetsStart": 0,
    "loanPortfolioEnd": 0,
    "loanPortfolioGrowthPct": 0,
    "loanPortfolioStart": 0,
    "membersEnd": 4,
    "membersGrowthPct": 0,
    "membersStart": 0,
    "monthly": [
      {
        "activeMembers": 0,
        "cashOnHand": 0,
        "groupAssets": 0,
        "loanPortfolio": 0,
        "loansDisbursed": 0,
        "loansIssued": 0,
        "month": "2025-11",
        "newMembers": 0,
        "savingsBalance": 0,
        "savingsRecorded": 0,
        "sharesBalance": 0,
        "socialFundBalance": 0,
        "totalSavings": 0
      },
      {
        "activeMembers": 4,
        "cashOnHand": 126640,
        "groupAssets": 126640,
        "loanPortfolio": 0,
        "loansDisbursed": 33000,
        "loansIssued": 3,
        "month": "2026-10",
        "newMembers": 4,
        "savingsBalance": 30000,
        "savingsRecorded": 35000,
        "sharesBalance": 3000,
        "socialFundBalance": 1000,
        "totalSavings": 33000
      }
    ],
    "savingsEnd": 33000,
    "savingsGrowthPct": 0,
    "savingsStart": 0
  },
  "loans": {
    "activeLoans": 0,
    "amountDisbursed": 33000,
    "averageLoanSize": 11000,
    "borrowers": 3,
    "borrowersWithOpenLoans": 0,
    "byAgeBand": [
      {
        "ageBand": "under 15",
        "borrowers": 0,
        "loanAmount": 0,
        "loans": 0
      },
      {
        "ageBand": "15-24",
        "borrowers": 1,
        "loanAmount": 3000,
        "loans": 1
      },
      {
        "ageBand": "25-35",
        "borrowers": 1,
        "loanAmount": 20000,
        "loans": 1
      },
      {
        "ageBand": "36-59",
        "borrowers": 0,
        "loanAmount": 0,
        "loans": 0
      },
      {
        "ageBand": "60+",
        "borrowers": 1,
        "loanAmount": 10000,
        "loans": 1
      },
      {
        "ageBand": "unknown",
        "borrowers": 0,
        "loanAmount": 0,
        "loans": 0
      }
    ],
    "byGender": [
      {
        "borrowers": 2,
        "gender": "Female",
        "loanAmount": 23000,
        "loans": 2
      },
      {
        "borrowers": 1,
        "gender": "Male",
        "loanAmount": 10000,
        "loans": 1
      },
      {
        "borrowers": 0,
        "gender": "unknown",
        "loanAmount": 0,
        "loans": 0
      }
    ],
    "interestCharged": 3300,
    "loansIssued": 3,
    "outstandingAtStart": 0,
    "outstandingBalance": 0
  },
  "membership": {
    "absent": 1,
    "activeMembers": 4,
    "attendanceRate": 50,
    "attendanceRows": 4,
    "byAgeBand": [
      {
        "activeMembers": 0,
        "ageBand": "under 15",
        "attendanceRate": 0,
        "membersAtStart": 0,
        "newMembers": 0,
        "retained": 0,
        "retentionRate": 0
      },
      {
        "activeMembers": 1,
        "ageBand": "15-24",
        "attendanceRate": 0,
        "membersAtStart": 0,
        "newMembers": 1,
        "retained": 0,
        "retentionRate": 0
      },
      {
        "activeMembers": 1,
        "ageBand": "25-35",
        "attendanceRate": 100,
        "membersAtStart": 0,
        "newMembers": 1,
        "retained": 0,
        "retentionRate": 0
      },
      {
        "activeMembers": 0,
        "ageBand": "36-59",
        "attendanceRate": 0,
        "membersAtStart": 0,
        "newMembers": 0,
        "retained": 0,
        "retentionRate": 0
      },
      {
        "activeMembers": 1,
        "ageBand": "60+",
        "attendanceRate": 100,
        "membersAtStart": 0,
        "newMembers": 1,
        "retained": 0,
        "retentionRate": 0
      },
      {
        "activeMembers": 1,
        "ageBand": "unknown",
        "attendanceRate": 0,
        "membersAtStart": 0,
        "newMembers": 1,
        "retained": 0,
        "retentionRate": 0
      }
    ],
    "byGender": [
      {
        "activeMembers": 2,
        "attendanceRate": 50,
        "gender": "Female",
        "membersAtStart": 0,
        "newMembers": 2,
        "retained": 0,
        "retentionRate": 0
      },
      {
        "activeMembers": 2,
        "attendanceRate": 50,
        "gender": "Male",
        "membersAtStart": 0,
        "newMembers": 2,
        "retained": 0,
        "retentionRate": 0
      },
      {
        "activeMembers": 0,
        "attendanceRate": 0,
        "gender": "unknown",
        "membersAtStart": 0,
        "newMembers": 0,
        "retained": 0,
        "retentionRate": 0
      }
    ],
    "byGenderAndAge": [
      {
        "activeMembers": 1,
        "ageBand": "15-24",
        "gender": "Female",
        "newMembers": 1
      },
      {
        "activeMembers": 1,
        "ageBand": "25-35",
        "gender": "Female",
        "newMembers": 1
      },
      {
        "activeMembers": 1,
        "ageBand": "60+",
        "gender": "Male",
        "newMembers": 1
      },
      {
        "activeMembers": 1,
        "ageBand": "unknown",
        "gender": "Male",
        "newMembers": 1
      }
    ],
    "dateOfBirthCoveragePct": 75,
    "excused": 1,
    "exits": 0,
    "femaleMembers": 2,
    "femalePct": 50,
    "late": 1,
    "maleMembers": 2,
    "meetingsHeld": 1,
    "membersAtStart": 0,
    "membersWithDateOfBirth": 3,
    "newMembers": 4,
    "present": 1,
    "retained": 0,
    "retentionRate": 0,
    "totalMembers": 4,
    "youthMembers": 2,
    "youthPct": 50
  },
  "period": {
    "asAt": "2026-10-23",
    "defaultedFrom": true,
    "from": "2025-11-01",
    "months": 12,
    "to": "2026-10-23"
  },
  "repayments": {
    "amountFallenDue": 0,
    "amountRepaidOnFallenDue": 0,
    "defaultAfterDays": 90,
    "defaultRate": 0,
    "defaultedAmount": 0,
    "defaultedLoans": 0,
    "fullyRepaidLoans": 0,
    "loansFallenDue": 0,
    "loansFallenDueAllTime": 0,
    "overdueAmount": 0,
    "overdueBorrowers": 0,
    "overdueLoans": 0,
    "par30Amount": 0,
    "par30Loans": 0,
    "par30Rate": 0,
    "repaymentRate": 0,
    "repaymentRateAllTime": 0,
    "repaymentsCollected": 36300
  },
  "savings": {
    "avgBalancePerMember": 8250,
    "avgPerContributor": 8750,
    "byAgeBand": [
      {
        "ageBand": "under 15",
        "savers": 0,
        "savingsBalance": 0,
        "savingsRecorded": 0
      },
      {
        "ageBand": "15-24",
        "savers": 1,
        "savingsBalance": 0,
        "savingsRecorded": 2000
      },
      {
        "ageBand": "25-35",
        "savers": 1,
        "savingsBalance": 8000,
        "savingsRecorded": 8000
      },
      {
        "ageBand": "36-59",
        "savers": 0,
        "savingsBalance": 0,
        "savingsRecorded": 0
      },
      {
        "ageBand": "60+",
        "savers": 1,
        "savingsBalance": 5000,
        "savingsRecorded": 5000
      },
      {
        "ageBand": "unknown",
        "savers": 1,
        "savingsBalance": 20000,
        "savingsRecorded": 20000
      }
    ],
    "byGender": [
      {
        "gender": "Female",
        "savers": 2,
        "savingsBalance": 8000,
        "savingsRecorded": 10000
      },
      {
        "gender": "Male",
        "savers": 2,
        "savingsBalance": 25000,
        "savingsRecorded": 25000
      },
      {
        "gender": "unknown",
        "savers": 0,
        "savingsBalance": 0,
        "savingsRecorded": 0
      }
    ],
    "contributingMembers": 4,
    "contributions": 32000,
    "growthAmount": 33000,
    "growthPct": 0,
    "mandatorySavings": 10000,
    "memberContributionMax": 20000,
    "memberContributionMedian": 6500,
    "memberContributionMin": 2000,
    "netSavings": 35000,
    "savingsBalance": 30000,
    "savingsRecorded": 35000,
    "sharesBalance": 3000,
    "sharesPurchased": 3000,
    "totalBalance": 33000,
    "totalBalanceAtStart": 0,
    "voluntarySavings": 22000,
    "withdrawals": 0
  },
  "scope": {
    "clusterId": "",
    "groupId": "6adb5e6451d2d5ae9fb94a17",
    "groups": 1,
    "label": "Umoja E2E 444458",
    "partnerId": ""
  },
  "socialFund": {
    "availableBalance": 1000,
    "balanceAtStart": 0,
    "beneficiaries": 1,
    "contributions": 4000,
    "contributors": 2,
    "disbursements": 3000,
    "payouts": 1
  }
}
```
