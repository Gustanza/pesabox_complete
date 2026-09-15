<script setup>
import { computed, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { getGroup, updateGroup, deleteGroup } from '../api/groups.js'

const route = useRoute()
const router = useRouter()

const group = ref(null)
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
    if (!group.value) error.value = 'Group not found'
  } catch (e) {
    error.value = e.message || 'Failed to load group'
  } finally {
    loading.value = false
  }
}

onMounted(load)

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
    ['Share value', 'TZS ' + Number(group.value.shareValue || 0).toLocaleString()],
    ['Min shares / meeting', String(group.value.minShares ?? '—')],
    ['Max shares / meeting', String(group.value.maxShares ?? '—')],
    ['Social Fund / meeting', 'TZS ' + Number(group.value.socialFundContribution || 0).toLocaleString()],
    ['Loan interest', (group.value.loanInterestRate ?? 0) + '%'],
    ['Max loan period', (group.value.maxLoanPeriodMonths ?? 0) + ' months'],
    ['Late meeting fine', 'TZS ' + Number(group.value.lateMeetingFine || 0).toLocaleString()],
    ['Absence fine', 'TZS ' + Number(group.value.absenceFine || 0).toLocaleString()],
    ['Late repayment fine', 'TZS ' + Number(group.value.lateLoanRepaymentFine || 0).toLocaleString()]
  ]
})
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
      <div class="card empty">
        <div class="ic">&#128101;</div>
        <p>Member management is coming soon. {{ group.memberCount }} member(s) are currently recorded for {{ group.name }}.</p>
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
        <div class="card-head" style="margin-top: 8px"><h3>Enabled services</h3></div>
        <div class="toggle-line" v-for="s in ALL_SERVICES" :key="s">
          <span class="chk" :class="(group.enabledServices || []).includes(s) ? 'on' : 'off'">&#10003;</span>{{ s }}
        </div>
      </div>
    </div>

    <!-- Meetings -->
    <div v-else-if="tab === 'meetings'">
      <div class="card empty">
        <div class="ic">&#128203;</div>
        <p>Meeting management is coming soon for {{ group.name }}.</p>
      </div>
    </div>

    <!-- Cycles -->
    <div v-else-if="tab === 'cycles'">
      <div class="card empty">
        <div class="ic">&#128203;</div>
        <p>Cycle tracking is coming soon. {{ group.name }} is on cycle {{ group.cycleCurrent }} of {{ group.cycleTotal }} meetings.</p>
      </div>
    </div>

    <!-- Transactions -->
    <div v-else-if="tab === 'transactions'">
      <div class="card empty">
        <div class="ic">&#128203;</div>
        <p>Transaction history is coming soon for {{ group.name }}.</p>
      </div>
    </div>
  </div>
</template>
