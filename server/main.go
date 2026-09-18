package main

import (
	"bufio"
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/robertkonga/yekonga-server-go/config"
	"github.com/robertkonga/yekonga-server-go/datatype"
	"github.com/robertkonga/yekonga-server-go/gateway"
	"github.com/robertkonga/yekonga-server-go/gateway/setting"
	"github.com/robertkonga/yekonga-server-go/helper"
	"github.com/robertkonga/yekonga-server-go/yekonga"
)

// memberOtpTTL is how long a member-phone verification code stays valid.
const memberOtpTTL = 5 * time.Minute

// loadDotEnv reads KEY=VALUE pairs from a .env file next to the binary (git-
// ignored — see server/.env) and applies them via os.Setenv, without
// overwriting a variable that's already set in the real environment (so
// `BEEM_API_KEY=... ./pesabox-server.exe` still wins over the file). Missing
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

// beemSenderID is the Sender ID approved on the Beem Africa account (see
// BEEM_SENDER_ID in server/.env to override without a rebuild).
const beemSenderID = "TUKIIO"

// normalizeTanzanianPhone converts a locally-typed number (e.g. the
// "0766555111" an admin types into Add Member) into the plain international
// MSISDN form Beem's API requires ("255766555111").
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

// beemSend delivers message to phone via Beem Africa's SMS API (see
// gateway/sms/beem.go) when BEEM_API_KEY/BEEM_SECRET_KEY are set, and returns
// (sent, note) so callers can record the outcome. Beem has no OTP concept of
// its own — it's a plain "send this text to this number" API — so any code
// sent here is generated and verified entirely on our side and just sent as
// an ordinary SMS.
//
// Falls back to "sent" + console log whenever no credentials are configured
// (dev mode), or "failed" + the reason if Beem rejects the send, so "the SMS
// never arrived" always has an answer in the server log either way.
//
// Previously this went through SMTZ's bulk-SMS API — switched to Beem after
// SMTZ proved unreliable; SMTZ_API_KEY/smtzSend are gone.
func beemSend(phone string, message string) (bool, string) {
	apiKey := os.Getenv("BEEM_API_KEY")
	secretKey := os.Getenv("BEEM_SECRET_KEY")
	if helper.IsEmpty(apiKey) || helper.IsEmpty(secretKey) {
		fmt.Printf("=== SMS (dev only, no BEEM_API_KEY/BEEM_SECRET_KEY) for %s: %s ===\n", phone, message)
		return true, "dev-mode (no Beem credentials)"
	}

	sender := os.Getenv("BEEM_SENDER_ID")
	if helper.IsEmpty(sender) {
		sender = beemSenderID
	}

	provider := gateway.NewSMSProvider(&config.SMSGatewayConfig{
		Provider:  config.ProviderBeem,
		Sender:    sender,
		APIKey:    apiKey,
		SecretKey: secretKey,
	})

	resp, err := provider.Send(setting.SendParams{
		Phone: normalizeTanzanianPhone(phone),
		Text:  message,
	}, nil)
	if err != nil {
		fmt.Println("beem: send failed:", err)
		return false, err.Error()
	}
	if resp.Status != "SUCCESS" {
		note := fmt.Sprintf("code %d: %s", resp.Code, resp.Message)
		fmt.Println("beem: send failed:", note)
		return false, note
	}
	fmt.Printf("beem: sent to %s (request_id=%s)\n", normalizeTanzanianPhone(phone), resp.RequestID)
	return true, ""
}

// sendOtpSms texts message to phone via beemSend — used for both the login OTP
// (UserVerification, via logOtp below) and the Add Member phone-verification
// OTP (MemberVerification, via /api/members/request-otp). It is a convenience
// wrapper that just drops the send result on the console; the richer
// sendSms helper (which also persists an SmsLog record) is used for the
// member-facing joined/fine/transaction confirmations.
func sendOtpSms(phone string, message string) {
	sent, note := beemSend(phone, message)
	if !sent {
		fmt.Printf("beem: OTP to %s failed: %s\n", phone, note)
	}
}

