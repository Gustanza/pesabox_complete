<script setup>
import { ref } from 'vue'
import { LOANS } from '../data/mock.js'

const activeTab = ref('savings')
const tabs = [
  ['Savings', 'savings'],
  ['Shares', 'shares'],
  ['Social Fund', 'social'],
  ['Loans', 'loans'],
  ['Fines', 'fines']
]

const kpis = {
  savings: ['Total Savings', 'TZS 1,245,000,000', '+ TZS 84,500,000'],
  shares: ['Total Shares', 'TZS 845,000,000', '+ TZS 32,100,000'],
  social: ['Social Fund Balance', 'TZS 185,000,000', '+ TZS 9,400,000'],
  fines: ['Total Fines Collected', 'TZS 21,300,000', '+ TZS 1,200,000']
}

const loansKpis = [
  { l: 'Active Loans', v: 'TZS 210M', d: '' },
  { l: 'Outstanding', v: 'TZS 145M', d: '' },
  { l: 'Paid This Month', v: 'TZS 65M', d: '' },
  { l: 'Overdue', v: 'TZS 18M', d: '', dir: 'down' }
]
</script>

<template>
  <div>
    <div class="page-head">
      <div>
        <h1>Finance</h1>
        <p>Platform-wide monitoring — not a place to edit group rules.</p>
      </div>
    </div>

    <div class="dtabs">
      <button v-for="[label, key] in tabs" :key="key" class="dtab" :class="{ active: activeTab === key }" @click="activeTab = key">
        {{ label }}
      </button>
    </div>

    <template v-if="activeTab === 'loans'">
      <div class="kpi-grid" style="grid-template-columns:repeat(4,1fr);">
        <div v-for="k in loansKpis" :key="k.l" class="kpi-card">
          <div class="l">{{ k.l }}</div>
          <div class="v">{{ k.v }}</div>
          <div v-if="k.d" class="d" :class="k.dir || ''">{{ k.d }}</div>
        </div>
      </div>
      <div class="card" style="padding:6px 20px;">
        <table class="dtable">
          <thead>
            <tr><th>Group</th><th>Borrower</th><th>Principal</th><th>Balance</th><th>Status</th></tr>
          </thead>
          <tbody>
            <tr v-for="(l, i) in LOANS" :key="l.group + l.borrower">
              <td class="cell-main">{{ l.group }}</td>
              <td>{{ l.borrower }}</td>
              <td>TZS {{ l.principal }}</td>
              <td>TZS {{ l.balance }}</td>
              <td>
                <span class="badge" :class="l.status === 'Overdue' ? 'red' : 'green'">{{ l.status }}</span>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </template>

    <template v-else>
      <div class="grid2">
        <div class="card">
          <div class="card-head"><h3>{{ kpis[activeTab][0] }}</h3></div>
          <div style="font-size:30px;font-weight:800;">{{ kpis[activeTab][1] }}</div>
          <div style="font-size:12.5px;color:var(--green-600);font-weight:700;margin-top:6px;">{{ kpis[activeTab][2] }} this month</div>
        </div>
        <div class="card">
          <div class="kv"><span class="k">Groups Contributing</span><span class="v">1,103</span></div>
          <div class="kv"><span class="k">Average / Group</span><span class="v">TZS 1,128,740</span></div>
        </div>
      </div>
    </template>
  </div>
</template>