// The built-in "User" model is protected from generic public GraphQL find
// queries (it carries password/token/otp fields), so this talks to the
// dedicated sanitized admin routes registered in server/main.go instead of
// the auto-generated CRUD API used by groups.js.
const ENDPOINT = '/api/admin/users'

async function request(path, options) {
  const res = await fetch(ENDPOINT + path, {
    credentials: 'include',
    headers: { 'Content-Type': 'application/json' },
    ...options
  })

  const json = await res.json().catch(() => null)

  if (!res.ok) {
    throw new Error(json?.error || 'Request failed')
  }

  return json
}

export const ROLES = [
  { value: 'super_admin', label: 'Super Admin' },
  { value: 'support_admin', label: 'Support Admin' },
  { value: 'group_admin', label: 'Group Admin' },
  { value: 'user', label: 'Member' }
]

export function roleLabel(value) {
  return ROLES.find((r) => r.value === value)?.label || value || 'Member'
}

export function listUsers() {
  return request('', { method: 'GET' })
}

export function updateUser(id, changes) {
  return request('/' + id, { method: 'POST', body: JSON.stringify(changes) })
}

export function deleteUser(id) {
  return request('/' + id, { method: 'DELETE' })
}
