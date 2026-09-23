package main

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/robertkonga/yekonga-server-go/datatype"
	"github.com/robertkonga/yekonga-server-go/helper"
	"github.com/robertkonga/yekonga-server-go/yekonga"
)

// ---------------------------------------------------------------------------
// User levels (see "Pesa Box User Levels.pdf" and TODO.md §4). User.role holds
// one of these; the order is the PDF's level 1 → 7.
// ---------------------------------------------------------------------------

const (
	RoleSuperAdmin   = "super_admin"
	RoleStaff        = "staff" // "PesaBox Staff" — internal operations team
	RolePartner      = "partner_user"
	RoleCluster      = "cluster_manager"
	RoleGroupAdmin   = "group_admin" // the group's Mwenyekiti
	RoleGroupOfficer = "group_officer"
	RoleMember       = "group_member"
)

var allRoles = []string{RoleSuperAdmin, RoleStaff, RolePartner, RoleCluster, RoleGroupAdmin, RoleGroupOfficer, RoleMember}

// normalizeRole maps the roles used before the 7-level model onto it. "user"
// (the framework's default for a fresh OTP sign-up) and anything unknown
// become "" — a login with no platform access.
func normalizeRole(r string) string {
	switch r {
	case "admin", "1", RoleSuperAdmin:
		return RoleSuperAdmin
	case "support_admin", RoleStaff:
		return RoleStaff
	}
	for _, known := range allRoles {
		if r == known {
			return r
		}
	}
	return ""
}

func validRole(r string) bool {
	for _, known := range allRoles {
		if r == known {
			return true
		}
	}
	return false
}

// Staff get one of three permission presets instead of per-permission
// toggles (TODO.md D8). Stored on AccessProfile.preset.
const (
	PresetViewer     = "viewer"
	PresetSupport    = "support"
	PresetOperations = "operations"
)

func validPreset(p string) bool {
	return p == PresetViewer || p == PresetSupport || p == PresetOperations
}

// Permissions. "Platform" ones apply across everything in the actor's scope;
// the group ones apply either across scope (staff/super admin) or only in the
// actor's home group (group admin/officer) — see Actor.CanIn.
const (
	PermPlatform      = "platform.manage" // users, roles, partners, clusters, staff, settings, SMS templates
	PermAudit         = "audit.view"
	PermDashboard     = "dashboard.view" // may use the web dashboard at all
	PermStructureView = "structure.view" // see partners / clusters
	PermReports       = "reports.view"
	PermSms           = "sms.view"
	PermGroupCreate   = "groups.create" // onboard new groups
	PermGroupSettings = "group.settings"
	PermGroupOfficers = "group.officers"
	PermGroupOperate  = "group.operate" // members, meetings, attendance, announcements
	PermFinanceWrite  = "finance.write" // transactions, loans, fines, expenses, government loans, reversals
)

var allPerms = []string{PermPlatform, PermAudit, PermDashboard, PermStructureView, PermReports, PermSms,
	PermGroupCreate, PermGroupSettings, PermGroupOfficers, PermGroupOperate, PermFinanceWrite}

func permSet(ps ...string) map[string]bool {
	out := map[string]bool{}
	for _, p := range ps {
		out[p] = true
	}
	return out
}

// platformPerms is what a role may do across its scope (partners / clusters /
// groups it is assigned to, or everything for "all").
func platformPerms(role, preset string) map[string]bool {
	switch role {
	case RoleSuperAdmin:
		return permSet(allPerms...)
	case RoleStaff:
		base := []string{PermDashboard, PermStructureView, PermReports, PermSms}
		switch preset {
		case PresetSupport:
			base = append(base, PermGroupCreate, PermGroupOperate)
		case PresetOperations:
			base = append(base, PermGroupCreate, PermGroupOperate, PermGroupSettings, PermGroupOfficers, PermFinanceWrite)
		}
		return permSet(base...)
	case RolePartner:
		// Monitoring only: the PDF restricts partners from editing group money.
		return permSet(PermDashboard, PermStructureView, PermReports)
	case RoleCluster:
		return permSet(PermDashboard, PermStructureView, PermReports, PermGroupCreate)
	}
	return map[string]bool{}
}

var (
	groupAdminPerms   = permSet(PermReports, PermSms, PermGroupSettings, PermGroupOfficers, PermGroupOperate, PermFinanceWrite)
	groupOfficerPerms = permSet(PermReports, PermGroupOperate, PermFinanceWrite)
)

