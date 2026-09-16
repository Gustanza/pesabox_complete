<script setup>
import { computed, ref } from 'vue'
import Svgs from '../components/Svgs.vue'
import { AUDIT } from '../data/mock.js'

const rows = AUDIT.map(a => ({ ...a, date: '12 Sep 2026' }))

const search = ref('')
const user = ref('All users')
const date = ref('All dates')

const users = computed(() => ['All users', ...new Set(rows.map(a => a.user))])
const dates = computed(() => ['All dates', ...new Set(rows.map(a => a.date))])

const filtered = computed(() => {
  const q = search.value.trim().toLowerCase()
  return rows.filter(a => {
    const matchQ = !q || a.action.toLowerCase().includes(q) || a.resource.toLowerCase().includes(q) || a.user.toLowerCase().includes(q)
    const matchU = user.value === 'All users' || a.user === user.value
    const matchD = date.value === 'All dates' || a.date === date.value
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
        <h1>Audit Logs</h1>
        <p>A complete, non-editable trail of who did what, when.</p>
      </div>
    </div>

    <div class="panel">
      <div class="toolbar">
        <div class="search-input">
          <Svgs name="search" />
          <input v-model="search" type="search" placeholder="Search audit logs..." />
        </div>
        <select v-model="user" class="filter-select">
          <option v-for="u in users" :key="u" :value="u">{{ u }}</option>
        </select>
        <select v-model="date" class="filter-select">
          <option v-for="d in dates" :key="d" :value="d">{{ d }}</option>
        </select>
      </div>
      <div style="overflow-x:auto;">
        <table class="dtable">
        <thead>
          <tr><th>Time</th><th>User</th><th>Action</th><th>Resource</th></tr>
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