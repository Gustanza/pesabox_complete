package main

import (
	"archive/zip"
	"bytes"
	"encoding/csv"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/go-pdf/fpdf"
	"github.com/robertkonga/yekonga-server-go/datatype"
	"github.com/robertkonga/yekonga-server-go/helper"
	"github.com/robertkonga/yekonga-server-go/yekonga"
	"github.com/xuri/excelize/v2"
)

// maxExportRows caps each dataset in one export so a single request cannot
// build an unbounded file in memory.
const maxExportRows = 50000

// reportDataset is one selectable dataset in the Reports screen. Key is the
// stable id the web sends back; Columns are the row keys buildReport fills,
// in display order (they double as the column ids in the export picker).
type reportDataset struct {
	Key      string
	Category string
	Sw       string
	En       string
	Columns  []string
	// Downloadable is false for datasets that are only meant to be viewed
	// live on the dashboard (GATA decides which — flip the flag here).
	Downloadable bool
}

var (
	summaryCols = []string{"Name", "Groups", "Active Groups", "Members", "Female", "Male", "Savings", "Shares",
		"Social Fund", "Loans Outstanding", "PAR 30 %", "Government Loans", "Attendance %"}
	txCols   = []string{"Date", "Group", "Member", "Type", "Amount", "Direction"}
	smsCols  = []string{"Date", "Group", "Phone", "Type", "Status"}
	cycCols  = []string{"Group", "Cycle Current", "Cycle Total", "Meeting Frequency"}
	loanCols = []string{"Loan #", "Group", "Borrower", "Principal", "Repaid", "Balance", "Status", "Issued", "Due"}
)

// reportDatasets is the registry every report endpoint reads. New report
// types agreed with GATA are added here plus one case in buildReport.
var reportDatasets = []reportDataset{
	{"summary-partner", "Programme", "Muhtasari kwa Mbia", "Summary by Partner", summaryCols, true},
	{"summary-cluster", "Programme", "Muhtasari kwa Klasta", "Summary by Cluster", summaryCols, true},
	{"summary-group", "Programme", "Muhtasari kwa Kikundi", "Summary by Group", summaryCols, true},
	{"group-performance", "Group", "Utendaji wa Kikundi", "Group Performance", []string{"Group", "Partner", "Cluster", "Region", "Members", "Savings", "Shares", "Loans Outstanding", "PAR 30 %", "Attendance %", "Last Meeting", "Activity", "Status"}, true},
	{"group-activity", "Group", "Shughuli za Kikundi", "Group Activity", txCols, true},
	{"group-growth", "Group", "Ukuaji wa Kikundi", "Group Growth", []string{"Group", "Formed", "Members", "Cycle", "Meeting Frequency"}, true},
	{"savings", "Financial", "Akiba", "Savings", txCols, true},
	{"shares", "Financial", "Hisa", "Shares", txCols, true},
	{"social-fund", "Financial", "Mfuko wa Jamii", "Social Fund", txCols, true},
	{"loans", "Financial", "Mikopo", "Loans", loanCols, true},
	{"portfolio-at-risk", "Financial", "Mikopo Hatarini (PAR 30)", "Portfolio at Risk (PAR 30)", []string{"Loan #", "Group", "Borrower", "Balance", "Due", "Days Overdue"}, true},
	{"government-loans", "Financial", "Mikopo ya Serikali", "Government Loans", []string{"Group", "Lender", "Programme", "Reference", "Principal", "Interest %", "Total Due", "Repaid", "Balance", "Issued", "Due", "Status"}, true},
	{"fines", "Financial", "Faini", "Fines", txCols, true},
	{"transactions", "Financial", "Miamala Yote", "All Transactions", []string{"Date", "Group", "Member", "Type", "Amount", "Direction", "Method", "Reference"}, true},
	{"meetings", "Operations", "Mikutano", "Meetings", []string{"Group", "Meeting #", "Title", "Date", "Status"}, true},
	{"attendance", "Operations", "Mahudhurio", "Attendance", []string{"Group", "Member", "Status", "Recorded"}, true},
	{"cycles", "Operations", "Mizunguko", "Cycles", cycCols, true},
	{"members", "Operations", "Wanachama", "Members", []string{"Group", "Name", "Phone", "Gender", "Member #", "Status", "Joined"}, true},
	{"sms-usage", "Communication", "Matumizi ya SMS", "SMS Usage", smsCols, true},
	{"sms-delivery", "Communication", "Uwasilishaji wa SMS", "SMS Delivery", smsCols, true},
}

func findDataset(key string) *reportDataset {
	for i := range reportDatasets {
		if reportDatasets[i].Key == key {
			return &reportDatasets[i]
		}
	}
	return nil
}

