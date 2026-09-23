<script setup>
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import Svgs from '../components/Svgs.vue'
import { AUDIT } from '../data/mock.js'

const { t } = useI18n()

const rows = AUDIT.map(a => ({ ...a, date: '12 Sep 2026' }))

const search = ref('')
// '' means "all" -- filters hold stable values, never translated text
const user = ref('')
const date = ref('')

const users = computed(() => [...new Set(rows.map(a => a.user))])
const dates = computed(() => [...new Set(rows.map(a => a.date))])

const filtered = computed(() => {
  const q = search.value.trim().toLowerCase()
  return rows.filter(a => {
    const matchQ = !q || a.action.toLowerCase().includes(q) || a.resource.toLowerCase().includes(q) || a.user.toLowerCase().includes(q)
    const matchU = !user.value || a.user === user.value
    const matchD = !date.value || a.date === date.value
    return matchQ && matchU && matchD
  })
})

function selectRow(a) {
  alert(a.action + ' on ' + a.resource)
}
</script>

<template>
  <div>
    <div class="page-head">
      <div>
        <h1>{{ t('audit.title') }}</h1>
        <p>{{ t('audit.subtitle') }}</p>
      </div>
    </div>

    <div class="panel">
      <div class="toolbar">
        <div class="search-input">
          <Svgs name="search" />
          <input v-model="search" type="search" :placeholder="t('audit.search')" />
        </div>
        <select v-model="user" class="filter-select">
          <option value="">{{ t('audit.allUsers') }}</option>
          <option v-for="u in users" :key="u" :value="u">{{ u }}</option>
        </select>
        <select v-model="date" class="filter-select">
          <option value="">{{ t('audit.allDates') }}</option>
          <option v-for="d in dates" :key="d" :value="d">{{ d }}</option>
        </select>
      </div>
      <div style="overflow-x:auto;">
        <table class="dtable">
        <thead>
          <tr>
            <th>{{ t('audit.time') }}</th>
            <th>{{ t('audit.user') }}</th>
            <th>{{ t('audit.action') }}</th>
            <th>{{ t('audit.resource') }}</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="a in filtered" :key="a.time" class="clickable" @click="selectRow(a)">
            <td>{{ a.time }}</td>
            <td class="cell-main">{{ a.user }}</td>
            <td>{{ a.action }}</td>
            <td>{{ a.resource }}</td>
</tr>
        </tbody>
        </table>
      </div>
    </div>
  </div>
</template>