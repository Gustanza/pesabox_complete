package main

import (
	"fmt"
	"math"
	"reflect"
	"sort"
	"strings"

	"github.com/robertkonga/yekonga-server-go/datatype"
	"github.com/robertkonga/yekonga-server-go/helper"
	"github.com/robertkonga/yekonga-server-go/yekonga"
)

// ---------------------------------------------------------------------------
// Group rules (the group's constitution): savings, shares, social fund, loan
// terms, fines and which services the group uses. Edited only through
// GET/PUT /api/main/group/rules (the app, home group) and
// /api/admin/groups/:id/rules (the web) by callers with group.settings in
// that group; GraphQL cannot write these fields (guard.go).
//
// Changes apply to NEW records only: a loan keeps the terms it was issued
// with (interestAmount / totalDue are stored on the loan at issue).
// ---------------------------------------------------------------------------

// knownServices are the services a group can switch on or off.
var knownServices = []string{"Shares", "Mandatory Savings", "Voluntary Savings", "Social Fund", "Loans", "Fines", "Membership Fee"}

// defaultServices applies to a group that never stored enabledServices.
var defaultServices = []string{"Shares", "Mandatory Savings", "Social Fund", "Loans", "Fines"}

// ruleDefaults mirror database.json "Groups" defaults; a group record that
// lacks a field uses these.
var ruleDefaults = map[string]float64{
	"shareValue": 5000, "minShares": 1, "maxShares": 5, "socialFundContribution": 2000,
	"mandatorySavingsAmount": 5000, "loanInterestRate": 10, "maxLoanPeriodMonths": 3, "maxLoanMultiplier": 0,
}

// numericRules: field -> (min, max, whole number). max < 0 = no upper bound.
var numericRules = []struct {
	key      string
	min, max float64
	whole    bool
	label    string
}{
	{"mandatorySavingsAmount", 0, -1, false, "Mandatory savings amount"},
	{"shareValue", 0, -1, false, "Share value"},
	{"minShares", 0, 1000, true, "Minimum shares per meeting"},
	{"maxShares", 0, 1000, true, "Maximum shares per meeting"},
	{"socialFundContribution", 0, -1, false, "Social fund contribution"},
	{"loanInterestRate", 0, 100, false, "Loan interest rate"},
	{"maxLoanPeriodMonths", 1, 36, true, "Loan repayment period"},
	{"maxLoanMultiplier", 0, 100, false, "Maximum loan multiplier"},
}

// ruleFields are every Group field that belongs to the constitution — the
// ones GraphQL may not write. The three old fine fields are no longer used
// (fines come from fineReasons) but stay in the schema for installed app
// versions that still read them; they are locked too.
var ruleFields = []string{
	"shareValue", "minShares", "maxShares", "socialFundContribution", "mandatorySavingsAmount", "fineReasons",
	"loanInterestRate", "maxLoanPeriodMonths", "maxLoanMultiplier", "enabledServices",
	"lateMeetingFine", "absenceFine", "lateLoanRepaymentFine",
}

// docMap reads a stored sub-document as a map. The Mongo driver hands nested
// documents back as ordered key/value lists (bson.D) rather than maps, so a
// plain map conversion would lose them (fine reasons used to fall back to
// the defaults this way).
func docMap(v interface{}) datatype.DataMap {
	if v == nil {
		return datatype.DataMap{}
	}
	rv := reflect.ValueOf(v)
	if rv.Kind() == reflect.Ptr && !rv.IsNil() {
		rv = rv.Elem()
	}
	if rv.Kind() == reflect.Map {
		return helper.ToDataMap(rv.Interface())
	}
	out := datatype.DataMap{}
	if rv.Kind() == reflect.Slice {
		for i := 0; i < rv.Len(); i++ {
			e := rv.Index(i)
			if e.Kind() == reflect.Interface {
				e = e.Elem()
			}
			if e.Kind() != reflect.Struct {
				continue
			}
			k, val := e.FieldByName("Key"), e.FieldByName("Value")
			if k.IsValid() && val.IsValid() && k.Kind() == reflect.String {
				out[k.String()] = val.Interface()
			}
		}
	}
	return out
}

// docList reads a stored array (the driver returns its own named array type,
// which helper.GetValueOfList does not accept).
func docList(v interface{}) []interface{} {
	rv := reflect.ValueOf(v)
	if v == nil || (rv.Kind() != reflect.Slice && rv.Kind() != reflect.Array) {
		return nil
	}
	out := make([]interface{}, 0, rv.Len())
	for i := 0; i < rv.Len(); i++ {
		out = append(out, rv.Index(i).Interface())
	}
	return out
}

