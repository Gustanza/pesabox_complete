<script setup>
import { computed, onMounted, ref } from 'vue'
import Svgs from '../components/Svgs.vue'
import { currentUser } from '@/api/auth'

const firstName = ref('')

onMounted(async () => {
  const me = await currentUser()
  if (me) firstName.value = me.firstName || me.username || ''
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

const stats = [
  { icon: 'groups', label: 'Total Groups', value: '1,248' },
  { icon: 'user', label: 'Total Members', value: '28,450' },
  { icon: 'wallet', label: 'Total Savings', value: 'TZS 1.24B' },
  { icon: 'doc', label: 'Loans Outstanding', value: 'TZS 210M' }
]

const statuses = [
  ['API', 'Operational', 'green'],
  ['Database', 'Operational', 'green'],
  ['Authentication', 'Operational', 'green'],
  ['SMS Provider', 'Operational', 'green'],
  ['Background Jobs', 'Operational', 'green']
]

const activity = [
  { time: '09:42', member: 'Neema Joseph', group: 'Upendo Vikoba', type: 'Share', amount: '+TZS 15,000', ok: true },
  { time: '09:38', member: 'Asha Mwangi', group: 'Upendo Vikoba', type: 'Saving', amount: '+TZS 5,000', ok: true },
  { time: '09:34', member: 'John Mfinanga', group: 'Tumaini Group', type: 'Repayment', amount: '+TZS 20,000', ok: true },
  { time: '09:31', member: 'Grace Peter', group: 'Umoja Women', type: 'Fine', amount: '-TZS 1,000', ok: false },
  { time: '09:26', member: 'Fatuma R.', group: 'Mshikamano', type: 'Loan', amount: '-TZS 50,000', ok: true }
]

const chartLine = computed(() => {
  const data = [42, 58, 95, 70, 86, 122, 104]
  const labels = ['Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat', 'Sun']
  const w = 660, h = 260, padL = 42, padB = 28, padT = 10
  const maxY = 140
  const stepX = (w - padL - 10) / (data.length - 1)
  const pts = data.map((v, i) => ({
    x: padL + i * stepX,
    y: padT + (h - padT - padB) * (1 - v / maxY)
  }))
  const path = pts.map((p, i) => (i === 0 ? 'M' : 'L') + p.x.toFixed(1) + ',' + p.y.toFixed(1)).join(' ')
  const area = path + ` L${pts[pts.length - 1].x.toFixed(1)},${h} L${pts[0].x.toFixed(1)},${h} Z`
  const grid = [0, 35, 70, 105, 140]
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
        <div v-html="chartLine"></div>
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
            <tr v-for="a in activity" :key="a.time + a.member">
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