// ---------------------------------------------------------------------------
// Actor: who is calling and what they can reach. Resolved from the User record
// + Assignments on every request (cached briefly), never from the JWT — role
// and assignments change without the user logging in again.
// ---------------------------------------------------------------------------

type Actor struct {
	UserID string
	Name   string
	Phone  string
	Role   string // normalized; "" = no platform role
	Preset string // staff only

	Perms map[string]bool // platform-level, apply across scope

	All        bool            // sees every partner / cluster / group
	PartnerIDs map[string]bool // assigned partners
	ClusterIDs map[string]bool // assigned clusters + clusters of assigned partners
	GroupIDs   map[string]bool // every group reachable through the above

	HomeGroupID string          // the group this user runs day-to-day, if any
	Position    string          // mwenyekiti / katibu / mweka_hazina / committee
	HomePerms   map[string]bool // group-level perms, only inside HomeGroupID
}

func (a *Actor) Can(perm string) bool { return a != nil && a.Perms[perm] }

// Sees reports whether the actor may read data of group id.
func (a *Actor) Sees(groupId string) bool {
	if a == nil || groupId == "" {
		return false
	}
	return a.All || a.GroupIDs[groupId] || groupId == a.HomeGroupID
}

// CanIn reports whether the actor may perform perm inside group id.
func (a *Actor) CanIn(perm, groupId string) bool {
	if a == nil || groupId == "" {
		return false
	}
	if a.Perms[perm] && (a.All || a.GroupIDs[groupId]) {
		return true
	}
	return groupId == a.HomeGroupID && a.HomePerms[perm]
}

// SeesCluster / SeesPartner are for the structure screens.
func (a *Actor) SeesCluster(id string) bool { return a != nil && (a.All || a.ClusterIDs[id]) }
func (a *Actor) SeesPartner(id string) bool { return a != nil && (a.All || a.PartnerIDs[id]) }

// VisibleGroupIDs is every group the actor may read, sorted (nil when All).
func (a *Actor) VisibleGroupIDs() []string {
	if a == nil || a.All {
		return nil
	}
	set := map[string]bool{}
	for id := range a.GroupIDs {
		set[id] = true
	}
	if a.HomeGroupID != "" {
		set[a.HomeGroupID] = true
	}
	return sortedKeys(set)
}

// PermList is the union of platform + home-group permissions, for the UI.
func (a *Actor) PermList() []string {
	set := map[string]bool{}
	for p := range a.Perms {
		set[p] = true
	}
	for p := range a.HomePerms {
		set[p] = true
	}
	return sortedKeys(set)
}

func sortedKeys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// ---------------------------------------------------------------------------
// Resolution + cache
// ---------------------------------------------------------------------------

var (
	actorCacheMu sync.Mutex
	actorCache   = map[string]actorCacheEntry{}
)

type actorCacheEntry struct {
	actor   *Actor
	expires time.Time
}

const actorCacheTTL = 10 * time.Second

// invalidateActors drops every cached Actor — call after changing roles,
// presets, assignments, clusters or a group's cluster/admin.
func invalidateActors() {
	actorCacheMu.Lock()
	actorCache = map[string]actorCacheEntry{}
	actorCacheMu.Unlock()
}

// actorFor resolves the Actor for a logged-in user id (nil if the user is gone
// or deactivated).
func actorFor(app *yekonga.YekongaData, userId, username string) *Actor {
	if userId == "" {
		return nil
	}
	actorCacheMu.Lock()
	if e, ok := actorCache[userId]; ok && time.Now().Before(e.expires) {
		actorCacheMu.Unlock()
		return e.actor
	}
	actorCacheMu.Unlock()

	a := resolveActor(app, userId, username)

	actorCacheMu.Lock()
	actorCache[userId] = actorCacheEntry{actor: a, expires: time.Now().Add(actorCacheTTL)}
	actorCacheMu.Unlock()
	return a
}

func actorFromRequest(app *yekonga.YekongaData, req *yekonga.Request) *Actor {
	auth := req.Auth()
	if auth == nil {
		return nil
	}
	return actorFor(app, auth.ID, auth.Username)
}

func actorFromContext(app *yekonga.YekongaData, ctx *yekonga.RequestContext) *Actor {
	if ctx == nil {
		return nil
	}
	if ctx.Auth != nil && ctx.Auth.ID != "" {
		return actorFor(app, ctx.Auth.ID, ctx.Auth.Username)
	}
	if ctx.Request != nil {
		return actorFromRequest(app, ctx.Request)
	}
	return nil
}

