package main

import (
	"testing"
	"time"

	"github.com/robertkonga/yekonga-server-go/datatype"
)

// monFixture is a small programme for the GATA indicators: Umoja (g1) and
// Amani (g2) in scope, Zeta (gx) out. "Now" is testNow (2026-09-25 12:00
// EAT); the tests use the period 2026-07-01 .. 2026-09-30. Every expected
// number in TestMonitoringSections is worked out by hand in the comments.
func monFixture() map[string][]datatype.DataMap {
	tx := func(id, gid, mid, mtg, typ string, amt float64, at string, extra ...interface{}) datatype.DataMap {
		r := datatype.DataMap{"id": id, "groupId": gid, "memberId": mid, "meetingId": mtg, "type": typ, "amount": amt, "createdAt": at, "createdBy": "u1"}
		for i := 0; i+1 < len(extra); i += 2 {
			r[extra[i].(string)] = extra[i+1]
		}
		return r
	}
	return map[string][]datatype.DataMap{
		"Partner": {{"id": "p1", "name": "GATA"}},
		"Cluster": {{"id": "c1", "name": "Arusha", "partnerId": "p1"}},
		"Group": {
			{"id": "g1", "name": "Umoja", "clusterId": "c1", "status": "Active", "createdAt": "2026-01-01T00:00:00Z"},
			// Stored totals match what is computed: no drift for Amani.
			{"id": "g2", "name": "Amani", "clusterId": "c1", "status": "Active", "createdAt": "2026-01-01T00:00:00Z", "totalSavings": 3000.0},
			{"id": "gx", "name": "Zeta", "clusterId": "c1", "status": "Active"},
		},
		"Member": {
			{"id": "m1", "groupId": "g1", "firstName": "Asha", "gender": "Female", "dateOfBirth": "2000-05-10", "status": "Active", "joinedAt": "2026-01-10T07:00:00Z"},
			{"id": "m2", "groupId": "g1", "firstName": "John", "gender": "Male", "dateOfBirth": "1960-01-01", "status": "Active", "joinedAt": "2026-02-01T07:00:00Z"},
			{"id": "m3", "groupId": "g1", "firstName": "Neema", "gender": "Female", "dateOfBirth": "2008-03-01", "status": "Active", "joinedAt": "2026-08-01T07:00:00Z"},
			{"id": "m4", "groupId": "g1", "firstName": "Juma", "gender": "Male", "status": "Inactive", "joinedAt": "2026-03-01T07:00:00Z", "updatedAt": "2026-08-15T07:00:00Z"},
			{"id": "m5", "groupId": "g1", "firstName": "Rehema", "gender": "Female", "dateOfBirth": "1980-06-15", "status": "Inactive", "joinedAt": "2026-01-01T07:00:00Z"},
			{"id": "m6", "groupId": "g2", "firstName": "Mary", "gender": "Female", "dateOfBirth": "1990-01-01", "status": "Active", "joinedAt": "2026-01-01T07:00:00Z"},
			{"id": "m9", "groupId": "gx", "firstName": "Out", "gender": "Male", "status": "Active", "joinedAt": "2026-08-01T07:00:00Z"},
		},
		"AuditLog": {
			{"id": "au1", "groupId": "g1", "model": "Member", "recordId": "m4", "action": "deactivate", "createdAt": "2026-08-20T07:00:00Z"},
			{"id": "au2", "groupId": "g1", "model": "Member", "recordId": "m5", "action": "deactivate", "createdAt": "2026-06-01T07:00:00Z"},
			{"id": "au3", "groupId": "g1", "model": "Meeting", "recordId": "mt3", "action": "create", "createdAt": "2026-09-10T07:00:00Z"},
			{"id": "au9", "groupId": "gx", "model": "Meeting", "action": "create", "createdAt": "2026-09-10T07:00:00Z"},
		},
		"Meeting": {
			{"id": "mt0", "groupId": "g1", "meetingNumber": 1, "date": "2026-06-10", "status": "completed"},
			{"id": "mt1", "groupId": "g1", "meetingNumber": 2, "date": "2026-07-10", "status": "completed"},
			{"id": "mt2", "groupId": "g1", "meetingNumber": 3, "date": "2026-08-10", "status": "completed"},
			{"id": "mt3", "groupId": "g1", "meetingNumber": 4, "date": "2026-09-10", "status": "completed"},
			{"id": "mt4", "groupId": "g1", "meetingNumber": 5, "date": "2026-09-20", "status": "cancelled"},
			{"id": "mt5", "groupId": "g2", "meetingNumber": 1, "date": "2026-09-15", "status": "completed"},
			{"id": "mt9", "groupId": "gx", "meetingNumber": 1, "date": "2026-09-15", "status": "completed"},
		},
		"MeetingAttendance": {
			{"id": "a0", "groupId": "g1", "meetingId": "mt0", "memberId": "m1", "status": "present"},
			{"id": "a1", "groupId": "g1", "meetingId": "mt1", "memberId": "m1", "status": "present"},
			{"id": "a2", "groupId": "g1", "meetingId": "mt1", "memberId": "m2", "status": "present"},
			{"id": "a3", "groupId": "g1", "meetingId": "mt1", "memberId": "m4", "status": "absent"},
			{"id": "a4", "groupId": "g1", "meetingId": "mt2", "memberId": "m1", "status": "late"},
			{"id": "a5", "groupId": "g1", "meetingId": "mt2", "memberId": "m2", "status": "absent"},
			{"id": "a6", "groupId": "g1", "meetingId": "mt2", "memberId": "m3", "status": "present"},
			{"id": "a7", "groupId": "g1", "meetingId": "mt2", "memberId": "m4", "status": "excused"},
			{"id": "a8", "groupId": "g1", "meetingId": "mt3", "memberId": "m1", "status": "present"},
			{"id": "a9", "groupId": "g1", "meetingId": "mt3", "memberId": "m2", "status": "present"},
			{"id": "a10", "groupId": "g1", "meetingId": "mt3", "memberId": "m3", "status": "present"},
			{"id": "a11", "groupId": "g1", "meetingId": "mt4", "memberId": "m1", "status": "present"}, // cancelled meeting
			{"id": "a12", "meetingId": "mt5", "memberId": "m6", "status": "present"},                  // group from its meeting
			{"id": "a19", "groupId": "gx", "meetingId": "mt9", "memberId": "m9", "status": "present"},
		},
		"Transaction": {
			tx("t0", "g1", "m1", "mt0", "contribution", 5000, "2026-06-10T07:00:00Z"),
			tx("t23", "g1", "m5", "", "contribution", 4000, "2026-03-01T07:00:00Z"),
			tx("t18", "g1", "m1", "mt0", "loan_disbursement", 10000, "2026-06-10T08:00:00Z", "reference", "LN-0000"),
			tx("t1", "g1", "m1", "mt1", "contribution", 5000, "2026-07-10T07:00:00Z"),
			tx("t2", "g1", "m2", "mt1", "contribution", 5000, "2026-07-10T07:01:00Z"),
			tx("t3", "g1", "m1", "mt1", "share", 10000, "2026-07-10T07:02:00Z"),
			tx("t10", "g1", "m1", "mt1", "social_fund", 2000, "2026-07-10T07:03:00Z"),
			tx("t11", "g1", "m2", "mt1", "social_fund", 2000, "2026-07-10T07:04:00Z"),
			tx("t13", "g1", "m2", "mt1", "loan_disbursement", 20000, "2026-07-10T08:00:00Z", "reference", "LN-0001"),
			tx("t19", "g1", "m1", "mt1", "loan_repayment", 5500, "2026-07-10T09:00:00Z", "reference", "LN-0000"),
			tx("t4", "g1", "m1", "mt2", "contribution", 5000, "2026-08-10T07:00:00Z"),
			tx("t5", "g1", "m3", "mt2", "contribution", 5000, "2026-08-10T07:01:00Z", "savingsType", "voluntary"),
			tx("t6", "g1", "m2", "mt2", "contribution", 5000, "2026-08-10T07:02:00Z", "reversed", true),
			tx("t15", "g1", "m4", "mt2", "fine", 1000, "2026-08-10T07:03:00Z"),
			tx("t21", "g1", "m1", "mt2", "loan_disbursement", 7000, "2026-08-10T08:00:00Z", "reference", "LN-0004", "reversed", true),
			tx("t24", "g1", "m1", "", "loan_disbursement", 3000, "2026-08-15T08:00:00Z", "reference", "LN-0005"),
			tx("t7", "g1", "m1", "mt3", "contribution", 5000, "2026-09-10T07:00:00Z"),
			tx("t8", "g1", "m2", "mt3", "contribution", 5000, "2026-09-10T07:01:00Z"),
			tx("t9", "g1", "m3", "mt3", "share", 5000, "2026-09-10T07:02:00Z"),
			tx("t14", "g1", "m2", "mt3", "loan_repayment", 11000, "2026-09-10T08:00:00Z", "reference", "LN-0001"),
			tx("t16", "g1", "", "mt3", "expense", 1500, "2026-09-10T09:00:00Z"),
			tx("t20", "g1", "m3", "mt3", "loan_disbursement", 5000, "2026-09-10T09:30:00Z", "reference", "LN-0003"),
			tx("t17", "g1", "m1", "", "withdrawal", 2000, "2026-09-11T07:00:00Z"),
			tx("t12", "g1", "m2", "", "social_fund_payout", 3000, "2026-09-12T07:00:00Z"),
			tx("t22", "g1", "m5", "", "share_out", 6000, "2026-09-12T08:00:00Z", "savingsReturned", 4000.0, "sharesReturned", 0.0, "profit", 2000.0, "cycle", 1),
			tx("t32", "g2", "m6", "", "loan_disbursement", 6000, "2026-07-01T07:00:00Z", "reference", "LN-0001"),
			tx("t31", "g2", "m6", "", "loan_repayment", 6600, "2026-08-30T07:00:00Z", "reference", "LN-0001"),
			tx("t30", "g2", "m6", "", "contribution", 3000, "2026-09-16T07:00:00Z"), // not at mt5 (another day)
			tx("t99", "gx", "m9", "mt9", "contribution", 999999, "2026-09-15T07:00:00Z"),
		},
		"Loan": {
			{"id": "l0", "groupId": "g1", "memberId": "m1", "loanNumber": "LN-0000", "amount": 10000.0, "interestAmount": 1000.0, "totalDue": 11000.0, "amountRepaid": 5500.0, "status": "active", "issuedDate": "2026-06-10T08:00:00Z", "dueDate": "2026-07-10T08:00:00Z"},
			{"id": "l1", "groupId": "g1", "memberId": "m2", "loanNumber": "LN-0001", "amount": 20000.0, "interestAmount": 2000.0, "totalDue": 22000.0, "amountRepaid": 11000.0, "status": "active", "issuedDate": "2026-07-10T08:00:00Z", "dueDate": "2026-10-10T08:00:00Z", "purpose": "business", "enterpriseType": "Mama lishe"},
			// legacy loan: no totalDue, no repayment transactions, long overdue → default
			{"id": "l2", "groupId": "g1", "memberId": "m4", "loanNumber": "LN-0002", "amount": 8000.0, "amountRepaid": 0.0, "status": "active", "issuedDate": "2026-03-01T08:00:00Z", "dueDate": "2026-05-01T08:00:00Z"},
			{"id": "l3", "groupId": "g1", "memberId": "m3", "loanNumber": "LN-0003", "amount": 5000.0, "interestAmount": 500.0, "totalDue": 5500.0, "amountRepaid": 0.0, "status": "active", "issuedDate": "2026-09-10T09:30:00Z", "dueDate": "2026-12-10T08:00:00Z", "purpose": "education", "enterpriseType": "Poultry "},
			{"id": "l4", "groupId": "g1", "memberId": "m1", "loanNumber": "LN-0004", "amount": 7000.0, "totalDue": 7700.0, "amountRepaid": 0.0, "status": "cancelled", "issuedDate": "2026-08-10T08:00:00Z", "dueDate": "2026-08-20T08:00:00Z", "purpose": "business"},
			{"id": "l6", "groupId": "g1", "memberId": "m1", "loanNumber": "LN-0005", "amount": 3000.0, "interestAmount": 0.0, "totalDue": 3000.0, "amountRepaid": 0.0, "status": "active", "issuedDate": "2026-08-15T08:00:00Z", "dueDate": "2026-11-15T08:00:00Z"},
			{"id": "l5", "groupId": "g2", "memberId": "m6", "loanNumber": "LN-0001", "amount": 6000.0, "interestAmount": 600.0, "totalDue": 6600.0, "amountRepaid": 6600.0, "status": "repaid", "issuedDate": "2026-07-01T07:00:00Z", "dueDate": "2026-09-01T07:00:00Z", "purpose": "livestock", "enterpriseType": "poultry"},
			{"id": "l9", "groupId": "gx", "memberId": "m9", "loanNumber": "LN-0001", "amount": 500000.0, "status": "active", "issuedDate": "2026-08-01T07:00:00Z", "purpose": "business"},
		},
		"GovernmentLoan": {
			{"id": "gl1", "groupId": "g2", "lender": "Halmashauri", "amount": 100000.0, "interestRate": 0.0, "amountRepaid": 20000.0, "receivedDate": "2026-08-01T07:00:00Z"},
		},
		"GovernmentLoanRepayment": {
			{"id": "gr1", "groupId": "g2", "governmentLoanId": "gl1", "amount": 20000.0, "date": "2026-09-01T07:00:00Z"},
		},
		"SmsLog": {
			{"id": "s1", "groupId": "g1", "memberId": "m1", "status": "sent", "sentAt": "2026-09-10T08:00:00Z"},
			{"id": "s2", "groupId": "g1", "memberId": "m2", "status": "failed", "sentAt": "2026-09-10T08:00:00Z"},
			{"id": "s3", "groupId": "g2", "memberId": "m6", "status": "sent", "sentAt": "2026-09-15T08:00:00Z"},
			{"id": "s4", "phone": "0755000009", "messageType": "login_otp", "status": "sent", "sentAt": "2026-09-15T08:00:00Z"},
			{"id": "s5", "groupId": "g1", "memberId": "m4", "status": "sent", "sentAt": "2026-08-10T08:00:00Z"},
		},
	}
}

