<script setup>
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { intlLocale } from '../i18n'
import { listGroups } from '@/api/groups'
import { listPartners, listClusters, getMonitoring } from '@/api/admin'
import { can } from '@/api/access'

// GATA's monitoring indicators (server/monitoring.go; contract and the
// definition of every number in server/MONITORING_API.md). The server does
// all the arithmetic and limits everything to the caller's scope; this page
// only lays the sections out. With no `from` the server takes the last 12
// months — the period it actually used is shown from `period`.

const { t, te, locale } = useI18n()
const TZ = 'Africa/Dar_es_Salaam'

const filters = reactive({ partnerId: '', clusterId: '', groupId: '', from: '', to: '' })
const partners = ref([])
const clusters = ref([])
const groups = ref([])
const data = ref(null)
const loading = ref(true)
const error = ref('')

// Partner / cluster filters only for roles that may see the structure.
const showStructure = computed(() => can('structure.view'))
const shownClusters = computed(() => clusters.value.filter((c) => !filters.partnerId || c.partnerId === filters.partnerId))
const shownGroups = computed(() => {
  if (filters.clusterId) return groups.value.filter((g) => g.clusterId === filters.clusterId)
  if (filters.partnerId) {
    const ids = new Set(shownClusters.value.map((c) => c.id))
    return groups.value.filter((g) => ids.has(g.clusterId))
  }
  return groups.value
})
watch(() => filters.partnerId, () => { filters.clusterId = ''; filters.groupId = '' })
watch(() => filters.clusterId, () => { filters.groupId = '' })

let seq = 0
async function load() {
  const id = ++seq
  loading.value = true
  error.value = ''
  try {
    const res = await getMonitoring({ ...filters })
    if (id === seq) data.value = res
  } catch (e) {
    if (id === seq) {
      data.value = null
      error.value = e.message || t('mon.loadFailed')
    }
  } finally {
    if (id === seq) loading.value = false
  }
}
watch(() => ({ ...filters }), load)

onMounted(async () => {
  try {
    groups.value = await listGroups()
    if (showStructure.value) {
      ;[partners.value, clusters.value] = await Promise.all([listPartners(), listClusters()])
    }
  } catch {
    // filters just won't have options; the indicators still load
  }
  await load()
})

// ---- formatting --------------------------------------------------------------
const num = (n, digits = 0) => Number(n || 0).toLocaleString(intlLocale(), { maximumFractionDigits: digits })
function fmt(kind, v) {
  if (v === null || v === undefined || v === '') return '—'
  if (kind === 'money') return 'TZS ' + num(v, 2)
  if (kind === 'pct') return num(v, 1) + '%'
  return num(v, 1)
}
function dayText(iso) {
  if (!iso) return '—'
  const d = new Date(iso + 'T12:00:00+03:00')
  return isNaN(d) ? iso : d.toLocaleDateString(intlLocale(), { timeZone: TZ, day: 'numeric', month: 'short', year: 'numeric' })
}
const MONEY = new Set([
  'savingsRecorded', 'netSavings', 'totalBalance', 'totalBalanceAtStart', 'avgPerContributor', 'avgBalancePerMember',
  'memberContributionMin', 'memberContributionMedian', 'memberContributionMax', 'mandatorySavings', 'voluntarySavings',
  'sharesPurchased', 'withdrawals', 'savingsBalance', 'amountDisbursed', 'averageLoanSize', 'outstandingBalance',
  'outstandingAtStart', 'interestCharged', 'loanAmount', 'repaymentsCollected', 'amountFallenDue', 'amountRepaidOnFallenDue',
  'overdueAmount', 'par30Amount', 'defaultedAmount', 'contributions', 'disbursements', 'availableBalance', 'balanceAtStart',
  'interestEarned', 'finesCollected', 'operationalExpenses', 'netIncome', 'savingsSharedOut', 'savingsReturned',
  'returnsToMembers', 'avgReturnPerMember', 'driftAmount', 'reversedAmount', 'expectedCashOnHand', 'cashOnHandAtStart',
  'savingsStart', 'savingsEnd', 'loanPortfolioStart', 'loanPortfolioEnd', 'assetsStart', 'assetsEnd',
  'avgSavingsPerMemberStart', 'avgSavingsPerMemberEnd', 'avgBorrowingPerMemberStart', 'avgBorrowingPerMemberEnd',
  'avgLoanPerBorrower', 'igaAmount', 'amount'
])
const kindOf = (field) => (MONEY.has(field) ? 'money' : /(Pct|Rate|RateAllTime)$/.test(field) ? 'pct' : 'count')
const fieldLabel = (f) => (te('mon.f.' + f) ? t('mon.f.' + f) : f)

