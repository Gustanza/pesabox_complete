package main

import (
	"archive/zip"
	"bytes"
	"math/rand"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/robertkonga/yekonga-server-go/datatype"
	"github.com/xuri/excelize/v2"
)

// testNow is "today" for every report test: 2026-09-25 12:00 EAT.
var testNow = time.Date(2026, 9, 25, 12, 0, 0, 0, eatZone)

// reportFixture is a small programme: Umoja (partner P1) and Amani (partner
// P2) in scope, Zeta out of scope. Amounts are chosen so every rule shows.
func reportFixture() map[string][]datatype.DataMap {
	return map[string][]datatype.DataMap{
		"Partner": {{"id": "p1", "name": "GATA Partner"}, {"id": "p2", "name": "Other Partner"}},
		"Cluster": {{"id": "c1", "name": "Arusha", "partnerId": "p1"}, {"id": "c2", "name": "Moshi", "partnerId": "p2"}},
		"Group": {
			{"id": "g1", "name": "Umoja", "clusterId": "c1", "region": "Arusha", "status": "Active", "createdAt": "2026-01-01T00:00:00Z",
				"totalSavings": 20000.0, "totalFines": 5000.0, "cycleCurrent": 3, "cycleTotal": 30, "meetingFrequency": "Weekly", "formationDate": "2026-01-05"},
			{"id": "g2", "name": "Amani", "clusterId": "c2", "status": "Active", "meetingFrequency": "Monthly"}, // no createdAt
			{"id": "g3", "name": "Zeta", "clusterId": "c2", "status": "Active"},
		},
		"Member": {
			{"id": "m1", "groupId": "g1", "firstName": "Asha", "lastName": "Juma", "gender": "Female", "status": "Active", "joinedAt": "2026-09-16T22:00:00Z"},
			{"id": "m2", "groupId": "g1", "firstName": "John", "lastName": "Mfinanga", "gender": "Male", "status": "Active", "joinedAt": "2026-08-01T07:00:00Z"},
			{"id": "m3", "groupId": "g1", "firstName": "Bakari", "lastName": "Said", "gender": "Male", "status": "Suspended", "joinedAt": "2026-01-10T07:00:00Z"},
			{"id": "m4", "groupId": "g2", "firstName": "Neema", "lastName": "Mushi", "gender": "Female", "joinedAt": "2026-09-01T07:00:00Z"},
			{"id": "m5", "groupId": "g3", "firstName": "Out", "lastName": "OfScope", "status": "Active"},
		},
		"Meeting": {
			{"id": "mt1", "groupId": "g1", "meetingNumber": 1, "date": "2026-09-10", "status": "completed"},
			{"id": "mt2", "groupId": "g1", "meetingNumber": 2, "date": "2026-09-17", "status": "completed"},
			{"id": "mt3", "groupId": "g1", "meetingNumber": 3, "date": "2026-09-20", "status": "cancelled"},
			{"id": "mt4", "groupId": "g2", "meetingNumber": 1, "date": "2026-09-17", "status": "completed"},
		},
		"MeetingAttendance": {
			{"id": "a1", "groupId": "g1", "meetingId": "mt1", "memberId": "m1", "status": "present", "createdAt": "2026-09-10T08:00:00Z"},
			{"id": "a2", "groupId": "g1", "meetingId": "mt1", "memberId": "m2", "status": "absent", "createdAt": "2026-09-10T08:00:00Z"},
			{"id": "a3", "groupId": "g1", "meetingId": "mt2", "memberId": "m1", "status": "late", "createdAt": "2026-09-17T08:00:00Z"},
			{"id": "a4", "groupId": "g1", "meetingId": "mt2", "memberId": "m2", "status": "present", "createdAt": "2026-09-17T08:00:00Z"},
			{"id": "a5", "groupId": "g1", "meetingId": "mt2", "memberId": "m3", "status": "absent", "createdAt": "2026-09-17T08:00:00Z"},
			{"id": "a6", "groupId": "g1", "meetingId": "mt3", "memberId": "m1", "status": "present"}, // cancelled meeting
			{"id": "a7", "meetingId": "mt4", "memberId": "m4", "status": "present"},                  // no groupId: from its meeting
		},
		"Transaction": {
			{"id": "t1", "groupId": "g1", "memberId": "m1", "meetingId": "mt1", "type": "contribution", "direction": "in", "amount": 5000.0, "createdAt": "2026-09-10T08:00:00Z", "method": "Cash"},
			{"id": "t2", "groupId": "g1", "memberId": "m2", "meetingId": "mt1", "type": "contribution", "direction": "in", "amount": 5000.0, "createdAt": "2026-09-10T08:05:00Z", "method": "Cash"},
			{"id": "t3", "groupId": "g1", "memberId": "m2", "meetingId": "mt1", "type": "contribution", "direction": "in", "amount": 7000.0, "createdAt": "2026-09-10T08:06:00Z", "reversed": true},
			// 2026-09-17 01:00 EAT = 2026-09-16 22:00 UTC; no meetingId → the group's meeting that EAT day.
			{"id": "t4", "groupId": "g1", "memberId": "m1", "type": "share", "direction": "in", "amount": 10000.0, "createdAt": "2026-09-16T22:00:00Z"},
			{"id": "t5", "groupId": "g1", "memberId": "m2", "meetingId": "mt2", "type": "fine", "direction": "in", "amount": 1000.0, "createdAt": "2026-09-17T07:00:00Z", "reference": "f1"},
			{"id": "t6", "groupId": "g1", "memberId": "m2", "meetingId": "mt1", "type": "loan_disbursement", "direction": "out", "amount": 50000.0, "createdAt": "2026-09-10T09:00:00Z", "reference": "LN-0001"},
			{"id": "t7", "groupId": "g1", "memberId": "m2", "meetingId": "mt2", "type": "loan_repayment", "direction": "in", "amount": 10000.0, "createdAt": "2026-09-17T09:00:00Z", "reference": "LN-0001"},
			{"id": "t8", "groupId": "g1", "meetingId": "mt2", "type": "expense", "direction": "out", "amount": 2000.0, "createdAt": "2026-09-17T10:00:00Z", "description": "Stationery"},
			{"id": "t9", "groupId": "g1", "memberId": "m1", "type": "withdrawal", "direction": "out", "amount": 1000.0, "createdAt": "2026-09-20T10:00:00Z"},
			{"id": "t10", "groupId": "g1", "memberId": "m1", "type": "loan_disbursement", "direction": "out", "amount": 20000.0, "createdAt": "2026-09-10T09:30:00Z", "reversed": true},
			{"id": "t11", "groupId": "g2", "memberId": "m4", "meetingId": "mt4", "type": "contribution", "direction": "in", "amount": 3000.0, "createdAt": "2026-09-17T08:00:00Z"},
			{"id": "t12", "groupId": "g1", "memberId": "m1", "type": "social_fund", "direction": "in", "amount": 500.0, "createdAt": "2026-08-01T08:00:00Z"},
			// 2026-09-18 00:30 EAT — just after a to=2026-09-17 range.
			{"id": "t13", "groupId": "g2", "memberId": "m4", "type": "contribution", "direction": "in", "amount": 100.0, "createdAt": "2026-09-17T21:30:00Z"},
			{"id": "t14", "groupId": "g3", "type": "contribution", "amount": 999999.0, "createdAt": "2026-09-17T08:00:00Z"},
		},
		"Loan": {
			{"id": "l1", "groupId": "g1", "memberId": "m2", "loanNumber": "LN-0001", "amount": 50000.0, "amountRepaid": 10000.0, "interestRate": 10.0, "status": "active", "issuedDate": "2026-06-01T08:00:00Z", "dueDate": "2026-08-01T08:00:00Z"},
			{"id": "l2", "groupId": "g1", "memberId": "m1", "loanNumber": "LN-0002", "amount": 20000.0, "amountRepaid": 0.0, "status": "cancelled", "issuedDate": "2026-09-10T09:30:00Z", "dueDate": "2026-06-01T08:00:00Z"},
			{"id": "l3", "groupId": "g1", "memberId": "m1", "loanNumber": "LN-0003", "amount": 8000.0, "amountRepaid": 3000.0, "status": "defaulted", "issuedDate": "2026-05-01T08:00:00Z", "dueDate": "2026-08-10T08:00:00Z"},
			{"id": "l4", "groupId": "g2", "memberId": "m4", "loanNumber": "LN-0001", "amount": 10000.0, "amountRepaid": 10000.0, "status": "repaid", "issuedDate": "2026-04-01T08:00:00Z"},
			{"id": "l5", "groupId": "g2", "memberId": "m4", "loanNumber": "LN-0002", "amount": 15000.0, "amountRepaid": 0.0, "status": "active", "issuedDate": "2026-09-01T08:00:00Z", "dueDate": "2026-12-01T08:00:00Z"},
			{"id": "l6", "groupId": "g1", "memberId": "m1", "loanNumber": "LN-0004", "amount": 5000.0, "amountRepaid": 6000.0, "status": "active", "issuedDate": "2026-09-01T08:00:00Z"}, // over-repaid: clamp at 0
		},
		"Fine": {
			{"id": "f1", "groupId": "g1", "memberId": "m2", "reason": "Late Attendance", "amount": 2000.0, "amountPaid": 1000.0, "status": "pending", "issuedAt": "2026-09-17T07:00:00Z"},
			{"id": "f2", "groupId": "g1", "memberId": "m1", "reason": "Absent", "amount": 2000.0, "amountPaid": 0.0, "status": "waived", "issuedAt": "2026-09-10T07:00:00Z"},
			{"id": "f3", "groupId": "g1", "memberId": "m1", "reason": "Other", "amount": 1000.0, "amountPaid": 1000.0, "status": "paid", "createdAt": "2026-09-01T07:00:00Z"},
		},
		"GovernmentLoan": {
			{"id": "gl1", "groupId": "g1", "lender": "Halmashauri ya Arusha", "programme": "10% Women Loan", "reference": "HA-7", "amount": 1000000.0, "interestRate": 10.0, "amountRepaid": 250000.0, "status": "active", "receivedDate": "2026-09-01T07:00:00Z"},
		},
		"GovernmentLoanRepayment": {
			{"id": "gr1", "groupId": "g1", "governmentLoanId": "gl1", "amount": 250000.0, "method": "Bank Transfer", "date": "2026-09-15T07:00:00Z"},
		},
		"SmsLog": {
			{"id": "s1", "groupId": "g1", "memberId": "m1", "phone": "0755000001", "messageType": "contribution", "status": "sent", "sentAt": "2026-09-10T08:00:00Z"},
			{"id": "s2", "groupId": "g1", "memberId": "m2", "phone": "0755000002", "messageType": "fine", "status": "failed", "sentAt": "2026-09-17T07:00:00Z"},
			{"id": "s3", "phone": "0755000009", "messageType": "login_otp", "status": "sent", "sentAt": "2026-09-17T07:00:00Z"},
			{"id": "s4", "groupId": "g2", "memberId": "m4", "phone": "0755000004", "messageType": "contribution", "status": "sent", "sentAt": "2026-09-17T08:00:00Z"},
		},
	}
}

