package main

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/robertkonga/yekonga-server-go/datatype"
	"github.com/robertkonga/yekonga-server-go/helper"
	"github.com/robertkonga/yekonga-server-go/yekonga"
)

// ---------------------------------------------------------------------------
// GATA monitoring indicators (GET /api/admin/monitoring, MONITORING_API.md).
// One pure computation (computeMonitoring) over the scoped records feeds the
// endpoint and the "Monitoring" report datasets. It follows the report rules:
// reversed transactions and cancelled loans never count, days are EAT days,
// "to" is inclusive, and balances are computed from the records (never the
// Group's stored running totals). At "now" its balances equal computeKPIs'.
// ---------------------------------------------------------------------------

const (
	// loanDefaultDays: an unpaid loan this many days past its due date (or
	// marked "defaulted") is a default.
	loanDefaultDays = 90
	// regularSaverPct: a member who saved or bought shares at this share of
	// their group's completed meetings in the period saves regularly.
	regularSaverPct = 75.0
	// appActiveDays: a group recording a meeting or a transaction in this many
	// days up to the end of the period is actively using the app.
	appActiveDays = 30
	// defaultMonitorMonths: the period when "from" is not given.
	defaultMonitorMonths = 12
	// maxSeriesMonths caps the monthly growth series.
	maxSeriesMonths = 60
	// maxEnterpriseRows caps the enterprise-type breakdown.
	maxEnterpriseRows = 25
)

// monitorData is every record the indicators read, limited to the scope.
type monitorData struct {
	kpiData
	AuditLogs, SmsLogs, GovRepayments []datatype.DataMap
	// Hide limits what the caller may see (nil = everything, e.g. tests).
	Hide *monitorHide
}

// monitorHide: SMS figures only for groups where the caller has sms.view
// (smsOK), the audit-log figures only for audit.view (super admin).
type monitorHide struct {
	smsOK   func(groupId string) bool
	noAudit bool
}

func loadMonitorData(app *yekonga.YekongaData, set map[string]bool) monitorData {
	q := func(model string) []datatype.DataMap {
		return reportQuery(app, model, set, "", time.Time{}, time.Time{})
	}
	return monitorData{kpiData: loadKPIData(app, set), AuditLogs: q("AuditLog"), SmsLogs: q("SmsLog"), GovRepayments: q("GovernmentLoanRepayment")}
}

// monitorPeriod resolves the period: "to" defaults to the end of today (EAT),
// "from" to the first day of the month 11 months before "to"'s month (12
// calendar months). asAt (where balances are taken) is "to", or now when
// "to" is in the future.
func monitorPeriod(from, to, now time.Time) (f, t, asAt time.Time, defaulted bool, err error) {
	t = to
	if t.IsZero() {
		t = dayStart(now).AddDate(0, 0, 1).Add(-time.Nanosecond)
	}
	f = from
	if f.IsZero() {
		e := t.In(eatZone)
		f = time.Date(e.Year(), e.Month()-(defaultMonitorMonths-1), 1, 0, 0, 0, 0, eatZone)
		defaulted = true
	}
	asAt = t
	if asAt.After(now) {
		asAt = now
	}
	if f.After(t) {
		err = fmt.Errorf("'from' (%s) is after 'to' (%s)", f.In(eatZone).Format("2006-01-02"), t.In(eatZone).Format("2006-01-02"))
	}
	return f, t, asAt, defaulted, err
}

type monTx struct {
	rec            datatype.DataMap
	typ, gid, mid  string
	amt            float64
	when           time.Time
	reversed       bool
	meeting        string // the meeting it belongs to ("" = none)
	sv, sh, profit float64
}

type monLoan struct {
	rec                     datatype.DataMap
	gid, mid, number        string
	issued, due, updated    time.Time
	amount, totalDue, inter float64
	cancelled               bool
	reps                    []*monTx
}

type monitorCtx struct {
	now, from, to, asAt time.Time
	groups              map[string]datatype.DataMap
	groupIDs            []string
	members             []datatype.DataMap
	memberByID          map[string]datatype.DataMap
	inactiveSince       map[string]time.Time
	txs                 []*monTx
	loans               []*monLoan
	meetings            map[string]datatype.DataMap
	kpis                []groupKPI
	d                   monitorData
}

func newMonitorCtx(d monitorData, set map[string]bool, now, from, to time.Time) *monitorCtx {
	c := &monitorCtx{now: now, d: d, groups: map[string]datatype.DataMap{}, memberByID: map[string]datatype.DataMap{},
		inactiveSince: map[string]time.Time{}, meetings: map[string]datatype.DataMap{}}
	c.from, c.to, c.asAt, _, _ = monitorPeriod(from, to, now)
	for _, g := range d.Groups {
		id := helper.GetValueOfString(g, "id")
		if set != nil && !set[id] {
			continue
		}
		c.groups[id] = g
		c.groupIDs = append(c.groupIDs, id)
	}
	for _, m := range d.Members {
		if _, ok := c.groups[helper.GetValueOfString(m, "groupId")]; ok {
			c.members = append(c.members, m)
			c.memberByID[helper.GetValueOfString(m, "id")] = m
		}
	}
	// When an inactive member stopped being active: their last "deactivate"
	// audit entry, else the record's last update.
	for _, a := range d.AuditLogs {
		if helper.GetValueOfString(a, "model") != "Member" || helper.GetValueOfString(a, "action") != "deactivate" {
			continue
		}
		id := helper.GetValueOfString(a, "recordId")
		if when := parseTimeOrZero(a["createdAt"]); when.After(c.inactiveSince[id]) {
			c.inactiveSince[id] = when
		}
	}
	for _, m := range c.members {
		id := helper.GetValueOfString(m, "id")
		if _, ok := c.inactiveSince[id]; !ok && !memberActive(m) {
			if u := parseTimeOrZero(m["updatedAt"]); !u.IsZero() {
				c.inactiveSince[id] = u
			}
		}
	}

	sameDay := map[string][]string{}
	for _, mt := range d.Meetings {
		if _, ok := c.groups[helper.GetValueOfString(mt, "groupId")]; !ok {
			continue
		}
		id := helper.GetValueOfString(mt, "id")
		c.meetings[id] = mt
		if helper.GetValueOfString(mt, "status") == "cancelled" {
			continue
		}
		if when := meetingWhen(mt); !when.IsZero() {
			k := helper.GetValueOfString(mt, "groupId") + "|" + when.In(eatZone).Format("2006-01-02")
			sameDay[k] = append(sameDay[k], id)
		}
	}
	for _, t := range d.Transactions {
		gid := helper.GetValueOfString(t, "groupId")
		if _, ok := c.groups[gid]; !ok {
			continue
		}
		x := &monTx{rec: t, typ: helper.GetValueOfString(t, "type"), gid: gid, mid: helper.GetValueOfString(t, "memberId"),
			amt: helper.GetValueOfFloat(t, "amount"), when: parseTimeOrZero(t["createdAt"]), reversed: !txCounted(t),
			sv: helper.GetValueOfFloat(t, "savingsReturned"), sh: helper.GetValueOfFloat(t, "sharesReturned"), profit: helper.GetValueOfFloat(t, "profit")}
		// Same meeting rule as the meeting-collections report.
		if mid := helper.GetValueOfString(t, "meetingId"); mid != "" {
			if _, ok := c.meetings[mid]; ok {
				x.meeting = mid
			}
		}
		if x.meeting == "" && !x.when.IsZero() {
			if ids := sameDay[gid+"|"+x.when.In(eatZone).Format("2006-01-02")]; len(ids) == 1 {
				x.meeting = ids[0]
			}
		}
		c.txs = append(c.txs, x)
	}
	byNumber := map[string]*monLoan{}
	for _, l := range d.Loans {
		gid := helper.GetValueOfString(l, "groupId")
		if _, ok := c.groups[gid]; !ok {
			continue
		}
		issued := parseTimeOrZero(l["issuedDate"])
		if issued.IsZero() {
			issued = parseTimeOrZero(l["createdAt"])
		}
		ml := &monLoan{rec: l, gid: gid, mid: helper.GetValueOfString(l, "memberId"), number: helper.GetValueOfString(l, "loanNumber"),
			issued: issued, due: parseTimeOrZero(l["dueDate"]), updated: parseTimeOrZero(l["updatedAt"]),
			amount: helper.GetValueOfFloat(l, "amount"), totalDue: loanTotalDue(l), inter: loanInterest(l),
			cancelled: helper.GetValueOfString(l, "status") == "cancelled"}
		c.loans = append(c.loans, ml)
		if ml.number != "" && !ml.cancelled {
			byNumber[gid+"|"+ml.number] = ml
		}
	}
	for _, x := range c.txs {
		if x.typ == "loan_repayment" && !x.reversed {
			if l := byNumber[x.gid+"|"+helper.GetValueOfString(x.rec, "reference")]; l != nil {
				l.reps = append(l.reps, x)
			}
		}
	}
	c.kpis = computeKPIs(d.kpiData, set, now, c.from, c.to)
	return c
}

