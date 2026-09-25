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

// reportQuery loads a model's records limited, in the DB query itself, to
// the group set (nil = every group) and — when dateField is given — to the
// [from, to] range on that field. Callers still re-check both in Go.
func reportQuery(app *yekonga.YekongaData, model string, set map[string]bool, dateField string, from, to time.Time) []datatype.DataMap {
	q := app.ModelQuery(model).SkipBeforeCommit()
	if set != nil {
		if len(set) == 0 {
			return nil
		}
		ids := make([]interface{}, 0, len(set))
		for _, id := range sortedKeys(set) {
			ids = append(ids, id)
		}
		field := "groupId"
		if model == "Group" {
			field = "id"
		}
		q = q.Where(field, map[string]interface{}{"in": ids})
	}
	if dateField != "" {
		cond := map[string]interface{}{}
		if !from.IsZero() {
			cond["greaterThanOrEqualTo"] = from.UTC()
		}
		if !to.IsZero() {
			cond["lessThanOrEqualTo"] = to.UTC()
		}
		if len(cond) > 0 {
			q = q.Where(dateField, cond)
		}
	}
	recs := q.Find(nil)
	if recs == nil {
		return nil
	}
	return *recs
}

// reportCtx is one report request: its scope, date range and the records it
// has loaded. Lookups (groups, members, clusters, partners) load once and are
// shared by every dataset in the request (an export may build several).
type reportCtx struct {
	set           map[string]bool // nil = every group
	from, to, now time.Time
	// fetch loads a model's records (already limited to set; dateField, when
	// not empty, may also be limited to the range in the query).
	fetch func(model, dateField string) []datatype.DataMap
	// byIds loads records of model by id, whatever their group (a member who
	// moved groups still has rows in the old group).
	byIds func(model string, ids []string) []datatype.DataMap
	// smsOK reports whether the caller may see SMS logs of a group.
	smsOK func(groupId string) bool

	cache       map[string][]datatype.DataMap
	groups      []datatype.DataMap // in scope, sorted by name
	groupById   map[string]datatype.DataMap
	memberById  map[string]datatype.DataMap
	clusterById map[string]datatype.DataMap
	partnerById map[string]datatype.DataMap
	meetingById map[string]datatype.DataMap
	kpis        []groupKPI
	kpisReady   bool
}

func newReportCtx(app *yekonga.YekongaData, a *Actor, set map[string]bool, from, to time.Time) *reportCtx {
	c := &reportCtx{set: set, from: from, to: to, now: time.Now()}
	c.fetch = func(model, dateField string) []datatype.DataMap {
		if model == "Cluster" || model == "Partner" {
			return listAll(app, model)
		}
		return reportQuery(app, model, set, dateField, from, to)
	}
	c.byIds = func(model string, ids []string) []datatype.DataMap {
		list := make([]interface{}, 0, len(ids))
		for _, id := range ids {
			list = append(list, id)
		}
		recs := app.ModelQuery(model).SkipBeforeCommit().Where("id", map[string]interface{}{"in": list}).Find(nil)
		if recs == nil {
			return nil
		}
		return *recs
	}
	c.smsOK = func(gid string) bool { return gid != "" && a.CanIn(PermSms, gid) }
	c.init()
	return c
}

// needMembers makes sure every member referenced by recs (memberId) can be
// named — including members who have since moved to a group outside the
// scope. One query for all the missing ids.
func (c *reportCtx) needMembers(recs []datatype.DataMap) {
	missing := map[string]bool{}
	for _, r := range recs {
		if id := helper.GetValueOfString(r, "memberId"); id != "" {
			if _, ok := c.memberById[id]; !ok {
				missing[id] = true
			}
		}
	}
	if len(missing) == 0 || c.byIds == nil {
		return
	}
	for _, m := range c.byIds("Member", sortedKeys(missing)) {
		c.memberById[helper.GetValueOfString(m, "id")] = m
	}
	for id := range missing { // unknown ids: don't ask again
		if _, ok := c.memberById[id]; !ok {
			c.memberById[id] = nil
		}
	}
}