func resolveActor(app *yekonga.YekongaData, userId, username string) *Actor {
	user := app.ModelQuery("User").SkipBeforeCommit().Where("id", userId).First(nil)
	if user == nil {
		return nil
	}
	u := *user
	if st := helper.GetValueOfString(u, "status"); st == "inactive" {
		return nil
	}
	if username == "" {
		username = helper.GetValueOfString(u, "username")
	}

	a := &Actor{
		UserID:     userId,
		Name:       strings.TrimSpace(helper.GetValueOfString(u, "firstName") + " " + helper.GetValueOfString(u, "lastName")),
		Phone:      helper.GetValueOfString(u, "phone"),
		Role:       normalizeRole(helper.GetValueOfString(u, "role")),
		PartnerIDs: map[string]bool{},
		ClusterIDs: map[string]bool{},
		GroupIDs:   map[string]bool{},
		HomePerms:  map[string]bool{},
	}
	if a.Phone == "" {
		a.Phone = username
	}
	if a.Role == RoleStaff {
		a.Preset = PresetViewer
		if p := app.ModelQuery("AccessProfile").SkipBeforeCommit().Where("userId", userId).First(nil); p != nil {
			if v := helper.GetValueOfString(*p, "preset"); validPreset(v) {
				a.Preset = v
			}
		}
	}
	a.Perms = platformPerms(a.Role, a.Preset)
	if a.Role == RoleSuperAdmin {
		a.All = true
	}

	groups := listAll(app, "Group")
	clusters := listAll(app, "Cluster")

	assignments := app.ModelQuery("Assignment").SkipBeforeCommit().Where("userId", userId).Find(nil)
	homeAssignment := ""
	homePosition := ""
	if assignments != nil {
		for _, as := range *assignments {
			scopeId := helper.GetValueOfString(as, "scopeId")
			switch helper.GetValueOfString(as, "scopeType") {
			case "all":
				if a.Role == RoleSuperAdmin || a.Role == RoleStaff {
					a.All = true
				}
			case "partner":
				a.PartnerIDs[scopeId] = true
			case "cluster":
				a.ClusterIDs[scopeId] = true
			case "group":
				if pos := helper.GetValueOfString(as, "position"); pos != "" {
					// A position makes this the user's home group (they run it);
					// the first one wins if someone was assigned twice.
					if homeAssignment == "" {
						homeAssignment, homePosition = scopeId, pos
					}
				} else {
					a.GroupIDs[scopeId] = true
				}
			}
		}
	}

	// Expand partner → clusters → groups.
	for _, c := range clusters {
		if a.PartnerIDs[helper.GetValueOfString(c, "partnerId")] {
			a.ClusterIDs[helper.GetValueOfString(c, "id")] = true
		}
	}
	for _, g := range groups {
		if a.ClusterIDs[helper.GetValueOfString(g, "clusterId")] {
			a.GroupIDs[helper.GetValueOfString(g, "id")] = true
		}
	}

	// Home group: an explicit group-position assignment, otherwise the group
	// this user created or whose admin phone is theirs (how group admins have
	// always been linked to their group — see adminGroup in main.go).
	if homeAssignment != "" {
		a.HomeGroupID = homeAssignment
		a.Position = homePosition
		if homePosition == "mwenyekiti" {
			a.HomePerms = groupAdminPerms
		} else {
			a.HomePerms = groupOfficerPerms
		}
	} else if g := ownedGroup(groups, userId, username); g != "" {
		a.HomeGroupID = g
		a.Position = "mwenyekiti"
		a.HomePerms = groupAdminPerms
	}
	// A plain group_officer role with no group assignment runs nothing.
	return a
}

func listAll(app *yekonga.YekongaData, model string) []datatype.DataMap {
	recs := app.ModelQuery(model).SkipBeforeCommit().Find(nil)
	if recs == nil {
		return nil
	}
	return *recs
}

