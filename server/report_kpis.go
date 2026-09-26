package main

import (
	"math"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/robertkonga/yekonga-server-go/datatype"
	"github.com/robertkonga/yekonga-server-go/helper"
	"github.com/robertkonga/yekonga-server-go/yekonga"
)

// ---------------------------------------------------------------------------
// Programme KPIs, computed per group and rolled up group → cluster → partner →
// organisation (TODO.md §5, D4). One computation feeds the live dashboard,
// the roll-up screens and the "summary-*" / "group-performance" report
// datasets, so the numbers always agree with each other.
//
// Money balances are computed from the non-reversed transactions and the
// non-cancelled loans, never from the Group's stored running totals
// (totalSavings, totalFines, ...), which can drift. See groupTotalsDrift.
// ---------------------------------------------------------------------------

// inactiveAfter: a group with no meeting for this long is flagged inactive.
const inactiveAfter = 30 * 24 * time.Hour

// parDays: a loan this many days past its due date counts as "at risk" (PAR 30).
const parDays = 30

// periodTotals are the flows inside the report's date range (the whole
// history when no range is set).
type periodTotals struct {
	Savings, Shares, SocialFund, Fines float64
	LoanRepayments, LoansDisbursed     float64
	Expenses, Withdrawals              float64
	Meetings, NewMembers               int
	AttendancePresent, AttendanceRows  int
}

func (p *periodTotals) add(o periodTotals) {
	p.Savings += o.Savings
	p.Shares += o.Shares
	p.SocialFund += o.SocialFund
	p.Fines += o.Fines
	p.LoanRepayments += o.LoanRepayments
	p.LoansDisbursed += o.LoansDisbursed
	p.Expenses += o.Expenses
	p.Withdrawals += o.Withdrawals
	p.Meetings += o.Meetings
	p.NewMembers += o.NewMembers
	p.AttendancePresent += o.AttendancePresent
	p.AttendanceRows += o.AttendanceRows
}

type groupKPI struct {
	GroupID, Name, ClusterID, PartnerID, Region string
	Status                                      string
	Members, MembersTotal, Female, Male         int // Members/Female/Male = active members only
	// Balances as at now, from every non-reversed transaction.
	Savings, Shares, SocialFund, Fines float64
	Expenses, Withdrawals              float64
	LoansOutstanding, PAR              float64
	GovLoansOutstanding                float64
	AttendancePresent, AttendanceRows  int
	LastMeeting                        time.Time
	Active                             bool
	Period                             periodTotals
}

type kpiTotals struct {
	Groups, ActiveGroups          int
	Members, MembersTotal         int
	Female, Male                  int
	Savings, Shares, SocialFund   float64
	Fines, Expenses, Withdrawals  float64
	LoansOutstanding, PAR         float64
	GovLoansOutstanding           float64
	AttendancePresent, AttendRows int
	Period                        periodTotals
}

func (t *kpiTotals) add(k groupKPI) {
	t.Groups++
	if k.Active {
		t.ActiveGroups++
	}
	t.Members += k.Members
	t.MembersTotal += k.MembersTotal
	t.Female += k.Female
	t.Male += k.Male
	t.Savings += k.Savings
	t.Shares += k.Shares
	t.SocialFund += k.SocialFund
	t.Fines += k.Fines
	t.Expenses += k.Expenses
	t.Withdrawals += k.Withdrawals
	t.LoansOutstanding += k.LoansOutstanding
	t.PAR += k.PAR
	t.GovLoansOutstanding += k.GovLoansOutstanding
	t.AttendancePresent += k.AttendancePresent
	t.AttendRows += k.AttendanceRows
	t.Period.add(k.Period)
}

func pct(part, whole float64) float64 {
	if whole <= 0 {
		return 0
	}
	return math.Round(part/whole*1000) / 10
}

func (t kpiTotals) json() datatype.DataMap {
	return datatype.DataMap{
		"groups":              t.Groups,
		"activeGroups":        t.ActiveGroups,
		"inactiveGroups":      t.Groups - t.ActiveGroups,
		"members":             t.Members,
		"membersTotal":        t.MembersTotal,
		"femaleMembers":       t.Female,
		"maleMembers":         t.Male,
		"savings":             t.Savings,
		"shares":              t.Shares,
		"socialFund":          t.SocialFund,
		"fines":               t.Fines,
		"expenses":            t.Expenses,
		"withdrawals":         t.Withdrawals,
		"loansOutstanding":    t.LoansOutstanding,
		"par30":               t.PAR,
		"par30Rate":           pct(t.PAR, t.LoansOutstanding),
		"govLoansOutstanding": t.GovLoansOutstanding,
		"attendanceRate":      pct(float64(t.AttendancePresent), float64(t.AttendRows)),
	}
}