func (c *reportCtx) init() {
	c.cache = map[string][]datatype.DataMap{}
	c.groupById = map[string]datatype.DataMap{}
	c.groups = nil
	for _, g := range c.list("Group") {
		id := helper.GetValueOfString(g, "id")
		if c.set != nil && !c.set[id] {
			continue
		}
		c.groupById[id] = g
		c.groups = append(c.groups, g)
	}
	sort.SliceStable(c.groups, func(i, j int) bool {
		return lessFold(helper.GetValueOfString(c.groups[i], "name"), helper.GetValueOfString(c.groups[j], "name"),
			helper.GetValueOfString(c.groups[i], "id"), helper.GetValueOfString(c.groups[j], "id"))
	})
	c.memberById = map[string]datatype.DataMap{}
	for _, m := range c.list("Member") {
		if c.inScope(helper.GetValueOfString(m, "groupId")) {
			c.memberById[helper.GetValueOfString(m, "id")] = m
		}
	}
	c.clusterById = indexById(c.list("Cluster"))
	c.partnerById = indexById(c.list("Partner"))
}

func indexById(recs []datatype.DataMap) map[string]datatype.DataMap {
	out := map[string]datatype.DataMap{}
	for _, r := range recs {
		out[helper.GetValueOfString(r, "id")] = r
	}
	return out
}

// list is every record of model in scope (cached for the request).
func (c *reportCtx) list(model string) []datatype.DataMap {
	if recs, ok := c.cache[model]; ok {
		return recs
	}
	recs := c.fetch(model, "")
	c.cache[model] = recs
	return recs
}

// listInRange is like list, but lets the query drop records outside the
// date range on dateField (only for fields that are always written as real
// dates by the server, e.g. Transaction.createdAt, SmsLog.sentAt).
func (c *reportCtx) listInRange(model, dateField string) []datatype.DataMap {
	if c.from.IsZero() && c.to.IsZero() {
		return c.list(model)
	}
	key := model + "@" + dateField
	if recs, ok := c.cache[key]; ok {
		return recs
	}
	recs := c.fetch(model, dateField)
	c.cache[key] = recs
	return recs
}

func (c *reportCtx) inScope(groupId string) bool {
	_, ok := c.groupById[groupId]
	return ok
}

func (c *reportCtx) rangeSet() bool { return !c.from.IsZero() || !c.to.IsZero() }

func (c *reportCtx) inRange(t time.Time) bool { return timeInRange(t, c.from, c.to) }

func (c *reportCtx) groupName(id string) string {
	return helper.GetValueOfString(c.groupById[id], "name")
}

func (c *reportCtx) memberName(id string) string {
	if m, ok := c.memberById[id]; ok && m != nil {
		return memberFullName(m)
	}
	return "—"
}

func (c *reportCtx) meetings() map[string]datatype.DataMap {
	if c.meetingById == nil {
		c.meetingById = indexById(c.list("Meeting"))
	}
	return c.meetingById
}

func (c *reportCtx) groupKPIs() []groupKPI {
	if !c.kpisReady {
		c.kpis = computeKPIs(kpiData{
			Groups: c.groups, Clusters: c.list("Cluster"), Members: c.list("Member"), Loans: c.list("Loan"),
			GovLoans: c.list("GovernmentLoan"), Meetings: c.list("Meeting"), Attendance: c.list("MeetingAttendance"),
			Transactions: c.list("Transaction"),
		}, c.set, c.now, c.from, c.to)
		c.kpisReady = true
	}
	return c.kpis
}

// lessFold orders by a (case-insensitive), then by the tie-breaker b.
func lessFold(a1, a2, b1, b2 string) bool {
	l1, l2 := strings.ToLower(a1), strings.ToLower(a2)
	if l1 != l2 {
		return l1 < l2
	}
	return b1 < b2
}

// hidden row keys ("_…") carry the parts of a rate so totals can recompute
// it; they are never shown or exported.
func stripHidden(rows []datatype.DataMap) []datatype.DataMap {
	out := make([]datatype.DataMap, len(rows))
	for i, r := range rows {
		clean := datatype.DataMap{}
		for k, v := range r {
			if !strings.HasPrefix(k, "_") {
				clean[k] = v
			}
		}
		out[i] = clean
	}
	return out
}

