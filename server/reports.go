package main

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/robertkonga/yekonga-server-go/datatype"
	"github.com/robertkonga/yekonga-server-go/helper"
	"github.com/robertkonga/yekonga-server-go/yekonga"
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
	// PointInTime datasets show the current position (balances, overdue
	// loans, cycles): the from/to range does not apply to them.
	PointInTime bool
	// Balances: point-in-time balance columns next to "(period)" columns.
	Balances bool
	// Totals adds a TOTAL row to exports (and a footer in the web preview).
	Totals bool
	// Sms datasets need the SMS permission for each group they show.
	Sms bool
	// Types overrides columnTypes for this dataset only (e.g. a Date that
	// carries a time of day).
	Types map[string]string
}

var (
	summaryCols = []string{"Name", "Status", "Groups", "Active Groups", "Members (active)", "Members (total)", "Female", "Male",
		"Savings Balance", "Shares Balance", "Social Fund Balance", "Loans Outstanding", "PAR 30 %", "Government Loans", "Attendance %",
		"Savings (period)", "Shares (period)", "Social Fund (period)", "Fines (period)", "Loan Repayments (period)",
		"Loans Disbursed (period)", "Expenses (period)", "Meetings Held (period)", "Attendance % (period)"}
	txCols      = []string{"Date", "Group", "Member", "Type", "Amount", "Direction"}
	smsCols     = []string{"Date", "Group", "Member", "Phone", "Message Type", "Status"}
	cycCols     = []string{"Group", "Cycle Current", "Cycle Total", "Meeting Frequency"}
	loanCols    = []string{"Loan #", "Group", "Borrower", "Principal", "Interest %", "Interest", "Total Due", "Repaid", "Balance", "Overpaid", "Status", "Issued", "Due"}
	txDateTime  = map[string]string{"Date": "datetime"}
	summaryDefs = func(key, sw, en string) reportDataset {
		return reportDataset{Key: key, Category: "Programme", Sw: sw, En: en, Columns: summaryCols, Downloadable: true, Balances: true, Totals: true}
	}
)