// ---- shared business rules -------------------------------------------------

// memberActive is the one "active member" rule every report uses: Member.status
// is Active (the schema default, so a missing status counts as Active).
// Inactive and Suspended members are not active.
func memberActive(m datatype.DataMap) bool {
	st := strings.TrimSpace(helper.GetValueOfString(m, "status"))
	return st == "" || strings.EqualFold(st, "Active")
}

// loanOpen: a loan still owes money unless it is repaid, completed or
// cancelled (so "defaulted" loans stay in Outstanding and PAR).
func loanOpen(l datatype.DataMap) bool {
	switch strings.ToLower(helper.GetValueOfString(l, "status")) {
	case "repaid", "completed", "cancelled":
		return false
	}
	return true
}

// loanOverpaid is how much more than the total due was repaid (0 normally).
// The balance is clamped at 0, so this is the only place it shows.
func loanOverpaid(l datatype.DataMap) float64 {
	return math.Max(0, helper.GetValueOfFloat(l, "amountRepaid")-loanTotalDue(l))
}

// notAssigned names the roll-up row of groups with no cluster / partner.
const notAssigned = "Not assigned"

// unitStatus is a partner's / cluster's status for reports (Active/Inactive).
func unitStatus(u datatype.DataMap) string {
	if strings.EqualFold(helper.GetValueOfString(u, "status"), "inactive") {
		return "Inactive"
	}
	return "Active"
}

// loanBalance is what an open loan still owes: its total due (principal +
// the flat interest fixed at issue; principal only for older loans) less
// what was repaid, never negative; 0 for a closed or cancelled loan.
func loanBalance(l datatype.DataMap) float64 {
	if !loanOpen(l) {
		return 0
	}
	return math.Max(0, loanTotalDue(l)-helper.GetValueOfFloat(l, "amountRepaid"))
}

// txCounted is false for reversed transactions — they never count anywhere.
func txCounted(t datatype.DataMap) bool { return !helper.GetValueOfBoolean(t, "reversed") }

// meetingWhen is a meeting's scheduled date (EAT), falling back to createdAt.
func meetingWhen(mt datatype.DataMap) time.Time {
	if d, ok := parseMeetingDate(helper.GetValueOfString(mt, "date")); ok {
		return d
	}
	return parseTimeOrZero(mt["createdAt"])
}

// attendanceGroup is the attendance row's group, falling back to its
// meeting's group (older rows were written without groupId).
func attendanceGroup(a datatype.DataMap, meetings map[string]datatype.DataMap) string {
	if gid := helper.GetValueOfString(a, "groupId"); gid != "" {
		return gid
	}
	return helper.GetValueOfString(meetings[helper.GetValueOfString(a, "meetingId")], "groupId")
}

func attended(status string) bool { return status == "present" || status == "late" }

// kpiData is every record the KPI computation reads, loaded once per request.
type kpiData struct {
	Groups, Clusters, Members, Loans, GovLoans []datatype.DataMap
	Meetings, Attendance, Transactions         []datatype.DataMap
}

// loadKPIData reads what computeKPIs needs, limited to the group set (nil =
// every group) in the query itself.
func loadKPIData(app *yekonga.YekongaData, set map[string]bool) kpiData {
	q := func(model string) []datatype.DataMap {
		return reportQuery(app, model, set, "", time.Time{}, time.Time{})
	}
	return kpiData{
		Groups: q("Group"), Clusters: listAll(app, "Cluster"), Members: q("Member"), Loans: q("Loan"),
		GovLoans: q("GovernmentLoan"), Meetings: q("Meeting"), Attendance: q("MeetingAttendance"),
		Transactions: q("Transaction"),
	}
}

// computeGroupKPIs returns one groupKPI per group in set (nil set = all),
// with period flows for the whole history.
func computeGroupKPIs(app *yekonga.YekongaData, set map[string]bool) []groupKPI {
	return computeKPIs(loadKPIData(app, set), set, time.Now(), time.Time{}, time.Time{})
}