var inScope = map[string]bool{"g1": true, "g2": true}

func testCtx(t *testing.T, fix map[string][]datatype.DataMap, set map[string]bool, from, to string, smsOK func(string) bool) *reportCtx {
	t.Helper()
	f, tt, err := reportRange(from, to)
	if err != nil {
		t.Fatal(err)
	}
	c := &reportCtx{set: set, from: f, to: tt, now: testNow, smsOK: smsOK}
	c.fetch = func(model, _ string) []datatype.DataMap { return fix[model] }
	c.byIds = func(model string, ids []string) []datatype.DataMap {
		want := map[string]bool{}
		for _, id := range ids {
			want[id] = true
		}
		out := []datatype.DataMap{}
		for _, r := range fix[model] {
			if want[toStr(r["id"])] {
				out = append(out, r)
			}
		}
		return out
	}
	c.init()
	return c
}

func mustBuild(t *testing.T, c *reportCtx, key string) []datatype.DataMap {
	t.Helper()
	rows, ok := buildReport(c, key)
	if !ok {
		t.Fatalf("unknown dataset %q", key)
	}
	return rows
}

func col(rows []datatype.DataMap, key string) []interface{} {
	out := []interface{}{}
	for _, r := range rows {
		out = append(out, r[key])
	}
	return out
}

func approx(t *testing.T, what string, got interface{}, want float64) {
	t.Helper()
	if g := num(got); g < want-0.001 || g > want+0.001 {
		t.Errorf("%s = %v, want %v", what, got, want)
	}
}