// reportDatasets is the registry every report endpoint reads. New report
// types agreed with GATA are added here plus one case in buildReport.
var reportDatasets = []reportDataset{
	summaryDefs("summary-partner", "Muhtasari kwa Mbia", "Summary by Partner"),
	summaryDefs("summary-cluster", "Muhtasari kwa Klasta", "Summary by Cluster"),
	summaryDefs("summary-group", "Muhtasari kwa Kikundi", "Summary by Group"),
	{Key: "group-performance", Category: "Group", Sw: "Utendaji wa Kikundi", En: "Group Performance", Downloadable: true, Balances: true, Totals: true,
		Columns: []string{"Group", "Partner", "Cluster", "Region", "Members (active)", "Savings Balance", "Shares Balance", "Loans Outstanding",
			"PAR 30 %", "Attendance %", "Savings (period)", "Shares (period)", "Loan Repayments (period)", "Meetings Held (period)",
			"Attendance % (period)", "Last Meeting", "Activity", "Status"}},
	{Key: "group-growth", Category: "Group", Sw: "Ukuaji wa Kikundi", En: "Group Growth", Downloadable: true,
		Columns: []string{"Group", "Formed", "Members (active)", "Members (total)", "New Members (period)", "Cycle", "Meeting Frequency"}},
	{Key: "savings", Totals: true, Category: "Financial", Sw: "Akiba", En: "Savings", Columns: txCols, Downloadable: true, Types: txDateTime},
	{Key: "shares", Totals: true, Category: "Financial", Sw: "Hisa", En: "Shares", Columns: txCols, Downloadable: true, Types: txDateTime},
	{Key: "social-fund", Totals: true, Category: "Financial", Sw: "Mfuko wa Jamii", En: "Social Fund", Columns: txCols, Downloadable: true, Types: txDateTime},
	{Key: "fines", Totals: true, Category: "Financial", Sw: "Malipo ya Faini", En: "Fine Payments", Columns: txCols, Downloadable: true, Types: txDateTime},
	{Key: "fines-outstanding", Totals: true, Category: "Financial", Sw: "Faini Zilizotozwa na Madeni", En: "Fines Charged & Outstanding", Downloadable: true,
		Columns: []string{"Group", "Member", "Reason", "Date", "Charged", "Paid", "Outstanding", "Status"}},
	{Key: "expenses", Totals: true, Category: "Financial", Sw: "Matumizi na Uondoaji", En: "Expenses & Withdrawals", Downloadable: true, Types: txDateTime,
		Columns: []string{"Date", "Group", "Type", "Description", "Member", "Amount", "Method"}},
	{Key: "transactions", Category: "Financial", Sw: "Miamala Yote", En: "All Transactions", Downloadable: true, Types: txDateTime,
		Columns: []string{"Date", "Group", "Member", "Type", "Amount", "Direction", "Method", "Reference"}},
	{Key: "loans", Totals: true, Category: "Financial", Sw: "Mikopo", En: "Loans", Columns: loanCols, Downloadable: true},
	{Key: "portfolio-at-risk", Category: "Financial", Sw: "Mikopo Hatarini (PAR 30)", En: "Portfolio at Risk (PAR 30)", Downloadable: true, PointInTime: true,
		Columns: []string{"Loan #", "Group", "Borrower", "Balance", "Due", "Days Overdue"}},
	{Key: "government-loans", Totals: true, Category: "Financial", Sw: "Mikopo ya Serikali", En: "Government Loans", Downloadable: true,
		Columns: []string{"Group", "Lender", "Programme", "Reference", "Principal", "Interest %", "Total Due", "Repaid", "Balance", "Issued", "Due", "Status"}},
	{Key: "gov-loan-repayments", Totals: true, Category: "Financial", Sw: "Marejesho ya Mikopo ya Serikali", En: "Government Loan Repayments", Downloadable: true,
		Columns: []string{"Date", "Group", "Lender", "Programme", "Loan Reference", "Amount", "Method", "Reference"}},
	{Key: "meetings", Category: "Operations", Sw: "Mikutano", En: "Meetings", Downloadable: true,
		Columns: []string{"Group", "Meeting #", "Title", "Date", "Status"}},
	{Key: "meeting-collections", Category: "Operations", Sw: "Makusanyo kwa Mkutano", En: "Collections per Meeting", Downloadable: true, Totals: true,
		Columns: []string{"Group", "Meeting #", "Date", "Status", "Present", "Late", "Absent", "Excused", "Savings", "Shares", "Social Fund",
			"Fines", "Loan Repayments", "Loans Disbursed", "Expenses", "Withdrawals", "Total In", "Total Out"}},
	{Key: "attendance", Category: "Operations", Sw: "Mahudhurio", En: "Attendance", Downloadable: true,
		Columns: []string{"Group", "Meeting #", "Meeting Date", "Member", "Status", "Recorded"}},
	{Key: "cycles", Category: "Operations", Sw: "Mizunguko", En: "Cycles", Columns: cycCols, Downloadable: true, PointInTime: true},
	{Key: "members", Category: "Operations", Sw: "Wanachama", En: "Members", Downloadable: true,
		Columns: []string{"Group", "Name", "Phone", "Gender", "Member #", "Status", "Joined"}},
	{Key: "sms-usage", Category: "Communication", Sw: "Matumizi ya SMS", En: "SMS Usage", Columns: smsCols, Downloadable: true, Sms: true, Types: txDateTime},
	{Key: "sms-delivery", Category: "Communication", Sw: "Uwasilishaji wa SMS", En: "SMS Delivery", Downloadable: true, Sms: true, Totals: true,
		Columns: []string{"Group", "Message Type", "Sent", "Failed", "Other", "Total", "Delivery %"}},
}

