<script setup>
import { onMounted, ref } from 'vue'
import { currentUser, updateProfile } from '@/api/auth'
import { initials, avaColor } from '../data/mock.js'

const fullName = ref('')
const email = ref('')
const phone = ref('')
const loading = ref(true)
const saving = ref(false)
const error = ref('')
const saved = ref(false)

const preview = () => fullName.value.trim() || phone.value || '?'

onMounted(async () => {
  const me = await currentUser()
  if (me) {
    fullName.value = [me.firstName, me.lastName].filter(Boolean).join(' ')
    email.value = me.email || ''
    phone.value = me.username || me.phone || ''
  }
  loading.value = false
})

async function save() {
  const name = fullName.value.trim()
  if (!name) {
    error.value = 'Enter your full name'
    return
  }

  const [firstName, ...rest] = name.split(/\s+/)
  const lastName = rest.join(' ')

  saving.value = true
  error.value = ''
  saved.value = false

  try {
    await updateProfile({ firstName, lastName, email: email.value.trim() })
    saved.value = true
  } catch (e) {
    error.value = e.message || 'Unable to save your profile right now'
  } finally {
    saving.value = false
  }
}
</script>

<template>
  <div class="page-head">
    <div>
      <h1>My Profile</h1>
      <p>Update the name and email shown across the admin dashboard.</p>
    </div>
  </div>

  <div v-if="loading" class="empty"><p>Loading profile…</p></div>
  <div v-else class="cg-card">
    <div class="cg-head">
      <div class="cg-avatar" :style="{ background: avaColor(preview()) }">{{ initials(preview()) }}</div>
      <div>
        <div class="cg-title">{{ fullName || 'Your name' }}</div>
        <div class="cg-sub">{{ phone }}</div>
      </div>
    </div>

    <div class="cg-section">Your details</div>
    <div class="field">
      <label>Full name</label>
      <div class="inp filled"><input v-model="fullName" placeholder="Jane Doe" @keyup.enter="save" /></div>
    </div>
    <div class="field">
      <label>Email <span class="opt">(optional)</span></label>
      <div class="inp filled"><input v-model="email" type="email" placeholder="jane@example.com" @keyup.enter="save" /></div>
    </div>
    <div class="field">
      <label>Phone number</label>
      <div class="inp filled"><input :value="phone" disabled /></div>
      <div class="hint">This is your login number — it can't be changed here.</div>
    </div>

    <div v-if="error" style="color: var(--danger); font-size: 13px; margin: 8px 0">{{ error }}</div>
    <div v-if="saved" style="color: var(--green-600); font-size: 13px; margin: 8px 0">Profile saved.</div>

    <div class="cg-actions">
      <button class="btn btn-primary" :disabled="saving" @click="save">
        {{ saving ? 'Saving…' : 'Save changes' }}
      </button>
    </div>
  </div>
</template>
