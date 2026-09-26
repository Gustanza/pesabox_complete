package main

import (
	"strings"
	"testing"

	"github.com/robertkonga/yekonga-server-go/helper"

	"github.com/robertkonga/yekonga-server-go/datatype"
)

func rulesGroup() datatype.DataMap {
	return datatype.DataMap{
		"id": "g1", "name": "Umoja", "shareValue": 5000.0, "minShares": 1, "maxShares": 5,
		"socialFundContribution": 2000.0, "mandatorySavingsAmount": 5000.0, "loanInterestRate": 10.0,
		"maxLoanPeriodMonths": 3, "enabledServices": []interface{}{"Shares", "Mandatory Savings", "Social Fund", "Loans", "Fines"},
	}
}

func TestValidateRules(t *testing.T) {
	g := rulesGroup()
	bad := []struct {
		body map[string]interface{}
		want string
	}{
		{map[string]interface{}{"loanInterestRate": 150.0}, "cannot be more than 100"},
		{map[string]interface{}{"loanInterestRate": -1.0}, "cannot be less than 0"},
		{map[string]interface{}{"shareValue": "5000"}, "must be a number"},
		{map[string]interface{}{"maxLoanPeriodMonths": 0.0}, "cannot be less than 1"},
		{map[string]interface{}{"maxLoanPeriodMonths": 37.0}, "cannot be more than 36"},
		{map[string]interface{}{"maxShares": 2.5}, "whole number"},
		{map[string]interface{}{"minShares": 6.0}, "minimum shares (6) cannot be more than maximum shares (5)"},
		{map[string]interface{}{"fineReasons": []interface{}{map[string]interface{}{"reason": " ", "amount": 1.0}}}, "needs a name"},
		{map[string]interface{}{"fineReasons": []interface{}{
			map[string]interface{}{"reason": "Late", "amount": 1.0}, map[string]interface{}{"reason": "late", "amount": 2.0}}}, "listed twice"},
		{map[string]interface{}{"fineReasons": []interface{}{map[string]interface{}{"reason": "Late", "amount": -5.0}}}, "cannot be negative"},
		{map[string]interface{}{"enabledServices": []interface{}{"Shares", "Casino"}}, "unknown service"},
		{map[string]interface{}{"totalSavings": 1.0}, "unknown rule"},
		{map[string]interface{}{}, "no rules"},
	}
	for _, c := range bad {
		if _, err := validateRules(g, c.body); err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("validateRules(%v) = %v, want error containing %q", c.body, err, c.want)
		}
	}

	changes, err := validateRules(g, map[string]interface{}{"rules": map[string]interface{}{
		"loanInterestRate": 12.0, "shareValue": 5000.0, "maxLoanMultiplier": 3.0,
		"fineReasons":     []interface{}{map[string]interface{}{"reason": " Late Attendance ", "amount": 1500.0}},
		"enabledServices": []interface{}{"loans", "Shares"},
	}})
	if err != nil {
		t.Fatal(err)
	}
	if changes["loanInterestRate"] != 12.0 || changes["maxLoanMultiplier"] != 3.0 {
		t.Errorf("changes = %v", changes)
	}
	if _, same := changes["shareValue"]; same {
		t.Error("an unchanged value is not a change")
	}
	if got := changes["enabledServices"].([]string); strings.Join(got, ",") != "Shares,Loans" {
		t.Errorf("services canonicalised = %v", got)
	}
	if fr := changes["fineReasons"].([]datatype.DataMap); fr[0]["reason"] != "Late Attendance" {
		t.Errorf("fine reason trimmed = %v", fr)
	}
	diff, summary := rulesDiff(groupRules(g), changes)
	if !strings.Contains(summary, "loanInterestRate 10 → 12") || diff["loanInterestRate"].(datatype.DataMap)["from"] != 10.0 {
		t.Errorf("audit diff = %v / %q", diff, summary)
	}
}

