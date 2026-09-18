<script setup>
import { computed, onMounted, ref } from 'vue'
import Svgs from '../components/Svgs.vue'
import { currentUser } from '@/api/auth'
import { getDashboard } from '@/api/dashboard'

const firstName = ref('')
const dash = ref(null)
const loading = ref(true)
const error = ref('')

onMounted(async () => {
  const me = await currentUser()
  if (me) firstName.value = me.firstName || me.username || ''
  try {
    dash.value = await getDashboard()
  } catch (e) {
    error.value = e.message || 'Failed to load dashboard'
  } finally {
    loading.value = false
  }
})

const greeting = computed(() => {
  const h = new Date().getHours()
  const part = h < 12 ? 'Good morning' : h < 18 ? 'Good afternoon' : 'Good evening'
  return `${part}${firstName.value ? ', ' + firstName.value : ''}`
})

const today = computed(() =>
  new Date().toLocaleDateString('en-GB', {
    weekday: 'long',
    day: 'numeric',
    month: 'long',
    year: 'numeric'
  })
)

// Compact form for the narrow stat cards, e.g. 1245000000 -> "1.25B".
function compactMoney(n) {
  n = Number(n) || 0
  const abs = Math.abs(n)
  if (abs >= 1e9) return (n / 1e9).toFixed(2).replace(/\.?0+$/, '') + 'B'
  if (abs >= 1e6) return (n / 1e6).toFixed(2).replace(/\.?0+$/, '') + 'M'
  if (abs >= 1e3) return (n / 1e3).toFixed(1).replace(/\.0$/, '') + 'K'
  return n.toLocaleString()
}

const stats = computed(() => [
  { icon: 'groups', label: 'Total Groups', value: dash.value ? dash.value.totalGroups.toLocaleString() : '—' },
  { icon: 'user', label: 'Total Members', value: dash.value ? dash.value.totalMembers.toLocaleString() : '—' },
  { icon: 'wallet', label: 'Total Savings', value: dash.value ? 'TZS ' + compactMoney(dash.value.totalSavings) : '—' },
  { icon: 'doc', label: 'Loans Outstanding', value: dash.value ? 'TZS ' + compactMoney(dash.value.totalLoansOutstanding) : '—' }
])

const STATUS_LABELS = [
  ['api', 'API'],
  ['database', 'Database'],
  ['authentication', 'Authentication'],
  ['smsProvider', 'SMS Provider'],
  ['backgroundJobs', 'Background Jobs']
]

const statuses = computed(() => {
  const s = dash.value?.status
  if (!s) return []
  return STATUS_LABELS.map(([key, name]) => [
    name,
    s[key] ? 'Operational' : key === 'backgroundJobs' ? 'Not configured' : 'Unavailable'
  ])
})

const activity = computed(() => {
  return (dash.value?.recentActivity || []).map((a) => ({
    time: a.time,
    member: a.member || '—',
    group: a.group || '—',
    type: a.type,
    amount: (a.direction === 'in' ? '+' : '-') + 'TZS ' + Number(a.amount || 0).toLocaleString(),
    ok: a.direction === 'in'
  }))
})

