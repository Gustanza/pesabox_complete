package main

import (
	"fmt"
	"math"
	"strings"
	"sync"
	"time"

	"github.com/robertkonga/yekonga-server-go/datatype"
	"github.com/robertkonga/yekonga-server-go/helper"
	"github.com/robertkonga/yekonga-server-go/yekonga"
)

// ---------------------------------------------------------------------------
// Money paid OUT to members that is not a loan: social fund payouts (help to
// a member in need) and the end-of-cycle share-out. Both are ordinary
// Transactions, so they are never deleted and are undone with
// POST /api/main/transactions/:id/reverse like every other money record.
//
//	POST /api/main/social-fund/payout  {amount, reason, memberId?, meetingId?, method?}
//	POST /api/main/share-out           {payouts: [{memberId, amount}], meetingId?, method?, closeCycle?}
// ---------------------------------------------------------------------------

// txIn: the transaction types that bring money into the group's cash box.
func txIn(typ string) bool {
	switch typ {
	case "contribution", "share", "social_fund", "loan_repayment", "fine":
		return true
	}
	return false
}

// groupCash is the group's cash box from its non-reversed transactions:
// everything in less everything out (loans, expenses, withdrawals, payouts,
// share-outs), and the social fund balance kept apart inside it.
//
// Government loans (gov_loans.go) are not Transactions. Money the group
// borrowed is not the members' to share, so it is left out — but once the
// group has repaid MORE than it received (the interest), that extra came out
// of the members' cash box and is subtracted:
//
//	cash = Σ in − Σ out − max(0, govRepaid − govReceived)
func groupCash(txs []datatype.DataMap, govReceived, govRepaid float64) (cash, socialFund float64) {
	for _, t := range txs {
		if !txCounted(t) {
			continue
		}
		amt := helper.GetValueOfFloat(t, "amount")
		typ := helper.GetValueOfString(t, "type")
		if txIn(typ) {
			cash += amt
		} else {
			cash -= amt
		}
		switch typ {
		case "social_fund":
			socialFund += amt
		case "social_fund_payout":
			socialFund -= amt
		}
	}
	cash -= math.Max(0, govRepaid-govReceived)
	return cash, socialFund
}

// govFlows sums a group's government loans received and repaid.
func govFlows(loans, repayments []datatype.DataMap) (received, repaid float64) {
	for _, l := range loans {
		received += helper.GetValueOfFloat(l, "amount")
	}
	for _, r := range repayments {
		repaid += helper.GetValueOfFloat(r, "amount")
	}
	return received, repaid
}

// memberPool is what a member has put in and not yet had back: savings
// (contributions − withdrawals − savings returned at earlier share-outs) and
// shares (bought − shares returned), each never negative.
func memberPool(txs []datatype.DataMap, memberId string) (savings, shares float64) {
	for _, t := range txs {
		if !txCounted(t) || helper.GetValueOfString(t, "memberId") != memberId {
			continue
		}
		amt := helper.GetValueOfFloat(t, "amount")
		switch helper.GetValueOfString(t, "type") {
		case "contribution":
			savings += amt
		case "withdrawal":
			savings -= amt
		case "share":
			shares += amt
		case "share_out":
			savings -= helper.GetValueOfFloat(t, "savingsReturned")
			shares -= helper.GetValueOfFloat(t, "sharesReturned")
		}
	}
	return math.Max(0, savings), math.Max(0, shares)
}

// splitShareOut splits a member's share-out payment: first their own savings
// come back, then their shares, and anything above that is their return
// (profit: interest, fines, ... earned by the group).
func splitShareOut(amount, savings, shares float64) (savingsReturned, sharesReturned, profit float64) {
	savingsReturned = math.Min(amount, savings)
	sharesReturned = math.Min(amount-savingsReturned, shares)
	profit = math.Round((amount-savingsReturned-sharesReturned)*100) / 100
	return savingsReturned, sharesReturned, profit
}

// shareOutTolerance: rounding allowed per member on the entitlement check.
const shareOutTolerance = 1.0

type shareOutReq struct {
	memberId string
	amount   float64
}

type shareOutPlan struct {
	memberId             string
	amount, sv, sh, gain float64
}