func byName(rows []datatype.DataMap, key, name string) datatype.DataMap {
	for _, r := range rows {
		if r[key] == name {
			return r
		}
	}
	return nil
}

// ---- registry --------------------------------------------------------------

func TestRegistryColumnsAreUniqueAndTranslated(t *testing.T) {
	seen := map[string]bool{}
	for _, d := range reportDatasets {
		if seen[d.Key] {
			t.Fatalf("duplicate dataset key %q", d.Key)
		}
		seen[d.Key] = true
		if len(d.Columns) == 0 {
			t.Fatalf("dataset %q has no columns", d.Key)
		}
		cols := map[string]bool{}
		for _, c := range d.Columns {
			if cols[c] {
				t.Errorf("dataset %q repeats column %q", d.Key, c)
			}
			cols[c] = true
			if _, ok := columnSw[c]; !ok {
				t.Errorf("dataset %q column %q has no Swahili label", d.Key, c)
			}
		}
		if d.Sw == "" || d.En == "" || categorySw[d.Category] == "" {
			t.Errorf("dataset %q is missing a label", d.Key)
		}
		if _, ok := buildReport(testCtx(t, reportFixture(), inScope, "", "", nil), d.Key); !ok {
			t.Errorf("dataset %q has no builder", d.Key)
		}
	}
	if findDataset("group-activity") == nil || findDataset("group-activity").Key != "transactions" {
		t.Error("group-activity must stay as an alias of transactions")
	}
	if seen["group-activity"] {
		t.Error("group-activity must not be listed twice")
	}
}

// ---- time: EAT days, parsing, validation -----------------------------------

func TestReportRangeEATAndValidation(t *testing.T) {
	from, to, err := reportRange("2026-09-17", "2026-09-17")
	if err != nil {
		t.Fatal(err)
	}
	if want := time.Date(2026, 9, 16, 21, 0, 0, 0, time.UTC); !from.Equal(want) {
		t.Errorf("from = %v, want %v", from.UTC(), want)
	}
	if want := time.Date(2026, 9, 17, 20, 59, 59, 999999999, time.UTC); !to.Equal(want) {
		t.Errorf("to = %v, want %v", to.UTC(), want)
	}
	for _, bad := range [][2]string{{"2026-13-01", ""}, {"", "17/09/2026"}, {"yesterday", ""}, {"2026-09-18", "2026-09-17"}} {
		if _, _, err := reportRange(bad[0], bad[1]); err == nil {
			t.Errorf("reportRange(%q, %q) should fail", bad[0], bad[1])
		}
	}
	if _, _, err := reportRange("", ""); err != nil {
		t.Errorf("an open range is valid: %v", err)
	}
}

func TestParseTimeOrZero(t *testing.T) {
	if !parseTimeOrZero(nil).IsZero() || !parseTimeOrZero("").IsZero() || !parseTimeOrZero("next friday").IsZero() {
		t.Error("missing / unreadable dates must be zero, not now")
	}
	if d := parseTimeOrZero("2026-09-17"); !d.Equal(time.Date(2026, 9, 17, 0, 0, 0, 0, eatZone)) {
		t.Errorf("bare date = %v, want EAT midnight", d)
	}
	if d := parseTimeOrZero("2026-09-16T22:00:00Z"); d.In(eatZone).Format("2006-01-02 15:04") != "2026-09-17 01:00" {
		t.Errorf("RFC3339 = %v", d)
	}
}

func TestEATBoundary(t *testing.T) {
	c := testCtx(t, reportFixture(), inScope, "2026-09-17", "2026-09-17", nil)
	rows := mustBuild(t, c, "transactions")
	ids := map[interface{}]bool{}
	for _, r := range rows {
		ids[r["_id"]] = true
	}
	if !ids["t4"] {
		t.Error("a transaction at 2026-09-17 01:00 EAT (09-16 22:00Z) must be in from=to=2026-09-17")
	}
	if ids["t13"] {
		t.Error("a transaction at 2026-09-18 00:30 EAT must not be in to=2026-09-17")
	}
	var t4 datatype.DataMap
	for _, r := range rows {
		if r["_id"] == "t4" {
			t4 = r
		}
	}
	ds := findDataset("transactions")
	if got := cellText("en", colType(ds, "Date"), t4["Date"], false); !strings.HasPrefix(got, "2026-09-17") {
		t.Errorf("t4 printed as %q, want 2026-09-17…", got)
	}
	if got := cellText("en", "date", t4["Date"], false); got != "2026-09-17" {
		t.Errorf("t4 as a date = %q", got)
	}
	// ...and bucketed on the 17th in the summary period, too.
	g1 := byName(mustBuild(t, c, "summary-group"), "Name", "Umoja")
	approx(t, "Umoja shares (period 17th)", g1["Shares (period)"], 10000)
}

// ---- R1: the date range applies ---------------------------------------------

func TestDateFilterMeetingsMembersAttendance(t *testing.T) {
	c := testCtx(t, reportFixture(), inScope, "2026-09-17", "2026-09-17", nil)

	meetings := mustBuild(t, c, "meetings")
	if got := col(meetings, "_id"); !reflect.DeepEqual(got, []interface{}{"mt4", "mt2"}) {
		t.Errorf("meetings on the 17th = %v, want [mt4 mt2] (Amani, then Umoja)", got)
	}

	members := mustBuild(t, c, "members")
	if got := col(members, "_id"); !reflect.DeepEqual(got, []interface{}{"m1"}) {
		t.Errorf("members joined on the 17th = %v, want [m1] (joined 01:00 EAT)", got)
	}

	att := mustBuild(t, c, "attendance")
	if len(att) != 4 {
		t.Fatalf("attendance rows = %d, want 4 (mt2 ×3 + mt4 ×1)", len(att))
	}
	for _, r := range att {
		if cellText("en", "date", r["Meeting Date"], false) != "2026-09-17" || r["Meeting #"] == nil {
			t.Errorf("attendance row without meeting #/date: %v", r)
		}
	}
	if byName(att, "Group", "Amani") == nil {
		t.Error("an attendance row without groupId must fall back to its meeting's group")
	}

	all := mustBuild(t, testCtx(t, reportFixture(), inScope, "", "", nil), "attendance")
	for _, r := range all {
		if r["_id"] == "a6" {
			t.Error("attendance of a cancelled meeting must be skipped")
		}
	}
}

