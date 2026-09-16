<script setup>
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { initials, avaColor } from '../data/mock.js'
import { listUsers, updateUser, deleteUser, ROLES } from '../api/users.js'
import { listGroups } from '../api/groups.js'

const router = useRouter()
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
    error.value = e.message || 'Failed to load users'
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
    error.value = e.message || 'Failed to update role'
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
    error.value = e.message || 'Failed to update status'
  }
}

async function remove(u) {
  if (!confirm(`Delete "${displayName(u)}"? This cannot be undone.`)) return
  try {
    await deleteUser(u.id)
    users.value = users.value.filter((x) => x.id !== u.id)
  } catch (e) {
    error.value = e.message || 'Failed to delete user'
  }
}
</script>

<template>
  <div>
    <div class="page-head">
      <div>
        <h1>Users</h1>
        <p>Manage every account on the platform.</p>
      </div>
    </div>

    <div class="dtabs">
      <button class="dtab" :class="{ active: activeTab === 'users' }" @click="activeTab = 'users'">Users</button>
      <button class="dtab" :class="{ active: activeTab === 'pending' }" @click="activeTab = 'pending'">
        Pending Admins ({{ pendingAdmins.length }})
      </button>
    </div>

    <div v-if="error" style="color: var(--danger); font-size: 13px; margin-bottom: 12px">
      {{ error }}
    </div>

    <template v-if="activeTab === 'pending'">
      <p style="font-size: 12.5px; color: var(--ink-600); margin-bottom: 14px; line-height: 1.5">
        Users promoted to Group Admin (see the Users tab) who don't have a group yet.
        Create a group and assign it to them below.
      </p>
      <div class="panel">
        <div v-if="loading" class="empty"><p>Loading…</p></div>
        <div v-else-if="!pendingAdmins.length" class="empty">
          <div class="ic">&#128100;</div>
          <p>No admins waiting for a group.</p>
        </div>
        <table v-else class="dtable">
          <thead>
            <tr><th>Admin</th><th>Registered</th><th>Status</th><th></th></tr>
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
              <td><span class="badge gold">No group yet</span></td>
              <td>
                <button class="btn btn-primary btn-sm" @click="goCreateGroup(a)">+ Create Group</button>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </template>

    <template v-else>
      <p style="font-size: 12.5px; color: var(--ink-600); margin-bottom: 14px; line-height: 1.5">
        Everyone registered on PesaBox. Promote a user to Group Admin, then create their group from the
        Pending Admins tab.
      </p>
      <div class="panel">
        <div v-if="loading" class="empty"><p>Loading…</p></div>
        <div v-else-if="!users.length" class="empty">
          <div class="ic">&#128100;</div>
          <p>No users registered yet.</p>
        </div>
        <table v-else class="dtable">
          <thead>
            <tr><th>Name</th><th>Role</th><th>Group</th><th>Status</th><th></th></tr>
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
                  <option v-for="r in ROLES" :key="r.value" :value="r.value">{{ r.label }}</option>
                </select>
              </td>
              <td>{{ groupOf(u.id)?.name || '—' }}</td>
              <td>
                <span
                  class="badge clickable"
                  :class="u.status === 'active' ? 'green' : 'grey'"
                  @click="toggleStatus(u)"
                >{{ u.status === 'active' ? 'Active' : 'Inactive' }}</span>
              </td>
              <td>
                <button class="btn btn-ghost btn-sm" @click="remove(u)">Delete</button>
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
