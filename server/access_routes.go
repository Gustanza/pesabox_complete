package main

import (
	"fmt"
	"strings"

	"github.com/robertkonga/yekonga-server-go/datatype"
	"github.com/robertkonga/yekonga-server-go/helper"
	"github.com/robertkonga/yekonga-server-go/yekonga"
)

// officerPositions are the group positions a Group Admin may hand out.
// "mwenyekiti" (Group Admin authority) is deliberately missing — only a
// super admin / staff can assign it (PDF: officers "cannot assign Group Admin
// authority").
var officerPositions = map[string]bool{"katibu": true, "mweka_hazina": true, "committee": true}

func actorJSON(app *yekonga.YekongaData, a *Actor) datatype.DataMap {
	homeGroup := datatype.DataMap(nil)
	if a.HomeGroupID != "" {
		if g := app.ModelQuery("Group").SkipBeforeCommit().Where("id", a.HomeGroupID).First(nil); g != nil {
			homeGroup = datatype.DataMap{"id": a.HomeGroupID, "name": helper.GetValueOfString(*g, "name")}
		}
	}
	return datatype.DataMap{
		"userId":      a.UserID,
		"name":        a.Name,
		"role":        a.Role,
		"preset":      a.Preset,
		"permissions": a.PermList(),
		"platform":    sortedKeys(a.Perms),
		"all":         a.All,
		"partnerIds":  sortedKeys(a.PartnerIDs),
		"clusterIds":  sortedKeys(a.ClusterIDs),
		"groupIds":    sortedKeys(a.GroupIDs),
		"homeGroup":   homeGroup,
		"position":    a.Position,
	}
}

// userSummary is the sanitized view of a User (never password/otp fields).
func userSummary(u datatype.DataMap, preset string, assignments []datatype.DataMap) datatype.DataMap {
	return datatype.DataMap{
		"id":          helper.GetValueOfString(u, "id"),
		"firstName":   helper.GetValueOfString(u, "firstName"),
		"lastName":    helper.GetValueOfString(u, "lastName"),
		"username":    helper.GetValueOfString(u, "username"),
		"phone":       helper.GetValueOfString(u, "phone"),
		"email":       helper.GetValueOfString(u, "email"),
		"role":        normalizeRole(helper.GetValueOfString(u, "role")),
		"preset":      preset,
		"status":      helper.GetValueOfString(u, "status"),
		"isActive":    helper.GetValueOfString(u, "status") != "inactive",
		"createdAt":   u["createdAt"],
		"assignments": assignments,
	}
}

// findOrCreateUser returns the User whose login phone matches phone, creating
// one (the same shape the OTP login would) so the person can sign in with an
// OTP straight away.
func findOrCreateUser(app *yekonga.YekongaData, phone, firstName, lastName, role string) (datatype.DataMap, bool) {
	tail := last9(phone)
	if tail == "" {
		return nil, false
	}
	for _, u := range listAll(app, "User") {
		if last9(helper.GetValueOfString(u, "username")) == tail {
			return u, false
		}
	}
	username := helper.PhoneFormat(phone)
	created := app.ModelQuery("User").SkipBeforeCommit().Create(datatype.DataMap{
		"usernameType": "phone",
		"username":     username,
		"phone":        username,
		"firstName":    firstName,
		"lastName":     lastName,
		"role":         role,
		"status":       "active",
		"isActive":     true,
		"userType":     "individual",
		"createdAt":    helper.GetTimestamp(nil),
		"updatedAt":    helper.GetTimestamp(nil),
	})
	if created == nil {
		return nil, false
	}
	if _, isErr := created.(error); isErr {
		return nil, false
	}
	return helper.ToDataMap(created), true
}

func activeSuperAdmins(app *yekonga.YekongaData) int {
	n := 0
	for _, u := range listAll(app, "User") {
		if normalizeRole(helper.GetValueOfString(u, "role")) == RoleSuperAdmin && helper.GetValueOfString(u, "status") != "inactive" {
			n++
		}
	}
	return n
}