// datasetAliases keeps retired keys working. "group-activity" was an exact
// copy of the transactions dataset.
var datasetAliases = map[string]string{"group-activity": "transactions"}

func findDataset(key string) *reportDataset {
	if alias, ok := datasetAliases[key]; ok {
		key = alias
	}
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
	"Members (active)": "Wanachama Hai", "Members (total)": "Wanachama Wote",
	"Savings Balance": "Salio la Akiba", "Shares Balance": "Salio la Hisa", "Social Fund Balance": "Salio la Mfuko wa Jamii",
	"Savings (period)": "Akiba (kipindi)", "Shares (period)": "Hisa (kipindi)", "Social Fund (period)": "Mfuko wa Jamii (kipindi)",
	"Fines (period)": "Faini (kipindi)", "Loan Repayments (period)": "Marejesho ya Mikopo (kipindi)",
	"Loans Disbursed (period)": "Mikopo Iliyotolewa (kipindi)", "Expenses (period)": "Matumizi (kipindi)",
	"Meetings Held (period)": "Mikutano Iliyofanyika (kipindi)", "Attendance % (period)": "Mahudhurio % (kipindi)",
	"New Members (period)": "Wanachama Wapya (kipindi)", "Meeting Date": "Tarehe ya Mkutano",
	"Present": "Waliohudhuria", "Late": "Waliochelewa", "Absent": "Wasiohudhuria", "Excused": "Wenye Udhuru",
	"Fines": "Faini", "Loan Repayments": "Marejesho ya Mikopo", "Loans Disbursed": "Mikopo Iliyotolewa",
	"Expenses": "Matumizi", "Withdrawals": "Uondoaji", "Total In": "Jumla Iliyoingia", "Total Out": "Jumla Iliyotoka",
	"Description": "Maelezo", "Loan Reference": "Kumbukumbu ya Mkopo", "Reason": "Sababu", "Charged": "Faini Iliyotozwa",
	"Paid": "Imelipwa", "Outstanding": "Inayodaiwa", "Message Type": "Aina ya Ujumbe", "Sent": "Zilizotumwa",
	"Failed": "Zilizoshindwa", "Other": "Nyingine", "Total": "Jumla", "Delivery %": "Uwasilishaji %",
	"Overpaid": "Imelipwa Zaidi", "Interest": "Riba",
}

// Column types drive formatting everywhere (web preview, CSV, PDF, XLSX):
//
//	money    summed in totals, thousands separators
//	count    summed in totals (people, groups, meetings, messages)
//	int      a whole number that is not summed (Meeting #, Days Overdue)
//	rate     a percentage (0-100), recomputed — never summed — in totals
//	date     a calendar day (EAT)
//	datetime a moment, shown with the EAT time of day
//	enum     a stored code translated through valueSw / valueEn
//	text     free text, never translated
var columnTypes = map[string]string{
	"Savings": "money", "Shares": "money", "Social Fund": "money", "Loans Outstanding": "money", "Government Loans": "money",
	"Amount": "money", "Principal": "money", "Repaid": "money", "Balance": "money", "Total Due": "money",
	"Savings Balance": "money", "Shares Balance": "money", "Social Fund Balance": "money",
	"Savings (period)": "money", "Shares (period)": "money", "Social Fund (period)": "money", "Fines (period)": "money",
	"Loan Repayments (period)": "money", "Loans Disbursed (period)": "money", "Expenses (period)": "money",
	"Fines": "money", "Loan Repayments": "money", "Loans Disbursed": "money", "Expenses": "money", "Withdrawals": "money",
	"Total In": "money", "Total Out": "money", "Charged": "money", "Paid": "money", "Outstanding": "money",
	"Overpaid": "money", "Interest": "money",

	"Groups": "count", "Active Groups": "count", "Members": "count", "Members (active)": "count", "Members (total)": "count",
	"Female": "count", "Male": "count", "Meetings Held (period)": "count", "New Members (period)": "count",
	"Present": "count", "Late": "count", "Absent": "count", "Excused": "count",
	"Sent": "count", "Failed": "count", "Other": "count", "Total": "count",

	"Meeting #": "int", "Cycle Current": "int", "Cycle Total": "int", "Days Overdue": "int",

	"PAR 30 %": "rate", "Attendance %": "rate", "Attendance % (period)": "rate", "Interest %": "rate", "Delivery %": "rate",

	"Date": "date", "Formed": "date", "Issued": "date", "Due": "date", "Joined": "date", "Last Meeting": "date",
	"Meeting Date": "date", "Recorded": "datetime",

	"Type": "enum", "Status": "enum", "Direction": "enum", "Gender": "enum", "Activity": "enum", "Method": "enum",
	"Meeting Frequency": "enum", "Message Type": "enum",
}