var categorySw = map[string]string{
	"Programme": "Programu", "Group": "Vikundi", "Financial": "Fedha", "Operations": "Uendeshaji", "Communication": "Mawasiliano",
}

var columnSw = map[string]string{
	"Group": "Kikundi", "Region": "Mkoa", "Members": "Wanachama", "Savings": "Akiba", "Shares": "Hisa",
	"Loans Outstanding": "Mikopo Inayodaiwa", "Status": "Hali", "Formed": "Ilianzishwa", "Cycle": "Mzunguko",
	"Meeting Frequency": "Mzunguko wa Mikutano", "Cycle Current": "Mzunguko wa Sasa", "Cycle Total": "Jumla ya Mizunguko",
	"Date": "Tarehe", "Member": "Mwanachama", "Type": "Aina", "Amount": "Kiasi", "Direction": "Mwelekeo",
	"Loan #": "Mkopo #", "Borrower": "Mkopaji", "Principal": "Kiasi cha Mkopo", "Repaid": "Kimelipwa",
	"Balance": "Salio", "Issued": "Ilitolewa", "Due": "Tarehe ya Kulipa", "Meeting #": "Mkutano #",
	"Title": "Kichwa", "Recorded": "Ilirekodiwa", "Phone": "Simu", "Name": "Jina", "Gender": "Jinsia",
	"Member #": "Namba ya Mwanachama", "Joined": "Alijiunga", "Method": "Njia", "Reference": "Kumbukumbu",
	"Partner": "Mbia", "Cluster": "Klasta", "Groups": "Vikundi", "Active Groups": "Vikundi Hai",
	"Female": "Wanawake", "Male": "Wanaume", "Social Fund": "Mfuko wa Jamii", "PAR 30 %": "PAR 30 %",
	"Government Loans": "Mikopo ya Serikali", "Attendance %": "Mahudhurio %", "Last Meeting": "Mkutano wa Mwisho",
	"Activity": "Shughuli", "Days Overdue": "Siku za Kuchelewa", "Lender": "Mkopeshaji", "Programme": "Programu",
	"Interest %": "Riba %", "Total Due": "Jumla Inayodaiwa",
}

// valueSw translates the stored enum-like cell values (statuses, transaction
// types, ...) for Swahili exports. Free text (names, titles) is never touched.
var valueSw = map[string]string{
	"Mandatory Savings": "Akiba ya Lazima", "Shares": "Hisa", "Social Fund": "Mfuko wa Jamii",
	"Loan Repayment": "Marejesho ya Mkopo", "Loan Disbursement": "Utoaji wa Mkopo", "Fine": "Faini",
	"Expense": "Matumizi", "Withdrawal": "Uondoaji", "Active": "Hai", "Inactive": "Haifanyi kazi",
	"Closed": "Imefungwa", "Suspended": "Amesimamishwa", "active": "Hai", "repaid": "Imelipwa",
	"defaulted": "Imeshindwa kulipwa", "pending": "Inasubiri", "paid": "Imelipwa", "waived": "Imesamehewa",
	"present": "Amehudhuria", "late": "Amechelewa", "absent": "Hayupo", "excused": "Ameomba udhuru",
	"upcoming": "Inakuja", "in_progress": "Inaendelea", "completed": "Imekamilika", "cancelled": "Imeghairiwa",
	"sent": "Imetumwa", "failed": "Imeshindwa", "in": "Ndani", "out": "Nje", "Male": "Mwanamume", "Female": "Mwanamke",
	"Inactive (no meeting in 30 days)": "Haifanyi kazi (hakuna mkutano siku 30)",
}

var dateColumns = map[string]bool{"Date": true, "Formed": true, "Issued": true, "Due": true, "Recorded": true, "Joined": true, "Last Meeting": true}

func normLang(l string) string {
	if strings.EqualFold(l, "en") {
		return "en"
	}
	return "sw"
}

func colLabel(lang, col string) string {
	if lang == "sw" {
		if v, ok := columnSw[col]; ok {
			return v
		}
	}
	return col
}

func dsLabel(lang string, d *reportDataset) string {
	if lang == "sw" {
		return d.Sw
	}
	return d.En
}

// reportRange parses the from/to query values (YYYY-MM-DD). "to" is made
// inclusive of the whole day, matching the original /api/admin/reports.
func reportRange(from, to string) (time.Time, time.Time) {
	var f, t time.Time
	if from != "" {
		f, _ = time.Parse("2006-01-02", from)
	}
	if to != "" {
		if p, err := time.Parse("2006-01-02", to); err == nil {
			t = p.Add(24 * time.Hour)
		}
	}
	return f, t
}