func registerAccessRoutes(app *yekonga.YekongaData) {
	// Who am I and what may I do — the web and the app use this to show/hide
	// menus and actions. The server re-checks everything anyway.
	app.Get("/api/access/me", func(req *yekonga.Request, res *yekonga.Response) {
		a := requireActor(app, req, res)
		if a == nil {
			return
		}
		res.Json(actorJSON(app, a))
	})

	// ---- users & roles (super admin) ------------------------------------

	app.Get("/api/admin/users", func(req *yekonga.Request, res *yekonga.Response) {
		if requirePerm(app, req, res, PermPlatform) == nil {
			return
		}
		presets := map[string]string{}
		for _, p := range listAll(app, "AccessProfile") {
			presets[helper.GetValueOfString(p, "userId")] = helper.GetValueOfString(p, "preset")
		}
		byUser := map[string][]datatype.DataMap{}
		for _, as := range listAll(app, "Assignment") {
			uid := helper.GetValueOfString(as, "userId")
			byUser[uid] = append(byUser[uid], assignmentJSON(app, as))
		}
		out := []datatype.DataMap{}
		for _, u := range listAll(app, "User") {
			id := helper.GetValueOfString(u, "id")
			out = append(out, userSummary(u, presets[id], byUser[id]))
		}
		res.Json(out)
	})

	// Add a person by phone (they then sign in with an OTP).
	app.Post("/api/admin/users", func(req *yekonga.Request, res *yekonga.Response) {
		a := requirePerm(app, req, res, PermPlatform)
		if a == nil {
			return
		}
		body := bodyMap(req)
		phone := helper.GetValueOfString(body, "phone")
		role := helper.GetValueOfString(body, "role")
		if last9(phone) == "" || !validRole(role) {
			deny(res, 400, "a phone number and a valid role are required")
			return
		}
		u, created := findOrCreateUser(app, phone, strings.TrimSpace(helper.GetValueOfString(body, "firstName")), strings.TrimSpace(helper.GetValueOfString(body, "lastName")), role)
		if u == nil {
			deny(res, 500, "could not create the user")
			return
		}
		id := helper.GetValueOfString(u, "id")
		if !created {
			app.ModelQuery("User").SkipBeforeCommit().Where("id", id).Update(datatype.DataMap{"role": role, "status": "active", "isActive": true}, nil)
		}
		if role == RoleStaff {
			preset := helper.GetValueOfString(body, "preset")
			if !validPreset(preset) {
				preset = PresetViewer
			}
			upsertPreset(app, id, preset, a.UserID)
		}
		invalidateActors()
		writeAudit(app, a, "create", "User", id, "", fmt.Sprintf("Added %s as %s", phone, role), nil)
		res.Json(datatype.DataMap{"success": true, "id": id, "created": created})
	})

	app.Post("/api/admin/users/:id", func(req *yekonga.Request, res *yekonga.Response) {
		a := requirePerm(app, req, res, PermPlatform)
		if a == nil {
			return
		}
		id := req.Param("id")
		user := app.ModelQuery("User").SkipBeforeCommit().Where("id", id).First(nil)
		if user == nil {
			deny(res, 404, "user not found")
			return
		}
		body := bodyMap(req)
		changes := datatype.DataMap{}
		currentRole := normalizeRole(helper.GetValueOfString(*user, "role"))

		if v, ok := body["role"]; ok {
			role := fmt.Sprint(v)
			if role != "" && !validRole(role) {
				deny(res, 400, "unknown role")
				return
			}
			if id == a.UserID && role != currentRole {
				deny(res, 400, "you cannot change your own role")
				return
			}
			if currentRole == RoleSuperAdmin && role != RoleSuperAdmin && activeSuperAdmins(app) <= 1 {
				deny(res, 400, "there must always be at least one super admin")
				return
			}
			changes["role"] = role
		}
		if v, ok := body["status"]; ok {
			status := fmt.Sprint(v)
			if status != "active" && status != "inactive" {
				deny(res, 400, "status must be active or inactive")
				return
			}
			if status == "inactive" && id == a.UserID {
				deny(res, 400, "you cannot deactivate yourself")
				return
			}
			if status == "inactive" && currentRole == RoleSuperAdmin && activeSuperAdmins(app) <= 1 {
				deny(res, 400, "there must always be at least one super admin")
				return
			}
			changes["status"] = status
			changes["isActive"] = status == "active"
		}
		if p, ok := body["preset"]; ok {
			preset := fmt.Sprint(p)
			if !validPreset(preset) {
				deny(res, 400, "preset must be viewer, support or operations")
				return
			}
			upsertPreset(app, id, preset, a.UserID)
		}
		if len(changes) > 0 {
			app.ModelQuery("User").SkipBeforeCommit().Where("id", id).Update(changes, nil)
		}
		invalidateActors()
		action := "update"
		if st, ok := changes["status"]; ok {
			action = map[bool]string{true: "reactivate", false: "deactivate"}[st == "active"]
		}
		writeAudit(app, a, action, "User", id, "", "Changed access for "+helper.GetValueOfString(*user, "username"), body)
		res.Json(map[string]bool{"success": true})
	})

	// People are never deleted (D5): "delete" deactivates the account.
	app.Delete("/api/admin/users/:id", func(req *yekonga.Request, res *yekonga.Response) {
		a := requirePerm(app, req, res, PermPlatform)
		if a == nil {
			return
		}
		id := req.Param("id")
		user := app.ModelQuery("User").SkipBeforeCommit().Where("id", id).First(nil)
		if user == nil {
			deny(res, 404, "user not found")
			return
		}
		if id == a.UserID {
			deny(res, 400, "you cannot deactivate yourself")
			return
		}
		if normalizeRole(helper.GetValueOfString(*user, "role")) == RoleSuperAdmin && activeSuperAdmins(app) <= 1 {
			deny(res, 400, "there must always be at least one super admin")
			return
		}
		app.ModelQuery("User").SkipBeforeCommit().Where("id", id).Update(datatype.DataMap{"status": "inactive", "isActive": false}, nil)
		invalidateActors()
		writeAudit(app, a, "deactivate", "User", id, "", "Deactivated "+helper.GetValueOfString(*user, "username"), nil)
		res.Json(map[string]bool{"success": true})
	})

	// ---- assignments -----------------------------------------------------

	app.Get("/api/admin/assignments", func(req *yekonga.Request, res *yekonga.Response) {
		if requirePerm(app, req, res, PermPlatform) == nil {
			return
		}
		out := []datatype.DataMap{}
		for _, as := range listAll(app, "Assignment") {
			if uid := req.Query("userId"); uid != "" && helper.GetValueOfString(as, "userId") != uid {
				continue
			}
			out = append(out, assignmentJSON(app, as))
		}
		res.Json(out)
	})

	app.Post("/api/admin/assignments", func(req *yekonga.Request, res *yekonga.Response) {
		a := requirePerm(app, req, res, PermPlatform)
		if a == nil {
			return
		}
		body := bodyMap(req)
		userId := helper.GetValueOfString(body, "userId")
		scopeType := helper.GetValueOfString(body, "scopeType")
		scopeId := helper.GetValueOfString(body, "scopeId")
		position := helper.GetValueOfString(body, "position")
		if app.ModelQuery("User").SkipBeforeCommit().Where("id", userId).First(nil) == nil {
			deny(res, 400, "user not found")
			return
		}
		switch scopeType {
		case "all":
			scopeId = ""
		case "partner":
			if app.ModelQuery("Partner").SkipBeforeCommit().Where("id", scopeId).First(nil) == nil {
				deny(res, 400, "partner not found")
				return
			}
		case "cluster":
			if app.ModelQuery("Cluster").SkipBeforeCommit().Where("id", scopeId).First(nil) == nil {
				deny(res, 400, "cluster not found")
				return
			}
		case "group":
			if app.ModelQuery("Group").SkipBeforeCommit().Where("id", scopeId).First(nil) == nil {
				deny(res, 400, "group not found")
				return
			}
			if position != "" && position != "mwenyekiti" && !officerPositions[position] {
				deny(res, 400, "unknown position")
				return
			}
		default:
			deny(res, 400, "scopeType must be all, partner, cluster or group")
			return
		}
		if scopeType != "group" {
			position = ""
		}
		for _, as := range listAll(app, "Assignment") {
			if helper.GetValueOfString(as, "userId") == userId && helper.GetValueOfString(as, "scopeType") == scopeType &&
				helper.GetValueOfString(as, "scopeId") == scopeId {
				deny(res, 400, "already assigned")
				return
			}
		}
		rec := datatype.DataMap{"userId": userId, "scopeType": scopeType, "scopeId": scopeId, "createdBy": a.UserID}
		if position != "" {
			rec["position"] = position
		}
		created := app.ModelQuery("Assignment").SkipBeforeCommit().Create(rec)
		invalidateActors()
		groupId := ""
		if scopeType == "group" {
			groupId = scopeId
		}
		writeAudit(app, a, "assign", "Assignment", helper.GetValueOfString(helper.ToDataMap(created), "id"), groupId,
			fmt.Sprintf("Assigned user to %s %s %s", scopeType, scopeName(app, scopeType, scopeId), position), rec)
		res.Json(created)
	})

	// Assignments are access configuration, not financial records: removing
	// one is allowed (and audited).
	app.Delete("/api/admin/assignments/:id", func(req *yekonga.Request, res *yekonga.Response) {
		a := requirePerm(app, req, res, PermPlatform)
		if a == nil {
			return
		}
		as := app.ModelQuery("Assignment").SkipBeforeCommit().Where("id", req.Param("id")).First(nil)
		if as == nil {
			deny(res, 404, "assignment not found")
			return
		}
		app.ModelQuery("Assignment").SkipBeforeCommit().Where("id", req.Param("id")).Delete(nil)
		invalidateActors()
		writeAudit(app, a, "unassign", "Assignment", req.Param("id"), "", "Removed assignment", assignmentJSON(app, *as))
		res.Json(map[string]bool{"success": true})
	})

	// ---- audit log -------------------------------------------------------

	app.Get("/api/admin/audit", func(req *yekonga.Request, res *yekonga.Response) {
		if requirePerm(app, req, res, PermAudit) == nil {
			return
		}
		from, to, err := reportRange(req.Query("from"), req.Query("to"))
		if err != nil {
			deny(res, 400, err.Error())
			return
		}
		limit := 300
		if n := helper.GetValueOfInt(datatype.DataMap{"n": req.Query("limit")}, "n"); n > 0 && n <= 2000 {
			limit = n
		}
		groupNames := map[string]string{}
		for _, g := range listAll(app, "Group") {
			groupNames[helper.GetValueOfString(g, "id")] = helper.GetValueOfString(g, "name")
		}
		recs := app.ModelQuery("AuditLog").SkipBeforeCommit().OrderBy("createdAt", "desc").Find(nil)
		out := []datatype.DataMap{}
		for _, r := range *recs {
			if m := req.Query("model"); m != "" && helper.GetValueOfString(r, "model") != m {
				continue
			}
			if act := req.Query("action"); act != "" && helper.GetValueOfString(r, "action") != act {
				continue
			}
			if g := req.Query("groupId"); g != "" && helper.GetValueOfString(r, "groupId") != g {
				continue
			}
			if u := req.Query("userId"); u != "" && helper.GetValueOfString(r, "userId") != u {
				continue
			}
			at := helper.GetTimestamp(r["createdAt"])
			if (!from.IsZero() && at.Before(from)) || (!to.IsZero() && at.After(to)) {
				continue
			}
			rec := datatype.DataMap{}
			for k, v := range r {
				rec[k] = v
			}
			rec["groupName"] = groupNames[helper.GetValueOfString(r, "groupId")]
			out = append(out, rec)
			if len(out) >= limit {
				break
			}
		}
		res.Json(out)
	})
}

