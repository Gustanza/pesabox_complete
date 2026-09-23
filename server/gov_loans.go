package main

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/robertkonga/yekonga-server-go/datatype"
	"github.com/robertkonga/yekonga-server-go/helper"
	"github.com/robertkonga/yekonga-server-go/yekonga"
)

// Government / outside loans (TODO.md §7, D6): money lent TO THE GROUP by a
// council programme (e.g. the 10% women/youth/PWD loans), a bank or an NGO.
// Kept apart from members' savings so group balances stay honest; a group that
// then lends it on to members records those as ordinary member loans.

// govLoanTotalDue is principal plus flat interest (interestRate % of principal).
func govLoanTotalDue(l datatype.DataMap) float64 {
	amount := helper.GetValueOfFloat(l, "amount")
	return amount + amount*helper.GetValueOfFloat(l, "interestRate")/100
}

func govLoanJSON(l datatype.DataMap, repayments []datatype.DataMap) datatype.DataMap {
	rec := datatype.DataMap{}
	for k, v := range l {
		rec[k] = v
	}
	due := govLoanTotalDue(l)
	repaid := helper.GetValueOfFloat(l, "amountRepaid")
	rec["totalDue"] = due
	rec["outstanding"] = math.Max(0, due-repaid)
	if dueDate := helper.GetValueOfDate(l, "dueDate"); !dueDate.IsZero() && time.Now().After(dueDate) && repaid < due {
		rec["overdue"] = true
	}
	if repayments != nil {
		rec["repayments"] = repayments
	}
	return rec
}

// govLoansFor lists a set of groups' government loans (nil set = all groups),
// with their repayments attached.
func govLoansFor(app *yekonga.YekongaData, groupIds map[string]bool) []datatype.DataMap {
	byLoan := map[string][]datatype.DataMap{}
	for _, r := range listAll(app, "GovernmentLoanRepayment") {
		lid := helper.GetValueOfString(r, "governmentLoanId")
		byLoan[lid] = append(byLoan[lid], r)
	}
	groupNames := map[string]string{}
	for _, g := range listAll(app, "Group") {
		groupNames[helper.GetValueOfString(g, "id")] = helper.GetValueOfString(g, "name")
	}
	out := []datatype.DataMap{}
	recs := app.ModelQuery("GovernmentLoan").SkipBeforeCommit().OrderBy("receivedDate", "desc").Find(nil)
	for _, l := range *recs {
		gid := helper.GetValueOfString(l, "groupId")
		if groupIds != nil && !groupIds[gid] {
			continue
		}
		rec := govLoanJSON(l, byLoan[helper.GetValueOfString(l, "id")])
		rec["groupName"] = groupNames[gid]
		out = append(out, rec)
	}
	return out
}

