package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/robertkonga/yekonga-server-go/datatype"
	"github.com/robertkonga/yekonga-server-go/helper"
	"github.com/robertkonga/yekonga-server-go/yekonga"
)

// memberOtpTTL is how long a member-phone verification code stays valid.
const memberOtpTTL = 5 * time.Minute

// loadDotEnv reads KEY=VALUE pairs from a .env file next to the binary (git-
// ignored — see server/.env) and applies them via os.Setenv, without
// overwriting a variable that's already set in the real environment (so
// `SMTZ_API_KEY=... ./pesabox-server.exe` still wins over the file). Missing
// file, blank lines, and lines starting with # are all fine and skipped.
func loadDotEnv(path string) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		value = strings.Trim(value, `"'`)
		if key == "" {
			continue
		}
		if _, alreadySet := os.LookupEnv(key); !alreadySet {
			os.Setenv(key, value)
		}
	}
}

// smtzSenderID must already be "approved" on the SMTZ account (checked via
// GET /sender-ids) — an unapproved one gets every send rejected with 403.
// "PesaBox" was never registered (it's "pending" as "HAFLA2"); "SMTZ" is
// approved on this account, so it's the default until "PesaBox" clears
// approval — then flip this back.
const smtzSenderID = "SMTZ"

// normalizeTanzanianPhone converts a locally-typed number (e.g. the
// "0766555111" an admin types into Add Member) into the plain international
// MSISDN form SMTZ's API requires ("255766555111") — its /campaigns example
// only ever shows numbers in that form, and it 400s on anything else.
func normalizeTanzanianPhone(phone string) string {
	digits := strings.TrimSpace(phone)
	digits = strings.TrimPrefix(digits, "+")
	switch {
	case strings.HasPrefix(digits, "0"):
		return "255" + digits[1:]
	case strings.HasPrefix(digits, "255"):
		return digits
	default:
		return digits
	}
}

