<script setup>
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import Svgs from '../components/Svgs.vue'
import { intlLocale } from '../i18n'
import { currentUser } from '@/api/auth'
import { can } from '@/api/access'
import { getDashboard, getRollup, listPartners, listClusters } from '@/api/admin'

// Live programme dashboard (server/report_kpis.go). Every number is limited to
// what the signed-in role may see, and can be narrowed to one partner or
// cluster; the roll-up table shows the same KPIs per partner / cluster / group.

const { t, te } = useI18n()
const router = useRouter()

const firstName = ref('')
const dash = ref(null)
const rollup = ref(null)
const loading = ref(true)
const error = ref('')

const showStructure = computed(() => can('structure.view'))
const partners = ref([])
const clusters = ref([])
const filters = reactive({ partnerId: '', clusterId: '' })
const level = ref('group')

const shownClusters = computed(() => clusters.value.filter((c) => !filters.partnerId || c.partnerId === filters.partnerId))

async function load() {
  loading.value = true
  error.value = ''
  try {
    const f = { partnerId: filters.partnerId, clusterId: filters.clusterId }
    ;[dash.value, rollup.value] = await Promise.all([getDashboard(f), getRollup(level.value, f)])
  } catch (e) {
    error.value = e.message || t('dash.loadFailed')
  } finally {
    loading.value = false
  }
}

onMounted(async () => {
  const me = await currentUser()
  if (me) firstName.value = me.firstName || me.username || ''
  if (showStructure.value) {
    try {
      ;[partners.value, clusters.value] = await Promise.all([listPartners(), listClusters()])
    } catch {
      // filters just stay empty
    }
  }
  await load()
})

watch(
  () => filters.partnerId,
  () => {
    filters.clusterId = ''
  }
)
watch(() => [filters.partnerId, filters.clusterId, level.value], load)

const greeting = computed(() => {
  const h = new Date().getHours()
  const part = h < 12 ? t('dash.morning') : h < 18 ? t('dash.afternoon') : t('dash.evening')
  return `${part}${firstName.value ? ', ' + firstName.value : ''}`
})