// sortByTimeDesc orders rows newest first (by the hidden "_t" time), then by
// the hidden "_id" so equal times keep a fixed order.
func sortByTimeDesc(rows []datatype.DataMap) {
	sort.SliceStable(rows, func(i, j int) bool {
		ti, _ := rows[i]["_t"].(time.Time)
		tj, _ := rows[j]["_t"].(time.Time)
		if !ti.Equal(tj) {
			return ti.After(tj)
		}
		return fmt.Sprint(rows[i]["_id"]) < fmt.Sprint(rows[j]["_id"])
	})
}

// sortByKeys orders rows by the given text/number columns in turn.
func sortByKeys(rows []datatype.DataMap, keys ...string) {
	sort.SliceStable(rows, func(i, j int) bool {
		for _, k := range keys {
			a, b := rows[i][k], rows[j][k]
			switch av := a.(type) {
			case int:
				bv, _ := b.(int)
				if av != bv {
					return av < bv
				}
				continue
			case float64:
				bv, _ := b.(float64)
				if av != bv {
					return av < bv
				}
				continue
			}
			as, bs := strings.ToLower(fmt.Sprint(a)), strings.ToLower(fmt.Sprint(b))
			if as != bs {
				return as < bs
			}
		}
		return fmt.Sprint(rows[i]["_id"]) < fmt.Sprint(rows[j]["_id"])
	})
}

