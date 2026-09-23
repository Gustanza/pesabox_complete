<script setup>
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { updateProfile } from '@/api/auth'
import Svgs from '../components/Svgs.vue'

const router = useRouter()
const { t } = useI18n()
const fullName = ref('')
const email = ref('')
const loading = ref(false)
const error = ref('')

async function save() {
  const name = fullName.value.trim()
  if (!name) {
    error.value = t('prof.enterName')
    return
  }

  const [firstName, ...rest] = name.split(/\s+/)
  const lastName = rest.join(' ')

  loading.value = true
  error.value = ''

  try {
    await updateProfile({ firstName, lastName, email: email.value.trim() })
    router.push('/')
  } catch (e) {
    error.value = e.message || t('prof.saveFailed')
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="login-split">
    <div class="login-left">
      <div class="login-shield"><Svgs name="shield" width="60" height="60" /></div>
      <h1>{{ t('app.name') }} <span>Admin</span></h1>
      <p class="sub">{{ t('prof.tagline') }}</p>
      <div class="login-feature"><div class="fi"><Svgs name="chart" /></div><span>{{ t('login.f1') }}</span></div>
      <div class="login-feature"><div class="fi"><Svgs name="doc" /></div><span>{{ t('login.f2') }}</span></div>
      <div class="login-feature"><div class="fi"><Svgs name="shield" /></div><span>{{ t('login.f3') }}</span></div>
    </div>
    <div class="login-right">
      <div class="login-form">
        <h1>{{ t('prof.completeTitle') }}</h1>
        <div class="sub">{{ t('prof.completeSub') }}</div>

        <label class="field-label">{{ t('prof.fullName') }}</label>
        <input v-model="fullName" type="text" placeholder="Jane Doe" style="margin-bottom: 18px" @keyup.enter="save" />

        <label class="field-label">{{ t('struct.email') }} <span style="color: var(--ink-400); font-weight: 400">({{ t('prof.optional') }})</span></label>
        <input v-model="email" type="email" placeholder="jane@example.com" style="margin-bottom: 18px" @keyup.enter="save" />

        <div v-if="error" style="color: var(--danger); font-size: 13.5px; margin-bottom: 12px">
          {{ error }}
        </div>

        <button class="btn btn-primary btn-block" :disabled="loading" @click="save">
          {{ loading ? t('struct.saving') : t('login.continue') }}
        </button>
      </div>
    </div>
  </div>
</template>