var monScope = map[string]bool{"g1": true, "g2": true}

func monInput(fix map[string][]datatype.DataMap) monitorData {
	return monitorData{
		kpiData: kpiData{Groups: fix["Group"], Clusters: fix["Cluster"], Members: fix["Member"], Loans: fix["Loan"],
			GovLoans: fix["GovernmentLoan"], Meetings: fix["Meeting"], Attendance: fix["MeetingAttendance"], Transactions: fix["Transaction"]},
		AuditLogs: fix["AuditLog"], SmsLogs: fix["SmsLog"], GovRepayments: fix["GovernmentLoanRepayment"],
	}
}

func monPeriod(t *testing.T) (time.Time, time.Time) {
	t.Helper()
	f, to, err := reportRange("2026-07-01", "2026-09-30")
	if err != nil {
		t.Fatal(err)
	}
	return f, to
}

func checkSection(t *testing.T, out datatype.DataMap, section string, want map[string]float64) {
	t.Helper()
	s, ok := out[section].(datatype.DataMap)
	if !ok {
		t.Fatalf("section %q missing", section)
	}
	for k, v := range want {
		if _, ok := s[k]; !ok {
			t.Errorf("%s.%s missing", section, k)
			continue
		}
		approx(t, section+"."+k, s[k], v)
	}
}

