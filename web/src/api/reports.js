// Reports — server/reports.go owns the data. One dataset registry feeds the
// live preview (fetchReport), the export picker (listDatasets) and the
// PDF / Excel / CSV download (exportReports). Every call is limited on the
// server to what the signed-in role may see; partnerId / clusterId / groupId
// narrow it further (organisation → partner → cluster → group).
import { apiFetch } from './http.js'

async function readError(res) {
  const json = await res.json().catch(() => null)
  return json?.error || 'Request failed'
}

// Live preview of one dataset:
// { key, columns, types: {col: type}, rows, totals?, pointInTime, balances,
//   asAt, enums: { sw: {code: label}, en: {...} } }
export async function fetchReport(type, { partnerId = '', clusterId = '', groupId = '', from = '', to = '' } = {}) {
  const params = new URLSearchParams({ type })
  if (partnerId) params.set('partnerId', partnerId)
  if (clusterId) params.set('clusterId', clusterId)
  if (groupId) params.set('groupId', groupId)
  if (from) params.set('from', from)
  if (to) params.set('to', to)

  const res = await apiFetch('/api/admin/reports?' + params.toString(), {
    headers: { 'Content-Type': 'application/json' }
  })
  if (!res.ok) throw new Error(await readError(res))
  return res.json()
}

// Catalog for the export picker:
// [{ key, category, categorySw, sw, en, pointInTime, balances, totals,
//    columns: [{ key, sw, en, type }] }]
// type is money | count | int | rate | date | datetime | enum | text. SMS
// datasets are only listed for roles that may read SMS logs.
export async function listDatasets() {
  const res = await apiFetch('/api/admin/reports/datasets')
  if (!res.ok) throw new Error(await readError(res))
  return res.json()
}

// Downloads the chosen datasets in one file.
//   datasets: ['loans', 'members']
//   columns:  { loans: ['Loan #', 'Balance'] }   (omit a key for "all columns")
//   format:   'xlsx' | 'pdf' | 'csv'   (csv with several datasets is a .zip)
export async function exportReports({ datasets, columns = {}, format, partnerId = '', clusterId = '', groupId = '', from = '', to = '', lang = 'en' }) {
  const res = await apiFetch('/api/admin/reports/export', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ datasets, columns, format, partnerId, clusterId, groupId, from, to, lang })
  })
  if (!res.ok) throw new Error(await readError(res))

  const filename =
    /filename="([^"]+)"/.exec(res.headers.get('Content-Disposition') || '')?.[1] || `helabox-report.${format}`
  const url = URL.createObjectURL(await res.blob())
  const a = document.createElement('a')
  a.href = url
  a.download = filename
  document.body.appendChild(a)
  a.click()
  a.remove()
  URL.revokeObjectURL(url)
}
