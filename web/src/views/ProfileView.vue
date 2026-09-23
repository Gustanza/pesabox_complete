<script setup>
import { onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { currentUser, updateProfile } from '@/api/auth'
import { initials, avaColor } from '../data/mock.js'

const { t } = useI18n()
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
    error.value = t('prof.enterName')
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
    error.value = e.message || t('prof.saveFailed')
  } finally {
    saving.value = false
  }
}
</script>

<template>
  <div class="page-head">
    <div>
      <h1>{{ t('prof.title') }}</h1>
      <p>{{ t('prof.subtitle') }}</p>
    </div>
  </div>

  <div v-if="loading" class="empty"><p>{{ t('common.loading') }}</p></div>
  <div v-else class="cg-card">
    <div class="cg-head">
      <div class="cg-avatar" :style="{ background: avaColor(preview()) }">{{ initials(preview()) }}</div>
      <div>
        <div class="cg-title">{{ fullName || t('prof.yourName') }}</div>
        <div class="cg-sub">{{ phone }}</div>
      </div>
    </div>

    <div class="cg-section">{{ t('prof.details') }}</div>
    <div class="field">
      <label>{{ t('prof.fullName') }}</label>
      <div class="inp filled"><input v-model="fullName" placeholder="Jane Doe" @keyup.enter="save" /></div>
    </div>
    <div class="field">
      <label>{{ t('struct.email') }} <span class="opt">({{ t('prof.optional') }})</span></label>
      <div class="inp filled"><input v-model="email" type="email" placeholder="jane@example.com" @keyup.enter="save" /></div>
    </div>
    <div class="field">
      <label>{{ t('auth.phone') }}</label>
      <div class="inp filled"><input :value="phone" disabled /></div>
      <div class="hint">{{ t('prof.phoneHint') }}</div>
    </div>

    <div v-if="error" style="color: var(--danger); font-size: 13px; margin: 8px 0">{{ error }}</div>
    <div v-if="saved" style="color: var(--green-600); font-size: 13px; margin: 8px 0">{{ t('prof.saved') }}</div>

    <div class="cg-actions">
      <button class="btn btn-primary" :disabled="saving" @click="save">
        {{ saving ? t('struct.saving') : t('cg.saveChanges') }}
      </button>
    </div>
  </div>
</template>