// buildReport is the single source for both the live preview
// (GET /api/admin/reports) and every export. It returns rows keyed by the
// dataset's registry columns; ok is false for an unknown key.
func buildReport(app *yekonga.YekongaData, key string, groupSet map[string]bool, fromDate, toDate time.Time) ([]datatype.DataMap, bool) {
	inRange := func(t time.Time) bool {
		if !fromDate.IsZero() && t.Before(fromDate) {
			return false
		}
		if !toDate.IsZero() && t.After(toDate) {
			return false
		}
		return true
	}

	groups := app.ModelQuery("Group").SkipBeforeCommit().Find(nil)
	groupById := map[string]datatype.DataMap{}
	groupIds := []string{}
	if groups != nil {
		for _, g := range *groups {
			id := helper.GetValueOfString(g, "id")
			if groupSet != nil && !groupSet[id] {
				continue
			}
			groupById[id] = g
			groupIds = append(groupIds, id)
		}
	}
	inGroupFilter := func(id string) bool {
		_, ok := groupById[id]
		return ok
	}
	groupName := func(id string) string { return helper.GetValueOfString(groupById[id], "name") }

	members := app.ModelQuery("Member").SkipBeforeCommit().Find(nil)
	memberById := map[string]datatype.DataMap{}
	if members != nil {
		for _, m := range *members {
			memberById[helper.GetValueOfString(m, "id")] = m
		}
	}
	memberName := func(id string) string {
		if m, ok := memberById[id]; ok {
			return memberFullName(m)
		}
		return "—"
	}

	rows := []datatype.DataMap{}

	switch key {
	case "summary-partner", "summary-cluster", "summary-group":
		level := strings.TrimPrefix(key, "summary-")
		for _, r := range rollup(app, computeGroupKPIs(app, groupSet), level) {
			rows = append(rows, datatype.DataMap{
				"Name": r["name"], "Groups": r["groups"], "Active Groups": r["activeGroups"], "Members": r["members"],
				"Female": r["femaleMembers"], "Male": r["maleMembers"], "Savings": r["savings"], "Shares": r["shares"],
				"Social Fund": r["socialFund"], "Loans Outstanding": r["loansOutstanding"], "PAR 30 %": r["par30Rate"],
				"Government Loans": r["govLoansOutstanding"], "Attendance %": r["attendanceRate"],
			})
		}

	case "group-performance":
		partnerNames := map[string]string{}
		for _, p := range listAll(app, "Partner") {
			partnerNames[helper.GetValueOfString(p, "id")] = helper.GetValueOfString(p, "name")
		}
		clusterNames := map[string]string{}
		for _, c := range listAll(app, "Cluster") {
			clusterNames[helper.GetValueOfString(c, "id")] = helper.GetValueOfString(c, "name")
		}
		for _, k := range computeGroupKPIs(app, groupSet) {
			g := groupById[k.GroupID]
			activity := "Active"
			if !k.Active {
				activity = "Inactive (no meeting in 30 days)"
			}
			var last interface{}
			if !k.LastMeeting.IsZero() {
				last = k.LastMeeting
			}
			rows = append(rows, datatype.DataMap{
				"Group": k.Name, "Partner": partnerNames[k.PartnerID], "Cluster": clusterNames[k.ClusterID],
				"Region": helper.GetValueOfString(g, "region"), "Members": k.Members, "Savings": k.Savings,
				"Shares": k.Shares, "Loans Outstanding": k.LoansOutstanding, "PAR 30 %": pct(k.PAR, k.LoansOutstanding),
				"Attendance %": pct(float64(k.AttendancePresent), float64(k.AttendanceRows)),
				"Last Meeting": last, "Activity": activity, "Status": k.Status,
			})
		}

	case "portfolio-at-risk":
		now := time.Now()
		for _, l := range listAll(app, "Loan") {
			if !inGroupFilter(helper.GetValueOfString(l, "groupId")) || helper.GetValueOfString(l, "status") != "active" {
				continue
			}
			due := helper.GetValueOfDate(l, "dueDate")
			if due.IsZero() || now.Sub(due) <= parDays*24*time.Hour {
				continue
			}
			rows = append(rows, datatype.DataMap{
				"Loan #": helper.GetValueOfString(l, "loanNumber"), "Group": groupName(helper.GetValueOfString(l, "groupId")),
				"Borrower": memberName(helper.GetValueOfString(l, "memberId")),
				"Balance":  helper.GetValueOfFloat(l, "amount") - helper.GetValueOfFloat(l, "amountRepaid"),
				"Due":      l["dueDate"], "Days Overdue": int(now.Sub(due).Hours() / 24),
			})
		}

	case "government-loans":
		for _, l := range govLoansFor(app, groupSet) {
			if !inRange(helper.GetTimestamp(l["receivedDate"])) {
				continue
			}
			rows = append(rows, datatype.DataMap{
				"Group": l["groupName"], "Lender": helper.GetValueOfString(l, "lender"),
				"Programme": helper.GetValueOfString(l, "programme"), "Reference": helper.GetValueOfString(l, "reference"),
				"Principal": helper.GetValueOfFloat(l, "amount"), "Interest %": helper.GetValueOfFloat(l, "interestRate"),
				"Total Due": l["totalDue"], "Repaid": helper.GetValueOfFloat(l, "amountRepaid"), "Balance": l["outstanding"],
				"Issued": l["receivedDate"], "Due": l["dueDate"], "Status": helper.GetValueOfString(l, "status"),
			})
		}

	case "group-growth":
		for _, id := range groupIds {
			g := groupById[id]
			rows = append(rows, datatype.DataMap{
				"Group": helper.GetValueOfString(g, "name"), "Formed": g["formationDate"],
				"Members":           helper.GetValueOfInt(g, "memberCount"),
				"Cycle":             fmt.Sprintf("%d/%d", helper.GetValueOfInt(g, "cycleCurrent"), helper.GetValueOfInt(g, "cycleTotal")),
				"Meeting Frequency": helper.GetValueOfString(g, "meetingFrequency"),
			})
		}

	case "cycles":
		for _, id := range groupIds {
			g := groupById[id]
			rows = append(rows, datatype.DataMap{
				"Group": helper.GetValueOfString(g, "name"), "Cycle Current": helper.GetValueOfInt(g, "cycleCurrent"),
				"Cycle Total": helper.GetValueOfInt(g, "cycleTotal"), "Meeting Frequency": helper.GetValueOfString(g, "meetingFrequency"),
			})
		}

	case "group-activity", "savings", "shares", "social-fund", "fines", "transactions":
		typeFilter := map[string]string{"savings": "contribution", "shares": "share", "social-fund": "social_fund", "fines": "fine"}[key]
		if txs := app.ModelQuery("Transaction").SkipBeforeCommit().Find(nil); txs != nil {
			for _, t := range *txs {
				if helper.GetValueOfBoolean(t, "reversed") || !inGroupFilter(helper.GetValueOfString(t, "groupId")) {
					continue
				}
				if typeFilter != "" && helper.GetValueOfString(t, "type") != typeFilter {
					continue
				}
				if !inRange(helper.GetTimestamp(t["createdAt"])) {
					continue
				}
				rows = append(rows, datatype.DataMap{
					"Date": t["createdAt"], "Group": groupName(helper.GetValueOfString(t, "groupId")),
					"Member": memberName(helper.GetValueOfString(t, "memberId")), "Type": txLabel(helper.GetValueOfString(t, "type")),
					"Amount": helper.GetValueOfFloat(t, "amount"), "Direction": helper.GetValueOfString(t, "direction"),
					"Method": helper.GetValueOfString(t, "method"), "Reference": helper.GetValueOfString(t, "reference"),
				})
			}
		}

	case "loans":
		if loans := app.ModelQuery("Loan").SkipBeforeCommit().Find(nil); loans != nil {
			for _, l := range *loans {
				if !inGroupFilter(helper.GetValueOfString(l, "groupId")) || !inRange(helper.GetTimestamp(l["issuedDate"])) {
					continue
				}
				amt := helper.GetValueOfFloat(l, "amount")
				repaid := helper.GetValueOfFloat(l, "amountRepaid")
				if helper.GetValueOfString(l, "status") == "cancelled" {
					repaid = amt // a cancelled (reversed) loan owes nothing
				}
				rows = append(rows, datatype.DataMap{
					"Loan #": helper.GetValueOfString(l, "loanNumber"), "Group": groupName(helper.GetValueOfString(l, "groupId")),
					"Borrower": memberName(helper.GetValueOfString(l, "memberId")), "Principal": amt, "Repaid": repaid, "Balance": amt - repaid,
					"Status": helper.GetValueOfString(l, "status"), "Issued": l["issuedDate"], "Due": l["dueDate"],
				})
			}
		}

	case "meetings":
		if meetings := app.ModelQuery("Meeting").SkipBeforeCommit().Find(nil); meetings != nil {
			for _, mt := range *meetings {
				if !inGroupFilter(helper.GetValueOfString(mt, "groupId")) {
					continue
				}
				rows = append(rows, datatype.DataMap{
					"Group":     groupName(helper.GetValueOfString(mt, "groupId")),
					"Meeting #": helper.GetValueOfInt(mt, "meetingNumber"), "Title": helper.GetValueOfString(mt, "title"),
					"Date": helper.GetValueOfString(mt, "date"), "Status": helper.GetValueOfString(mt, "status"),
				})
			}
		}

	case "attendance":
		meetingGroup := map[string]string{}
		if meetings := app.ModelQuery("Meeting").SkipBeforeCommit().Find(nil); meetings != nil {
			for _, mt := range *meetings {
				meetingGroup[helper.GetValueOfString(mt, "id")] = helper.GetValueOfString(mt, "groupId")
			}
		}
		if att := app.ModelQuery("MeetingAttendance").SkipBeforeCommit().Find(nil); att != nil {
			for _, a := range *att {
				gid := helper.GetValueOfString(a, "groupId")
				if gid == "" {
					gid = meetingGroup[helper.GetValueOfString(a, "meetingId")]
				}
				if !inGroupFilter(gid) {
					continue
				}
				rows = append(rows, datatype.DataMap{
					"Group": groupName(gid), "Member": memberName(helper.GetValueOfString(a, "memberId")),
					"Status": helper.GetValueOfString(a, "status"), "Recorded": a["createdAt"],
				})
			}
		}

	case "members":
		for _, m := range memberById {
			if !inGroupFilter(helper.GetValueOfString(m, "groupId")) {
				continue
			}
			rows = append(rows, datatype.DataMap{
				"Group": groupName(helper.GetValueOfString(m, "groupId")), "Name": memberFullName(m),
				"Phone": helper.GetValueOfString(m, "phone"), "Gender": helper.GetValueOfString(m, "gender"),
				"Member #": helper.GetValueOfString(m, "memberNumber"), "Status": helper.GetValueOfString(m, "status"),
				"Joined": m["joinedAt"],
			})
		}

	case "sms-usage", "sms-delivery":
		if logs := app.ModelQuery("SmsLog").SkipBeforeCommit().OrderBy("sentAt", "desc").Find(nil); logs != nil {
			for _, l := range *logs {
				gid := helper.GetValueOfString(l, "groupId")
				// Scoped reports only show their groups' messages (OTP logs carry
				// no group and are only in the organisation-wide view).
				if groupSet != nil && !inGroupFilter(gid) {
					continue
				}
				if !inRange(helper.GetTimestamp(l["sentAt"])) {
					continue
				}
				rows = append(rows, datatype.DataMap{
					"Date": l["sentAt"], "Group": groupName(gid),
					"Phone": helper.GetValueOfString(l, "phone"), "Type": helper.GetValueOfString(l, "messageType"),
					"Status": helper.GetValueOfString(l, "status"),
				})
			}
		}

	default:
		return nil, false
	}

	return rows, true
}