// Row labels in breakdown tables: gender, age band, purpose, audit action, yes/no.
function rowLabel(field, v) {
  if (field === 'purpose') return te('rules.purposes.' + v) ? t('rules.purposes.' + v) : v
  if (field === 'action') return te('aud.actions.' + v) ? t('aud.actions.' + v) : v
  if (field === 'incomeGenerating') return v ? t('mon.v.yes') : t('mon.v.no')
  if (field === 'gender' || field === 'ageBand') return te('mon.v.' + v) ? t('mon.v.' + v) : v
  return v === '' || v === null || v === undefined ? '—' : String(v)
}
const LABEL_FIELDS = ['gender', 'ageBand', 'purpose', 'enterpriseType', 'action', 'incomeGenerating']

// ---- the eleven GATA sections ---------------------------------------------------
// src: the response object; tiles: fields shown as stat tiles; tables: breakdown
// arrays as [field, columns]; datasets: matching export datasets on Reports.
const SECTIONS = [
  {
    key: 'savings', src: 'savings', datasets: ['member-savings', 'monitoring-summary'],
    tiles: ['savingsRecorded', 'netSavings', 'totalBalance', 'growthPct', 'contributingMembers', 'avgPerContributor',
      'mandatorySavings', 'voluntarySavings', 'sharesPurchased', 'withdrawals', 'memberContributionMin', 'memberContributionMedian', 'memberContributionMax'],
    tables: [['byGender', ['gender', 'savers', 'savingsRecorded', 'savingsBalance']], ['byAgeBand', ['ageBand', 'savers', 'savingsRecorded', 'savingsBalance']]]
  },
  {
    key: 'loans', src: 'loans', datasets: ['monitoring-summary', 'loans-by-purpose'],
    tiles: ['loansIssued', 'amountDisbursed', 'borrowers', 'averageLoanSize', 'interestCharged', 'outstandingBalance', 'outstandingAtStart', 'activeLoans', 'borrowersWithOpenLoans'],
    tables: [['byGender', ['gender', 'loans', 'borrowers', 'loanAmount']], ['byAgeBand', ['ageBand', 'loans', 'borrowers', 'loanAmount']]]
  },
  {
    key: 'repayments', src: 'repayments', datasets: ['monitoring-summary'],
    tiles: ['repaymentsCollected', 'repaymentRate', 'repaymentRateAllTime', 'loansFallenDue', 'amountFallenDue', 'amountRepaidOnFallenDue',
      'fullyRepaidLoans', 'overdueLoans', 'overdueAmount', 'overdueBorrowers', 'par30Rate', 'par30Amount', 'defaultRate', 'defaultedLoans', 'defaultedAmount']
  },
  {
    key: 'socialFund', src: 'socialFund', datasets: ['social-fund-payouts'],
    tiles: ['contributions', 'disbursements', 'availableBalance', 'balanceAtStart', 'contributors', 'beneficiaries', 'payouts']
  },
  {
    key: 'membership', src: 'membership', datasets: ['members-by-age', 'monitoring-summary'],
    tiles: ['activeMembers', 'totalMembers', 'femaleMembers', 'maleMembers', 'femalePct', 'youthMembers', 'youthPct', 'newMembers', 'exits',
      'membersAtStart', 'retentionRate', 'attendanceRate', 'meetingsHeld', 'dateOfBirthCoveragePct'],
    tables: [
      ['byGender', ['gender', 'activeMembers', 'newMembers', 'retentionRate', 'attendanceRate']],
      ['byAgeBand', ['ageBand', 'activeMembers', 'newMembers', 'retentionRate', 'attendanceRate']],
      ['byGenderAndAge', ['gender', 'ageBand', 'activeMembers', 'newMembers']]
    ]
  },
  {
    key: 'digital', src: 'digitalAdoption', datasets: ['monitoring-summary'],
    tiles: ['groups', 'groupsUsingApp', 'adoptionRate', 'groupsUsingAppInPeriod', 'digitalRecordingRate', 'meetingsCompleted', 'meetingsFullyRecorded', 'transactionsRecorded', 'recordingUsers']
  },
  {
    key: 'financial', src: 'financialPerformance', datasets: ['share-outs', 'monitoring-summary'],
    tiles: ['interestEarned', 'interestCharged', 'finesCollected', 'operationalExpenses', 'netIncome', 'savingsSharedOut', 'savingsReturned',
      'returnsToMembers', 'returnOnSavingsPct', 'membersPaidOut', 'avgReturnPerMember']
  },
  {
    key: 'accountability', src: 'accountability', datasets: ['monitoring-summary'],
    tiles: ['recordsAccuracyPct', 'groupsChecked', 'groupsWithDrift', 'transactions', 'reversals', 'reversalRate', 'reversedAmount', 'overpaidLoans',
      'transactionsAtMeetingPct', 'expectedCashOnHand', 'auditEntries', 'smsSent', 'smsFailed', 'smsDeliveryRate', 'membersNotifiedPct'],
    tables: [['auditByAction', ['action', 'count']]]
  },
  {
    key: 'growth', src: 'growth', datasets: ['monthly-growth'], chart: true,
    tiles: ['savingsStart', 'savingsEnd', 'savingsGrowthPct', 'loanPortfolioStart', 'loanPortfolioEnd', 'loanPortfolioGrowthPct',
      'assetsStart', 'assetsEnd', 'assetsGrowthPct', 'membersStart', 'membersEnd', 'membersGrowthPct']
  },
  {
    key: 'access', src: 'access', datasets: ['members-by-age', 'member-savings'],
    tiles: ['savers', 'saversPct', 'regularSavers', 'regularSaversPct', 'borrowers', 'accessingLoansPct', 'membersWithOpenLoans', 'membersWithOpenLoansPct',
      'avgSavingsPerMemberStart', 'avgSavingsPerMemberEnd', 'avgSavingsChangePct', 'avgBorrowingPerMemberStart', 'avgBorrowingPerMemberEnd', 'avgBorrowingChangePct', 'avgLoanPerBorrower'],
    tables: [
      ['byGender', ['gender', 'activeMembers', 'eligibleSavers', 'regularSavers', 'regularSaversPct', 'borrowers', 'borrowersPct']],
      ['byAgeBand', ['ageBand', 'activeMembers', 'eligibleSavers', 'regularSavers', 'regularSaversPct', 'borrowers', 'borrowersPct']]
    ]
  },
  {
    key: 'economic', src: 'economicBenefits', datasets: ['loans-by-purpose', 'loans-by-enterprise'],
    tiles: ['igaLoans', 'igaLoansPct', 'igaBorrowers', 'igaBorrowersPct', 'igaAmount', 'loansWithPurpose', 'unspecifiedLoans', 'interestEarned', 'returnsToMembers', 'avgReturnPerMember'],
    tables: [
      ['byPurpose', ['purpose', 'incomeGenerating', 'loans', 'borrowers', 'amount']],
      ['byEnterpriseType', ['enterpriseType', 'loans', 'borrowers', 'amount']],
      ['igaBorrowersByGender', ['gender', 'igaBorrowers']],
      ['igaBorrowersByAgeBand', ['ageBand', 'igaBorrowers']]
    ]
  }
]