// Only the Mwenyekiti (group admin), Super Admin and Operations staff may
// change a group's rules.
func TestRulesPermissionPerRole(t *testing.T) {
	scope := map[string]bool{"g1": true}
	cases := []struct {
		name string
		a    *Actor
		want bool
	}{
		{"super admin", &Actor{Perms: platformPerms(RoleSuperAdmin, ""), All: true}, true},
		{"operations staff", &Actor{Perms: platformPerms(RoleStaff, PresetOperations), GroupIDs: scope}, true},
		{"support staff", &Actor{Perms: platformPerms(RoleStaff, PresetSupport), GroupIDs: scope}, false},
		{"viewer staff", &Actor{Perms: platformPerms(RoleStaff, PresetViewer), GroupIDs: scope}, false},
		{"partner user", &Actor{Perms: platformPerms(RolePartner, ""), GroupIDs: scope}, false},
		{"cluster manager", &Actor{Perms: platformPerms(RoleCluster, ""), GroupIDs: scope}, false},
		{"mwenyekiti (group admin)", &Actor{Perms: map[string]bool{}, HomeGroupID: "g1", HomePerms: groupAdminPerms}, true},
		{"katibu (group officer)", &Actor{Perms: map[string]bool{}, HomeGroupID: "g1", HomePerms: groupOfficerPerms}, false},
		{"operations staff, other group", &Actor{Perms: platformPerms(RoleStaff, PresetOperations), GroupIDs: map[string]bool{"g2": true}}, false},
	}
	for _, c := range cases {
		if got := c.a.CanIn(PermGroupSettings, "g1"); got != c.want {
			t.Errorf("%s: CanIn(group.settings) = %v, want %v", c.name, got, c.want)
		}
	}
}

func TestLoanTermsLegacyVsNew(t *testing.T) {
	interest, total := loanTerms(100000, 10)
	if interest != 10000 || total != 110000 {
		t.Errorf("flat interest = %v / %v", interest, total)
	}
	legacy := datatype.DataMap{"amount": 50000.0, "amountRepaid": 10000.0, "interestRate": 10.0, "status": "active"}
	fresh := datatype.DataMap{"amount": 100000.0, "amountRepaid": 10000.0, "interestRate": 10.0, "interestAmount": 10000.0, "totalDue": 110000.0, "status": "active"}
	if loanTotalDue(legacy) != 50000 || loanBalance(legacy) != 40000 || loanInterest(legacy) != 0 {
		t.Errorf("a legacy loan owes its principal only: due %v balance %v", loanTotalDue(legacy), loanBalance(legacy))
	}
	if loanTotalDue(fresh) != 110000 || loanBalance(fresh) != 100000 || loanInterest(fresh) != 10000 {
		t.Errorf("a new loan owes principal + interest: due %v balance %v", loanTotalDue(fresh), loanBalance(fresh))
	}
	fresh["amountRepaid"] = 111000.0
	if loanOverpaid(fresh) != 1000 || loanBalance(fresh) != 0 {
		t.Errorf("overpaid is measured against the total due: %v", loanOverpaid(fresh))
	}
}

func TestRepaymentCapAtTotalDue(t *testing.T) {
	loan := datatype.DataMap{"amount": 100000.0, "amountRepaid": 90000.0, "totalDue": 110000.0, "status": "active"}
	if _, _, msg := loanRepayment(loan, 20001); !strings.Contains(msg, "exceeds the remaining balance (TZS 20,000)") {
		t.Errorf("over-repayment message = %q", msg)
	}
	repaid, status, msg := loanRepayment(loan, 15000)
	if msg != "" || repaid != 105000 || status != "active" {
		t.Errorf("part repayment past the principal = %v %v %q", repaid, status, msg)
	}
	repaid, status, _ = loanRepayment(loan, 20000)
	if repaid != 110000 || status != "repaid" {
		t.Errorf("full repayment = %v %v", repaid, status)
	}
	legacy := datatype.DataMap{"amount": 50000.0, "amountRepaid": 45000.0, "status": "active"}
	if _, _, msg := loanRepayment(legacy, 6000); msg == "" {
		t.Error("a legacy loan caps at its principal")
	}
	if _, status, _ := loanRepayment(legacy, 5000); status != "repaid" {
		t.Error("a legacy loan is repaid at its principal")
	}
	if _, _, msg := loanRepayment(datatype.DataMap{"amount": 1.0, "status": "cancelled"}, 1); msg == "" {
		t.Error("a cancelled loan takes no repayments")
	}
}

