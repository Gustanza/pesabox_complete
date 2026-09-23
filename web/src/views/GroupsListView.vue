<script setup>
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import Svgs from '../components/Svgs.vue'
import { initials, avaColor } from '../data/mock.js'
import { listGroups, deleteGroup } from '../api/groups.js'
import { listClusters } from '../api/admin.js'
import { can } from '../api/access.js'

const router = useRouter()
const route = useRoute()
const PAGE_SIZE = 8
const { t, te } = useI18n()
const statusText = (st) => (te('grp.st.' + st) ? t('grp.st.' + st) : st)
const search = ref('')
// '' means "all" — filters hold stable values, never translated text.
const region = ref('')
const status = ref('')
const clusterId = ref(route.query.clusterId || '')
const page = ref(1)
const clusters = ref([])
const canCreate = computed(() => can('groups.create'))
const canDelete = computed(() => can('group.settings'))
const clusterName = (id) => clusters.value.find((c) => c.id === id)?.name || '—'

const groups = ref([])
const loading = ref(true)
const error = ref('')

async function load() {
  loading.value = true
  error.value = ''
  try {
    groups.value = await listGroups()
    if (can('structure.view')) clusters.value = await listClusters().catch(() => [])
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
    const matchC = !clusterId.value || g.clusterId === clusterId.value
    return matchQ && matchR && matchS && matchC
  })
})

const totalPages = computed(() => Math.max(1, Math.ceil(filtered.value.length / PAGE_SIZE)))
const pageRows = computed(() => filtered.value.slice((page.value - 1) * PAGE_SIZE, page.value * PAGE_SIZE))
watch([search, region, status, clusterId], () => {
  page.value = 1
})

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
      <div v-if="canCreate" class="page-actions">
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
        <select v-if="clusters.length" v-model="clusterId" class="filter-select">
          <option value="">{{ t('struct.allClusters') }}</option>
          <option v-for="c in clusters" :key="c.id" :value="c.id">{{ c.name }}</option>
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
              <th v-if="clusters.length">{{ t('struct.cluster') }}</th>
              <th>{{ t('grp.admin') }}</th>
              <th>{{ t('grp.members') }}</th>
              <th>{{ t('grp.cycle') }}</th>
              <th>{{ t('common.status') }}</th>
              <th></th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="g in pageRows" :key="g.id" class="clickable" @click="go(g.id)">
              <td>
                <div class="tname">
                  <div class="tav" :style="{ background: avaColor(g.name) }">{{ initials(g.name) }}</div>
                  <div>
                    <div class="cell-main">{{ g.name }}</div>
                    <div class="cell-sub">{{ g.region || '—' }}</div>
                  </div>
                </div>
              </td>
              <td v-if="clusters.length" class="cell-muted">{{ clusterName(g.clusterId) }}</td>
              <td class="cell-muted">{{ g.adminName || '—' }}</td>
              <td>{{ g.memberCount }}</td>
              <td class="cell-muted">{{ g.cycleCurrent }}/{{ g.cycleTotal }}</td>
              <td>
                <span class="badge" :class="g.status === 'Active' ? 'green' : 'grey'">{{ statusText(g.status) }}</span>
              </td>
              <td>
                <button v-if="canDelete" class="btn btn-ghost btn-sm" @click="remove(g, $event)">{{ t('common.delete') }}</button>
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