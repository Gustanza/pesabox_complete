<script setup>
import { computed, onMounted, reactive, ref } from 'vue'
import { useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { listGroups } from '@/api/groups'
import { fetchReport, listDatasets, exportReports } from '@/api/reports'

const { t, locale } = useI18n()
const route = useRoute()

// ?preset=finance (from the Finance page) pre-selects the money datasets.
const PRESETS = { finance: ['savings', 'shares', 'social-fund', 'loans', 'fines', 'transactions'] }

const datasets = ref([])
const selected = reactive({}) // datasetKey -> true
const pickedColumns = reactive({}) // datasetKey -> { columnKey: true }
const expanded = reactive({}) // datasetKey -> column picker open
const format = ref('xlsx')
const groups = ref([])
const groupId = ref('')
const from = ref('')
const to = ref('')
const generating = ref('')
const exporting = ref(false)
const lastReport = ref(null)
const error = ref('')

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
    datasets.value = await listDatasets()
    for (const d of datasets.value) {
      pickedColumns[d.key] = Object.fromEntries(d.columns.map((c) => [c.key, true]))
    }
    for (const k of PRESETS[route.query.preset] || []) selected[k] = true
  } catch (e) {
    error.value = e.message || t('reports.loadFailed')
  }
  try {
    groups.value = await listGroups()
  } catch {
    // group filter just won't have options; the report itself still works
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
  for (const d of datasets.value) {
    const c = d.columns.find((x) => x.key === key)
    if (c) return colLabel(c)
  }
  return key
}

async function generate(d) {
  generating.value = d.key
  error.value = ''
  try {
    const report = await fetchReport(d.key, { groupId: groupId.value, from: from.value, to: to.value })
    lastReport.value = { label: label(d), ...report }
  } catch (e) {
    error.value = e.message || t('reports.generateFailed')
  } finally {
    generating.value = ''
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
      groupId: groupId.value,
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

function printPreview() {
  window.print()
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

    <div v-if="error" class="no-print" style="color: var(--danger); font-size: 13.5px; margin-bottom: 12px">
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
                <span class="k">{{ label(d) }}</span>
              </label>
              <span style="display: flex; gap: 14px">
                <span
                  style="color: var(--ink-400); font-size: 12px; cursor: pointer"
                  @click="expanded[d.key] = !expanded[d.key]"
                >{{ expanded[d.key] ? t('reports.hideColumns') : t('reports.columns') }}</span>
                <span
                  style="color: var(--green-600); font-weight: 700; font-size: 12px; cursor: pointer"
                  @click="generate(d)"
                >{{ generating === d.key ? t('reports.loadingShort') : t('reports.preview') }}</span>
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
              <input v-model="from" type="date" class="box" style="flex: 1" />
              <input v-model="to" type="date" class="box" style="flex: 1" />
            </div>
          </div>
          <div class="field-view">
            <label>{{ t('reports.group') }}</label>
            <select v-model="groupId" class="box" style="width: 100%">
              <option value="">{{ t('reports.allGroups') }}</option>
              <option v-for="g in groups" :key="g.id" :value="g.id">{{ g.name }}</option>
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
              class="btn"
              :class="format === f[0] ? 'btn-primary' : 'btn-outline'"
              @click="format = f[0]"
            >{{ f[1] }}</button>
          </div>
          <button
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
      <div class="panel-inner" style="padding-bottom: 0; display: flex; justify-content: space-between">
        <h3 style="font-size: 15px">{{ lastReport.label }}</h3>
        <button class="btn btn-outline btn-sm no-print" @click="printPreview">{{ t('common.print') }}</button>
      </div>
      <div style="overflow-x: auto; margin-top: 14px">
        <div v-if="!lastReport.rows.length" class="empty"><p>{{ t('reports.noRecords') }}</p></div>
        <table v-else class="dtable">
          <thead>
            <tr><th v-for="c in lastReport.columns" :key="c">{{ previewColLabel(c) }}</th></tr>
          </thead>
          <tbody>
            <tr v-for="(row, i) in lastReport.rows.slice(0, 50)" :key="i">
              <td v-for="c in lastReport.columns" :key="c">{{ row[c] }}</td>
            </tr>
          </tbody>
        </table>
        <p v-if="lastReport.rows.length > 50" class="no-print" style="font-size: 12px; color: var(--ink-400); padding: 10px">
          {{ t('reports.showingFirst', { n: lastReport.rows.length }) }}
        </p>
      </div>
      <div style="height: 20px"></div>
    </div>
  </div>
</template>

<style>
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