func scopeName(app *yekonga.YekongaData, scopeType, scopeId string) string {
	model := map[string]string{"partner": "Partner", "cluster": "Cluster", "group": "Group"}[scopeType]
	if model == "" || scopeId == "" {
		return ""
	}
	if rec := app.ModelQuery(model).SkipBeforeCommit().Where("id", scopeId).First(nil); rec != nil {
		return helper.GetValueOfString(*rec, "name")
	}
	return ""
}

func assignmentJSON(app *yekonga.YekongaData, as datatype.DataMap) datatype.DataMap {
	scopeType := helper.GetValueOfString(as, "scopeType")
	scopeId := helper.GetValueOfString(as, "scopeId")
	return datatype.DataMap{
		"id":        helper.GetValueOfString(as, "id"),
		"userId":    helper.GetValueOfString(as, "userId"),
		"scopeType": scopeType,
		"scopeId":   scopeId,
		"scopeName": scopeName(app, scopeType, scopeId),
		"position":  helper.GetValueOfString(as, "position"),
	}
}

// registerOfficerRoutes: the Group Admin (Mwenyekiti) manages the group's
// officers from the mobile app. Needs requireGroupPerm from main.go.
func registerOfficerRoutes(app *yekonga.YekongaData,
	requireGroup func(*yekonga.Request, *yekonga.Response) *datatype.DataMap,
	requireGroupPerm func(*yekonga.Request, *yekonga.Response, string) (*Actor, *datatype.DataMap)) {

	// The app's "which group am I running, and as what" call.
	app.Get("/api/main/group", func(req *yekonga.Request, res *yekonga.Response) {
		g := requireGroup(req, res)
		if g == nil {
			return
		}
		a := actorFromRequest(app, req)
		gid := helper.GetValueOfString(*g, "id")
		perms := []string{}
		for _, p := range allPerms {
			if a.CanIn(p, gid) {
				perms = append(perms, p)
			}
		}
		res.Json(datatype.DataMap{
			"group":       *g,
			"position":    a.Position,
			"role":        a.Role,
			"permissions": perms,
		})
	})

	app.Get("/api/main/officers", func(req *yekonga.Request, res *yekonga.Response) {
		g := requireGroup(req, res)
		if g == nil {
			return
		}
		gid := helper.GetValueOfString(*g, "id")
		users := map[string]datatype.DataMap{}
		for _, u := range listAll(app, "User") {
			users[helper.GetValueOfString(u, "id")] = u
		}
		out := []datatype.DataMap{}
		for _, as := range listAll(app, "Assignment") {
			if helper.GetValueOfString(as, "scopeType") != "group" || helper.GetValueOfString(as, "scopeId") != gid ||
				helper.GetValueOfString(as, "position") == "" {
				continue
			}
			u := users[helper.GetValueOfString(as, "userId")]
			out = append(out, datatype.DataMap{
				"id":        helper.GetValueOfString(as, "id"),
				"userId":    helper.GetValueOfString(as, "userId"),
				"position":  helper.GetValueOfString(as, "position"),
				"firstName": helper.GetValueOfString(u, "firstName"),
				"lastName":  helper.GetValueOfString(u, "lastName"),
				"phone":     helper.GetValueOfString(u, "username"),
				"active":    helper.GetValueOfString(u, "status") != "inactive",
			})
		}
		res.Json(out)
	})

	app.Post("/api/main/officers", func(req *yekonga.Request, res *yekonga.Response) {
		a, g := requireGroupPerm(req, res, PermGroupOfficers)
		if g == nil {
			return
		}
		gid := helper.GetValueOfString(*g, "id")
		body := bodyMap(req)
		phone := helper.GetValueOfString(body, "phone")
		position := helper.GetValueOfString(body, "position")
		if last9(phone) == "" || !officerPositions[position] {
			deny(res, 400, "a phone number and a position (katibu, mweka_hazina or committee) are required")
			return
		}
		u, _ := findOrCreateUser(app, phone, strings.TrimSpace(helper.GetValueOfString(body, "firstName")), strings.TrimSpace(helper.GetValueOfString(body, "lastName")), RoleGroupOfficer)
		if u == nil {
			deny(res, 500, "could not create the officer's account")
			return
		}
		uid := helper.GetValueOfString(u, "id")
		if uid == a.UserID {
			deny(res, 400, "you are already the group admin")
			return
		}
		// Never downgrade someone who already has a platform role.
		if normalizeRole(helper.GetValueOfString(u, "role")) == "" {
			app.ModelQuery("User").SkipBeforeCommit().Where("id", uid).Update(datatype.DataMap{"role": RoleGroupOfficer}, nil)
		}
		for _, as := range listAll(app, "Assignment") {
			if helper.GetValueOfString(as, "userId") == uid && helper.GetValueOfString(as, "scopeType") == "group" &&
				helper.GetValueOfString(as, "position") != "" {
				deny(res, 400, "this person is already an officer of a group")
				return
			}
		}
		created := app.ModelQuery("Assignment").SkipBeforeCommit().Create(datatype.DataMap{
			"userId": uid, "scopeType": "group", "scopeId": gid, "position": position, "createdBy": a.UserID,
		})
		invalidateActors()
		writeAudit(app, a, "assign", "Assignment", helper.GetValueOfString(helper.ToDataMap(created), "id"), gid,
			fmt.Sprintf("Made %s %s", phone, position), nil)
		res.Json(created)
	})

	app.Delete("/api/main/officers/:id", func(req *yekonga.Request, res *yekonga.Response) {
		a, g := requireGroupPerm(req, res, PermGroupOfficers)
		if g == nil {
			return
		}
		gid := helper.GetValueOfString(*g, "id")
		as := app.ModelQuery("Assignment").SkipBeforeCommit().Where("id", req.Param("id")).First(nil)
		if as == nil || helper.GetValueOfString(*as, "scopeId") != gid || !officerPositions[helper.GetValueOfString(*as, "position")] {
			deny(res, 404, "officer not found in your group")
			return
		}
		app.ModelQuery("Assignment").SkipBeforeCommit().Where("id", req.Param("id")).Delete(nil)
		invalidateActors()
		writeAudit(app, a, "unassign", "Assignment", req.Param("id"), gid, "Removed officer "+helper.GetValueOfString(*as, "position"), nil)
		res.Json(map[string]bool{"success": true})
	})
}