func (c *monitorCtx) inPeriod(t time.Time) bool { return !t.IsZero() && timeInRange(t, c.from, c.to) }

// atOrBefore: a record dated t exists at time T (an undated one always does).
func atOrBefore(t, T time.Time) bool { return t.IsZero() || !t.After(T) }

// activeAt: the member had joined by T and was not (yet) inactive.
func (c *monitorCtx) activeAt(m datatype.DataMap, T time.Time) bool {
	if j := memberJoined(m); !j.IsZero() && j.After(T) {
		return false
	}
	if memberActive(m) {
		return true
	}
	since, ok := c.inactiveSince[helper.GetValueOfString(m, "id")]
	return ok && T.Before(since)
}

// repaidAt is what was repaid on a loan by T.
func (c *monitorCtx) repaidAt(l *monLoan, T time.Time) float64 {
	if !T.Before(c.now) {
		return helper.GetValueOfFloat(l.rec, "amountRepaid")
	}
	if len(l.reps) == 0 { // repaid before repayments were recorded as transactions
		if atOrBefore(l.updated, T) {
			return helper.GetValueOfFloat(l.rec, "amountRepaid")
		}
		return 0
	}
	var s float64
	for _, r := range l.reps {
		if atOrBefore(r.when, T) {
			s += r.amt
		}
	}
	return s
}

// balanceAt is what a loan still owed at T (loanBalance at now).
func (c *monitorCtx) balanceAt(l *monLoan, T time.Time) float64 {
	if l.cancelled || (!l.issued.IsZero() && l.issued.After(T)) {
		return 0
	}
	if !T.Before(c.now) {
		return loanBalance(l.rec)
	}
	return math.Max(0, l.totalDue-c.repaidAt(l, T))
}

type monSnap struct {
	Savings, Shares, SocialFund, Cash, Portfolio float64
	Members                                      int
}

func (s monSnap) assets() float64 { return s.Cash + s.Portfolio }

// snapshot is the position at T: savings / shares / social fund balances,
// the cash box (transactions + government loan money), the loan portfolio
// (total due less repaid) and the active members.
func (c *monitorCtx) snapshot(T time.Time) monSnap {
	var s monSnap
	type gm struct{ sv, sh, sf float64 }
	per := map[string]*gm{}
	for _, x := range c.txs {
		if x.reversed || (T.Before(c.now) && !atOrBefore(x.when, T)) {
			continue
		}
		g := per[x.gid]
		if g == nil {
			g = &gm{}
			per[x.gid] = g
		}
		if txIn(x.typ) {
			s.Cash += x.amt
		} else {
			s.Cash -= x.amt
		}
		switch x.typ {
		case "contribution":
			g.sv += x.amt
		case "withdrawal":
			g.sv -= x.amt
		case "share":
			g.sh += x.amt
		case "share_out":
			g.sv -= x.sv
			g.sh -= x.sh
		case "social_fund":
			g.sf += x.amt
		case "social_fund_payout":
			g.sf -= x.amt
		}
	}
	for _, g := range per { // never negative per group, as in computeKPIs
		s.Savings += math.Max(0, g.sv)
		s.Shares += math.Max(0, g.sh)
		s.SocialFund += math.Max(0, g.sf)
	}
	for _, l := range c.d.GovLoans {
		if _, ok := c.groups[helper.GetValueOfString(l, "groupId")]; ok && atOrBefore(parseTimeOrZero(l["receivedDate"]), T) {
			s.Cash += helper.GetValueOfFloat(l, "amount")
		}
	}
	for _, r := range c.d.GovRepayments {
		when := parseTimeOrZero(r["date"])
		if when.IsZero() {
			when = parseTimeOrZero(r["createdAt"])
		}
		if _, ok := c.groups[helper.GetValueOfString(r, "groupId")]; ok && atOrBefore(when, T) {
			s.Cash -= helper.GetValueOfFloat(r, "amount")
		}
	}
	for _, l := range c.loans {
		s.Portfolio += c.balanceAt(l, T)
	}
	for _, m := range c.members {
		if c.activeAt(m, T) {
			s.Members++
		}
	}
	return s
}