// reportScope decides which groups a request may report on (nil = every
// group). Filters ("partnerId", "clusterId", "groupId") narrow it; the result
// is always limited to what the caller's role can see. Someone who only runs
// a group (Group Admin / Officer) is pinned to that group. label names the
// narrowest filter used, for report headers. ok is false when an error
// response was already written.
func reportScope(app *yekonga.YekongaData, req *yekonga.Request, res *yekonga.Response, filters map[string]string) (map[string]bool, string, bool) {
	a := requireActor(app, req, res)
	if a == nil {
		return nil, "", false
	}
	if !a.Can(PermReports) {
		if a.HomeGroupID != "" && a.HomePerms[PermReports] {
			return map[string]bool{a.HomeGroupID: true}, scopeName(app, "group", a.HomeGroupID), true
		}
		deny(res, 403, "your role does not allow reports")
		return nil, "", false
	}

	var set map[string]bool // nil = everything
	if !a.All {
		set = map[string]bool{}
		for _, id := range a.VisibleGroupIDs() {
			set[id] = true
		}
	}
	narrow := func(keep func(g datatype.DataMap) bool) {
		next := map[string]bool{}
		for _, g := range listAll(app, "Group") {
			id := helper.GetValueOfString(g, "id")
			if (set == nil || set[id]) && keep(g) {
				next[id] = true
			}
		}
		set = next
	}
	label := ""
	if pid := filters["partnerId"]; pid != "" {
		clusters := map[string]bool{}
		for _, c := range listAll(app, "Cluster") {
			if helper.GetValueOfString(c, "partnerId") == pid {
				clusters[helper.GetValueOfString(c, "id")] = true
			}
		}
		narrow(func(g datatype.DataMap) bool { return clusters[helper.GetValueOfString(g, "clusterId")] })
		label = scopeName(app, "partner", pid)
	}
	if cid := filters["clusterId"]; cid != "" {
		narrow(func(g datatype.DataMap) bool { return helper.GetValueOfString(g, "clusterId") == cid })
		label = scopeName(app, "cluster", cid)
	}
	if gid := filters["groupId"]; gid != "" {
		narrow(func(g datatype.DataMap) bool { return helper.GetValueOfString(g, "id") == gid })
		label = scopeName(app, "group", gid)
	}
	return set, label, true
}