// sendOtpSms texts message to phone via SMTZ (161.97.99.40:5173's bulk-SMS
// api) when SMTZ_API_KEY is set — used for both the login OTP (UserVerification,
// via logOtp below) and the Add Member phone-verification OTP
// (MemberVerification, via /api/members/request-otp). SMTZ has no OTP
// concept of its own — it's a plain "send this text to these numbers" API —
// so the code is generated and verified entirely on our side and just sent
// as an ordinary SMS through SMTZ's POST /campaigns.
//
// Falls back to logging to the console whenever no key is configured, or
// logs the failure reason if SMTZ rejects the send (e.g. status/body), so
// "the OTP never arrived" always has an answer in the server log either way.
func sendOtpSms(phone string, message string) {
	apiKey := os.Getenv("SMTZ_API_KEY")
	if helper.IsEmpty(apiKey) {
		fmt.Printf("=== OTP SMS (dev only, no SMTZ_API_KEY) for %s: %s ===\n", phone, message)
		return
	}

	payload, _ := json.Marshal(map[string]interface{}{
		"senderId":   smtzSenderID,
		"content":    message,
		"recipients": []string{normalizeTanzanianPhone(phone)},
	})

	req, err := http.NewRequest(
		"POST",
		"http://161.97.99.40:3010/api/v1/campaigns",
		bytes.NewReader(payload),
	)
	if err != nil {
		fmt.Println("smtz: could not build request:", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Println("smtz: send failed:", err)
		return
	}
	defer res.Body.Close()
	if res.StatusCode >= 300 {
		body, _ := io.ReadAll(res.Body)
		fmt.Printf("smtz: send failed with status %d: %s\n", res.StatusCode, body)
		return
	}
	fmt.Printf("smtz: sent OTP campaign to %s\n", normalizeTanzanianPhone(phone))
}

func main() {
	loadDotEnv("./.env")
	if key := os.Getenv("SMTZ_API_KEY"); key != "" {
		masked := key
		if len(masked) > 10 {
			masked = masked[:10] + "…"
		}
		fmt.Printf("SMTZ_API_KEY loaded (%s) — member OTPs will send real SMS.\n", masked)
	} else {
		fmt.Println("SMTZ_API_KEY not set — member OTPs stay in dev mode (constant code, console log only).")
	}

	yekonga.ServerLoad("./config.json", "./database.json")
	app := yekonga.Server

	// Texts the login OTP via SMTZ (same sendOtpSms used for Add Member's
	// phone verification) when SMTZ_API_KEY is set; otherwise falls back to
	// printing it, same as before. UserVerification's usernameType can be
	// "email" for a non-phone identifier, in which case there's nothing to
	// text — that case still just logs.
	logOtp := func(req *yekonga.RequestContext, ctx *yekonga.QueryContext) (interface{}, error) {
		data := helper.ToDataMap(ctx.Data)
		phone := helper.GetValueOfString(data, "username")
		usernameType := helper.GetValueOfString(data, "usernameType")
		code := helper.GetValueOfString(data, "otpCode")

		if usernameType == "phone" && helper.IsNotEmpty(phone) && helper.IsNotEmpty(code) {
			sendOtpSms(phone, fmt.Sprintf("Your PesaBox login code is %s", code))
			return nil, nil
		}

		fmt.Printf("=== OTP CODE (dev only): %v ===\n", ctx.Data)
		return nil, nil
	}

	app.AfterCreate("UserVerification", nil, nil, logOtp)
	app.AfterUpdate("UserVerification", nil, nil, logOtp)

	app.Post("/api/members/request-otp", func(req *yekonga.Request, res *yekonga.Response) {
		if req.Auth() == nil {
			res.Status(401)
			res.Json(map[string]string{"error": "unauthorized"})
			return
		}

		body := helper.ToDataMap(req.Body())
		phone := helper.GetValueOfString(body, "phone")
		if helper.IsEmpty(phone) {
			res.Status(400)
			res.Json(map[string]string{"error": "phone is required"})
			return
		}

		code := yekonga.DevConstantOTP
		if helper.IsNotEmpty(os.Getenv("SMTZ_API_KEY")) {
			code = helper.GetRandomInt(4)
		}

		app.ModelQuery("MemberVerification").SkipBeforeCommit().Create(datatype.DataMap{
			"phone":     phone,
			"code":      code,
			"verified":  false,
			"expiresAt": time.Now().Add(memberOtpTTL),
		})

		sendOtpSms(phone, fmt.Sprintf("Your PesaBox member verification code is %s", code))

		res.Json(map[string]bool{"success": true})
	})

	app.Post("/api/members/verify-otp", func(req *yekonga.Request, res *yekonga.Response) {
		if req.Auth() == nil {
			res.Status(401)
			res.Json(map[string]string{"error": "unauthorized"})
			return
		}

		body := helper.ToDataMap(req.Body())
		phone := helper.GetValueOfString(body, "phone")
		code := helper.GetValueOfString(body, "code")
		if helper.IsEmpty(phone) || helper.IsEmpty(code) {
			res.Status(400)
			res.Json(map[string]string{"error": "phone and code are required"})
			return
		}

		records := app.ModelQuery("MemberVerification").SkipBeforeCommit().
			Where("phone", phone).Find(nil)

		var latest *datatype.DataMap
		for i, rec := range *records {
			if helper.GetValueOfString(rec, "code") != code {
				continue
			}
			if latest == nil || helper.GetTimestamp(rec["createdAt"]).After(helper.GetTimestamp((*latest)["createdAt"])) {
				latest = &(*records)[i]
			}
		}

		if latest == nil {
			res.Status(400)
			res.Json(map[string]string{"error": "invalid code"})
			return
		}
		if expiresAt, ok := (*latest)["expiresAt"]; ok && helper.IsNotEmpty(expiresAt) {
			if time.Now().After(helper.GetTimestamp(expiresAt)) {
				res.Status(400)
				res.Json(map[string]string{"error": "code expired — request a new one"})
				return
			}
		}

		id := helper.GetValueOfString(*latest, "id")
		app.ModelQuery("MemberVerification").SkipBeforeCommit().
			Where("id", id).Update(datatype.DataMap{"verified": true}, nil)

		res.Json(map[string]bool{"success": true})
	})

	// Creates a Member, but only once /api/members/verify-otp has confirmed
	// the phone — this replaces the auto-generated GraphQL createMember
	// mutation for the app's Add Member flow specifically so the "phone not
	// verified" rejection can actually be checked and reported as a real
	// error (a BeforeCreate trigger can't reject a create here: the
	// framework's Create() only aborts on a literal `false` return and
	// silently discards whatever error a trigger returns, so there'd be no
	// way to tell the app why creation failed).
	app.Post("/api/members", func(req *yekonga.Request, res *yekonga.Response) {
		if req.Auth() == nil {
			res.Status(401)
			res.Json(map[string]string{"error": "unauthorized"})
			return
		}

		body := helper.ToDataMap(req.Body())
		phone := helper.GetValueOfString(body, "phone")
		groupId := helper.GetValueOfString(body, "groupId")
		if helper.IsEmpty(phone) || helper.IsEmpty(groupId) {
			res.Status(400)
			res.Json(map[string]string{"error": "groupId and phone are required"})
			return
		}

		verified := false
		records := app.ModelQuery("MemberVerification").SkipBeforeCommit().
			Where("phone", phone).Find(nil)
		for _, rec := range *records {
			if helper.GetValueOfBoolean(rec, "verified") {
				verified = true
				break
			}
		}
		if !verified {
			res.Status(400)
			res.Json(map[string]string{"error": "phone not verified — request and confirm an OTP first"})
			return
		}

		created := app.ModelQuery("Member").SkipBeforeCommit().Create(datatype.DataMap{
			"groupId":      groupId,
			"firstName":    helper.GetValueOfString(body, "firstName"),
			"lastName":     helper.GetValueOfString(body, "lastName"),
			"phone":        phone,
			"gender":       helper.GetValueOfString(body, "gender"),
			"memberNumber": helper.GetValueOfString(body, "memberNumber"),
		})
		if created == nil {
			res.Status(500)
			res.Json(map[string]string{"error": "could not create member"})
			return
		}
		if err, ok := created.(error); ok {
			res.Status(500)
			res.Json(map[string]string{"error": err.Error()})
			return
		}

		res.Json(created)
	})

	// The built-in "User" model is protected from generic public GraphQL
	// find queries by default (it carries password/token/otp fields), so the
	// admin dashboard's user list/role-management goes through these two
	// dedicated, sanitized routes instead of the auto-generated CRUD API.
	app.Get("/api/admin/users", func(req *yekonga.Request, res *yekonga.Response) {
		if req.Auth() == nil {
			res.Status(401)
			res.Json(map[string]string{"error": "unauthorized"})
			return
		}

		list := app.ModelQuery("User").SkipBeforeCommit().Find(nil)
		result := make([]datatype.DataMap, 0, len(*list))

		for _, u := range *list {
			result = append(result, datatype.DataMap{
				"id":        helper.GetValueOfString(u, "id"),
				"firstName": helper.GetValueOfString(u, "firstName"),
				"lastName":  helper.GetValueOfString(u, "lastName"),
				"username":  helper.GetValueOfString(u, "username"),
				"phone":     helper.GetValueOfString(u, "phone"),
				"email":     helper.GetValueOfString(u, "email"),
				"role":      helper.GetValueOfString(u, "role"),
				"status":    helper.GetValueOfString(u, "status"),
				"isActive":  helper.GetValueOfBoolean(u, "isActive"),
				"createdAt": u["createdAt"],
			})
		}

		res.Json(result)
	})

	app.Post("/api/admin/users/:id", func(req *yekonga.Request, res *yekonga.Response) {
		if req.Auth() == nil {
			res.Status(401)
			res.Json(map[string]string{"error": "unauthorized"})
			return
		}

		id := req.Param("id")
		body := helper.ToDataMap(req.Body())
		changes := datatype.DataMap{}

		if role, ok := body["role"]; ok {
			changes["role"] = role
		}
		if status, ok := body["status"]; ok {
			changes["status"] = status
			changes["isActive"] = status == "active"
		}

		if len(changes) == 0 {
			res.Status(400)
			res.Json(map[string]string{"error": "no changes provided"})
			return
		}

		updated := app.ModelQuery("User").SkipBeforeCommit().Where("id", id).Update(changes, nil)
		if updated == nil {
			res.Status(404)
			res.Json(map[string]string{"error": "user not found"})
			return
		}

		res.Json(map[string]bool{"success": true})
	})

	app.Delete("/api/admin/users/:id", func(req *yekonga.Request, res *yekonga.Response) {
		if req.Auth() == nil {
			res.Status(401)
			res.Json(map[string]string{"error": "unauthorized"})
			return
		}

		id := req.Param("id")
		app.ModelQuery("User").SkipBeforeCommit().Where("id", id).Delete(nil)
		res.Json(map[string]bool{"success": true})
	})

	// ---------------------------------------------------------------------------
	// Group-scoped data API ("/api/main/*") — the mobile app's live data. Every
	// endpoint resolves the caller's group exactly like the app's
	// checkGroupAssignment does: a Group whose createdBy == this user's id, or
	// whose adminPhone matches the last 9 digits of this user's login username
	// (phone). Reads are scoped to that group; writes update the Group's running
	// totals (totalSavings/totalShares/totalSocialFund/totalLoans/...) so the
	// dashboard balances stay live without a separate aggregation step.
	// ---------------------------------------------------------------------------

	// last9Digits keeps only the numeric digits of a phone-ish string, truncated
	// to the last 9 — mirrors AppState._phoneTail.
	last9Digits := func(s string) string {
		digits := strings.Map(func(r rune) rune {
			if r >= '0' && r <= '9' {
				return r
			}
			return -1
		}, s)
		if len(digits) > 9 {
			digits = digits[len(digits)-9:]
		}
		return digits
	}

	// adminGroup returns the Group this session administers, or nil.
	adminGroup := func(req *yekonga.Request) *datatype.DataMap {
		if req.Auth() == nil {
			return nil
		}
		auth := helper.ToDataMap(req.Auth())
		userId := helper.GetValueOfString(auth, "id")
		phoneTail := last9Digits(helper.GetValueOfString(auth, "username"))
		if userId == "" && phoneTail == "" {
			return nil
		}
		all := app.ModelQuery("Group").SkipBeforeCommit().Find(nil)
		if all == nil {
			return nil
		}
		var match *datatype.DataMap
		for i := range *all {
			g := (*all)[i]
			createdBy := helper.GetValueOfString(g, "createdBy")
			adminPhone := last9Digits(helper.GetValueOfString(g, "adminPhone"))
			if (userId != "" && createdBy == userId) || (phoneTail != "" && adminPhone != "" && phoneTail == adminPhone) {
				if createdBy == userId {
					match = &g
					break
				}
				if match == nil {
					match = &g
				}
			}
		}
		return match
	}

	// requireGroup 403s when this session has no group assigned yet.
	requireGroup := func(req *yekonga.Request, res *yekonga.Response) *datatype.DataMap {
		g := adminGroup(req)
		if g == nil {
			res.Status(403)
			res.Json(map[string]string{"error": "no group assigned to this account"})
			return nil
		}
		return g
	}

	fullName := func(m datatype.DataMap) string {
		return strings.TrimSpace(helper.GetValueOfString(m, "firstName") + " " + helper.GetValueOfString(m, "lastName"))
	}

	// adjustGroupTotal adds delta to a Group total column, never going negative.
	adjustGroupTotal := func(groupId, field string, delta float64) {
		if delta == 0 {
			return
		}
		groupRec := app.ModelQuery("Group").SkipBeforeCommit().Where("id", groupId).First(nil)
		if groupRec == nil {
			return
		}
		newVal := helper.GetValueOfFloat(*groupRec, field) + delta
		if newVal < 0 {
			newVal = 0
		}
		app.ModelQuery("Group").SkipBeforeCommit().Where("id", groupId).Update(datatype.DataMap{field: newVal}, nil)
	}

	// createTransaction writes a Transaction and adjusts group totals for the
	// balance-affecting types. Returns the created record (or error).
	createTransaction := func(groupId string, x datatype.DataMap) interface{} {
		typ := helper.GetValueOfString(x, "type")
		amount := helper.GetValueOfFloat(x, "amount")
		direction := "out"
		switch typ {
		case "contribution", "share", "social_fund", "loan_repayment", "fine":
			direction = "in"
		}
		description := helper.GetValueOfString(x, "description")
		if description == "" {
			switch typ {
			case "contribution":
				description = "Mandatory savings contribution"
			case "share":
				description = "Share purchase"
			case "social_fund":
				description = "Social fund contribution"
			case "loan_disbursement":
				description = "Loan disbursement"
			case "loan_repayment":
				description = "Loan repayment"
			case "fine":
				description = "Fine payment"
			case "expense":
				description = "Group expense"
			}
		}
		method := helper.GetValueOfString(x, "method")
		if method == "" {
			method = "Cash"
		}
		created := app.ModelQuery("Transaction").SkipBeforeCommit().Create(datatype.DataMap{
			"groupId":     groupId,
			"meetingId":   helper.GetValueOfString(x, "meetingId"),
			"memberId":    helper.GetValueOfString(x, "memberId"),
			"type":        typ,
			"direction":   direction,
			"amount":      amount,
			"method":      method,
			"reference":   helper.GetValueOfString(x, "reference"),
			"description": description,
		})
		if created != nil {
			if _, isErr := created.(error); !isErr {
				switch typ {
				case "contribution":
					adjustGroupTotal(groupId, "totalSavings", amount)
				case "share":
					adjustGroupTotal(groupId, "totalShares", amount)
				case "social_fund":
					adjustGroupTotal(groupId, "totalSocialFund", amount)
				case "loan_repayment":
					adjustGroupTotal(groupId, "totalLoans", -amount)
				case "loan_disbursement":
					adjustGroupTotal(groupId, "totalLoans", amount)
				case "withdrawal":
					adjustGroupTotal(groupId, "totalSavings", -amount)
				case "expense":
					adjustGroupTotal(groupId, "totalExpenses", amount)
				}
			}
		}
		return created
	}

	// enrichList copies each record and attaches the member's name/phone so the
	// app's list screens don't need a second join.
	enrichList := func(model, groupId, orderField string) []datatype.DataMap {
		recs := app.ModelQuery(model).SkipBeforeCommit().Where("groupId", groupId).OrderBy(orderField, "desc").Find(nil)
		members := app.ModelQuery("Member").SkipBeforeCommit().Where("groupId", groupId).Find(nil)
		byId := map[string]datatype.DataMap{}
		for _, m := range *members {
			byId[helper.GetValueOfString(m, "id")] = m
		}
		result := []datatype.DataMap{}
		for _, r := range *recs {
			rec := datatype.DataMap{}
			for k, v := range r {
				rec[k] = v
			}
			if mm, ok := byId[helper.GetValueOfString(r, "memberId")]; ok {
				nm := fullName(mm)
				rec["memberName"] = nm
				rec["fullName"] = nm
				rec["memberPhone"] = helper.GetValueOfString(mm, "phone")
			}
			result = append(result, rec)
		}
		return result
	}

	// Member list with each member's live financial position, computed from the
	// group's Transactions/Loans/Fines rather than a stored balance column.
	app.Get("/api/main/members/balances", func(req *yekonga.Request, res *yekonga.Response) {
		g := requireGroup(req, res)
		if g == nil {
			return
		}
		groupId := helper.GetValueOfString(*g, "id")

		members := app.ModelQuery("Member").SkipBeforeCommit().Where("groupId", groupId).Find(nil)
		txs := app.ModelQuery("Transaction").SkipBeforeCommit().Where("groupId", groupId).Find(nil)
		loans := app.ModelQuery("Loan").SkipBeforeCommit().Where("groupId", groupId).Find(nil)
		fines := app.ModelQuery("Fine").SkipBeforeCommit().Where("groupId", groupId).Find(nil)

		savings := map[string]float64{}
		shares := map[string]float64{}
		shareCount := map[string]int{}
		socialFund := map[string]float64{}
		loansTaken := map[string]int{}

		for _, t := range *txs {
			if helper.GetValueOfBoolean(t, "reversed") {
				continue
			}
			mid := helper.GetValueOfString(t, "memberId")
			if mid == "" {
				continue
			}
			switch helper.GetValueOfString(t, "type") {
			case "contribution":
				savings[mid] += helper.GetValueOfFloat(t, "amount")
			case "share":
				shares[mid] += helper.GetValueOfFloat(t, "amount")
			}
			// contribution/share share the same "money in" pool; count shares
			// separately so the app can render a share count.
		}
		contribCount := map[string]int{}
		for _, t := range *txs {
			if helper.GetValueOfBoolean(t, "reversed") {
				continue
			}
			if helper.GetValueOfString(t, "type") == "contribution" {
				contribCount[helper.GetValueOfString(t, "memberId")]++
			}
		}
		for _, t := range *txs {
			if helper.GetValueOfBoolean(t, "reversed") {
				continue
			}
			if helper.GetValueOfString(t, "type") == "social_fund" {
				socialFund[helper.GetValueOfString(t, "memberId")] += helper.GetValueOfFloat(t, "amount")
			}
		}

		outstanding := map[string]float64{}
		for _, l := range *loans {
			mid := helper.GetValueOfString(l, "memberId")
			loansTaken[mid]++
			if helper.GetValueOfString(l, "status") == "active" {
				outstanding[mid] += helper.GetValueOfFloat(l, "amount") - helper.GetValueOfFloat(l, "amountRepaid")
			}
		}

		finesCharged := map[string]float64{}
		finesPaid := map[string]float64{}
		for _, f := range *fines {
			mid := helper.GetValueOfString(f, "memberId")
			if helper.GetValueOfString(f, "status") != "waived" {
				finesCharged[mid] += helper.GetValueOfFloat(f, "amount")
			}
			finesPaid[mid] += helper.GetValueOfFloat(f, "amountPaid")
		}

		shareValue := helper.GetValueOfFloat(*g, "shareValue")
		result := []datatype.DataMap{}
		for _, m := range *members {
			rec := datatype.DataMap{}
			for k, v := range m {
				rec[k] = v
			}
			mid := helper.GetValueOfString(m, "id")
			rec["fullName"] = fullName(m)
			rec["savings"] = savings[mid]
			rec["contributionCount"] = contribCount[mid]
			rec["shares"] = shares[mid]
			rec["shareCount"] = shareCount[mid]
			if shareValue > 0 {
				rec["shareCount"] = int(shares[mid]/shareValue + 0.5)
			}
			rec["socialFund"] = socialFund[mid]
			rec["loansTaken"] = loansTaken[mid]
			rec["outstanding"] = outstanding[mid]
			rec["outstandingLoan"] = outstanding[mid]
			rec["finesCharged"] = finesCharged[mid]
			rec["finesPaid"] = finesPaid[mid]
			rec["finesOwed"] = finesCharged[mid] - finesPaid[mid]
			result = append(result, rec)
		}
		res.Json(result)
	})

	app.Get("/api/main/meetings", func(req *yekonga.Request, res *yekonga.Response) {
		g := requireGroup(req, res)
		if g == nil {
			return
		}
		groupId := helper.GetValueOfString(*g, "id")
		recs := app.ModelQuery("Meeting").SkipBeforeCommit().Where("groupId", groupId).OrderBy("meetingNumber", "desc").Find(nil)
		result := []datatype.DataMap{}
		for _, m := range *recs {
			rec := datatype.DataMap{}
			for k, v := range m {
				rec[k] = v
			}
			meetingId := helper.GetValueOfString(m, "id")
			att := app.ModelQuery("MeetingAttendance").SkipBeforeCommit().Where("meetingId", meetingId).Find(nil)
			present, late, absent, excused := 0, 0, 0, 0
			for _, a := range *att {
				switch helper.GetValueOfString(a, "status") {
				case "present":
					present++
				case "late":
					late++
				case "absent":
					absent++
				case "excused":
					excused++
				}
			}
			rec["present"] = present
			rec["late"] = late
			rec["absent"] = absent
			rec["excused"] = excused
			rec["attended"] = present + late
			result = append(result, rec)
		}
		res.Json(result)
	})

	app.Get("/api/main/transactions", func(req *yekonga.Request, res *yekonga.Response) {
		g := requireGroup(req, res)
		if g == nil {
			return
		}
		res.Json(enrichList("Transaction", helper.GetValueOfString(*g, "id"), "createdAt"))
	})

	app.Get("/api/main/loans", func(req *yekonga.Request, res *yekonga.Response) {
		g := requireGroup(req, res)
		if g == nil {
			return
		}
		res.Json(enrichList("Loan", helper.GetValueOfString(*g, "id"), "createdAt"))
	})

	app.Get("/api/main/fines", func(req *yekonga.Request, res *yekonga.Response) {
		g := requireGroup(req, res)
		if g == nil {
			return
		}
		res.Json(enrichList("Fine", helper.GetValueOfString(*g, "id"), "createdAt"))
	})

	app.Get("/api/main/announcements", func(req *yekonga.Request, res *yekonga.Response) {
		g := requireGroup(req, res)
		if g == nil {
			return
		}
		recs := app.ModelQuery("Announcement").SkipBeforeCommit().Where("groupId", helper.GetValueOfString(*g, "id")).OrderBy("createdAt", "desc").Find(nil)
		result := []datatype.DataMap{}
		for _, r := range *recs {
			rec := datatype.DataMap{}
			for k, v := range r {
				rec[k] = v
			}
			rec["author"] = fullName(helper.ToDataMap(app.ModelQuery("User").SkipBeforeCommit().Where("id", helper.GetValueOfString(r, "createdBy")).First(nil)))
			result = append(result, rec)
		}
		res.Json(result)
	})

	// SMS campaign history is not persisted yet — return an empty list rather
	// than fabricate records.
	app.Get("/api/main/sms/activity", func(req *yekonga.Request, res *yekonga.Response) {
		if req.Auth() == nil {
			res.Status(401)
			res.Json(map[string]string{"error": "unauthorized"})
			return
		}
		res.Json([]datatype.DataMap{})
	})

	app.Post("/api/main/meetings", func(req *yekonga.Request, res *yekonga.Response) {
		g := requireGroup(req, res)
		if g == nil {
			return
		}
		groupId := helper.GetValueOfString(*g, "id")
		body := helper.ToDataMap(req.Body())
		title := helper.GetValueOfString(body, "title")
		if title == "" {
			res.Status(400)
			res.Json(map[string]string{"error": "title is required"})
			return
		}
		number := app.ModelQuery("Meeting").SkipBeforeCommit().Where("groupId", groupId).Count(nil) + 1
		created := app.ModelQuery("Meeting").SkipBeforeCommit().Create(datatype.DataMap{
			"groupId":       groupId,
			"meetingNumber": number,
			"title":         title,
			"date":          helper.GetValueOfString(body, "date"),
			"time":          helper.GetValueOfString(body, "time"),
			"location":      helper.GetValueOfString(body, "location"),
			"status":        "upcoming",
			"createdBy":     helper.GetValueOfString(helper.ToDataMap(req.Auth()), "id"),
		})
		if created == nil {
			res.Status(500)
			res.Json(map[string]string{"error": "could not create meeting"})
			return
		}
		if err, ok := created.(error); ok {
			res.Status(500)
			res.Json(map[string]string{"error": err.Error()})
			return
		}
		res.Json(created)
	})

	app.Post("/api/main/meetings/:id/start", func(req *yekonga.Request, res *yekonga.Response) {
		g := requireGroup(req, res)
		if g == nil {
			return
		}
		groupId := helper.GetValueOfString(*g, "id")
		updated := app.ModelQuery("Meeting").SkipBeforeCommit().Where("id", req.Param("id")).Where("groupId", groupId).Update(datatype.DataMap{"status": "in_progress"}, nil)
		if updated == nil {
			res.Status(404)
			res.Json(map[string]string{"error": "meeting not found"})
			return
		}
		res.Json(map[string]bool{"success": true})
	})

	// Attendance rows for a meeting (used by the app's Attendance screen).
	app.Get("/api/main/meetings/:id/attendance", func(req *yekonga.Request, res *yekonga.Response) {
		g := requireGroup(req, res)
		if g == nil {
			return
		}
		rows := app.ModelQuery("MeetingAttendance").SkipBeforeCommit().Where("meetingId", req.Param("id")).Find(nil)
		members := app.ModelQuery("Member").SkipBeforeCommit().Find(nil)
		byId := map[string]datatype.DataMap{}
		for _, m := range *members {
			byId[helper.GetValueOfString(m, "id")] = m
		}
		result := []datatype.DataMap{}
		for _, r := range *rows {
			rec := datatype.DataMap{}
			for k, v := range r {
				rec[k] = v
			}
			if mm, ok := byId[helper.GetValueOfString(r, "memberId")]; ok {
				rec["memberName"] = fullName(mm)
				rec["fullName"] = fullName(mm)
			}
			result = append(result, rec)
		}
		res.Json(result)
	})

	// Accepts either {"memberId":"...","status":"present"} or a "members" list.
	app.Post("/api/main/meetings/:id/attendance", func(req *yekonga.Request, res *yekonga.Response) {
		g := requireGroup(req, res)
		if g == nil {
			return
		}
		groupId := helper.GetValueOfString(*g, "id")
		meetingId := req.Param("id")
		mtg := app.ModelQuery("Meeting").SkipBeforeCommit().Where("id", meetingId).Where("groupId", groupId).First(nil)
		if mtg == nil {
			res.Status(404)
			res.Json(map[string]string{"error": "meeting not found"})
			return
		}
		body := helper.ToDataMap(req.Body())
		rows := []datatype.DataMap{}
		if list := helper.GetValueOfList(body, "members"); len(list) > 0 {
			for _, x := range list {
				rows = append(rows, helper.ToDataMap(x))
			}
		} else if mid := helper.GetValueOfString(body, "memberId"); mid != "" {
			rows = append(rows, body)
		}
		for _, row := range rows {
			memberId := helper.GetValueOfString(row, "memberId")
			status := helper.GetValueOfString(row, "status")
			if memberId == "" || status == "" {
				continue
			}
			existing := app.ModelQuery("MeetingAttendance").SkipBeforeCommit().Where("meetingId", meetingId).Where("memberId", memberId).First(nil)
			if existing != nil {
				app.ModelQuery("MeetingAttendance").SkipBeforeCommit().Where("id", helper.GetValueOfString(*existing, "id")).Update(datatype.DataMap{"status": status}, nil)
			} else {
				app.ModelQuery("MeetingAttendance").SkipBeforeCommit().Create(datatype.DataMap{
					"meetingId": meetingId,
					"memberId":  memberId,
					"status":    status,
				})
			}
		}
		res.Json(map[string]bool{"success": true})
	})

	app.Post("/api/main/transactions", func(req *yekonga.Request, res *yekonga.Response) {
		g := requireGroup(req, res)
		if g == nil {
			return
		}
		groupId := helper.GetValueOfString(*g, "id")
		body := helper.ToDataMap(req.Body())
		typ := helper.GetValueOfString(body, "type")
		amount := helper.GetValueOfFloat(body, "amount")
		if typ == "" || amount <= 0 {
			res.Status(400)
			res.Json(map[string]string{"error": "type and a positive amount are required"})
			return
		}
		created := createTransaction(groupId, body)
		if created == nil {
			res.Status(500)
			res.Json(map[string]string{"error": "could not create transaction"})
			return
		}
		if err, ok := created.(error); ok {
			res.Status(500)
			res.Json(map[string]string{"error": err.Error()})
			return
		}
		res.Json(created)
	})

	app.Post("/api/main/loans", func(req *yekonga.Request, res *yekonga.Response) {
		g := requireGroup(req, res)
		if g == nil {
			return
		}
		groupId := helper.GetValueOfString(*g, "id")
		body := helper.ToDataMap(req.Body())
		memberId := helper.GetValueOfString(body, "memberId")
		amount := helper.GetValueOfFloat(body, "amount")
		if memberId == "" || amount <= 0 {
			res.Status(400)
			res.Json(map[string]string{"error": "memberId and a positive amount are required"})
			return
		}
		rec := app.ModelQuery("Member").SkipBeforeCommit().Where("id", memberId).Where("groupId", groupId).First(nil)
		if rec == nil {
			res.Status(404)
			res.Json(map[string]string{"error": "member not found in your group"})
			return
		}
		rate := helper.GetValueOfFloat(body, "interestRate")
		if rate <= 0 {
			rate = helper.GetValueOfFloat(*g, "loanInterestRate")
		}
		if rate <= 0 {
			rate = 10
		}
		periodMonths := helper.GetValueOfInt(*g, "maxLoanPeriodMonths")
		if periodMonths <= 0 {
			periodMonths = 3
		}
		number := app.ModelQuery("Loan").SkipBeforeCommit().Where("groupId", groupId).Count(nil) + 1
		created := app.ModelQuery("Loan").SkipBeforeCommit().Create(datatype.DataMap{
			"groupId":         groupId,
			"memberId":        memberId,
			"loanNumber":      fmt.Sprintf("LN-%04d", number),
			"amount":          amount,
			"amountRepaid":    0,
			"interestRate":    rate,
			"issuedDate":      time.Now(),
			"dueDate":         time.Now().AddDate(0, periodMonths, 0),
			"status":          "active",
			"issuedMeetingId": helper.GetValueOfString(body, "meetingId"),
		})
		if created == nil {
			res.Status(500)
			res.Json(map[string]string{"error": "could not create loan"})
			return
		}
		if err, ok := created.(error); ok {
			res.Status(500)
			res.Json(map[string]string{"error": err.Error()})
			return
		}
		if createTransaction(groupId, datatype.DataMap{
			"groupId":   groupId,
			"memberId":  memberId,
			"meetingId": helper.GetValueOfString(body, "meetingId"),
			"type":      "loan_disbursement",
			"amount":    amount,
			"method":    helper.GetValueOfString(body, "method"),
			"reference": fmt.Sprintf("LN-%04d", number),
		}) == nil {
			res.Status(500)
			res.Json(map[string]string{"error": "could not record loan disbursement"})
			return
		}
		res.Json(created)
	})

	app.Post("/api/main/loans/:id/repayment", func(req *yekonga.Request, res *yekonga.Response) {
		g := requireGroup(req, res)
		if g == nil {
			return
		}
		groupId := helper.GetValueOfString(*g, "id")
		body := helper.ToDataMap(req.Body())
		amount := helper.GetValueOfFloat(body, "amount")
		if amount <= 0 {
			res.Status(400)
			res.Json(map[string]string{"error": "a positive amount is required"})
			return
		}
		loan := app.ModelQuery("Loan").SkipBeforeCommit().Where("id", req.Param("id")).Where("groupId", groupId).First(nil)
		if loan == nil {
			res.Status(404)
			res.Json(map[string]string{"error": "loan not found"})
			return
		}
		loanAmt := helper.GetValueOfFloat(*loan, "amount")
		repaid := helper.GetValueOfFloat(*loan, "amountRepaid")
		if repaid+amount > loanAmt {
			res.Status(400)
			res.Json(map[string]string{"error": "repayment exceeds the remaining balance"})
			return
		}
		repaid += amount
		status := "active"
		if repaid >= loanAmt {
			status = "repaid"
		}
		updated := app.ModelQuery("Loan").SkipBeforeCommit().Where("id", req.Param("id")).Update(datatype.DataMap{
			"amountRepaid": repaid,
			"status":       status,
		}, nil)
		if updated == nil {
			res.Status(404)
			res.Json(map[string]string{"error": "loan not found"})
			return
		}
		createTransaction(groupId, datatype.DataMap{
			"groupId":   groupId,
			"memberId":  helper.GetValueOfString(*loan, "memberId"),
			"meetingId": helper.GetValueOfString(body, "meetingId"),
			"type":      "loan_repayment",
			"amount":    amount,
			"method":    helper.GetValueOfString(body, "method"),
			"reference": helper.GetValueOfString(*loan, "loanNumber"),
		})
		res.Json(map[string]bool{"success": true})
	})

	app.Post("/api/main/fines", func(req *yekonga.Request, res *yekonga.Response) {
		g := requireGroup(req, res)
		if g == nil {
			return
		}
		groupId := helper.GetValueOfString(*g, "id")
		body := helper.ToDataMap(req.Body())
		memberId := helper.GetValueOfString(body, "memberId")
		amount := helper.GetValueOfFloat(body, "amount")
		if memberId == "" || amount <= 0 {
			res.Status(400)
			res.Json(map[string]string{"error": "memberId and a positive amount are required"})
			return
		}
		rec := app.ModelQuery("Member").SkipBeforeCommit().Where("id", memberId).Where("groupId", groupId).First(nil)
		if rec == nil {
			res.Status(404)
			res.Json(map[string]string{"error": "member not found in your group"})
			return
		}
		created := app.ModelQuery("Fine").SkipBeforeCommit().Create(datatype.DataMap{
			"groupId":   groupId,
			"memberId":  memberId,
			"meetingId": helper.GetValueOfString(body, "meetingId"),
			"reason":    helper.GetValueOfString(body, "reason"),
			"amount":    amount,
			"amountPaid": 0,
			"status":    "pending",
			"issuedAt":  time.Now(),
		})
		if created == nil {
			res.Status(500)
			res.Json(map[string]string{"error": "could not create fine"})
			return
		}
		if err, ok := created.(error); ok {
			res.Status(500)
			res.Json(map[string]string{"error": err.Error()})
			return
		}
		res.Json(created)
	})

	app.Post("/api/main/fines/:id/pay", func(req *yekonga.Request, res *yekonga.Response) {
		g := requireGroup(req, res)
		if g == nil {
			return
		}
		groupId := helper.GetValueOfString(*g, "id")
		body := helper.ToDataMap(req.Body())
		amount := helper.GetValueOfFloat(body, "amount")
		if amount <= 0 {
			res.Status(400)
			res.Json(map[string]string{"error": "a positive amount is required"})
			return
		}
		fine := app.ModelQuery("Fine").SkipBeforeCommit().Where("id", req.Param("id")).Where("groupId", groupId).First(nil)
		if fine == nil {
			res.Status(404)
			res.Json(map[string]string{"error": "fine not found"})
			return
		}
		if helper.GetValueOfString(*fine, "status") == "waived" {
			res.Status(400)
			res.Json(map[string]string{"error": "waived fines cannot be paid"})
			return
		}
		paid := helper.GetValueOfFloat(*fine, "amountPaid") + amount
		if paid > helper.GetValueOfFloat(*fine, "amount") {
			res.Status(400)
			res.Json(map[string]string{"error": "payment exceeds the fine amount"})
			return
		}
		status := "pending"
		if paid >= helper.GetValueOfFloat(*fine, "amount") {
			status = "paid"
		}
		updated := app.ModelQuery("Fine").SkipBeforeCommit().Where("id", req.Param("id")).Update(datatype.DataMap{
			"amountPaid": paid,
			"status":     status,
		}, nil)
		if updated == nil {
			res.Status(404)
			res.Json(map[string]string{"error": "fine not found"})
			return
		}
		createTransaction(groupId, datatype.DataMap{
			"groupId":   groupId,
			"memberId":  helper.GetValueOfString(*fine, "memberId"),
			"meetingId": helper.GetValueOfString(body, "meetingId"),
			"type":      "fine",
			"amount":    amount,
			"method":    helper.GetValueOfString(body, "method"),
			"reference": helper.GetValueOfString(*fine, "id"),
		})
		res.Json(map[string]bool{"success": true})
	})

	app.Post("/api/main/expenses", func(req *yekonga.Request, res *yekonga.Response) {
		g := requireGroup(req, res)
		if g == nil {
			return
		}
		groupId := helper.GetValueOfString(*g, "id")
		body := helper.ToDataMap(req.Body())
		amount := helper.GetValueOfFloat(body, "amount")
		if amount <= 0 {
			res.Status(400)
			res.Json(map[string]string{"error": "a positive amount is required"})
			return
		}
		created := createTransaction(groupId, datatype.DataMap{
			"groupId":     groupId,
			"memberId":    helper.GetValueOfString(body, "memberId"),
			"meetingId":   helper.GetValueOfString(body, "meetingId"),
			"type":        "expense",
			"amount":      amount,
			"method":      helper.GetValueOfString(body, "method"),
			"description": helper.GetValueOfString(body, "description"),
		})
		if created == nil {
			res.Status(500)
			res.Json(map[string]string{"error": "could not record expense"})
			return
		}
		if err, ok := created.(error); ok {
			res.Status(500)
			res.Json(map[string]string{"error": err.Error()})
			return
		}
		res.Json(created)
	})

	app.Post("/api/main/announcements", func(req *yekonga.Request, res *yekonga.Response) {
		g := requireGroup(req, res)
		if g == nil {
			return
		}
		body := helper.ToDataMap(req.Body())
		title := helper.GetValueOfString(body, "title")
		if title == "" {
			res.Status(400)
			res.Json(map[string]string{"error": "title is required"})
			return
		}
		created := app.ModelQuery("Announcement").SkipBeforeCommit().Create(datatype.DataMap{
			"groupId":   helper.GetValueOfString(*g, "id"),
			"title":     title,
			"body":      helper.GetValueOfString(body, "body"),
			"priority":  helper.GetValueOfString(body, "priority"),
			"createdBy": helper.GetValueOfString(helper.ToDataMap(req.Auth()), "id"),
		})
		if created == nil {
			res.Status(500)
			res.Json(map[string]string{"error": "could not create announcement"})
			return
		}
		if err, ok := created.(error); ok {
			res.Status(500)
			res.Json(map[string]string{"error": err.Error()})
			return
		}
		res.Json(created)
	})

	// Edit an existing member (used by the mobile app's Edit Member screen).
	app.Post("/api/main/members/:id", func(req *yekonga.Request, res *yekonga.Response) {
		g := requireGroup(req, res)
		if g == nil {
			return
		}
		groupId := helper.GetValueOfString(*g, "id")
		rec := app.ModelQuery("Member").SkipBeforeCommit().Where("id", req.Param("id")).Where("groupId", groupId).First(nil)
		if rec == nil {
			res.Status(404)
			res.Json(map[string]string{"error": "member not found"})
			return
		}
		body := helper.ToDataMap(req.Body())
		changes := datatype.DataMap{}
		for _, k := range []string{"firstName", "lastName", "phone", "gender", "memberNumber", "status"} {
			if v, ok := body[k]; ok {
				changes[k] = v
			}
		}
		if len(changes) == 0 {
			res.Status(400)
			res.Json(map[string]string{"error": "no changes provided"})
			return
		}
		updated := app.ModelQuery("Member").SkipBeforeCommit().Where("id", req.Param("id")).Update(changes, nil)
		if updated == nil {
			res.Status(404)
			res.Json(map[string]string{"error": "member not found"})
			return
		}
		res.Json(map[string]bool{"success": true})
	})

	app.Start(8090)
}
