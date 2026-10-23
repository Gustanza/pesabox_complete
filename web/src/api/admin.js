// Admin REST routes (server/access_routes.go, structure_routes.go,
// gov_loans.go, report_kpis.go). Every route checks the caller's role and
// scope on the server; errors come back as { error } with a 4xx status.
import { apiFetch, apiUrl } from './http.js'

async function call(method, path, body) {
  const res = await apiFetch(path, {
    method,
    headers: { 'Content-Type': 'application/json' },
    body: body === undefined ? undefined : JSON.stringify(body)
  })
  const json = await res.json().catch(() => null)
  if (!res.ok) throw new Error(json?.error || 'Request failed')
  return json
}

const qs = (params) => {
  const p = new URLSearchParams()
  for (const [k, v] of Object.entries(params || {})) if (v) p.set(k, v)
  const s = p.toString()
  return s ? '?' + s : ''
}

// ---- users, roles, assignments ------------------------------------------
export const listUsers = () => call('GET', '/api/admin/users')
export const addUser = (user) => call('POST', '/api/admin/users', user)
export const updateUser = (id, changes) => call('POST', '/api/admin/users/' + id, changes)
export const deactivateUser = (id) => call('DELETE', '/api/admin/users/' + id)
export const addAssignment = (a) => call('POST', '/api/admin/assignments', a)
export const removeAssignment = (id) => call('DELETE', '/api/admin/assignments/' + id)

// ---- partners & clusters -------------------------------------------------
export const listPartners = () => call('GET', '/api/admin/partners')
export const createPartner = (p) => call('POST', '/api/admin/partners', p)
export const updatePartner = (id, p) => call('POST', '/api/admin/partners/' + id, p)
export const listClusters = (partnerId) => call('GET', '/api/admin/clusters' + qs({ partnerId }))
export const createCluster = (c) => call('POST', '/api/admin/clusters', c)
export const updateCluster = (id, c) => call('POST', '/api/admin/clusters/' + id, c)

// ---- groups ---------------------------------------------------------------
export const moveGroup = (groupId, clusterId) => call('POST', `/api/admin/groups/${groupId}/cluster`, { clusterId })
// Only succeeds while the group has no financial records (otherwise close it).
export const deleteGroupSafe = (groupId) => call('DELETE', '/api/admin/groups/' + groupId)

// Group rules (the constitution, server/group_rules.go): { rules, canEdit,
// services, note }. Saving sends only rule fields; the server validates and
// answers { error } with a clear message. Changes apply to new records only.
export const getGroupRules = (groupId) => call('GET', `/api/admin/groups/${groupId}/rules`)
export const saveGroupRules = (groupId, rules) => call('PUT', `/api/admin/groups/${groupId}/rules`, rules)

// ---- monitoring -----------------------------------------------------------
export const getDashboard = (filters) => call('GET', '/api/admin/dashboard' + qs(filters))
export const getRollup = (level, filters) => call('GET', '/api/admin/rollup' + qs({ level, ...filters }))
// GATA monitoring indicators (server/monitoring.go, contract in
// server/MONITORING_API.md). Filters: partnerId, clusterId, groupId, from, to.
// With no `from` the server uses the last 12 months and says so in `period`.
export const getMonitoring = (filters) => call('GET', '/api/admin/monitoring' + qs(filters))
export const listGovLoans = (groupId) => call('GET', '/api/admin/gov-loans' + qs({ groupId }))
export const listAudit = (filters) => call('GET', '/api/admin/audit' + qs(filters))

// ---- mobile app releases (server/app_releases.go, super admin) --------------
export const listAppReleases = () => call('GET', '/api/admin/app-releases')
export const deleteAppRelease = (id) => call('DELETE', '/api/admin/app-releases/' + id)

// Upload an APK with progress. XHR rather than fetch: fetch has no upload
// progress events. onProgress receives 0..1. Resolves { release }.
export function uploadAppRelease({ file, version, notes }, onProgress) {
  return new Promise((resolve, reject) => {
    const form = new FormData()
    form.append('version', version)
    form.append('notes', notes || '')
    form.append('file', file)
    const xhr = new XMLHttpRequest()
    xhr.open('POST', apiUrl('/api/admin/app-releases'))
    xhr.withCredentials = true
    xhr.setRequestHeader('Accept', 'application/json')
    xhr.upload.onprogress = (e) => {
      if (e.lengthComputable && onProgress) onProgress(e.loaded / e.total)
    }
    xhr.onload = () => {
      let json = null
      try { json = JSON.parse(xhr.responseText) } catch { /* not JSON (e.g. proxy error page) */ }
      if (xhr.status >= 200 && xhr.status < 300) return resolve(json)
      if (xhr.status === 413) return reject(new Error('too_large'))
      if (xhr.status === 401) return reject(new Error('session'))
      reject(new Error(json?.error || 'Request failed'))
    }
    xhr.onerror = () => reject(new Error('network'))
    xhr.send(form)
  })
}