// ---------------------------------------------------------------------------
// Member-facing SMS helpers. Every important member action (joined, fine,
// transaction) goes through sendSmsWithLog: it delivers via beemSend AND
// persists an SmsLog record so /api/main/sms/activity can show the history.
// These are package-level (not closures inside main) so route handlers
// registered earlier in main() can call them without declaration-order
// constraints.
// ---------------------------------------------------------------------------

// commaInt adds thousands separators to a base-10 integer ("5000" -> "5,000").
func commaInt(n int64) string {
	s := strconv.FormatInt(n, 10)
	neg := strings.HasPrefix(s, "-")
	if neg {
		s = s[1:]
	}
	out := []byte{}
	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			out = append(out, ',')
		}
		out = append(out, byte(c))
	}
	if neg {
		return "-" + string(out)
	}
	return string(out)
}

// fmtTZS renders a money amount as the "TZS 5,000" form used in SMS copy.
func fmtTZS(v float64) string {
	return "TZS " + commaInt(int64(math.Round(v)))
}

// dmy renders a time as the dd/mm/yyyy form used in SMS copy.
func dmy(t time.Time) string { return t.Format("02/01/2006") }

// smsDate turns a stored meeting date (string) into dd/mm/yyyy when it's a
// known ISO layout, otherwise returns it untouched, defaulting to today.
func smsDate(s string) string {
	if s == "" {
		return dmy(time.Now())
	}
	if t, err := time.Parse("2006-01-02", s); err == nil {
		return dmy(t)
	}
	return s
}

// memberFullName joins a Member record's first and last name.
func memberFullName(m datatype.DataMap) string {
	return strings.TrimSpace(helper.GetValueOfString(m, "firstName") + " " + helper.GetValueOfString(m, "lastName"))
}

// meetingSmsDate finds the scheduled date string of a meeting (falling back to
// today) so transaction/fine SMS copy can say "cha tarehe 12/09/2026".
func meetingSmsDate(meetingId string) string {
	if meetingId != "" {
		mtg := yekonga.Server.ModelQuery("Meeting").SkipBeforeCommit().Where("id", meetingId).First(nil)
		if mtg != nil {
			return smsDate(helper.GetValueOfString(*mtg, "date"))
		}
	}
	return smsDate("")
}

// txLabel maps a stored Transaction.type to the human name used in SMS copy
// and the SMS log.
func txLabel(typ string) string {
	switch typ {
	case "contribution":
		return "Mandatory Savings"
	case "share":
		return "Shares"
	case "social_fund":
		return "Social Fund"
	case "loan_repayment":
		return "Loan Repayment"
	case "loan_disbursement":
		return "Loan Disbursement"
	case "fine":
		return "Fine"
	case "expense":
		return "Expense"
	case "withdrawal":
		return "Withdrawal"
	}
	return typ
}

// defaultFineReasons is what a group uses until its admin configures its own
// fine reasons (spec §6) — 'Other' carries a 0 so the admin always has to type
// a custom amount for it.
var defaultFineReasons = []datatype.DataMap{
	{"reason": "Late Attendance", "amount": 1000.0},
	{"reason": "Absent", "amount": 2000.0},
	{"reason": "Missed Contribution", "amount": 1000.0},
	{"reason": "Late Loan Repayment", "amount": 2000.0},
	{"reason": "Other", "amount": 0.0},
}

// groupFineReasons returns the group's configured fine reasons, falling back
// to the platform defaults when the group hasn't set its own yet.
func groupFineReasons(g datatype.DataMap) []datatype.DataMap {
	raw := helper.GetValueOfList(g, "fineReasons")
	if len(raw) == 0 {
		return defaultFineReasons
	}
	out := []datatype.DataMap{}
	for _, r := range raw {
		m := helper.ToDataMap(r)
		if helper.GetValueOfString(m, "reason") != "" {
			out = append(out, m)
		}
	}
	if len(out) == 0 {
		return defaultFineReasons
	}
	return out
}