// colType is the column's type in this dataset (ds may be nil).
func colType(ds *reportDataset, col string) string {
	if ds != nil {
		if t, ok := ds.Types[col]; ok {
			return t
		}
	}
	if t, ok := columnTypes[col]; ok {
		return t
	}
	return "text"
}

// valueSw translates the stored enum-like cell values (statuses, transaction
// types, ...) for Swahili. Only applied to enum columns — free text (names,
// titles, reasons) is never touched.
var valueSw = map[string]string{
	"Mandatory Savings": "Akiba ya Lazima", "Shares": "Hisa", "Social Fund": "Mfuko wa Jamii",
	"Loan Repayment": "Marejesho ya Mkopo", "Loan Disbursement": "Utoaji wa Mkopo", "Fine": "Faini",
	"Fine Payment": "Malipo ya Faini", "Expense": "Matumizi", "Withdrawal": "Uondoaji", "Active": "Hai",
	"Inactive": "Haifanyi kazi", "Closed": "Imefungwa", "Suspended": "Amesimamishwa", "active": "Hai", "inactive": "Haifanyi kazi",
	"repaid": "Imelipwa", "defaulted": "Imeshindwa kulipwa", "pending": "Inasubiri", "paid": "Imelipwa", "waived": "Imesamehewa",
	"present": "Amehudhuria", "late": "Amechelewa", "absent": "Hayupo", "excused": "Ameomba udhuru",
	"upcoming": "Inakuja", "in_progress": "Inaendelea", "completed": "Imekamilika", "cancelled": "Imeghairiwa",
	"sent": "Imetumwa", "failed": "Imeshindwa", "in": "Ndani", "out": "Nje", "Male": "Mwanamume", "Female": "Mwanamke",
	"Inactive (no meeting in 30 days)": "Haifanyi kazi (hakuna mkutano siku 30)",
	"Cash":                             "Taslimu", "Mobile Money": "Pesa kwa Simu", "Bank Transfer": "Uhamisho wa Benki",
	"Weekly": "Kila Wiki", "Biweekly": "Kila Wiki Mbili", "Monthly": "Kila Mwezi",
	"login_otp": "OTP ya Kuingia", "member_otp": "OTP ya Mwanachama", "member_joined": "Amejiunga",
	"fine": "Faini", "contribution": "Akiba ya Lazima", "share": "Hisa", "social_fund": "Mfuko wa Jamii",
	"loan_disbursement": "Utoaji wa Mkopo", "loan_repayment": "Marejesho ya Mkopo", "fine_payment": "Malipo ya Faini",
	"meeting_reminder": "Kikumbusho cha Mkutano", "loan_due_soon": "Mkopo Unakaribia Kuisha",
	"loan_overdue": "Mkopo Umechelewa", "other": "Nyingine", "expense": "Matumizi", "withdrawal": "Uondoaji",
}

