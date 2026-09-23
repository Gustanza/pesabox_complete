<script setup>
import { computed, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import Svgs from '../components/Svgs.vue'
import { intlLocale } from '../i18n'
import { listSmsActivity, listSmsTemplates, saveSmsTemplate } from '../api/sms.js'

const { t, te, locale } = useI18n()

const activeTab = ref('dashboard')
const tabs = computed(() => [
  [t('sms.tabOverview'), 'dashboard'],
  [t('sms.tabLogs'), 'logs'],
  [t('sms.tabTemplates'), 'templates']
])

const logs = ref([])
const loading = ref(true)
const error = ref('')

function typeLabel(type) {
  const key = 'smsd.types.' + (type || 'other')
  return te(key) ? t(key) : type || t('smsd.types.other')
}

onMounted(async () => {
  try {
    logs.value = await listSmsActivity()
  } catch (e) {
    error.value = e.message || t('smsd.loadFailed')
  } finally {
    loading.value = false
  }
})

const kpis = computed(() => {
  const now = new Date()
  const today = logs.value.filter((l) => {
    const d = new Date(l.sentAt || l.createdAt)
    return !isNaN(d) && d.toDateString() === now.toDateString()
  })
  const failed = today.filter((l) => l.status === 'failed').length
  const delivered = today.length - failed
  const rate = today.length ? Math.round((delivered / today.length) * 1000) / 10 : 0
  return [
    { l: t('smsd.kpiSentToday'), v: today.length.toLocaleString(intlLocale()), d: '' },
    { l: t('smsd.kpiDelivered'), v: delivered.toLocaleString(intlLocale()), d: '' },
    { l: t('smsd.kpiFailed'), v: failed.toLocaleString(intlLocale()), d: '' },
    { l: t('smsd.kpiRate'), v: rate + '%', d: '' }
  ]
})

const delivery = computed(() => {
  const total = Math.max(1, logs.value.length)
  const delivered = logs.value.filter((l) => l.status === 'sent').length
  const failed = logs.value.filter((l) => l.status === 'failed').length
  return [
    { lbl: t('smsd.kpiDelivered'), v: delivered, max: total, color: 'var(--green-600)' },
    { lbl: t('smsd.kpiFailed'), v: failed, max: total, color: 'var(--danger)' }
  ]
})

const logSearch = ref('')
const logStatus = ref('all') // 'all' | 'sent' | 'failed' — stable values, not display text

function logStatusLabel(l) {
  return l.status === 'sent' ? t('smsd.kpiDelivered') : t('smsd.kpiFailed')
}

const filteredLogs = computed(() => {
  const q = logSearch.value.trim().toLowerCase()
  return logs.value.filter((l) => {
    const haystack = [l.phone, l.memberName, l.groupName, typeLabel(l.messageType)].filter(Boolean).join(' ').toLowerCase()
    const matchQ = !q || haystack.includes(q)
    const matchS = logStatus.value === 'all' || l.status === logStatus.value
    return matchQ && matchS
  })
})

function timeOf(l) {
  const ts = l.sentAt || l.createdAt
  if (!ts) return '—'
  const d = new Date(ts)
  return isNaN(d) ? String(ts) : d.toLocaleString(intlLocale())
}

function recipientOf(l) {
  return [l.memberName, l.phone].filter(Boolean).join(' · ') || '—'
}

// ---- Templates -------------------------------------------------------------
const templates = ref([])
const templatesLoaded = ref(false)
const tplLang = ref(locale.value) // language of the message being edited
const selectedType = ref('')
const draft = ref('')
const saving = ref(false)
const saveMsg = ref('')
const saveErr = ref('')

async function loadTemplates() {
  try {
    templates.value = await listSmsTemplates()
    templatesLoaded.value = true
    if (!selectedType.value && templates.value.length) selectedType.value = templates.value[0].type
  } catch (e) {
    saveErr.value = e.message || t('common.requestFailed')
  }
}

watch(activeTab, (tab) => {
  if (tab === 'templates' && !templatesLoaded.value) loadTemplates()
})

const rows = computed(() => templates.value.filter((x) => x.language === tplLang.value))
const current = computed(() => rows.value.find((x) => x.type === selectedType.value) || null)

watch([current], () => {
  draft.value = current.value ? current.value.body : ''
  saveMsg.value = ''
  saveErr.value = ''
})

function tplName(x) {
  return locale.value === 'sw' ? x.sw : x.en
}

function insertVar(v) {
  draft.value += (draft.value && !draft.value.endsWith(' ') ? ' ' : '') + '{' + v + '}'
}

async function save(restore = false) {
  if (!current.value) return
  saving.value = true
  saveMsg.value = ''
  saveErr.value = ''
  try {
    await saveSmsTemplate({
      type: current.value.type,
      language: tplLang.value,
      body: restore ? '' : draft.value
    })
    await loadTemplates()
    saveMsg.value = restore ? t('sms.restoreDefault') + ' ✓' : t('sms.saved')
  } catch (e) {
    saveErr.value = /only the super admin/i.test(e.message || '') ? t('sms.onlySuperAdmin') : e.message || t('sms.templateSaveFailed')
  } finally {
    saving.value = false
  }
}

function barWidth(v, max) {
  return Math.round((v / max) * 100) + '%'
}
</script>

<template>
  <div>
    <div class="page-head">
      <div>
        <h1>{{ t('sms.title') }}</h1>
        <p>{{ t('sms.subtitle') }}</p>
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
            <input v-model="logSearch" type="search" :placeholder="t('smsd.searchRecipient')" />
          </div>
          <select v-model="logStatus" class="filter-select">
            <option value="all">{{ t('smsd.allStatuses') }}</option>
            <option value="sent">{{ t('smsd.kpiDelivered') }}</option>
            <option value="failed">{{ t('smsd.kpiFailed') }}</option>
          </select>
        </div>
        <div v-if="loading" class="empty"><p>{{ t('common.loading') }}</p></div>
        <div v-else-if="!filteredLogs.length" class="empty">
          <div class="ic">&#128227;</div>
          <p>{{ t('smsd.noneRecorded') }}</p>
        </div>
        <div v-else style="overflow-x:auto;">
          <table class="dtable">
            <thead>
              <tr>
                <th>{{ t('smsd.time') }}</th>
                <th>{{ t('smsd.recipient') }}</th>
                <th>{{ t('smsd.group') }}</th>
                <th>{{ t('smsd.type') }}</th>
                <th>{{ t('common.status') }}</th>
              </tr>
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
      <div style="display: flex; gap: 8px; margin-bottom: 14px">
        <button
          v-for="[code, label] in [['sw', t('sms.langSw')], ['en', t('sms.langEn')]]"
          :key="code"
          class="btn btn-sm"
          :class="tplLang === code ? 'btn-primary' : 'btn-outline'"
          @click="tplLang = code"
        >{{ label }}</button>
      </div>
      <div class="panel">
        <div v-if="!templatesLoaded && !saveErr" class="empty"><p>{{ t('common.loading') }}</p></div>
        <table v-else class="dtable">
          <thead>
            <tr><th>{{ t('sms.template') }}</th><th>{{ t('sms.category') }}</th><th>{{ t('common.status') }}</th></tr>
          </thead>
          <tbody>
            <tr
              v-for="x in rows"
              :key="x.type"
              style="cursor: pointer"
              :style="x.type === selectedType ? 'background: var(--green-100)' : ''"
              @click="selectedType = x.type"
            >
              <td class="cell-main">{{ tplName(x) }}</td>
              <td>{{ x.category }}</td>
              <td>
                <span class="badge" :class="x.custom ? 'gold' : 'green'">{{ x.custom ? t('sms.custom') : t('sms.builtin') }}</span>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
      <div v-if="current" class="card">
        <div class="card-head"><h3>{{ tplName(current) }}</h3></div>
        <div class="field">
          <label>{{ t('sms.message') }}</label>
          <div class="inp" style="align-items:flex-start;padding:12px;">
            <textarea v-model="draft" rows="4" maxlength="480" style="border:none;outline:none;flex:1;font-family:'Inter';font-size:13px;background:transparent;resize:none;"></textarea>
          </div>
        </div>
        <div style="font-size:11.5px;color:var(--ink-400);margin-bottom:14px;">
          {{ t('sms.variables') }}:
          <a v-for="v in current.variables" :key="v" href="#" style="margin-right: 8px" @click.prevent="insertVar(v)">{{ '{' + v + '}' }}</a>
        </div>
        <div v-if="saveErr" style="color: var(--danger); font-size: 13px; margin-bottom: 10px">{{ saveErr }}</div>
        <div v-if="saveMsg" style="color: var(--green-600); font-size: 13px; margin-bottom: 10px">{{ saveMsg }}</div>
        <div style="display: flex; gap: 10px">
          <button class="btn btn-primary" :disabled="saving || !draft.trim()" @click="save(false)">
            {{ saving ? t('sms.saving') : t('sms.saveTemplate') }}
          </button>
          <button v-if="current.custom" class="btn btn-outline" :disabled="saving" @click="save(true)">
            {{ t('sms.restoreDefault') }}
          </button>
        </div>
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
        <div class="card-head"><h3>{{ t('smsd.delivery') }}</h3></div>
        <div v-for="d in delivery" :key="d.lbl" class="chartbar-row">
          <div class="lbl">{{ d.lbl }}</div>
          <div class="bar"><div :style="{ width: barWidth(d.v, d.max), background: d.color }"></div></div>
          <div class="val">{{ d.v }}</div>
        </div>
      </div>
    </template>
  </div>
</template>
