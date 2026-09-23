<script setup>
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import Svgs from '../components/Svgs.vue'
import { initials, avaColor } from '../data/mock.js'
import { listGroups, deleteGroup } from '../api/groups.js'

const router = useRouter()
const { t, te } = useI18n()
const statusText = (st) => (te('grp.st.' + st) ? t('grp.st.' + st) : st)
const search = ref('')
// '' means "all" — filters hold stable values, never translated text.
const region = ref('')
const status = ref('')
const page = ref(1)

const groups = ref([])
const loading = ref(true)
const error = ref('')

async function load() {
  loading.value = true
  error.value = ''
  try {
    groups.value = await listGroups()
  } catch (e) {
    error.value = e.message || t('grp.loadFailed')
  } finally {
    loading.value = false
  }
}

onMounted(load)

const regions = computed(() => [...new Set(groups.value.map((g) => g.region).filter(Boolean))])
const statuses = computed(() => [...new Set(groups.value.map((g) => g.status).filter(Boolean))])

const filtered = computed(() => {
  return groups.value.filter((g) => {
    const q = search.value.trim().toLowerCase()
    const matchQ =
      !q ||
      g.name?.toLowerCase().includes(q) ||
      g.adminName?.toLowerCase().includes(q) ||
      g.region?.toLowerCase().includes(q)
    const matchR = !region.value || g.region === region.value
    const matchS = !status.value || g.status === status.value
    return matchQ && matchR && matchS
  })
})

const totalPages = computed(() => Math.max(1, Math.ceil(filtered.value.length / 8)))

function go(id) {
  router.push('/groups/' + id)
}

async function remove(g, event) {
  event.stopPropagation()
  if (!confirm(t('grp.confirmDelete', { name: g.name }))) return
  try {
    await deleteGroup(g.id)
    groups.value = groups.value.filter((x) => x.id !== g.id)
  } catch (e) {
    alert(e.message || t('grp.deleteFailed'))
  }
}
</script>

<template>
  <div>
    <div class="page-head">
      <div>
        <h1>{{ t('grp.title') }}</h1>
        <p>{{ t('grp.subtitle') }}</p>
      </div>
      <div class="page-actions">
        <button class="btn btn-primary" @click="router.push('/groups/create')">
          <Svgs name="plus" /> {{ t('grp.create') }}
        </button>
      </div>
    </div>

    <div v-if="error" style="color: var(--danger); font-size: 13.5px; margin-bottom: 12px">
      {{ error }}
    </div>

    <div class="panel">
      <div class="toolbar">
        <div class="search-input">
          <Svgs name="search" />
          <input v-model="search" type="search" :placeholder="t('grp.search')" />
        </div>
        <select v-model="region" class="filter-select">
          <option value="">{{ t('grp.allRegions') }}</option>
          <option v-for="r in regions" :key="r" :value="r">{{ r }}</option>
        </select>
        <select v-model="status" class="filter-select">
          <option value="">{{ t('grp.allStatuses') }}</option>
          <option v-for="s in statuses" :key="s" :value="s">{{ statusText(s) }}</option>
        </select>
      </div>
      <div style="overflow-x: auto">
        <div v-if="loading" class="empty">
          <p>{{ t('grp.loadingGroups') }}</p>
        </div>
        <div v-else-if="!filtered.length" class="empty">
          <div class="ic">&#128203;</div>
          <p>{{ t('grp.empty') }}</p>
        </div>
        <table v-else class="dtable">
          <thead>
            <tr>
              <th>{{ t('grp.group') }}</th>
              <th>{{ t('grp.admin') }}</th>
              <th>{{ t('grp.members') }}</th>
              <th>{{ t('grp.cycle') }}</th>
              <th>{{ t('common.status') }}</th>
              <th></th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="g in filtered" :key="g.id" class="clickable" @click="go(g.id)">
              <td>
                <div class="tname">
                  <div class="tav" :style="{ background: avaColor(g.name) }">{{ initials(g.name) }}</div>
                  <div>
                    <div class="cell-main">{{ g.name }}</div>
                    <div class="cell-sub">{{ g.region || '—' }}</div>
                  </div>
                </div>
              </td>
              <td class="cell-muted">{{ g.adminName || '—' }}</td>
              <td>{{ g.memberCount }}</td>
              <td class="cell-muted">{{ g.cycleCurrent }}/{{ g.cycleTotal }}</td>
              <td>
                <span class="badge" :class="g.status === 'Active' ? 'green' : 'grey'">{{ statusText(g.status) }}</span>
              </td>
              <td>
                <button class="btn btn-ghost btn-sm" @click="remove(g, $event)">{{ t('common.delete') }}</button>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
      <div class="pagination">
        <button class="page-btn" :disabled="page <= 1" @click="page--">&#8249;</button>
        <button v-for="p in totalPages" :key="p" class="page-btn" :class="{ active: page === p }" @click="page = p">{{ p }}</button>
        <button class="page-btn" :disabled="page >= totalPages" @click="page++">&#8250;</button>
      </div>
    </div>
  </div>
</template>