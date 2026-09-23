<script setup>
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { initials, avaColor } from '../data/mock.js'
import { listUsers, updateUser, deleteUser, ROLES } from '../api/users.js'
import { listGroups } from '../api/groups.js'

const router = useRouter()
const { t } = useI18n()
const activeTab = ref('users')

const users = ref([])
const groups = ref([])
const loading = ref(true)
const error = ref('')

async function load() {
  loading.value = true
  error.value = ''
  try {
    ;[users.value, groups.value] = await Promise.all([listUsers(), listGroups()])
  } catch (e) {
    error.value = e.message || t('users.loadFailed')
  } finally {
    loading.value = false
  }
}

onMounted(load)

function displayName(u) {
  const name = [u.firstName, u.lastName].filter(Boolean).join(' ')
  return name || u.username
}

function groupOf(userId) {
  return groups.value.find((g) => g.createdBy === userId)
}

const pendingAdmins = computed(() =>
  users.value.filter((u) => u.role === 'group_admin' && !groupOf(u.id))
)

function goGroup(userId) {
  const g = groupOf(userId)
  if (g) router.push('/groups/' + g.id)
}

function goCreateGroup(u) {
  router.push({
    path: '/groups/create',
    query: { adminId: u.id, adminName: displayName(u), adminPhone: u.username }
  })
}

async function changeRole(u, event) {
  const role = event.target.value
  try {
    await updateUser(u.id, { role })
    u.role = role
  } catch (e) {
    error.value = e.message || t('usr.roleFailed')
    load()
  }
}

async function toggleStatus(u) {
  const status = u.status === 'active' ? 'inactive' : 'active'
  try {
    await updateUser(u.id, { status })
    u.status = status
    u.isActive = status === 'active'
  } catch (e) {
    error.value = e.message || t('users.statusFailed')
  }
}

async function remove(u) {
  if (!confirm(t('users.confirmDelete', { name: displayName(u) }))) return
  try {
    await deleteUser(u.id)
    users.value = users.value.filter((x) => x.id !== u.id)
  } catch (e) {
    error.value = e.message || t('users.deleteFailed')
  }
}
</script>

<template>
  <div>
    <div class="page-head">
      <div>
        <h1>{{ t('users.title') }}</h1>
        <p>{{ t('users.subtitle') }}</p>
      </div>
    </div>

    <div class="dtabs">
      <button class="dtab" :class="{ active: activeTab === 'users' }" @click="activeTab = 'users'">{{ t('users.title') }}</button>
      <button class="dtab" :class="{ active: activeTab === 'pending' }" @click="activeTab = 'pending'">
        {{ t('usr.pendingTab', { n: pendingAdmins.length }) }}
      </button>
    </div>

    <div v-if="error" style="color: var(--danger); font-size: 13px; margin-bottom: 12px">
      {{ error }}
    </div>

    <template v-if="activeTab === 'pending'">
      <p style="font-size: 12.5px; color: var(--ink-600); margin-bottom: 14px; line-height: 1.5">
        {{ t('usr.pendingNote') }}
      </p>
      <div class="panel">
        <div v-if="loading" class="empty"><p>{{ t('common.loading') }}</p></div>
        <div v-else-if="!pendingAdmins.length" class="empty">
          <div class="ic">&#128100;</div>
          <p>{{ t('usr.noPending') }}</p>
        </div>
        <table v-else class="dtable">
          <thead>
            <tr>
              <th>{{ t('usr.admin') }}</th>
              <th>{{ t('usr.registered') }}</th>
              <th>{{ t('common.status') }}</th>
              <th></th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="a in pendingAdmins" :key="a.id">
              <td>
                <div class="tname">
                  <div class="tav" :style="{ background: avaColor(displayName(a)) }">{{ initials(displayName(a)) }}</div>
                  <div>
                    <div class="cell-main">{{ displayName(a) }}</div>
                    <div class="cell-sub">{{ a.username }}</div>
                  </div>
                </div>
              </td>
              <td>{{ a.createdAt ? new Date(a.createdAt).toLocaleDateString() : '—' }}</td>
              <td><span class="badge gold">{{ t('usr.noGroupYet') }}</span></td>
              <td>
                <button class="btn btn-primary btn-sm" @click="goCreateGroup(a)">{{ t('usr.createGroup') }}</button>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </template>

    <template v-else>
      <p style="font-size: 12.5px; color: var(--ink-600); margin-bottom: 14px; line-height: 1.5">
        {{ t('usr.everyoneNote') }}
      </p>
      <div class="panel">
        <div v-if="loading" class="empty"><p>{{ t('common.loading') }}</p></div>
        <div v-else-if="!users.length" class="empty">
          <div class="ic">&#128100;</div>
          <p>{{ t('users.empty') }}</p>
        </div>
        <table v-else class="dtable">
          <thead>
            <tr>
              <th>{{ t('users.name') }}</th>
              <th>{{ t('users.role') }}</th>
              <th>{{ t('users.group') }}</th>
              <th>{{ t('common.status') }}</th>
              <th></th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="u in users" :key="u.id">
              <td class="clickable" @click="goGroup(u.id)">
                <div class="tname">
                  <div class="tav" :style="{ background: avaColor(displayName(u)) }">{{ initials(displayName(u)) }}</div>
                  <div>
                    <div class="cell-main">{{ displayName(u) }}</div>
                    <div class="cell-sub">{{ u.username }}</div>
                  </div>
                </div>
              </td>
              <td>
                <select :value="u.role" @change="changeRole(u, $event)" class="role-select">
                  <option v-for="r in ROLES" :key="r.value" :value="r.value">{{ t('roles.' + r.value) }}</option>
                </select>
              </td>
              <td>{{ groupOf(u.id)?.name || '—' }}</td>
              <td>
                <span
                  class="badge clickable"
                  :class="u.status === 'active' ? 'green' : 'grey'"
                  @click="toggleStatus(u)"
                >{{ u.status === 'active' ? t('users.active') : t('users.inactive') }}</span>
              </td>
              <td>
                <button class="btn btn-ghost btn-sm" @click="remove(u)">{{ t('common.delete') }}</button>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </template>
  </div>
</template>

<style scoped>
.role-select {
  border: 1.5px solid var(--line);
  border-radius: 8px;
  padding: 6px 8px;
  font-size: 12.5px;
  font-family: 'Inter', sans-serif;
  color: var(--ink-700);
  background: #fff;
}
</style>
