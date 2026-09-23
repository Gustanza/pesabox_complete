<script setup>
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { LOANS } from '../data/mock.js'

const { t, te } = useI18n()

const activeTab = ref('savings')
const tabs = computed(() => [
  [t('fin.tabSavings'), 'savings'],
  [t('fin.tabShares'), 'shares'],
  [t('fin.tabSocial'), 'social'],
  [t('fin.tabLoans'), 'loans'],
  [t('fin.tabFines'), 'fines']
])
const statusText = (st) => (te('fin.st.' + st) ? t('fin.st.' + st) : st)

const kpis = {
  savings: ['fin.totalSavings', 'TZS 1,245,000,000', '+ TZS 84,500,000'],
  shares: ['fin.totalShares', 'TZS 845,000,000', '+ TZS 32,100,000'],
  social: ['fin.socialBalance', 'TZS 185,000,000', '+ TZS 9,400,000'],
  fines: ['fin.finesCollected', 'TZS 21,300,000', '+ TZS 1,200,000']
}

const loansKpis = [
  { l: 'fin.activeLoans', v: 'TZS 210M', d: '' },
  { l: 'fin.outstanding', v: 'TZS 145M', d: '' },
  { l: 'fin.paidMonth', v: 'TZS 65M', d: '' },
  { l: 'fin.overdue', v: 'TZS 18M', d: '', dir: 'down' }
]
</script>

<template>
  <div>
    <div class="page-head">
      <div>
        <h1>{{ t('fin.title') }}</h1>
        <p>{{ t('fin.subtitle') }}</p>
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
          <div class="l">{{ t(k.l) }}</div>
          <div class="v">{{ k.v }}</div>
          <div v-if="k.d" class="d" :class="k.dir || ''">{{ k.d }}</div>
        </div>
      </div>
      <div class="panel">
        <table class="dtable">
          <thead>
            <tr>
              <th>{{ t('fin.group') }}</th>
              <th>{{ t('fin.borrower') }}</th>
              <th>{{ t('fin.principal') }}</th>
              <th>{{ t('fin.balance') }}</th>
              <th>{{ t('common.status') }}</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="(l, i) in LOANS" :key="l.group + l.borrower">
              <td class="cell-main">{{ l.group }}</td>
              <td>{{ l.borrower }}</td>
              <td>TZS {{ l.principal }}</td>
              <td>TZS {{ l.balance }}</td>
              <td>
                <span class="badge" :class="l.status === 'Overdue' ? 'red' : 'green'">{{ statusText(l.status) }}</span>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </template>

    <template v-else>
      <div class="grid2">
        <div class="card">
          <div class="card-head"><h3>{{ t(kpis[activeTab][0]) }}</h3></div>
          <div style="font-size:30px;font-weight:800;">{{ kpis[activeTab][1] }}</div>
          <div style="font-size:12.5px;color:var(--green-600);font-weight:700;margin-top:6px;">{{ kpis[activeTab][2] }} {{ t('fin.thisMonth') }}</div>
        </div>
        <div class="card">
          <div class="kv"><span class="k">{{ t('fin.groupsContributing') }}</span><span class="v">1,103</span></div>
          <div class="kv"><span class="k">{{ t('fin.avgGroup') }}</span><span class="v">TZS 1,128,740</span></div>
        </div>
      </div>
    </template>
  </div>
</template>