// cellText renders a row value for CSV/PDF. Date columns are formatted from
// whatever the DB driver returned (time, string or millis); numbers drop a
// meaningless ".00".
func cellText(lang, col string, v interface{}) string {
	if v == nil {
		return ""
	}
	if dateColumns[col] {
		if s, ok := v.(string); ok {
			return s
		}
		if t := helper.GetTimestamp(v); !t.IsZero() {
			return t.Format("2006-01-02")
		}
		return ""
	}
	switch n := v.(type) {
	case float64:
		if n == float64(int64(n)) {
			return strconv.FormatInt(int64(n), 10)
		}
		return strconv.FormatFloat(n, 'f', 2, 64)
	case string:
		if lang == "sw" {
			if t, ok := valueSw[n]; ok {
				return t
			}
		}
		return n
	}
	return fmt.Sprint(v)
}

// cellValue is the typed value for an Excel cell (numbers stay numeric so
// the sheet can be summed/sorted; text is translated like cellText).
func cellValue(lang, col string, v interface{}) interface{} {
	switch v.(type) {
	case float64, int, int32, int64:
		if !dateColumns[col] {
			return v
		}
	}
	return cellText(lang, col, v)
}

// exportSet is one dataset ready to write: its title, the chosen column ids
// and the (already scoped) rows.
type exportSet struct {
	ds   *reportDataset
	cols []string
	rows []datatype.DataMap
}

