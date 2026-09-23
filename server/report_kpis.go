package main

import (
	"math"
	"os"
	"sort"
	"time"

	"github.com/robertkonga/yekonga-server-go/datatype"
	"github.com/robertkonga/yekonga-server-go/helper"
	"github.com/robertkonga/yekonga-server-go/yekonga"
)

// ---------------------------------------------------------------------------
// Programme KPIs, computed per group and rolled up group → cluster → partner →
// organisation (TODO.md §5, D4). One computation feeds the live dashboard,
// the roll-up screens and the "summary-*" report datasets, so the numbers
// always agree with each other.
// ---------------------------------------------------------------------------

// inactiveAfter: a group with no meeting for this long is flagged inactive.
const inactiveAfter = 30 * 24 * time.Hour

// parDays: a loan this many days past its due date counts as "at risk" (PAR 30).
const parDays = 30

type groupKPI struct {
	GroupID, Name, ClusterID, PartnerID string
	Status                              string
	Members, Female, Male               int
	Savings, Shares, SocialFund         float64
	LoansOutstanding, PAR               float64
	GovLoansOutstanding                 float64
	AttendancePresent, AttendanceRows   int
	LastMeeting                         time.Time
	Active                              bool
}

type kpiTotals struct {
	Groups, ActiveGroups          int
	Members, Female, Male         int
	Savings, Shares, SocialFund   float64
	LoansOutstanding, PAR         float64
	GovLoansOutstanding           float64
	AttendancePresent, AttendRows int
}

func (t *kpiTotals) add(k groupKPI) {
	t.Groups++
	if k.Active {
		t.ActiveGroups++
	}
	t.Members += k.Members
	t.Female += k.Female
	t.Male += k.Male
	t.Savings += k.Savings
	t.Shares += k.Shares
	t.SocialFund += k.SocialFund
	t.LoansOutstanding += k.LoansOutstanding
	t.PAR += k.PAR
	t.GovLoansOutstanding += k.GovLoansOutstanding
	t.AttendancePresent += k.AttendancePresent
	t.AttendRows += k.AttendanceRows
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
		"femaleMembers":       t.Female,
		"maleMembers":         t.Male,
		"savings":             t.Savings,
		"shares":              t.Shares,
		"socialFund":          t.SocialFund,
		"loansOutstanding":    t.LoansOutstanding,
		"par30":               t.PAR,
		"par30Rate":           pct(t.PAR, t.LoansOutstanding),
		"govLoansOutstanding": t.GovLoansOutstanding,
		"attendanceRate":      pct(float64(t.AttendancePresent), float64(t.AttendRows)),
	}
}