func rowBy(list interface{}, key, val string) datatype.DataMap {
	for _, r := range list.([]datatype.DataMap) {
		if r[key] == val {
			return r
		}
	}
	return nil
}

func TestMonitoringSections(t *testing.T) {
	f, to := monPeriod(t)
	out := computeMonitoring(monInput(monFixture()), monScope, testNow, f, to)

	// Savings: contributions in the period t1,t2,t4,t5,t7,t8,t30 = 33,000
	// (t6 reversed; t5 voluntary), shares t3+t9 = 15,000. Balances now:
	// Umoja savings 39,000 − 2,000 withdrawn − 4,000 returned at the
	// share-out = 33,000, shares 15,000; Amani 3,000. At the start: t0 + t23.
	checkSection(t, out, "savings", map[string]float64{
		"savingsRecorded": 48000, "contributions": 33000, "mandatorySavings": 28000, "voluntarySavings": 5000,
		"sharesPurchased": 15000, "withdrawals": 2000, "netSavings": 46000,
		"savingsBalance": 36000, "sharesBalance": 15000, "totalBalance": 51000, "totalBalanceAtStart": 9000,
		"growthAmount": 42000, "growthPct": 466.7, "contributingMembers": 4, "avgPerContributor": 12000,
		"avgBalancePerMember": 12750, "memberContributionMin": 3000, "memberContributionMedian": 10000, "memberContributionMax": 25000,
	})
	// Loans issued in the period (l4 cancelled, l0 before): l1 20,000, l3
	// 5,000, l6 3,000, l5 6,000. Outstanding now: l0 5,500 + l1 11,000 + l2
	// 8,000 (legacy, principal) + l3 5,500 + l6 3,000. At the start: l0
	// 11,000 (its repayment came later) + l2 8,000.
	checkSection(t, out, "loans", map[string]float64{
		"loansIssued": 4, "amountDisbursed": 34000, "interestCharged": 3100, "borrowers": 4, "averageLoanSize": 8500,
		"outstandingBalance": 33000, "outstandingAtStart": 19000, "activeLoans": 5, "borrowersWithOpenLoans": 4,
	})
	// Fallen due in the period: l0 (11,000 due, 5,500 repaid) and l5 (6,600,
	// all repaid) → 12,100 / 17,600 = 68.75 %. All time adds l2 (8,000, 0).
	// Overdue: l0 (77 days) and l2 (147 days, a default).
	checkSection(t, out, "repayments", map[string]float64{
		"repaymentsCollected": 23100, "loansFallenDue": 2, "amountFallenDue": 17600, "amountRepaidOnFallenDue": 12100,
		"repaymentRate": 68.8, "loansFallenDueAllTime": 3, "repaymentRateAllTime": 47.3, "fullyRepaidLoans": 1,
		"overdueLoans": 2, "overdueAmount": 13500, "overdueBorrowers": 2, "par30Loans": 2, "par30Amount": 13500, "par30Rate": 40.9,
		"defaultedLoans": 1, "defaultedAmount": 8000, "defaultRate": 24.2, "defaultAfterDays": 90,
	})
	checkSection(t, out, "socialFund", map[string]float64{
		"contributions": 4000, "contributors": 2, "disbursements": 3000, "payouts": 1, "beneficiaries": 1,
		"availableBalance": 1000, "balanceAtStart": 0,
	})
	// Active at the start: m1, m2, m4 (left 20 Aug), m6; at the end: m1, m2,
	// m3 (new), m6. Retained 3 of 4. Attendance: 11 rows in the period
	// (mt0 before it, mt4 cancelled), 7 present + 1 late.
	checkSection(t, out, "membership", map[string]float64{
		"activeMembers": 4, "totalMembers": 6, "newMembers": 1, "membersAtStart": 4, "retained": 3, "exits": 1,
		"retentionRate": 75, "femaleMembers": 3, "maleMembers": 1, "femalePct": 75, "youthMembers": 2, "youthPct": 50,
		"membersWithDateOfBirth": 4, "dateOfBirthCoveragePct": 100, "meetingsHeld": 4, "attendanceRows": 11,
		"present": 7, "late": 1, "absent": 2, "excused": 1, "attendanceRate": 72.7,
	})
	// mt5 has attendance but no money (t30 was another day) → 3 of 4.
	checkSection(t, out, "digitalAdoption", map[string]float64{
		"groups": 2, "groupsUsingApp": 2, "adoptionRate": 100, "groupsUsingAppInPeriod": 2, "transactionsRecorded": 23,
		"recordingUsers": 1, "meetingsCompleted": 4, "meetingsFullyRecorded": 3, "digitalRecordingRate": 75,
	})
	// Interest earned pro rata: t14 11,000 × 2,000/22,000 + t19 5,500 ×
	// 1,000/11,000 + t31 6,600 × 600/6,600 = 1,000 + 500 + 600.
	checkSection(t, out, "financialPerformance", map[string]float64{
		"interestEarned": 2100, "interestCharged": 3100, "finesCollected": 1000, "operationalExpenses": 1500, "netIncome": 1600,
		"savingsSharedOut": 6000, "savingsReturned": 4000, "returnsToMembers": 2000, "returnOnSavingsPct": 50,
		"membersPaidOut": 1, "shareOutPayments": 1, "avgReturnPerMember": 2000,
	})
	// 25 transactions dated in the period, 2 reversed (5,000 + 7,000); 16 of
	// the other 23 belong to a meeting. Umoja has no stored totals → drift.
	// Cash now: Umoja in 75,500 − out 50,500; Amani 9,600 − 6,000 + the
	// government loan 100,000 − 20,000 repaid.
	checkSection(t, out, "accountability", map[string]float64{
		"transactions": 25, "reversals": 2, "reversalRate": 8, "reversedAmount": 12000,
		"transactionsAtMeeting": 16, "transactionsAtMeetingPct": 69.6, "groupsChecked": 2, "groupsWithDrift": 1,
		"recordsAccuracyPct": 50, "driftAmount": 84500, "overpaidLoans": 0, "expectedCashOnHand": 108600, "cashOnHandAtStart": -1000,
		"auditEntries": 2, "smsSent": 3, "smsFailed": 1, "smsDeliveryRate": 75, "membersNotified": 2, "membersNotifiedPct": 50,
	})
	checkSection(t, out, "growth", map[string]float64{
		"membersStart": 4, "membersEnd": 4, "membersGrowthPct": 0, "savingsStart": 9000, "savingsEnd": 51000,
		"loanPortfolioStart": 19000, "loanPortfolioEnd": 33000, "assetsStart": 18000, "assetsEnd": 141600,
	})
	// Regular savers: m1 saved at 3/3 meetings, m3 at 2/2 (joined in August);
	// m2 2/3 (t6 reversed), m6 0/1.
	checkSection(t, out, "access", map[string]float64{
		"activeMembers": 4, "eligibleMembers": 4, "regularSavers": 2, "regularSaversPct": 50, "savers": 4, "saversPct": 100,
		"borrowers": 4, "accessingLoansPct": 100, "membersWithOpenLoans": 3, "membersWithOpenLoansPct": 75,
		"avgSavingsPerMemberStart": 1250, "avgSavingsPerMemberEnd": 12750, "avgSavingsChangePct": 920,
		"avgBorrowingPerMemberStart": 4750, "avgBorrowingPerMemberEnd": 6250, "avgBorrowingChangePct": 31.6, "avgLoanPerBorrower": 8500,
	})
	checkSection(t, out, "economicBenefits", map[string]float64{
		"igaLoans": 2, "igaBorrowers": 2, "igaAmount": 26000, "igaLoansPct": 50, "igaBorrowersPct": 50,
		"loansWithPurpose": 3, "unspecifiedLoans": 1, "returnsToMembers": 2000, "interestEarned": 2100,
	})

	// Breakdowns.
	eb := out["economicBenefits"].(datatype.DataMap)
	purposes := []string{}
	for _, r := range eb["byPurpose"].([]datatype.DataMap) {
		purposes = append(purposes, r["purpose"].(string))
	}
	if want := []string{"business", "livestock", "education", "unspecified"}; !equalStrings(purposes, want) {
		t.Errorf("byPurpose order = %v, want %v", purposes, want)
	}
	if r := rowBy(eb["byPurpose"], "purpose", "livestock"); r["incomeGenerating"] != true || r["loans"] != 1 {
		t.Errorf("livestock row = %v", r)
	}
	ent := eb["byEnterpriseType"].([]datatype.DataMap)
	if len(ent) != 2 || ent[0]["enterpriseType"] != "Mama lishe" || ent[1]["enterpriseType"] != "Poultry" || ent[1]["loans"] != 2 || ent[1]["borrowers"] != 2 {
		t.Errorf("byEnterpriseType = %v (Poultry and poultry are one type)", ent)
	}
	mem := out["membership"].(datatype.DataMap)
	if r := rowBy(mem["byAgeBand"], "ageBand", "36-59"); r["activeMembers"] != 1 || r["membersAtStart"] != 1 {
		t.Errorf("36-59 = %v (m6 active; m5 left before the period)", r)
	}
	if r := rowBy(mem["byAgeBand"], "ageBand", "unknown"); r["membersAtStart"] != 1 || r["retained"] != 0 || r["retentionRate"] != 0.0 {
		t.Errorf("unknown age = %v (m4, no DOB, left)", r)
	}
	if r := rowBy(mem["byGender"], "gender", "Male"); r["membersAtStart"] != 2 || r["retained"] != 1 || r["retentionRate"] != 50.0 {
		t.Errorf("male retention = %v", r)
	}
	if r := rowBy(mem["byGender"], "gender", "Female"); r["attendanceRate"] != 100.0 {
		t.Errorf("female attendance = %v (m1 4/4 present or late, m3 2/2, m6 1/1)", r)
	}
	ga := mem["byGenderAndAge"].([]datatype.DataMap)
	if len(ga) != 4 {
		t.Errorf("byGenderAndAge = %v", ga)
	}
	acc := out["access"].(datatype.DataMap)
	if r := rowBy(acc["byAgeBand"], "ageBand", "15-24"); r["regularSavers"] != 1 || r["regularSaversPct"] != 100.0 {
		t.Errorf("15-24 regular savers = %v", r)
	}

	// Monthly series: July, August, September.
	monthly := out["growth"].(datatype.DataMap)["monthly"].([]datatype.DataMap)
	if len(monthly) != 3 || monthly[0]["month"] != "2026-07" || monthly[2]["month"] != "2026-09" {
		t.Fatalf("monthly = %v", monthly)
	}
	// End of July: Umoja t0,t23,t1,t2 + share t3 = 29,000; portfolio l0
	// 5,500 + l1 22,000 + l2 8,000 + l5 6,600 = 42,100; 4 members.
	approx(t, "jul totalSavings", monthly[0]["totalSavings"], 29000)
	approx(t, "jul loanPortfolio", monthly[0]["loanPortfolio"], 42100)
	approx(t, "jul activeMembers", monthly[0]["activeMembers"], 4)
	approx(t, "aug newMembers", monthly[1]["newMembers"], 1)
	approx(t, "sep loanPortfolio", monthly[2]["loanPortfolio"], 33000)
	approx(t, "sep groupAssets", monthly[2]["groupAssets"], 141600)
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// At "now" the indicators' balances are the dashboard's (computeKPIs).
func TestMonitoringMatchesKPIs(t *testing.T) {
	fix := monFixture()
	out := computeMonitoring(monInput(fix), monScope, testNow, time.Time{}, time.Time{})
	var tot kpiTotals
	for _, k := range computeKPIs(monInput(fix).kpiData, monScope, testNow, time.Time{}, time.Time{}) {
		tot.add(k)
	}
	approx(t, "savings", out["savings"].(datatype.DataMap)["savingsBalance"], tot.Savings)
	approx(t, "shares", out["savings"].(datatype.DataMap)["sharesBalance"], tot.Shares)
	approx(t, "social fund", out["socialFund"].(datatype.DataMap)["availableBalance"], tot.SocialFund)
	approx(t, "loans", out["loans"].(datatype.DataMap)["outstandingBalance"], tot.LoansOutstanding)
	approx(t, "members", out["membership"].(datatype.DataMap)["activeMembers"], float64(tot.Members))
	approx(t, "kpi social fund after payout", tot.SocialFund, 1000)
	approx(t, "kpi savings after share-out", tot.Savings, 36000)
	// Default period: 12 calendar months to today.
	p := out["period"].(datatype.DataMap)
	if p["from"] != "2025-10-01" || p["to"] != "2026-09-25" || p["months"] != 12 {
		t.Errorf("default period = %v", p)
	}
}

// Reversed rows never count: reversing the share-out and the payout puts the
// balances back.
func TestMonitoringReversedExcluded(t *testing.T) {
	fix := monFixture()
	for _, r := range fix["Transaction"] {
		if r["id"] == "t12" || r["id"] == "t22" {
			r["reversed"] = true
		}
	}
	f, to := monPeriod(t)
	out := computeMonitoring(monInput(fix), monScope, testNow, f, to)
	checkSection(t, out, "socialFund", map[string]float64{"availableBalance": 4000, "disbursements": 0})
	checkSection(t, out, "savings", map[string]float64{"savingsBalance": 40000})
	checkSection(t, out, "financialPerformance", map[string]float64{"savingsSharedOut": 0, "returnsToMembers": 0})
	checkSection(t, out, "accountability", map[string]float64{"reversals": 4})
}

func TestAgeBandsAndDOB(t *testing.T) {
	at := time.Date(2026, 9, 25, 12, 0, 0, 0, eatZone)
	for age, want := range map[int]string{10: ageUnder15, 14: ageUnder15, 15: age15to24, 24: age15to24, 25: age25to35, 35: age25to35, 36: age36to59, 59: age36to59, 60: age60plus, 95: age60plus} {
		if got := ageBand(age); got != want {
			t.Errorf("ageBand(%d) = %q, want %q", age, got, want)
		}
	}
	// Birthday tomorrow: still 24.
	if got := ageBandOf(datatype.DataMap{"dateOfBirth": "2001-09-26"}, at); got != age15to24 {
		t.Errorf("day before 25th birthday = %q", got)
	}
	if got := ageBandOf(datatype.DataMap{"dateOfBirth": "2001-09-25"}, at); got != age25to35 {
		t.Errorf("25th birthday = %q", got)
	}
	if got := ageBandOf(datatype.DataMap{}, at); got != ageUnknown {
		t.Errorf("no DOB = %q", got)
	}
	if !isYouth(datatype.DataMap{"dateOfBirth": "1991-01-01"}, at) || isYouth(datatype.DataMap{"dateOfBirth": "1990-01-01"}, at) {
		t.Error("youth is 15-35")
	}

	ok := map[interface{}]string{nil: "", "": "", " 1990-04-12 ": "1990-04-12", "1990-04-12T00:00:00Z": "1990-04-12", "2016-09-25": "2016-09-25"}
	for in, want := range ok {
		got, err := validateDOB(in, at)
		if err != nil || got != want {
			t.Errorf("validateDOB(%v) = %q, %v; want %q", in, got, err, want)
		}
	}
	for _, bad := range []interface{}{"1990-02-30", "12/04/1990", "2026-09-26", "2016-09-26", "1915-01-01", 1990} {
		if _, err := validateDOB(bad, at); err == nil {
			t.Errorf("validateDOB(%v) should fail", bad)
		}
	}
}

func TestLoanPurposeValidation(t *testing.T) {
	p, e, err := validateLoanPurpose(" Business ", "  Mama   lishe ")
	if err != nil || p != "business" || e != "Mama lishe" {
		t.Errorf("got %q %q %v", p, e, err)
	}
	if p, _, err := validateLoanPurpose("", ""); err != nil || p != "" {
		t.Error("purpose is optional")
	}
	if _, _, err := validateLoanPurpose("boda", ""); err == nil {
		t.Error("unknown purpose must fail")
	}
	long := make([]byte, maxEnterpriseType+1)
	for i := range long {
		long[i] = 'x'
	}
	if _, _, err := validateLoanPurpose("other", string(long)); err == nil {
		t.Error("too-long enterpriseType must fail")
	}
	if loanPurpose(datatype.DataMap{}) != purposeUnspecified || loanPurpose(datatype.DataMap{"purpose": "health"}) != "health" {
		t.Error("loanPurpose")
	}
}

func TestShareOutSplitAndCash(t *testing.T) {
	sv, sh, pr := splitShareOut(12000, 8000, 3000)
	if sv != 8000 || sh != 3000 || pr != 1000 {
		t.Errorf("split = %v %v %v", sv, sh, pr)
	}
	sv, sh, pr = splitShareOut(5000, 8000, 3000)
	if sv != 5000 || sh != 0 || pr != 0 {
		t.Errorf("partial split = %v %v %v", sv, sh, pr)
	}
	txs := monFixture()["Transaction"]
	// m5: 4,000 saved, all returned at the share-out.
	if s, h := memberPool(txs, "m5"); s != 0 || h != 0 {
		t.Errorf("m5 pool after share-out = %v %v", s, h)
	}
	// m1: 5,000×4 saved − 2,000 withdrawn; 10,000 shares.
	if s, h := memberPool(txs, "m1"); s != 18000 || h != 10000 {
		t.Errorf("m1 pool = %v %v", s, h)
	}
	if got := memberSavingsShares(txs, "m5"); got != 0 {
		t.Errorf("loan limit base after share-out = %v", got)
	}
	g1 := []datatype.DataMap{}
	for _, r := range txs {
		if r["groupId"] == "g1" {
			g1 = append(g1, r)
		}
	}
	cash, fund := groupCash(g1, 0, 0)
	approx(t, "g1 cash", cash, 25000)
	approx(t, "g1 social fund after payout", fund, 1000)
	// Government money: borrowed money is not the members' (left out); once
	// more was repaid than received, the extra (interest) left the box.
	cash, _ = groupCash(g1, 100000, 20000)
	approx(t, "gov loan still owed: cash unchanged", cash, 25000)
	cash, _ = groupCash(g1, 100000, 105000)
	approx(t, "gov interest paid comes out of the box", cash, 20000)
	rec, rep := govFlows([]datatype.DataMap{{"amount": 100000.0}}, []datatype.DataMap{{"amount": 60000.0}, {"amount": 45000.0}})
	if rec != 100000 || rep != 105000 {
		t.Errorf("govFlows = %v %v", rec, rep)
	}
}

// A member is capped at their own savings + shares plus their pro-rata share
// of the group's profit; the profit paid in total is capped too.
func TestPlanShareOutEntitlement(t *testing.T) {
	txs := []datatype.DataMap{
		{"memberId": "a", "type": "contribution", "amount": 10000.0},
		{"memberId": "b", "type": "contribution", "amount": 90000.0},
		{"memberId": "b", "type": "contribution", "amount": 50000.0, "reversed": true},
		{"type": "fine", "amount": 10000.0}, // profit
	}
	avail := 110000.0 // pools 100,000 → profit 10,000: a may get 11,000, b 99,000
	if _, msg := planShareOut(txs, avail, []shareOutReq{{"a", 100000}}); msg == "" {
		t.Fatal("a must not be paid out of b's savings")
	}
	if _, msg := planShareOut(txs, avail, []shareOutReq{{"a", 11001.5}}); msg == "" {
		t.Error("a above entitlement (+1 TZS tolerance) must fail")
	}
	plans, msg := planShareOut(txs, avail, []shareOutReq{{"a", 11000}, {"b", 99000}})
	if msg != "" || len(plans) != 2 {
		t.Fatalf("full share-out refused: %s", msg)
	}
	if plans[0].sv != 10000 || plans[0].gain != 1000 || plans[1].gain != 9000 {
		t.Errorf("plans = %+v", plans)
	}
	// Loans still out: the box holds less than the pools → no profit; a
	// member can still take back their own savings, not more.
	if _, msg := planShareOut(txs, 50000, []shareOutReq{{"a", 10500}}); msg == "" {
		t.Error("no profit to share when the box is short")
	}
	if _, msg := planShareOut(txs, 50000, []shareOutReq{{"a", 10000}}); msg != "" {
		t.Errorf("own savings back: %s", msg)
	}
	if _, msg := planShareOut(txs, 50000, []shareOutReq{{"a", 10000}, {"b", 45000}}); msg == "" {
		t.Error("total above the cash must fail")
	}
	// Tolerance: rounding up to 1 TZS per member is accepted.
	if _, msg := planShareOut(txs, avail, []shareOutReq{{"a", 11000.6}, {"b", 98999.4}}); msg != "" {
		t.Errorf("rounding tolerance: %s", msg)
	}
}

func TestCycleAfterReversal(t *testing.T) {
	if n, ok := cycleAfterReversal(3, 2, false); !ok || n != 2 {
		t.Errorf("last payment reversed: %v %v", n, ok)
	}
	if _, ok := cycleAfterReversal(3, 2, true); ok {
		t.Error("other payments of the close remain: no change")
	}
	if _, ok := cycleAfterReversal(2, 2, false); ok {
		t.Error("never below the cycle that was closed")
	}
}

func TestPayoutLockSerialises(t *testing.T) {
	unlock := lockGroupPayouts("g-lock")
	done := make(chan bool)
	go func() {
		u := lockGroupPayouts("g-lock")
		u()
		done <- true
	}()
	select {
	case <-done:
		t.Fatal("second payout ran while the first held the lock")
	case <-time.After(50 * time.Millisecond):
	}
	other := lockGroupPayouts("g-other") // other groups are not blocked
	other()
	unlock()
	<-done
}

func TestMonitoringDatasets(t *testing.T) {
	c := testCtx(t, monFixture(), monScope, "2026-07-01", "2026-09-30", nil)
	sum := mustBuild(t, c, "monitoring-summary")
	if len(sum) != 2 {
		t.Fatalf("monitoring-summary rows = %d", len(sum))
	}
	umoja := byName(sum, "Group", "Umoja")
	approx(t, "Umoja retention", umoja["Retention %"], 66.7) // m1, m2 of m1, m2, m4
	approx(t, "Umoja repayment rate", umoja["Repayment Rate %"], 50)
	approx(t, "Umoja outstanding", umoja["Loans Outstanding"], 33000)
	if umoja["Partner"] != "GATA" || umoja["Using App"] != "yes" {
		t.Errorf("Umoja row = %v", umoja)
	}
	ds := findDataset("monitoring-summary")
	tot := reportTotals(ds, sum)
	approx(t, "total retention (recomputed)", tot["Retention %"], 75)
	approx(t, "total repayment rate (recomputed)", tot["Repayment Rate %"], 68.8)
	approx(t, "total regular savers (recomputed)", tot["Regular Savers %"], 50)
	approx(t, "total fully recorded (recomputed)", tot["Meetings Fully Recorded %"], 75)
	approx(t, "total members", tot["Members (active)"], 4)

	age := mustBuild(t, c, "members-by-age")
	if r := byName(age, "Age Band", "36-59"); r == nil || r["Female"] != 1 || r["Members (active)"] != 1 {
		t.Errorf("36-59 row = %v", r)
	}
	approx(t, "age totals retention", reportTotals(findDataset("members-by-age"), age)["Retention %"], 75)

	purpose := mustBuild(t, c, "loans-by-purpose")
	if len(purpose) != 4 || purpose[0]["Purpose"] != "business" || purpose[0]["Income-generating"] != "yes" {
		t.Errorf("loans-by-purpose = %v", purpose)
	}
	approx(t, "purpose principal total", reportTotals(findDataset("loans-by-purpose"), purpose)["Principal"], 34000)

	ms := mustBuild(t, c, "member-savings")
	asha := byName(ms, "Member", "Asha")
	if asha == nil || asha["Regular Saver"] != "yes" || asha["Age Band"] != age25to35 {
		t.Errorf("Asha = %v", asha)
	}
	approx(t, "Asha balance", asha["Savings & Shares Balance"], 28000)
	approx(t, "Asha owes", asha["Loans Outstanding"], 8500)
	if byName(ms, "Member", "Out") != nil {
		t.Error("out-of-scope member listed")
	}

	growth := mustBuild(t, c, "monthly-growth")
	if len(growth) != 3 || growth[0]["Month"] != "2026-07" {
		t.Errorf("monthly-growth = %v", growth)
	}
	payouts := mustBuild(t, c, "social-fund-payouts")
	if len(payouts) != 1 || payouts[0]["Amount"] != 3000.0 {
		t.Errorf("social-fund-payouts = %v", payouts)
	}
	so := mustBuild(t, c, "share-outs")
	if len(so) != 1 || so[0]["Return"] != 2000.0 || so[0]["Savings Returned"] != 4000.0 || so[0]["Cycle #"] != 1 {
		t.Errorf("share-outs = %v", so)
	}
	loans := mustBuild(t, c, "loans")
	if r := byName(loans, "Loan #", "LN-0005"); r == nil || r["Purpose"] != purposeUnspecified {
		t.Errorf("loan purpose column = %v", r)
	}
	members := mustBuild(t, testCtx(t, monFixture(), monScope, "", "", nil), "members")
	if r := byName(members, "Name", "Asha"); r == nil || r["Age Band"] != age25to35 || r["Date of Birth"] == nil {
		t.Errorf("members DOB columns = %v", r)
	}
}

func TestReviewFixes(t *testing.T) {
	// Date-picker timestamp: its EAT day (21:00 UTC is the next day in EAT).
	if got, err := validateDOB("1990-04-11T21:00:00Z", testNow); err != nil || got != "1990-04-12" {
		t.Errorf("RFC 3339 DOB = %q, %v", got, err)
	}
	if _, err := validateDOB("1990-04-11Tnonsense", testNow); err == nil {
		t.Error("bad timestamp must fail")
	}

	// from after to → error (also a future from with no to).
	f, _, _ := reportRange("2026-10-01", "")
	if _, _, _, _, err := monitorPeriod(f, time.Time{}, testNow); err == nil {
		t.Error("from after the default to must fail")
	}
	if _, _, err := reportRange("2999-01-01", ""); err == nil {
		t.Error("future from with no to must fail")
	}

	// Monthly points add up to the period totals when from is mid-month.
	fix := monFixture()
	from, to, err := reportRange("2026-07-15", "2026-09-30")
	if err != nil {
		t.Fatal(err)
	}
	out := computeMonitoring(monInput(fix), monScope, testNow, from, to)
	var saved, disb float64
	var joined int
	for _, m := range out["growth"].(datatype.DataMap)["monthly"].([]datatype.DataMap) {
		saved += num(m["savingsRecorded"])
		disb += num(m["loansDisbursed"])
		joined += int(num(m["newMembers"]))
	}
	approx(t, "Σ monthly savings = period", saved, num(out["savings"].(datatype.DataMap)["savingsRecorded"]))
	approx(t, "Σ monthly disbursed = period", disb, num(out["loans"].(datatype.DataMap)["amountDisbursed"]))
	approx(t, "Σ monthly new members = period", float64(joined), num(out["membership"].(datatype.DataMap)["newMembers"]))
	approx(t, "loans from 15 July: l3 + l6 (l1, l5 issued earlier)", out["loans"].(datatype.DataMap)["amountDisbursed"], 8000)

	// Social fund never negative (legacy payouts above the fund).
	fix = monFixture()
	fix["Transaction"] = append(fix["Transaction"], datatype.DataMap{"id": "tx", "groupId": "g1", "type": "social_fund_payout", "amount": 5000.0, "createdAt": "2026-09-13T07:00:00Z"})
	f2, t2 := monPeriod(t)
	out = computeMonitoring(monInput(fix), monScope, testNow, f2, t2)
	approx(t, "monitoring social fund clamped", out["socialFund"].(datatype.DataMap)["availableBalance"], 0)
	for _, k := range computeKPIs(monInput(fix).kpiData, monScope, testNow, time.Time{}, time.Time{}) {
		if k.SocialFund < 0 {
			t.Errorf("kpi social fund %s = %v", k.GroupID, k.SocialFund)
		}
	}

	// SMS figures only for groups with sms.view, audit only with audit.view.
	d := monInput(monFixture())
	d.Hide = &monitorHide{smsOK: func(gid string) bool { return gid == "g2" }, noAudit: true}
	out = computeMonitoring(d, monScope, testNow, f2, t2)
	acc := out["accountability"].(datatype.DataMap)
	if acc["smsSent"] != 1 || acc["smsFailed"] != 0 || acc["smsHidden"] != true || acc["auditEntries"] != 0 || acc["auditHidden"] != true {
		t.Errorf("hidden figures = %v", acc)
	}
}

func TestNextMemberNumber(t *testing.T) {
	cases := []struct {
		in   []string
		want string
	}{
		{nil, "001"},
		{[]string{"", "  "}, "001"},
		{[]string{"001", "002"}, "003"},
		{[]string{"M-007", "3", "abc"}, "008"},
		{[]string{"001", "005"}, "006"}, // gaps are not reused
		{[]string{"1234"}, "1235"},
	}
	for _, c := range cases {
		if got := nextMemberNumber(c.in); got != c.want {
			t.Errorf("nextMemberNumber(%v) = %q, want %q", c.in, got, c.want)
		}
	}
}
