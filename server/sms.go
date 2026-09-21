package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/robertkonga/yekonga-server-go/datatype"
	"github.com/robertkonga/yekonga-server-go/helper"
	"github.com/robertkonga/yekonga-server-go/yekonga"
)

// SMS automation: editable message templates (Swahili default, English
// available), on/off switches persisted in PlatformSettings, and a
// background worker for meeting / loan reminders. Delivery itself stays in
// main.go (smtzSend / sendSmsWithLog) — every member-facing SMS already
// passes through sendSmsWithLog, so the switches are enforced there.

// smsTemplateDef describes one editable message. Bodies use {PLACEHOLDERS}:
//
//	{JINA} member name   {KIKUNDI} group name   {KIASI} amount (TZS)
//	{TAREHE} date        {SABABU} fine reason   {IDADI} number of shares
//	{CODE} one-time code (OTP templates only)
type smsTemplateDef struct {
	Type     string
	Category string
	Sw       string // label shown in the admin UI
	En       string
	Vars     []string
	Body     map[string]string // language -> default body
}

var smsTemplateDefs = []smsTemplateDef{
	{"login_otp", "OTP", "Nambari ya Kuingia (OTP)", "Login Code (OTP)", []string{"CODE"}, map[string]string{
		"sw": "PESABOX: Nambari yako ya kuingia ni {CODE}. Usimpe mtu yeyote.",
		"en": "PESABOX: Your login code is {CODE}. Do not share it with anyone.",
	}},
	{"member_otp", "OTP", "Nambari ya Kuthibitisha Mwanachama", "Member Verification Code", []string{"CODE"}, map[string]string{
		"sw": "PESABOX: Nambari yako ya kuthibitisha ni {CODE}. Usimpe mtu yeyote.",
		"en": "PESABOX: Your verification code is {CODE}. Do not share it with anyone.",
	}},
	{"member_joined", "Members", "Mwanachama Amejiunga", "Member Joined", []string{"JINA", "KIKUNDI"}, map[string]string{
		"sw": "PESABOX: Habari {JINA}, umethibitishwa kuwa mwanachama wa {KIKUNDI}. Karibu kwenye kikundi.",
		"en": "PESABOX: Hello {JINA}, you are confirmed as a member of {KIKUNDI}. Welcome to the group.",
	}},
	{"contribution", "Financial", "Mchango wa Lazima", "Mandatory Savings", []string{"JINA", "KIKUNDI", "KIASI", "TAREHE"}, map[string]string{
		"sw": "PESABOX: Habari {JINA}, umethibitishwa kuwa umechangia {KIASI} kama Mandatory Savings kwenye kikao cha {KIKUNDI} cha tarehe {TAREHE}. Asante.",
		"en": "PESABOX: Hello {JINA}, your mandatory savings of {KIASI} at the {KIKUNDI} meeting on {TAREHE} is confirmed. Thank you.",
	}},
	{"share", "Financial", "Ununuzi wa Hisa", "Share Purchase", []string{"JINA", "KIKUNDI", "KIASI", "IDADI", "TAREHE"}, map[string]string{
		"sw": "PESABOX: Habari {JINA}, umethibitishwa kununua shares {IDADI} zenye thamani ya {KIASI} kwenye {KIKUNDI} tarehe {TAREHE}. Asante.",
		"en": "PESABOX: Hello {JINA}, your purchase of {IDADI} share(s) worth {KIASI} in {KIKUNDI} on {TAREHE} is confirmed. Thank you.",
	}},
	{"social_fund", "Financial", "Mfuko wa Jamii", "Social Fund", []string{"JINA", "KIKUNDI", "KIASI", "TAREHE"}, map[string]string{
		"sw": "PESABOX: Habari {JINA}, umethibitishwa kuwa umechangia {KIASI} kama Social Fund kwenye kikao cha {KIKUNDI} cha tarehe {TAREHE}. Asante.",
		"en": "PESABOX: Hello {JINA}, your social fund contribution of {KIASI} at the {KIKUNDI} meeting on {TAREHE} is confirmed. Thank you.",
	}},
	{"loan_disbursement", "Loans", "Mkopo Umetolewa", "Loan Disbursed", []string{"JINA", "KIKUNDI", "KIASI", "TAREHE"}, map[string]string{
		"sw": "PESABOX: Habari {JINA}, umethibitishwa kupokea mkopo wa {KIASI} kutoka {KIKUNDI} tarehe {TAREHE}. Mrejesho ni kulingana na mkataba wa kikundi.",
		"en": "PESABOX: Hello {JINA}, you have received a loan of {KIASI} from {KIKUNDI} on {TAREHE}. Repayment follows the group agreement.",
	}},
	{"loan_repayment", "Loans", "Marejesho ya Mkopo", "Loan Repayment", []string{"JINA", "KIKUNDI", "KIASI", "TAREHE"}, map[string]string{
		"sw": "PESABOX: Habari {JINA}, umethibitishwa kulipa {KIASI} kama malipo ya mkopo kwenye {KIKUNDI} tarehe {TAREHE}. Asante.",
		"en": "PESABOX: Hello {JINA}, your loan repayment of {KIASI} in {KIKUNDI} on {TAREHE} is confirmed. Thank you.",
	}},
	{"fine_issued", "Financial", "Faini Imetolewa", "Fine Issued", []string{"JINA", "KIKUNDI", "KIASI", "SABABU", "TAREHE"}, map[string]string{
		"sw": "PESABOX: Habari {JINA}, umepewa faini ya {KIASI} kutokana na {SABABU} kwenye kikao cha {KIKUNDI} cha tarehe {TAREHE}.",
		"en": "PESABOX: Hello {JINA}, you have been fined {KIASI} for {SABABU} at the {KIKUNDI} meeting on {TAREHE}.",
	}},
	{"fine", "Financial", "Faini Imelipwa", "Fine Paid", []string{"JINA", "KIKUNDI", "KIASI", "TAREHE"}, map[string]string{
		"sw": "PESABOX: Habari {JINA}, umelipa faini ya {KIASI} kwenye {KIKUNDI} tarehe {TAREHE}. Asante.",
		"en": "PESABOX: Hello {JINA}, your fine payment of {KIASI} in {KIKUNDI} on {TAREHE} is confirmed. Thank you.",
	}},
	{"generic", "Financial", "Muamala Mwingine", "Other Transaction", []string{"JINA", "KIKUNDI", "KIASI", "TAREHE"}, map[string]string{
		"sw": "PESABOX: Habari {JINA}, umethibitishwa kuchangia {KIASI} kwenye {KIKUNDI} tarehe {TAREHE}. Asante.",
		"en": "PESABOX: Hello {JINA}, your payment of {KIASI} in {KIKUNDI} on {TAREHE} is confirmed. Thank you.",
	}},
	{"meeting_reminder", "Reminders", "Kikumbusho cha Mkutano", "Meeting Reminder", []string{"JINA", "KIKUNDI", "TAREHE"}, map[string]string{
		"sw": "PESABOX: Habari {JINA}, tunakukumbusha kikao cha {KIKUNDI} kitafanyika kesho, tarehe {TAREHE}. Karibu.",
		"en": "PESABOX: Hello {JINA}, a reminder that the {KIKUNDI} meeting is tomorrow, {TAREHE}. See you there.",
	}},
	{"loan_due_soon", "Reminders", "Mkopo Unakaribia Kuisha", "Loan Due Soon", []string{"JINA", "KIKUNDI", "KIASI", "TAREHE"}, map[string]string{
		"sw": "PESABOX: Habari {JINA}, mkopo wako wa {KIKUNDI} wenye salio la {KIASI} unaisha tarehe {TAREHE}. Tafadhali lipa kwa wakati.",
		"en": "PESABOX: Hello {JINA}, your {KIKUNDI} loan with a balance of {KIASI} is due on {TAREHE}. Please pay on time.",
	}},
	{"loan_overdue", "Reminders", "Mkopo Umechelewa", "Loan Overdue", []string{"JINA", "KIKUNDI", "KIASI", "TAREHE"}, map[string]string{
		"sw": "PESABOX: Habari {JINA}, mkopo wako wa {KIKUNDI} wenye salio la {KIASI} ulipaswa kulipwa tarehe {TAREHE}. Tafadhali lipa haraka.",
		"en": "PESABOX: Hello {JINA}, your {KIKUNDI} loan with a balance of {KIASI} was due on {TAREHE}. Please pay as soon as possible.",
	}},
}