// computeGroupKPIs returns one groupKPI per group in set (nil set = all).
func computeGroupKPIs(app *yekonga.YekongaData, set map[string]bool) []groupKPI {
	now := time.Now()
	clusterPartner := map[string]string{}
	for _, c := range listAll(app, "Cluster") {
		clusterPartner[helper.GetValueOfString(c, "id")] = helper.GetValueOfString(c, "partnerId")
	}

	byGroup := map[string]*groupKPI{}
	order := []string{}
	for _, g := range listAll(app, "Group") {
		id := helper.GetValueOfString(g, "id")
		if set != nil && !set[id] {
			continue
		}
		cid := helper.GetValueOfString(g, "clusterId")
		k := &groupKPI{
			GroupID: id, Name: helper.GetValueOfString(g, "name"), ClusterID: cid, PartnerID: clusterPartner[cid],
			Status:     helper.GetValueOfString(g, "status"),
			Savings:    helper.GetValueOfFloat(g, "totalSavings"),
			Shares:     helper.GetValueOfFloat(g, "totalShares"),
			SocialFund: helper.GetValueOfFloat(g, "totalSocialFund"),
		}
		// A group formed recently counts as active until it has had time to meet.
		if created := helper.GetTimestamp(g["createdAt"]); !created.IsZero() && now.Sub(created) < inactiveAfter {
			k.Active = true
		}
		byGroup[id] = k
		order = append(order, id)
	}

	for _, m := range listAll(app, "Member") {
		k := byGroup[helper.GetValueOfString(m, "groupId")]
		if k == nil || helper.GetValueOfString(m, "status") == "Inactive" {
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

	for _, l := range listAll(app, "Loan") {
		k := byGroup[helper.GetValueOfString(l, "groupId")]
		if k == nil || helper.GetValueOfString(l, "status") != "active" {
			continue
		}
		bal := math.Max(0, helper.GetValueOfFloat(l, "amount")-helper.GetValueOfFloat(l, "amountRepaid"))
		k.LoansOutstanding += bal
		if due := helper.GetValueOfDate(l, "dueDate"); !due.IsZero() && now.Sub(due) > parDays*24*time.Hour {
			k.PAR += bal
		}
	}

	for _, l := range listAll(app, "GovernmentLoan") {
		k := byGroup[helper.GetValueOfString(l, "groupId")]
		if k == nil {
			continue
		}
		k.GovLoansOutstanding += math.Max(0, govLoanTotalDue(l)-helper.GetValueOfFloat(l, "amountRepaid"))
	}

	for _, mt := range listAll(app, "Meeting") {
		k := byGroup[helper.GetValueOfString(mt, "groupId")]
		if k == nil {
			continue
		}
		st := helper.GetValueOfString(mt, "status")
		if st != "completed" && st != "in_progress" {
			continue
		}
		when, ok := parseMeetingDate(helper.GetValueOfString(mt, "date"))
		if !ok {
			when = helper.GetTimestamp(mt["createdAt"])
		}
		if when.After(k.LastMeeting) {
			k.LastMeeting = when
		}
	}

	for _, a := range listAll(app, "MeetingAttendance") {
		k := byGroup[helper.GetValueOfString(a, "groupId")]
		if k == nil {
			continue
		}
		k.AttendanceRows++
		if st := helper.GetValueOfString(a, "status"); st == "present" || st == "late" {
			k.AttendancePresent++
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

// rollup groups KPIs by level ("partner", "cluster" or "group") and returns
// one row per unit, largest savings first.
func rollup(app *yekonga.YekongaData, kpis []groupKPI, level string) []datatype.DataMap {
	names := map[string]string{}
	switch level {
	case "partner":
		for _, p := range listAll(app, "Partner") {
			names[helper.GetValueOfString(p, "id")] = helper.GetValueOfString(p, "name")
		}
	case "cluster":
		for _, c := range listAll(app, "Cluster") {
			names[helper.GetValueOfString(c, "id")] = helper.GetValueOfString(c, "name")
		}
	}
	totals := map[string]*kpiTotals{}
	keys := []string{}
	for _, k := range kpis {
		key := k.GroupID
		switch level {
		case "partner":
			key = k.PartnerID
		case "cluster":
			key = k.ClusterID
		default:
			names[k.GroupID] = k.Name
		}
		if totals[key] == nil {
			totals[key] = &kpiTotals{}
			keys = append(keys, key)
		}
		totals[key].add(k)
	}
	rows := []datatype.DataMap{}
	for _, key := range keys {
		row := totals[key].json()
		row["id"] = key
		row["name"] = names[key]
		if row["name"] == "" {
			row["name"] = "—"
		}
		rows = append(rows, row)
	}
	sort.SliceStable(rows, func(i, j int) bool {
		return helper.GetValueOfFloat(rows[i], "savings") > helper.GetValueOfFloat(rows[j], "savings")
	})
	return rows
}

// registerDashboard wires the live, scoped dashboard + roll-up endpoints.
func registerDashboard(app *yekonga.YekongaData) {
	// Live programme snapshot for whatever the caller may see, optionally
	// narrowed with ?partnerId= / ?clusterId= / ?groupId=.
	app.Get("/api/admin/dashboard", func(req *yekonga.Request, res *yekonga.Response) {
		set, _, ok := reportScope(app, req, res, map[string]string{
			"partnerId": req.Query("partnerId"), "clusterId": req.Query("clusterId"), "groupId": req.Query("groupId"),
		})
		if !ok {
			return
		}
		kpis := computeGroupKPIs(app, set)
		var totals kpiTotals
		for _, k := range kpis {
			totals.add(k)
		}
		inSet := func(id string) bool { return set == nil || set[id] }

		groupNames := map[string]string{}
		for _, k := range kpis {
			groupNames[k.GroupID] = k.Name
		}
		memberById := map[string]datatype.DataMap{}
		for _, m := range listAll(app, "Member") {
			memberById[helper.GetValueOfString(m, "id")] = m
		}
		typeLabels := map[string]string{
			"contribution": "Saving", "share": "Share", "social_fund": "Social Fund", "loan_disbursement": "Loan",
			"loan_repayment": "Repayment", "fine": "Fine", "expense": "Expense", "withdrawal": "Withdrawal",
		}

		now := time.Now()
		chartByDay := map[string]int{}
		dayOrder := make([]string, 0, 7)
		for i := 6; i >= 0; i-- {
			key := now.AddDate(0, 0, -i).Format("2006-01-02")
			chartByDay[key] = 0
			dayOrder = append(dayOrder, key)
		}
		weekAgo := now.AddDate(0, 0, -7)
		recentActivity := []datatype.DataMap{}
		txns := app.ModelQuery("Transaction").SkipBeforeCommit().OrderBy("createdAt", "desc").Find(nil)
		for _, t := range *txns {
			if helper.GetValueOfBoolean(t, "reversed") || !inSet(helper.GetValueOfString(t, "groupId")) {
				continue
			}
			createdAt := helper.GetValueOfDate(t, "createdAt")
			if !createdAt.IsZero() && !createdAt.Before(weekAgo) {
				if _, ok := chartByDay[createdAt.Format("2006-01-02")]; ok {
					chartByDay[createdAt.Format("2006-01-02")]++
				}
			}
			if len(recentActivity) >= 8 {
				continue
			}
			typ := helper.GetValueOfString(t, "type")
			label := typeLabels[typ]
			if label == "" {
				label = typ
			}
			memberName := ""
			if mm, ok := memberById[helper.GetValueOfString(t, "memberId")]; ok {
				memberName = memberFullName(mm)
			}
			recentActivity = append(recentActivity, datatype.DataMap{
				"time":      createdAt.Format("15:04"),
				"member":    memberName,
				"group":     groupNames[helper.GetValueOfString(t, "groupId")],
				"type":      label,
				"amount":    helper.GetValueOfFloat(t, "amount"),
				"direction": helper.GetValueOfString(t, "direction"),
			})
		}
		chart := make([]datatype.DataMap, 0, 7)
		for _, key := range dayOrder {
			d, _ := time.Parse("2006-01-02", key)
			chart = append(chart, datatype.DataMap{"day": d.Format("Mon"), "count": chartByDay[key]})
		}

		res.Json(datatype.DataMap{
			"totalGroups":           totals.Groups,
			"totalMembers":          totals.Members,
			"totalSavings":          totals.Savings,
			"totalLoansOutstanding": totals.LoansOutstanding,
			"kpis":                  totals.json(),
			"chart":                 chart,
			"recentActivity":        recentActivity,
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
		set, _, ok := reportScope(app, req, res, map[string]string{
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
}
