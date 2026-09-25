package main

import (
	"fmt"

	"github.com/robertkonga/yekonga-server-go/datatype"
	"github.com/robertkonga/yekonga-server-go/helper"
	"github.com/robertkonga/yekonga-server-go/yekonga"
)

// ---------------------------------------------------------------------------
// GraphQL guard. The framework auto-generates GraphQL CRUD for every model in
// database.json, and config.json lets any logged-in user use all of it — so
// without this, anyone could read any group's data or delete a transaction.
//
// The framework's "All" triggers see every ModelQuery. A nil RequestContext
// means an internal server call (REST handlers here always use
// SkipBeforeCommit, which skips Before* triggers anyway), so those pass
// untouched; everything with a RequestContext came from an API client and is
// scoped / checked against the caller's Actor.
//
// Returning false from a Before* trigger makes the framework drop the
// operation (find → empty list, create/update/delete → no-op).
// ---------------------------------------------------------------------------

// scopedModels: model → field holding the group id (or "id" for Group itself).
var scopedModels = map[string]string{
	"Group":                   "id",
	"Member":                  "groupId",
	"Meeting":                 "groupId",
	"MeetingAttendance":       "groupId",
	"Transaction":             "groupId",
	"Loan":                    "groupId",
	"Fine":                    "groupId",
	"Announcement":            "groupId",
	"SmsLog":                  "groupId",
	"GovernmentLoan":          "groupId",
	"GovernmentLoanRepayment": "groupId",
}

// hiddenModels are never readable over GraphQL except by a super admin (OTP
// codes, access rules, audit trail, settings). The web reads them through
// dedicated REST routes instead.
var hiddenModels = map[string]bool{
	"MemberVerification": true,
	"Assignment":         true,
	"AccessProfile":      true,
	"AuditLog":           true,
	"PlatformSetting":    true,
	"SmsTemplate":        true,
}

// businessModels are ours (not framework built-ins). GraphQL may never delete
// them (TODO.md D5/N1-N6) and may only write the few noted in guardWrite.
var businessModels = map[string]bool{
	"Partner": true, "Cluster": true, "Group": true, "Member": true, "MemberVerification": true,
	"Meeting": true, "MeetingAttendance": true, "Transaction": true, "Loan": true, "Fine": true,
	"Announcement": true, "SmsLog": true, "SmsTemplate": true, "PlatformSetting": true,
	"Assignment": true, "AccessProfile": true, "AuditLog": true,
	"GovernmentLoan": true, "GovernmentLoanRepayment": true,
}

// groupReadonlyFields are running totals the server maintains; a client may
// never set them directly.
var groupReadonlyFields = []string{
	"totalSavings", "totalShares", "totalSocialFund", "totalLoans", "totalFines", "totalExpenses",
	"memberCount", "femaleMembers", "maleMembers", "createdBy",
}

var meetingEditableFields = map[string]bool{
	"groupId": true, "title": true, "date": true, "time": true, "location": true, "notes": true, "status": true,
}

func registerGuards(app *yekonga.YekongaData) {
	app.BeforeFindAll(func(model *yekonga.DataModel, ctx *yekonga.RequestContext, q *yekonga.QueryContext) (interface{}, error) {
		if ctx == nil || model == nil {
			return nil, nil
		}
		name := model.Name
		_, scoped := scopedModels[name]
		if !scoped && !hiddenModels[name] && name != "Partner" && name != "Cluster" {
			return nil, nil
		}
		a := actorFromContext(app, ctx)
		if a == nil {
			return false, nil // guests read nothing of ours
		}
		if hiddenModels[name] {
			if a.Can(PermPlatform) {
				return nil, nil
			}
			return false, nil
		}
		if a.All {
			return nil, nil
		}
		switch name {
		case "Partner":
			return inFilter("id", sortedKeys(a.PartnerIDs))
		case "Cluster":
			return inFilter("id", sortedKeys(a.ClusterIDs))
		}
		return inFilter(scopedModels[name], a.VisibleGroupIDs())
	})

	app.BeforeCreateAll(func(model *yekonga.DataModel, ctx *yekonga.RequestContext, q *yekonga.QueryContext) (interface{}, error) {
		return guardWrite(app, "create", model, ctx, q)
	})
	app.BeforeUpdateAll(func(model *yekonga.DataModel, ctx *yekonga.RequestContext, q *yekonga.QueryContext) (interface{}, error) {
		return guardWrite(app, "update", model, ctx, q)
	})
	app.BeforeDeleteAll(func(model *yekonga.DataModel, ctx *yekonga.RequestContext, q *yekonga.QueryContext) (interface{}, error) {
		if ctx == nil || model == nil || !businessModels[model.Name] {
			return nil, nil
		}
		// Deletes of our data only ever happen through REST routes that check
		// "no financial activity" first (e.g. DELETE /api/admin/groups/:id).
		return false, nil
	})

	// Audit what GraphQL writes did get through (REST routes write their own,
	// richer entries).
	audit := func(action string) yekonga.TriggerAllFunction {
		return func(model *yekonga.DataModel, ctx *yekonga.RequestContext, q *yekonga.QueryContext) (interface{}, error) {
			if ctx == nil || model == nil || !businessModels[model.Name] || model.Name == "AuditLog" {
				return nil, nil
			}
			a := actorFromContext(app, ctx)
			if a == nil {
				return nil, nil
			}
			data := helper.ToDataMap(q.Data)
			recordId := helper.GetValueOfString(data, "id")
			groupId := helper.GetValueOfString(data, "groupId")
			if model.Name == "Group" {
				groupId = recordId
			}
			writeAudit(app, a, action, model.Name, recordId, groupId, fmt.Sprintf("%s %s", action, model.Name), helper.ToDataMap(q.Input))
			if model.Name == "Group" {
				invalidateActors()
			}
			return nil, nil
		}
	}
	app.AfterCreateAll(audit("create"))
	app.AfterUpdateAll(audit("update"))
}