func findSmsTemplate(typ string) *smsTemplateDef {
	for i := range smsTemplateDefs {
		if smsTemplateDefs[i].Type == typ {
			return &smsTemplateDefs[i]
		}
	}
	return nil
}

// ---- persisted switches ---------------------------------------------------

const (
	settingOtpSms           = "otpSms"
	settingTransactionalSms = "transactionalSms"
	settingRemindersSms     = "remindersSms"
	settingSmsLanguage      = "smsLanguage"
)

func getSetting(key, def string) string {
	rec := yekonga.Server.ModelQuery("PlatformSetting").SkipBeforeCommit().Where("key", key).First(nil)
	if rec == nil {
		return def
	}
	if v := helper.GetValueOfString(*rec, "value"); v != "" {
		return v
	}
	return def
}

func setSetting(key, value string) {
	q := yekonga.Server.ModelQuery("PlatformSetting").SkipBeforeCommit()
	if rec := q.Where("key", key).First(nil); rec != nil {
		yekonga.Server.ModelQuery("PlatformSetting").SkipBeforeCommit().
			Where("id", helper.GetValueOfString(*rec, "id")).
			Update(datatype.DataMap{"value": value, "updatedAt": time.Now()}, nil)
		return
	}
	yekonga.Server.ModelQuery("PlatformSetting").SkipBeforeCommit().Create(datatype.DataMap{"key": key, "value": value})
}

