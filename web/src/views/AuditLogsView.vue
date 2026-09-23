<script setup>
import { computed, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import Svgs from '../components/Svgs.vue'
import { intlLocale } from '../i18n'
import { listAudit } from '@/api/admin'

// The append-only audit trail (server: AuditLog model, written by every
// create / edit / reversal / (de)activation / assignment). Nobody can edit or
// delete entries — TODO.md D5 / U12.

const { t, te } = useI18n()

const rows = ref([])
const loading = ref(true)
const error = ref('')
const search = ref('')
const action = ref('')
const model = ref('')
const from = ref('')
const to = ref('')

const ACTIONS = ['create', 'update', 'reverse', 'delete', 'deactivate', 'reactivate', 'assign', 'unassign']
const MODELS = ['Transaction', 'Loan', 'Fine', 'GovernmentLoan', 'GovernmentLoanRepayment', 'Member', 'Meeting', 'MeetingAttendance', 'Group', 'Cluster', 'Partner', 'User', 'Assignment', 'Announcement']

async function load() {
  loading.value = true
  error.value = ''
  try {
    rows.value = await listAudit({ action: action.value, model: model.value, from: from.value, to: to.value, limit: 1000 })
  } catch (e) {
    error.value = e.message || t('audit.loadFailed')
  } finally {
    loading.value = false
  }
}
onMounted(load)
watch([action, model, from, to], load)

const filtered = computed(() => {
  const q = search.value.trim().toLowerCase()
  if (!q) return rows.value
  return rows.value.filter((a) =>
    [a.summary, a.userName, a.groupName, a.model].some((v) => (v || '').toLowerCase().includes(q))
  )
})

const actionText = (a) => (te('aud.actions.' + a) ? t('aud.actions.' + a) : a)
const modelText = (m) => (te('aud.models.' + m) ? t('aud.models.' + m) : m)
const roleText = (r) => (r && te('roles.' + r) ? t('roles.' + r) : '')
const when = (v) => {
  const d = new Date(v)
  return isNaN(d) ? '—' : d.toLocaleString(intlLocale(), { day: 'numeric', month: 'short', year: 'numeric', hour: '2-digit', minute: '2-digit' })
}
const actionClass = (a) => ({ reverse: 'gold', delete: 'red', deactivate: 'grey', reactivate: 'green', create: 'green' })[a] || ''
</script>

<template>
  <div>
    <div class="page-head">
      <div>
        <h1>{{ t('audit.title') }}</h1>
        <p>{{ t('aud.subtitle') }}</p>
      </div>
    </div>

    <div v-if="error" style="color: var(--danger); font-size: 13px; margin-bottom: 12px">{{ error }}</div>

    <div class="panel">
      <div class="toolbar" style="flex-wrap: wrap">
        <div class="search-input">
          <Svgs name="search" />
          <input v-model="search" type="search" :placeholder="t('audit.search')" />
        </div>
        <select v-model="action" class="filter-select">
          <option value="">{{ t('aud.allActions') }}</option>
          <option v-for="a in ACTIONS" :key="a" :value="a">{{ actionText(a) }}</option>
        </select>
        <select v-model="model" class="filter-select">
          <option value="">{{ t('aud.allRecords') }}</option>
          <option v-for="m in MODELS" :key="m" :value="m">{{ modelText(m) }}</option>
        </select>
        <input v-model="from" type="date" class="filter-select" :title="t('aud.from')" />
        <input v-model="to" type="date" class="filter-select" :title="t('aud.to')" />
      </div>
      <div v-if="loading" class="empty"><p>{{ t('common.loading') }}</p></div>
      <div v-else-if="!filtered.length" class="empty"><div class="ic">&#128220;</div><p>{{ t('audit.empty') }}</p></div>
      <div v-else style="overflow-x: auto">
        <table class="dtable">
          <thead>
            <tr>
              <th>{{ t('audit.time') }}</th>
              <th>{{ t('audit.user') }}</th>
              <th>{{ t('audit.action') }}</th>
              <th>{{ t('audit.resource') }}</th>
              <th>{{ t('aud.details') }}</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="a in filtered" :key="a.id">
              <td class="cell-muted" style="white-space: nowrap">{{ when(a.createdAt) }}</td>
              <td>
                <div class="cell-main">{{ a.userName || '—' }}</div>
                <div class="cell-sub">{{ roleText(a.role) }}</div>
              </td>
              <td><span class="badge" :class="actionClass(a.action)">{{ actionText(a.action) }}</span></td>
              <td>
                <div class="cell-main">{{ modelText(a.model) }}</div>
                <div class="cell-sub">{{ a.groupName || '' }}</div>
              </td>
              <td class="cell-muted">{{ a.summary }}</td>
            </tr>
          </tbody>
        </table>
      </div>
      <p v-if="filtered.length >= 1000" style="font-size: 12px; color: var(--ink-400); padding: 10px 16px">{{ t('aud.capped') }}</p>
      <div style="height: 16px"></div>
    </div>
  </div>
</template>
