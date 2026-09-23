package main

import (
	"fmt"
	"strings"

	"github.com/robertkonga/yekonga-server-go/datatype"
	"github.com/robertkonga/yekonga-server-go/helper"
	"github.com/robertkonga/yekonga-server-go/yekonga"
)

// Organisation structure (TODO.md §4a / D7): Partner → Cluster → Group. A group
// belongs to exactly one cluster, a cluster to exactly one partner, so roll-up
// reports never count a group twice. Partners and clusters are deactivated,
// never deleted.

var partnerFields = []string{"name", "type", "contactPerson", "phone", "email", "regions", "status"}
var clusterFields = []string{"name", "partnerId", "region", "district", "ward", "status"}

func pick(body datatype.DataMap, fields []string) datatype.DataMap {
	out := datatype.DataMap{}
	for _, f := range fields {
		if v, ok := body[f]; ok {
			if s, isStr := v.(string); isStr {
				v = strings.TrimSpace(s)
			}
			out[f] = v
		}
	}
	return out
}

// groupHasMoney reports whether any financial record points at the group.
func groupHasMoney(app *yekonga.YekongaData, groupId string) bool {
	for _, model := range []string{"Transaction", "Loan", "Fine", "GovernmentLoan"} {
		if app.ModelQuery(model).SkipBeforeCommit().Where("groupId", groupId).Count(nil) > 0 {
			return true
		}
	}
	return false
}

func memberHasMoney(app *yekonga.YekongaData, memberId string) bool {
	for _, model := range []string{"Transaction", "Loan", "Fine"} {
		if app.ModelQuery(model).SkipBeforeCommit().Where("memberId", memberId).Count(nil) > 0 {
			return true
		}
	}
	return false
}

