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