func TestSummaryPeriodColumns(t *testing.T) {
	c := testCtx(t, reportFixture(), inScope, "2026-09-17", "2026-09-17", nil)
	rows := mustBuild(t, c, "summary-group")
	u, a := byName(rows, "Name", "Umoja"), byName(rows, "Name", "Amani")
	if u == nil || a == nil || len(rows) != 2 {
		t.Fatalf("summary-group rows = %v", col(rows, "Name"))
	}
	// Period (17th only)
	approx(t, "Umoja savings (period)", u["Savings (period)"], 0)
	approx(t, "Umoja fines (period)", u["Fines (period)"], 1000)
	approx(t, "Umoja repayments (period)", u["Loan Repayments (period)"], 10000)
	approx(t, "Umoja expenses (period)", u["Expenses (period)"], 2000)
	approx(t, "Umoja meetings (period)", u["Meetings Held (period)"], 1)
	approx(t, "Umoja attendance % (period)", u["Attendance % (period)"], 66.7)
	approx(t, "Amani savings (period)", a["Savings (period)"], 3000) // t13 is on the 18th EAT
	// Balances (all time, point in time)
	approx(t, "Umoja savings balance", u["Savings Balance"], 9000) // 5000+5000-1000 withdrawal; t3 reversed
	approx(t, "Umoja shares balance", u["Shares Balance"], 10000)
	approx(t, "Umoja social fund balance", u["Social Fund Balance"], 500)
	approx(t, "Amani savings balance", a["Savings Balance"], 3100)
	approx(t, "Umoja members (active)", u["Members (active)"], 2) // m3 is Suspended
	approx(t, "Umoja members (total)", u["Members (total)"], 3)
	approx(t, "Umoja attendance %", u["Attendance %"], 60) // 3 of 5, cancelled meeting skipped

	growth := mustBuild(t, c, "group-growth")
	approx(t, "Umoja new members (period)", byName(growth, "Group", "Umoja")["New Members (period)"], 1)
	approx(t, "Umoja members (active) in growth", byName(growth, "Group", "Umoja")["Members (active)"], 2)
}

// ---- R4 / R18 / R15: computed balances, loans --------------------------------

func TestKPIsFromTransactionsNotStoredTotals(t *testing.T) {
	fix := reportFixture()
	kpis := computeKPIs(kpiData{
		Groups: fix["Group"], Clusters: fix["Cluster"], Members: fix["Member"], Loans: fix["Loan"],
		GovLoans: fix["GovernmentLoan"], Meetings: fix["Meeting"], Attendance: fix["MeetingAttendance"], Transactions: fix["Transaction"],
	}, inScope, testNow, time.Time{}, time.Time{})
	if len(kpis) != 2 {
		t.Fatalf("kpis for %d groups, want 2", len(kpis))
	}
	var u, a groupKPI
	for _, k := range kpis {
		if k.GroupID == "g1" {
			u = k
		} else {
			a = k
		}
	}
	approx(t, "fines", u.Fines, 1000) // stored totalFines says 5000
	approx(t, "expenses", u.Expenses, 2000)
	// l1 40000 + l3 (defaulted) 5000; l2 cancelled = 0; l6 over-repaid clamps to 0.
	approx(t, "loans outstanding", u.LoansOutstanding, 45000)
	approx(t, "PAR", u.PAR, 45000)
	approx(t, "gov loans", u.GovLoansOutstanding, 850000)
	if !u.Active {
		t.Error("Umoja met 8 days ago: active")
	}
	if a.Active != true { // met on the 17th
		t.Error("Amani met on the 17th: active")
	}
	drift := groupTotalsDrift(fix["Group"][0], u)
	if _, ok := drift["totalFines"]; !ok {
		t.Errorf("drift should report totalFines (stored 5000, computed 1000): %v", drift)
	}

	// No createdAt and no meeting → not auto-"recent".
	lonely := computeKPIs(kpiData{Groups: []datatype.DataMap{{"id": "gx", "name": "New"}}}, nil, testNow, time.Time{}, time.Time{})
	if lonely[0].Active {
		t.Error("a group with no createdAt must not count as recently formed")
	}
}

func TestLoansCancelledAndReversed(t *testing.T) {
	c := testCtx(t, reportFixture(), inScope, "", "", nil)
	loans := mustBuild(t, c, "loans")
	l2 := byName(loans, "Loan #", "LN-0002")
	for _, r := range loans {
		if r["_id"] == "l2" {
			l2 = r
		}
	}
	approx(t, "cancelled loan repaid (real amount)", l2["Repaid"], 0)
	approx(t, "cancelled loan balance", l2["Balance"], 0)
	for _, r := range loans {
		if r["_id"] == "l6" {
			approx(t, "over-repaid loan balance clamps", r["Balance"], 0)
		}
		if r["_id"] == "l1" {
			approx(t, "interest rate column", r["Interest %"], 10)
		}
	}
	for _, key := range []string{"transactions", "savings"} {
		for _, r := range mustBuild(t, c, key) {
			if r["_id"] == "t3" || r["_id"] == "t10" {
				t.Errorf("%s includes reversed transaction %v", key, r["_id"])
			}
			if r["_id"] == "t14" {
				t.Errorf("%s includes an out-of-scope group", key)
			}
		}
	}
	par := mustBuild(t, c, "portfolio-at-risk")
	if got := col(par, "_id"); !reflect.DeepEqual(got, []interface{}{"l1", "l3"}) {
		t.Errorf("PAR loans = %v, want [l1 l3] (defaulted stays at risk, most overdue first)", got)
	}
}

// ---- R8c: totals -----------------------------------------------------------