// memberBalances is each member's savings + shares at T (never negative).
func (c *monitorCtx) memberBalances(T time.Time) map[string]float64 {
	out := map[string]float64{}
	for _, x := range c.txs {
		if x.reversed || x.mid == "" || (T.Before(c.now) && !atOrBefore(x.when, T)) {
			continue
		}
		switch x.typ {
		case "contribution", "share":
			out[x.mid] += x.amt
		case "withdrawal":
			out[x.mid] -= x.amt
		case "share_out":
			out[x.mid] -= x.sv + x.sh
		}
	}
	for k, v := range out {
		out[k] = math.Max(0, v)
	}
	return out
}

// memberOwedAt is each member's loan balance at T.
func (c *monitorCtx) memberOwedAt(T time.Time) map[string]float64 {
	out := map[string]float64{}
	for _, l := range c.loans {
		if b := c.balanceAt(l, T); b > 0 {
			out[l.mid] += b
		}
	}
	return out
}

// ---- small helpers ---------------------------------------------------------

func round2(v float64) float64 { return math.Round(v*100) / 100 }

func avg(total float64, n int) float64 {
	if n <= 0 {
		return 0
	}
	return round2(total / float64(n))
}

// changePct is the % change from a to b (0 when a is 0).
func changePct(a, b float64) float64 {
	if a == 0 {
		return 0
	}
	return math.Round((b-a)/math.Abs(a)*1000) / 10
}

// dim is a breakdown (by gender, age band, purpose, ...) of named counters.
type dim struct {
	order []string
	rows  map[string]map[string]float64
}

func newDim(keys ...string) *dim {
	d := &dim{rows: map[string]map[string]float64{}}
	for _, k := range keys {
		d.row(k)
	}
	return d
}

func (d *dim) row(k string) map[string]float64 {
	if d.rows[k] == nil {
		d.rows[k] = map[string]float64{}
		d.order = append(d.order, k)
	}
	return d.rows[k]
}

func (d *dim) add(k, field string, v float64) { d.row(k)[field] += v }

// list renders the rows; ints are the counter fields shown as whole numbers,
// rates are computed fields name -> [numerator, denominator].
func (d *dim) list(keyName string, ints []string, money []string, rates map[string][2]string) []datatype.DataMap {
	out := []datatype.DataMap{}
	for _, k := range d.order {
		r := d.rows[k]
		row := datatype.DataMap{keyName: k}
		for _, f := range ints {
			row[f] = int(r[f])
		}
		for _, f := range money {
			row[f] = round2(r[f])
		}
		for f, p := range rates {
			row[f] = pct(r[p[0]], r[p[1]])
		}
		out = append(out, row)
	}
	return out
}

// distinct counts distinct strings.
type distinct map[string]bool

func (s distinct) n() int { return len(s) }

// ---- the computation -------------------------------------------------------

// monitorExtras are the finer breakdowns the report datasets need.
type monitorExtras struct {
	byAge, byGenderAge *dim
	members            []memberStat
}

// memberStat is one member's line in the member-savings dataset.
type memberStat struct {
	m                                 datatype.DataMap
	gender, band                      string
	saved, shares, balance, owed      float64
	meetingsSaved, meetingsHeld       int
	regular, eligible, active, newMem bool
}

// computeMonitoring returns every GATA section for the scoped records.
func computeMonitoring(d monitorData, set map[string]bool, now, from, to time.Time) datatype.DataMap {
	out, _ := monitor(d, set, now, from, to)
	return out
}