// valueEn gives the stored lower-case codes a readable English form.
var valueEn = map[string]string{
	"active": "Active", "inactive": "Inactive", "repaid": "Repaid", "defaulted": "Defaulted", "pending": "Pending",
	"paid": "Paid", "waived": "Waived", "present": "Present", "late": "Late", "absent": "Absent", "excused": "Excused",
	"upcoming": "Upcoming", "in_progress": "In progress", "completed": "Completed", "cancelled": "Cancelled",
	"sent": "Sent", "failed": "Failed", "in": "In", "out": "Out",
	"login_otp": "Login OTP", "member_otp": "Member OTP", "member_joined": "Member joined", "fine": "Fine",
	"contribution": "Mandatory Savings", "share": "Shares", "social_fund": "Social Fund",
	"loan_disbursement": "Loan Disbursement", "loan_repayment": "Loan Repayment", "fine_payment": "Fine Payment",
	"meeting_reminder": "Meeting reminder", "loan_due_soon": "Loan due soon", "loan_overdue": "Loan overdue",
	"other": "Other", "expense": "Expense", "withdrawal": "Withdrawal",
}

// textSw translates the few fixed values the server itself writes into text
// columns (user-entered text is never translated).
var textSw = map[string]string{notAssigned: "Haijapangwa"}

// enumLabel translates a stored enum value for lang.
func enumLabel(lang, v string) string {
	if lang == "sw" {
		if t, ok := valueSw[v]; ok {
			return t
		}
	} else if t, ok := valueEn[v]; ok {
		return t
	}
	return v
}

// reportTxLabel is the Type shown in reports; a "fine" transaction is a
// payment towards a fine, so it says so.
func reportTxLabel(typ string) string {
	if typ == "fine" {
		return "Fine Payment"
	}
	return txLabel(typ)
}

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

// ---------------------------------------------------------------------------
// Time: every report day is an East Africa Time day (eatZone, sms.go).
// ---------------------------------------------------------------------------

// parseTimeOrZero reads a stored date: a time value, a BSON date, an
// RFC 3339 string, or a bare YYYY-MM-DD (an EAT calendar day). Anything
// missing or unreadable is the zero time — never "now".
func parseTimeOrZero(v interface{}) time.Time {
	switch x := v.(type) {
	case nil:
		return time.Time{}
	case time.Time:
		return x
	case *time.Time:
		if x == nil {
			return time.Time{}
		}
		return *x
	case string:
		s := strings.TrimSpace(x)
		if s == "" {
			return time.Time{}
		}
		for _, layout := range []string{time.RFC3339Nano, time.RFC3339} {
			if t, err := time.Parse(layout, s); err == nil {
				return t
			}
		}
		for _, layout := range []string{"2006-01-02", "2006-01-02T15:04:05", "2006-01-02 15:04:05", "2006-01-02T15:04:05.000", "02/01/2006"} {
			if t, err := time.ParseInLocation(layout, s, eatZone); err == nil {
				return t
			}
		}
		return time.Time{}
	}
	if t := helper.StringToDatetime(v); t != nil {
		return *t
	}
	return time.Time{}
}

// timeInRange: from/to are inclusive bounds, zero = open. A zero t is never
// in a set range (a record with no date cannot be placed in a period).
func timeInRange(t, from, to time.Time) bool {
	if from.IsZero() && to.IsZero() {
		return true
	}
	if t.IsZero() {
		return false
	}
	if !from.IsZero() && t.Before(from) {
		return false
	}
	if !to.IsZero() && t.After(to) {
		return false
	}
	return true
}

// dateVal is a row value for a date column: nil when unknown, else the time in EAT.
func dateVal(t time.Time) interface{} {
	if t.IsZero() {
		return nil
	}
	return t.In(eatZone)
}

