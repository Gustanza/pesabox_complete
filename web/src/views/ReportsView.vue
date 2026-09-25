<script setup>
import { computed, nextTick, onMounted, reactive, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { intlLocale } from '../i18n'
import { listGroups } from '@/api/groups'
import { listPartners, listClusters } from '@/api/admin'
import { can } from '@/api/access'
import { fetchReport, listDatasets, exportReports } from '@/api/reports'

const { t, locale } = useI18n()
const route = useRoute()

// Every date in a report is an East Africa Time day (server/reports.go).
const TZ = 'Africa/Dar_es_Salaam'
const PREVIEW_ROWS = 50

// ?preset=finance (from the Finance page) pre-selects the money datasets.
const PRESETS = { finance: ['savings', 'shares', 'social-fund', 'loans', 'fines', 'fines-outstanding', 'expenses', 'transactions'] }

const datasets = ref([])
const selected = reactive({}) // datasetKey -> true
const pickedColumns = reactive({}) // datasetKey -> { columnKey: true }
const expanded = reactive({}) // datasetKey -> column picker open
const format = ref('xlsx')
const groups = ref([])
const groupId = ref('')
const partners = ref([])
const clusters = ref([])
const partnerId = ref('')
const clusterId = ref('')
// Partner / cluster filters only for roles that may see the structure; a
// group role reports on its own group only (the server refuses anything else).
const showStructure = computed(() => can('structure.view'))
const shownClusters = computed(() => clusters.value.filter((c) => !partnerId.value || c.partnerId === partnerId.value))
const shownGroups = computed(() => {
  if (clusterId.value) return groups.value.filter((g) => g.clusterId === clusterId.value)
  if (partnerId.value) {
    const ids = new Set(shownClusters.value.map((c) => c.id))
    return groups.value.filter((g) => ids.has(g.clusterId))
  }
  return groups.value
})
const scope = computed(() => ({ partnerId: partnerId.value, clusterId: clusterId.value, groupId: groupId.value }))
watch(partnerId, () => {
  clusterId.value = ''
  groupId.value = ''
})
watch(clusterId, () => {
  groupId.value = ''
})
const from = ref('')
const to = ref('')
const generating = ref('')
const exporting = ref(false)
const lastReport = ref(null)
const printing = ref(false)
const error = ref('')

// A preview must always match the controls: any change of dates, filters or
// the dataset catalog drops it (and any response still on its way).
let requestSeq = 0
function clearPreview() {
  requestSeq++
  lastReport.value = null
  generating.value = ''
}
watch([from, to, partnerId, clusterId, groupId, datasets], clearPreview)

const categories = computed(() => {
  const map = new Map()
  for (const d of datasets.value) {
    if (!map.has(d.category)) map.set(d.category, { key: d.category, name: locale.value === 'sw' ? d.categorySw : d.category, items: [] })
    map.get(d.category).items.push(d)
  }
  return [...map.values()]
})

const selectedKeys = computed(() => datasets.value.filter((d) => selected[d.key]).map((d) => d.key))

onMounted(async () => {
  try {
    const list = await listDatasets()
    for (const d of list) {
      pickedColumns[d.key] = Object.fromEntries(d.columns.map((c) => [c.key, true]))
    }
    datasets.value = list
    for (const k of PRESETS[route.query.preset] || []) selected[k] = true
  } catch (e) {
    error.value = e.message || t('reports.loadFailed')
  }
  try {
    groups.value = await listGroups()
    if (showStructure.value) {
      ;[partners.value, clusters.value] = await Promise.all([listPartners(), listClusters()])
    }
  } catch {
    // filters just won't have options; the report itself still works
  }
})

function toggleAll(on) {
  for (const d of datasets.value) selected[d.key] = on
}

function label(d) {
  return locale.value === 'sw' ? d.sw : d.en
}

function colLabel(c) {
  return locale.value === 'sw' ? c.sw : c.en
}

// Preview columns arrive as English keys; show them in the active language.
function previewColLabel(key) {
  const d = datasets.value.find((x) => x.key === lastReport.value?.key)
  const c = d?.columns.find((x) => x.key === key)
  return c ? colLabel(c) : key
}

// One formatter per column type (the server's column-type map), so the
// preview reads like the PDF / Excel export.
function fmtCell(type, v) {
  if (v === null || v === undefined || v === '') return '—'
  const loc = intlLocale()
  switch (type) {
    case 'date':
    case 'datetime': {
      const d = new Date(v)
      if (isNaN(d)) return String(v)
      return type === 'date'
        ? d.toLocaleDateString(loc, { timeZone: TZ, year: 'numeric', month: 'short', day: 'numeric' })
        : d.toLocaleString(loc, { timeZone: TZ, year: 'numeric', month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit' })
    }
    case 'money':
      return Number(v).toLocaleString(loc, { maximumFractionDigits: 2 })
    case 'count':
      return Number(v).toLocaleString(loc)
    case 'rate':
      return Number(v).toLocaleString(loc, { maximumFractionDigits: 1 }) + '%'
    case 'enum': {
      const map = lastReport.value?.enums?.[locale.value === 'sw' ? 'sw' : 'en'] || {}
      return map[v] ?? String(v)
    }
    default:
      return String(v)
  }
}

const isNumeric = (type) => ['money', 'count', 'int', 'rate'].includes(type)
const colType = (c) => lastReport.value?.types?.[c] || 'text'

// Where the footer's TOTAL label goes: the first column without a total.
const totalsLabelCol = computed(() => {
  const r = lastReport.value
  if (!r?.totals) return null
  return r.columns.find((c) => !(c in r.totals)) ?? null
})

const shownRows = computed(() => {
  const rows = lastReport.value?.rows || []
  return printing.value ? rows : rows.slice(0, PREVIEW_ROWS)
})

const previewNote = computed(() => {
  const r = lastReport.value
  if (!r) return ''
  const date = fmtCell('date', r.asAt)
  if (r.pointInTime) return t('reports.asAt', { date })
  if (r.balances) return t('reports.balancesNote', { date })
  return ''
})

async function generate(d) {
  const id = ++requestSeq
  generating.value = d.key
  error.value = ''
  try {
    const report = await fetchReport(d.key, { ...scope.value, from: from.value, to: to.value })
    if (id !== requestSeq) return // the controls changed while this was loading
    lastReport.value = { label: label(d), ...report }
  } catch (e) {
    if (id === requestSeq) error.value = e.message || t('reports.generateFailed')
  } finally {
    if (id === requestSeq) generating.value = ''
  }
}

async function doExport() {
  error.value = ''
  if (!selectedKeys.value.length) {
    error.value = t('reports.chooseDataset')
    return
  }
  // Only send a column list for datasets where the user unticked something.
  const columns = {}
  for (const d of datasets.value) {
    if (!selected[d.key]) continue
    const keep = d.columns.filter((c) => pickedColumns[d.key][c.key]).map((c) => c.key)
    if (!keep.length) {
      error.value = t('reports.chooseColumn', { name: label(d) })
      return
    }
    if (keep.length !== d.columns.length) columns[d.key] = keep
  }
  exporting.value = true
  try {
    await exportReports({
      datasets: selectedKeys.value,
      columns,
      format: format.value,
      ...scope.value,
      from: from.value,
      to: to.value,
      lang: locale.value
    })
  } catch (e) {
    error.value = e.message || t('reports.exportFailed')
  } finally {
    exporting.value = false
  }
}

// Printing shows every row of the preview, not just the first 50.
async function printPreview() {
  printing.value = true
  await nextTick()
  try {
    window.print()
  } finally {
    printing.value = false
  }
}
</script>

<template>
  <div>
    <div class="page-head no-print">
      <div>
        <h1>{{ t('reports.title') }}</h1>
        <p>{{ t('reports.subtitle') }}</p>
      </div>
    </div>

    <div v-if="error" class="no-print" role="alert" style="color: var(--danger); font-size: 13.5px; margin-bottom: 12px">
      {{ error }}
    </div>

    <div class="grid2 no-print">
      <div>
        <div v-for="cat in categories" :key="cat.key" class="card">
          <div class="card-head"><h3>{{ cat.name }}</h3></div>
          <div v-for="d in cat.items" :key="d.key">
            <div class="kv">
              <label style="display: flex; align-items: center; gap: 8px; cursor: pointer">
                <input v-model="selected[d.key]" type="checkbox" />
                <span class="k">
                  {{ label(d) }}
                  <span v-if="d.pointInTime" class="cell-sub" style="font-weight: 400">({{ t('reports.pointInTime') }})</span>
                </span>
              </label>
              <span style="display: flex; gap: 14px">
                <button
                  type="button"
                  class="link-btn"
                  :aria-expanded="!!expanded[d.key]"
                  @click="expanded[d.key] = !expanded[d.key]"
                >{{ expanded[d.key] ? t('reports.hideColumns') : t('reports.columns') }}</button>
                <button
                  type="button"
                  class="link-btn strong"
                  :disabled="generating === d.key"
                  @click="generate(d)"
                >{{ generating === d.key ? t('reports.loadingShort') : t('reports.preview') }}</button>
              </span>
            </div>
            <div v-if="expanded[d.key]" style="display: flex; flex-wrap: wrap; gap: 6px 16px; padding: 4px 0 12px 26px">
              <label
                v-for="c in d.columns"
                :key="c.key"
                style="display: flex; align-items: center; gap: 6px; font-size: 12.5px; cursor: pointer"
              >
                <input v-model="pickedColumns[d.key][c.key]" type="checkbox" />
                {{ colLabel(c) }}
              </label>
            </div>
          </div>
        </div>
      </div>

      <div>
        <div class="card">
          <div class="card-head"><h3>{{ t('reports.config') }}</h3></div>
          <div class="field-view">
            <label>{{ t('reports.dateRange') }}</label>
            <div style="display: flex; gap: 8px">
              <input v-model="from" type="date" class="box" style="flex: 1" :max="to || undefined" :aria-label="t('reports.from')" />
              <input v-model="to" type="date" class="box" style="flex: 1" :min="from || undefined" :aria-label="t('reports.to')" />
            </div>
          </div>
          <div v-if="showStructure" class="field-view">
            <label>{{ t('struct.partner') }}</label>
            <select v-model="partnerId" class="box" style="width: 100%">
              <option value="">{{ t('struct.allPartners') }}</option>
              <option v-for="p in partners" :key="p.id" :value="p.id">{{ p.name }}</option>
            </select>
          </div>
          <div v-if="showStructure" class="field-view">
            <label>{{ t('struct.cluster') }}</label>
            <select v-model="clusterId" class="box" style="width: 100%">
              <option value="">{{ t('struct.allClusters') }}</option>
              <option v-for="c in shownClusters" :key="c.id" :value="c.id">{{ c.name }}</option>
            </select>
          </div>
          <div class="field-view">
            <label>{{ t('reports.group') }}</label>
            <select v-model="groupId" class="box" style="width: 100%">
              <option value="">{{ t('reports.allGroups') }}</option>
              <option v-for="g in shownGroups" :key="g.id" :value="g.id">{{ g.name }}</option>
            </select>
          </div>
        </div>

        <div class="card">
          <div class="card-head">
            <h3>{{ t('reports.export') }}</h3>
            <span style="font-size: 12px; color: var(--ink-400)">
              <a href="#" @click.prevent="toggleAll(true)">{{ t('common.selectAll') }}</a> ·
              <a href="#" @click.prevent="toggleAll(false)">{{ t('common.none') }}</a>
            </span>
          </div>
          <div style="font-size: 13px; margin-bottom: 12px">
            {{ t('reports.selected', { n: selectedKeys.length }) }}
          </div>
          <div style="display: flex; gap: 10px; margin-bottom: 14px">
            <button
              v-for="f in [['pdf', 'PDF'], ['xlsx', 'Excel'], ['csv', 'CSV']]"
              :key="f[0]"
              type="button"
              class="btn"
              :class="format === f[0] ? 'btn-primary' : 'btn-outline'"
              :aria-pressed="format === f[0]"
              @click="format = f[0]"
            >{{ f[1] }}</button>
          </div>
          <button
            type="button"
            class="btn btn-primary btn-block"
            :disabled="!selectedKeys.length || exporting"
            @click="doExport"
          >{{ exporting ? t('reports.preparing') : t('reports.exportN', { n: selectedKeys.length }) }}</button>
          <p style="font-size: 12px; color: var(--ink-400); margin: 10px 0 0">
            {{ t('reports.exportHint') }}
          </p>
        </div>
      </div>
    </div>

    <div v-if="lastReport" class="panel print-area" style="margin-top: 20px">
      <div class="panel-inner" style="padding-bottom: 0; display: flex; justify-content: space-between; gap: 12px">
        <div>
          <h3 style="font-size: 15px">{{ lastReport.label }}</h3>
          <p v-if="previewNote" class="cell-sub" style="margin: 4px 0 0">{{ previewNote }}</p>
        </div>
        <button type="button" class="btn btn-outline btn-sm no-print" @click="printPreview">{{ t('common.print') }}</button>
      </div>
      <div style="overflow-x: auto; margin-top: 14px">
        <div v-if="!lastReport.rows.length" class="empty"><p>{{ t('reports.noRecords') }}</p></div>
        <table v-else class="dtable">
          <thead>
            <tr>
              <th v-for="c in lastReport.columns" :key="c" :style="isNumeric(colType(c)) ? 'text-align: right' : ''">{{ previewColLabel(c) }}</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="(row, i) in shownRows" :key="i">
              <td v-for="c in lastReport.columns" :key="c" :style="isNumeric(colType(c)) ? 'text-align: right; white-space: nowrap' : ''">
                {{ fmtCell(colType(c), row[c]) }}
              </td>
            </tr>
          </tbody>
          <tfoot v-if="lastReport.totals">
            <tr class="totals-row">
              <td v-for="c in lastReport.columns" :key="c" :style="isNumeric(colType(c)) ? 'text-align: right; white-space: nowrap' : ''">
                <template v-if="c === totalsLabelCol">{{ t('reports.total') }}</template>
                <template v-else-if="c in lastReport.totals">{{ fmtCell(colType(c), lastReport.totals[c]) }}</template>
              </td>
            </tr>
          </tfoot>
        </table>
        <p v-if="!printing && lastReport.rows.length > PREVIEW_ROWS" class="no-print" style="font-size: 12px; color: var(--ink-400); padding: 10px">
          {{ t('reports.showingFirst', { n: lastReport.rows.length }) }}
        </p>
      </div>
      <div style="height: 20px"></div>
    </div>
  </div>
</template>

<style>
.link-btn {
  background: none;
  border: 0;
  padding: 2px 0;
  font: inherit;
  font-size: 12px;
  color: var(--ink-400);
  cursor: pointer;
}
.link-btn.strong {
  color: var(--green-600);
  font-weight: 700;
}
.link-btn:hover,
.link-btn:focus-visible {
  text-decoration: underline;
}
.link-btn:disabled {
  cursor: default;
  text-decoration: none;
}
.totals-row td {
  font-weight: 800;
  background: var(--green-100);
  border-top: 2px solid var(--green-600);
}
@media print {
  .no-print,
  .sidebar,
  .topbar {
    display: none !important;
  }
  .print-area {
    margin: 0 !important;
    box-shadow: none !important;
  }
}
</style>
