<script setup>
import { computed, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { intlLocale } from '../i18n'
import { getGroup, updateGroup, deleteGroup } from '../api/groups.js'
import { listMembers } from '../api/members.js'
import { listMeetings, listAllAttendance } from '../api/meetings.js'
import { listTransactions } from '../api/transactions.js'
import { listClusters, moveGroup, listGovLoans } from '../api/admin.js'
import { can } from '../api/access.js'

const { t, te } = useI18n()
const route = useRoute()
const router = useRouter()

const group = ref(null)
const members = ref([])
const meetings = ref([])
const attendance = ref([])
const transactions = ref([])
const govLoans = ref([])
const clusters = ref([])
const loading = ref(true)
const error = ref('')
const tab = ref('overview')
const saving = ref(false)

const canEdit = computed(() => can('group.settings'))
const canMove = computed(() => can('groups.create'))

const tabKeys = ['overview', 'members', 'financial', 'meetings', 'cycles', 'transactions', 'govloans']
const tabLabel = (k) => ({
  overview: t('gd.tabOverview'),
  members: t('gd.tabMembers'),
  financial: t('gd.tabFinancial'),
  meetings: t('gd.tabMeetings'),
  cycles: t('gd.tabCycles'),
  transactions: t('gd.tabTransactions'),
  govloans: t('gl.tab')
})[k]
const statusText = (st) => (te('grp.st.' + st) ? t('grp.st.' + st) : st)
const svcText = (sv) => (te('gd.svc.' + sv) ? t('gd.svc.' + sv) : sv)
const meetingStatus = (st) => (te('gdx.meetingSt.' + st) ? t('gdx.meetingSt.' + st) : st)
const memberStatus = (st) => (te('mem.st.' + st) ? t('mem.st.' + st) : st)
const num = (n) => Number(n || 0).toLocaleString(intlLocale())

async function load() {
  loading.value = true
  error.value = ''
  try {
    group.value = await getGroup(route.params.id)
    if (!group.value) {
      error.value = t('gd.notFound')
      return
    }
    const groupId = group.value.id
    const [membersList, meetingsList, attendanceList, transactionsList, loans, clusterList] = await Promise.all([
      listMembers(groupId),
      listMeetings(groupId),
      listAllAttendance(),
      listTransactions(groupId),
      listGovLoans(groupId).catch(() => []),
      can('structure.view') ? listClusters().catch(() => []) : Promise.resolve([])
    ])
    members.value = membersList || []
    meetings.value = meetingsList || []
    attendance.value = attendanceList || []
    transactions.value = transactionsList || []
    govLoans.value = loans || []
    clusters.value = clusterList || []
  } catch (e) {
    error.value = e.message || t('gd.loadFailed')
  } finally {
    loading.value = false
  }
}

onMounted(load)

const cluster = computed(() => clusters.value.find((c) => c.id === group.value?.clusterId))

function memberName(id) {
  const m = members.value.find((x) => x.id === id)
  return m ? [m.firstName, m.lastName].filter(Boolean).join(' ') : '—'
}

// Present + late both count as "attended" for the summary shown per meeting.
function attendanceSummary(meetingId) {
  const rows = attendance.value.filter((a) => a.meetingId === meetingId)
  const attended = rows.filter((a) => a.status === 'present' || a.status === 'late').length
  return `${attended}/${members.value.length || rows.length}`
}

const txLabel = (type) => (te('gdx.tx.' + type) ? t('gdx.tx.' + type) : type)

function formatDateTime(value) {
  const d = new Date(value)
  return isNaN(d) ? '—' : d.toLocaleString(intlLocale(), { day: 'numeric', month: 'short', year: 'numeric', hour: '2-digit', minute: '2-digit' })
}
function formatDate(value) {
  const d = new Date(value)
  return isNaN(d) || !value ? '—' : d.toLocaleDateString(intlLocale(), { day: 'numeric', month: 'short', year: 'numeric' })
}

function editGroup() {
  router.push('/groups/' + group.value.id + '/edit')
}

// ---- Status, cluster, delete ----
async function toggleStatus() {
  const nextStatus = group.value.status === 'Active' ? 'Closed' : 'Active'
  saving.value = true
  try {
    const result = await updateGroup(group.value.id, { status: nextStatus })
    if (result?.success) group.value = result.data
    else error.value = result?.message || t('gd.statusFailed')
  } catch (e) {
    error.value = e.message || t('gd.statusFailed')
  } finally {
    saving.value = false
  }
}

async function changeCluster(event) {
  const clusterId = event.target.value
  if (!clusterId || clusterId === group.value.clusterId) return
  saving.value = true
  error.value = ''
  try {
    await moveGroup(group.value.id, clusterId)
    group.value.clusterId = clusterId
  } catch (e) {
    error.value = e.message || t('common.requestFailed')
  } finally {
    saving.value = false
  }
}

async function removeGroup() {
  if (!confirm(t('gd.confirmDelete', { name: group.value.name }))) return
  try {
    await deleteGroup(group.value.id)
    router.push('/groups')
  } catch (e) {
    // The server refuses when the group has financial records (TODO.md D5).
    error.value = e.message || t('gd.deleteFailed')
  }
}

const ALL_SERVICES = ['Shares', 'Mandatory Savings', 'Voluntary Savings', 'Social Fund', 'Loans', 'Fines', 'Membership Fee']

const rules = computed(() => {
  if (!group.value) return []
  return [
    [t('gd.ruleMandatory'), 'TZS ' + num(group.value.mandatorySavingsAmount)],
    [t('gd.ruleShareValue'), 'TZS ' + num(group.value.shareValue)],
    [t('gd.ruleMinShares'), String(group.value.minShares ?? '—')],
    [t('gd.ruleMaxShares'), String(group.value.maxShares ?? '—')],
    [t('gd.ruleSocial'), 'TZS ' + num(group.value.socialFundContribution)],
    [t('gd.ruleInterest'), (group.value.loanInterestRate ?? 0) + '%'],
    [t('gd.ruleMaxPeriod'), t('gd.monthsN', { n: group.value.maxLoanPeriodMonths ?? 0 })]
  ]
})

const fineReasons = computed(() => (group.value?.fineReasons || []).filter((r) => r?.reason))
const sortedMeetings = computed(() => [...meetings.value].sort((a, b) => (b.meetingNumber || 0) - (a.meetingNumber || 0)))
const sortedTransactions = computed(() => [...transactions.value].sort((a, b) => new Date(b.createdAt) - new Date(a.createdAt)))
const govOutstanding = computed(() => govLoans.value.reduce((s, l) => s + (l.outstanding || 0), 0))
</script>

<template>
  <div v-if="loading" class="empty">
    <p>{{ t('gd.loading') }}</p>
  </div>
  <div v-else-if="error && !group" class="empty">
    <div class="ic">&#9888;</div>
    <p>{{ error }}</p>
  </div>
  <div v-else-if="group">
    <div class="crumb">
      <b @click="router.push('/groups')">{{ t('gd.groups') }}</b> / {{ group.name }}
    </div>
    <div class="page-head">
      <div>
        <h1>{{ group.name }}</h1>
        <p class="sub-row">
          <span class="badge" :class="group.status === 'Active' ? 'green' : 'grey'">{{ statusText(group.status) }}</span>
          <span v-if="cluster" class="cell-muted" style="font-size: 13px">{{ cluster.partnerName }} › {{ cluster.name }}</span>
        </p>
      </div>
      <div v-if="canEdit" class="page-actions">
        <button class="btn btn-outline" @click="editGroup">{{ t('gd.editGroup') }}</button>
        <button class="btn btn-outline" :disabled="saving" @click="toggleStatus">
          {{ group.status === 'Active' ? t('gdx.close') : t('gd.activate') }}
        </button>
        <button class="btn btn-danger" @click="removeGroup">{{ t('gd.deleteGroup') }}</button>
      </div>
    </div>

    <div v-if="error" style="color: var(--danger); font-size: 13px; margin-bottom: 12px">{{ error }}</div>

    <div class="dtabs">
      <button v-for="k in tabKeys" :key="k" class="dtab" :class="{ active: tab === k }" @click="tab = k">
        {{ tabLabel(k) }}
      </button>
    </div>

    <!-- Overview -->
    <div v-if="tab === 'overview'" class="grid2">
      <div class="card">
        <div class="card-head">
          <h3>{{ t('gd.details') }}</h3>
          <a v-if="canEdit" href="#" @click.prevent="editGroup">{{ t('gd.edit') }}</a>
        </div>
        <div class="kv"><span class="k">{{ t('offc.admin') }}</span><span class="v">{{ group.adminName || '—' }}</span></div>
        <div class="kv"><span class="k">{{ t('offc.adminPhone') }}</span><span class="v">{{ group.adminPhone || '—' }}</span></div>
        <div class="kv"><span class="k">{{ t('gd.members') }}</span><span class="v">{{ t('gd.membersLine', { n: group.memberCount, f: group.femaleMembers, m: group.maleMembers, y: group.youthMembers }) }}</span></div>
        <div class="kv"><span class="k">{{ t('gd.location') }}</span><span class="v">{{ [group.village, group.ward, group.district, group.region].filter(Boolean).join(', ') || '—' }}</span></div>
        <div class="kv"><span class="k">{{ t('gd.frequency') }}</span><span class="v">{{ group.meetingFrequency }}</span></div>
        <div class="kv"><span class="k">{{ t('gd.currentCycle') }}</span><span class="v">{{ group.cycleCurrent }} / {{ group.cycleTotal }}</span></div>
        <div class="kv">
          <span class="k">{{ t('struct.cluster') }}</span>
          <span class="v">
            <select v-if="canMove && clusters.length" :value="group.clusterId" class="role-select" :disabled="saving" @change="changeCluster">
              <option v-for="c in clusters" :key="c.id" :value="c.id">{{ c.name }} — {{ c.partnerName }}</option>
            </select>
            <template v-else>{{ cluster ? cluster.name : '—' }}</template>
          </span>
        </div>
      </div>
      <div class="card">
        <div class="card-head"><h3>{{ t('gd.finSummary') }}</h3></div>
        <div class="kv"><span class="k">{{ t('gd.savings') }}</span><span class="v">TZS {{ num(group.totalSavings) }}</span></div>
        <div class="kv"><span class="k">{{ t('gd.shares') }}</span><span class="v">TZS {{ num(group.totalShares) }}</span></div>
        <div class="kv"><span class="k">{{ t('gd.socialFund') }}</span><span class="v">TZS {{ num(group.totalSocialFund) }}</span></div>
        <div class="kv"><span class="k">{{ t('gd.loansOut') }}</span><span class="v">TZS {{ num(group.totalLoans) }}</span></div>
        <div class="kv"><span class="k">{{ t('gl.outstanding') }}</span><span class="v">TZS {{ num(govOutstanding) }}</span></div>
        <p style="font-size: 11.5px; color: var(--ink-400); margin-top: 10px">{{ t('gd.autoUpdated') }}</p>
      </div>
    </div>

    <!-- Members -->
    <div v-else-if="tab === 'members'">
      <div v-if="!members.length" class="card empty">
        <div class="ic">&#128101;</div>
        <p>{{ t('gdx.noMembers', { name: group.name }) }}</p>
      </div>
      <div v-else class="panel">
        <div style="overflow-x: auto">
          <table class="dtable">
            <thead>
              <tr><th>{{ t('gdx.no') }}</th><th>{{ t('users.name') }}</th><th>{{ t('struct.phone') }}</th><th>{{ t('common.status') }}</th><th>{{ t('gdx.joined') }}</th></tr>
            </thead>
            <tbody>
              <tr v-for="m in members" :key="m.id">
                <td class="cell-muted">{{ m.memberNumber || '—' }}</td>
                <td class="cell-strong">{{ [m.firstName, m.lastName].filter(Boolean).join(' ') || '—' }}</td>
                <td class="cell-muted">{{ m.phone || '—' }}</td>
                <td><span class="badge" :class="m.status === 'Active' ? 'green' : 'grey'">{{ memberStatus(m.status) }}</span></td>
                <td class="cell-muted">{{ formatDate(m.joinedAt) }}</td>
              </tr>
            </tbody>
          </table>
        </div>
        <div style="height: 16px"></div>
      </div>
    </div>

    <!-- Financial -->
    <div v-else-if="tab === 'financial'">
      <div class="card">
        <div class="card-head"><h3>{{ t('gd.constitution') }}</h3></div>
        <p style="font-size: 12.5px; color: var(--ink-600); margin-bottom: 14px; line-height: 1.5">{{ t('gd.constitutionNote') }}</p>
        <div class="grid3">
          <div v-for="r in rules" :key="r[0]" class="field-view">
            <label>{{ r[0] }}</label>
            <div class="box">{{ r[1] }}</div>
          </div>
        </div>
        <div class="card-head" style="margin-top: 8px"><h3>{{ t('gd.fineReasonsTitle') }}</h3></div>
        <div v-if="fineReasons.length" class="grid3">
          <div v-for="r in fineReasons" :key="r.reason" class="field-view">
            <label>{{ r.reason }}</label>
            <div class="box">TZS {{ num(r.amount) }}</div>
          </div>
        </div>
        <p v-else style="font-size: 12px; color: var(--ink-400)">{{ t('gd.noFineReasons') }}</p>
        <div class="card-head" style="margin-top: 8px"><h3>{{ t('gd.enabledServices') }}</h3></div>
        <div v-for="s in ALL_SERVICES" :key="s" class="toggle-line">
          <span class="chk" :class="(group.enabledServices || []).includes(s) ? 'on' : 'off'">&#10003;</span>{{ svcText(s) }}
        </div>
      </div>
    </div>

    <!-- Meetings -->
    <div v-else-if="tab === 'meetings'">
      <div v-if="!meetings.length" class="card empty">
        <div class="ic">&#128203;</div>
        <p>{{ t('gdx.noMeetings', { name: group.name }) }}</p>
      </div>
      <div v-else class="panel">
        <div style="overflow-x: auto">
          <table class="dtable">
            <thead>
              <tr><th>#</th><th>{{ t('gdx.title') }}</th><th>{{ t('gdx.date') }}</th><th>{{ t('common.status') }}</th><th>{{ t('gdx.attendance') }}</th></tr>
            </thead>
            <tbody>
              <tr v-for="m in sortedMeetings" :key="m.id">
                <td class="cell-muted">{{ m.meetingNumber }}</td>
                <td class="cell-strong">{{ m.title || '—' }}</td>
                <td class="cell-muted">{{ m.date }}</td>
                <td><span class="badge" :class="m.status === 'completed' ? 'green' : 'grey'">{{ meetingStatus(m.status) }}</span></td>
                <td class="cell-muted">{{ attendanceSummary(m.id) }}</td>
              </tr>
            </tbody>
          </table>
        </div>
        <div style="height: 16px"></div>
      </div>
    </div>

    <!-- Cycles -->
    <div v-else-if="tab === 'cycles'">
      <div class="card">
        <div class="card-head"><h3>{{ t('gdx.cycleProgress') }}</h3></div>
        <div class="kv"><span class="k">{{ t('gd.currentCycle') }}</span><span class="v">{{ t('gdx.cycleOf', { n: group.cycleCurrent, total: group.cycleTotal }) }}</span></div>
        <div class="kv"><span class="k">{{ t('gdx.meetingsHeld') }}</span><span class="v">{{ meetings.length }}</span></div>
        <div class="kv"><span class="k">{{ t('cg.formation') }}</span><span class="v">{{ formatDate(group.formationDate) }}</span></div>
        <p style="font-size: 11.5px; color: var(--ink-400); margin-top: 10px">{{ t('gdx.cycleNote') }}</p>
      </div>
    </div>

    <!-- Transactions -->
    <div v-else-if="tab === 'transactions'">
      <div v-if="!transactions.length" class="card empty">
        <div class="ic">&#128203;</div>
        <p>{{ t('gdx.noTransactions', { name: group.name }) }}</p>
      </div>
      <div v-else class="panel">
        <div style="overflow-x: auto">
          <table class="dtable">
            <thead>
              <tr><th>{{ t('gdx.date') }}</th><th>{{ t('dash.member') }}</th><th>{{ t('dash.type') }}</th><th>{{ t('gdx.method') }}</th><th>{{ t('dash.amount') }}</th></tr>
            </thead>
            <tbody>
              <tr v-for="tx in sortedTransactions" :key="tx.id" :style="tx.reversed ? 'opacity: .6' : ''">
                <td class="cell-muted">{{ formatDateTime(tx.createdAt) }}</td>
                <td class="cell-strong">{{ memberName(tx.memberId) }}</td>
                <td class="cell-muted">
                  {{ txLabel(tx.type) }}
                  <span v-if="tx.reversed" class="badge gold" :title="tx.reversalReason || ''">{{ t('dash.reversed') }}</span>
                  <div v-if="tx.reversed && tx.reversalReason" class="cell-sub">{{ tx.reversalReason }}</div>
                </td>
                <td class="cell-muted">{{ tx.method || '—' }}</td>
                <td class="cell-strong" :style="{ color: tx.direction === 'in' ? 'var(--green-600)' : 'var(--danger)', textDecoration: tx.reversed ? 'line-through' : '' }">
                  {{ (tx.direction === 'in' ? '+' : '-') + 'TZS ' + num(tx.amount) }}
                </td>
              </tr>
            </tbody>
          </table>
        </div>
        <div style="height: 16px"></div>
      </div>
    </div>

    <!-- Government loans -->
    <div v-else-if="tab === 'govloans'">
      <p style="font-size: 12.5px; color: var(--ink-600); margin-bottom: 14px; line-height: 1.5">{{ t('gl.note') }}</p>
      <div v-if="!govLoans.length" class="card empty">
        <div class="ic">&#127963;</div>
        <p>{{ t('gl.empty') }}</p>
      </div>
      <div v-else class="panel">
        <div style="overflow-x: auto">
          <table class="dtable">
            <thead>
              <tr>
                <th>{{ t('gl.lender') }}</th>
                <th>{{ t('gl.received') }}</th>
                <th>{{ t('gl.principal') }}</th>
                <th>{{ t('gl.interest') }}</th>
                <th>{{ t('gl.repaid') }}</th>
                <th>{{ t('gl.outstanding') }}</th>
                <th>{{ t('gl.due') }}</th>
                <th>{{ t('common.status') }}</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="l in govLoans" :key="l.id">
                <td>
                  <div class="cell-main">{{ l.lender }}</div>
                  <div class="cell-sub">{{ [l.programme, l.reference].filter(Boolean).join(' · ') }}</div>
                </td>
                <td class="cell-muted">{{ formatDate(l.receivedDate) }}</td>
                <td>TZS {{ num(l.amount) }}</td>
                <td>{{ l.interestRate || 0 }}%</td>
                <td>TZS {{ num(l.amountRepaid) }}</td>
                <td class="cell-strong">TZS {{ num(l.outstanding) }}</td>
                <td class="cell-muted" :style="l.overdue ? 'color: var(--danger); font-weight: 700' : ''">{{ formatDate(l.dueDate) }}</td>
                <td><span class="badge" :class="l.status === 'repaid' ? 'green' : l.overdue ? 'red' : 'gold'">{{ l.overdue ? t('gl.overdue') : t('gl.st.' + l.status) }}</span></td>
              </tr>
            </tbody>
          </table>
        </div>
        <div style="height: 16px"></div>
      </div>
    </div>
  </div>
</template>