func exportXLSX(lang string, sets []exportSet) ([]byte, error) {
	f := excelize.NewFile()
	head, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Color: "#FFFFFF"},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"#0F6E56"}, Pattern: 1},
	})
	for i, s := range sets {
		// Sheet names: max 31 chars, none of : \ / ? * [ ]
		name := strings.NewReplacer(":", " ", "\\", " ", "/", " ", "?", " ", "*", " ", "[", " ", "]", " ").Replace(dsLabel(lang, s.ds))
		if len(name) > 31 {
			name = name[:31]
		}
		if i == 0 {
			_ = f.SetSheetName("Sheet1", name)
		} else if _, err := f.NewSheet(name); err != nil {
			return nil, err
		}
		for c, col := range s.cols {
			cell, _ := excelize.CoordinatesToCellName(c+1, 1)
			_ = f.SetCellValue(name, cell, colLabel(lang, col))
			_ = f.SetCellStyle(name, cell, cell, head)
			colName, _ := excelize.ColumnNumberToName(c + 1)
			_ = f.SetColWidth(name, colName, colName, 20)
		}
		for r, row := range s.rows {
			for c, col := range s.cols {
				cell, _ := excelize.CoordinatesToCellName(c+1, r+2)
				_ = f.SetCellValue(name, cell, cellValue(lang, col, row[col]))
			}
		}
	}
	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func csvBytes(lang string, s exportSet) []byte {
	var buf bytes.Buffer
	buf.WriteString("\xEF\xBB\xBF") // BOM so Excel opens UTF-8 correctly
	w := csv.NewWriter(&buf)
	head := make([]string, len(s.cols))
	for i, c := range s.cols {
		head[i] = colLabel(lang, c)
	}
	_ = w.Write(head)
	for _, row := range s.rows {
		rec := make([]string, len(s.cols))
		for i, c := range s.cols {
			rec[i] = cellText(lang, c, row[c])
		}
		_ = w.Write(rec)
	}
	w.Flush()
	return buf.Bytes()
}

func exportCSV(lang string, sets []exportSet) ([]byte, string, error) {
	if len(sets) == 1 {
		return csvBytes(lang, sets[0]), "csv", nil
	}
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for _, s := range sets {
		w, err := zw.Create(s.ds.Key + ".csv")
		if err != nil {
			return nil, "", err
		}
		_, _ = w.Write(csvBytes(lang, s))
	}
	if err := zw.Close(); err != nil {
		return nil, "", err
	}
	return buf.Bytes(), "zip", nil
}

// pdfFit trims s with "..." so it fits inside width w (mm) at the current font.
func pdfFit(p *fpdf.Fpdf, s string, w float64) string {
	w -= 2
	if p.GetStringWidth(s) <= w {
		return s
	}
	r := []rune(s)
	for len(r) > 1 && p.GetStringWidth(string(r)+"...") > w {
		r = r[:len(r)-1]
	}
	return string(r) + "..."
}

