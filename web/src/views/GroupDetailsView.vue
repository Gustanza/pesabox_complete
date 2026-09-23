<script setup>
import { computed, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { getGroup, updateGroup, deleteGroup } from '../api/groups.js'
import { listMembers } from '../api/members.js'
import { listMeetings, listAllAttendance } from '../api/meetings.js'
import { listTransactions } from '../api/transactions.js'

const route = useRoute()
const router = useRouter()

const group = ref(null)
const members = ref([])
const meetings = ref([])
const attendance = ref([])
const transactions = ref([])
const loading = ref(true)
const error = ref('')
const tab = ref('overview')
const saving = ref(false)

const tabs = ['Overview', 'Members', 'Financial Setup', 'Meetings', 'Cycles', 'Transactions']
const tabKeys = ['overview', 'members', 'financial', 'meetings', 'cycles', 'transactions']

async function load() {
  loading.value = true
  error.value = ''
  try {
    group.value = await getGroup(route.params.id)
    if (!group.value) {
      error.value = 'Group not found'
      return
    }

    const groupId = group.value.id
    const [membersList, meetingsList, attendanceList, transactionsList] = await Promise.all([
      listMembers(groupId),
      listMeetings(groupId),
      listAllAttendance(),
      listTransactions(groupId)
    ])
    members.value = membersList || []
    meetings.value = meetingsList || []
    attendance.value = attendanceList || []
    transactions.value = transactionsList || []
  } catch (e) {
    error.value = e.message || 'Failed to load group'
  } finally {
    loading.value = false
  }
}

onMounted(load)

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

const TX_TYPE_LABELS = {
  contribution: 'Saving',
  share: 'Share',
  social_fund: 'Social Fund',
  loan_disbursement: 'Loan',
  loan_repayment: 'Repayment',
  fine: 'Fine',
  expense: 'Expense',
  withdrawal: 'Withdrawal'
}

function txLabel(type) {
  return TX_TYPE_LABELS[type] || type
}

function formatDateTime(value) {
  const d = new Date(value)
  return isNaN(d) ? '—' : d.toLocaleString('en-GB', { day: 'numeric', month: 'short', hour: '2-digit', minute: '2-digit' })
}

function editGroup() {
  router.push('/groups/' + group.value.id + '/edit')
}

// ---- Status + delete ----
async function toggleStatus() {
  const nextStatus = group.value.status === 'Active' ? 'Closed' : 'Active'
  saving.value = true
  try {
    const result = await updateGroup(group.value.id, { status: nextStatus })
    if (result?.success) group.value = result.data
  } catch (e) {
    error.value = e.message || 'Failed to update status'
  } finally {
    saving.value = false
  }
}

async function removeGroup() {
  if (!confirm(`Delete "${group.value.name}"? This cannot be undone.`)) return
  try {
    await deleteGroup(group.value.id)
    router.push('/groups')
  } catch (e) {
    error.value = e.message || 'Failed to delete group'
  }
}

const ALL_SERVICES = ['Shares', 'Mandatory Savings', 'Voluntary Savings', 'Social Fund', 'Loans', 'Fines', 'Membership Fee']

const rules = computed(() => {
  if (!group.value) return []
  return [
    ['Mandatory Savings / meeting', 'TZS ' + Number(group.value.mandatorySavingsAmount || 0).toLocaleString()],
    ['Share value', 'TZS ' + Number(group.value.shareValue || 0).toLocaleString()],
    ['Min shares / meeting', String(group.value.minShares ?? '—')],
    ['Max shares / meeting', String(group.value.maxShares ?? '—')],
    ['Social Fund / meeting', 'TZS ' + Number(group.value.socialFundContribution || 0).toLocaleString()],
    ['Loan interest', (group.value.loanInterestRate ?? 0) + '%'],
    ['Max loan period', (group.value.maxLoanPeriodMonths ?? 0) + ' months']
  ]
})

const fineReasons = computed(() => {
  if (!group.value?.fineReasons) return []
  return group.value.fineReasons.filter((r) => r?.reason)
})

const sortedMeetings = computed(() =>
  [...meetings.value].sort((a, b) => (b.meetingNumber || 0) - (a.meetingNumber || 0))
)

const sortedTransactions = computed(() =>
  [...transactions.value].sort((a, b) => new Date(b.createdAt) - new Date(a.createdAt))
)
</script>

<template>
  <div v-if="loading" class="empty">
    <p>Loading group…</p>
  </div>
  <div v-else-if="error && !group" class="empty">
    <div class="ic">&#9888;</div>
    <p>{{ error }}</p>
  </div>
  <div v-else-if="group">
    <div class="crumb">
      <b @click="router.push('/groups')">Groups</b> / {{ group.name }}
    </div>
    <div class="page-head">
      <div>
        <h1>{{ group.name }}</h1>
        <p class="sub-row">
          <span class="badge" :class="group.status === 'Active' ? 'green' : 'grey'">{{ group.status }}</span>
        </p>
      </div>
      <div class="page-actions">
        <button class="btn btn-outline" @click="editGroup">Edit Group</button>
        <button class="btn btn-outline" :disabled="saving" @click="toggleStatus">
          {{ group.status === 'Active' ? 'Suspend Group' : 'Activate Group' }}
        </button>
        <button class="btn btn-danger" @click="removeGroup">Delete Group</button>
      </div>
    </div>

    <div v-if="error" style="color: var(--danger); font-size: 13px; margin-bottom: 12px">
      {{ error }}
    </div>

    <div class="dtabs">
      <button v-for="(t, i) in tabs" :key="t" class="dtab" :class="{ active: tab === tabKeys[i] }" @click="tab = tabKeys[i]">
        {{ t }}
      </button>
    </div>

    <!-- Overview -->
    <div v-if="tab === 'overview'" class="grid2">
      <div class="card">
        <div class="card-head">
          <h3>Group details</h3>
          <a href="#" @click.prevent="editGroup">Edit</a>
        </div>
        <div class="kv"><span class="k">Admin</span><span class="v">{{ group.adminName || '—' }}</span></div>
        <div class="kv"><span class="k">Admin phone</span><span class="v">{{ group.adminPhone || '—' }}</span></div>
        <div class="kv"><span class="k">Members</span><span class="v">{{ group.memberCount }} (F: {{ group.femaleMembers }}, M: {{ group.maleMembers }}, Youth: {{ group.youthMembers }})</span></div>
        <div class="kv"><span class="k">Location</span><span class="v">{{ [group.village, group.ward, group.district, group.region].filter(Boolean).join(', ') || '—' }}</span></div>
        <div class="kv"><span class="k">Meeting Frequency</span><span class="v">{{ group.meetingFrequency }}</span></div>
        <div class="kv"><span class="k">Current Cycle</span><span class="v">{{ group.cycleCurrent }} / {{ group.cycleTotal }}</span></div>
      </div>
      <div class="card">
        <div class="card-head"><h3>Financial Summary</h3></div>
        <div class="kv"><span class="k">Savings</span><span class="v">TZS {{ Number(group.totalSavings || 0).toLocaleString() }}</span></div>
        <div class="kv"><span class="k">Shares</span><span class="v">TZS {{ Number(group.totalShares || 0).toLocaleString() }}</span></div>
        <div class="kv"><span class="k">Social Fund</span><span class="v">TZS {{ Number(group.totalSocialFund || 0).toLocaleString() }}</span></div>
        <div class="kv"><span class="k">Loans Outstanding</span><span class="v">TZS {{ Number(group.totalLoans || 0).toLocaleString() }}</span></div>
        <p style="font-size: 11.5px; color: var(--ink-400); margin-top: 10px">
          Updated automatically once meetings and transactions are recorded.
        </p>
      </div>
    </div>

    <!-- Members -->
    <div v-else-if="tab === 'members'">
      <div v-if="!members.length" class="card empty">
        <div class="ic">&#128101;</div>
        <p>No members recorded yet for {{ group.name }}.</p>
      </div>
      <div v-else class="panel">
        <div style="overflow-x: auto">
          <table class="dtable">
            <thead>
              <tr><th>No.</th><th>Name</th><th>Phone</th><th>Status</th><th>Joined</th></tr>
            </thead>
            <tbody>
              <tr v-for="m in members" :key="m.id">
                <td class="cell-muted">{{ m.memberNumber || '—' }}</td>
                <td class="cell-strong">{{ [m.firstName, m.lastName].filter(Boolean).join(' ') || '—' }}</td>
                <td class="cell-muted">{{ m.phone || '—' }}</td>
                <td><span class="badge" :class="m.status === 'Active' ? 'green' : 'grey'">{{ m.status }}</span></td>
                <td class="cell-muted">{{ formatDateTime(m.joinedAt) }}</td>
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
        <div class="card-head"><h3>Group Constitution</h3></div>
        <p style="font-size: 12.5px; color: var(--ink-600); margin-bottom: 14px; line-height: 1.5">
          Set from the group's own app once it exists — shown here read-only for now.
        </p>
        <div class="grid3">
          <div v-for="r in rules" :key="r[0]" class="field-view">
            <label>{{ r[0] }}</label>
            <div class="box">{{ r[1] }}</div>
          </div>
        </div>
        <div class="card-head" style="margin-top: 8px"><h3>Fine reasons</h3></div>
        <div v-if="fineReasons.length" class="grid3">
          <div v-for="r in fineReasons" :key="r.reason" class="field-view">
            <label>{{ r.reason }}</label>
            <div class="box">TZS {{ Number(r.amount || 0).toLocaleString() }}</div>
          </div>
        </div>
        <p v-else style="font-size: 12px; color: var(--ink-400)">
          No fine reasons configured yet — the group uses the platform defaults.
        </p>
        <div class="card-head" style="margin-top: 8px"><h3>Enabled services</h3></div>
        <div class="toggle-line" v-for="s in ALL_SERVICES" :key="s">
          <span class="chk" :class="(group.enabledServices || []).includes(s) ? 'on' : 'off'">&#10003;</span>{{ s }}
        </div>
      </div>
    </div>

    <!-- Meetings -->
    <div v-else-if="tab === 'meetings'">
      <div v-if="!meetings.length" class="card empty">
        <div class="ic">&#128203;</div>
        <p>No meetings recorded yet for {{ group.name }}.</p>
      </div>
      <div v-else class="panel">
        <div style="overflow-x: auto">
          <table class="dtable">
            <thead>
              <tr><th>#</th><th>Title</th><th>Date</th><th>Status</th><th>Attendance</th></tr>
            </thead>
            <tbody>
              <tr v-for="m in sortedMeetings" :key="m.id">
                <td class="cell-muted">{{ m.meetingNumber }}</td>
                <td class="cell-strong">{{ m.title || '—' }}</td>
                <td class="cell-muted">{{ m.date }}</td>
                <td><span class="badge" :class="m.status === 'completed' ? 'green' : 'grey'">{{ m.status }}</span></td>
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
        <div class="card-head"><h3>Cycle progress</h3></div>
        <div class="kv"><span class="k">Current cycle</span><span class="v">{{ group.cycleCurrent }} / {{ group.cycleTotal }} meetings</span></div>
        <div class="kv"><span class="k">Meetings held</span><span class="v">{{ meetings.length }}</span></div>
        <div class="kv"><span class="k">Formation date</span><span class="v">{{ group.formationDate ? formatDateTime(group.formationDate) : '—' }}</span></div>
        <p style="font-size: 11.5px; color: var(--ink-400); margin-top: 10px">
          There's no separate cycle-history record in the schema yet — this reflects the group's live
          cycleCurrent/cycleTotal counters plus its recorded meetings, not a per-cycle archive.
        </p>
      </div>
    </div>

    <!-- Transactions -->
    <div v-else-if="tab === 'transactions'">
      <div v-if="!transactions.length" class="card empty">
        <div class="ic">&#128203;</div>
        <p>No transactions recorded yet for {{ group.name }}.</p>
      </div>
      <div v-else class="panel">
        <div style="overflow-x: auto">
          <table class="dtable">
            <thead>
              <tr><th>Date</th><th>Member</th><th>Type</th><th>Method</th><th>Amount</th></tr>
            </thead>
            <tbody>
              <tr v-for="t in sortedTransactions" :key="t.id">
                <td class="cell-muted">{{ formatDateTime(t.createdAt) }}</td>
                <td class="cell-strong">{{ memberName(t.memberId) }}</td>
                <td class="cell-muted">{{ txLabel(t.type) }}</td>
                <td class="cell-muted">{{ t.method || '—' }}</td>
                <td class="cell-strong" :style="{ color: t.direction === 'in' ? 'var(--green-600)' : 'var(--danger)' }">
                  {{ (t.direction === 'in' ? '+' : '-') + 'TZS ' + Number(t.amount || 0).toLocaleString() }}
                </td>
              </tr>
            </tbody>
          </table>
        </div>
        <div style="height: 16px"></div>
      </div>
    </div>
  </div>
</template>
