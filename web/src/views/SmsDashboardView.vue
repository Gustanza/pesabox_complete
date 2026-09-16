<script setup>
import { computed, onMounted, ref } from 'vue'
import Svgs from '../components/Svgs.vue'
import { listSmsActivity } from '../api/sms.js'

const activeTab = ref('dashboard')
const tabs = [
  ['Overview', 'dashboard'],
  ['Logs', 'logs'],
  ['Templates', 'templates']
]

const logs = ref([])
const loading = ref(true)
const error = ref('')

const TYPE_LABELS = {
  member_otp: 'Member OTP',
  member_joined: 'Joined Group',
  contribution: 'Mandatory Savings',
  share: 'Shares',
  social_fund: 'Social Fund',
  loan_disbursement: 'Loan Disbursement',
  loan_repayment: 'Loan Repayment',
  fine: 'Fine Issued',
  fine_payment: 'Fine Payment',
  login_otp: 'Login OTP',
  other: 'Other'
}

function typeLabel(t) {
  return TYPE_LABELS[t] || t || 'Other'
}

onMounted(async () => {
  try {
    logs.value = await listSmsActivity()
  } catch (e) {
    error.value = e.message || 'Failed to load SMS activity'
  } finally {
    loading.value = false
  }
})

const kpis = computed(() => {
  const now = new Date()
  const today = logs.value.filter((l) => {
    const t = new Date(l.sentAt || l.createdAt)
    return !isNaN(t) && t.toDateString() === now.toDateString()
  })
  const failed = today.filter((l) => l.status === 'failed').length
  const delivered = today.length - failed
  const rate = today.length ? Math.round((delivered / today.length) * 1000) / 10 : 0
  return [
    { l: 'Sent Today', v: today.length.toLocaleString(), d: '' },
    { l: 'Delivered', v: delivered.toLocaleString(), d: '' },
    { l: 'Failed', v: failed.toLocaleString(), d: '' },
    { l: 'Delivery Rate', v: rate + '%', d: '' }
  ]
})

const delivery = computed(() => {
  const total = Math.max(1, logs.value.length)
  const delivered = logs.value.filter((l) => l.status === 'sent').length
  const failed = logs.value.filter((l) => l.status === 'failed').length
  return [
    { lbl: 'Delivered', v: delivered, max: total, color: 'var(--green-600)' },
    { lbl: 'Failed', v: failed, max: total, color: 'var(--danger)' }
  ]
})

const logSearch = ref('')
const logStatus = ref('All statuses')

function logStatusLabel(l) {
  return l.status === 'sent' ? 'Delivered' : 'Failed'
}

const filteredLogs = computed(() => {
  const q = logSearch.value.trim().toLowerCase()
  return logs.value.filter((l) => {
    const haystack = [l.phone, l.memberName, l.groupName, typeLabel(l.messageType)].filter(Boolean).join(' ').toLowerCase()
    const matchQ = !q || haystack.includes(q)
    const matchS = logStatus.value === 'All statuses' || logStatusLabel(l) === logStatus.value
    return matchQ && matchS
  })
})

function timeOf(l) {
  const t = l.sentAt || l.createdAt
  if (!t) return '—'
  const d = new Date(t)
  return isNaN(d) ? String(t) : d.toLocaleString()
}

function recipientOf(l) {
  return [l.memberName, l.phone].filter(Boolean).join(' · ') || '—'
}

const templates = [
  { name: 'Contribution Recorded', cat: 'Financial' },
  { name: 'Meeting Reminder', cat: 'Meeting' },
  { name: 'Loan Repayment', cat: 'Loan' },
  { name: 'Fine Notification', cat: 'Financial' },
  { name: 'Meeting Closed', cat: 'Meeting' }
]
const message = ref('Ndugu {JINA}, umeweka mchango wa TZS {KIASI} katika {TUKIO}.')