func exportPDF(lang, groupLabel, period string, sets []exportSet) ([]byte, error) {
	p := fpdf.New("L", "mm", "A4", "")
	// Core fonts are cp1252, which covers every Swahili/English character we
	// print; the translator maps our UTF-8 strings (e.g. the "—" placeholder).
	tr := p.UnicodeTranslatorFromDescriptor("")
	p.SetMargins(10, 12, 10)
	p.SetAutoPageBreak(true, 14)
	pageLabel := map[string]string{"sw": "Ukurasa", "en": "Page"}[lang]
	p.SetFooterFunc(func() {
		p.SetY(-10)
		p.SetFont("Helvetica", "", 8)
		p.SetTextColor(120, 120, 120)
		p.CellFormat(0, 6, tr(fmt.Sprintf("%s  |  %s %d", brandName, pageLabel, p.PageNo())), "", 0, "C", false, 0, "")
	})
	p.AddPage()

	loc := map[string]map[string]string{
		"sw": {"title": "Ripoti ya " + brandName, "group": "Kikundi", "period": "Kipindi", "gen": "Imetengenezwa", "none": "Hakuna data kwa vigezo hivi.", "all": "Vikundi vyote", "any": "Muda wote"},
		"en": {"title": brandName + " Report", "group": "Group", "period": "Period", "gen": "Generated", "none": "No data for these filters.", "all": "All groups", "any": "All time"},
	}[lang]
	if groupLabel == "" {
		groupLabel = loc["all"]
	}
	if period == "" {
		period = loc["any"]
	}

	p.SetFont("Helvetica", "B", 18)
	p.SetTextColor(15, 110, 86)
	p.CellFormat(0, 10, tr(loc["title"]), "", 1, "L", false, 0, "")
	p.SetFont("Helvetica", "", 10)
	p.SetTextColor(70, 70, 70)
	p.CellFormat(0, 6, tr(fmt.Sprintf("%s: %s    %s: %s    %s: %s", loc["group"], groupLabel, loc["period"], period, loc["gen"], time.Now().Format("2006-01-02 15:04"))), "", 1, "L", false, 0, "")
	p.Ln(4)

	usable := 277.0 // A4 landscape 297 - 2*10 margins
	for _, s := range sets {
		if p.GetY() > 170 {
			p.AddPage()
		}
		p.SetFont("Helvetica", "B", 13)
		p.SetTextColor(15, 110, 86)
		p.CellFormat(0, 8, tr(fmt.Sprintf("%s (%d)", dsLabel(lang, s.ds), len(s.rows))), "", 1, "L", false, 0, "")

		if len(s.rows) == 0 {
			p.SetFont("Helvetica", "I", 10)
			p.SetTextColor(120, 120, 120)
			p.CellFormat(0, 7, tr(loc["none"]), "", 1, "L", false, 0, "")
			p.Ln(3)
			continue
		}

		w := usable / float64(len(s.cols))
		header := func() {
			p.SetFont("Helvetica", "B", 9)
			p.SetFillColor(15, 110, 86)
			p.SetTextColor(255, 255, 255)
			for _, c := range s.cols {
				p.CellFormat(w, 7, tr(pdfFit(p, colLabel(lang, c), w)), "1", 0, "L", true, 0, "")
			}
			p.Ln(-1)
			p.SetFont("Helvetica", "", 9)
			p.SetTextColor(30, 30, 30)
		}
		header()
		for i, row := range s.rows {
			if p.GetY() > 190 {
				p.AddPage()
				header()
			}
			if i%2 == 1 {
				p.SetFillColor(240, 246, 244)
			} else {
				p.SetFillColor(255, 255, 255)
			}
			for _, c := range s.cols {
				align := "L"
				switch row[c].(type) {
				case float64, int, int32, int64:
					if !dateColumns[c] {
						align = "R"
					}
				}
				p.CellFormat(w, 6.5, tr(pdfFit(p, cellText(lang, c, row[c]), w)), "1", 0, align, true, 0, "")
			}
			p.Ln(-1)
		}
		p.Ln(6)
	}

	var buf bytes.Buffer
	if err := p.Output(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func strList(v interface{}) []string {
	out := []string{}
	if arr, ok := v.([]interface{}); ok {
		for _, x := range arr {
			if s, ok := x.(string); ok && s != "" {
				out = append(out, s)
			}
		}
	}
	return out
}

// registerReports wires the report endpoints:
//
//	GET  /api/admin/reports?type=&groupId=&from=&to=   live preview (one dataset)
//	GET  /api/admin/reports/datasets                   catalog for the export picker
//	POST /api/admin/reports/export                     download chosen datasets
//
// Register after adminGroup is defined; it is passed in for group scoping.
func registerReports(app *yekonga.YekongaData) {
	app.Get("/api/admin/reports", func(req *yekonga.Request, res *yekonga.Response) {
		groupSet, _, ok := reportScope(app, req, res, map[string]string{
			"partnerId": req.Query("partnerId"), "clusterId": req.Query("clusterId"), "groupId": req.Query("groupId"),
		})
		if !ok {
			return
		}
		ds := findDataset(req.Query("type"))
		if ds == nil {
			res.Status(400)
			res.Json(map[string]string{"error": "unknown report type"})
			return
		}
		from, to := reportRange(req.Query("from"), req.Query("to"))
		rows, _ := buildReport(app, ds.Key, groupSet, from, to)
		res.Json(datatype.DataMap{"columns": ds.Columns, "rows": rows})
	})

	app.Get("/api/admin/reports/datasets", func(req *yekonga.Request, res *yekonga.Response) {
		if req.Auth() == nil {
			res.Status(401)
			res.Json(map[string]string{"error": "unauthorized"})
			return
		}
		out := []datatype.DataMap{}
		for i := range reportDatasets {
			d := &reportDatasets[i]
			if !d.Downloadable {
				continue
			}
			cols := []datatype.DataMap{}
			for _, c := range d.Columns {
				cols = append(cols, datatype.DataMap{"key": c, "sw": colLabel("sw", c), "en": c})
			}
			cat := categorySw[d.Category]
			if cat == "" {
				cat = d.Category
			}
			out = append(out, datatype.DataMap{
				"key": d.Key, "category": d.Category, "categorySw": cat,
				"sw": d.Sw, "en": d.En, "columns": cols,
			})
		}
		res.Json(out)
	})

	app.Post("/api/admin/reports/export", func(req *yekonga.Request, res *yekonga.Response) {
		body := bodyMap(req)
		groupSet, groupLabel, ok := reportScope(app, req, res, map[string]string{
			"partnerId": helper.GetValueOfString(body, "partnerId"),
			"clusterId": helper.GetValueOfString(body, "clusterId"),
			"groupId":   helper.GetValueOfString(body, "groupId"),
		})
		if !ok {
			return
		}
		lang := normLang(helper.GetValueOfString(body, "lang"))
		format := strings.ToLower(helper.GetValueOfString(body, "format"))
		if format != "xlsx" && format != "pdf" && format != "csv" {
			res.Status(400)
			res.Json(map[string]string{"error": "format must be xlsx, pdf or csv"})
			return
		}
		keys := strList(body["datasets"])
		if len(keys) == 0 {
			res.Status(400)
			res.Json(map[string]string{"error": "choose at least one dataset"})
			return
		}
		colPick := map[string][]string{}
		if m, ok := body["columns"].(map[string]interface{}); ok {
			for k, v := range m {
				colPick[k] = strList(v)
			}
		}
		fromS, toS := helper.GetValueOfString(body, "from"), helper.GetValueOfString(body, "to")
		from, to := reportRange(fromS, toS)

		sets := []exportSet{}
		seen := map[string]bool{}
		for _, k := range keys {
			ds := findDataset(k)
			if ds == nil || !ds.Downloadable || seen[k] {
				res.Status(400)
				res.Json(map[string]string{"error": "unknown or unavailable dataset: " + k})
				return
			}
			seen[k] = true
			cols := ds.Columns
			if picked := colPick[k]; len(picked) > 0 {
				want := map[string]bool{}
				for _, c := range picked {
					want[c] = true
				}
				cols = []string{}
				for _, c := range ds.Columns { // keep registry order, ignore unknown ids
					if want[c] {
						cols = append(cols, c)
					}
				}
				if len(cols) == 0 {
					res.Status(400)
					res.Json(map[string]string{"error": "no valid columns chosen for " + k})
					return
				}
			}
			rows, _ := buildReport(app, k, groupSet, from, to)
			if len(rows) > maxExportRows {
				rows = rows[:maxExportRows]
			}
			sets = append(sets, exportSet{ds: ds, cols: cols, rows: rows})
		}

		period := ""
		if fromS != "" || toS != "" {
			period = strings.TrimSpace(fromS + " → " + toS)
			period = strings.ReplaceAll(period, "→", "-")
		}

		var data []byte
		var err error
		ext, ctype := format, ""
		switch format {
		case "xlsx":
			data, err = exportXLSX(lang, sets)
			ctype = "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
		case "pdf":
			data, err = exportPDF(lang, groupLabel, period, sets)
			ctype = "application/pdf"
		case "csv":
			data, ext, err = exportCSV(lang, sets)
			ctype = map[string]string{"csv": "text/csv; charset=utf-8", "zip": "application/zip"}[ext]
		}
		if err != nil {
			res.Status(500)
			res.Json(map[string]string{"error": "could not build the export"})
			return
		}

		name := fmt.Sprintf("%s-report-%s.%s", strings.ToLower(brandName), time.Now().Format("20060102-1504"), ext)
		res.SetHeader("Content-Type", ctype)
		res.SetHeader("Content-Disposition", `attachment; filename="`+name+`"`)
		res.Byte(data)
	})
}