// ruleFloat is a group's value for a numeric rule, falling back to the
// schema default when the record has none (0 is a real value, not "unset").
func ruleFloat(g datatype.DataMap, key string) float64 {
	if g != nil {
		if v, ok := g[key]; ok && v != nil {
			return helper.GetValueOfFloat(g, key)
		}
	}
	return ruleDefaults[key]
}

// groupServices is the group's enabled services (defaults when never set).
func groupServices(g datatype.DataMap) []string {
	raw, ok := g["enabledServices"]
	if !ok || raw == nil {
		return append([]string{}, defaultServices...)
	}
	out := []string{}
	for _, s := range docList(raw) {
		if str, ok := s.(string); ok && str != "" {
			out = append(out, str)
		}
	}
	return out
}

func serviceEnabled(g datatype.DataMap, svc string) bool {
	for _, s := range groupServices(g) {
		if strings.EqualFold(s, svc) {
			return true
		}
	}
	return false
}

// txService is the service a transaction type belongs to ("" = always allowed:
// repayments of existing loans, fine payments, expenses, withdrawals).
func txService(typ string) []string {
	switch typ {
	case "contribution":
		return []string{"Mandatory Savings", "Voluntary Savings"}
	case "share":
		return []string{"Shares"}
	case "social_fund":
		return []string{"Social Fund"}
	case "loan_disbursement":
		return []string{"Loans"}
	}
	return nil
}

// serviceOff returns a clear error when typ belongs to a service the group
// has switched off, else "".
func serviceOff(g datatype.DataMap, typ string) string {
	svcs := txService(typ)
	if len(svcs) == 0 {
		return ""
	}
	for _, s := range svcs {
		if serviceEnabled(g, s) {
			return ""
		}
	}
	return fmt.Sprintf("%s is switched off in this group's rules", strings.Join(svcs, " / "))
}

// groupRules is the normalised constitution the routes return.
func groupRules(g datatype.DataMap) datatype.DataMap {
	out := datatype.DataMap{}
	for _, r := range numericRules {
		v := ruleFloat(g, r.key)
		if r.whole {
			out[r.key] = int(math.Round(v))
		} else {
			out[r.key] = v
		}
	}
	reasons := []datatype.DataMap{}
	for _, fr := range groupFineReasons(g) {
		reasons = append(reasons, datatype.DataMap{"reason": helper.GetValueOfString(fr, "reason"), "amount": helper.GetValueOfFloat(fr, "amount")})
	}
	out["fineReasons"] = reasons
	out["enabledServices"] = groupServices(g)
	return out
}

// validateRules merges the requested changes over the group's current rules
// and checks the result. It returns only the fields that change.
func validateRules(g datatype.DataMap, body datatype.DataMap) (datatype.DataMap, error) {
	if inner, ok := body["rules"].(map[string]interface{}); ok {
		body = datatype.DataMap(inner)
	}
	known := map[string]bool{"fineReasons": true, "enabledServices": true}
	for _, r := range numericRules {
		known[r.key] = true
	}
	keys := make([]string, 0, len(body))
	for k := range body {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		if !known[k] {
			return nil, fmt.Errorf("unknown rule %q", k)
		}
	}
	if len(body) == 0 {
		return nil, fmt.Errorf("no rules to change")
	}

	current := groupRules(g)
	next := datatype.DataMap{}
	for k, v := range current {
		next[k] = v
	}
	for _, r := range numericRules {
		v, present := body[r.key]
		if !present {
			continue
		}
		n, ok := v.(float64)
		if !ok {
			if i, isInt := v.(int); isInt {
				n, ok = float64(i), true
			}
		}
		if !ok || math.IsNaN(n) || math.IsInf(n, 0) {
			return nil, fmt.Errorf("%s must be a number", r.label)
		}
		if n < r.min {
			return nil, fmt.Errorf("%s cannot be less than %s", r.label, plainNumber(r.min))
		}
		if r.max >= 0 && n > r.max {
			return nil, fmt.Errorf("%s cannot be more than %s", r.label, plainNumber(r.max))
		}
		if r.whole {
			if n != math.Trunc(n) {
				return nil, fmt.Errorf("%s must be a whole number", r.label)
			}
			next[r.key] = int(n)
		} else {
			next[r.key] = n
		}
	}
	if helper.GetValueOfInt(next, "minShares") > helper.GetValueOfInt(next, "maxShares") {
		return nil, fmt.Errorf("minimum shares (%d) cannot be more than maximum shares (%d)",
			helper.GetValueOfInt(next, "minShares"), helper.GetValueOfInt(next, "maxShares"))
	}

	if raw, present := body["fineReasons"]; present {
		list, ok := raw.([]interface{})
		if !ok {
			return nil, fmt.Errorf("fine reasons must be a list")
		}
		if len(list) > 30 {
			return nil, fmt.Errorf("at most 30 fine reasons")
		}
		seen := map[string]bool{}
		reasons := []datatype.DataMap{}
		for i, item := range list {
			m, ok := item.(map[string]interface{})
			if !ok {
				return nil, fmt.Errorf("fine reason %d is not valid", i+1)
			}
			name := strings.TrimSpace(helper.GetValueOfString(datatype.DataMap(m), "reason"))
			if name == "" {
				return nil, fmt.Errorf("fine reason %d needs a name", i+1)
			}
			if seen[strings.ToLower(name)] {
				return nil, fmt.Errorf("fine reason %q is listed twice", name)
			}
			seen[strings.ToLower(name)] = true
			amt, ok := m["amount"].(float64)
			if !ok && m["amount"] != nil {
				return nil, fmt.Errorf("the amount for fine reason %q must be a number", name)
			}
			if amt < 0 {
				return nil, fmt.Errorf("the amount for fine reason %q cannot be negative", name)
			}
			reasons = append(reasons, datatype.DataMap{"reason": name, "amount": amt})
		}
		next["fineReasons"] = reasons
	}

	if raw, present := body["enabledServices"]; present {
		list, ok := raw.([]interface{})
		if !ok {
			return nil, fmt.Errorf("enabled services must be a list")
		}
		want := map[string]bool{}
		for _, item := range list {
			s, _ := item.(string)
			match := ""
			for _, k := range knownServices {
				if strings.EqualFold(k, strings.TrimSpace(s)) {
					match = k
				}
			}
			if match == "" {
				return nil, fmt.Errorf("unknown service %q (allowed: %s)", s, strings.Join(knownServices, ", "))
			}
			want[match] = true
		}
		services := []string{}
		for _, k := range knownServices { // stored in the canonical order
			if want[k] {
				services = append(services, k)
			}
		}
		next["enabledServices"] = services
	}

	changes := datatype.DataMap{}
	for k, v := range next {
		if fmt.Sprint(v) != fmt.Sprint(current[k]) {
			changes[k] = v
		}
	}
	return changes, nil
}

