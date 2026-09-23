// Reports — server/reports.go owns the data. One dataset registry feeds the
// live preview (fetchReport), the export picker (listDatasets) and the
// PDF / Excel / CSV download (exportReports).

async function readError(res) {
  const json = await res.json().catch(() => null)
  return json?.error || 'Request failed'
}

export async function fetchReport(type, { groupId = '', from = '', to = '' } = {}) {
  const params = new URLSearchParams({ type })
  if (groupId) params.set('groupId', groupId)
  if (from) params.set('from', from)
  if (to) params.set('to', to)

  const res = await fetch('/api/admin/reports?' + params.toString(), {
    credentials: 'include',
    headers: { 'Content-Type': 'application/json' }
  })
  if (!res.ok) throw new Error(await readError(res))
  return res.json()
}

// Catalog for the export picker:
// [{ key, category, categorySw, sw, en, columns: [{ key, sw, en }] }]
export async function listDatasets() {
  const res = await fetch('/api/admin/reports/datasets', { credentials: 'include' })
  if (!res.ok) throw new Error(await readError(res))
  return res.json()
}

// Downloads the chosen datasets in one file.
//   datasets: ['loans', 'members']
//   columns:  { loans: ['Loan #', 'Balance'] }   (omit a key for "all columns")
//   format:   'xlsx' | 'pdf' | 'csv'   (csv with several datasets is a .zip)
export async function exportReports({ datasets, columns = {}, format, groupId = '', from = '', to = '', lang = 'sw' }) {
  const res = await fetch('/api/admin/reports/export', {
    method: 'POST',
    credentials: 'include',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ datasets, columns, format, groupId, from, to, lang })
  })
  if (!res.ok) throw new Error(await readError(res))

  const filename =
    /filename="([^"]+)"/.exec(res.headers.get('Content-Disposition') || '')?.[1] || `pesabox-report.${format}`
  const url = URL.createObjectURL(await res.blob())
  const a = document.createElement('a')
  a.href = url
  a.download = filename
  document.body.appendChild(a)
  a.click()
  a.remove()
  URL.revokeObjectURL(url)
}