func TestTotalsRowMath(t *testing.T) {
	c := testCtx(t, reportFixture(), inScope, "", "", nil)
	ds := findDataset("summary-group")
	rows := mustBuild(t, c, "summary-group")
	tot := reportTotals(ds, rows)
	approx(t, "total savings balance", tot["Savings Balance"], 12100)
	approx(t, "total members (active)", tot["Members (active)"], 3)
	approx(t, "total groups", tot["Groups"], 2)
	approx(t, "total loans outstanding", tot["Loans Outstanding"], 60000)
	// Umoja PAR 100%, Amani 0% → recomputed 45000/60000 = 75%, not 100 or 50.
	approx(t, "total PAR %", tot["PAR 30 %"], 75)
	approx(t, "total attendance %", tot["Attendance %"], 66.7) // (3+1)/(5+1)
	if _, ok := tot["Name"]; ok {
		t.Error("the label cell is the writer's job")
	}

	perf := reportTotals(findDataset("group-performance"), mustBuild(t, c, "group-performance"))
	approx(t, "group-performance PAR %", perf["PAR 30 %"], 75)
	if _, ok := perf["Last Meeting"]; ok {
		t.Error("dates are not totalled")
	}

	// CSV carries the TOTAL row under the label column.
	set := exportSet{ds: ds, cols: []string{"Name", "Savings Balance", "PAR 30 %"}, rows: rows, totals: tot, total: len(rows)}
	data, _, _ := exportCSV(exportMeta{lang: "en", asAt: "2026-09-25"}, []exportSet{set})
	if !strings.Contains(string(data), "TOTAL,12100,75") {
		t.Errorf("csv totals row missing:\n%s", data)
	}
}

// ---- R21: deterministic order -------------------------------------------------

func TestDeterministicOrder(t *testing.T) {
	base := map[string][]string{}
	for _, d := range reportDatasets {
		for _, r := range mustBuild(t, testCtx(t, reportFixture(), inScope, "", "", func(string) bool { return true }), d.Key) {
			base[d.Key] = append(base[d.Key], strings.TrimSpace(strings.Join([]string{toStr(r["_id"]), toStr(r["Name"]), toStr(r["Group"])}, "|")))
		}
	}
	rng := rand.New(rand.NewSource(7))
	for i := 0; i < 5; i++ {
		fix := reportFixture()
		for _, recs := range fix {
			rng.Shuffle(len(recs), func(a, b int) { recs[a], recs[b] = recs[b], recs[a] })
		}
		for _, d := range reportDatasets {
			got := []string{}
			for _, r := range mustBuild(t, testCtx(t, fix, inScope, "", "", func(string) bool { return true }), d.Key) {
				got = append(got, strings.TrimSpace(strings.Join([]string{toStr(r["_id"]), toStr(r["Name"]), toStr(r["Group"])}, "|")))
			}
			if !reflect.DeepEqual(got, base[d.Key]) {
				t.Fatalf("%s order changed with input order:\n%v\n%v", d.Key, base[d.Key], got)
			}
		}
	}
	tx := mustBuild(t, testCtx(t, reportFixture(), inScope, "", "", nil), "transactions")
	for i := 1; i < len(tx); i++ {
		if tx[i]["_t"].(time.Time).After(tx[i-1]["_t"].(time.Time)) {
			t.Fatal("transactions must be newest first")
		}
	}
	members := mustBuild(t, testCtx(t, reportFixture(), inScope, "", "", nil), "members")
	if got := col(members, "Name"); !reflect.DeepEqual(got, []interface{}{"Neema Mushi", "Asha Juma", "Bakari Said", "John Mfinanga"}) {
		t.Errorf("members by group then name = %v", got)
	}
}

func toStr(v interface{}) string {
	if v == nil {
		return ""
	}
	return strings.TrimSpace(strings.ReplaceAll(strings.ToLower(cellText("en", "text", v, false)), "\n", " "))
}

// ---- R5: SMS permission --------------------------------------------------------

func TestSmsPermissionGating(t *testing.T) {
	admin := &Actor{HomeGroupID: "g1", HomePerms: groupAdminPerms, Perms: map[string]bool{}}
	officer := &Actor{HomeGroupID: "g1", HomePerms: groupOfficerPerms, Perms: map[string]bool{}}
	partner := &Actor{Perms: platformPerms(RolePartner, ""), GroupIDs: map[string]bool{"g1": true, "g2": true}}
	staff := &Actor{Perms: platformPerms(RoleStaff, PresetViewer), GroupIDs: map[string]bool{"g1": true}}
	if !hasSmsReports(admin) || hasSmsReports(officer) || hasSmsReports(partner) || !hasSmsReports(staff) {
		t.Error("SMS datasets: group admin and staff yes; officer and partner no")
	}
	if !admin.CanIn(PermSms, "g1") || admin.CanIn(PermSms, "g2") || staff.CanIn(PermSms, "g2") {
		t.Error("SMS permission must be per group")
	}

	// Rows only for groups the caller may read SMS of; OTP / groupless never.
	onlyG1 := testCtx(t, reportFixture(), inScope, "", "", func(g string) bool { return staff.CanIn(PermSms, g) })
	usage := mustBuild(t, onlyG1, "sms-usage")
	if got := col(usage, "_id"); !reflect.DeepEqual(got, []interface{}{"s2", "s1"}) {
		t.Errorf("sms-usage for a g1-only caller = %v, want [s2 s1]", got)
	}
	everyone := testCtx(t, reportFixture(), nil, "", "", func(g string) bool { return g != "" })
	for _, r := range mustBuild(t, everyone, "sms-usage") {
		if r["_id"] == "s3" {
			t.Error("OTP / groupless SMS must never be in a report")
		}
	}
	noSms := testCtx(t, reportFixture(), inScope, "", "", nil)
	if n := len(mustBuild(t, noSms, "sms-usage")); n != 0 {
		t.Errorf("without an SMS check no rows may be shown, got %d", n)
	}

	delivery := mustBuild(t, everyone, "sms-delivery")
	if len(delivery) != 3 {
		t.Fatalf("sms-delivery rows = %d, want 3 (g1×contribution, g1×fine, g2×contribution)", len(delivery))
	}
	for _, r := range delivery {
		if r["Group"] == "Umoja" && r["Message Type"] == "fine" && (r["Failed"] != 1 || r["Sent"] != 0 || num(r["Delivery %"]) != 0) {
			t.Errorf("fine delivery row = %v", r)
		}
	}
}

// ---- R6 / R8: new datasets -----------------------------------------------------

func TestFinesOutstanding(t *testing.T) {
	rows := mustBuild(t, testCtx(t, reportFixture(), inScope, "", "", nil), "fines-outstanding")
	if got := col(rows, "_id"); !reflect.DeepEqual(got, []interface{}{"f1", "f2", "f3"}) {
		t.Fatalf("fines-outstanding = %v, want newest first [f1 f2 f3]", got)
	}
	approx(t, "f1 outstanding", rows[0]["Outstanding"], 1000)
	approx(t, "f1 charged", rows[0]["Charged"], 2000)
	approx(t, "f2 waived outstanding", rows[1]["Outstanding"], 0)
	approx(t, "f3 paid outstanding", rows[2]["Outstanding"], 0)
	if rows[2]["Date"] == nil {
		t.Error("a fine without issuedAt uses createdAt")
	}
	if findDataset("fines").En != "Fine Payments" {
		t.Error("the transaction-based fines dataset is labelled Fine Payments")
	}
}