// last9 keeps only digits and truncates to the last 9 — how phone numbers are
// compared everywhere (0712…, 255712…, +255712… all match).
func last9(s string) string {
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

// ownedGroup returns the id of the group createdBy userId (preferred) or whose
// adminPhone matches username's last 9 digits.
func ownedGroup(groups []datatype.DataMap, userId, username string) string {
	tail := last9(username)
	match := ""
	for _, g := range groups {
		if helper.GetValueOfString(g, "status") == "Closed" {
			continue
		}
		id := helper.GetValueOfString(g, "id")
		if helper.GetValueOfString(g, "createdBy") == userId {
			return id
		}
		if match == "" && tail != "" && last9(helper.GetValueOfString(g, "adminPhone")) == tail {
			match = id
		}
	}
	return match
}

// ---------------------------------------------------------------------------
// REST guards
// ---------------------------------------------------------------------------

func deny(res *yekonga.Response, status int, msg string) {
	res.Status(status)
	res.Json(map[string]string{"error": msg})
}

// requireActor 401s when there is no (active) session.
func requireActor(app *yekonga.YekongaData, req *yekonga.Request, res *yekonga.Response) *Actor {
	a := actorFromRequest(app, req)
	if a == nil {
		deny(res, 401, "unauthorized")
		return nil
	}
	return a
}

// requirePerm 403s unless the actor has the platform-level permission.
func requirePerm(app *yekonga.YekongaData, req *yekonga.Request, res *yekonga.Response, perm string) *Actor {
	a := requireActor(app, req, res)
	if a == nil {
		return nil
	}
	if !a.Can(perm) {
		deny(res, 403, "your role does not allow this")
		return nil
	}
	return a
}

// ---------------------------------------------------------------------------
// Audit log (append-only; TODO.md U12 / N5)
// ---------------------------------------------------------------------------

func writeAudit(app *yekonga.YekongaData, a *Actor, action, model, recordId, groupId, summary string, changes interface{}) {
	rec := datatype.DataMap{
		"action":   action,
		"model":    model,
		"recordId": recordId,
		"groupId":  groupId,
		"summary":  summary,
		"changes":  changes,
	}
	if a != nil {
		rec["userId"] = a.UserID
		rec["userName"] = a.Name
		rec["role"] = a.Role
		if rec["role"] == "" && a.HomeGroupID != "" {
			rec["role"] = RoleGroupAdmin
		}
	}
	if r := app.ModelQuery("AuditLog").SkipBeforeCommit().Create(rec); r == nil {
		fmt.Println("audit: could not write log")
	} else if err, ok := r.(error); ok {
		fmt.Println("audit: could not write log:", err)
	}
}

// ---------------------------------------------------------------------------
// Startup migration: move existing data onto the new structure without
// locking anyone out. Safe to run on every start (idempotent).
// ---------------------------------------------------------------------------

const (
	defaultPartnerName = "Default Partner"
	defaultClusterName = "Default Cluster"
)

func migrateAccess(app *yekonga.YekongaData) {
	users := listAll(app, "User")

	// 1. Old role names → new ones. support_admin used to see everything, so
	//    it becomes Staff with the Operations preset and an "all" assignment.
	for _, u := range users {
		id := helper.GetValueOfString(u, "id")
		switch helper.GetValueOfString(u, "role") {
		case "admin", "1":
			app.ModelQuery("User").SkipBeforeCommit().Where("id", id).Update(datatype.DataMap{"role": RoleSuperAdmin}, nil)
		case "support_admin":
			app.ModelQuery("User").SkipBeforeCommit().Where("id", id).Update(datatype.DataMap{"role": RoleStaff}, nil)
			upsertPreset(app, id, PresetOperations, "")
			if app.ModelQuery("Assignment").SkipBeforeCommit().Where("userId", id).Where("scopeType", "all").First(nil) == nil {
				app.ModelQuery("Assignment").SkipBeforeCommit().Create(datatype.DataMap{"userId": id, "scopeType": "all"})
			}
			fmt.Printf("access: support_admin %s → staff (operations, all groups)\n", helper.GetValueOfString(u, "username"))
		}
	}

	// 2. Make sure someone can run the platform. SUPER_ADMIN_PHONES (comma
	//    separated) wins; otherwise, only when there is no super admin at all,
	//    the earliest account is promoted so the dashboard is never orphaned.
	if phones := strings.TrimSpace(os.Getenv("SUPER_ADMIN_PHONES")); phones != "" {
		want := map[string]bool{}
		for _, p := range strings.Split(phones, ",") {
			if t := last9(p); t != "" {
				want[t] = true
			}
		}
		for _, u := range users {
			if want[last9(helper.GetValueOfString(u, "username"))] && helper.GetValueOfString(u, "role") != RoleSuperAdmin {
				app.ModelQuery("User").SkipBeforeCommit().Where("id", helper.GetValueOfString(u, "id")).Update(datatype.DataMap{"role": RoleSuperAdmin}, nil)
				fmt.Printf("access: %s promoted to super_admin (SUPER_ADMIN_PHONES)\n", helper.GetValueOfString(u, "username"))
			}
		}
	} else if app.ModelQuery("User").SkipBeforeCommit().Where("role", RoleSuperAdmin).First(nil) == nil && len(users) > 0 {
		first := users[0]
		for _, u := range users[1:] {
			if helper.GetTimestamp(u["createdAt"]).Before(helper.GetTimestamp(first["createdAt"])) {
				first = u
			}
		}
		app.ModelQuery("User").SkipBeforeCommit().Where("id", helper.GetValueOfString(first, "id")).Update(datatype.DataMap{"role": RoleSuperAdmin}, nil)
		fmt.Printf("access: no super_admin found — promoted the first account %s (set SUPER_ADMIN_PHONES to choose)\n", helper.GetValueOfString(first, "username"))
	}

	// 3. Every group must sit in a cluster (and so a partner) for roll-up
	//    reports. Groups created before clusters existed go to a default one.
	groups := listAll(app, "Group")
	orphans := []string{}
	for _, g := range groups {
		if helper.GetValueOfString(g, "clusterId") == "" {
			orphans = append(orphans, helper.GetValueOfString(g, "id"))
		}
	}
	if len(orphans) > 0 {
		clusterId := ensureDefaultCluster(app)
		for _, id := range orphans {
			app.ModelQuery("Group").SkipBeforeCommit().Where("id", id).Update(datatype.DataMap{"clusterId": clusterId}, nil)
		}
		fmt.Printf("access: %d group(s) placed in %q\n", len(orphans), defaultClusterName)
	}

	// 4. Attendance rows get their meeting's groupId so they can be scoped.
	attendance := listAll(app, "MeetingAttendance")
	meetingGroup := map[string]string{}
	for _, row := range attendance {
		if helper.GetValueOfString(row, "groupId") != "" {
			continue
		}
		mid := helper.GetValueOfString(row, "meetingId")
		gid, ok := meetingGroup[mid]
		if !ok {
			if m := app.ModelQuery("Meeting").SkipBeforeCommit().Where("id", mid).First(nil); m != nil {
				gid = helper.GetValueOfString(*m, "groupId")
			}
			meetingGroup[mid] = gid
		}
		if gid != "" {
			app.ModelQuery("MeetingAttendance").SkipBeforeCommit().Where("id", helper.GetValueOfString(row, "id")).Update(datatype.DataMap{"groupId": gid}, nil)
		}
	}
	invalidateActors()
}

func ensureDefaultCluster(app *yekonga.YekongaData) string {
	partner := app.ModelQuery("Partner").SkipBeforeCommit().Where("name", defaultPartnerName).First(nil)
	partnerId := ""
	if partner != nil {
		partnerId = helper.GetValueOfString(*partner, "id")
	} else if created := app.ModelQuery("Partner").SkipBeforeCommit().Create(datatype.DataMap{"name": defaultPartnerName, "type": "Other", "status": "active"}); created != nil {
		partnerId = helper.GetValueOfString(helper.ToDataMap(created), "id")
	}
	cluster := app.ModelQuery("Cluster").SkipBeforeCommit().Where("name", defaultClusterName).First(nil)
	if cluster != nil {
		return helper.GetValueOfString(*cluster, "id")
	}
	created := app.ModelQuery("Cluster").SkipBeforeCommit().Create(datatype.DataMap{"name": defaultClusterName, "partnerId": partnerId, "status": "active"})
	return helper.GetValueOfString(helper.ToDataMap(created), "id")
}

func upsertPreset(app *yekonga.YekongaData, userId, preset, by string) {
	if existing := app.ModelQuery("AccessProfile").SkipBeforeCommit().Where("userId", userId).First(nil); existing != nil {
		app.ModelQuery("AccessProfile").SkipBeforeCommit().Where("id", helper.GetValueOfString(*existing, "id")).Update(datatype.DataMap{"preset": preset, "updatedBy": by}, nil)
		return
	}
	rec := datatype.DataMap{"userId": userId, "preset": preset}
	if by != "" {
		rec["updatedBy"] = by
	}
	app.ModelQuery("AccessProfile").SkipBeforeCommit().Create(rec)
}
