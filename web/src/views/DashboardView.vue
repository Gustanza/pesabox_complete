<script setup>
import { useRouter } from 'vue-router'

const router = useRouter()

const kpiRow1 = [
  { l: 'Total Groups', v: '1,248', d: '+18 this month', dir: 'up' },
  { l: 'Active Groups', v: '1,103', d: '88% of total' },
  { l: 'Group Admins', v: '1,180', d: '' }
]
const kpiRow2 = [
  { l: 'Total Members', v: '28,450', d: '+640 this month', dir: 'up' },
  { l: 'Active Cycles', v: '1,092', d: '' },
  { l: 'Meetings This Month', v: '4,820', d: '' }
]
const kpiRow3 = [
  { l: 'Total Savings', v: 'TZS 1.24B', d: '+6.8%', dir: 'up' },
  { l: 'Total Shares', v: 'TZS 845M', d: '' },
  { l: 'Social Fund', v: 'TZS 185M', d: '' }
]
const kpiRow4 = [
  { l: 'Loans Outstanding', v: 'TZS 210M', d: '' },
  { l: 'Repayments', v: 'TZS 65M', d: 'this month' },
  { l: 'Transactions', v: '38,420', d: 'this month' }
]

const health = [
  { label: 'Healthy', value: 782, max: 900, color: 'var(--green-600)' },
  { label: 'Normal', value: 245, max: 900, color: 'var(--gold-500)' },
  { label: 'Needs Attention', value: 61, max: 900, color: '#F0A500' },
  { label: 'Inactive', value: 15, max: 900, color: 'var(--danger)' }
]

const alerts = [
  { icon: '\u26A0', text: '12 groups have not recorded a meeting in 30 days', action: 'View Groups', route: '/groups', bg: 'var(--danger-100)', fg: 'var(--danger)' },
  { icon: '\u26A0', text: 'SMS delivery rate below 90%', action: 'View SMS Logs', route: '/sms', bg: 'var(--gold-100)', fg: '#A87A1F' },
  { icon: '\u26A0', text: '5 admin accounts have repeated failed logins', action: 'View Security', route: '/settings', bg: 'var(--danger-100)', fg: 'var(--danger)' },
  { icon: '\u25CF', text: '24 meetings completed today', action: 'View Groups', route: '/groups', bg: 'var(--green-100)', fg: 'var(--green-600)' }
]

function barWidth(v, max) {
  return Math.round((v / max) * 100) + '%'
}
</script>

<template>
  <div>
    <div class="page-head">
      <div>
        <h1>Good morning, Raymond</h1>
        <p>Here's what's happening across PesaBox today. &middot; Last updated 07:52 AM</p>
      </div>
    </div>

    <div class="kpi-grid">
      <div v-for="k in kpiRow1" :key="k.l" class="kpi-card">
        <div class="l">{{ k.l }}</div>
        <div class="v">{{ k.v }}</div>
        <div v-if="k.d" class="d" :class="k.dir || ''">{{ k.d }}</div>
      </div>
    </div>
    <div class="kpi-grid">
      <div v-for="k in kpiRow2" :key="k.l" class="kpi-card">
        <div class="l">{{ k.l }}</div>
        <div class="v">{{ k.v }}</div>
        <div v-if="k.d" class="d" :class="k.dir || ''">{{ k.d }}</div>
      </div>
    </div>
    <div class="kpi-grid">
      <div v-for="k in kpiRow3" :key="k.l" class="kpi-card">
        <div class="l">{{ k.l }}</div>
        <div class="v">{{ k.v }}</div>
        <div v-if="k.d" class="d" :class="k.dir || ''">{{ k.d }}</div>
      </div>
    </div>
    <div class="kpi-grid">
      <div v-for="k in kpiRow4" :key="k.l" class="kpi-card">
        <div class="l">{{ k.l }}</div>
        <div class="v">{{ k.v }}</div>
        <div v-if="k.d" class="d" :class="k.dir || ''">{{ k.d }}</div>
      </div>
    </div>

    <div class="grid2">
      <div class="card">
        <div class="card-head"><h3>Group Health</h3></div>
        <div v-for="h in health" :key="h.label" class="chartbar-row">
          <div class="lbl">{{ h.label }}</div>
          <div class="bar"><div :style="{ width: barWidth(h.value, h.max), background: h.color }"></div></div>
          <div class="val">{{ h.value }}</div>
        </div>
      </div>
      <div class="card">
        <div class="card-head"><h3>Needs Attention</h3></div>
        <div v-for="a in alerts" :key="a.text" class="alert-row">
          <div class="ic" :style="{ background: a.bg, color: a.fg }">{{ a.icon }}</div>
          <div>
            <div class="t">{{ a.text }}</div>
            <button class="a" @click="router.push(a.route)">{{ a.action }} &rarr;</button>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>