// computeKPIs is the pure computation (unit-tested with in-memory records).
// from/to bound the Period flows; zero = open.
func computeKPIs(d kpiData, set map[string]bool, now, from, to time.Time) []groupKPI {
	inRange := func(t time.Time) bool { return timeInRange(t, from, to) }
	clusterPartner := map[string]string{}
	for _, c := range d.Clusters {
		clusterPartner[helper.GetValueOfString(c, "id")] = helper.GetValueOfString(c, "partnerId")
	}

	byGroup := map[string]*groupKPI{}
	order := []string{}
	for _, g := range d.Groups {
		id := helper.GetValueOfString(g, "id")
		if set != nil && !set[id] {
			continue
		}
		cid := helper.GetValueOfString(g, "clusterId")
		k := &groupKPI{
			GroupID: id, Name: helper.GetValueOfString(g, "name"), ClusterID: cid, PartnerID: clusterPartner[cid],
			Region: helper.GetValueOfString(g, "region"), Status: helper.GetValueOfString(g, "status"),
		}
		// A group formed recently counts as active until it has had time to
		// meet. No createdAt = unknown age, so never auto-"recent".
		if created := parseTimeOrZero(g["createdAt"]); !created.IsZero() && now.Sub(created) < inactiveAfter {
			k.Active = true
		}
		byGroup[id] = k
		order = append(order, id)
	}

	for _, m := range d.Members {
		k := byGroup[helper.GetValueOfString(m, "groupId")]
		if k == nil {
			continue
		}
		k.MembersTotal++
		if joined := memberJoined(m); (from.IsZero() && to.IsZero()) || (!joined.IsZero() && inRange(joined)) {
			k.Period.NewMembers++
		}
		if !memberActive(m) {
			continue
		}
		k.Members++
		switch helper.GetValueOfString(m, "gender") {
		case "Female":
			k.Female++
		case "Male":
			k.Male++
		}
	}

	// Same type → bucket mapping as createTransaction / the reversal route.
	for _, t := range d.Transactions {
		k := byGroup[helper.GetValueOfString(t, "groupId")]
		if k == nil || !txCounted(t) {
			continue
		}
		amt := helper.GetValueOfFloat(t, "amount")
		when := parseTimeOrZero(t["createdAt"])
		inP := (from.IsZero() && to.IsZero()) || (!when.IsZero() && inRange(when))
		switch helper.GetValueOfString(t, "type") {
		case "contribution":
			k.Savings += amt
			if inP {
				k.Period.Savings += amt
			}
		case "withdrawal":
			k.Savings -= amt
			k.Withdrawals += amt
			if inP {
				k.Period.Withdrawals += amt
			}
		case "share":
			k.Shares += amt
			if inP {
				k.Period.Shares += amt
			}
		case "social_fund":
			k.SocialFund += amt
			if inP {
				k.Period.SocialFund += amt
			}
		case "fine":
			k.Fines += amt
			if inP {
				k.Period.Fines += amt
			}
		case "expense":
			k.Expenses += amt
			if inP {
				k.Period.Expenses += amt
			}
		case "loan_repayment":
			if inP {
				k.Period.LoanRepayments += amt
			}
		case "loan_disbursement":
			if inP {
				k.Period.LoansDisbursed += amt
			}
		}
	}
	for _, k := range byGroup {
		k.Savings = math.Max(0, k.Savings)
	}

	for _, l := range d.Loans {
		k := byGroup[helper.GetValueOfString(l, "groupId")]
		if k == nil {
			continue
		}
		bal := loanBalance(l)
		k.LoansOutstanding += bal
		if due := parseTimeOrZero(l["dueDate"]); bal > 0 && !due.IsZero() && now.Sub(due) > parDays*24*time.Hour {
			k.PAR += bal
		}
	}

	for _, l := range d.GovLoans {
		k := byGroup[helper.GetValueOfString(l, "groupId")]
		if k == nil {
			continue
		}
		k.GovLoansOutstanding += math.Max(0, govLoanTotalDue(l)-helper.GetValueOfFloat(l, "amountRepaid"))
	}

	meetingById := map[string]datatype.DataMap{}
	for _, mt := range d.Meetings {
		meetingById[helper.GetValueOfString(mt, "id")] = mt
		k := byGroup[helper.GetValueOfString(mt, "groupId")]
		if k == nil {
			continue
		}
		st := helper.GetValueOfString(mt, "status")
		when := meetingWhen(mt)
		if st == "completed" && !when.IsZero() && inRange(when) {
			k.Period.Meetings++
		}
		if st != "completed" && st != "in_progress" {
			continue
		}
		if when.After(k.LastMeeting) {
			k.LastMeeting = when
		}
	}

	// One attendance rule (shared with the attendance dataset): the row's
	// group falls back to its meeting's group; cancelled meetings are skipped.
	for _, a := range d.Attendance {
		mt := meetingById[helper.GetValueOfString(a, "meetingId")]
		if helper.GetValueOfString(mt, "status") == "cancelled" {
			continue
		}
		k := byGroup[attendanceGroup(a, meetingById)]
		if k == nil {
			continue
		}
		present := attended(helper.GetValueOfString(a, "status"))
		k.AttendanceRows++
		if present {
			k.AttendancePresent++
		}
		when := meetingWhen(mt)
		if when.IsZero() {
			when = parseTimeOrZero(a["createdAt"])
		}
		if (from.IsZero() && to.IsZero()) || (!when.IsZero() && inRange(when)) {
			k.Period.AttendanceRows++
			if present {
				k.Period.AttendancePresent++
			}
		}
	}

	out := make([]groupKPI, 0, len(order))
	for _, id := range order {
		k := byGroup[id]
		if !k.LastMeeting.IsZero() && now.Sub(k.LastMeeting) < inactiveAfter {
			k.Active = true
		}
		if k.Status == "Closed" || k.Status == "Inactive" {
			k.Active = false
		}
		out = append(out, *k)
	}
	return out
}