// fineAmountFor resolves the amount for a chosen fine reason from the group's
// configured rules — an explicit admin-typed amount always wins, 'Other' falls
// through to whatever the admin types because its configured amount is 0.
func fineAmountFor(g datatype.DataMap, reason string, amount float64) float64 {
	if amount > 0 {
		return amount
	}
	for _, fr := range groupFineReasons(g) {
		if helper.GetValueOfString(fr, "reason") == reason {
			return helper.GetValueOfFloat(fr, "amount")
		}
	}
	return amount
}

// sendSmsWithLog delivers a member-facing message AND records it in the SmsLog
// model. The send result decides the log's status ('sent'/'failed'), with dev
// mode (no BEEM_API_KEY/BEEM_SECRET_KEY) counting as sent. A logging failure
// is console-only and never fails the caller's own request.
func sendSmsWithLog(authId, groupId, memberId, phone, messageType, message string) {
	if helper.IsEmpty(phone) {
		return
	}
	sent, note := beemSend(phone, message)
	status := "sent"
	if !sent {
		status = "failed"
	}
	rec := yekonga.Server.ModelQuery("SmsLog").SkipBeforeCommit().Create(datatype.DataMap{
		"groupId":          groupId,
		"memberId":         memberId,
		"phone":            phone,
		"messageType":      messageType,
		"message":          message,
		"status":           status,
		"providerResponse": note,
		"createdBy":        authId,
	})
	if rec == nil {
		fmt.Println("smslog: could not persist log record")
	} else if err, ok := rec.(error); ok {
		fmt.Println("smslog: could not persist log record:", err)
	}
}

// joinedSmsText is the "member added to group" confirmation (spec §8).
func joinedSmsText(g datatype.DataMap, m datatype.DataMap) string {
	gName := helper.GetValueOfString(g, "name")
	if gName == "" {
		gName = "kikundi"
	}
	fn := memberFullName(m)
	if fn == "" {
		fn = "Ndugu"
	}
	return fmt.Sprintf("PESABOX: Habari %s, umethibitishwa kuwa mwanachama wa %s. Karibu kwenye kikundi.", fn, gName)
}

// fineSmsText is the "you were fined" confirmation (spec §13).
func fineSmsText(g datatype.DataMap, m datatype.DataMap, reason string, amount float64, evDate string) string {
	gName := helper.GetValueOfString(g, "name")
	if gName == "" {
		gName = "kikundi"
	}
	fn := memberFullName(m)
	if fn == "" {
		fn = "Ndugu"
	}
	return fmt.Sprintf("PESABOX: Habari %s, umepewa faini ya %s kutokana na %s kwenye kikao cha %s cha tarehe %s.", fn, fmtTZS(amount), reason, gName, evDate)
}

// txSmsText is the "contribution/share/fine-payment/loan confirmed" message
// (spec §16); reminder/template wording depends on the transaction type.
func txSmsText(g datatype.DataMap, m datatype.DataMap, typ string, amount float64, evDate string) string {
	gName := helper.GetValueOfString(g, "name")
	if gName == "" {
		gName = "kikundi"
	}
	fn := memberFullName(m)
	if fn == "" {
		fn = "Ndugu"
	}
	switch typ {
	case "contribution":
		return fmt.Sprintf("PESABOX: Habari %s, umethibitishwa kuwa umechangia %s kama Mandatory Savings kwenye kikao cha %s cha tarehe %s. Asante.", fn, fmtTZS(amount), gName, evDate)
	case "share":
		cnt := 1
		if sv := helper.GetValueOfFloat(g, "shareValue"); sv > 0 {
			cnt = int(amount/sv + 0.5)
		}
		return fmt.Sprintf("PESABOX: Habari %s, umethibitishwa kununua shares %d zenye thamani ya %s kwenye %s tarehe %s. Asante.", fn, cnt, fmtTZS(amount), gName, evDate)
	case "social_fund":
		return fmt.Sprintf("PESABOX: Habari %s, umethibitishwa kuwa umechangia %s kama Social Fund kwenye kikao cha %s cha tarehe %s. Asante.", fn, fmtTZS(amount), gName, evDate)
	case "loan_repayment":
		return fmt.Sprintf("PESABOX: Habari %s, umethibitishwa kulipa %s kama malipo ya mkopo kwenye %s tarehe %s. Asante.", fn, fmtTZS(amount), gName, evDate)
	case "loan_disbursement":
		return fmt.Sprintf("PESABOX: Habari %s, umethibitishwa kupokea mkopo wa %s kutoka %s tarehe %s. Mrejesho ni kulingana na mkataba wa kikundi.", fn, fmtTZS(amount), gName, evDate)
	case "fine":
		return fmt.Sprintf("PESABOX: Habari %s, umelipa faini ya %s kwenye %s tarehe %s. Asante.", fn, fmtTZS(amount), gName, evDate)
	}
	return fmt.Sprintf("PESABOX: Habari %s, umethibitishwa kuchangia %s kwenye %s tarehe %s. Asante.", fn, fmtTZS(amount), gName, evDate)
}

