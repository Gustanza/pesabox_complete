<script setup>
import { ref } from 'vue'
import Svgs from '../components/Svgs.vue'

const activeTab = ref('dashboard')
const tabs = [
  ['Overview', 'dashboard'],
  ['Logs', 'logs'],
  ['Templates', 'templates']
]

const kpis = [
  { l: 'Sent Today', v: '4,820', d: '' },
  { l: 'Delivered', v: '4,650', d: '' },
  { l: 'Failed', v: '170', d: '', dir: 'down' },
  { l: 'Delivery Rate', v: '96.5%', d: '' }
]

const delivery = [
  { lbl: 'Delivered', v: 4650, max: 4820, color: 'var(--green-600)' },
  { lbl: 'Failed', v: 170, max: 4820, color: 'var(--danger)' }
]

const logs = [
  { time: '09:31', rec: '0712xxxxxx', group: 'Upendo', type: 'Contribution', status: 'Delivered' },
  { time: '09:32', rec: '0754xxxxxx', group: 'Upendo', type: 'Meeting', status: 'Delivered' },
  { time: '09:35', rec: '0788xxxxxx', group: 'Tumaini', type: 'Loan', status: 'Failed' }
]

const logSearch = ref('')
const logStatus = ref('All statuses')
const filteredLogs = ref([...logs])

function applyLogFilters() {
  filteredLogs.value = logs.filter(l => {
    const q = logSearch.value.trim().toLowerCase()
    const matchQ = !q || l.rec.toLowerCase().includes(q) || l.group.toLowerCase().includes(q) || l.type.toLowerCase().includes(q)
    const matchS = logStatus.value === 'All statuses' || l.status === logStatus.value
    return matchQ && matchS
  })
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
        <p>PESABOX is the only external integration in this MVP — treat it seriously.</p>
      </div>
    </div>

    <div class="dtabs">
      <button v-for="[label, key] in tabs" :key="key" class="dtab" :class="{ active: activeTab === key }" @click="activeTab = key">
        {{ label }}
      </button>
    </div>

    <template v-if="activeTab === 'logs'">
      <div class="filters">
        <div class="fsearch">
          <Svgs name="search" />
          <input v-model="logSearch" placeholder="Search recipient..." @input="applyLogFilters" />
        </div>
        <div class="fselect">
          <select v-model="logStatus" @change="applyLogFilters">
            <option>All statuses</option>
            <option>Delivered</option>
            <option>Failed</option>
          </select>
          <Svgs name="chev" />
        </div>
      </div>
      <div class="card" style="padding:6px 20px;">
        <table class="dtable">
          <thead>
            <tr><th>Time</th><th>Recipient</th><th>Group</th><th>Type</th><th>Status</th></tr>
          </thead>
          <tbody>
            <tr v-for="l in filteredLogs" :key="l.time">
              <td>{{ l.time }}</td>
              <td>{{ l.rec }}</td>
              <td>{{ l.group }}</td>
              <td>{{ l.type }}</td>
              <td><span class="badge" :class="l.status === 'Delivered' ? 'green' : 'red'">{{ l.status }}</span></td>
            </tr>
          </tbody>
        </table>
      </div>
    </template>

    <template v-else-if="activeTab === 'templates'">
      <div class="card" style="padding:6px 20px;">
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