// Values for the one-line "how it's calculated" hints.
const hintArgs = computed(() => {
  const d = data.value?.definitions || {}
  return {
    asAt: dayText(data.value?.period?.asAt),
    par: d.parDays ?? 30,
    def: d.defaultAfterDays ?? 90,
    reg: d.regularSaverThresholdPct ?? 75,
    youth: d.youthAges || '15-35',
    days: d.appActiveDays ?? 30,
    iga: (d.incomeGeneratingPurposes || []).map((p) => rowLabel('purpose', p)).join(', ')
  }
})

const sections = computed(() => {
  if (!data.value) return []
  return SECTIONS.map((s) => {
    const src = data.value[s.src] || {}
    return {
      ...s,
      tiles: s.tiles.filter((f) => f in src).map((f) => ({ f, label: fieldLabel(f), value: fmt(kindOf(f), src[f]) })),
      tables: (s.tables || [])
        .map(([key, cols]) => ({ key, cols, rows: Array.isArray(src[key]) ? src[key] : [] }))
        .filter((tb) => tb.rows.length)
    }
  })
})

function cell(col, row) {
  return LABEL_FIELDS.includes(col) ? rowLabel(col, row[col]) : fmt(kindOf(col), row[col])
}

const period = computed(() => data.value?.period || null)
const empty = computed(() => data.value && !data.value.scope?.groups)