const chartLine = computed(() => {
  const points = dash.value?.chart || []
  if (!points.length) return ''

  const data = points.map((p) => p.count)
  const labels = points.map((p) => p.day)
  const w = 660, h = 260, padL = 42, padB = 28, padT = 10
  const peak = Math.max(...data, 0)
  const maxY = peak > 0 ? Math.ceil(peak * 1.25) : 5
  const stepX = (w - padL - 10) / (data.length - 1)
  const pts = data.map((v, i) => ({
    x: padL + i * stepX,
    y: padT + (h - padT - padB) * (1 - v / maxY)
  }))
  const path = pts.map((p, i) => (i === 0 ? 'M' : 'L') + p.x.toFixed(1) + ',' + p.y.toFixed(1)).join(' ')
  const area = path + ` L${pts[pts.length - 1].x.toFixed(1)},${h} L${pts[0].x.toFixed(1)},${h} Z`
  const grid = [0, 0.25, 0.5, 0.75, 1].map((f) => Math.round(maxY * f))
  const gridLines = grid
    .map((g) => {
      const y = padT + (h - padT - padB) * (1 - g / maxY)
      return `<line x1="${padL}" y1="${y}" x2="${w}" y2="${y}" stroke="#EFEDEA" stroke-dasharray="4 4"/><text x="0" y="${y + 4}" font-size="11" fill="#8A9895" font-family="Inter">${g}</text>`
    })
    .join('')
  const xLabels = labels
    .map((l, i) => `<text x="${pts[i].x}" y="${h + 2}" font-size="11" fill="#8A9895" font-family="Inter" text-anchor="middle">${l}</text>`)
    .join('')
  const dots = pts
    .map((p) => `<circle cx="${p.x}" cy="${p.y}" r="4.5" fill="#18A672" stroke="#fff" stroke-width="2"/>`)
    .join('')

  return `<svg viewBox="0 0 ${w} ${h + 18}" style="width:100%;height:auto">
    ${gridLines}
    <path d="${area}" fill="url(#heroGrad)" opacity=".18"/>
    <path d="${path}" fill="none" stroke="#18A672" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"/>
    ${dots}
    ${xLabels}
    <defs><linearGradient id="heroGrad" x1="0" y1="0" x2="0" y2="1">
      <stop offset="0%" stop-color="#18A672"/><stop offset="100%" stop-color="#18A672"/>
    </linearGradient></defs>
  </svg>`
})
</script>

<template>
  <div>
    <div class="page-head">
      <div>
        <h1>{{ greeting }}</h1>
        <p>Platform overview &nbsp;|&nbsp; {{ today }}</p>
      </div>
    </div>

    <div v-if="error" style="color: var(--danger); font-size: 13px; margin-bottom: 12px">
      {{ error }}
    </div>

    <div class="stat-grid">
      <div v-for="s in stats" :key="s.label" class="stat-card">
        <div class="stat-icon" style="background: var(--green-100); color: var(--green-600)">
          <Svgs :name="s.icon" />
        </div>
        <div>
          <div class="stat-label">{{ s.label }}</div>
          <div class="stat-value">{{ s.value }}</div>
        </div>
      </div>
    </div>

    <div class="chart-row">
      <div class="chart-card">
        <h3>Transactions &middot; last 7 days</h3>
        <p v-if="!loading && !chartLine" class="cell-muted" style="font-size: 13px">No transactions recorded yet.</p>
        <div v-else v-html="chartLine"></div>
      </div>
      <div class="chart-card">
        <h3>System Status</h3>
        <div v-for="[name, state2] in statuses" :key="name" class="status-row">
          <span>{{ name }}</span>
          <span class="status-dot" :class="state2 === 'Operational' ? '' : 'red'"></span>
        </div>
      </div>
    </div>

    <div class="panel">
      <div class="panel-inner" style="padding-bottom: 0">
        <h3 style="font-size: 17px">Recent Activity</h3>
      </div>
      <div style="overflow-x: auto; margin-top: 14px">
        <table class="dtable">
          <thead>
            <tr><th>Time</th><th>Member</th><th>Group</th><th>Type</th><th>Amount</th></tr>
          </thead>
          <tbody>
            <tr v-if="!loading && !activity.length">
              <td colspan="5" class="cell-muted">No activity recorded yet.</td>
            </tr>
            <tr v-for="(a, i) in activity" :key="a.time + a.member + i">
              <td class="cell-muted">{{ a.time }}</td>
              <td class="cell-strong">{{ a.member }}</td>
              <td class="cell-muted">{{ a.group }}</td>
              <td class="cell-muted">{{ a.type }}</td>
              <td class="cell-strong" :style="{ color: a.ok ? 'var(--green-600)' : 'var(--danger)' }">{{ a.amount }}</td>
            </tr>
          </tbody>
        </table>
      </div>
      <div style="height: 20px"></div>
    </div>
  </div>
</template>