// settingOn: every switch defaults to ON until an admin turns it off.
func settingOn(key string) bool { return getSetting(key, "true") != "false" }

func smsLang() string { return normLang(getSetting(settingSmsLanguage, "sw")) }

// smsAllowed reports whether a message of this type may be sent right now.
func smsAllowed(messageType string) bool {
	switch messageType {
	case "login_otp", "member_otp":
		return settingOn(settingOtpSms)
	case "meeting_reminder", "loan_due_soon", "loan_overdue":
		return settingOn(settingRemindersSms)
	}
	return settingOn(settingTransactionalSms)
}

func notificationSettings() datatype.DataMap {
	return datatype.DataMap{
		settingOtpSms:           settingOn(settingOtpSms),
		settingTransactionalSms: settingOn(settingTransactionalSms),
		settingRemindersSms:     settingOn(settingRemindersSms),
		settingSmsLanguage:      smsLang(),
	}
}

// ---- rendering ------------------------------------------------------------

// smsTemplateBody returns the message body for (type, language): the admin's
// saved override when it is active, else the built-in default.
func smsTemplateBody(typ, lang string) string {
	if rec := yekonga.Server.ModelQuery("SmsTemplate").SkipBeforeCommit().Where("type", typ).Where("language", lang).First(nil); rec != nil {
		if helper.GetValueOfBoolean(*rec, "active") {
			if b := strings.TrimSpace(helper.GetValueOfString(*rec, "body")); b != "" {
				return b
			}
		}
	}
	if def := findSmsTemplate(typ); def != nil {
		if b := def.Body[lang]; b != "" {
			return b
		}
		return def.Body["sw"]
	}
	return ""
}

func fillSms(body string, vars map[string]string) string {
	pairs := make([]string, 0, len(vars)*2)
	for k, v := range vars {
		pairs = append(pairs, "{"+k+"}", v)
	}
	return strings.NewReplacer(pairs...).Replace(body)
}

// renderSms builds the final message text for a template type in the
// platform's SMS language (Swahili unless an admin switched it).
func renderSms(typ string, vars map[string]string) string {
	return fillSms(smsTemplateBody(typ, smsLang()), vars)
}

func smsNames(g, m datatype.DataMap) (string, string) {
	gName := helper.GetValueOfString(g, "name")
	fn := memberFullName(m)
	if smsLang() == "en" {
		if gName == "" {
			gName = "the group"
		}
		if fn == "" {
			fn = "Member"
		}
	} else {
		if gName == "" {
			gName = "kikundi"
		}
		if fn == "" {
			fn = "Ndugu"
		}
	}
	return gName, fn
}

// ---- reminders ------------------------------------------------------------

var eatZone = time.FixedZone("EAT", 3*3600) // East Africa Time, no DST

func parseMeetingDate(s string) (time.Time, bool) {
	for _, layout := range []string{"2006-01-02", "02/01/2006", time.RFC3339, "2006-01-02T15:04:05"} {
		if t, err := time.ParseInLocation(layout, strings.TrimSpace(s), eatZone); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}

func dayStart(t time.Time) time.Time {
	t = t.In(eatZone)
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, eatZone)
}

// recentlySent stops the worker texting the same member the same reminder
// again after a restart or on the next 30-minute pass.
func recentlySent(memberId, messageType string, window time.Duration) bool {
	logs := yekonga.Server.ModelQuery("SmsLog").SkipBeforeCommit().
		Where("memberId", memberId).Where("messageType", messageType).Find(nil)
	if logs == nil {
		return false
	}
	cutoff := time.Now().Add(-window)
	for _, l := range *logs {
		if helper.GetTimestamp(l["createdAt"]).After(cutoff) {
			return true
		}
	}
	return false
}

