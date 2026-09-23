// Admin REST routes (server/access_routes.go, structure_routes.go,
// gov_loans.go, report_kpis.go). Every route checks the caller's role and
// scope on the server; errors come back as { error } with a 4xx status.
import { apiFetch } from './http.js'

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

// ---- monitoring -----------------------------------------------------------
export const getDashboard = (filters) => call('GET', '/api/admin/dashboard' + qs(filters))
export const getRollup = (level, filters) => call('GET', '/api/admin/rollup' + qs({ level, ...filters }))
export const listGovLoans = (groupId) => call('GET', '/api/admin/gov-loans' + qs({ groupId }))
export const listAudit = (filters) => call('GET', '/api/admin/audit' + qs(filters))