// memberJoined is when a member joined (joinedAt, else createdAt; zero if neither).
func memberJoined(m datatype.DataMap) time.Time {
	if t := parseTimeOrZero(m["joinedAt"]); !t.IsZero() {
		return t
	}
	return parseTimeOrZero(m["createdAt"])
}

// groupTotalsDrift compares a group's stored running totals with the values
// computed from its transactions. Read-only: it reports, never rewrites.
func groupTotalsDrift(g datatype.DataMap, k groupKPI) datatype.DataMap {
	out := datatype.DataMap{}
	check := func(field string, computed float64) {
		stored := helper.GetValueOfFloat(g, field)
		if math.Abs(stored-computed) > 0.005 {
			out[field] = datatype.DataMap{"stored": stored, "computed": computed, "difference": stored - computed}
		}
	}
	check("totalSavings", k.Savings)
	check("totalShares", k.Shares)
	check("totalSocialFund", k.SocialFund)
	check("totalFines", k.Fines)
	check("totalExpenses", k.Expenses)
	check("totalLoans", k.LoansOutstanding)
	return out
}

// overpaidLoans lists loans repaid past their principal (shown with balance
// 0 everywhere, so they would otherwise go unnoticed). Read-only.
func overpaidLoans(loans []datatype.DataMap, groups map[string]datatype.DataMap) []datatype.DataMap {
	out := []datatype.DataMap{}
	for _, l := range loans {
		if over := loanOverpaid(l); over > 0.005 {
			gid := helper.GetValueOfString(l, "groupId")
			out = append(out, datatype.DataMap{
				"loanId": helper.GetValueOfString(l, "id"), "loanNumber": helper.GetValueOfString(l, "loanNumber"),
				"groupId": gid, "groupName": helper.GetValueOfString(groups[gid], "name"),
				"amount": helper.GetValueOfFloat(l, "amount"), "totalDue": loanTotalDue(l), "amountRepaid": helper.GetValueOfFloat(l, "amountRepaid"),
				"overpaid": over, "status": helper.GetValueOfString(l, "status"),
			})
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		return lessFold(helper.GetValueOfString(out[i], "groupName"), helper.GetValueOfString(out[j], "groupName"),
			helper.GetValueOfString(out[i], "loanNumber"), helper.GetValueOfString(out[j], "loanNumber"))
	})
	return out
}