const today = computed(() =>
  new Date().toLocaleDateString(intlLocale(), { weekday: 'long', day: 'numeric', month: 'long', year: 'numeric' })
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
const tzs = (n) => 'TZS ' + compactMoney(n)
const num = (n) => Number(n || 0).toLocaleString(intlLocale())

const k = computed(() => dash.value?.kpis || {})

const stats = computed(() => [
  { key: 'dash.totalGroups', icon: 'groups', value: dash.value ? num(dash.value.totalGroups) : '—' },
  { key: 'dash.totalMembers', icon: 'user', value: dash.value ? num(dash.value.totalMembers) : '—' },
  { key: 'dash.totalSavings', icon: 'wallet', value: dash.value ? tzs(dash.value.totalSavings) : '—' },
  { key: 'dash.loansOutstanding', icon: 'doc', value: dash.value ? tzs(dash.value.totalLoansOutstanding) : '—' }
])

const kpis = computed(() => [
  { l: t('kpi.activeGroups'), v: num(k.value.activeGroups), d: t('kpi.inactiveN', { n: num(k.value.inactiveGroups) }), warn: k.value.inactiveGroups > 0 },
  { l: t('kpi.members'), v: num(k.value.members), d: t('kpi.womenMen', { f: num(k.value.femaleMembers), m: num(k.value.maleMembers) }) },
  { l: t('kpi.par30'), v: (k.value.par30Rate || 0) + '%', d: tzs(k.value.par30) + ' ' + t('kpi.atRisk'), warn: k.value.par30Rate > 5 },
  { l: t('kpi.attendance'), v: (k.value.attendanceRate || 0) + '%', d: t('kpi.attendanceNote') },
  { l: t('kpi.shares'), v: tzs(k.value.shares), d: t('kpi.socialFund') + ': ' + tzs(k.value.socialFund) },
  { l: t('kpi.govLoans'), v: tzs(k.value.govLoansOutstanding), d: t('kpi.govLoansNote') }
])

const STATUS_KEYS = [
  ['api', 'dash2.api'],
  ['database', 'dash2.database'],
  ['authentication', 'dash2.authentication'],
  ['smsProvider', 'dash.smsProvider'],
  ['backgroundJobs', 'dash2.jobs']
]
const statuses = computed(() => {
  const s = dash.value?.status
  if (!s) return []
  return STATUS_KEYS.map(([key, label]) => ({
    label,
    ok: !!s[key],
    note: s[key] ? t('dash.operational') : key === 'smsProvider' ? t('dash.devMode') : t('kpi.off')
  }))
})

const activity = computed(() =>
  (dash.value?.recentActivity || []).map((a) => ({
    time: a.time,
    member: a.member || '—',
    group: a.group || '—',
    type: a.type,
    amount: (a.direction === 'in' ? '+' : '-') + 'TZS ' + Number(a.amount || 0).toLocaleString(),
    ok: a.direction === 'in'
  }))
)

const chartLine = computed(() => {
  const points = dash.value?.chart || []
  if (!points.length || !points.some((p) => p.count > 0)) return ''

  const data = points.map((p) => p.count)
  const labels = points.map((p) => p.day)
  const w = 660, h = 260, padL = 42, padB = 28, padT = 10
  const peak = Math.max(...data, 0)
  const maxY = peak > 0 ? Math.ceil(peak * 1.25) : 5
  const stepX = (w - padL - 10) / (data.length - 1)
  const pts = data.map((v, i) => ({ x: padL + i * stepX, y: padT + (h - padT - padB) * (1 - v / maxY) }))
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
  const dots = pts.map((p) => `<circle cx="${p.x}" cy="${p.y}" r="4.5" fill="#18A672" stroke="#fff" stroke-width="2"/>`).join('')

  return `<svg viewBox="0 0 ${w} ${h + 18}" style="width:100%;height:auto">
    ${gridLines}
    <path d="${area}" fill="#18A672" opacity=".18"/>
    <path d="${path}" fill="none" stroke="#18A672" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"/>
    ${dots}
    ${xLabels}
  </svg>`
})

function openRow(r) {
  if (level.value === 'group') router.push('/groups/' + r.id)
  else if (level.value === 'cluster') filters.clusterId = r.id
  else filters.partnerId = r.id
}
</script>

<template>
  <div>
    <div class="page-head">
      <div>
        <h1>{{ greeting }}</h1>
        <p>{{ t('dash.overview') }} &nbsp;|&nbsp; {{ today }}</p>
      </div>
      <div v-if="showStructure" class="page-actions">
        <select v-model="filters.partnerId" class="filter-select">
          <option value="">{{ t('struct.allPartners') }}</option>
          <option v-for="p in partners" :key="p.id" :value="p.id">{{ p.name }}</option>
        </select>
        <select v-model="filters.clusterId" class="filter-select">
          <option value="">{{ t('struct.allClusters') }}</option>
          <option v-for="c in shownClusters" :key="c.id" :value="c.id">{{ c.name }}</option>
        </select>
      </div>
    </div>

    <div v-if="error" style="color: var(--danger); font-size: 13px; margin-bottom: 12px">{{ error }}</div>

    <div class="stat-grid">
      <div v-for="s in stats" :key="s.key" class="stat-card">
        <div class="stat-icon" style="background: var(--green-100); color: var(--green-600)">
          <Svgs :name="s.icon" />
        </div>
        <div>
          <div class="stat-label">{{ t(s.key) }}</div>
          <div class="stat-value">{{ s.value }}</div>
        </div>
      </div>
    </div>

    <div class="kpi-grid">
      <div v-for="x in kpis" :key="x.l" class="kpi-card">
        <div class="l">{{ x.l }}</div>
        <div class="v">{{ dash ? x.v : '—' }}</div>
        <div class="d" :style="x.warn ? 'color: var(--danger)' : ''">{{ dash ? x.d : '' }}</div>
      </div>
    </div>

    <div class="chart-row">
      <div class="chart-card">
        <h3>{{ t('dash.txChart') }}</h3>
        <p v-if="!loading && !chartLine" class="cell-muted" style="font-size: 13px">{{ t('kpi.noTx7') }}</p>
        <div v-else v-html="chartLine"></div>
      </div>
      <div class="chart-card">
        <h3>{{ t('dash.systemStatus') }}</h3>
        <div v-for="s in statuses" :key="s.label" class="status-row">
          <span>{{ t(s.label) }}</span>
          <span style="display: flex; align-items: center; gap: 8px; font-size: 12px; color: var(--ink-400)">
            {{ s.note }}<span class="status-dot" :class="s.ok ? '' : 'red'"></span>
          </span>
        </div>
      </div>
    </div>

    <div class="panel">
      <div class="panel-inner" style="padding-bottom: 0; display: flex; justify-content: space-between; align-items: center; flex-wrap: wrap; gap: 10px">
        <h3 style="font-size: 17px">{{ t('kpi.rollupTitle') }}</h3>
        <div class="dtabs" style="margin: 0">
          <button v-if="showStructure" class="dtab" :class="{ active: level === 'partner' }" @click="level = 'partner'">{{ t('kpi.byPartner') }}</button>
          <button v-if="showStructure" class="dtab" :class="{ active: level === 'cluster' }" @click="level = 'cluster'">{{ t('kpi.byCluster') }}</button>
          <button class="dtab" :class="{ active: level === 'group' }" @click="level = 'group'">{{ t('kpi.byGroup') }}</button>
        </div>
      </div>
      <div style="overflow-x: auto; margin-top: 14px">
        <table class="dtable">
          <thead>
            <tr>
              <th>{{ t('kpi.levels.' + level) }}</th>
              <th v-if="level !== 'group'">{{ t('struct.groups') }}</th>
              <th>{{ t('kpi.members') }}</th>
              <th>{{ t('dash.totalSavings') }}</th>
              <th>{{ t('dash.loansOutstanding') }}</th>
              <th>{{ t('kpi.par30') }}</th>
              <th>{{ t('kpi.attendance') }}</th>
              <th>{{ t('kpi.govLoans') }}</th>
            </tr>
          </thead>
          <tbody>
            <tr v-if="!loading && !(rollup?.rows || []).length"><td colspan="8" class="cell-muted">{{ t('kpi.noRows') }}</td></tr>
            <tr v-for="r in rollup?.rows || []" :key="r.id" class="clickable" @click="openRow(r)">
              <td class="cell-strong">
                {{ r.name }}
                <span v-if="level === 'group' && !r.activeGroups" class="badge grey" style="margin-left: 6px">{{ t('kpi.inactive') }}</span>
              </td>
              <td v-if="level !== 'group'">{{ r.groups }} <span class="cell-sub">({{ t('kpi.activeN', { n: r.activeGroups }) }})</span></td>
              <td>{{ num(r.members) }} <span class="cell-sub">{{ t('kpi.fm', { f: r.femaleMembers, m: r.maleMembers }) }}</span></td>
              <td>{{ tzs(r.savings) }}</td>
              <td>{{ tzs(r.loansOutstanding) }}</td>
              <td :style="r.par30Rate > 5 ? 'color: var(--danger); font-weight: 700' : ''">{{ r.par30Rate }}%</td>
              <td>{{ r.attendanceRate }}%</td>
              <td>{{ tzs(r.govLoansOutstanding) }}</td>
            </tr>
          </tbody>
        </table>
      </div>
      <div style="height: 20px"></div>
    </div>

    <div class="panel">
      <div class="panel-inner" style="padding-bottom: 0">
        <h3 style="font-size: 17px">{{ t('dash.recent') }}</h3>
      </div>
      <div style="overflow-x: auto; margin-top: 14px">
        <table class="dtable">
          <thead>
            <tr>
              <th>{{ t('dash.time') }}</th>
              <th>{{ t('dash.member') }}</th>
              <th>{{ t('dash.group') }}</th>
              <th>{{ t('dash.type') }}</th>
              <th>{{ t('dash.amount') }}</th>
            </tr>
          </thead>
          <tbody>
            <tr v-if="!loading && !activity.length">
              <td colspan="5" class="cell-muted">{{ t('dash.noActivity') }}</td>
            </tr>
            <tr v-for="(a, i) in activity" :key="a.time + a.member + i">
              <td class="cell-muted">{{ a.time }}</td>
              <td class="cell-strong">{{ a.member }}</td>
              <td class="cell-muted">{{ a.group }}</td>
              <td class="cell-muted">{{ te('dash2.act.' + a.type.toLowerCase()) ? t('dash2.act.' + a.type.toLowerCase()) : a.type }}</td>
              <td class="cell-strong" :style="{ color: a.ok ? 'var(--green-600)' : 'var(--danger)' }">{{ a.amount }}</td>
            </tr>
          </tbody>
        </table>
      </div>
      <div style="height: 20px"></div>
    </div>
  </div>
</template>