// planShareOut checks a share-out against the group's money and splits each
// payment. avail is the cash that may be shared (groupCash less the social
// fund). The group's profit is what is left of avail once every member's own
// savings and shares (memberPool) are covered; each member may receive at
// most their own pool plus their share of that profit, pro rata to their
// pool, and the profit paid in total may not exceed the group's profit.
func planShareOut(txs []datatype.DataMap, avail float64, reqs []shareOutReq) ([]shareOutPlan, string) {
	pools := map[string]float64{}
	var totalPool float64
	for _, t := range txs {
		if mid := helper.GetValueOfString(t, "memberId"); mid != "" {
			if _, done := pools[mid]; !done {
				sv, sh := memberPool(txs, mid)
				pools[mid] = sv + sh
				totalPool += sv + sh
			}
		}
	}
	profit := math.Max(0, avail-totalPool)
	var total, gains float64
	plans := []shareOutPlan{}
	for _, r := range reqs {
		sv, sh := memberPool(txs, r.memberId)
		entitled := pools[r.memberId]
		if totalPool > 0 {
			entitled += profit * pools[r.memberId] / totalPool
		}
		if r.amount > entitled+shareOutTolerance {
			return nil, fmt.Sprintf("%s is more than this member can receive (%s: their savings and shares %s plus their share of the group's profit)",
				fmtTZS(r.amount), fmtTZS(entitled), fmtTZS(pools[r.memberId]))
		}
		a, b, c := splitShareOut(r.amount, sv, sh)
		plans = append(plans, shareOutPlan{r.memberId, r.amount, a, b, c})
		total += r.amount
		gains += c
	}
	if total > avail+0.005 {
		return nil, fmt.Sprintf("the share-out (%s) is more than the cash available to share (%s, social fund kept apart)",
			fmtTZS(total), fmtTZS(math.Max(0, avail)))
	}
	if gains > profit+shareOutTolerance*float64(len(reqs)) {
		return nil, fmt.Sprintf("the share-out pays %s of profit but the group has only %s of profit to share",
			fmtTZS(gains), fmtTZS(profit))
	}
	return plans, ""
}

// cycleAfterReversal: when the last remaining payment of a share-out that
// closed cycle `cycle` is reversed, the group goes back one cycle — never
// below `cycle` (its value before the close). ok is false when nothing moves.
func cycleAfterReversal(current, cycle int, othersRemain bool) (int, bool) {
	if othersRemain || current <= cycle {
		return current, false
	}
	return current - 1, true
}

// reopenCycle undoes closeCycle once every payment of the share-out that
// closed the cycle has been reversed (called after txn is marked reversed).
func reopenCycle(app *yekonga.YekongaData, a *Actor, groupId string, txn datatype.DataMap) {
	if !helper.GetValueOfBoolean(txn, "cycleClosed") {
		return
	}
	cycle := helper.GetValueOfInt(txn, "cycle")
	others := false
	if txs := app.ModelQuery("Transaction").SkipBeforeCommit().Where("groupId", groupId).Where("type", "share_out").Find(nil); txs != nil {
		for _, t := range *txs {
			if txCounted(t) && helper.GetValueOfBoolean(t, "cycleClosed") && helper.GetValueOfInt(t, "cycle") == cycle &&
				helper.GetValueOfString(t, "id") != helper.GetValueOfString(txn, "id") {
				others = true
			}
		}
	}
	g := app.ModelQuery("Group").SkipBeforeCommit().Where("id", groupId).First(nil)
	if g == nil {
		return
	}
	current := helper.GetValueOfInt(*g, "cycleCurrent")
	next, ok := cycleAfterReversal(current, cycle, others)
	if !ok {
		return
	}
	app.ModelQuery("Group").SkipBeforeCommit().Where("id", groupId).Update(datatype.DataMap{"cycleCurrent": next}, nil)
	writeAudit(app, a, "update", "Group", groupId, groupId,
		fmt.Sprintf("Cycle %d reopened: its share-out was reversed (cycle %d → %d)", cycle, current, next),
		datatype.DataMap{"cycleCurrent": datatype.DataMap{"from": current, "to": next}})
}

// payoutLocks serialises money paid out of one group (check + write), so two
// requests at once cannot both pass the cash check.
var payoutLocks sync.Map

func lockGroupPayouts(groupId string) func() {
	m, _ := payoutLocks.LoadOrStore(groupId, &sync.Mutex{})
	mu := m.(*sync.Mutex)
	mu.Lock()
	return mu.Unlock
}

// maxShareOutPayouts caps one share-out request.
const maxShareOutPayouts = 1000