// buildReport is the single source for both the live preview
// (GET /api/admin/reports) and every export. It returns rows keyed by the
// dataset's registry columns (plus hidden "_…" keys); ok is false for an
// unknown key. Every dataset has a fixed row order.
func buildReport(c *reportCtx, key string) ([]datatype.DataMap, bool) {
	if ds := findDataset(key); ds != nil {
		key = ds.Key
	}
	rows := []datatype.DataMap{}

	switch key {
	case "summary-partner", "summary-cluster", "summary-group":
		level := strings.TrimPrefix(key, "summary-")
		kpis := c.groupKPIs()
		names := map[string]string{}
		switch level {
		case "partner":
			for id, p := range c.partnerById {
				names[id] = helper.GetValueOfString(p, "name")
			}
		case "cluster":
			for id, cl := range c.clusterById {
				names[id] = helper.GetValueOfString(cl, "name")
			}
		default:
			for _, k := range kpis {
				names[k.GroupID] = k.Name
			}
		}
		statuses := map[string]interface{}{}
		switch level {
		case "partner":
			for id, p := range c.partnerById {
				statuses[id] = unitStatus(p)
			}
		case "cluster":
			for id, cl := range c.clusterById {
				statuses[id] = unitStatus(cl)
			}
		default:
			for _, k := range kpis {
				statuses[k.GroupID] = k.Status
			}
		}
		keys, totals := rollupTotals(kpis, level)
		for _, id := range keys {
			row := summaryRow(*totals[id])
			row["Name"] = names[id]
			row["Status"] = statuses[id]
			if id == "" || names[id] == "" {
				// Groups with no cluster / partner: one honest row, no status.
				row["Name"], row["Status"] = notAssigned, nil
			}
			row["_id"] = id
			rows = append(rows, row)
		}
		sortByKeys(rows, "Name")

	case "group-performance":
		for _, k := range c.groupKPIs() {
			var t kpiTotals
			t.add(k)
			row := summaryRow(t)
			activity := "Active"
			if !k.Active {
				activity = "Inactive (no meeting in 30 days)"
			}
			cl := c.clusterById[k.ClusterID]
			row["Group"] = k.Name
			row["Partner"] = helper.GetValueOfString(c.partnerById[k.PartnerID], "name")
			row["Cluster"] = helper.GetValueOfString(cl, "name")
			row["Region"] = k.Region
			row["Last Meeting"] = dateVal(k.LastMeeting)
			row["Activity"] = activity
			row["Status"] = k.Status
			row["_id"] = k.GroupID
			rows = append(rows, row)
		}
		sortByKeys(rows, "Group")

	case "portfolio-at-risk":
		c.needMembers(c.list("Loan"))
		for _, l := range c.list("Loan") {
			gid := helper.GetValueOfString(l, "groupId")
			bal := loanBalance(l)
			if !c.inScope(gid) || bal <= 0 {
				continue
			}
			due := parseTimeOrZero(l["dueDate"])
			if due.IsZero() || c.now.Sub(due) <= parDays*24*time.Hour {
				continue
			}
			rows = append(rows, datatype.DataMap{
				"Loan #": helper.GetValueOfString(l, "loanNumber"), "Group": c.groupName(gid),
				"Borrower": c.memberName(helper.GetValueOfString(l, "memberId")),
				"Balance":  bal, "Due": dateVal(due), "Days Overdue": int(c.now.Sub(due).Hours() / 24),
				"_id": helper.GetValueOfString(l, "id"), "_od": -int(c.now.Sub(due).Hours() / 24),
			})
		}
		sortByKeys(rows, "_od", "Group", "Loan #")

	case "government-loans":
		for _, l := range c.list("GovernmentLoan") {
			gid := helper.GetValueOfString(l, "groupId")
			received := parseTimeOrZero(l["receivedDate"])
			if !c.inScope(gid) || !c.inRange(received) {
				continue
			}
			rec := govLoanJSON(l, nil)
			rows = append(rows, datatype.DataMap{
				"Group": c.groupName(gid), "Lender": helper.GetValueOfString(l, "lender"),
				"Programme": helper.GetValueOfString(l, "programme"), "Reference": helper.GetValueOfString(l, "reference"),
				"Principal": helper.GetValueOfFloat(l, "amount"), "Interest %": helper.GetValueOfFloat(l, "interestRate"),
				"Total Due": rec["totalDue"], "Repaid": helper.GetValueOfFloat(l, "amountRepaid"), "Balance": rec["outstanding"],
				"Issued": dateVal(received), "Due": dateVal(parseTimeOrZero(l["dueDate"])), "Status": helper.GetValueOfString(l, "status"),
				"_t": received, "_id": helper.GetValueOfString(l, "id"),
			})
		}
		sortByTimeDesc(rows)

	case "gov-loan-repayments":
		loans := indexById(c.list("GovernmentLoan"))
		for _, r := range c.list("GovernmentLoanRepayment") {
			gid := helper.GetValueOfString(r, "groupId")
			when := parseTimeOrZero(r["date"])
			if when.IsZero() {
				when = parseTimeOrZero(r["createdAt"])
			}
			if !c.inScope(gid) || !c.inRange(when) {
				continue
			}
			l := loans[helper.GetValueOfString(r, "governmentLoanId")]
			rows = append(rows, datatype.DataMap{
				"Date": dateVal(when), "Group": c.groupName(gid), "Lender": helper.GetValueOfString(l, "lender"),
				"Programme": helper.GetValueOfString(l, "programme"), "Loan Reference": helper.GetValueOfString(l, "reference"),
				"Amount": helper.GetValueOfFloat(r, "amount"), "Method": helper.GetValueOfString(r, "method"),
				"Reference": helper.GetValueOfString(r, "reference"),
				"_t":        when, "_id": helper.GetValueOfString(r, "id"),
			})
		}
		sortByTimeDesc(rows)

	case "group-growth":
		kpiBy := map[string]groupKPI{}
		for _, k := range c.groupKPIs() {
			kpiBy[k.GroupID] = k
		}
		for _, g := range c.groups {
			id := helper.GetValueOfString(g, "id")
			k := kpiBy[id]
			rows = append(rows, datatype.DataMap{
				"Group": helper.GetValueOfString(g, "name"), "Formed": dateVal(parseTimeOrZero(g["formationDate"])),
				"Members (active)": k.Members, "Members (total)": k.MembersTotal, "New Members (period)": k.Period.NewMembers,
				"Cycle":             fmt.Sprintf("%d/%d", helper.GetValueOfInt(g, "cycleCurrent"), helper.GetValueOfInt(g, "cycleTotal")),
				"Meeting Frequency": helper.GetValueOfString(g, "meetingFrequency"),
				"_id":               id,
			})
		}

	case "cycles":
		for _, g := range c.groups {
			rows = append(rows, datatype.DataMap{
				"Group": helper.GetValueOfString(g, "name"), "Cycle Current": helper.GetValueOfInt(g, "cycleCurrent"),
				"Cycle Total": helper.GetValueOfInt(g, "cycleTotal"), "Meeting Frequency": helper.GetValueOfString(g, "meetingFrequency"),
				"_id": helper.GetValueOfString(g, "id"),
			})
		}

	case "savings", "shares", "social-fund", "fines", "transactions", "expenses":
		types := map[string]map[string]bool{
			"savings": {"contribution": true}, "shares": {"share": true}, "social-fund": {"social_fund": true},
			"fines": {"fine": true}, "expenses": {"expense": true, "withdrawal": true},
		}[key]
		c.needMembers(c.listInRange("Transaction", "createdAt"))
		for _, t := range c.listInRange("Transaction", "createdAt") {
			gid := helper.GetValueOfString(t, "groupId")
			typ := helper.GetValueOfString(t, "type")
			if !txCounted(t) || !c.inScope(gid) || (types != nil && !types[typ]) {
				continue
			}
			when := parseTimeOrZero(t["createdAt"])
			if !c.inRange(when) {
				continue
			}
			rows = append(rows, datatype.DataMap{
				"Date": dateVal(when), "Group": c.groupName(gid),
				"Member": c.memberName(helper.GetValueOfString(t, "memberId")), "Type": reportTxLabel(typ),
				"Amount": helper.GetValueOfFloat(t, "amount"), "Direction": helper.GetValueOfString(t, "direction"),
				"Method": helper.GetValueOfString(t, "method"), "Reference": helper.GetValueOfString(t, "reference"),
				"Description": helper.GetValueOfString(t, "description"),
				"_t":          when, "_id": helper.GetValueOfString(t, "id"),
			})
		}
		sortByTimeDesc(rows)

	case "loans":
		c.needMembers(c.list("Loan"))
		for _, l := range c.list("Loan") {
			gid := helper.GetValueOfString(l, "groupId")
			issued := parseTimeOrZero(l["issuedDate"])
			if issued.IsZero() {
				issued = parseTimeOrZero(l["createdAt"])
			}
			if !c.inScope(gid) || !c.inRange(issued) {
				continue
			}
			// A cancelled loan owes nothing but keeps its real repaid amount; it
			// was never lent, so it stays out of the totals.
			rows = append(rows, datatype.DataMap{
				"Loan #": helper.GetValueOfString(l, "loanNumber"), "Group": c.groupName(gid),
				"Borrower":   c.memberName(helper.GetValueOfString(l, "memberId")),
				"Principal":  helper.GetValueOfFloat(l, "amount"),
				"Interest %": helper.GetValueOfFloat(l, "interestRate"),
				"Interest":   loanInterest(l), "Total Due": loanTotalDue(l),
				"Repaid": helper.GetValueOfFloat(l, "amountRepaid"), "Balance": loanBalance(l),
				"Overpaid": loanOverpaid(l),
				"Status":   helper.GetValueOfString(l, "status"), "Issued": dateVal(issued),
				"Due": dateVal(parseTimeOrZero(l["dueDate"])),
				"_t":  issued, "_id": helper.GetValueOfString(l, "id"),
				"_noTotals": helper.GetValueOfString(l, "status") == "cancelled",
			})
		}
		sortByTimeDesc(rows)

	case "meetings":
		for _, mt := range c.list("Meeting") {
			gid := helper.GetValueOfString(mt, "groupId")
			when := meetingWhen(mt)
			if !c.inScope(gid) || !c.inRange(when) {
				continue
			}
			rows = append(rows, datatype.DataMap{
				"Group":     c.groupName(gid),
				"Meeting #": helper.GetValueOfInt(mt, "meetingNumber"), "Title": helper.GetValueOfString(mt, "title"),
				"Date": dateVal(when), "Status": helper.GetValueOfString(mt, "status"),
				"_id": helper.GetValueOfString(mt, "id"),
			})
		}
		sortByKeys(rows, "Group", "Meeting #")

	case "meeting-collections":
		rows = c.meetingCollections()

	case "attendance":
		meetings := c.meetings()
		c.needMembers(c.list("MeetingAttendance"))
		for _, a := range c.list("MeetingAttendance") {
			mt := meetings[helper.GetValueOfString(a, "meetingId")]
			if helper.GetValueOfString(mt, "status") == "cancelled" {
				continue
			}
			gid := attendanceGroup(a, meetings)
			when := meetingWhen(mt)
			if when.IsZero() {
				when = parseTimeOrZero(a["createdAt"])
			}
			if !c.inScope(gid) || !c.inRange(when) {
				continue
			}
			var number interface{}
			if mt != nil {
				number = helper.GetValueOfInt(mt, "meetingNumber")
			}
			rows = append(rows, datatype.DataMap{
				"Group": c.groupName(gid), "Meeting #": number, "Meeting Date": dateVal(when),
				"Member": c.memberName(helper.GetValueOfString(a, "memberId")),
				"Status": helper.GetValueOfString(a, "status"), "Recorded": dateVal(parseTimeOrZero(a["createdAt"])),
				"_id": helper.GetValueOfString(a, "id"), "_n": helper.GetValueOfInt(mt, "meetingNumber"),
			})
		}
		sortByKeys(rows, "Group", "_n", "Member")

	case "members":
		for _, m := range c.list("Member") {
			joined := memberJoined(m)
			if !c.inScope(helper.GetValueOfString(m, "groupId")) || !c.inRange(joined) {
				continue
			}
			rows = append(rows, datatype.DataMap{
				"Group": c.groupName(helper.GetValueOfString(m, "groupId")), "Name": memberFullName(m),
				"Phone": helper.GetValueOfString(m, "phone"), "Gender": helper.GetValueOfString(m, "gender"),
				"Member #": helper.GetValueOfString(m, "memberNumber"), "Status": helper.GetValueOfString(m, "status"),
				"Joined": dateVal(joined), "_id": helper.GetValueOfString(m, "id"),
			})
		}
		sortByKeys(rows, "Group", "Name")

	case "fines-outstanding":
		c.needMembers(c.list("Fine"))
		for _, f := range c.list("Fine") {
			gid := helper.GetValueOfString(f, "groupId")
			when := parseTimeOrZero(f["issuedAt"])
			if when.IsZero() {
				when = parseTimeOrZero(f["createdAt"])
			}
			if !c.inScope(gid) || !c.inRange(when) {
				continue
			}
			charged := helper.GetValueOfFloat(f, "amount")
			paid := helper.GetValueOfFloat(f, "amountPaid")
			status := helper.GetValueOfString(f, "status")
			outstanding := math.Max(0, charged-paid)
			if status == "waived" {
				outstanding = 0
			}
			rows = append(rows, datatype.DataMap{
				"Group": c.groupName(gid), "Member": c.memberName(helper.GetValueOfString(f, "memberId")),
				"Reason": helper.GetValueOfString(f, "reason"), "Date": dateVal(when),
				"Charged": charged, "Paid": paid, "Outstanding": outstanding, "Status": status,
				"_t": when, "_id": helper.GetValueOfString(f, "id"),
			})
		}
		sortByTimeDesc(rows)

	case "sms-usage", "sms-delivery":
		type agg struct{ sent, failed, other int }
		byKey := map[[2]string]*agg{}
		if key == "sms-usage" {
			c.needMembers(c.listInRange("SmsLog", "sentAt"))
		}
		for _, l := range c.listInRange("SmsLog", "sentAt") {
			gid := helper.GetValueOfString(l, "groupId")
			// OTP and other groupless messages are never part of a report.
			if gid == "" || !c.inScope(gid) || c.smsOK == nil || !c.smsOK(gid) {
				continue
			}
			when := parseTimeOrZero(l["sentAt"])
			if when.IsZero() {
				when = parseTimeOrZero(l["createdAt"])
			}
			if !c.inRange(when) {
				continue
			}
			typ, status := helper.GetValueOfString(l, "messageType"), helper.GetValueOfString(l, "status")
			if key == "sms-usage" {
				rows = append(rows, datatype.DataMap{
					"Date": dateVal(when), "Group": c.groupName(gid),
					"Member": c.memberName(helper.GetValueOfString(l, "memberId")),
					"Phone":  helper.GetValueOfString(l, "phone"), "Message Type": typ, "Status": status,
					"_t": when, "_id": helper.GetValueOfString(l, "id"),
				})
				continue
			}
			k := [2]string{gid, typ}
			if byKey[k] == nil {
				byKey[k] = &agg{}
			}
			switch status {
			case "sent":
				byKey[k].sent++
			case "failed":
				byKey[k].failed++
			default:
				byKey[k].other++
			}
		}
		if key == "sms-usage" {
			sortByTimeDesc(rows)
			break
		}
		for k, a := range byKey {
			total := a.sent + a.failed + a.other
			rows = append(rows, datatype.DataMap{
				"Group": c.groupName(k[0]), "Message Type": k[1], "Sent": a.sent, "Failed": a.failed,
				"Other": a.other, "Total": total, "Delivery %": pct(float64(a.sent), float64(total)),
				"_id": k[0] + "/" + k[1],
			})
		}
		sortByKeys(rows, "Group", "Message Type")

	default:
		return nil, false
	}

	return rows, true
}