func monitor(d monitorData, set map[string]bool, now, from, to time.Time) (datatype.DataMap, *monitorExtras) {
	c := newMonitorCtx(d, set, now, from, to)
	start := c.from.Add(-time.Nanosecond) // the position just before the period
	asAt := c.asAt
	snapStart, snapEnd := c.snapshot(start), c.snapshot(asAt)

	// ---- members: who is active, new, retained; their gender / age band
	ageAt := asAt
	gender := map[string]string{}
	band := map[string]string{}
	activeEnd := distinct{}
	activeStart := distinct{}
	newMembers := distinct{}
	for _, m := range c.members {
		id := helper.GetValueOfString(m, "id")
		gender[id], band[id] = genderKey(m), ageBandOf(m, ageAt)
		if c.activeAt(m, asAt) {
			activeEnd[id] = true
		}
		if c.activeAt(m, start) {
			activeStart[id] = true
		}
		if c.inPeriod(memberJoined(m)) {
			newMembers[id] = true
		}
	}
	retained := 0
	for id := range activeStart {
		if activeEnd[id] {
			retained++
		}
	}

	// ---- meetings + attendance in the period
	completed := map[string]datatype.DataMap{} // completed meetings in the period
	groupMeetings := map[string][]time.Time{}  // their dates per group
	for id, mt := range c.meetings {
		if helper.GetValueOfString(mt, "status") == "completed" && c.inPeriod(meetingWhen(mt)) {
			completed[id] = mt
			gid := helper.GetValueOfString(mt, "groupId")
			groupMeetings[gid] = append(groupMeetings[gid], meetingWhen(mt))
		}
	}
	var attP, attL, attA, attE, attRows int
	attended := map[string]bool{} // meetings with attendance recorded
	attByMember := map[string][2]int{}
	for _, a := range d.Attendance {
		mt := c.meetings[helper.GetValueOfString(a, "meetingId")]
		if mt == nil || helper.GetValueOfString(mt, "status") == "cancelled" {
			continue
		}
		when := meetingWhen(mt)
		if when.IsZero() {
			when = parseTimeOrZero(a["createdAt"])
		}
		if !c.inPeriod(when) {
			continue
		}
		attended[helper.GetValueOfString(a, "meetingId")] = true
		st := helper.GetValueOfString(a, "status")
		attRows++
		switch st {
		case "present":
			attP++
		case "late":
			attL++
		case "absent":
			attA++
		case "excused":
			attE++
		}
		mid := helper.GetValueOfString(a, "memberId")
		v := attByMember[mid]
		v[1]++
		if attendedStatus := st == "present" || st == "late"; attendedStatus {
			v[0]++
		}
		attByMember[mid] = v
	}

	// ---- transactions in the period
	var contributions, mandatory, voluntary, shares, withdrawals float64
	var sfIn, sfOut, repaid, fines, expenses, sharedOut, svBack, shBack, profit, interestEarned, reversedAmt float64
	var txCount, txReversed, txAtMeeting, sfPayouts, shareOuts int
	contribByMember := map[string]float64{}
	sharesByMember := map[string]float64{}
	saversAt := map[string]map[string]bool{} // member -> completed meetings they saved at
	sfContributors, sfBeneficiaries, paidOut, recorders := distinct{}, distinct{}, distinct{}, distinct{}
	moneyAt := map[string]bool{} // completed meetings with money recorded
	loanByNumber := map[string]*monLoan{}
	for _, l := range c.loans {
		if !l.cancelled {
			loanByNumber[l.gid+"|"+l.number] = l
		}
	}
	for _, x := range c.txs {
		if !c.inPeriod(x.when) {
			continue
		}
		txCount++
		if x.reversed {
			txReversed++
			reversedAmt += x.amt
			continue
		}
		if x.meeting != "" {
			txAtMeeting++
			if _, ok := completed[x.meeting]; ok {
				moneyAt[x.meeting] = true
			}
		}
		if by := helper.GetValueOfString(x.rec, "createdBy"); by != "" {
			recorders[by] = true
		}
		switch x.typ {
		case "contribution":
			contributions += x.amt
			if helper.GetValueOfString(x.rec, "savingsType") == "voluntary" {
				voluntary += x.amt
			} else {
				mandatory += x.amt
			}
		case "share":
			shares += x.amt
		case "withdrawal":
			withdrawals += x.amt
		case "social_fund":
			sfIn += x.amt
			if x.mid != "" {
				sfContributors[x.mid] = true
			}
		case "social_fund_payout":
			sfOut += x.amt
			sfPayouts++
			if x.mid != "" {
				sfBeneficiaries[x.mid] = true
			}
		case "loan_repayment":
			repaid += x.amt
			// Interest earned: each repayment is split pro rata between
			// principal and the loan's flat interest.
			if l := loanByNumber[x.gid+"|"+helper.GetValueOfString(x.rec, "reference")]; l != nil && l.totalDue > 0 {
				interestEarned += x.amt * l.inter / l.totalDue
			}
		case "fine":
			fines += x.amt
		case "expense":
			expenses += x.amt
		case "share_out":
			sharedOut += x.amt
			svBack += x.sv
			shBack += x.sh
			profit += x.profit
			shareOuts++
			if x.mid != "" {
				paidOut[x.mid] = true
			}
		}
		if (x.typ == "contribution" || x.typ == "share") && x.mid != "" {
			contribByMember[x.mid] += x.amt
			if x.typ == "share" {
				sharesByMember[x.mid] += x.amt
			}
			if _, ok := completed[x.meeting]; ok {
				if saversAt[x.mid] == nil {
					saversAt[x.mid] = map[string]bool{}
				}
				saversAt[x.mid][x.meeting] = true
			}
		}
	}

	// ---- loans
	var issuedN int
	var issuedAmt, issuedInterest float64
	borrowers := distinct{}
	byPurpose := newDim()
	purposeBorrowers := map[string]distinct{}
	byEnterprise := newDim()
	enterpriseBorrowers := map[string]distinct{}
	enterpriseLabel := map[string]string{}
	igaBorrowers := distinct{}
	var igaN, specifiedN int
	var igaAmt float64
	for _, l := range c.loans {
		if l.cancelled || !c.inPeriod(l.issued) {
			continue
		}
		issuedN++
		issuedAmt += l.amount
		issuedInterest += l.inter
		borrowers[l.mid] = true
		p := loanPurpose(l.rec)
		if p != purposeUnspecified {
			specifiedN++
		}
		byPurpose.add(p, "loans", 1)
		byPurpose.add(p, "amount", l.amount)
		if purposeBorrowers[p] == nil {
			purposeBorrowers[p] = distinct{}
		}
		purposeBorrowers[p][l.mid] = true
		if igaPurposes[p] {
			igaN++
			igaAmt += l.amount
			igaBorrowers[l.mid] = true
		}
		if e := strings.TrimSpace(helper.GetValueOfString(l.rec, "enterpriseType")); e != "" {
			key := strings.ToLower(strings.Join(strings.Fields(e), " "))
			if _, ok := enterpriseLabel[key]; !ok {
				enterpriseLabel[key] = strings.Join(strings.Fields(e), " ")
			}
			byEnterprise.add(key, "loans", 1)
			byEnterprise.add(key, "amount", l.amount)
			if enterpriseBorrowers[key] == nil {
				enterpriseBorrowers[key] = distinct{}
			}
			enterpriseBorrowers[key][l.mid] = true
		}
	}
	// Point-in-time loan quality at asAt.
	var activeLoans, overdueN, defaultN, par30N int
	var overdueAmt, defaultAmt, par30Amt float64
	openBorrowers, overdueBorrowers := distinct{}, distinct{}
	var dueInP, repaidInP, dueAll, repaidAll float64
	var dueInPN, dueAllN, fullyRepaidN int
	for _, l := range c.loans {
		if l.cancelled || (!l.issued.IsZero() && l.issued.After(asAt)) {
			continue
		}
		bal := c.balanceAt(l, asAt)
		if bal > 0 {
			activeLoans++
			openBorrowers[l.mid] = true
		}
		if !l.due.IsZero() && !l.due.After(asAt) {
			got := math.Min(c.repaidAt(l, asAt), l.totalDue)
			dueAll += l.totalDue
			repaidAll += got
			dueAllN++
			if bal <= 0.005 {
				fullyRepaidN++
			}
			if !l.due.Before(c.from) {
				dueInP += l.totalDue
				repaidInP += got
				dueInPN++
			}
		}
		if bal <= 0 || l.due.IsZero() || !asAt.After(l.due) {
			if bal > 0 && strings.EqualFold(helper.GetValueOfString(l.rec, "status"), "defaulted") {
				defaultN++
				defaultAmt += bal
			}
			continue
		}
		late := asAt.Sub(l.due)
		overdueN++
		overdueAmt += bal
		overdueBorrowers[l.mid] = true
		if late > parDays*24*time.Hour {
			par30N++
			par30Amt += bal
		}
		if late > loanDefaultDays*24*time.Hour || strings.EqualFold(helper.GetValueOfString(l.rec, "status"), "defaulted") {
			defaultN++
			defaultAmt += bal
		}
	}

	// ---- per member: balances, borrowing, regular saving
	balStart, balEnd := c.memberBalances(start), c.memberBalances(asAt)
	owedStart, owedEnd := c.memberOwedAt(start), c.memberOwedAt(asAt)
	var sumBalStart, sumBalEnd, sumOwedStart, sumOwedEnd float64
	for id := range activeStart {
		sumBalStart += balStart[id]
		sumOwedStart += owedStart[id]
	}
	for id := range activeEnd {
		sumBalEnd += balEnd[id]
		sumOwedEnd += owedEnd[id]
	}
	regular := distinct{}
	eligible := distinct{}
	heldFor := map[string]int{}
	for id := range activeEnd {
		m := c.memberByID[id]
		joined := memberJoined(m)
		held := 0
		for _, when := range groupMeetings[helper.GetValueOfString(m, "groupId")] {
			if joined.IsZero() || !when.Before(dayStart(joined)) {
				held++
			}
		}
		if held == 0 {
			continue
		}
		heldFor[id] = held
		eligible[id] = true
		if float64(len(saversAt[id]))*100 >= regularSaverPct*float64(held) {
			regular[id] = true
		}
	}

	// ---- breakdowns by gender and age band
	byGender, byAge := newDim(genderKeys...), newDim(ageBands...)
	byGenderAge := newDim()
	for _, m := range c.members {
		id := helper.GetValueOfString(m, "id")
		add := func(field string, v float64) {
			byGender.add(gender[id], field, v)
			byAge.add(band[id], field, v)
		}
		if activeEnd[id] {
			add("activeMembers", 1)
			byGenderAge.add(gender[id]+"|"+band[id], "activeMembers", 1)
			add("savingsBalance", balEnd[id])
		}
		if newMembers[id] {
			add("newMembers", 1)
			byGenderAge.add(gender[id]+"|"+band[id], "newMembers", 1)
		}
		if activeStart[id] {
			add("membersAtStart", 1)
			if activeEnd[id] {
				add("retained", 1)
			}
		}
		if a := attByMember[id]; a[1] > 0 {
			add("attended", float64(a[0]))
			add("attendanceRows", float64(a[1]))
		}
		if amt := contribByMember[id]; amt > 0 {
			add("savers", 1)
			add("savingsRecorded", amt)
		}
		if borrowers[id] {
			add("borrowers", 1)
		}
		if igaBorrowers[id] {
			add("igaBorrowers", 1)
		}
		if eligible[id] {
			add("eligibleSavers", 1)
		}
		if regular[id] {
			add("regularSavers", 1)
		}
	}
	for _, l := range c.loans {
		if l.cancelled || !c.inPeriod(l.issued) {
			continue
		}
		byGender.add(gender[l.mid], "loans", 1)
		byGender.add(gender[l.mid], "loanAmount", l.amount)
		byAge.add(bandOr(band, l.mid), "loans", 1)
		byAge.add(bandOr(band, l.mid), "loanAmount", l.amount)
	}

	// ---- digital adoption
	windowStart := asAt.Add(-appActiveDays * 24 * time.Hour)
	usingApp, usedInPeriod := distinct{}, distinct{}
	for _, mt := range c.meetings {
		st := helper.GetValueOfString(mt, "status")
		if st != "completed" && st != "in_progress" {
			continue
		}
		when := meetingWhen(mt)
		gid := helper.GetValueOfString(mt, "groupId")
		if !when.IsZero() && !when.Before(windowStart) && !when.After(asAt) {
			usingApp[gid] = true
		}
		if c.inPeriod(when) {
			usedInPeriod[gid] = true
		}
	}
	for _, x := range c.txs {
		if x.reversed || x.when.IsZero() {
			continue
		}
		if !x.when.Before(windowStart) && !x.when.After(asAt) {
			usingApp[x.gid] = true
		}
		if c.inPeriod(x.when) {
			usedInPeriod[x.gid] = true
		}
	}
	fullyRecorded := 0
	for id := range completed {
		if attended[id] && moneyAt[id] {
			fullyRecorded++
		}
	}

	// ---- accountability
	groupsWithDrift := 0
	var driftAbs float64
	for _, k := range c.kpis {
		dr := groupTotalsDrift(c.groups[k.GroupID], k)
		if len(dr) > 0 {
			groupsWithDrift++
		}
		for _, v := range dr {
			driftAbs += math.Abs(helper.GetValueOfFloat(helper.ToDataMap(v), "difference"))
		}
	}
	overpaid := len(overpaidLoans(d.Loans, c.groups))
	auditN := 0
	auditBy := newDim()
	auditHidden := d.Hide != nil && d.Hide.noAudit
	smsHidden := false
	if d.Hide != nil && d.Hide.smsOK != nil {
		for _, gid := range c.groupIDs {
			if !d.Hide.smsOK(gid) {
				smsHidden = true
			}
		}
	}
	for _, a := range d.AuditLogs {
		if auditHidden {
			break
		}
		if _, ok := c.groups[helper.GetValueOfString(a, "groupId")]; !ok || !c.inPeriod(parseTimeOrZero(a["createdAt"])) {
			continue
		}
		auditN++
		auditBy.add(helper.GetValueOfString(a, "action"), "count", 1)
	}
	var smsSent, smsFailed int
	notified := distinct{}
	for _, l := range d.SmsLogs {
		gid := helper.GetValueOfString(l, "groupId")
		if _, ok := c.groups[gid]; !ok || (d.Hide != nil && d.Hide.smsOK != nil && !d.Hide.smsOK(gid)) {
			continue
		}
		when := parseTimeOrZero(l["sentAt"])
		if when.IsZero() {
			when = parseTimeOrZero(l["createdAt"])
		}
		if !c.inPeriod(when) {
			continue
		}
		if helper.GetValueOfString(l, "status") == "sent" {
			smsSent++
			if mid := helper.GetValueOfString(l, "memberId"); mid != "" && activeEnd[mid] {
				notified[mid] = true
			}
		} else {
			smsFailed++
		}
	}

	// ---- growth: one point per calendar month (EAT), up to asAt
	monthly := []datatype.DataMap{}
	f := c.from.In(eatZone)
	first := time.Date(f.Year(), f.Month(), 1, 0, 0, 0, 0, eatZone)
	e := asAt.In(eatZone)
	last := time.Date(e.Year(), e.Month(), 1, 0, 0, 0, 0, eatZone)
	if months := (last.Year()-first.Year())*12 + int(last.Month()-first.Month()) + 1; months > maxSeriesMonths {
		first = last.AddDate(0, -(maxSeriesMonths - 1), 0)
	}
	for m := first; !m.After(last); m = m.AddDate(0, 1, 0) {
		end := m.AddDate(0, 1, 0).Add(-time.Nanosecond)
		if end.After(asAt) {
			end = asAt
		}
		// The flows of the first month start at "from", so the points add up
		// to the period totals.
		winStart := m
		if c.from.After(winStart) {
			winStart = c.from
		}
		s := c.snapshot(end)
		var joined, disbN int
		var disb, saved float64
		for _, mm := range c.members {
			if j := memberJoined(mm); !j.IsZero() && !j.Before(winStart) && !j.After(end) {
				joined++
			}
		}
		for _, l := range c.loans {
			if !l.cancelled && !l.issued.IsZero() && !l.issued.Before(winStart) && !l.issued.After(end) {
				disbN++
				disb += l.amount
			}
		}
		for _, x := range c.txs {
			if !x.reversed && (x.typ == "contribution" || x.typ == "share") && !x.when.IsZero() && !x.when.Before(winStart) && !x.when.After(end) {
				saved += x.amt
			}
		}
		monthly = append(monthly, datatype.DataMap{
			"month": m.Format("2006-01"), "activeMembers": s.Members, "newMembers": joined,
			"savingsRecorded": round2(saved), "savingsBalance": round2(s.Savings), "sharesBalance": round2(s.Shares),
			"totalSavings": round2(s.Savings + s.Shares), "socialFundBalance": round2(s.SocialFund),
			"loansDisbursed": round2(disb), "loansIssued": disbN, "loanPortfolio": round2(s.Portfolio),
			"cashOnHand": round2(s.Cash), "groupAssets": round2(s.assets()),
		})
	}

	// ---- sections
	contribs := []float64{}
	for _, v := range contribByMember {
		contribs = append(contribs, v)
	}
	sort.Float64s(contribs)
	median := 0.0
	if n := len(contribs); n > 0 {
		median = contribs[n/2]
		if n%2 == 0 {
			median = (contribs[n/2-1] + contribs[n/2]) / 2
		}
	}
	minC, maxC := 0.0, 0.0
	if len(contribs) > 0 {
		minC, maxC = contribs[0], contribs[len(contribs)-1]
	}
	totalStart, totalEnd := snapStart.Savings+snapStart.Shares, snapEnd.Savings+snapEnd.Shares
	savingsRecorded := contributions + shares
	youth, female, male := 0, 0, 0
	for id := range activeEnd {
		if b := band[id]; b == age15to24 || b == age25to35 {
			youth++
		}
		switch gender[id] {
		case "Female":
			female++
		case "Male":
			male++
		}
	}
	withDOB := 0
	for id := range activeEnd {
		if band[id] != ageUnknown {
			withDOB++
		}
	}
	nActive := activeEnd.n()
	memberRates := map[string][2]string{"retentionRate": {"retained", "membersAtStart"}, "attendanceRate": {"attended", "attendanceRows"}}
	accessRates := map[string][2]string{"regularSaversPct": {"regularSavers", "eligibleSavers"}, "borrowersPct": {"borrowers", "activeMembers"}}

	purposeRows := []datatype.DataMap{}
	for _, p := range append(append([]string{}, loanPurposes...), purposeUnspecified) {
		r := byPurpose.rows[p]
		if r == nil {
			continue
		}
		purposeRows = append(purposeRows, datatype.DataMap{"purpose": p, "incomeGenerating": igaPurposes[p], "loans": int(r["loans"]),
			"borrowers": purposeBorrowers[p].n(), "amount": round2(r["amount"])})
	}
	entRows := []datatype.DataMap{}
	for _, k := range byEnterprise.order {
		r := byEnterprise.rows[k]
		entRows = append(entRows, datatype.DataMap{"enterpriseType": enterpriseLabel[k], "loans": int(r["loans"]),
			"borrowers": enterpriseBorrowers[k].n(), "amount": round2(r["amount"])})
	}
	sort.SliceStable(entRows, func(i, j int) bool {
		ai, aj := helper.GetValueOfFloat(entRows[i], "amount"), helper.GetValueOfFloat(entRows[j], "amount")
		if ai != aj {
			return ai > aj
		}
		return lessFold(helper.GetValueOfString(entRows[i], "enterpriseType"), helper.GetValueOfString(entRows[j], "enterpriseType"), "", "")
	})
	if len(entRows) > maxEnterpriseRows {
		entRows = entRows[:maxEnterpriseRows]
	}
	genderAge := []datatype.DataMap{}
	for _, g := range genderKeys {
		for _, b := range ageBands {
			if r := byGenderAge.rows[g+"|"+b]; r != nil {
				genderAge = append(genderAge, datatype.DataMap{"gender": g, "ageBand": b,
					"activeMembers": int(r["activeMembers"]), "newMembers": int(r["newMembers"])})
			}
		}
	}

	nGroups := len(c.groupIDs)
	activeGroups := 0
	for _, k := range c.kpis {
		if k.Active {
			activeGroups++
		}
	}

	extras := &monitorExtras{byAge: byAge, byGenderAge: byGenderAge}
	for _, m := range c.members {
		id := helper.GetValueOfString(m, "id")
		extras.members = append(extras.members, memberStat{m: m, gender: gender[id], band: band[id],
			saved: contribByMember[id] - sharesByMember[id], shares: sharesByMember[id], balance: balEnd[id], owed: owedEnd[id],
			meetingsSaved: len(saversAt[id]), meetingsHeld: heldFor[id], regular: regular[id], eligible: eligible[id],
			active: activeEnd[id], newMem: newMembers[id]})
	}

	return datatype.DataMap{
		"savings": datatype.DataMap{
			"savingsRecorded": round2(savingsRecorded), "contributions": round2(contributions),
			"mandatorySavings": round2(mandatory), "voluntarySavings": round2(voluntary), "sharesPurchased": round2(shares),
			"withdrawals": round2(withdrawals), "netSavings": round2(savingsRecorded - withdrawals),
			"savingsBalance": round2(snapEnd.Savings), "sharesBalance": round2(snapEnd.Shares), "totalBalance": round2(totalEnd),
			"totalBalanceAtStart": round2(totalStart), "growthAmount": round2(totalEnd - totalStart), "growthPct": changePct(totalStart, totalEnd),
			"contributingMembers": len(contribByMember), "avgPerContributor": avg(savingsRecorded, len(contribByMember)),
			"avgBalancePerMember":   avg(sumBalEnd, nActive),
			"memberContributionMin": round2(minC), "memberContributionMedian": round2(median), "memberContributionMax": round2(maxC),
			"byGender":  byGender.list("gender", []string{"savers"}, []string{"savingsRecorded", "savingsBalance"}, nil),
			"byAgeBand": byAge.list("ageBand", []string{"savers"}, []string{"savingsRecorded", "savingsBalance"}, nil),
		},
		"loans": datatype.DataMap{
			"loansIssued": issuedN, "amountDisbursed": round2(issuedAmt), "interestCharged": round2(issuedInterest),
			"borrowers": borrowers.n(), "averageLoanSize": avg(issuedAmt, issuedN),
			"outstandingBalance": round2(snapEnd.Portfolio), "outstandingAtStart": round2(snapStart.Portfolio),
			"activeLoans": activeLoans, "borrowersWithOpenLoans": openBorrowers.n(),
			"byGender":  byGender.list("gender", []string{"borrowers", "loans"}, []string{"loanAmount"}, nil),
			"byAgeBand": byAge.list("ageBand", []string{"borrowers", "loans"}, []string{"loanAmount"}, nil),
		},
		"repayments": datatype.DataMap{
			"repaymentsCollected": round2(repaid),
			"loansFallenDue":      dueInPN, "amountFallenDue": round2(dueInP), "amountRepaidOnFallenDue": round2(repaidInP),
			"repaymentRate":         pct(repaidInP, dueInP),
			"loansFallenDueAllTime": dueAllN, "repaymentRateAllTime": pct(repaidAll, dueAll), "fullyRepaidLoans": fullyRepaidN,
			"overdueLoans": overdueN, "overdueAmount": round2(overdueAmt), "overdueBorrowers": overdueBorrowers.n(),
			"par30Loans": par30N, "par30Amount": round2(par30Amt), "par30Rate": pct(par30Amt, snapEnd.Portfolio),
			"defaultedLoans": defaultN, "defaultedAmount": round2(defaultAmt), "defaultRate": pct(defaultAmt, snapEnd.Portfolio),
			"defaultAfterDays": loanDefaultDays,
		},
		"socialFund": datatype.DataMap{
			"contributions": round2(sfIn), "contributors": sfContributors.n(),
			"disbursements": round2(sfOut), "payouts": sfPayouts, "beneficiaries": sfBeneficiaries.n(),
			"availableBalance": round2(snapEnd.SocialFund), "balanceAtStart": round2(snapStart.SocialFund),
		},
		"membership": datatype.DataMap{
			"activeMembers": nActive, "totalMembers": len(c.members), "newMembers": newMembers.n(),
			"membersAtStart": activeStart.n(), "retained": retained, "exits": activeStart.n() - retained,
			"retentionRate": pct(float64(retained), float64(activeStart.n())),
			"femaleMembers": female, "maleMembers": male, "femalePct": pct(float64(female), float64(nActive)),
			"youthMembers": youth, "youthPct": pct(float64(youth), float64(nActive)),
			"membersWithDateOfBirth": withDOB, "dateOfBirthCoveragePct": pct(float64(withDOB), float64(nActive)),
			"meetingsHeld": len(completed), "attendanceRows": attRows, "present": attP, "late": attL, "absent": attA, "excused": attE,
			"attendanceRate": pct(float64(attP+attL), float64(attRows)),
			"byGender":       byGender.list("gender", []string{"activeMembers", "newMembers", "membersAtStart", "retained"}, nil, memberRates),
			"byAgeBand":      byAge.list("ageBand", []string{"activeMembers", "newMembers", "membersAtStart", "retained"}, nil, memberRates),
			"byGenderAndAge": genderAge,
		},
		"digitalAdoption": datatype.DataMap{
			"groups": nGroups, "activeGroups": activeGroups, "groupsUsingApp": usingApp.n(), "adoptionRate": pct(float64(usingApp.n()), float64(nGroups)),
			"groupsUsingAppInPeriod": usedInPeriod.n(), "activeDays": appActiveDays,
			"transactionsRecorded": txCount - txReversed, "recordingUsers": recorders.n(),
			"meetingsCompleted": len(completed), "meetingsFullyRecorded": fullyRecorded,
			"digitalRecordingRate": pct(float64(fullyRecorded), float64(len(completed))),
		},
		"financialPerformance": datatype.DataMap{
			"interestEarned": round2(interestEarned), "interestCharged": round2(issuedInterest), "finesCollected": round2(fines),
			"operationalExpenses": round2(expenses), "netIncome": round2(interestEarned + fines - expenses),
			"savingsSharedOut": round2(sharedOut), "savingsReturned": round2(svBack + shBack), "returnsToMembers": round2(profit),
			"returnOnSavingsPct": pct(profit, svBack+shBack), "membersPaidOut": paidOut.n(), "shareOutPayments": shareOuts,
			"avgReturnPerMember": avg(profit, paidOut.n()),
		},
		"accountability": datatype.DataMap{
			"transactions": txCount, "reversals": txReversed, "reversalRate": pct(float64(txReversed), float64(txCount)),
			"reversedAmount":        round2(reversedAmt),
			"transactionsAtMeeting": txAtMeeting, "transactionsAtMeetingPct": pct(float64(txAtMeeting), float64(txCount-txReversed)),
			"groupsChecked": nGroups, "groupsWithDrift": groupsWithDrift, "recordsAccuracyPct": pct(float64(nGroups-groupsWithDrift), float64(nGroups)),
			"driftAmount": round2(driftAbs), "overpaidLoans": overpaid,
			"expectedCashOnHand": round2(snapEnd.Cash), "cashOnHandAtStart": round2(snapStart.Cash),
			"auditEntries": auditN, "auditByAction": auditBy.list("action", []string{"count"}, nil, nil), "auditHidden": auditHidden,
			"smsHidden": smsHidden,
			"smsSent":   smsSent, "smsFailed": smsFailed, "smsDeliveryRate": pct(float64(smsSent), float64(smsSent+smsFailed)),
			"membersNotified": notified.n(), "membersNotifiedPct": pct(float64(notified.n()), float64(nActive)),
		},
		"growth": datatype.DataMap{
			"membersStart": snapStart.Members, "membersEnd": snapEnd.Members, "membersGrowthPct": changePct(float64(snapStart.Members), float64(snapEnd.Members)),
			"savingsStart": round2(totalStart), "savingsEnd": round2(totalEnd), "savingsGrowthPct": changePct(totalStart, totalEnd),
			"loanPortfolioStart": round2(snapStart.Portfolio), "loanPortfolioEnd": round2(snapEnd.Portfolio),
			"loanPortfolioGrowthPct": changePct(snapStart.Portfolio, snapEnd.Portfolio),
			"assetsStart":            round2(snapStart.assets()), "assetsEnd": round2(snapEnd.assets()),
			"assetsGrowthPct": changePct(snapStart.assets(), snapEnd.assets()),
			"monthly":         monthly,
		},
		"access": datatype.DataMap{
			"activeMembers": nActive, "eligibleMembers": eligible.n(), "regularSavers": regular.n(),
			"regularSaversPct": pct(float64(regular.n()), float64(eligible.n())), "regularSaverThresholdPct": regularSaverPct,
			"savers": countIn(contribByMember, activeEnd), "saversPct": pct(float64(countIn(contribByMember, activeEnd)), float64(nActive)),
			"borrowers": borrowers.n(), "accessingLoansPct": pct(float64(borrowers.n()), float64(nActive)),
			"membersWithOpenLoans": countSet(openBorrowers, activeEnd), "membersWithOpenLoansPct": pct(float64(countSet(openBorrowers, activeEnd)), float64(nActive)),
			"avgSavingsPerMemberStart": avg(sumBalStart, activeStart.n()), "avgSavingsPerMemberEnd": avg(sumBalEnd, nActive),
			"avgSavingsChangePct":        changePct(avg(sumBalStart, activeStart.n()), avg(sumBalEnd, nActive)),
			"avgBorrowingPerMemberStart": avg(sumOwedStart, activeStart.n()), "avgBorrowingPerMemberEnd": avg(sumOwedEnd, nActive),
			"avgBorrowingChangePct": changePct(avg(sumOwedStart, activeStart.n()), avg(sumOwedEnd, nActive)),
			"avgLoanPerBorrower":    avg(issuedAmt, borrowers.n()),
			"byGender":              byGender.list("gender", []string{"activeMembers", "eligibleSavers", "regularSavers", "borrowers"}, nil, accessRates),
			"byAgeBand":             byAge.list("ageBand", []string{"activeMembers", "eligibleSavers", "regularSavers", "borrowers"}, nil, accessRates),
		},
		"economicBenefits": datatype.DataMap{
			"igaLoans": igaN, "igaBorrowers": igaBorrowers.n(), "igaAmount": round2(igaAmt),
			"igaLoansPct": pct(float64(igaN), float64(issuedN)), "igaBorrowersPct": pct(float64(igaBorrowers.n()), float64(nActive)),
			"loansWithPurpose": specifiedN, "unspecifiedLoans": issuedN - specifiedN,
			"byPurpose": purposeRows, "byEnterpriseType": entRows,
			"igaBorrowersByGender":  byGender.list("gender", []string{"igaBorrowers"}, nil, nil),
			"igaBorrowersByAgeBand": byAge.list("ageBand", []string{"igaBorrowers"}, nil, nil),
			"returnsToMembers":      round2(profit), "avgReturnPerMember": avg(profit, paidOut.n()),
			"returnOnSavingsPct": pct(profit, svBack+shBack), "interestEarned": round2(interestEarned),
		},
		"period": datatype.DataMap{
			"from": c.from.In(eatZone).Format("2006-01-02"), "to": c.to.In(eatZone).Format("2006-01-02"),
			"asAt": asAt.In(eatZone).Format("2006-01-02"), "months": len(monthly),
		},
	}, extras
}