function barWidth(v, max) {
  return Math.round((v / max) * 100) + '%'
}
</script>

<template>
  <div>
    <div class="page-head">
      <div>
        <h1>SMS Management</h1>
        <p>Outgoing messages, delivery and templates.</p>
      </div>
    </div>

    <div class="dtabs">
      <button v-for="[label, key] in tabs" :key="key" class="dtab" :class="{ active: activeTab === key }" @click="activeTab = key">
        {{ label }}
      </button>
    </div>

    <template v-if="activeTab === 'logs'">
      <div class="panel">
        <div v-if="error" style="color: var(--danger); font-size: 13px; margin-bottom: 12px">
          {{ error }}
        </div>
        <div class="toolbar">
          <div class="search-input">
            <Svgs name="search" />
            <input v-model="logSearch" type="search" placeholder="Search recipient..." />
          </div>
          <select v-model="logStatus" class="filter-select">
            <option>All statuses</option>
            <option>Delivered</option>
            <option>Failed</option>
          </select>
        </div>
        <div v-if="loading" class="empty"><p>Loading…</p></div>
        <div v-else-if="!filteredLogs.length" class="empty">
          <div class="ic">&#128227;</div>
          <p>No member-facing SMS recorded yet.</p>
        </div>
        <div v-else style="overflow-x:auto;">
          <table class="dtable">
            <thead>
              <tr><th>Time</th><th>Recipient</th><th>Group</th><th>Type</th><th>Status</th></tr>
            </thead>
            <tbody>
              <tr v-for="(l, i) in filteredLogs" :key="l.id || i">
                <td>{{ timeOf(l) }}</td>
                <td>{{ recipientOf(l) }}</td>
                <td>{{ l.groupName || '—' }}</td>
                <td>{{ typeLabel(l.messageType) }}</td>
                <td><span class="badge" :class="l.status === 'sent' ? 'green' : 'red'">{{ logStatusLabel(l) }}</span></td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>
    </template>

    <template v-else-if="activeTab === 'templates'">
      <div class="panel">
        <table class="dtable">
          <thead>
            <tr><th>Template</th><th>Category</th><th>Status</th></tr>
          </thead>
          <tbody>
            <tr v-for="t in templates" :key="t.name">
              <td class="cell-main">{{ t.name }}</td>
              <td>{{ t.cat }}</td>
              <td><span class="badge green">Active</span></td>
            </tr>
          </tbody>
        </table>
      </div>
      <div class="card">
        <div class="card-head"><h3>Contribution Recorded</h3></div>
        <div class="field">
          <label>Message</label>
          <div class="inp" style="align-items:flex-start;padding:12px;">
            <textarea v-model="message" rows="3" style="border:none;outline:none;flex:1;font-family:'Inter';font-size:13px;background:transparent;resize:none;"></textarea>
          </div>
        </div>
        <div style="font-size:11.5px;color:var(--ink-400);margin-bottom:14px;">
          Variables: {JINA} {TUKIO} {KIASI} {SALIO} {TAREHE}
        </div>
        <button class="btn btn-outline">Save template</button>
      </div>
    </template>

    <template v-else>
      <div class="kpi-grid" style="grid-template-columns:repeat(4,1fr);">
        <div v-for="k in kpis" :key="k.l" class="kpi-card">
          <div class="l">{{ k.l }}</div>
          <div class="v">{{ k.v }}</div>
          <div v-if="k.d" class="d" :class="k.dir || ''">{{ k.d }}</div>
        </div>
      </div>
      <div class="card">
        <div class="card-head"><h3>Delivery</h3></div>
        <div v-for="d in delivery" :key="d.lbl" class="chartbar-row">
          <div class="lbl">{{ d.lbl }}</div>
          <div class="bar"><div :style="{ width: barWidth(d.v, d.max), background: d.color }"></div></div>
          <div class="val">{{ d.v }}</div>
        </div>
      </div>
    </template>
  </div>
</template>