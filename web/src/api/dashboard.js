// Cross-group platform snapshot for the Super Admin dashboard, served by
// server/main.go at /api/admin/dashboard (not group-scoped, unlike /api/main/*).
import { apiFetch } from './http.js'

const ENDPOINT = '/api/admin/dashboard'

export async function getDashboard() {
  const res = await apiFetch(ENDPOINT, {
    headers: { 'Content-Type': 'application/json' }
  })
  const json = await res.json().catch(() => null)
  if (!res.ok) {
    throw new Error(json?.error || 'Request failed')
  }
  return json
}