// bandOr is a member's age band ("unknown" for a member out of scope).
func bandOr(band map[string]string, id string) string {
	if b, ok := band[id]; ok {
		return b
	}
	return ageUnknown
}

func countSet(s, in distinct) int {
	n := 0
	for id := range s {
		if in[id] {
			n++
		}
	}
	return n
}

func countIn(m map[string]float64, in distinct) int {
	n := 0
	for id := range m {
		if in[id] {
			n++
		}
	}
	return n
}

// registerMonitoring wires GET /api/admin/monitoring (same scope, filters and
// permission as the dashboard and reports).
func registerMonitoring(app *yekonga.YekongaData) {
	app.Get("/api/admin/monitoring", func(req *yekonga.Request, res *yekonga.Response) {
		from, to, err := reportRange(req.Query("from"), req.Query("to"))
		if err != nil {
			deny(res, 400, err.Error())
			return
		}
		filters := map[string]string{"partnerId": req.Query("partnerId"), "clusterId": req.Query("clusterId"), "groupId": req.Query("groupId")}
		now := time.Now()
		if _, _, _, _, err := monitorPeriod(from, to, now); err != nil {
			deny(res, 400, err.Error())
			return
		}
		a, set, label, ok := reportScope(app, req, res, filters)
		if !ok {
			return
		}
		data := loadMonitorData(app, set)
		data.Hide = &monitorHide{smsOK: func(gid string) bool { return a.CanIn(PermSms, gid) }, noAudit: !a.Can(PermAudit)}
		out := computeMonitoring(data, set, now, from, to)
		groups := out["digitalAdoption"].(datatype.DataMap)["groups"]
		out["scope"] = datatype.DataMap{"label": label, "partnerId": filters["partnerId"], "clusterId": filters["clusterId"],
			"groupId": filters["groupId"], "groups": groups}
		p := out["period"].(datatype.DataMap)
		p["defaultedFrom"] = req.Query("from") == ""
		out["definitions"] = datatype.DataMap{
			"defaultAfterDays": loanDefaultDays, "parDays": parDays, "regularSaverThresholdPct": regularSaverPct,
			"appActiveDays": appActiveDays, "youthAges": "15-35", "ageBands": ageBands,
			"incomeGeneratingPurposes": []string{"business", "agriculture", "livestock"}, "loanPurposes": loanPurposes,
		}
		res.Json(out)
	})
}

