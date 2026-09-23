<script setup>
import { computed, onMounted, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import Svgs from '../components/Svgs.vue'
import { initials, avaColor } from '../data/mock.js'
import { listUsers, addUser, updateUser, addAssignment, removeAssignment, ROLES, PRESETS, POSITIONS } from '../api/users.js'
import { listPartners, listClusters } from '../api/admin.js'
import { listGroups } from '../api/groups.js'
import { access } from '../api/access.js'

// Users & roles (TODO.md §4): the 7 levels from "Pesa Box User Levels.pdf",
// staff permission presets, and assignments that decide which partners /
// clusters / groups each person can see. Nobody is deleted — accounts are
// deactivated and can be reactivated.

const router = useRouter()
const { t, te } = useI18n()
const activeTab = ref('users')

const users = ref([])
const groups = ref([])
const partners = ref([])
const clusters = ref([])
const loading = ref(true)
const error = ref('')
const busy = ref(false)
const search = ref('')
const roleFilter = ref('')

async function load() {
  loading.value = true
  error.value = ''
  try {
    ;[users.value, groups.value, partners.value, clusters.value] = await Promise.all([
      listUsers(),
      listGroups(),
      listPartners(),
      listClusters()
    ])
  } catch (e) {
    error.value = e.message || t('users.loadFailed')
  } finally {
    loading.value = false
  }
}
onMounted(load)

const roleText = (r) => (r && te('roles.' + r) ? t('roles.' + r) : t('ur.noRole'))
const positionText = (p) => (te('ur.pos.' + p) ? t('ur.pos.' + p) : p)
const displayName = (u) => [u.firstName, u.lastName].filter(Boolean).join(' ') || u.username
const meId = computed(() => access.data?.userId)

const filtered = computed(() => {
  const q = search.value.trim().toLowerCase()
  return users.value.filter((u) => {
    const matchQ = !q || displayName(u).toLowerCase().includes(q) || (u.username || '').includes(q)
    const matchR = !roleFilter.value || (roleFilter.value === 'none' ? !u.role : u.role === roleFilter.value)
    return matchQ && matchR
  })
})

async function run(fn, reload = true) {
  busy.value = true
  error.value = ''
  try {
    await fn()
    if (reload) await load()
    return true
  } catch (e) {
    error.value = e.message || t('common.requestFailed')
    await load()
    return false
  } finally {
    busy.value = false
  }
}

function changeRole(u, event) {
  const role = event.target.value
  run(() => updateUser(u.id, { role }))
}

function changePreset(u, event) {
  run(() => updateUser(u.id, { preset: event.target.value }))
}

function toggleStatus(u) {
  const status = u.isActive ? 'inactive' : 'active'
  if (status === 'inactive' && !confirm(t('ur.confirmDeactivate', { name: displayName(u) }))) return
  run(() => updateUser(u.id, { status }))
}

function unassign(a) {
  if (!confirm(t('ur.confirmUnassign', { name: a.scopeName || t('ur.scope.all') }))) return
  run(() => removeAssignment(a.id))
}

// ---- assign form (one user at a time) ----
const assignFor = ref(null)
const assign = reactive({ scopeType: 'cluster', scopeId: '', position: '' })
function openAssign(u) {
  assignFor.value = u
  Object.assign(assign, { scopeType: u.role === 'partner_user' ? 'partner' : u.role === 'group_officer' || u.role === 'group_admin' ? 'group' : 'cluster', scopeId: '', position: u.role === 'group_admin' ? 'mwenyekiti' : u.role === 'group_officer' ? 'katibu' : '' })
}
const scopeOptions = computed(() => {
  switch (assign.scopeType) {
    case 'partner':
      return partners.value.map((p) => ({ id: p.id, name: p.name }))
    case 'cluster':
      return clusters.value.map((c) => ({ id: c.id, name: c.name + (c.partnerName ? ' — ' + c.partnerName : '') }))
    case 'group':
      return groups.value.map((g) => ({ id: g.id, name: g.name }))
  }
  return []
})
async function saveAssign() {
  if (assign.scopeType !== 'all' && !assign.scopeId) {
    error.value = t('ur.chooseScope')
    return
  }
  const ok = await run(() =>
    addAssignment({
      userId: assignFor.value.id,
      scopeType: assign.scopeType,
      scopeId: assign.scopeType === 'all' ? '' : assign.scopeId,
      position: assign.scopeType === 'group' ? assign.position : ''
    })
  )
  if (ok) assignFor.value = null
}

// ---- add person ----
const showAdd = ref(false)
const newUser = reactive({ phone: '', firstName: '', lastName: '', role: 'staff', preset: 'viewer' })
async function saveNew() {
  if (!newUser.phone.trim()) {
    error.value = t('ur.phoneRequired')
    return
  }
  const ok = await run(() => addUser({ ...newUser }))
  if (ok) {
    showAdd.value = false
    Object.assign(newUser, { phone: '', firstName: '', lastName: '', role: 'staff', preset: 'viewer' })
  }
}

// ---- group admins without a group ----
function groupOf(u) {
  const tail = (u.username || '').replace(/\D/g, '').slice(-9)
  return groups.value.find(
    (g) => g.createdBy === u.id || (tail && (g.adminPhone || '').replace(/\D/g, '').slice(-9) === tail)
  )
}
const pendingAdmins = computed(() => users.value.filter((u) => u.role === 'group_admin' && u.isActive && !groupOf(u)))
function goCreateGroup(u) {
  router.push({ path: '/groups/create', query: { adminId: u.id, adminName: displayName(u), adminPhone: u.username } })
}
</script>

<template>
  <div>
    <div class="page-head">
      <div>
        <h1>{{ t('ur.title') }}</h1>
        <p>{{ t('ur.subtitle') }}</p>
      </div>
      <div class="page-actions">
        <button class="btn btn-primary" @click="showAdd = !showAdd"><Svgs name="plus" /> {{ t('ur.addPerson') }}</button>
      </div>
    </div>

    <div class="dtabs">
      <button class="dtab" :class="{ active: activeTab === 'users' }" @click="activeTab = 'users'">{{ t('ur.people') }} ({{ users.length }})</button>
      <button class="dtab" :class="{ active: activeTab === 'pending' }" @click="activeTab = 'pending'">
        {{ t('usr.pendingTab', { n: pendingAdmins.length }) }}
      </button>
    </div>

    <div v-if="error" style="color: var(--danger); font-size: 13px; margin-bottom: 12px">{{ error }}</div>

    <div v-if="showAdd" class="card" style="max-width: 720px">
      <div class="card-head"><h3>{{ t('ur.addPerson') }}</h3></div>
      <p style="font-size: 12.5px; color: var(--ink-600); margin-bottom: 12px">{{ t('ur.addNote') }}</p>
      <div class="field-row">
        <div class="field">
          <label>{{ t('struct.phone') }}</label>
          <div class="inp filled"><input v-model="newUser.phone" type="tel" placeholder="0712 345 678" /></div>
        </div>
        <div class="field">
          <label>{{ t('users.role') }}</label>
          <div class="inp filled">
            <select v-model="newUser.role">
              <option v-for="r in ROLES" :key="r" :value="r">{{ roleText(r) }}</option>
            </select>
          </div>
        </div>
      </div>
      <div class="field-row">
        <div class="field">
          <label>{{ t('ur.firstName') }}</label>
          <div class="inp filled"><input v-model="newUser.firstName" /></div>
        </div>
        <div class="field">
          <label>{{ t('ur.lastName') }}</label>
          <div class="inp filled"><input v-model="newUser.lastName" /></div>
        </div>
      </div>
      <div v-if="newUser.role === 'staff'" class="field">
        <label>{{ t('ur.preset') }}</label>
        <div class="inp filled">
          <select v-model="newUser.preset">
            <option v-for="p in PRESETS" :key="p" :value="p">{{ t('ur.presets.' + p) }}</option>
          </select>
        </div>
        <div class="hint">{{ t('ur.presetHint.' + newUser.preset) }}</div>
      </div>
      <div class="cg-actions">
        <button class="btn btn-primary" :disabled="busy" @click="saveNew">{{ t('common.save') }}</button>
        <button class="btn btn-ghost" @click="showAdd = false">{{ t('common.cancel') }}</button>
      </div>
    </div>

    <div v-if="assignFor" class="card" style="max-width: 720px">
      <div class="card-head"><h3>{{ t('ur.assignTitle', { name: displayName(assignFor) }) }}</h3></div>
      <div class="field-row">
        <div class="field">
          <label>{{ t('ur.scopeType') }}</label>
          <div class="inp filled">
            <select v-model="assign.scopeType" @change="assign.scopeId = ''">
              <option v-if="assignFor.role === 'staff' || assignFor.role === 'super_admin'" value="all">{{ t('ur.scope.all') }}</option>
              <option value="partner">{{ t('ur.scope.partner') }}</option>
              <option value="cluster">{{ t('ur.scope.cluster') }}</option>
              <option value="group">{{ t('ur.scope.group') }}</option>
            </select>
          </div>
        </div>
        <div v-if="assign.scopeType !== 'all'" class="field">
          <label>{{ t('ur.scope.' + assign.scopeType) }}</label>
          <div class="inp filled">
            <select v-model="assign.scopeId">
              <option value="" disabled>{{ t('ur.choose') }}</option>
              <option v-for="o in scopeOptions" :key="o.id" :value="o.id">{{ o.name }}</option>
            </select>
          </div>
        </div>
      </div>
      <div v-if="assign.scopeType === 'group'" class="field">
        <label>{{ t('ur.position') }}</label>
        <div class="inp filled">
          <select v-model="assign.position">
            <option value="">{{ t('ur.noPosition') }}</option>
            <option v-for="p in POSITIONS" :key="p" :value="p">{{ positionText(p) }}</option>
          </select>
        </div>
        <div class="hint">{{ t('ur.positionHint') }}</div>
      </div>
      <div class="cg-actions">
        <button class="btn btn-primary" :disabled="busy" @click="saveAssign">{{ t('ur.assign') }}</button>
        <button class="btn btn-ghost" @click="assignFor = null">{{ t('common.cancel') }}</button>
      </div>
    </div>

    <template v-if="activeTab === 'pending'">
      <p style="font-size: 12.5px; color: var(--ink-600); margin-bottom: 14px; line-height: 1.5">{{ t('usr.pendingNote') }}</p>
      <div class="panel">
        <div v-if="loading" class="empty"><p>{{ t('common.loading') }}</p></div>
        <div v-else-if="!pendingAdmins.length" class="empty"><div class="ic">&#128100;</div><p>{{ t('usr.noPending') }}</p></div>
        <table v-else class="dtable">
          <thead>
            <tr><th>{{ t('usr.admin') }}</th><th>{{ t('usr.registered') }}</th><th></th></tr>
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
              <td><button class="btn btn-primary btn-sm" @click="goCreateGroup(a)">{{ t('usr.createGroup') }}</button></td>
            </tr>
          </tbody>
        </table>
      </div>
    </template>

    <div v-else class="panel">
      <div class="toolbar">
        <div class="search-input">
          <Svgs name="search" />
          <input v-model="search" type="search" :placeholder="t('ur.search')" />
        </div>
        <select v-model="roleFilter" class="filter-select">
          <option value="">{{ t('ur.allRoles') }}</option>
          <option v-for="r in ROLES" :key="r" :value="r">{{ roleText(r) }}</option>
          <option value="none">{{ t('ur.noRole') }}</option>
        </select>
      </div>
      <div v-if="loading" class="empty"><p>{{ t('common.loading') }}</p></div>
      <div v-else-if="!filtered.length" class="empty"><div class="ic">&#128100;</div><p>{{ t('users.empty') }}</p></div>
      <div v-else style="overflow-x: auto">
        <table class="dtable">
          <thead>
            <tr>
              <th>{{ t('users.name') }}</th>
              <th>{{ t('users.role') }}</th>
              <th>{{ t('ur.access') }}</th>
              <th>{{ t('common.status') }}</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="u in filtered" :key="u.id" :style="u.isActive ? '' : 'opacity: .55'">
              <td>
                <div class="tname">
                  <div class="tav" :style="{ background: avaColor(displayName(u)) }">{{ initials(displayName(u)) }}</div>
                  <div>
                    <div class="cell-main">{{ displayName(u) }}<span v-if="u.id === meId" class="cell-sub"> · {{ t('ur.you') }}</span></div>
                    <div class="cell-sub">{{ u.username }}</div>
                  </div>
                </div>
              </td>
              <td>
                <select :value="u.role" class="role-select" :disabled="busy || u.id === meId" @change="changeRole(u, $event)">
                  <option value="">{{ t('ur.noRole') }}</option>
                  <option v-for="r in ROLES" :key="r" :value="r">{{ roleText(r) }}</option>
                </select>
                <select v-if="u.role === 'staff'" :value="u.preset || 'viewer'" class="role-select" style="margin-left: 6px" :disabled="busy" @change="changePreset(u, $event)">
                  <option v-for="p in PRESETS" :key="p" :value="p">{{ t('ur.presets.' + p) }}</option>
                </select>
              </td>
              <td>
                <div style="display: flex; flex-wrap: wrap; gap: 6px; align-items: center">
                  <span v-if="u.role === 'super_admin'" class="badge green">{{ t('ur.scope.all') }}</span>
                  <span v-for="a in u.assignments || []" :key="a.id" class="badge" :class="a.scopeType === 'all' ? 'green' : 'gold'">
                    {{ a.scopeType === 'all' ? t('ur.scope.all') : t('ur.scope.' + a.scopeType) + ': ' + (a.scopeName || '—') }}
                    <template v-if="a.position"> ({{ positionText(a.position) }})</template>
                    <b style="cursor: pointer; margin-left: 4px" :title="t('ur.unassign')" @click="unassign(a)">&times;</b>
                  </span>
                  <a v-if="u.isActive && u.role && u.role !== 'super_admin'" href="#" class="link" style="font-size: 12px" @click.prevent="openAssign(u)">+ {{ t('ur.assign') }}</a>
                </div>
              </td>
              <td>
                <span
                  class="badge"
                  :class="[u.isActive ? 'green' : 'grey', u.id === meId ? '' : 'clickable']"
                  :title="u.id === meId ? '' : u.isActive ? t('struct.deactivate') : t('struct.reactivate')"
                  @click="u.id !== meId && toggleStatus(u)"
                >{{ u.isActive ? t('users.active') : t('users.inactive') }}</span>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
      <div style="height: 16px"></div>
    </div>
  </div>
</template>