func registerStructureRoutes(app *yekonga.YekongaData) {
	groupCounts := func() (map[string]int, map[string]int, map[string]string) {
		groupsPerCluster := map[string]int{}
		for _, g := range listAll(app, "Group") {
			groupsPerCluster[helper.GetValueOfString(g, "clusterId")]++
		}
		clustersPerPartner := map[string]int{}
		clusterPartner := map[string]string{}
		for _, c := range listAll(app, "Cluster") {
			pid := helper.GetValueOfString(c, "partnerId")
			clustersPerPartner[pid]++
			clusterPartner[helper.GetValueOfString(c, "id")] = pid
		}
		return groupsPerCluster, clustersPerPartner, clusterPartner
	}

	// ---- partners ---------------------------------------------------------

	app.Get("/api/admin/partners", func(req *yekonga.Request, res *yekonga.Response) {
		a := requirePerm(app, req, res, PermStructureView)
		if a == nil {
			return
		}
		groupsPerCluster, clustersPerPartner, clusterPartner := groupCounts()
		groupsPerPartner := map[string]int{}
		for cid, n := range groupsPerCluster {
			groupsPerPartner[clusterPartner[cid]] += n
		}
		out := []datatype.DataMap{}
		for _, p := range listAll(app, "Partner") {
			id := helper.GetValueOfString(p, "id")
			if !a.SeesPartner(id) {
				continue
			}
			rec := datatype.DataMap{}
			for k, v := range p {
				rec[k] = v
			}
			rec["clusterCount"] = clustersPerPartner[id]
			rec["groupCount"] = groupsPerPartner[id]
			out = append(out, rec)
		}
		res.Json(out)
	})

	app.Post("/api/admin/partners", func(req *yekonga.Request, res *yekonga.Response) {
		a := requirePerm(app, req, res, PermPlatform)
		if a == nil {
			return
		}
		rec := pick(bodyMap(req), partnerFields)
		if helper.GetValueOfString(rec, "name") == "" {
			deny(res, 400, "a partner name is required")
			return
		}
		rec["status"] = "active"
		rec["createdBy"] = a.UserID
		created := app.ModelQuery("Partner").SkipBeforeCommit().Create(rec)
		if err, ok := created.(error); ok || created == nil {
			deny(res, 500, fmt.Sprint("could not create partner ", err))
			return
		}
		writeAudit(app, a, "create", "Partner", helper.GetValueOfString(helper.ToDataMap(created), "id"), "", "Created partner "+helper.GetValueOfString(rec, "name"), rec)
		res.Json(created)
	})

	app.Post("/api/admin/partners/:id", func(req *yekonga.Request, res *yekonga.Response) {
		a := requirePerm(app, req, res, PermPlatform)
		if a == nil {
			return
		}
		p := app.ModelQuery("Partner").SkipBeforeCommit().Where("id", req.Param("id")).First(nil)
		if p == nil {
			deny(res, 404, "partner not found")
			return
		}
		changes := pick(bodyMap(req), partnerFields)
		if st, ok := changes["status"]; ok && st != "active" && st != "inactive" {
			deny(res, 400, "status must be active or inactive")
			return
		}
		if len(changes) == 0 {
			deny(res, 400, "no changes provided")
			return
		}
		app.ModelQuery("Partner").SkipBeforeCommit().Where("id", req.Param("id")).Update(changes, nil)
		invalidateActors()
		writeAudit(app, a, statusAction(changes, "update"), "Partner", req.Param("id"), "", "Edited partner "+helper.GetValueOfString(*p, "name"), changes)
		res.Json(map[string]bool{"success": true})
	})

	// ---- clusters ---------------------------------------------------------

	app.Get("/api/admin/clusters", func(req *yekonga.Request, res *yekonga.Response) {
		a := requirePerm(app, req, res, PermStructureView)
		if a == nil {
			return
		}
		groupsPerCluster, _, _ := groupCounts()
		partnerNames := map[string]string{}
		for _, p := range listAll(app, "Partner") {
			partnerNames[helper.GetValueOfString(p, "id")] = helper.GetValueOfString(p, "name")
		}
		out := []datatype.DataMap{}
		for _, c := range listAll(app, "Cluster") {
			id := helper.GetValueOfString(c, "id")
			if !a.SeesCluster(id) {
				continue
			}
			if pid := req.Query("partnerId"); pid != "" && helper.GetValueOfString(c, "partnerId") != pid {
				continue
			}
			rec := datatype.DataMap{}
			for k, v := range c {
				rec[k] = v
			}
			rec["partnerName"] = partnerNames[helper.GetValueOfString(c, "partnerId")]
			rec["groupCount"] = groupsPerCluster[id]
			out = append(out, rec)
		}
		res.Json(out)
	})

	app.Post("/api/admin/clusters", func(req *yekonga.Request, res *yekonga.Response) {
		a := requirePerm(app, req, res, PermPlatform)
		if a == nil {
			return
		}
		rec := pick(bodyMap(req), clusterFields)
		if helper.GetValueOfString(rec, "name") == "" {
			deny(res, 400, "a cluster name is required")
			return
		}
		if app.ModelQuery("Partner").SkipBeforeCommit().Where("id", helper.GetValueOfString(rec, "partnerId")).First(nil) == nil {
			deny(res, 400, "choose the partner this cluster belongs to")
			return
		}
		rec["status"] = "active"
		rec["createdBy"] = a.UserID
		created := app.ModelQuery("Cluster").SkipBeforeCommit().Create(rec)
		if err, ok := created.(error); ok || created == nil {
			deny(res, 500, fmt.Sprint("could not create cluster ", err))
			return
		}
		invalidateActors()
		writeAudit(app, a, "create", "Cluster", helper.GetValueOfString(helper.ToDataMap(created), "id"), "", "Created cluster "+helper.GetValueOfString(rec, "name"), rec)
		res.Json(created)
	})

	app.Post("/api/admin/clusters/:id", func(req *yekonga.Request, res *yekonga.Response) {
		a := requirePerm(app, req, res, PermPlatform)
		if a == nil {
			return
		}
		c := app.ModelQuery("Cluster").SkipBeforeCommit().Where("id", req.Param("id")).First(nil)
		if c == nil {
			deny(res, 404, "cluster not found")
			return
		}
		changes := pick(bodyMap(req), clusterFields)
		if st, ok := changes["status"]; ok && st != "active" && st != "inactive" {
			deny(res, 400, "status must be active or inactive")
			return
		}
		if pid, ok := changes["partnerId"]; ok && app.ModelQuery("Partner").SkipBeforeCommit().Where("id", fmt.Sprint(pid)).First(nil) == nil {
			deny(res, 400, "partner not found")
			return
		}
		if len(changes) == 0 {
			deny(res, 400, "no changes provided")
			return
		}
		app.ModelQuery("Cluster").SkipBeforeCommit().Where("id", req.Param("id")).Update(changes, nil)
		invalidateActors()
		writeAudit(app, a, statusAction(changes, "update"), "Cluster", req.Param("id"), "", "Edited cluster "+helper.GetValueOfString(*c, "name"), changes)
		res.Json(map[string]bool{"success": true})
	})

	// ---- groups: move between clusters, safe delete ------------------------

	app.Post("/api/admin/groups/:id/cluster", func(req *yekonga.Request, res *yekonga.Response) {
		a := requirePerm(app, req, res, PermGroupCreate)
		if a == nil {
			return
		}
		gid := req.Param("id")
		g := app.ModelQuery("Group").SkipBeforeCommit().Where("id", gid).First(nil)
		if g == nil || !a.Sees(gid) {
			deny(res, 404, "group not found")
			return
		}
		clusterId := helper.GetValueOfString(bodyMap(req), "clusterId")
		if app.ModelQuery("Cluster").SkipBeforeCommit().Where("id", clusterId).First(nil) == nil || !a.SeesCluster(clusterId) {
			deny(res, 400, "choose a cluster you manage")
			return
		}
		app.ModelQuery("Group").SkipBeforeCommit().Where("id", gid).Update(datatype.DataMap{"clusterId": clusterId}, nil)
		invalidateActors()
		writeAudit(app, a, "update", "Group", gid, gid, fmt.Sprintf("Moved %s to cluster %s", helper.GetValueOfString(*g, "name"), scopeName(app, "cluster", clusterId)), nil)
		res.Json(map[string]bool{"success": true})
	})

	// A group can only be deleted while nothing financial was ever recorded
	// for it (D5) — otherwise it is closed instead (status "Closed").
	app.Delete("/api/admin/groups/:id", func(req *yekonga.Request, res *yekonga.Response) {
		a := requireActor(app, req, res)
		if a == nil {
			return
		}
		gid := req.Param("id")
		g := app.ModelQuery("Group").SkipBeforeCommit().Where("id", gid).First(nil)
		if g == nil || !a.Sees(gid) {
			deny(res, 404, "group not found")
			return
		}
		if !a.CanIn(PermGroupSettings, gid) {
			deny(res, 403, "your role does not allow deleting this group")
			return
		}
		if groupHasMoney(app, gid) {
			deny(res, 400, "this group has financial records and cannot be deleted — set its status to Closed instead")
			return
		}
		for _, model := range []string{"MeetingAttendance", "Meeting", "Announcement", "Member"} {
			app.ModelQuery(model).SkipBeforeCommit().Where("groupId", gid).Delete(nil)
		}
		for _, as := range listAll(app, "Assignment") {
			if helper.GetValueOfString(as, "scopeType") == "group" && helper.GetValueOfString(as, "scopeId") == gid {
				app.ModelQuery("Assignment").SkipBeforeCommit().Where("id", helper.GetValueOfString(as, "id")).Delete(nil)
			}
		}
		app.ModelQuery("Group").SkipBeforeCommit().Where("id", gid).Delete(nil)
		invalidateActors()
		writeAudit(app, a, "delete", "Group", gid, gid, "Deleted group "+helper.GetValueOfString(*g, "name")+" (no financial records)", *g)
		res.Json(map[string]bool{"success": true})
	})

	// Same rule for a member added by mistake: deletable only with no money
	// history, by the Group Admin or above; otherwise deactivate.
	app.Delete("/api/main/members/:id", func(req *yekonga.Request, res *yekonga.Response) {
		a := requireActor(app, req, res)
		if a == nil {
			return
		}
		m := app.ModelQuery("Member").SkipBeforeCommit().Where("id", req.Param("id")).First(nil)
		if m == nil {
			deny(res, 404, "member not found")
			return
		}
		gid := helper.GetValueOfString(*m, "groupId")
		if !a.CanIn(PermGroupSettings, gid) {
			deny(res, 403, "only the group admin can delete a member")
			return
		}
		if memberHasMoney(app, req.Param("id")) {
			deny(res, 400, "this member has financial records and cannot be deleted — deactivate them instead")
			return
		}
		app.ModelQuery("MeetingAttendance").SkipBeforeCommit().Where("memberId", req.Param("id")).Delete(nil)
		app.ModelQuery("Member").SkipBeforeCommit().Where("id", req.Param("id")).Delete(nil)
		recalcMemberCounts(app, gid)
		writeAudit(app, a, "delete", "Member", req.Param("id"), gid, "Deleted member "+memberFullName(*m)+" (no financial records)", *m)
		res.Json(map[string]bool{"success": true})
	})
}

func statusAction(changes datatype.DataMap, def string) string {
	switch changes["status"] {
	case "inactive":
		return "deactivate"
	case "active":
		return "reactivate"
	}
	return def
}

// recalcMemberCounts refreshes a group's denormalized member counters.
func recalcMemberCounts(app *yekonga.YekongaData, groupId string) {
	if groupId == "" {
		return
	}
	var total, female, male int
	for _, m := range *app.ModelQuery("Member").SkipBeforeCommit().Where("groupId", groupId).Find(nil) {
		total++
		switch helper.GetValueOfString(m, "gender") {
		case "Female":
			female++
		case "Male":
			male++
		}
	}
	app.ModelQuery("Group").SkipBeforeCommit().Where("id", groupId).Update(datatype.DataMap{
		"memberCount": total, "femaleMembers": female, "maleMembers": male,
	}, nil)
}