// splitMonitorData partitions the records by group (attendance rows without
// a groupId follow their meeting), so per-group indicators cost one pass.
func splitMonitorData(d monitorData) map[string]monitorData {
	out := map[string]monitorData{}
	get := func(gid string) *monitorData {
		x := out[gid]
		x.Clusters = d.Clusters
		out[gid] = x
		return &x
	}
	put := func(gid string, f func(x *monitorData)) {
		x := get(gid)
		f(x)
		out[gid] = *x
	}
	gidOf := func(r datatype.DataMap) string { return helper.GetValueOfString(r, "groupId") }
	for _, r := range d.Groups {
		put(helper.GetValueOfString(r, "id"), func(x *monitorData) { x.Groups = append(x.Groups, r) })
	}
	for _, r := range d.Members {
		put(gidOf(r), func(x *monitorData) { x.Members = append(x.Members, r) })
	}
	for _, r := range d.Loans {
		put(gidOf(r), func(x *monitorData) { x.Loans = append(x.Loans, r) })
	}
	for _, r := range d.GovLoans {
		put(gidOf(r), func(x *monitorData) { x.GovLoans = append(x.GovLoans, r) })
	}
	meetings := map[string]datatype.DataMap{}
	for _, r := range d.Meetings {
		meetings[helper.GetValueOfString(r, "id")] = r
		put(gidOf(r), func(x *monitorData) { x.Meetings = append(x.Meetings, r) })
	}
	for _, r := range d.Attendance {
		put(attendanceGroup(r, meetings), func(x *monitorData) { x.Attendance = append(x.Attendance, r) })
	}
	for _, r := range d.Transactions {
		put(gidOf(r), func(x *monitorData) { x.Transactions = append(x.Transactions, r) })
	}
	for _, r := range d.AuditLogs {
		put(gidOf(r), func(x *monitorData) { x.AuditLogs = append(x.AuditLogs, r) })
	}
	for _, r := range d.SmsLogs {
		put(gidOf(r), func(x *monitorData) { x.SmsLogs = append(x.SmsLogs, r) })
	}
	for _, r := range d.GovRepayments {
		put(gidOf(r), func(x *monitorData) { x.GovRepayments = append(x.GovRepayments, r) })
	}
	return out
}