// W4: (savings + shares − withdrawals) × multiplier, less what is still owed.
func TestLoanMultiplierLimit(t *testing.T) {
	g := rulesGroup()
	txs := []datatype.DataMap{
		{"memberId": "m1", "type": "contribution", "amount": 20000.0},
		{"memberId": "m1", "type": "share", "amount": 10000.0},
		{"memberId": "m1", "type": "contribution", "amount": 50000.0, "reversed": true},
		{"memberId": "m1", "type": "withdrawal", "amount": 5000.0},
		{"memberId": "m2", "type": "share", "amount": 99000.0},
	}
	base := memberSavingsShares(txs, "m1")
	if base != 25000 {
		t.Fatalf("savings + shares − withdrawals = %v, want 25000", base)
	}
	loans := []datatype.DataMap{
		{"memberId": "m1", "amount": 20000.0, "amountRepaid": 12000.0, "totalDue": 22000.0, "status": "active"}, // owes 10000
		{"memberId": "m1", "amount": 9000.0, "amountRepaid": 0.0, "status": "cancelled"},                        // owes nothing
		{"memberId": "m2", "amount": 50000.0, "amountRepaid": 0.0, "status": "active"},
	}
	owed := memberOwed(loans, "m1")
	if owed != 10000 {
		t.Fatalf("owed = %v, want 10000", owed)
	}
	if msg := loanLimitError(g, 1000000, base, owed); msg != "" || availableToBorrow(g, base, owed) != -1 {
		t.Errorf("no multiplier = no limit, got %q", msg)
	}
	g["maxLoanMultiplier"] = 3.0
	if avail := availableToBorrow(g, base, owed); avail != 65000 {
		t.Errorf("available = %v, want 3×25000 − 10000 = 65000", avail)
	}
	if msg := loanLimitError(g, 65000, base, owed); msg != "" {
		t.Errorf("65000 is exactly what is available: %q", msg)
	}
	want := "3× savings and shares (TZS 75,000) less TZS 10,000 still owed leaves TZS 65,000 available"
	if msg := loanLimitError(g, 65001, base, owed); !strings.Contains(msg, want) {
		t.Errorf("limit message = %q, want it to contain %q", msg, want)
	}
	if availableToBorrow(g, 1000, 50000) != 0 {
		t.Error("available never goes below 0")
	}
}

// W2: the plain transaction route takes savings, shares, social fund and
// withdrawals only.
func TestTransactionTypesAllowed(t *testing.T) {
	for _, typ := range []string{"contribution", "share", "social_fund", "withdrawal"} {
		if msg := txTypeError(typ); msg != "" {
			t.Errorf("%s refused: %q", typ, msg)
		}
	}
	for typ, want := range map[string]string{
		"loan_disbursement": "/api/main/loans)",
		"loan_repayment":    "/api/main/loans/:id/repayment",
		"fine":              "/api/main/fines/:id/pay",
		"expense":           "/api/main/expenses",
		"membership_fee":    "unknown transaction type",
		"bogus":             "unknown transaction type",
	} {
		if msg := txTypeError(typ); !strings.Contains(msg, want) {
			t.Errorf("%s: %q, want it to name %q", typ, msg, want)
		}
	}
}

// W8b: mandatory vs voluntary savings.
func TestContributionKind(t *testing.T) {
	g := rulesGroup() // Mandatory on (5000), Voluntary off
	if kind, msg := contributionKind(g, "", 5000); kind != "mandatory" || msg != "" {
		t.Errorf("default = %q %q", kind, msg)
	}
	if _, msg := contributionKind(g, "", 3000); !strings.Contains(msg, "Mandatory Savings is TZS 5,000 per meeting") {
		t.Errorf("mandatory amount check: %q", msg)
	}
	if _, msg := contributionKind(g, "voluntary", 3000); !strings.Contains(msg, "Voluntary Savings is switched off") {
		t.Errorf("voluntary while off: %q", msg)
	}
	g["enabledServices"] = []interface{}{"Mandatory Savings", "Voluntary Savings"}
	if kind, msg := contributionKind(g, "voluntary", 3000); kind != "voluntary" || msg != "" {
		t.Errorf("voluntary is exempt from the fixed amount: %q %q", kind, msg)
	}
	if _, msg := contributionKind(g, "", 3000); msg == "" {
		t.Error("without the flag the mandatory check still applies")
	}
	g["enabledServices"] = []interface{}{"Voluntary Savings"}
	if kind, msg := contributionKind(g, "", 1234); kind != "voluntary" || msg != "" {
		t.Errorf("only voluntary on: %q %q", kind, msg)
	}
	if _, msg := contributionKind(g, "mandatory", 5000); !strings.Contains(msg, "Mandatory Savings is switched off") {
		t.Errorf("mandatory while off: %q", msg)
	}
	if _, msg := contributionKind(g, "weekly", 5000); !strings.Contains(msg, "mandatory or voluntary") {
		t.Errorf("bad kind: %q", msg)
	}
}