// runReminders sends meeting reminders (the day before) and loan reminders
// (due within 3 days, and overdue weekly). Sends only between 08:00 and 19:00
// EAT so nobody is texted at night.
func runReminders(app *yekonga.YekongaData, now time.Time) {
	if !settingOn(settingRemindersSms) {
		return
	}
	if h := now.In(eatZone).Hour(); h < 8 || h >= 19 {
		return
	}
	today := dayStart(now)

	groups := map[string]datatype.DataMap{}
	if all := app.ModelQuery("Group").SkipBeforeCommit().Find(nil); all != nil {
		for _, g := range *all {
			if st := helper.GetValueOfString(g, "status"); st == "Inactive" || st == "Closed" {
				continue
			}
			groups[helper.GetValueOfString(g, "id")] = g
		}
	}
	members := map[string]datatype.DataMap{}
	byGroup := map[string][]datatype.DataMap{}
	if all := app.ModelQuery("Member").SkipBeforeCommit().Find(nil); all != nil {
		for _, m := range *all {
			if helper.GetValueOfString(m, "status") == "Suspended" {
				continue
			}
			members[helper.GetValueOfString(m, "id")] = m
			gid := helper.GetValueOfString(m, "groupId")
			byGroup[gid] = append(byGroup[gid], m)
		}
	}

	send := func(typ, gid string, m datatype.DataMap, window time.Duration, vars map[string]string) {
		mid := helper.GetValueOfString(m, "id")
		phone := helper.GetValueOfString(m, "phone")
		if phone == "" || recentlySent(mid, typ, window) {
			return
		}
		g := groups[gid]
		gName, fn := smsNames(g, m)
		vars["JINA"], vars["KIKUNDI"] = fn, gName
		sendSmsWithLog("", gid, mid, phone, typ, renderSms(typ, vars))
	}

	if meetings := app.ModelQuery("Meeting").SkipBeforeCommit().Where("status", "upcoming").Find(nil); meetings != nil {
		for _, mt := range *meetings {
			gid := helper.GetValueOfString(mt, "groupId")
			if _, ok := groups[gid]; !ok {
				continue
			}
			d, ok := parseMeetingDate(helper.GetValueOfString(mt, "date"))
			if !ok || !dayStart(d).Equal(today.AddDate(0, 0, 1)) {
				continue
			}
			for _, m := range byGroup[gid] {
				send("meeting_reminder", gid, m, 20*time.Hour, map[string]string{"TAREHE": dmy(d)})
			}
		}
	}

	if loans := app.ModelQuery("Loan").SkipBeforeCommit().Where("status", "active").Find(nil); loans != nil {
		for _, l := range *loans {
			gid := helper.GetValueOfString(l, "groupId")
			m, ok := members[helper.GetValueOfString(l, "memberId")]
			if _, gok := groups[gid]; !gok || !ok {
				continue
			}
			balance := helper.GetValueOfFloat(l, "amount") - helper.GetValueOfFloat(l, "amountRepaid")
			due := helper.GetTimestamp(l["dueDate"])
			if balance <= 0 || due.IsZero() {
				continue
			}
			days := int(dayStart(due).Sub(today).Hours() / 24)
			vars := map[string]string{"KIASI": fmtTZS(balance), "TAREHE": dmy(due)}
			switch {
			case days < 0:
				send("loan_overdue", gid, m, 7*24*time.Hour, vars)
			case days <= 3:
				send("loan_due_soon", gid, m, 48*time.Hour, vars)
			}
		}
	}
}

// startSmsReminders runs the reminder pass every 30 minutes. It is opt-in
// (PESABOX_REMINDERS=1): with a real SMTZ key it texts real members, so it must
// never start by accident — e.g. when a developer runs the server against a
// copy of production data.
func startSmsReminders(app *yekonga.YekongaData) {
	if os.Getenv("PESABOX_REMINDERS") != "1" {
		fmt.Println("sms reminders: off (opt-in — set PESABOX_REMINDERS=1 to send meeting/loan reminders)")
		return
	}
	go func() {
		for {
			func() {
				defer func() {
					if r := recover(); r != nil {
						fmt.Println("sms reminders: recovered from panic:", r)
					}
				}()
				runReminders(app, time.Now())
			}()
			time.Sleep(30 * time.Minute)
		}
	}()
	fmt.Println("sms reminders: worker started (every 30 minutes, 08:00-19:00 EAT)")
}

// ---- admin endpoints ------------------------------------------------------

func boolField(body datatype.DataMap, key string) (bool, bool) {
	v, ok := body[key]
	if !ok {
		return false, false
	}
	switch b := v.(type) {
	case bool:
		return b, true
	case string:
		return b == "true", true
	}
	return false, false
}