func TestMeetingCollections(t *testing.T) {
	rows := mustBuild(t, testCtx(t, reportFixture(), inScope, "", "", nil), "meeting-collections")
	if got := col(rows, "_id"); !reflect.DeepEqual(got, []interface{}{"mt4", "mt1", "mt2", "mt3"}) {
		t.Fatalf("meeting order = %v", got)
	}
	mt1, mt2, mt3 := rows[1], rows[2], rows[3]
	approx(t, "mt1 savings (reversed excluded)", mt1["Savings"], 10000)
	approx(t, "mt1 loans disbursed", mt1["Loans Disbursed"], 50000)
	approx(t, "mt1 total out", mt1["Total Out"], 50000)
	approx(t, "mt1 present", mt1["Present"], 1)
	approx(t, "mt1 absent", mt1["Absent"], 1)
	approx(t, "mt2 shares (same-day fallback)", mt2["Shares"], 10000)
	approx(t, "mt2 total in", mt2["Total In"], 21000) // 10000 shares + 1000 fine + 10000 repayment
	approx(t, "mt2 total out", mt2["Total Out"], 2000)
	approx(t, "mt2 late", mt2["Late"], 1)
	approx(t, "mt3 (cancelled) gets no fallback money", mt3["Total Out"], 0)

	ranged := mustBuild(t, testCtx(t, reportFixture(), inScope, "2026-09-17", "2026-09-17", nil), "meeting-collections")
	if len(ranged) != 2 {
		t.Errorf("meetings on the 17th = %d, want 2", len(ranged))
	}
	tot := reportTotals(findDataset("meeting-collections"), rows)
	approx(t, "total in over all meetings", tot["Total In"], 34000) // 10000 + 21000 + 3000
}

func TestExpensesAndGovRepayments(t *testing.T) {
	c := testCtx(t, reportFixture(), inScope, "", "", nil)
	exp := mustBuild(t, c, "expenses")
	if got := col(exp, "_id"); !reflect.DeepEqual(got, []interface{}{"t9", "t8"}) {
		t.Errorf("expenses = %v, want [t9 t8]", got)
	}
	rep := mustBuild(t, c, "gov-loan-repayments")
	if len(rep) != 1 || rep[0]["Lender"] != "Halmashauri ya Arusha" {
		t.Errorf("gov-loan-repayments = %v", rep)
	}
	approx(t, "gov repayment", rep[0]["Amount"], 250000)
}

// ---- formatting / exports ------------------------------------------------------

func TestCellTextTypes(t *testing.T) {
	cases := []struct {
		typ    string
		v      interface{}
		pretty bool
		want   string
	}{
		{"money", 1234567.0, true, "1,234,567"},
		{"money", 1234567.0, false, "1234567"},
		{"money", 12500.5, true, "12,500.50"},
		{"rate", 66.7, true, "66.7%"},
		{"count", 1200, true, "1,200"},
		{"int", 1200, true, "1200"},
		{"enum", "in_progress", false, "Inaendelea"},
		{"enum", "Weekly", false, "Kila Wiki"},
		{"text", "Active", false, "Active"}, // free text is never translated
		{"date", nil, false, ""},
		{"date", "2026-09-17", false, "2026-09-17"},
	}
	for _, c := range cases {
		if got := cellText("sw", c.typ, c.v, c.pretty); got != c.want {
			t.Errorf("cellText(%s, %v, %v) = %q, want %q", c.typ, c.v, c.pretty, got, c.want)
		}
	}
	if got := cellText("en", "enum", "in_progress", false); got != "In progress" {
		t.Errorf("english enum = %q", got)
	}
}

func sampleSets() []exportSet {
	when := time.Date(2026, 9, 16, 22, 0, 0, 0, time.UTC)
	rows := []datatype.DataMap{
		{"Date": when, "Group": "Umoja", "Type": "Mandatory Savings", "Amount": 5000.0},
		{"Date": nil, "Group": "Amani — B", "Type": "Shares", "Amount": 12500.5},
	}
	return []exportSet{
		{ds: findDataset("savings"), cols: []string{"Date", "Group", "Type", "Amount"}, rows: rows, total: 2},
		{ds: findDataset("members"), cols: []string{"Name", "Gender"}, rows: []datatype.DataMap{{"Name": "Asha", "Gender": "Female"}}, total: 1},
	}
}