// summaryRow is one summary / group-performance row from summed KPIs. The
// hidden "_…" parts let reportTotals recompute the rates.
func summaryRow(t kpiTotals) datatype.DataMap {
	return datatype.DataMap{
		"Groups": t.Groups, "Active Groups": t.ActiveGroups, "Members (active)": t.Members, "Members (total)": t.MembersTotal,
		"Female": t.Female, "Male": t.Male, "Savings Balance": t.Savings, "Shares Balance": t.Shares,
		"Social Fund Balance": t.SocialFund, "Loans Outstanding": t.LoansOutstanding, "PAR 30 %": pct(t.PAR, t.LoansOutstanding),
		"Government Loans": t.GovLoansOutstanding, "Attendance %": pct(float64(t.AttendancePresent), float64(t.AttendRows)),
		"Savings (period)": t.Period.Savings, "Shares (period)": t.Period.Shares, "Social Fund (period)": t.Period.SocialFund,
		"Fines (period)": t.Period.Fines, "Loan Repayments (period)": t.Period.LoanRepayments,
		"Loans Disbursed (period)": t.Period.LoansDisbursed, "Expenses (period)": t.Period.Expenses,
		"Meetings Held (period)": t.Period.Meetings,
		"Attendance % (period)":  pct(float64(t.Period.AttendancePresent), float64(t.Period.AttendanceRows)),
		"_par":                   t.PAR, "_lo": t.LoansOutstanding,
		"_attP": t.AttendancePresent, "_attN": t.AttendRows,
		"_pattP": t.Period.AttendancePresent, "_pattN": t.Period.AttendanceRows,
	}
}

