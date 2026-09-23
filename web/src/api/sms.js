// Every member-facing SMS (Joined, Fine, Transaction, Loan) is recorded in the
// SmsLog model and served by server/main.go at /api/main/sms/activity.
// scope=all asks for everything the signed-in role may see (every group for a
// super admin, assigned partners/clusters/groups for staff and partners)
// instead of just the caller's own group.
import { apiFetch } from './http.js'

const ENDPOINT = '/api/main/sms/activity?scope=all'

async function request() {
  const res = await apiFetch(ENDPOINT, {
    headers: { 'Content-Type': 'application/json' }
  })
  const json = await res.json().catch(() => null)
  if (!res.ok) {
    throw new Error(json?.error || 'Request failed')
  }
  return json
}

export function listSmsActivity() {
  return request()
}
// Editable message templates (server/sms.go). Each entry is one
// (type, language) pair: { type, category, sw, en, language, body,
// defaultBody, custom, active, variables }.
async function send(path, options) {
  const res = await fetch(path, {
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

export function listSmsTemplates() {
  return send('/api/admin/sms/templates', { method: 'GET' })
}

// An empty body restores the built-in default for that type + language.
export function saveSmsTemplate({ type, language, body, active = true }) {
  return send('/api/admin/sms/templates', { method: 'POST', body: JSON.stringify({ type, language, body, active }) })
}