func TestExportXLSX(t *testing.T) {
	data, err := exportXLSX(exportMeta{lang: "sw", asAt: "2026-09-25"}, sampleSets())
	if err != nil {
		t.Fatal(err)
	}
	f, err := excelize.OpenReader(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	if got := f.GetSheetList(); len(got) != 2 || got[0] != "Akiba" || got[1] != "Wanachama" {
		t.Fatalf("sheets = %v", got)
	}
	if v, _ := f.GetCellValue("Akiba", "B1"); v != "Kikundi" {
		t.Errorf("header = %q, want Kikundi", v)
	}
	if v, _ := f.GetCellValue("Akiba", "C2"); v != "Akiba ya Lazima" {
		t.Errorf("translated type = %q", v)
	}
	if v, _ := f.GetCellValue("Akiba", "D3", excelize.Options{RawCellValue: true}); v != "12500.5" {
		t.Errorf("amount = %q, want numeric 12500.5", v)
	}
	if v, _ := f.GetCellValue("Akiba", "D3"); v != "12,501" {
		t.Errorf("formatted amount = %q, want #,##0", v)
	}
	if v, _ := f.GetCellValue("Akiba", "A2"); v != "2026-09-17 01:00" {
		t.Errorf("datetime cell = %q, want EAT 2026-09-17 01:00", v)
	}
	if typ, _ := f.GetCellType("Akiba", "A2"); typ == excelize.CellTypeInlineString || typ == excelize.CellTypeSharedString {
		t.Error("date must be a real date cell, not text")
	}
	if v, _ := f.GetCellValue("Akiba", "A3"); v != "" {
		t.Errorf("nil date = %q, want an empty cell", v)
	}
	if hf, _ := f.GetHeaderFooter("Akiba"); hf == nil || !strings.Contains(hf.OddHeader, "HelaBox") {
		t.Errorf("printed page header missing: %+v", hf)
	}
	panes, _ := f.GetPanes("Akiba")
	if !panes.Freeze || panes.YSplit != 1 {
		t.Errorf("header row must be frozen: %+v", panes)
	}
	styleID, _ := f.GetCellStyle("Akiba", "A1")
	if st, _ := f.GetStyle(styleID); st == nil || st.Font == nil || !st.Font.Bold {
		t.Error("header must be bold")
	}
}

func TestExportCSVSingleZipAndNotes(t *testing.T) {
	sets := sampleSets()
	data, ext, err := exportCSV(exportMeta{lang: "en"}, sets[:1])
	if err != nil || ext != "csv" {
		t.Fatalf("single: ext=%q err=%v", ext, err)
	}
	if !strings.Contains(string(data), "2026-09-17 01:00,Umoja,Mandatory Savings,5000") {
		t.Errorf("csv body = %q", data)
	}
	cut := sets[0]
	cut.total = 70000
	data, _, _ = exportCSV(exportMeta{lang: "en"}, []exportSet{cut})
	if !strings.Contains(string(data), "# Only the first 2 of 70,000 rows are included.") {
		t.Errorf("truncation note missing: %q", data)
	}

	data, ext, err = exportCSV(exportMeta{lang: "en"}, sets)
	if err != nil || ext != "zip" {
		t.Fatalf("multi: ext=%q err=%v", ext, err)
	}
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil || len(zr.File) != 2 {
		t.Fatalf("zip files=%d err=%v", len(zr.File), err)
	}
}

func TestExportPDF(t *testing.T) {
	c := testCtx(t, reportFixture(), inScope, "", "", nil)
	ds := findDataset("summary-group")
	rows := mustBuild(t, c, ds.Key)
	wide := exportSet{ds: ds, cols: ds.Columns, rows: rows, totals: reportTotals(ds, rows), total: len(rows)}
	for _, lang := range []string{"sw", "en"} {
		data, err := exportPDF(exportMeta{lang: lang, scope: "Umoja", from: "2026-01-01", to: "2026-03-31", asAt: "2026-09-25", generated: testNow},
			append(sampleSets(), wide))
		if err != nil {
			t.Fatalf("%s: %v", lang, err)
		}
		if !bytes.HasPrefix(data, []byte("%PDF-")) || len(data) < 1000 {
			t.Fatalf("%s: not a PDF (%d bytes)", lang, len(data))
		}
		// REPORT_PDF_OUT=dir keeps the files for a visual check.
		if dir := os.Getenv("REPORT_PDF_OUT"); dir != "" {
			_ = os.WriteFile(filepath.Join(dir, "test_"+lang+".pdf"), data, 0o644)
		}
	}
	// A dataset with no rows must still render (with a "no data" note).
	empty := []exportSet{{ds: findDataset("loans"), cols: loanCols}}
	if _, err := exportPDF(exportMeta{lang: "sw"}, empty); err != nil {
		t.Fatal(err)
	}
}

func TestExportFilename(t *testing.T) {
	now := testNow
	if got := exportFilename("Umoja Wanawake — Arusha", []string{"savings"}, "2026-01-01", "2026-03-31", now, "xlsx"); got != "helabox-umoja-wanawake-arusha-savings-2026-01-01_2026-03-31.xlsx" {
		t.Errorf("filename = %q", got)
	}
	if got := exportFilename("", []string{"savings", "loans"}, "", "", now, "zip"); got != "helabox-all-multi-start_2026-09-25.zip" {
		t.Errorf("filename = %q", got)
	}
}

// ---- follow-up round (V1–V8) -------------------------------------------------

// V1: a TOTAL row always has a label, even when only totalled columns are picked.
func TestTotalsLabelWithOnlyNumericColumns(t *testing.T) {
	c := testCtx(t, reportFixture(), inScope, "", "", nil)
	ds := findDataset("summary-group")
	rows := mustBuild(t, c, ds.Key)
	set := exportSet{ds: ds, cols: []string{"Savings Balance", "PAR 30 %"}, rows: rows, totals: reportTotals(ds, rows), total: len(rows)}
	data, _, _ := exportCSV(exportMeta{lang: "sw", asAt: "2026-09-25"}, []exportSet{set})
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if last := strings.TrimSpace(lines[len(lines)-1]); last != "JUMLA,12100,75" {
		t.Errorf("totals row = %q, want JUMLA,12100,75 (label column put back)", last)
	}
	if !strings.Contains(string(data), "Jina,Salio la Akiba,PAR 30 %") {
		t.Errorf("the label column must be added in front: %q", data)
	}
	if _, err := exportXLSX(exportMeta{lang: "en"}, []exportSet{set}); err != nil {
		t.Fatal(err)
	}
	if _, err := exportPDF(exportMeta{lang: "en"}, []exportSet{set}); err != nil {
		t.Fatal(err)
	}
}

// V2: a member who moved to another group is still named in the old group's rows.
func TestMovedMemberStillNamed(t *testing.T) {
	fix := reportFixture()
	for _, m := range fix["Member"] {
		if m["id"] == "m2" {
			m["groupId"] = "g3" // John moved to Zeta, outside the scope
		}
	}
	c := testCtx(t, fix, inScope, "", "", nil)
	seen := 0
	for _, r := range mustBuild(t, c, "loans") {
		if r["_id"] == "l1" {
			seen++
			if r["Borrower"] != "John Mfinanga" {
				t.Errorf("borrower of l1 = %v, want John Mfinanga", r["Borrower"])
			}
		}
	}
	for _, r := range mustBuild(t, c, "transactions") {
		if r["_id"] == "t7" {
			seen++
			if r["Member"] != "John Mfinanga" {
				t.Errorf("member of t7 = %v", r["Member"])
			}
		}
	}
	if seen != 2 {
		t.Fatalf("rows l1/t7 missing (%d)", seen)
	}
	for _, r := range mustBuild(t, c, "members") {
		if r["_id"] == "m2" {
			t.Error("the members list shows members of the groups in scope only")
		}
	}
}

// V3: money lists have totals of their money columns only.
func TestMoneyListTotals(t *testing.T) {
	c := testCtx(t, reportFixture(), inScope, "", "", nil)
	for _, key := range []string{"savings", "shares", "social-fund", "fines", "fines-outstanding", "expenses", "loans", "government-loans", "gov-loan-repayments"} {
		ds := findDataset(key)
		if !ds.Totals {
			t.Errorf("%s should have a TOTAL row", key)
			continue
		}
		tot := reportTotals(ds, mustBuild(t, c, key))
		if len(tot) == 0 {
			t.Errorf("%s has no totals", key)
		}
		for col := range tot {
			if colType(ds, col) != "money" {
				t.Errorf("%s totals %q (%s): only money columns are summed", key, col, colType(ds, col))
			}
		}
	}
	sav := reportTotals(findDataset("savings"), mustBuild(t, c, "savings"))
	approx(t, "savings total", sav["Amount"], 13100) // 5000+5000+3000+100; reversed 7000 out
	if _, ok := sav["Date"]; ok {
		t.Error("dates are never totalled")
	}
	loans := reportTotals(findDataset("loans"), mustBuild(t, c, "loans"))
	approx(t, "loans principal total (cancelled 20000 left out)", loans["Principal"], 88000)
	approx(t, "loans balance total", loans["Balance"], 60000)
	if _, ok := loans["Interest %"]; ok {
		t.Error("an interest rate is not totalled")
	}
	fo := reportTotals(findDataset("fines-outstanding"), mustBuild(t, c, "fines-outstanding"))
	approx(t, "fines outstanding total", fo["Outstanding"], 1000)
}

// V6: over-repaid loans are flagged.
func TestOverpaidLoans(t *testing.T) {
	c := testCtx(t, reportFixture(), inScope, "", "", nil)
	for _, r := range mustBuild(t, c, "loans") {
		want := 0.0
		if r["_id"] == "l6" {
			want = 1000
		}
		approx(t, "overpaid "+toStr(r["_id"]), r["Overpaid"], want)
	}
	fix := reportFixture()
	over := overpaidLoans(fix["Loan"], indexById(fix["Group"]))
	if len(over) != 1 || over[0]["loanId"] != "l6" || num(over[0]["overpaid"]) != 1000 || over[0]["groupName"] != "Umoja" {
		t.Errorf("overpaid loans = %v", over)
	}
}

// V7: status on summaries; groups with no cluster/partner are "Not assigned".
func TestSummaryStatusAndUnassigned(t *testing.T) {
	fix := reportFixture()
	fix["Cluster"][1]["status"] = "inactive"
	fix["Group"] = append(fix["Group"], datatype.DataMap{"id": "g4", "name": "Loose", "status": "Active"}) // no cluster
	set := map[string]bool{"g1": true, "g2": true, "g4": true}
	c := testCtx(t, fix, set, "", "", nil)
	cl := mustBuild(t, c, "summary-cluster")
	if r := byName(cl, "Name", "Moshi"); r == nil || r["Status"] != "Inactive" {
		t.Errorf("inactive cluster status: %v", r)
	}
	if r := byName(cl, "Name", "Arusha"); r == nil || r["Status"] != "Active" {
		t.Errorf("active cluster status: %v", r)
	}
	un := byName(cl, "Name", notAssigned)
	if un == nil || un["Status"] != nil || num(un["Groups"]) != 1 {
		t.Fatalf("unassigned row = %v", un)
	}
	if got := cellText("sw", "text", un["Name"], false); got != "Haijapangwa" {
		t.Errorf("sw label = %q", got)
	}
	if got := cellText("sw", "enum", "Inactive", false); got != "Haifanyi kazi" {
		t.Errorf("sw status = %q", got)
	}
	pr := mustBuild(t, c, "summary-partner")
	if byName(pr, "Name", notAssigned) == nil || byName(pr, "Name", "GATA Partner")["Status"] != "Active" {
		t.Errorf("summary-partner = %v", pr)
	}
	rows := rollupRows(c.groupKPIs(), "cluster", map[string]string{"c1": "Arusha", "c2": "Moshi"})
	flagged := 0
	for _, r := range rows {
		if r["unassigned"] == true {
			flagged++
			if r["id"] != "" {
				t.Errorf("unassigned row id = %v", r["id"])
			}
		}
	}
	if flagged != 1 {
		t.Errorf("roll-up unassigned rows = %d, want 1", flagged)
	}
}

// V8 / V5: CSV notes go before the header; a zip carries them in README.txt.
func TestCSVNotesPlacement(t *testing.T) {
	c := testCtx(t, reportFixture(), inScope, "", "", nil)
	ds := findDataset("meeting-collections")
	rows := mustBuild(t, c, ds.Key)
	set := exportSet{ds: ds, cols: ds.Columns, rows: rows, totals: reportTotals(ds, rows), total: len(rows) + 5}
	data, _, _ := exportCSV(exportMeta{lang: "en"}, []exportSet{set})
	body := strings.TrimPrefix(string(data), "\xEF\xBB\xBF")
	lines := strings.Split(body, "\r\n")
	if !strings.HasPrefix(lines[0], "# Money recorded without a meeting") || !strings.HasPrefix(lines[1], "# Only the first") {
		t.Errorf("notes must come first: %q", lines[:3])
	}
	if !strings.HasPrefix(lines[2], "Group,Meeting #") {
		t.Errorf("header after the notes: %q", lines[2])
	}
	for _, l := range lines[3:] {
		if strings.HasPrefix(l, "# ") {
			t.Errorf("no notes after the data: %q", l)
		}
	}

	zipped, ext, _ := exportCSV(exportMeta{lang: "en"}, []exportSet{set, sampleSets()[1]})
	if ext != "zip" {
		t.Fatal(ext)
	}
	zr, _ := zip.NewReader(bytes.NewReader(zipped), int64(len(zipped)))
	var readme string
	for _, f := range zr.File {
		rc, _ := f.Open()
		b := new(bytes.Buffer)
		_, _ = b.ReadFrom(rc)
		rc.Close()
		if f.Name == "README.txt" {
			readme = b.String()
		} else if strings.Contains(b.String(), "\n# ") || strings.HasPrefix(strings.TrimPrefix(b.String(), "\xEF\xBB\xBF"), "# ") {
			t.Errorf("%s must not carry # notes inside a zip", f.Name)
		}
	}
	if !strings.Contains(readme, "meeting-collections.csv") || !strings.Contains(readme, "Only the first") || !strings.Contains(readme, "same day") {
		t.Errorf("README.txt = %q", readme)
	}
}