// rollup groups KPIs by level ("partner", "cluster" or "group") and returns
// one row per unit, largest savings first.
func rollup(app *yekonga.YekongaData, kpis []groupKPI, level string) []datatype.DataMap {
	names := map[string]string{}
	statuses := map[string]string{}
	switch level {
	case "partner":
		for _, p := range listAll(app, "Partner") {
			names[helper.GetValueOfString(p, "id")] = helper.GetValueOfString(p, "name")
			statuses[helper.GetValueOfString(p, "id")] = unitStatus(p)
		}
	case "cluster":
		for _, c := range listAll(app, "Cluster") {
			names[helper.GetValueOfString(c, "id")] = helper.GetValueOfString(c, "name")
			statuses[helper.GetValueOfString(c, "id")] = unitStatus(c)
		}
	}
	rows := rollupRows(kpis, level, names)
	for _, r := range rows {
		if st, ok := statuses[helper.GetValueOfString(r, "id")]; ok {
			r["status"] = st
		}
	}
	sort.SliceStable(rows, func(i, j int) bool {
		return helper.GetValueOfFloat(rows[i], "savings") > helper.GetValueOfFloat(rows[j], "savings")
	})
	return rows
}

// rollupTotals sums KPIs per partner / cluster / group; keys keep the
// first-seen order.
func rollupTotals(kpis []groupKPI, level string) ([]string, map[string]*kpiTotals) {
	totals := map[string]*kpiTotals{}
	keys := []string{}
	for _, k := range kpis {
		key := k.GroupID
		switch level {
		case "partner":
			key = k.PartnerID
		case "cluster":
			key = k.ClusterID
		}
		if totals[key] == nil {
			totals[key] = &kpiTotals{}
			keys = append(keys, key)
		}
		totals[key].add(k)
	}
	return keys, totals
}

// rollupRows is rollupTotals as JSON rows; names maps partner/cluster ids to
// names (group names come from the KPIs).
func rollupRows(kpis []groupKPI, level string, names map[string]string) []datatype.DataMap {
	if level != "partner" && level != "cluster" {
		for _, k := range kpis {
			names[k.GroupID] = k.Name
		}
	}
	keys, totals := rollupTotals(kpis, level)
	rows := []datatype.DataMap{}
	for _, key := range keys {
		row := totals[key].json()
		row["id"] = key
		row["name"] = names[key]
		// Groups with no cluster / partner: the web labels this row itself
		// ("Not assigned") and does not drill into it.
		row["unassigned"] = key == "" || names[key] == ""
		if row["name"] == "" {
			row["name"] = notAssigned
		}
		rows = append(rows, row)
	}
	return rows
}

// registerDashboard wires the live, scoped dashboard + roll-up endpoints.
func registerDashboard(app *yekonga.YekongaData) {
	// Live programme snapshot for whatever the caller may see, optionally
	// narrowed with ?partnerId= / ?clusterId= / ?groupId=.
	app.Get("/api/admin/dashboard", func(req *yekonga.Request, res *yekonga.Response) {
		_, set, _, ok := reportScope(app, req, res, map[string]string{
			"partnerId": req.Query("partnerId"), "clusterId": req.Query("clusterId"), "groupId": req.Query("groupId"),
		})
		if !ok {
			return
		}
		data := loadKPIData(app, set)
		now := time.Now()
		kpis := computeKPIs(data, set, now, time.Time{}, time.Time{})
		var totals kpiTotals
		for _, k := range kpis {
			totals.add(k)
		}
		chart, recent := dashboardActivity(data, kpis, now)

		res.Json(datatype.DataMap{
			"totalGroups":           totals.Groups,
			"totalMembers":          totals.Members,
			"totalSavings":          totals.Savings,
			"totalLoansOutstanding": totals.LoansOutstanding,
			"kpis":                  totals.json(),
			"chart":                 chart,
			"recentActivity":        recent,
			"status": datatype.DataMap{
				"api":            true,
				"database":       true,
				"authentication": true,
				"smsProvider":    helper.IsNotEmpty(os.Getenv("BEEM_API_KEY")) && helper.IsNotEmpty(os.Getenv("BEEM_SECRET_KEY")),
				"backgroundJobs": os.Getenv("PESABOX_REMINDERS") == "1",
			},
		})
	})

	// Roll-up table: ?level=partner|cluster|group (default group), same
	// filters as the dashboard.
	app.Get("/api/admin/rollup", func(req *yekonga.Request, res *yekonga.Response) {
		_, set, _, ok := reportScope(app, req, res, map[string]string{
			"partnerId": req.Query("partnerId"), "clusterId": req.Query("clusterId"), "groupId": req.Query("groupId"),
		})
		if !ok {
			return
		}
		level := req.Query("level")
		if level != "partner" && level != "cluster" {
			level = "group"
		}
		kpis := computeGroupKPIs(app, set)
		var totals kpiTotals
		for _, k := range kpis {
			totals.add(k)
		}
		res.Json(datatype.DataMap{"level": level, "rows": rollup(app, kpis, level), "totals": totals.json()})
	})

	// Read-only check of the stored Group running totals against the values
	// computed from transactions (super admin). Nothing is rewritten.
	app.Get("/api/admin/totals-drift", func(req *yekonga.Request, res *yekonga.Response) {
		a := requirePerm(app, req, res, PermPlatform)
		if a == nil {
			return
		}
		data := loadKPIData(app, nil)
		kpis := computeKPIs(data, nil, time.Now(), time.Time{}, time.Time{})
		byId := map[string]datatype.DataMap{}
		for _, g := range data.Groups {
			byId[helper.GetValueOfString(g, "id")] = g
		}
		out := []datatype.DataMap{}
		for _, k := range kpis {
			if d := groupTotalsDrift(byId[k.GroupID], k); len(d) > 0 {
				out = append(out, datatype.DataMap{"groupId": k.GroupID, "name": k.Name, "drift": d})
			}
		}
		res.Json(datatype.DataMap{"groups": out, "overpaidLoans": overpaidLoans(data.Loans, byId)})
	})
}