// reportRange parses the from/to query values (YYYY-MM-DD, EAT days). "to"
// is inclusive through 23:59:59.999 EAT.
func reportRange(from, to string) (time.Time, time.Time, error) {
	var f, t time.Time
	if from = strings.TrimSpace(from); from != "" {
		p, err := time.ParseInLocation("2006-01-02", from, eatZone)
		if err != nil {
			return f, t, fmt.Errorf("invalid 'from' date %q: use YYYY-MM-DD", from)
		}
		f = p
	}
	if to = strings.TrimSpace(to); to != "" {
		p, err := time.ParseInLocation("2006-01-02", to, eatZone)
		if err != nil {
			return f, t, fmt.Errorf("invalid 'to' date %q: use YYYY-MM-DD", to)
		}
		t = p.AddDate(0, 0, 1).Add(-time.Nanosecond)
	}
	if !f.IsZero() && !t.IsZero() && f.After(t) {
		return f, t, fmt.Errorf("'from' (%s) must be on or before 'to' (%s)", from, to)
	}
	return f, t, nil
}

// ---------------------------------------------------------------------------
// Scope and permissions
// ---------------------------------------------------------------------------

// reportScope decides which groups a request may report on (nil = every
// group). Filters ("partnerId", "clusterId", "groupId") narrow it; the result
// is always limited to what the caller's role can see. Someone who only runs
// a group (Group Admin / Officer) is pinned to that group, and a filter
// pointing anywhere else is refused (403). label names the narrowest filter
// used, for report headers. ok is false when an error response was already
// written.
func reportScope(app *yekonga.YekongaData, req *yekonga.Request, res *yekonga.Response, filters map[string]string) (*Actor, map[string]bool, string, bool) {
	a := requireActor(app, req, res)
	if a == nil {
		return nil, nil, "", false
	}
	if !a.Can(PermReports) {
		if a.HomeGroupID == "" || !a.HomePerms[PermReports] {
			deny(res, 403, "your role does not allow reports")
			return nil, nil, "", false
		}
		home := a.HomeGroupID
		homeCluster, homePartner := "", ""
		if g := app.ModelQuery("Group").SkipBeforeCommit().Where("id", home).First(nil); g != nil {
			homeCluster = helper.GetValueOfString(*g, "clusterId")
		}
		if homeCluster != "" {
			if c := app.ModelQuery("Cluster").SkipBeforeCommit().Where("id", homeCluster).First(nil); c != nil {
				homePartner = helper.GetValueOfString(*c, "partnerId")
			}
		}
		if (filters["groupId"] != "" && filters["groupId"] != home) ||
			(filters["clusterId"] != "" && filters["clusterId"] != homeCluster) ||
			(filters["partnerId"] != "" && filters["partnerId"] != homePartner) {
			deny(res, 403, "you can only report on your own group")
			return nil, nil, "", false
		}
		return a, map[string]bool{home: true}, scopeName(app, "group", home), true
	}

	var set map[string]bool // nil = everything
	if !a.All {
		set = map[string]bool{}
		for _, id := range a.VisibleGroupIDs() {
			set[id] = true
		}
	}
	var groups []datatype.DataMap
	if filters["partnerId"] != "" || filters["clusterId"] != "" || filters["groupId"] != "" {
		groups = listAll(app, "Group")
	}
	narrow := func(keep func(g datatype.DataMap) bool) {
		next := map[string]bool{}
		for _, g := range groups {
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
	return a, set, label, true
}

// hasSmsReports: the SMS datasets are listed only for callers who may read
// SMS logs somewhere (platform sms.view, or sms.view in their home group).
func hasSmsReports(a *Actor) bool {
	return a != nil && (a.Can(PermSms) || (a.HomeGroupID != "" && a.HomePerms[PermSms]))
}

// smsGate checks an SMS dataset request the way /api/main/sms/activity
// decides: the caller needs sms.view, and in a named group they need it in
// that group. It writes the 403 itself.
func smsGate(a *Actor, groupId string, res *yekonga.Response) bool {
	if !hasSmsReports(a) || (groupId != "" && !a.CanIn(PermSms, groupId)) {
		deny(res, 403, "your role does not allow SMS reports")
		return false
	}
	return true
}

// ---------------------------------------------------------------------------
// Routes
// ---------------------------------------------------------------------------

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

// columnMeta is the column catalog entry the web uses for labels + formatting.
func columnMeta(d *reportDataset) []datatype.DataMap {
	cols := []datatype.DataMap{}
	for _, c := range d.Columns {
		cols = append(cols, datatype.DataMap{"key": c, "sw": colLabel("sw", c), "en": c, "type": colType(d, c)})
	}
	return cols
}

var slugRe = regexp.MustCompile(`[^a-z0-9]+`)

// fileSlug is an ASCII-only, lower-case, dash-separated version of s.
func fileSlug(s string) string {
	s = strings.ToLower(s)
	s = strings.NewReplacer("á", "a", "à", "a", "â", "a", "ä", "a", "é", "e", "è", "e", "ê", "e", "ë", "e",
		"í", "i", "ï", "i", "ó", "o", "ô", "o", "ö", "o", "ú", "u", "ü", "u", "ç", "c", "ñ", "n", "&", "and").Replace(s)
	s = strings.Trim(slugRe.ReplaceAllString(s, "-"), "-")
	if len(s) > 40 {
		s = strings.Trim(s[:40], "-")
	}
	return s
}

// exportFilename: helabox-<scope>-<dataset|multi>-<from>_<to>.<ext>
func exportFilename(scopeLabel string, keys []string, fromS, toS string, now time.Time, ext string) string {
	scope := fileSlug(scopeLabel)
	if scope == "" {
		scope = "all"
	}
	what := "multi"
	if len(keys) == 1 {
		what = keys[0]
	}
	if fromS == "" {
		fromS = "start"
	}
	if toS == "" {
		toS = now.In(eatZone).Format("2006-01-02")
	}
	return fmt.Sprintf("%s-%s-%s-%s_%s.%s", fileSlug(brandName), scope, what, fromS, toS, ext)
}

// registerReports wires the report endpoints:
//
//	GET  /api/admin/reports?type=&groupId=&from=&to=   live preview (one dataset)
//	GET  /api/admin/reports/datasets                   catalog for the export picker
//	POST /api/admin/reports/export                     download chosen datasets
func registerReports(app *yekonga.YekongaData) {
	app.Get("/api/admin/reports", func(req *yekonga.Request, res *yekonga.Response) {
		ds := findDataset(req.Query("type"))
		if ds == nil {
			deny(res, 400, "unknown report type")
			return
		}
		from, to, err := reportRange(req.Query("from"), req.Query("to"))
		if err != nil {
			deny(res, 400, err.Error())
			return
		}
		a, groupSet, _, ok := reportScope(app, req, res, map[string]string{
			"partnerId": req.Query("partnerId"), "clusterId": req.Query("clusterId"), "groupId": req.Query("groupId"),
		})
		if !ok {
			return
		}
		if ds.Sms && !smsGate(a, req.Query("groupId"), res) {
			return
		}
		ctx := newReportCtx(app, a, groupSet, from, to)
		rows, _ := buildReport(ctx, ds.Key)
		types := map[string]string{}
		for _, c := range ds.Columns {
			types[c] = colType(ds, c)
		}
		out := datatype.DataMap{
			"key": ds.Key, "columns": ds.Columns, "types": types, "rows": stripHidden(rows),
			"pointInTime": ds.PointInTime, "balances": ds.Balances,
			"asAt":  ctx.now.In(eatZone).Format("2006-01-02"),
			"enums": datatype.DataMap{"sw": valueSw, "en": valueEn},
		}
		if ds.Totals {
			out["totals"] = reportTotals(ds, rows)
		}
		res.Json(out)
	})

	app.Get("/api/admin/reports/datasets", func(req *yekonga.Request, res *yekonga.Response) {
		a := requireActor(app, req, res)
		if a == nil {
			return
		}
		out := []datatype.DataMap{}
		for i := range reportDatasets {
			d := &reportDatasets[i]
			if !d.Downloadable || (d.Sms && !hasSmsReports(a)) {
				continue
			}
			cat := categorySw[d.Category]
			if cat == "" {
				cat = d.Category
			}
			out = append(out, datatype.DataMap{
				"key": d.Key, "category": d.Category, "categorySw": cat,
				"sw": d.Sw, "en": d.En, "columns": columnMeta(d),
				"pointInTime": d.PointInTime, "balances": d.Balances, "totals": d.Totals,
			})
		}
		res.Json(out)
	})

	app.Post("/api/admin/reports/export", func(req *yekonga.Request, res *yekonga.Response) {
		body := bodyMap(req)
		lang := normLang(helper.GetValueOfString(body, "lang"))
		format := strings.ToLower(helper.GetValueOfString(body, "format"))
		if format != "xlsx" && format != "pdf" && format != "csv" {
			deny(res, 400, "format must be xlsx, pdf or csv")
			return
		}
		fromS, toS := strings.TrimSpace(helper.GetValueOfString(body, "from")), strings.TrimSpace(helper.GetValueOfString(body, "to"))
		from, to, err := reportRange(fromS, toS)
		if err != nil {
			deny(res, 400, err.Error())
			return
		}
		// Unknown datasets are an error; a dataset named twice is exported once.
		keys := []string{}
		seen := map[string]bool{}
		for _, k := range strList(body["datasets"]) {
			ds := findDataset(k)
			if ds == nil || !ds.Downloadable {
				deny(res, 400, "unknown or unavailable dataset: "+k)
				return
			}
			if !seen[ds.Key] {
				seen[ds.Key] = true
				keys = append(keys, ds.Key)
			}
		}
		if len(keys) == 0 {
			deny(res, 400, "choose at least one dataset")
			return
		}
		groupFilter := helper.GetValueOfString(body, "groupId")
		a, groupSet, groupLabel, ok := reportScope(app, req, res, map[string]string{
			"partnerId": helper.GetValueOfString(body, "partnerId"),
			"clusterId": helper.GetValueOfString(body, "clusterId"),
			"groupId":   groupFilter,
		})
		if !ok {
			return
		}
		colPick := map[string][]string{}
		if m, ok := body["columns"].(map[string]interface{}); ok {
			for k, v := range m {
				if ds := findDataset(k); ds != nil {
					colPick[ds.Key] = strList(v)
				}
			}
		}

		ctx := newReportCtx(app, a, groupSet, from, to)
		sets := []exportSet{}
		for _, k := range keys {
			ds := findDataset(k)
			if ds.Sms && !smsGate(a, groupFilter, res) {
				return
			}
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
					deny(res, 400, "no valid columns chosen for "+k)
					return
				}
			}
			rows, _ := buildReport(ctx, k)
			set := exportSet{ds: ds, cols: cols, total: len(rows)}
			if ds.Totals {
				set.totals = reportTotals(ds, rows)
			}
			if len(rows) > maxExportRows {
				rows = rows[:maxExportRows]
			}
			set.rows = rows
			sets = append(sets, set)
		}

		meta := exportMeta{
			lang: lang, scope: groupLabel, from: fromS, to: toS,
			asAt: ctx.now.In(eatZone).Format("2006-01-02"), generated: ctx.now,
		}
		var data []byte
		ext, ctype := format, ""
		switch format {
		case "xlsx":
			data, err = exportXLSX(meta, sets)
			ctype = "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
		case "pdf":
			data, err = exportPDF(meta, sets)
			ctype = "application/pdf"
		case "csv":
			data, ext, err = exportCSV(meta, sets)
			ctype = map[string]string{"csv": "text/csv; charset=utf-8", "zip": "application/zip"}[ext]
		}
		if err != nil {
			deny(res, 500, "could not build the export")
			return
		}

		name := exportFilename(groupLabel, keys, fromS, toS, ctx.now, ext)
		res.SetHeader("Content-Type", ctype)
		res.SetHeader("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"; filename*=UTF-8''%s`, name, name))
		res.SetHeader("Access-Control-Expose-Headers", "Content-Disposition")
		res.Byte(data)
	})
}