// Link to the matching export dataset on Reports, carrying the same filters.
function reportLink(key) {
  const query = { dataset: key }
  for (const [k, v] of Object.entries(filters)) if (v) query[k] = v
  return { path: '/reports', query }
}
const datasetLabel = (key) => (te('mon.ds.' + key) ? t('mon.ds.' + key) : key)

// growth.monthly as grouped bars: savings + shares balance and the loan
// portfolio at each month end. Plain SVG, no chart library.
const chart = computed(() => {
  const pts = data.value?.growth?.monthly || []
  if (!pts.length || !pts.some((p) => p.totalSavings || p.loanPortfolio)) return null
  const w = 720, h = 220, padL = 64, padB = 26, padT = 10
  const max = Math.max(...pts.map((p) => Math.max(p.totalSavings || 0, p.loanPortfolio || 0)), 1)
  const slot = (w - padL - 8) / pts.length
  const bw = Math.max(3, Math.min(18, slot / 2 - 3))
  const y = (v) => padT + (h - padT - padB) * (1 - v / max)
  const ticks = [0, 0.5, 1].map((f) => max * f)
  const short = (n) => (n >= 1e6 ? num(n / 1e6, 1) + 'M' : n >= 1e3 ? num(n / 1e3, 1) + 'K' : num(n))
  const loc = intlLocale()
  const bars = pts.map((p, i) => {
    const x = padL + i * slot + slot / 2
    const label = new Date(p.month + '-15T12:00:00+03:00').toLocaleDateString(loc, { month: 'short', year: '2-digit', timeZone: TZ })
    return {
      x, label,
      a: { x: x - bw - 1, y: y(p.totalSavings || 0), h: h - padB - y(p.totalSavings || 0) },
      b: { x: x + 1, y: y(p.loanPortfolio || 0), h: h - padB - y(p.loanPortfolio || 0) },
      title: `${label}: ${t('mon.chartSavings')} TZS ${num(p.totalSavings)} · ${t('mon.chartLoans')} TZS ${num(p.loanPortfolio)} · ${t('mon.f.activeMembers')} ${num(p.activeMembers)}`
    }
  })
  const every = Math.ceil(pts.length / 12)
  return { w, h, padL, padB, bw, bars, every, ticks: ticks.map((v) => ({ y: y(v), label: short(v) })) }
})
</script>