// meetingCollections: one row per meeting in the range with its attendance
// and the money recorded at it. A transaction belongs to the meeting named
// by its meetingId; one recorded without a meetingId falls back to the
// group's only (not cancelled) meeting on the same EAT day, if there is
// exactly one.
func (c *reportCtx) meetingCollections() []datatype.DataMap {
	type sums struct {
		savings, shares, social, fines, repay, disb, expenses, withdrawals float64
		present, late, absent, excused                                     int
	}
	byMeeting := map[string]*sums{}
	get := func(id string) *sums {
		if byMeeting[id] == nil {
			byMeeting[id] = &sums{}
		}
		return byMeeting[id]
	}
	meetings := c.meetings()
	sameDay := map[string][]string{} // groupId|EAT date -> meeting ids
	for id, mt := range meetings {
		if helper.GetValueOfString(mt, "status") == "cancelled" {
			continue
		}
		if when := meetingWhen(mt); !when.IsZero() {
			k := helper.GetValueOfString(mt, "groupId") + "|" + when.In(eatZone).Format("2006-01-02")
			sameDay[k] = append(sameDay[k], id)
		}
	}
	for _, t := range c.list("Transaction") {
		if !txCounted(t) {
			continue
		}
		mid := helper.GetValueOfString(t, "meetingId")
		if _, ok := meetings[mid]; !ok {
			mid = ""
			if when := parseTimeOrZero(t["createdAt"]); !when.IsZero() {
				if ids := sameDay[helper.GetValueOfString(t, "groupId")+"|"+when.In(eatZone).Format("2006-01-02")]; len(ids) == 1 {
					mid = ids[0]
				}
			}
		}
		if mid == "" {
			continue
		}
		s, amt := get(mid), helper.GetValueOfFloat(t, "amount")
		switch helper.GetValueOfString(t, "type") {
		case "contribution":
			s.savings += amt
		case "share":
			s.shares += amt
		case "social_fund":
			s.social += amt
		case "fine":
			s.fines += amt
		case "loan_repayment":
			s.repay += amt
		case "loan_disbursement":
			s.disb += amt
		case "expense":
			s.expenses += amt
		case "withdrawal":
			s.withdrawals += amt
		}
	}
	for _, a := range c.list("MeetingAttendance") {
		s := get(helper.GetValueOfString(a, "meetingId"))
		switch helper.GetValueOfString(a, "status") {
		case "present":
			s.present++
		case "late":
			s.late++
		case "absent":
			s.absent++
		case "excused":
			s.excused++
		}
	}

	rows := []datatype.DataMap{}
	for id, mt := range meetings {
		gid := helper.GetValueOfString(mt, "groupId")
		when := meetingWhen(mt)
		if !c.inScope(gid) || !c.inRange(when) {
			continue
		}
		s := get(id)
		rows = append(rows, datatype.DataMap{
			"Group": c.groupName(gid), "Meeting #": helper.GetValueOfInt(mt, "meetingNumber"), "Date": dateVal(when),
			"Status": helper.GetValueOfString(mt, "status"), "Present": s.present, "Late": s.late, "Absent": s.absent,
			"Excused": s.excused, "Savings": s.savings, "Shares": s.shares, "Social Fund": s.social, "Fines": s.fines,
			"Loan Repayments": s.repay, "Loans Disbursed": s.disb, "Expenses": s.expenses, "Withdrawals": s.withdrawals,
			"Total In":  s.savings + s.shares + s.social + s.fines + s.repay,
			"Total Out": s.disb + s.expenses + s.withdrawals,
			"_id":       id,
		})
	}
	sortByKeys(rows, "Group", "Meeting #")
	return rows
}