// W6 / W7: money rules and service-dependent rules.
func TestRulesMoneyAndServiceChecks(t *testing.T) {
	g := rulesGroup()
	bad := []struct {
		body map[string]interface{}
		want string
	}{
		{map[string]interface{}{"shareValue": 5000.5}, "Share value must be a whole number of shillings"},
		{map[string]interface{}{"mandatorySavingsAmount": 100000001.0}, "cannot be more than 100000000"},
		{map[string]interface{}{"fineReasons": []interface{}{map[string]interface{}{"reason": "Late", "amount": 10.5}}}, "whole number of shillings"},
		{map[string]interface{}{"shareValue": 0.0}, "share value must be more than 0 while Shares is switched on"},
		{map[string]interface{}{"maxShares": 0.0, "minShares": 0.0}, "maximum shares per meeting must be at least 1 while Shares is switched on"},
		{map[string]interface{}{"socialFundContribution": 0.0}, "social fund contribution must be more than 0 while Social Fund is switched on"},
		{map[string]interface{}{"mandatorySavingsAmount": 0.0}, "mandatory savings amount must be more than 0 while Mandatory Savings is switched on"},
		{map[string]interface{}{"fineReasons": []interface{}{}}, "add at least one fine reason while Fines is switched on"},
	}
	for _, c := range bad {
		if _, err := validateRules(g, c.body); err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("validateRules(%v) = %v, want %q", c.body, err, c.want)
		}
	}
	// With the service off, the same values are fine.
	changes, err := validateRules(g, map[string]interface{}{
		"enabledServices": []interface{}{"Mandatory Savings", "Loans"}, "shareValue": 0.0, "socialFundContribution": 0.0, "fineReasons": []interface{}{},
	})
	if err != nil {
		t.Fatal(err)
	}
	g2 := datatype.DataMap{}
	for k, v := range g {
		g2[k] = v
	}
	for k, v := range changes {
		g2[k] = v
	}
	g2["fineReasons"] = []interface{}{} // stored empty
	if fr := groupFineReasons(g2); len(fr) != 0 {
		t.Errorf("an explicitly empty list must not show the defaults: %v", fr)
	}
	if fr := groupFineReasons(datatype.DataMap{}); len(fr) != 5 {
		t.Errorf("never configured = the 5 defaults, got %d", len(fr))
	}
}

// W1: a meeting can repeat its own groupId but never move.
func TestMeetingMoveRefused(t *testing.T) {
	target := []datatype.DataMap{{"id": "mt1", "groupId": "g1"}}
	if meetingMoveRefused(datatype.DataMap{"status": "completed"}, target) {
		t.Error("no groupId in the input is fine")
	}
	if meetingMoveRefused(datatype.DataMap{"groupId": "g1", "status": "completed"}, target) {
		t.Error("the app's close call repeats its own groupId")
	}
	if !meetingMoveRefused(datatype.DataMap{"groupId": "g2"}, target) {
		t.Error("moving a meeting to another group must be refused")
	}
	if !meetingMoveRefused(datatype.DataMap{"groupId": ""}, target) || !meetingMoveRefused(datatype.DataMap{"groupId": nil}, target) {
		t.Error("clearing the groupId must be refused")
	}
}

// W3: the stored totalLoans running total is part of the drift check.
func TestDriftIncludesTotalLoans(t *testing.T) {
	d := groupTotalsDrift(datatype.DataMap{"totalLoans": 100000.0}, groupKPI{LoansOutstanding: 110000})
	if _, ok := d["totalLoans"]; !ok {
		t.Errorf("drift = %v, want totalLoans", d)
	}
}

// W10: a cancelled loan shows no interest / total due.
func TestCancelledLoanTermsBlank(t *testing.T) {
	fix := reportFixture()
	fix["Loan"][1]["totalDue"] = 22000.0 // l2 (cancelled) had interest charged
	c := testCtx(t, fix, inScope, "", "", nil)
	for _, r := range mustBuild(t, c, "loans") {
		if r["_id"] == "l2" && (r["Interest"] != nil || r["Total Due"] != nil) {
			t.Errorf("cancelled loan terms = %v / %v, want blank", r["Interest"], r["Total Due"])
		}
	}
}