func main() {
	loadDotEnv("./.env")
	if apiKey, secretKey := os.Getenv("BEEM_API_KEY"), os.Getenv("BEEM_SECRET_KEY"); apiKey != "" && secretKey != "" {
		masked := apiKey
		if len(masked) > 6 {
			masked = masked[:6] + "…"
		}
		fmt.Printf("BEEM_API_KEY loaded (%s) — member OTPs will send real SMS via Beem Africa.\n", masked)
	} else {
		fmt.Println("BEEM_API_KEY/BEEM_SECRET_KEY not set — member OTPs stay in dev mode (constant code, console log only).")
	}

	yekonga.ServerLoad("./config.json", "./database.json")
	app := yekonga.Server

	// Texts the login OTP via Beem (same sendOtpSms used for Add Member's
	// phone verification) when Beem credentials are set; otherwise falls back
	// to printing it, same as before. UserVerification's usernameType can be
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

	// Group.memberCount/femaleMembers/maleMembers are denormalized counters
	// (see database.json) that the dashboard and groups list read directly
	// instead of counting Members on every request. Nothing kept them in
	// sync with the Members collection, so a group's counts stayed frozen
	// at their creation-time default of 0 no matter how many members were
	// actually added. Recomputing from a live count on every Member
	// create/update/delete keeps them accurate.
	recalcGroupMemberCounts := func(groupId string) {
		if helper.IsEmpty(groupId) {
			return
		}

		members := app.ModelQuery("Member").SkipBeforeCommit().Where("groupId", groupId).Find(nil)

		var total, female, male int
		for _, member := range *members {
			total++
			switch helper.GetValueOfString(member, "gender") {
			case "Female":
				female++
			case "Male":
				male++
			}
		}

		app.ModelQuery("Group").SkipBeforeCommit().Where("id", groupId).Update(datatype.DataMap{
			"memberCount":   total,
			"femaleMembers": female,
			"maleMembers":   male,
		}, nil)
	}

	recalcGroupMemberCountsTrigger := func(req *yekonga.RequestContext, ctx *yekonga.QueryContext) (interface{}, error) {
		data := helper.ToDataMap(ctx.Data)
		recalcGroupMemberCounts(helper.GetValueOfString(data, "groupId"))
		return nil, nil
	}

	app.AfterCreate("Member", nil, nil, recalcGroupMemberCountsTrigger)
	app.AfterUpdate("Member", nil, nil, recalcGroupMemberCountsTrigger)
	app.AfterDelete("Member", nil, nil, recalcGroupMemberCountsTrigger)

	// One-time backfill so groups created/joined before the triggers above
	// existed (their counters are stuck at the creation-time default of 0)
	// get corrected as soon as the server restarts, without waiting for
	// their next member change.
	for _, group := range *app.ModelQuery("Group").SkipBeforeCommit().Find(nil) {
		recalcGroupMemberCounts(helper.GetValueOfString(group, "id"))
	}

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
		if helper.IsNotEmpty(os.Getenv("BEEM_API_KEY")) && helper.IsNotEmpty(os.Getenv("BEEM_SECRET_KEY")) {
			code = helper.GetRandomInt(4)
		}

		app.ModelQuery("MemberVerification").SkipBeforeCommit().Create(datatype.DataMap{
			"phone":     phone,
			"code":      code,
			"verified":  false,
			"expiresAt": time.Now().Add(memberOtpTTL),
		})

		message := fmt.Sprintf("Your PesaBox member verification code is %s", code)
		sendOtpSms(phone, message)
		sendSmsWithLog(
			helper.GetValueOfString(helper.ToDataMap(req.Auth()), "id"),
			"", "", phone, "member_otp", "PESABOX: "+message,
		)

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

		// Member Joined confirmation SMS (spec §8): the member does not have a
		// PesaBox login, so the SMS is their proof of membership.
		member := helper.ToDataMap(created)
		groupRec := app.ModelQuery("Group").SkipBeforeCommit().Where("id", groupId).First(nil)
		if groupRec != nil {
			sendSmsWithLog(
				helper.GetValueOfString(helper.ToDataMap(req.Auth()), "id"),
				groupId,
				helper.GetValueOfString(member, "id"),
				phone,
				"member_joined",
				joinedSmsText(*groupRec, member),
			)
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

	// Cross-group platform snapshot for the Super Admin dashboard: totals
	// rolled up from every Group's live counters, a 7-day transaction chart,
	// the latest activity across all groups, and a best-effort status check
	// of the services the dashboard cares about. Unlike "/api/main/*" this
	// is deliberately not group-scoped — a Super Admin has no adminGroup().
	app.Get("/api/admin/dashboard", func(req *yekonga.Request, res *yekonga.Response) {
		if req.Auth() == nil {
			res.Status(401)
			res.Json(map[string]string{"error": "unauthorized"})
			return
		}

		groups := app.ModelQuery("Group").SkipBeforeCommit().Find(nil)
		groupById := map[string]datatype.DataMap{}
		totalSavings, totalLoans := 0.0, 0.0
		if groups != nil {
			for _, g := range *groups {
				groupById[helper.GetValueOfString(g, "id")] = g
				totalSavings += helper.GetValueOfFloat(g, "totalSavings")
				totalLoans += helper.GetValueOfFloat(g, "totalLoans")
			}
		}

		members := app.ModelQuery("Member").SkipBeforeCommit().Find(nil)
		memberById := map[string]datatype.DataMap{}
		if members != nil {
			for _, m := range *members {
				memberById[helper.GetValueOfString(m, "id")] = m
			}
		}

		txns := app.ModelQuery("Transaction").SkipBeforeCommit().OrderBy("createdAt", "desc").Find(nil)

		typeLabels := map[string]string{
			"contribution":      "Saving",
			"share":             "Share",
			"social_fund":       "Social Fund",
			"loan_disbursement": "Loan",
			"loan_repayment":    "Repayment",
			"fine":              "Fine",
			"expense":           "Expense",
			"withdrawal":        "Withdrawal",
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
		if txns != nil {
			for _, t := range *txns {
				createdAt := helper.GetValueOfDate(t, "createdAt")
				if !createdAt.IsZero() && !createdAt.Before(weekAgo) {
					key := createdAt.Format("2006-01-02")
					if _, ok := chartByDay[key]; ok {
						chartByDay[key]++
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
				groupName := ""
				if gg, ok := groupById[helper.GetValueOfString(t, "groupId")]; ok {
					groupName = helper.GetValueOfString(gg, "name")
				}
				recentActivity = append(recentActivity, datatype.DataMap{
					"time":      createdAt.Format("15:04"),
					"member":    memberName,
					"group":     groupName,
					"type":      label,
					"amount":    helper.GetValueOfFloat(t, "amount"),
					"direction": helper.GetValueOfString(t, "direction"),
				})
			}
		}

		chart := make([]datatype.DataMap, 0, 7)
		for _, key := range dayOrder {
			d, _ := time.Parse("2006-01-02", key)
			chart = append(chart, datatype.DataMap{"day": d.Format("Mon"), "count": chartByDay[key]})
		}

		totalGroups := 0
		if groups != nil {
			totalGroups = len(*groups)
		}
		totalMembers := 0
		if members != nil {
			totalMembers = len(*members)
		}

		res.Json(datatype.DataMap{
			"totalGroups":           totalGroups,
			"totalMembers":          totalMembers,
			"totalSavings":          totalSavings,
			"totalLoansOutstanding": totalLoans,
			"chart":                 chart,
			"recentActivity":        recentActivity,
			"status": datatype.DataMap{
				"api":            true,
				"database":       groups != nil,
				"authentication": true,
				"smsProvider":    helper.IsNotEmpty(os.Getenv("BEEM_API_KEY")) && helper.IsNotEmpty(os.Getenv("BEEM_SECRET_KEY")),
				"backgroundJobs": false,
			},
		})
	})

	// Self-service profile completion: the phone-OTP flow auto-creates a User
	// with no firstName/lastName (see yekonga's GetUser), so the web app sends
	// the new admin here right after their first OTP verification to fill
	// those in before letting them into the dashboard. Deliberately does not
	// touch "username" — that field is the phone number the OTP login looks
	// the account up by (see GetUser's Where("username", ...)), so changing it
	// here would lock the account out of its own login.
	app.Post("/api/me", func(req *yekonga.Request, res *yekonga.Response) {
		auth := req.Auth()
		if auth == nil {
			res.Status(401)
			res.Json(map[string]string{"error": "unauthorized"})
			return
		}

		body := helper.ToDataMap(req.Body())
		firstName := strings.TrimSpace(helper.GetValueOfString(body, "firstName"))
		lastName := strings.TrimSpace(helper.GetValueOfString(body, "lastName"))
		email := strings.TrimSpace(helper.GetValueOfString(body, "email"))

		if firstName == "" {
			res.Status(400)
			res.Json(map[string]string{"error": "firstName is required"})
			return
		}

		changes := datatype.DataMap{"firstName": firstName, "lastName": lastName}
		if email != "" {
			changes["email"] = email
		}

		updated := app.ModelQuery("User").SkipBeforeCommit().Where("id", auth.ID).Update(changes, nil)
		if updated == nil {
			res.Status(404)
			res.Json(map[string]string{"error": "user not found"})
			return
		}

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
		auth := req.Auth()
		if auth == nil {
			return nil
		}
		// req.Auth() returns a typed *AuthPayload struct, not a map — reading
		// it through helper.ToDataMap (which only knows how to convert an
		// actual map/pointer-to-map) silently yields an empty map, so userId
		// and phoneTail always come back "" and every one of these
		// requireGroup-protected endpoints 403s with "no group assigned to
		// this account" even for a session that legitimately owns a group.
		// Read the struct's fields directly instead.
		userId := auth.ID
		phoneTail := last9Digits(auth.Username)
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
				case "fine":
					adjustGroupTotal(groupId, "totalFines", amount)
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

	// Every member-facing SMS write goes through sendSmsWithLog, so this is the
	// full SMS history. Group admins see only their own group's messages; a
	// super admin (no group assigned) sees everything.
	app.Get("/api/main/sms/activity", func(req *yekonga.Request, res *yekonga.Response) {
		if req.Auth() == nil {
			res.Status(401)
			res.Json(map[string]string{"error": "unauthorized"})
			return
		}
		recs := app.ModelQuery("SmsLog").SkipBeforeCommit().OrderBy("sentAt", "desc").Find(nil)
		result := []datatype.DataMap{}
		if g := adminGroup(req); g != nil {
			members := app.ModelQuery("Member").SkipBeforeCommit().Where("groupId", helper.GetValueOfString(*g, "id")).Find(nil)
			byId := map[string]datatype.DataMap{}
			for _, m := range *members {
				byId[helper.GetValueOfString(m, "id")] = m
			}
			for _, r := range *recs {
				if helper.GetValueOfString(r, "groupId") != helper.GetValueOfString(*g, "id") {
					continue
				}
				rec := datatype.DataMap{}
				for k, v := range r {
					rec[k] = v
				}
				if mm, ok := byId[helper.GetValueOfString(r, "memberId")]; ok {
					rec["memberName"] = memberFullName(mm)
				}
				result = append(result, rec)
			}
		} else {
			members := app.ModelQuery("Member").SkipBeforeCommit().Find(nil)
			byId := map[string]datatype.DataMap{}
			for _, m := range *members {
				byId[helper.GetValueOfString(m, "id")] = m
			}
			groups := app.ModelQuery("Group").SkipBeforeCommit().Find(nil)
			groupById := map[string]datatype.DataMap{}
			for _, gr := range *groups {
				groupById[helper.GetValueOfString(gr, "id")] = gr
			}
			for _, r := range *recs {
				rec := datatype.DataMap{}
				for k, v := range r {
					rec[k] = v
				}
				if mm, ok := byId[helper.GetValueOfString(r, "memberId")]; ok {
					rec["memberName"] = memberFullName(mm)
				}
				if gr, ok := groupById[helper.GetValueOfString(r, "groupId")]; ok {
					rec["groupName"] = helper.GetValueOfString(gr, "name")
				}
				result = append(result, rec)
			}
		}
		res.Json(result)
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

		// Shares can be entered as a count ("3 shares"): total = count × value.
		if typ == "share" && amount <= 0 {
			if cnt := helper.GetValueOfInt(body, "shareCount"); cnt > 0 {
				if sv := helper.GetValueOfFloat(*g, "shareValue"); sv > 0 {
					amount = float64(cnt) * sv
				}
			}
		}
		if typ == "" || amount <= 0 {
			res.Status(400)
			res.Json(map[string]string{"error": "type and a positive amount are required"})
			return
		}

		// Rule validation (spec §15) — the admin configures rules on the group,
		// and the server rejects transactions that violate them instead of
		// silently recording an off-rule amount.
		msa := helper.GetValueOfFloat(*g, "mandatorySavingsAmount")
		if msa > 0 && typ == "contribution" && amount != msa {
			res.Status(400)
			res.Json(map[string]string{"error": fmt.Sprintf("Mandatory Savings is %s per meeting", fmtTZS(msa))})
			return
		}
		if typ == "share" {
			sv := helper.GetValueOfFloat(*g, "shareValue")
			mn := helper.GetValueOfInt(*g, "minShares")
			mx := helper.GetValueOfInt(*g, "maxShares")
			if sv > 0 {
				cnt := int(amount/sv + 0.5)
				if math.Abs(amount-float64(cnt)*sv) > 0.01 {
					res.Status(400)
					res.Json(map[string]string{"error": fmt.Sprintf("Share amounts must be exact multiples of the share value (%s)", fmtTZS(sv))})
					return
				}
				if mn > 0 && cnt < mn {
					res.Status(400)
					res.Json(map[string]string{"error": fmt.Sprintf("Minimum shares per meeting is %d", mn)})
					return
				}
				if mx > 0 && cnt > mx {
					res.Status(400)
					res.Json(map[string]string{"error": fmt.Sprintf("Maximum shares per meeting is %d", mx)})
					return
				}
			}
		}

		// Reflect any derived amount (e.g. shares entered as a count) back into
		// the record createTransaction persists.
		body["amount"] = amount

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

		// Contribution confirmation SMS (spec §16) — only for member-scoped,
		// money-in transactions; expenses/withdrawals don't text anyone.
		memberId := helper.GetValueOfString(body, "memberId")
		switch typ {
		case "contribution", "share", "social_fund", "loan_repayment", "fine":
			if memberId != "" {
				if member := app.ModelQuery("Member").SkipBeforeCommit().Where("id", memberId).Where("groupId", groupId).First(nil); member != nil {
					sendSmsWithLog(
						helper.GetValueOfString(helper.ToDataMap(req.Auth()), "id"),
						groupId,
						memberId,
						helper.GetValueOfString(*member, "phone"),
						typ,
						txSmsText(*g, *member, typ, amount, meetingSmsDate(helper.GetValueOfString(body, "meetingId"))),
					)
				}
			}
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

		// Loan disbursement confirmation SMS (spec §23).
		sendSmsWithLog(
			helper.GetValueOfString(helper.ToDataMap(req.Auth()), "id"),
			groupId,
			memberId,
			helper.GetValueOfString(*rec, "phone"),
			"loan_disbursement",
			txSmsText(*g, *rec, "loan_disbursement", amount, meetingSmsDate(helper.GetValueOfString(body, "meetingId"))),
		)

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

		// Loan repayment confirmation SMS (spec §23).
		if memberId := helper.GetValueOfString(*loan, "memberId"); memberId != "" {
			if member := app.ModelQuery("Member").SkipBeforeCommit().Where("id", memberId).Where("groupId", groupId).First(nil); member != nil {
				sendSmsWithLog(
					helper.GetValueOfString(helper.ToDataMap(req.Auth()), "id"),
					groupId,
					memberId,
					helper.GetValueOfString(*member, "phone"),
					"loan_repayment",
					txSmsText(*g, *member, "loan_repayment", amount, meetingSmsDate(helper.GetValueOfString(body, "meetingId"))),
				)
			}
		}

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
		reason := helper.GetValueOfString(body, "reason")
		amount := helper.GetValueOfFloat(body, "amount")

		// The amount comes from the admin's pick of a configured fine reason
		// (spec §12): "Late Attendance — TZS 1,000". An explicit typed amount
		// still wins (used for the 'Other' reason where configured amount is 0).
		amount = fineAmountFor(*g, reason, amount)

		if memberId == "" || reason == "" || amount <= 0 {
			res.Status(400)
			res.Json(map[string]string{"error": "memberId, a reason and a positive amount are required (or a configured fine reason)"})
			return
		}
		rec := app.ModelQuery("Member").SkipBeforeCommit().Where("id", memberId).Where("groupId", groupId).First(nil)
		if rec == nil {
			res.Status(404)
			res.Json(map[string]string{"error": "member not found in your group"})
			return
		}
		created := app.ModelQuery("Fine").SkipBeforeCommit().Create(datatype.DataMap{
			"groupId":    groupId,
			"memberId":   memberId,
			"meetingId":  helper.GetValueOfString(body, "meetingId"),
			"reason":     reason,
			"amount":     amount,
			"amountPaid": 0,
			"status":     "pending",
			"issuedAt":   time.Now(),
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

		// Fine confirmation SMS (spec §13) — the SMS is the member's proof.
		sendSmsWithLog(
			helper.GetValueOfString(helper.ToDataMap(req.Auth()), "id"),
			groupId,
			memberId,
			helper.GetValueOfString(*rec, "phone"),
			"fine",
			fineSmsText(*g, *rec, reason, amount, meetingSmsDate(helper.GetValueOfString(body, "meetingId"))),
		)

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

		// Fine payment confirmation SMS (spec §23 — "Fine Applied → ...").
		if memberId := helper.GetValueOfString(*fine, "memberId"); memberId != "" {
			if member := app.ModelQuery("Member").SkipBeforeCommit().Where("id", memberId).Where("groupId", groupId).First(nil); member != nil {
				sendSmsWithLog(
					helper.GetValueOfString(helper.ToDataMap(req.Auth()), "id"),
					groupId,
					memberId,
					helper.GetValueOfString(*member, "phone"),
					"fine_payment",
					txSmsText(*g, *member, "fine", amount, meetingSmsDate(helper.GetValueOfString(body, "meetingId"))),
				)
			}
		}

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
