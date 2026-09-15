<script setup>
import { computed, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { createGroup, updateGroup, getGroup } from '../api/groups.js'
import { currentUser } from '../api/auth.js'
import { initials, avaColor } from '../data/mock.js'

const route = useRoute()
const router = useRouter()

// This same form handles both /groups/create and /groups/:id/edit — a
// group's basic details are edited the same way they're created. The
// constitution and membership aren't part of either: they're populated by
// the group itself once the dedicated group-management app exists.
const editingId = route.params.id || ''
const isEdit = !!editingId

// Coming from the "Group Admins without a group" list pre-fills the
// responsible officer and links the new group back to that admin's account.
// Otherwise a newly created group is linked to whoever is creating it.
let createdBy = route.query.adminId || ''

const form = ref({
  name: '',
  region: '',
  district: '',
  ward: '',
  village: '',
  formationDate: '',
  meetingFrequency: 'Weekly',
  adminName: route.query.adminName || '',
  adminPhone: route.query.adminPhone || '',
  projectedEndDate: ''
})

const loading = ref(isEdit)
const existingCycleTotal = ref(0)

function toDateInput(value) {
  if (!value) return ''
  const d = new Date(value)
  return isNaN(d) ? '' : d.toISOString().slice(0, 10)
}

onMounted(async () => {
  if (isEdit) {
    try {
      const g = await getGroup(editingId)
      if (g) {
        form.value.name = g.name || ''
        form.value.region = g.region || ''
        form.value.district = g.district || ''
        form.value.ward = g.ward || ''
        form.value.village = g.village || ''
        form.value.formationDate = toDateInput(g.formationDate)
        form.value.meetingFrequency = g.meetingFrequency || 'Weekly'
        form.value.adminName = g.adminName || ''
        form.value.adminPhone = g.adminPhone || ''
        existingCycleTotal.value = g.cycleTotal || 0
      }
    } catch (e) {
      error.value = e.message || 'Failed to load group'
    } finally {
      loading.value = false
    }
    return
  }

  if (!createdBy) {
    const me = await currentUser()
    if (me) createdBy = me.id
  }
})

const preview = computed(() => form.value.name.trim() || 'New Group')

// Cycle length (number of meetings) is derived from formation date ->
// projected end date at the chosen meeting frequency, rather than typed in
// directly. Falls back to today when no formation date is set yet. Left
// blank while editing, an existing group's cycle length is simply kept.
const cycleTotal = computed(() => {
  if (!form.value.projectedEndDate) return 0

  const start = form.value.formationDate ? new Date(form.value.formationDate) : new Date()
  const end = new Date(form.value.projectedEndDate)
  if (isNaN(start) || isNaN(end) || end <= start) return 0

  if (form.value.meetingFrequency === 'Monthly') {
    let months = (end.getFullYear() - start.getFullYear()) * 12 + (end.getMonth() - start.getMonth())
    if (end.getDate() < start.getDate()) months -= 1
    return Math.max(1, months)
  }

  const days = Math.round((end - start) / 86400000)
  const perMeetingDays = form.value.meetingFrequency === 'Biweekly' ? 14 : 7
  return Math.max(1, Math.ceil(days / perMeetingDays))
})

const saving = ref(false)
const error = ref('')

function backBreadcrumb() {
  router.push(isEdit ? '/groups/' + editingId : '/groups')
}

async function submit() {
  if (!form.value.name.trim()) {
    error.value = 'Group name is required'
    return
  }

  saving.value = true
  error.value = ''

  try {
    const input = { ...form.value }
    delete input.projectedEndDate
    if (!input.formationDate) delete input.formationDate
    if (cycleTotal.value) input.cycleTotal = cycleTotal.value

    if (isEdit) {
      const result = await updateGroup(editingId, input)
      if (!result?.success) {
        error.value = result?.message || 'Failed to save changes'
        return
      }
      router.push('/groups/' + editingId)
    } else {
      if (createdBy) input.createdBy = createdBy
      const result = await createGroup(input)
      if (!result?.success) {
        error.value = result?.message || 'Failed to create group'
        return
      }
      router.push('/groups/' + result.data.id)
    }
  } catch (e) {
    error.value = e.message || 'Failed to save group'
  } finally {
    saving.value = false
  }
}
</script>

<template>
  <div>
    <div class="crumb">
      <b @click="backBreadcrumb">{{ isEdit ? preview : 'Groups' }}</b> / {{ isEdit ? 'Edit' : 'Create Group' }}
    </div>
    <div class="page-head">
      <div>
        <h1>{{ isEdit ? 'Edit Group' : 'Create Group' }}</h1>
        <p>{{ isEdit ? "Update this group's basic details." : 'Set up a new savings group.' }}</p>
      </div>
    </div>

    <div v-if="error" style="color: var(--danger); font-size: 13px; margin-bottom: 12px">
      {{ error }}
    </div>

    <div v-if="loading" class="empty"><p>Loading group…</p></div>
    <div v-else class="cg-card">
      <div class="cg-head">
        <div class="cg-avatar" :style="{ background: avaColor(preview) }">{{ initials(preview) }}</div>
        <div>
          <div class="cg-title">{{ preview }}</div>
          <div class="cg-sub">{{ [form.village, form.district, form.region].filter(Boolean).join(', ') || 'Location not set yet' }}</div>
        </div>
      </div>

      <div class="cg-section">Group details</div>
      <div class="field">
        <label>Group name</label>
        <div class="inp filled"><input v-model="form.name" placeholder="e.g. Upendo Vikoba" /></div>
      </div>
      <div class="field-row">
        <div class="field">
          <label>Region</label>
          <div class="inp filled"><input v-model="form.region" placeholder="e.g. Dar es Salaam" /></div>
        </div>
        <div class="field">
          <label>District</label>
          <div class="inp filled"><input v-model="form.district" placeholder="e.g. Ilala" /></div>
        </div>
      </div>
      <div class="field-row">
        <div class="field">
          <label>Ward <span class="opt">(optional)</span></label>
          <div class="inp filled"><input v-model="form.ward" placeholder="e.g. Kariakoo" /></div>
        </div>
        <div class="field">
          <label>Village / Street <span class="opt">(optional)</span></label>
          <div class="inp filled"><input v-model="form.village" placeholder="e.g. Mchikichini" /></div>
        </div>
      </div>
      <div class="field-row">
        <div class="field">
          <label>Formation date</label>
          <div class="inp filled"><input v-model="form.formationDate" type="date" /></div>
        </div>
        <div class="field">
          <label>Meeting frequency</label>
          <div class="inp filled">
            <select v-model="form.meetingFrequency">
              <option>Weekly</option>
              <option>Biweekly</option>
              <option>Monthly</option>
            </select>
          </div>
        </div>
      </div>
      <div class="field">
        <label>Projected end date <span class="opt">(optional)</span></label>
        <div class="inp filled"><input v-model="form.projectedEndDate" type="date" /></div>
        <div class="hint">
          <template v-if="cycleTotal">≈ {{ cycleTotal }} meetings in this cycle</template>
          <template v-else-if="isEdit && existingCycleTotal">
            Current cycle length is {{ existingCycleTotal }} meetings — set an end date to change it
          </template>
          <template v-else>Set an end date to work out the cycle length automatically</template>
        </div>
      </div>

      <div class="cg-section">Responsible officer</div>
      <div class="field-row">
        <div class="field">
          <label>Admin / Chairperson name</label>
          <div class="inp filled"><input v-model="form.adminName" placeholder="e.g. Neema Joseph" /></div>
        </div>
        <div class="field">
          <label>Admin phone</label>
          <div class="inp filled"><input v-model="form.adminPhone" type="tel" placeholder="0712 345 678" /></div>
        </div>
      </div>

      <div class="cg-actions">
        <button class="btn btn-primary" :disabled="saving" @click="submit">
          {{ saving ? 'Saving…' : isEdit ? 'Save changes' : 'Create group' }}
        </button>
        <button class="btn btn-ghost" @click="backBreadcrumb">Cancel</button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.cg-card {
  max-width: 720px;
  background: #fff;
  border: 1px solid var(--line);
  border-radius: var(--radius-lg);
  box-shadow: var(--shadow);
  padding: 28px 32px 32px;
}

.cg-head {
  display: flex;
  align-items: center;
  gap: 14px;
  padding-bottom: 20px;
  margin-bottom: 20px;
  border-bottom: 1px solid var(--line);
}

.cg-avatar {
  width: 52px;
  height: 52px;
  min-width: 52px;
  border-radius: 14px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #fff;
  font-weight: 800;
  font-size: 17px;
  font-family: 'Plus Jakarta Sans', sans-serif;
  transition: background 0.15s ease;
}

.cg-title {
  font-size: 17px;
  font-weight: 800;
  color: var(--ink-900);
}

.cg-sub {
  font-size: 12.5px;
  color: var(--ink-400);
  margin-top: 2px;
}

.cg-section {
  font-size: 12px;
  font-weight: 800;
  letter-spacing: 0.04em;
  text-transform: uppercase;
  color: var(--green-600);
  margin: 24px 0 14px;
}

.cg-section:first-of-type {
  margin-top: 0;
}

.field-row {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 16px;
}

@media (max-width: 560px) {
  .field-row {
    grid-template-columns: 1fr;
  }
}

.inp.filled {
  background: var(--cream);
  border: 1.5px solid transparent;
  transition: border-color 0.15s ease, background 0.15s ease;
}

.inp.filled:focus-within {
  background: #fff;
  border-color: var(--green-600);
}

.inp.filled select {
  border: none;
  outline: none;
  flex: 1;
  font-size: 14px;
  font-family: 'Inter', sans-serif;
  color: var(--ink-900);
  background: transparent;
}

.cg-actions {
  display: flex;
  gap: 10px;
  margin-top: 24px;
  padding-top: 20px;
  border-top: 1px solid var(--line);
}
</style>