// txActivityType is the stable snake-case key the dashboard sends for a
// transaction type (the web translates it as dash2.act.<key>).
func txActivityType(typ string) string {
	switch typ {
	case "contribution", "share", "social_fund", "loan_disbursement", "loan_repayment", "fine", "expense", "withdrawal":
		return typ
	}
	return "other"
}

// dashboardActivity builds the 7-day transaction chart (EAT days, ISO dates
// the web formats in the viewer's locale) and the 8 latest transactions,
// reusing the transactions already loaded for the KPIs.
func dashboardActivity(data kpiData, kpis []groupKPI, now time.Time) ([]datatype.DataMap, []datatype.DataMap) {
	groupNames := map[string]string{}
	for _, k := range kpis {
		groupNames[k.GroupID] = k.Name
	}
	memberById := map[string]datatype.DataMap{}
	for _, m := range data.Members {
		memberById[helper.GetValueOfString(m, "id")] = m
	}

	today := dayStart(now)
	chartByDay := map[string]int{}
	dayOrder := make([]string, 0, 7)
	for i := 6; i >= 0; i-- {
		key := today.AddDate(0, 0, -i).Format("2006-01-02")
		chartByDay[key] = 0
		dayOrder = append(dayOrder, key)
	}

	type txAt struct {
		t    datatype.DataMap
		when time.Time
	}
	txs := make([]txAt, 0, len(data.Transactions))
	for _, t := range data.Transactions {
		if !txCounted(t) {
			continue
		}
		if _, ok := groupNames[helper.GetValueOfString(t, "groupId")]; !ok {
			continue
		}
		when := parseTimeOrZero(t["createdAt"])
		if when.IsZero() {
			continue
		}
		txs = append(txs, txAt{t, when})
		if _, ok := chartByDay[when.In(eatZone).Format("2006-01-02")]; ok {
			chartByDay[when.In(eatZone).Format("2006-01-02")]++
		}
	}
	sort.SliceStable(txs, func(i, j int) bool { return txs[i].when.After(txs[j].when) })

	recent := []datatype.DataMap{}
	for _, x := range txs {
		if len(recent) >= 8 {
			break
		}
		t := x.t
		memberName := ""
		if mm, ok := memberById[helper.GetValueOfString(t, "memberId")]; ok {
			memberName = memberFullName(mm)
		}
		typ := helper.GetValueOfString(t, "type")
		local := x.when.In(eatZone)
		recent = append(recent, datatype.DataMap{
			"at":        local.Format(time.RFC3339),
			"date":      local.Format("2006-01-02"),
			"time":      local.Format("15:04"),
			"member":    memberName,
			"group":     groupNames[helper.GetValueOfString(t, "groupId")],
			"type":      txActivityType(typ),
			"label":     txLabel(typ),
			"amount":    helper.GetValueOfFloat(t, "amount"),
			"direction": helper.GetValueOfString(t, "direction"),
		})
	}

	chart := make([]datatype.DataMap, 0, 7)
	for _, key := range dayOrder {
		d, _ := time.ParseInLocation("2006-01-02", key, eatZone)
		chart = append(chart, datatype.DataMap{"date": key, "day": d.Format("Mon"), "count": chartByDay[key]})
	}
	return chart, recent
}
