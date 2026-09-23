// SMS switches and language, served by server/sms.go.
//   GET  /api/admin/sms/settings  -> { otpSms, transactionalSms, remindersSms, smsLanguage }
//   POST /api/admin/settings      -> saves the keys sent (platform admin only)

async function call(path, options) {
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

export function getNotificationSettings() {
  return call('/api/admin/sms/settings', { method: 'GET' })
}

// Send only the keys that changed. Resolves to the full, saved set.
export function saveNotificationSettings(changes) {
  return call('/api/admin/settings', { method: 'POST', body: JSON.stringify(changes) })
}