// rateParts says how a totals row recomputes each rate column: the ratio of
// two summed (usually hidden) row values.
var rateParts = map[string][2]string{
	"PAR 30 %":              {"_par", "_lo"},
	"Attendance %":          {"_attP", "_attN"},
	"Attendance % (period)": {"_pattP", "_pattN"},
	"Delivery %":            {"Sent", "Total"},
}

func num(v interface{}) float64 {
	switch n := v.(type) {
	case float64:
		return n
	case int:
		return float64(n)
	case int64:
		return float64(n)
	case int32:
		return float64(n)
	}
	return 0
}

// reportTotals is the TOTAL row of a dataset: money and counts are summed,
// rates are recomputed from their summed parts (never summed themselves),
// everything else (dates, text, ids, a rate with no parts such as
// Interest %) is left empty. Rows marked "_noTotals" (cancelled loans) are
// left out. The label cell is added by the writer.
func reportTotals(ds *reportDataset, all []datatype.DataMap) datatype.DataMap {
	rows := make([]datatype.DataMap, 0, len(all))
	for _, r := range all {
		if skip, _ := r["_noTotals"].(bool); !skip {
			rows = append(rows, r)
		}
	}
	out := datatype.DataMap{}
	for _, col := range ds.Columns {
		switch colType(ds, col) {
		case "money":
			var s float64
			for _, r := range rows {
				s += num(r[col])
			}
			out[col] = math.Round(s*100) / 100
		case "count":
			s := 0
			for _, r := range rows {
				s += int(num(r[col]))
			}
			out[col] = s
		case "rate":
			parts, ok := rateParts[col]
			if !ok {
				continue
			}
			var n, d float64
			for _, r := range rows {
				n += num(r[parts[0]])
				d += num(r[parts[1]])
			}
			out[col] = pct(n, d)
		}
	}
	return out
}