// rulesDiff is the audit record of a change: field -> {from, to}.
func rulesDiff(before datatype.DataMap, changes datatype.DataMap) (datatype.DataMap, string) {
	diff := datatype.DataMap{}
	keys := make([]string, 0, len(changes))
	for k := range changes {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := []string{}
	for _, k := range keys {
		diff[k] = datatype.DataMap{"from": before[k], "to": changes[k]}
		switch k {
		case "fineReasons", "enabledServices":
			parts = append(parts, k)
		default:
			parts = append(parts, fmt.Sprintf("%s %v → %v", k, before[k], changes[k]))
		}
	}
	return diff, "Updated group rules: " + strings.Join(parts, ", ")
}

// ---- loans: flat interest, fixed at issue -----------------------------------

// loanTerms is the flat interest and total due for a new loan.
func loanTerms(amount, rate float64) (interest, totalDue float64) {
	interest = math.Round(amount*rate/100*100) / 100
	return interest, amount + interest
}

// loanTotalDue is what a loan must repay in total. Loans issued before
// interest was charged have no stored totalDue: they owe the principal only.
func loanTotalDue(l datatype.DataMap) float64 {
	if v, ok := l["totalDue"]; ok && v != nil {
		if t := helper.GetValueOfFloat(l, "totalDue"); t > 0 {
			return t
		}
	}
	return helper.GetValueOfFloat(l, "amount")
}

// loanRepayment applies a repayment to a loan: the new repaid amount and
// status, or an error message. A loan owes its stored total due (principal +
// flat interest); a loan from before interest was charged owes the principal.
func loanRepayment(l datatype.DataMap, amount float64) (float64, string, string) {
	if st := helper.GetValueOfString(l, "status"); st == "cancelled" || st == "repaid" {
		return 0, "", "this loan is " + st + " and takes no more repayments"
	}
	totalDue := loanTotalDue(l)
	repaid := helper.GetValueOfFloat(l, "amountRepaid")
	if repaid+amount > totalDue+0.005 {
		return 0, "", fmt.Sprintf("repayment exceeds the remaining balance (%s)", fmtTZS(math.Max(0, totalDue-repaid)))
	}
	repaid += amount
	status := "active"
	if repaid >= totalDue-0.005 {
		status = "repaid"
	}
	return repaid, status, ""
}

// loanInterest is the interest charged on a loan (0 for legacy loans).
func loanInterest(l datatype.DataMap) float64 {
	return math.Max(0, loanTotalDue(l)-helper.GetValueOfFloat(l, "amount"))
}

// memberSavingsShares is a member's non-reversed savings (contributions less
// withdrawals) plus shares — the base of the maxLoanMultiplier limit.
func memberSavingsShares(txs []datatype.DataMap, memberId string) float64 {
	var total float64
	for _, t := range txs {
		if !txCounted(t) || helper.GetValueOfString(t, "memberId") != memberId {
			continue
		}
		switch helper.GetValueOfString(t, "type") {
		case "contribution", "share":
			total += helper.GetValueOfFloat(t, "amount")
		case "withdrawal":
			total -= helper.GetValueOfFloat(t, "amount")
		}
	}
	return math.Max(0, total)
}

// loanLimitError checks a new loan against maxLoanMultiplier ("" = fine).
func loanLimitError(g datatype.DataMap, amount float64, base float64) string {
	mult := ruleFloat(g, "maxLoanMultiplier")
	if mult <= 0 {
		return ""
	}
	limit := base * mult
	if amount > limit+0.005 {
		return fmt.Sprintf("this loan is above the group's limit of %s× the member's savings and shares (%s for this member)",
			plainNumber(mult), fmtTZS(limit))
	}
	return ""
}

// ---- routes -------------------------------------------------------------------

func rulesResponse(g datatype.DataMap, canEdit bool) datatype.DataMap {
	return datatype.DataMap{
		"groupId": helper.GetValueOfString(g, "id"), "groupName": helper.GetValueOfString(g, "name"),
		"rules": groupRules(g), "canEdit": canEdit, "services": knownServices,
		"note": "Changes apply to new records only; existing loans keep the terms they were issued with.",
	}
}

func registerGroupRules(app *yekonga.YekongaData) {
	loadGroup := func(id string) datatype.DataMap {
		if id == "" {
			return nil
		}
		if g := app.ModelQuery("Group").SkipBeforeCommit().Where("id", id).First(nil); g != nil {
			return *g
		}
		return nil
	}

	get := func(res *yekonga.Response, a *Actor, g datatype.DataMap) {
		res.Json(rulesResponse(g, a.CanIn(PermGroupSettings, helper.GetValueOfString(g, "id"))))
	}

	put := func(req *yekonga.Request, res *yekonga.Response, a *Actor, g datatype.DataMap) {
		gid := helper.GetValueOfString(g, "id")
		if !a.CanIn(PermGroupSettings, gid) {
			deny(res, 403, "your role does not allow changing this group's rules")
			return
		}
		changes, err := validateRules(g, bodyMap(req))
		if err != nil {
			deny(res, 400, err.Error())
			return
		}
		if len(changes) == 0 {
			res.Json(rulesResponse(g, true))
			return
		}
		before := groupRules(g)
		updated := app.ModelQuery("Group").SkipBeforeCommit().Where("id", gid).Update(changes, nil)
		if updated == nil {
			deny(res, 500, "could not save the rules")
			return
		}
		if e, ok := updated.(error); ok {
			deny(res, 500, "could not save the rules: "+e.Error())
			return
		}
		diff, summary := rulesDiff(before, changes)
		writeAudit(app, a, "update", "GroupRules", gid, gid, summary, diff)
		get(res, a, loadGroup(gid))
	}

	// The app: the caller's home group.
	home := func(req *yekonga.Request, res *yekonga.Response) (*Actor, datatype.DataMap) {
		a := requireActor(app, req, res)
		if a == nil {
			return nil, nil
		}
		g := loadGroup(a.HomeGroupID)
		if g == nil {
			deny(res, 403, "no group assigned to this account")
			return nil, nil
		}
		return a, g
	}
	app.Get("/api/main/group/rules", func(req *yekonga.Request, res *yekonga.Response) {
		if a, g := home(req, res); g != nil {
			get(res, a, g)
		}
	})
	for _, register := range []func(string, yekonga.Handler){app.Put, app.Post} {
		register("/api/main/group/rules", func(req *yekonga.Request, res *yekonga.Response) {
			if a, g := home(req, res); g != nil {
				put(req, res, a, g)
			}
		})
	}

	// The web: any group the caller can see.
	byId := func(req *yekonga.Request, res *yekonga.Response) (*Actor, datatype.DataMap) {
		a := requireActor(app, req, res)
		if a == nil {
			return nil, nil
		}
		gid := req.Param("id")
		g := loadGroup(gid)
		if g == nil || !a.Sees(gid) {
			deny(res, 404, "group not found")
			return nil, nil
		}
		return a, g
	}
	app.Get("/api/admin/groups/:id/rules", func(req *yekonga.Request, res *yekonga.Response) {
		if a, g := byId(req, res); g != nil {
			get(res, a, g)
		}
	})
	for _, register := range []func(string, yekonga.Handler){app.Put, app.Post} {
		register("/api/admin/groups/:id/rules", func(req *yekonga.Request, res *yekonga.Response) {
			if a, g := byId(req, res); g != nil {
				put(req, res, a, g)
			}
		})
	}
}
