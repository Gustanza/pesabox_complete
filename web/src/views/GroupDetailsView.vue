<script setup>
import { computed, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { intlLocale } from '../i18n'
import { getGroup, updateGroup, deleteGroup } from '../api/groups.js'

const { t, te } = useI18n()
const route = useRoute()
const router = useRouter()

const group = ref(null)
const loading = ref(true)
const error = ref('')
const tab = ref('overview')
const saving = ref(false)

const tabKeys = ['overview', 'members', 'financial', 'meetings', 'cycles', 'transactions']
const tabLabels = computed(() => [t('gd.tabOverview'), t('gd.tabMembers'), t('gd.tabFinancial'), t('gd.tabMeetings'), t('gd.tabCycles'), t('gd.tabTransactions')])
const statusText = (st) => (te('grp.st.' + st) ? t('grp.st.' + st) : st)
const svcText = (sv) => (te('gd.svc.' + sv) ? t('gd.svc.' + sv) : sv)
const num = (n) => Number(n || 0).toLocaleString(intlLocale())

async function load() {
  loading.value = true
  error.value = ''
  try {
    group.value = await getGroup(route.params.id)
    if (!group.value) error.value = t('gd.notFound')
  } catch (e) {
    error.value = e.message || t('gd.loadFailed')
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
    error.value = e.message || t('gd.statusFailed')
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

const fineReasons = computed(() => {
  if (!group.value?.fineReasons) return []
  return group.value.fineReasons.filter((r) => r?.reason)
})
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
        </p>
      </div>
      <div class="page-actions">
        <button class="btn btn-outline" @click="editGroup">{{ t('gd.editGroup') }}</button>
        <button class="btn btn-outline" :disabled="saving" @click="toggleStatus">
          {{ group.status === 'Active' ? t('gd.suspend') : t('gd.activate') }}
        </button>
        <button class="btn btn-danger" @click="removeGroup">{{ t('gd.deleteGroup') }}</button>
      </div>
    </div>

    <div v-if="error" style="color: var(--danger); font-size: 13px; margin-bottom: 12px">
      {{ error }}
    </div>

    <div class="dtabs">
      <button v-for="(label, i) in tabLabels" :key="tabKeys[i]" class="dtab" :class="{ active: tab === tabKeys[i] }" @click="tab = tabKeys[i]">
        {{ label }}
      </button>
    </div>

    <!-- Overview -->
    <div v-if="tab === 'overview'" class="grid2">
      <div class="card">
        <div class="card-head">
          <h3>{{ t('gd.details') }}</h3>
          <a href="#" @click.prevent="editGroup">{{ t('gd.edit') }}</a>
        </div>
        <div class="kv"><span class="k">{{ t('offc.admin') }}</span><span class="v">{{ group.adminName || '—' }}</span></div>
        <div class="kv"><span class="k">{{ t('offc.adminPhone') }}</span><span class="v">{{ group.adminPhone || '—' }}</span></div>
        <div class="kv"><span class="k">{{ t('gd.members') }}</span><span class="v">{{ t('gd.membersLine', { n: group.memberCount, f: group.femaleMembers, m: group.maleMembers, y: group.youthMembers }) }}</span></div>
        <div class="kv"><span class="k">{{ t('gd.location') }}</span><span class="v">{{ [group.village, group.ward, group.district, group.region].filter(Boolean).join(', ') || '—' }}</span></div>
        <div class="kv"><span class="k">{{ t('gd.frequency') }}</span><span class="v">{{ group.meetingFrequency }}</span></div>
        <div class="kv"><span class="k">{{ t('gd.currentCycle') }}</span><span class="v">{{ group.cycleCurrent }} / {{ group.cycleTotal }}</span></div>
      </div>
      <div class="card">
        <div class="card-head"><h3>{{ t('gd.finSummary') }}</h3></div>
        <div class="kv"><span class="k">{{ t('gd.savings') }}</span><span class="v">TZS {{ num(group.totalSavings) }}</span></div>
        <div class="kv"><span class="k">{{ t('gd.shares') }}</span><span class="v">TZS {{ num(group.totalShares) }}</span></div>
        <div class="kv"><span class="k">{{ t('gd.socialFund') }}</span><span class="v">TZS {{ num(group.totalSocialFund) }}</span></div>
        <div class="kv"><span class="k">{{ t('gd.loansOut') }}</span><span class="v">TZS {{ num(group.totalLoans) }}</span></div>
        <p style="font-size: 11.5px; color: var(--ink-400); margin-top: 10px">
          {{ t('gd.autoUpdated') }}
        </p>
      </div>
    </div>

    <!-- Members -->
    <div v-else-if="tab === 'members'">
      <div class="card empty">
        <div class="ic">&#128101;</div>
        <p>{{ t('gd.membersSoon', { n: group.memberCount, name: group.name }) }}</p>
      </div>
    </div>

    <!-- Financial -->
    <div v-else-if="tab === 'financial'">
      <div class="card">
        <div class="card-head"><h3>{{ t('gd.constitution') }}</h3></div>
        <p style="font-size: 12.5px; color: var(--ink-600); margin-bottom: 14px; line-height: 1.5">
          {{ t('gd.constitutionNote') }}
        </p>
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
        <p v-else style="font-size: 12px; color: var(--ink-400)">
          {{ t('gd.noFineReasons') }}
        </p>
        <div class="card-head" style="margin-top: 8px"><h3>{{ t('gd.enabledServices') }}</h3></div>
        <div class="toggle-line" v-for="s in ALL_SERVICES" :key="s">
          <span class="chk" :class="(group.enabledServices || []).includes(s) ? 'on' : 'off'">&#10003;</span>{{ svcText(s) }}
        </div>
      </div>
    </div>

    <!-- Meetings -->
    <div v-else-if="tab === 'meetings'">
      <div class="card empty">
        <div class="ic">&#128203;</div>
        <p>{{ t('gd.meetingsSoon', { name: group.name }) }}</p>
      </div>
    </div>

    <!-- Cycles -->
    <div v-else-if="tab === 'cycles'">
      <div class="card empty">
        <div class="ic">&#128203;</div>
        <p>{{ t('gd.cyclesSoon', { name: group.name, cur: group.cycleCurrent, total: group.cycleTotal }) }}</p>
      </div>
    </div>

    <!-- Transactions -->
    <div v-else-if="tab === 'transactions'">
      <div class="card empty">
        <div class="ic">&#128203;</div>
        <p>{{ t('gd.txSoon', { name: group.name }) }}</p>
      </div>
    </div>
  </div>
</template>