func registerGovernmentLoans(app *yekonga.YekongaData,
	requireGroup func(*yekonga.Request, *yekonga.Response) *datatype.DataMap,
	requireGroupPerm func(*yekonga.Request, *yekonga.Response, string) (*Actor, *datatype.DataMap)) {

	app.Get("/api/main/gov-loans", func(req *yekonga.Request, res *yekonga.Response) {
		g := requireGroup(req, res)
		if g == nil {
			return
		}
		res.Json(govLoansFor(app, map[string]bool{helper.GetValueOfString(*g, "id"): true}))
	})

	app.Post("/api/main/gov-loans", func(req *yekonga.Request, res *yekonga.Response) {
		a, g := requireGroupPerm(req, res, PermFinanceWrite)
		if g == nil {
			return
		}
		gid := helper.GetValueOfString(*g, "id")
		body := bodyMap(req)
		lender := strings.TrimSpace(helper.GetValueOfString(body, "lender"))
		amount := helper.GetValueOfFloat(body, "amount")
		if lender == "" || amount <= 0 {
			deny(res, 400, "the lender and a positive amount are required")
			return
		}
		rate := helper.GetValueOfFloat(body, "interestRate")
		if rate < 0 {
			deny(res, 400, "interest rate cannot be negative")
			return
		}
		received := helper.GetTimestamp(body["receivedDate"])
		if helper.IsEmpty(body["receivedDate"]) {
			received = time.Now()
		}
		term := helper.GetValueOfInt(body, "termMonths")
		rec := datatype.DataMap{
			"groupId":      gid,
			"lender":       lender,
			"programme":    strings.TrimSpace(helper.GetValueOfString(body, "programme")),
			"reference":    strings.TrimSpace(helper.GetValueOfString(body, "reference")),
			"amount":       amount,
			"amountRepaid": 0,
			"interestRate": rate,
			"termMonths":   term,
			"receivedDate": received,
			"status":       "active",
			"notes":        strings.TrimSpace(helper.GetValueOfString(body, "notes")),
			"createdBy":    a.UserID,
		}
		if !helper.IsEmpty(body["dueDate"]) {
			rec["dueDate"] = helper.GetTimestamp(body["dueDate"])
		} else if term > 0 {
			rec["dueDate"] = received.AddDate(0, term, 0)
		}
		created := app.ModelQuery("GovernmentLoan").SkipBeforeCommit().Create(rec)
		if err, ok := created.(error); ok || created == nil {
			deny(res, 500, fmt.Sprint("could not record the loan ", err))
			return
		}
		writeAudit(app, a, "create", "GovernmentLoan", helper.GetValueOfString(helper.ToDataMap(created), "id"), gid,
			fmt.Sprintf("Recorded government loan %s from %s", fmtTZS(amount), lender), nil)
		res.Json(govLoanJSON(helper.ToDataMap(created), nil))
	})

	app.Post("/api/main/gov-loans/:id/repayments", func(req *yekonga.Request, res *yekonga.Response) {
		a, g := requireGroupPerm(req, res, PermFinanceWrite)
		if g == nil {
			return
		}
		gid := helper.GetValueOfString(*g, "id")
		loan := app.ModelQuery("GovernmentLoan").SkipBeforeCommit().Where("id", req.Param("id")).Where("groupId", gid).First(nil)
		if loan == nil {
			deny(res, 404, "loan not found")
			return
		}
		body := bodyMap(req)
		amount := helper.GetValueOfFloat(body, "amount")
		if amount <= 0 {
			deny(res, 400, "a positive amount is required")
			return
		}
		due := govLoanTotalDue(*loan)
		repaid := helper.GetValueOfFloat(*loan, "amountRepaid")
		if repaid+amount > due+0.01 {
			deny(res, 400, fmt.Sprintf("repayment exceeds the remaining balance (%s)", fmtTZS(due-repaid)))
			return
		}
		method := helper.GetValueOfString(body, "method")
		if method == "" {
			method = "Cash"
		}
		date := time.Now()
		if !helper.IsEmpty(body["date"]) {
			date = helper.GetTimestamp(body["date"])
		}
		app.ModelQuery("GovernmentLoanRepayment").SkipBeforeCommit().Create(datatype.DataMap{
			"groupId":          gid,
			"governmentLoanId": req.Param("id"),
			"amount":           amount,
			"date":             date,
			"method":           method,
			"reference":        helper.GetValueOfString(body, "reference"),
			"createdBy":        a.UserID,
		})
		repaid += amount
		status := "active"
		if repaid >= due-0.01 {
			status = "repaid"
		}
		app.ModelQuery("GovernmentLoan").SkipBeforeCommit().Where("id", req.Param("id")).Update(datatype.DataMap{
			"amountRepaid": repaid, "status": status,
		}, nil)
		writeAudit(app, a, "create", "GovernmentLoanRepayment", req.Param("id"), gid,
			fmt.Sprintf("Repaid %s on government loan from %s", fmtTZS(amount), helper.GetValueOfString(*loan, "lender")), nil)
		res.Json(map[string]interface{}{"success": true, "status": status, "outstanding": math.Max(0, due-repaid)})
	})

	// Web: government loans across the caller's scope (optionally one group).
	app.Get("/api/admin/gov-loans", func(req *yekonga.Request, res *yekonga.Response) {
		a := requireActor(app, req, res)
		if a == nil {
			return
		}
		if !a.Can(PermReports) && a.HomeGroupID == "" {
			deny(res, 403, "your role does not allow this")
			return
		}
		set := map[string]bool{}
		if gid := req.Query("groupId"); gid != "" {
			if !a.Sees(gid) {
				deny(res, 404, "group not found")
				return
			}
			set[gid] = true
		} else if a.All {
			set = nil
		} else {
			for _, id := range a.VisibleGroupIDs() {
				set[id] = true
			}
		}
		res.Json(govLoansFor(app, set))
	})
}