func TestDisabledServiceRefused(t *testing.T) {
	g := rulesGroup()
	for _, typ := range []string{"contribution", "share", "social_fund", "loan_disbursement", "loan_repayment", "fine", "expense"} {
		if msg := serviceOff(g, typ); msg != "" {
			t.Errorf("%s refused while its service is on: %q", typ, msg)
		}
	}
	g["enabledServices"] = []interface{}{"Voluntary Savings"}
	if msg := serviceOff(g, "contribution"); msg != "" {
		t.Errorf("voluntary savings allow contributions: %q", msg)
	}
	for typ, want := range map[string]string{"share": "Shares is switched off", "social_fund": "Social Fund is switched off", "loan_disbursement": "Loans is switched off"} {
		if msg := serviceOff(g, typ); !strings.Contains(msg, want) {
			t.Errorf("%s: %q, want %q", typ, msg, want)
		}
	}
	for _, typ := range []string{"loan_repayment", "fine", "expense", "withdrawal"} {
		if msg := serviceOff(g, typ); msg != "" {
			t.Errorf("%s must stay possible (existing loans / fines): %q", typ, msg)
		}
	}
	if serviceEnabled(g, "Fines") {
		t.Error("Fines is off")
	}
	if !serviceEnabled(datatype.DataMap{}, "Loans") {
		t.Error("a group without enabledServices uses the defaults")
	}
}

func TestRulesLockedOverGraphQL(t *testing.T) {
	locked := map[string]bool{}
	for _, f := range ruleFields {
		locked[f] = true
	}
	for _, f := range []string{"shareValue", "loanInterestRate", "enabledServices", "fineReasons", "maxLoanMultiplier", "lateMeetingFine"} {
		if !locked[f] {
			t.Errorf("%s must be locked out of GraphQL", f)
		}
	}
}

// Reports, KPIs and PAR use the loan's total due; a legacy loan does not
// start owing interest.
func TestReportsUseTotalDue(t *testing.T) {
	fix := reportFixture()
	fix["Loan"] = append(fix["Loan"], datatype.DataMap{
		"id": "l7", "groupId": "g2", "memberId": "m4", "loanNumber": "LN-0003", "amount": 100000.0, "amountRepaid": 0.0,
		"interestRate": 12.0, "interestAmount": 12000.0, "totalDue": 112000.0, "status": "active",
		"issuedDate": "2026-05-01T08:00:00Z", "dueDate": "2026-07-01T08:00:00Z",
	})
	c := testCtx(t, fix, inScope, "", "", nil)
	for _, r := range mustBuild(t, c, "loans") {
		switch r["_id"] {
		case "l7":
			approx(t, "new loan interest", r["Interest"], 12000)
			approx(t, "new loan total due", r["Total Due"], 112000)
			approx(t, "new loan balance", r["Balance"], 112000)
		case "l1":
			approx(t, "legacy loan interest", r["Interest"], 0)
			approx(t, "legacy loan total due", r["Total Due"], 50000)
			approx(t, "legacy loan balance", r["Balance"], 40000)
		}
	}
	amani := byName(mustBuild(t, c, "summary-group"), "Name", "Amani")
	approx(t, "Amani outstanding (15000 + 112000)", amani["Loans Outstanding"], 127000)
	approx(t, "Amani PAR % (112000 / 127000)", amani["PAR 30 %"], 88.2)
	par := mustBuild(t, c, "portfolio-at-risk")
	for _, r := range par {
		if r["_id"] == "l7" {
			approx(t, "PAR balance includes interest", r["Balance"], 112000)
		}
	}
}

// Fine reasons stored in Mongo come back as ordered key/value documents.
func TestFineReasonsFromStoredDocuments(t *testing.T) {
	type kv struct {
		Key   string
		Value interface{}
	}
	type array []interface{} // like the driver's own array type
	g := datatype.DataMap{"fineReasons": array{
		[]kv{{"reason", "Late Attendance"}, {"amount", 1500.0}},
		map[string]interface{}{"reason": "Absent", "amount": 2500.0},
	}}
	got := groupFineReasons(g)
	if len(got) != 2 || got[0]["reason"] != "Late Attendance" || fineAmountFor(g, "Late Attendance", 0) != 1500 || fineAmountFor(g, "Absent", 0) != 2500 {
		t.Errorf("fine reasons = %v", got)
	}
}

// W1: the id may arrive as an ObjectID (the framework converts ID fields).
func TestMeetingMoveRefusedObjectID(t *testing.T) {
	oid := helper.ObjectID("6ab66062023cc96de1930b6d")
	target := []datatype.DataMap{{"id": "mt1", "groupId": "6ab66062023cc96de1930b6d"}}
	if meetingMoveRefused(datatype.DataMap{"groupId": oid}, target) {
		t.Error("the meeting's own groupId as an ObjectID must be accepted")
	}
	if !meetingMoveRefused(datatype.DataMap{"groupId": helper.ObjectID("6ab66062023cc96de1930b6f")}, target) {
		t.Error("another group's ObjectID must be refused")
	}
}
