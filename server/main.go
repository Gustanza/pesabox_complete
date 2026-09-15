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

	app.Start(8090)
}
