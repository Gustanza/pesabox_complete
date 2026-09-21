// Every member-facing SMS (Joined, Fine, Transaction, Loan) is recorded in the
// SmsLog model and served by server/main.go at /api/main/sms/activity. The
// super-admin dashboard shows the full platform history; a group admin only
// sees their own group's messages.
const ENDPOINT = '/api/main/sms/activity'

async function request() {
  const res = await fetch(ENDPOINT, {
    credentials: 'include',
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