<template>
  <div>
    <div class="page-head">
      <div>
        <h1>{{ t('mon.title') }}</h1>
        <p>{{ t('mon.subtitle') }}</p>
      </div>
    </div>

    <div class="card mon-filters">
      <div v-if="showStructure" class="field-view">
        <label>{{ t('struct.partner') }}</label>
        <select v-model="filters.partnerId" class="box">
          <option value="">{{ t('struct.allPartners') }}</option>
          <option v-for="p in partners" :key="p.id" :value="p.id">{{ p.name }}</option>
        </select>
      </div>
      <div v-if="showStructure" class="field-view">
        <label>{{ t('struct.cluster') }}</label>
        <select v-model="filters.clusterId" class="box">
          <option value="">{{ t('struct.allClusters') }}</option>
          <option v-for="c in shownClusters" :key="c.id" :value="c.id">{{ c.name }}</option>
        </select>
      </div>
      <div class="field-view">
        <label>{{ t('reports.group') }}</label>
        <select v-model="filters.groupId" class="box">
          <option value="">{{ t('reports.allGroups') }}</option>
          <option v-for="g in shownGroups" :key="g.id" :value="g.id">{{ g.name }}</option>
        </select>
      </div>
      <div class="field-view">
        <label>{{ t('mon.from') }}</label>
        <input v-model="filters.from" type="date" class="box" :max="filters.to || undefined" />
      </div>
      <div class="field-view">
        <label>{{ t('mon.to') }}</label>
        <input v-model="filters.to" type="date" class="box" :min="filters.from || undefined" />
      </div>
    </div>

    <p v-if="period" class="mon-period">
      <b>{{ t('mon.period', { from: dayText(period.from), to: dayText(period.to) }) }}</b>
      · {{ t('mon.asAt', { date: dayText(period.asAt) }) }}
      <template v-if="data.scope"> · {{ data.scope.label || t('mon.allScope') }} · {{ t('mon.groupsN', { n: data.scope.groups }) }}</template>
      <span v-if="period.defaultedFrom" class="cell-sub"> — {{ t('mon.defaultPeriod', { n: period.months }) }}</span>
    </p>

    <div v-if="error" role="alert" style="color: var(--danger); font-size: 13px; margin-bottom: 12px">{{ error }}</div>
    <div v-if="loading && !data" class="empty"><p>{{ t('common.loading') }}</p></div>
    <div v-else-if="empty" class="card empty">
      <div class="ic">&#128202;</div>
      <p>{{ t('mon.empty') }}</p>
    </div>

    <template v-else-if="data">
      <section v-for="s in sections" :key="s.key" class="card mon-section" :class="{ 'mon-loading': loading }">
        <div class="mon-head">
          <h3>{{ t('mon.s.' + s.key) }}</h3>
          <span class="mon-links">
            <router-link v-for="ds in s.datasets" :key="ds" :to="reportLink(ds)" class="btn btn-outline btn-sm">
              {{ t('mon.export', { name: datasetLabel(ds) }) }}
            </router-link>
          </span>
        </div>
        <p class="mon-hint">{{ t('mon.hint.' + s.key, hintArgs) }}</p>

        <div class="mon-tiles">
          <div v-for="x in s.tiles" :key="x.f" class="mon-tile">
            <div class="l">{{ x.label }}</div>
            <div class="v">{{ x.value }}</div>
          </div>
        </div>

        <div v-if="s.chart" class="mon-chart">
          <h4>{{ t('mon.chartTitle') }}</h4>
          <p v-if="!chart" class="cell-muted" style="font-size: 13px">{{ t('mon.noMonthly') }}</p>
          <template v-else>
            <div class="mon-legend">
              <span><i style="background: #1598B7"></i>{{ t('mon.chartSavings') }}</span>
              <span><i style="background: #F0913A"></i>{{ t('mon.chartLoans') }}</span>
            </div>
            <svg :viewBox="`0 0 ${chart.w} ${chart.h + 4}`" role="img" :aria-label="t('mon.chartTitle')" style="width: 100%; height: auto">
              <g v-for="tk in chart.ticks" :key="tk.y">
                <line :x1="chart.padL" :x2="chart.w" :y1="tk.y" :y2="tk.y" stroke="#E3E8ED" stroke-dasharray="4 4" />
                <text x="0" :y="tk.y + 4" font-size="11" fill="#8A94A0">{{ tk.label }}</text>
              </g>
              <g v-for="(b, i) in chart.bars" :key="b.label + i">
                <title>{{ b.title }}</title>
                <rect :x="b.a.x" :y="b.a.y" :width="chart.bw" :height="Math.max(0, b.a.h)" rx="2" fill="#1598B7" />
                <rect :x="b.b.x" :y="b.b.y" :width="chart.bw" :height="Math.max(0, b.b.h)" rx="2" fill="#F0913A" />
                <text v-if="i % chart.every === 0" :x="b.x" :y="chart.h - chart.padB + 16" font-size="11" fill="#8A94A0" text-anchor="middle">{{ b.label }}</text>
              </g>
            </svg>
          </template>
        </div>

        <div v-if="s.tables.length" class="mon-tables">
          <div v-for="tb in s.tables" :key="tb.key" class="mon-table">
            <h4>{{ t('mon.tb.' + tb.key) }}</h4>
            <div style="overflow-x: auto">
              <table class="dtable">
                <thead>
                  <tr>
                    <th v-for="c in tb.cols" :key="c" :style="LABEL_FIELDS.includes(c) ? '' : 'text-align: right'">{{ fieldLabel(c) }}</th>
                  </tr>
                </thead>
                <tbody>
                  <tr v-for="(r, i) in tb.rows" :key="i">
                    <td v-for="c in tb.cols" :key="c" :style="LABEL_FIELDS.includes(c) ? '' : 'text-align: right; white-space: nowrap'">{{ cell(c, r) }}</td>
                  </tr>
                </tbody>
              </table>
            </div>
          </div>
        </div>
      </section>
    </template>
  </div>
