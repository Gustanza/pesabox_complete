// What the signed-in user may do — served by server/access_routes.go at
// /api/access/me. The router and the sidebar read it to hide pages and
// buttons the role can't use; the server re-checks every request anyway, so
// this is about showing the right screens, not about security.
import { reactive } from 'vue'
import { apiFetch } from './http.js'

export const access = reactive({ loaded: false, data: null })

let inflight = null

export function loadAccess(force = false) {
  if (access.loaded && !force) return Promise.resolve(access.data)
  if (!inflight) {
    inflight = apiFetch('/api/access/me')
      .then((res) => (res.ok ? res.json() : null))
      .catch(() => null)
      .then((data) => {
        access.data = data
        access.loaded = true
        return data
      })
      .finally(() => {
        inflight = null
      })
  }
  return inflight
}

export function clearAccess() {
  access.loaded = false
  access.data = null
}

// can: a platform-level permission (dashboard, reports, structure, ...), i.e.
// one that applies across the user's partners / clusters / groups.
export function can(perm) {
  return !!access.data?.platform?.includes(perm)
}

export const PERMS = {
  platform: 'platform.manage',
  audit: 'audit.view',
  dashboard: 'dashboard.view',
  structure: 'structure.view',
  reports: 'reports.view',
  sms: 'sms.view',
  groupCreate: 'groups.create',
  groupSettings: 'group.settings',
  finance: 'finance.write'
}