func inFilter(field string, ids []string) (interface{}, error) {
	if len(ids) == 0 {
		return false, nil
	}
	list := make([]interface{}, 0, len(ids))
	for _, id := range ids {
		list = append(list, id)
	}
	return datatype.DataMap{field: map[string]interface{}{"in": list}}, nil
}

// guardWrite decides create/update over GraphQL. Only two models are
// writable there — Group (the web's create/edit group form) and Meeting (the
// app's close-meeting call); everything else has a checked REST route.
func guardWrite(app *yekonga.YekongaData, action string, model *yekonga.DataModel, ctx *yekonga.RequestContext, q *yekonga.QueryContext) (interface{}, error) {
	if ctx == nil || model == nil || !businessModels[model.Name] {
		return nil, nil
	}
	a := actorFromContext(app, ctx)
	if a == nil {
		return false, nil
	}
	input := helper.ToDataMap(q.Input)

	switch model.Name {
	case "Group":
		requestedOwner := helper.GetValueOfString(input, "createdBy")
		for _, f := range groupReadonlyFields {
			delete(input, f)
		}
		// The constitution changes only through the checked rules route
		// (group_rules.go); a new group starts from the schema defaults.
		for _, f := range ruleFields {
			delete(input, f)
		}
		// Nothing writable left: refuse. (The framework treats an EMPTY map
		// returned from a trigger as "no change" and would write the original
		// input — which is how read-only fields used to slip through.)
		if len(input) == 0 {
			return false, nil
		}
		if clusterId := helper.GetValueOfString(input, "clusterId"); clusterId != "" && !a.SeesCluster(clusterId) {
			return false, nil
		}
		if action == "create" {
			if !a.Can(PermGroupCreate) {
				return false, nil
			}
			if helper.GetValueOfString(input, "clusterId") == "" {
				if !a.All {
					return false, nil // non-global staff must place the group in one of their clusters
				}
				input["clusterId"] = ensureDefaultCluster(app) // every group sits in a cluster (roll-ups)
			}
			// createdBy links a group to the account that runs it. The web's
			// "create a group for this Group Admin" flow sends that admin's id;
			// otherwise the creator is linked (the existing behaviour).
			input["createdBy"] = a.UserID
			if requestedOwner != "" && app.ModelQuery("User").SkipBeforeCommit().Where("id", requestedOwner).First(nil) != nil {
				input["createdBy"] = requestedOwner
			}
			invalidateActors()
			return input, nil
		}
		for _, g := range guardTargets(app, "Group", q) {
			if !a.CanIn(PermGroupSettings, helper.GetValueOfString(g, "id")) {
				return false, nil
			}
		}
		if _, moving := input["clusterId"]; moving && !a.Can(PermGroupCreate) {
			return false, nil
		}
		invalidateActors()
		return input, nil

	case "Meeting":
		if action != "update" {
			return false, nil
		}
		for k := range input {
			if !meetingEditableFields[k] {
				delete(input, k)
			}
		}
		if len(input) == 0 {
			return false, nil // see the Group case: an empty result would write the original input
		}
		for _, m := range guardTargets(app, "Meeting", q) {
			if !a.CanIn(PermGroupOperate, helper.GetValueOfString(m, "groupId")) {
				return false, nil
			}
			if helper.GetValueOfString(m, "status") == "completed" {
				return false, nil // closed meetings are locked (N3)
			}
		}
		return input, nil
	}
	return false, nil
}

// guardTargets loads the records an update's where-clause points at. An
// update with no where-clause would hit everything — treat as none allowed.
func guardTargets(app *yekonga.YekongaData, model string, q *yekonga.QueryContext) []datatype.DataMap {
	if q == nil || q.Filters == nil || len(*q.Filters) == 0 {
		return []datatype.DataMap{{"id": "", "groupId": ""}}
	}
	recs := app.ModelQuery(model).SkipBeforeCommit().WhereAll(*q.Filters).Find(nil)
	if recs == nil || len(*recs) == 0 {
		return nil
	}
	return *recs
}