</template>

<style scoped>
.mon-filters {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(170px, 1fr));
  gap: 12px;
  margin-bottom: 14px;
}
.mon-filters .field-view {
  margin: 0;
}
.mon-filters .box {
  width: 100%;
}
.mon-period {
  font-size: 13px;
  color: var(--ink-600);
  margin: 0 0 16px;
  line-height: 1.5;
}
.mon-section {
  margin-bottom: 18px;
}
.mon-loading {
  opacity: 0.6;
}
.mon-head {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 10px;
  flex-wrap: wrap;
}
.mon-head h3 {
  font-size: 17px;
}
.mon-links {
  display: flex;
  gap: 6px;
  flex-wrap: wrap;
}
.mon-hint {
  font-size: 12.5px;
  color: var(--ink-600);
  margin: 6px 0 14px;
  line-height: 1.5;
}
.mon-tiles {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(170px, 1fr));
  gap: 10px;
}
.mon-tile {
  border: 1px solid var(--line);
  border-radius: 12px;
  padding: 10px 12px;
  min-width: 0;
}
.mon-tile .l {
  font-size: 11.5px;
  color: var(--ink-400);
  line-height: 1.3;
  margin-bottom: 4px;
}
.mon-tile .v {
  font-size: 17px;
  font-weight: 800;
  overflow-wrap: anywhere;
}
.mon-chart,
.mon-tables {
  margin-top: 16px;
}
.mon-tables {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(420px, 1fr));
  gap: 14px;
}
.mon-table,
.mon-chart {
  min-width: 0;
}
.mon-table h4,
.mon-chart h4 {
  font-size: 13.5px;
  margin-bottom: 6px;
}
.mon-legend {
  display: flex;
  gap: 16px;
  font-size: 12px;
  color: var(--ink-600);
  margin-bottom: 6px;
}
.mon-legend i {
  display: inline-block;
  width: 10px;
  height: 10px;
  border-radius: 2px;
  margin-right: 6px;
}
@media (max-width: 760px) {
  .mon-tiles {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
  .mon-tables {
    grid-template-columns: minmax(0, 1fr);
  }
  .mon-table th {
    white-space: normal;
  }
}
</style>