// registerSmsAdmin wires:
//
//	GET  /api/admin/sms/templates   every template (sw + en), with overrides applied
//	POST /api/admin/sms/templates   save {type, language, body, active}; body "" restores the default
//	POST /api/admin/settings        save {otpSms, transactionalSms, remindersSms, smsLanguage}
//
// Writes are super-admin only.
func registerSmsAdmin(app *yekonga.YekongaData) {
	superOnly := func(req *yekonga.Request, res *yekonga.Response) (string, bool) {
		auth := req.Auth()
		if auth == nil {
			res.Status(401)
			res.Json(map[string]string{"error": "unauthorized"})
			return "", false
		}
		if !isPlatformAdmin(sessionRole(app, auth.ID)) {
			res.Status(403)
			res.Json(map[string]string{"error": "only the super admin can change SMS settings"})
			return "", false
		}
		return auth.ID, true
	}

	app.Get("/api/admin/sms/templates", func(req *yekonga.Request, res *yekonga.Response) {
		if req.Auth() == nil {
			res.Status(401)
			res.Json(map[string]string{"error": "unauthorized"})
			return
		}
		saved := map[string]datatype.DataMap{}
		if all := app.ModelQuery("SmsTemplate").SkipBeforeCommit().Find(nil); all != nil {
			for _, r := range *all {
				saved[helper.GetValueOfString(r, "type")+"|"+helper.GetValueOfString(r, "language")] = r
			}
		}
		out := []datatype.DataMap{}
		for _, d := range smsTemplateDefs {
			for _, lang := range []string{"sw", "en"} {
				body, custom, active := d.Body[lang], false, true
				if r, ok := saved[d.Type+"|"+lang]; ok {
					custom = true
					active = helper.GetValueOfBoolean(r, "active")
					if b := strings.TrimSpace(helper.GetValueOfString(r, "body")); b != "" {
						body = b
					}
				}
				out = append(out, datatype.DataMap{
					"type": d.Type, "category": d.Category, "sw": d.Sw, "en": d.En, "language": lang,
					"body": body, "defaultBody": d.Body[lang], "custom": custom, "active": active, "variables": d.Vars,
				})
			}
		}
		res.Json(out)
	})

	app.Post("/api/admin/sms/templates", func(req *yekonga.Request, res *yekonga.Response) {
		userId, ok := superOnly(req, res)
		if !ok {
			return
		}
		body := bodyMap(req)
		typ := helper.GetValueOfString(body, "type")
		lang := strings.ToLower(helper.GetValueOfString(body, "language"))
		text := strings.TrimSpace(helper.GetValueOfString(body, "body"))
		def := findSmsTemplate(typ)
		if def == nil || (lang != "sw" && lang != "en") {
			res.Status(400)
			res.Json(map[string]string{"error": "unknown template type or language"})
			return
		}
		if len([]rune(text)) > 480 {
			res.Status(400)
			res.Json(map[string]string{"error": "message is too long (max 480 characters)"})
			return
		}

		existing := app.ModelQuery("SmsTemplate").SkipBeforeCommit().Where("type", typ).Where("language", lang).First(nil)
		if text == "" { // empty body = go back to the built-in default
			if existing != nil {
				app.ModelQuery("SmsTemplate").SkipBeforeCommit().Where("id", helper.GetValueOfString(*existing, "id")).Delete(nil)
			}
			res.Json(map[string]interface{}{"ok": true, "restored": true})
			return
		}
		active := true
		if b, present := boolField(body, "active"); present {
			active = b
		}
		data := datatype.DataMap{"body": text, "active": active, "updatedBy": userId, "updatedAt": time.Now()}
		if existing != nil {
			app.ModelQuery("SmsTemplate").SkipBeforeCommit().Where("id", helper.GetValueOfString(*existing, "id")).Update(data, nil)
		} else {
			data["type"], data["language"] = typ, lang
			app.ModelQuery("SmsTemplate").SkipBeforeCommit().Create(data)
		}
		res.Json(map[string]interface{}{"ok": true})
	})

	app.Post("/api/admin/settings", func(req *yekonga.Request, res *yekonga.Response) {
		if _, ok := superOnly(req, res); !ok {
			return
		}
		body := bodyMap(req)
		for _, k := range []string{settingOtpSms, settingTransactionalSms, settingRemindersSms} {
			if b, present := boolField(body, k); present {
				setSetting(k, fmt.Sprint(b))
			}
		}
		if l := helper.GetValueOfString(body, settingSmsLanguage); l == "sw" || l == "en" {
			setSetting(settingSmsLanguage, l)
		}
		res.Json(notificationSettings())
	})
}