func registerPayouts(app *yekonga.YekongaData,
	requireGroupPerm func(*yekonga.Request, *yekonga.Response, string) (*Actor, *datatype.DataMap),
	createTransaction func(string, datatype.DataMap) interface{},
	meetingLocked func(*yekonga.Response, string, string) bool) {

	groupTxs := func(groupId string) []datatype.DataMap {
		if txs := app.ModelQuery("Transaction").SkipBeforeCommit().Where("groupId", groupId).Find(nil); txs != nil {
			return *txs
		}
		return nil
	}
	// groupMoney is the group's cash box (groupCash) and social fund now.
	groupMoney := func(groupId string) (txs []datatype.DataMap, cash, fund float64) {
		txs = groupTxs(groupId)
		var loans, reps []datatype.DataMap
		if l := app.ModelQuery("GovernmentLoan").SkipBeforeCommit().Where("groupId", groupId).Find(nil); l != nil {
			loans = *l
		}
		if r := app.ModelQuery("GovernmentLoanRepayment").SkipBeforeCommit().Where("groupId", groupId).Find(nil); r != nil {
			reps = *r
		}
		received, repaid := govFlows(loans, reps)
		cash, fund = groupCash(txs, received, repaid)
		return txs, cash, fund
	}
	memberIn := func(groupId, memberId string) *datatype.DataMap {
		if memberId == "" {
			return nil
		}
		return app.ModelQuery("Member").SkipBeforeCommit().Where("id", memberId).Where("groupId", groupId).First(nil)
	}
	notify := func(req *yekonga.Request, g datatype.DataMap, m *datatype.DataMap, typ string, amount float64, meetingId string) {
		if m == nil {
			return
		}
		sendSmsWithLog(helper.GetValueOfString(helper.ToDataMap(req.Auth()), "id"), helper.GetValueOfString(g, "id"),
			helper.GetValueOfString(*m, "id"), helper.GetValueOfString(*m, "phone"), typ,
			txSmsText(g, *m, typ, amount, meetingSmsDate(meetingId)))
	}

	// Social fund payout: money from the social fund to a member in need (or
	// a group cause, without memberId). Never more than the fund holds.
	app.Post("/api/main/social-fund/payout", func(req *yekonga.Request, res *yekonga.Response) {
		actor, g := requireGroupPerm(req, res, PermFinanceWrite)
		if g == nil {
			return
		}
		groupId := helper.GetValueOfString(*g, "id")
		body := bodyMap(req)
		amount := helper.GetValueOfFloat(body, "amount")
		reason := strings.TrimSpace(helper.GetValueOfString(body, "reason"))
		if amount <= 0 || reason == "" {
			deny(res, 400, "a positive amount and a reason are required")
			return
		}
		memberId := helper.GetValueOfString(body, "memberId")
		member := memberIn(groupId, memberId)
		if memberId != "" && member == nil {
			deny(res, 404, "member not found in your group")
			return
		}
		meetingId := helper.GetValueOfString(body, "meetingId")
		if meetingLocked(res, groupId, meetingId) {
			return
		}
		unlock := lockGroupPayouts(groupId)
		if _, _, fund := groupMoney(groupId); amount > fund+0.005 {
			unlock()
			deny(res, 400, fmt.Sprintf("the social fund only holds %s", fmtTZS(math.Max(0, fund))))
			return
		}
		created := createTransaction(groupId, datatype.DataMap{
			"memberId": memberId, "meetingId": meetingId, "type": "social_fund_payout", "amount": amount,
			"method": helper.GetValueOfString(body, "method"), "description": reason, "createdBy": actor.UserID,
		})
		unlock()
		if created == nil {
			deny(res, 500, "could not record the payout")
			return
		}
		if err, ok := created.(error); ok {
			deny(res, 500, err.Error())
			return
		}
		to := "the group"
		if member != nil {
			to = memberFullName(*member)
		}
		writeAudit(app, actor, "create", "Transaction", helper.GetValueOfString(helper.ToDataMap(created), "id"), groupId,
			fmt.Sprintf("Social fund payout %s to %s: %s", fmtTZS(amount), to, reason), nil)
		notify(req, *g, member, "social_fund_payout", amount, meetingId)
		res.Json(created)
	})

	// Share-out at the end of a cycle: one payment per member. Each payment
	// is split into the member's own savings and shares coming back and the
	// return on them (profit), stored on the transaction. closeCycle moves the
	// group on to its next cycle (needs group.settings).
	app.Post("/api/main/share-out", func(req *yekonga.Request, res *yekonga.Response) {
		actor, g := requireGroupPerm(req, res, PermFinanceWrite)
		if g == nil {
			return
		}
		groupId := helper.GetValueOfString(*g, "id")
		body := bodyMap(req)
		closeCycle := helper.GetValueOfBoolean(body, "closeCycle")
		if closeCycle && !actor.CanIn(PermGroupSettings, groupId) {
			deny(res, 403, "only the group admin can close the cycle")
			return
		}
		list := docList(body["payouts"])
		if len(list) == 0 {
			deny(res, 400, "payouts must list at least one {memberId, amount}")
			return
		}
		if len(list) > maxShareOutPayouts {
			deny(res, 400, fmt.Sprintf("at most %d payouts in one share-out", maxShareOutPayouts))
			return
		}
		meetingId := helper.GetValueOfString(body, "meetingId")
		if meetingLocked(res, groupId, meetingId) {
			return
		}
		seen := map[string]bool{}
		reqs := []shareOutReq{}
		members := map[string]datatype.DataMap{}
		for i, raw := range list {
			p := docMap(raw)
			mid := helper.GetValueOfString(p, "memberId")
			amt := helper.GetValueOfFloat(p, "amount")
			if mid == "" || amt <= 0 {
				deny(res, 400, fmt.Sprintf("payout %d needs a memberId and a positive amount", i+1))
				return
			}
			if seen[mid] {
				deny(res, 400, "a member is listed twice in this share-out")
				return
			}
			seen[mid] = true
			m := memberIn(groupId, mid)
			if m == nil {
				deny(res, 404, fmt.Sprintf("payout %d: member not found in your group", i+1))
				return
			}
			members[mid] = *m
			reqs = append(reqs, shareOutReq{mid, amt})
		}

		// Check and write under the group's payout lock.
		unlock := lockGroupPayouts(groupId)
		txs, cash, fund := groupMoney(groupId)
		// The social fund stays in the box: only the rest can be shared out.
		plans, msg := planShareOut(txs, cash-math.Max(0, fund), reqs)
		if msg != "" {
			unlock()
			deny(res, 400, msg)
			return
		}
		cycle := helper.GetValueOfInt(*g, "cycleCurrent")
		if cycle <= 0 {
			cycle = 1
		}
		ids := []string{}
		var total float64
		failed := ""
		for _, p := range plans {
			created := createTransaction(groupId, datatype.DataMap{
				"memberId": p.memberId, "meetingId": meetingId, "type": "share_out", "amount": p.amount,
				"method": helper.GetValueOfString(body, "method"), "createdBy": actor.UserID,
				"savingsReturned": p.sv, "sharesReturned": p.sh, "profit": p.gain, "cycle": cycle, "cycleClosed": closeCycle,
				"description": fmt.Sprintf("Cycle %d share-out", cycle),
			})
			if created == nil {
				failed = "could not record the share-out"
			} else if err, ok := created.(error); ok {
				failed = err.Error()
			}
			if failed != "" {
				break
			}
			ids = append(ids, helper.GetValueOfString(helper.ToDataMap(created), "id"))
			total += p.amount
		}
		if closeCycle && failed == "" {
			app.ModelQuery("Group").SkipBeforeCommit().Where("id", groupId).Update(datatype.DataMap{"cycleCurrent": cycle + 1}, nil)
		}
		unlock()
		// Audited even when a write failed half way, with what was recorded.
		summary := fmt.Sprintf("Cycle %d share-out: %s to %d member(s)", cycle, fmtTZS(total), len(ids))
		if failed != "" {
			summary += fmt.Sprintf(" — stopped after %d of %d payments: %s", len(ids), len(plans), failed)
		}
		writeAudit(app, actor, "create", "ShareOut", groupId, groupId, summary,
			datatype.DataMap{"transactionIds": ids, "closeCycle": closeCycle && failed == "", "at": time.Now()})
		if failed != "" {
			deny(res, 500, fmt.Sprintf("the share-out stopped after %d of %d payments (%s): reverse those or record the rest", len(ids), len(plans), failed))
			return
		}
		// Texts go out once every payment is recorded.
		for _, p := range plans {
			m := members[p.memberId]
			notify(req, *g, &m, "share_out", p.amount, meetingId)
		}
		res.Json(datatype.DataMap{"success": true, "cycle": cycle, "total": total, "transactionIds": ids, "cycleClosed": closeCycle})
	})
}
