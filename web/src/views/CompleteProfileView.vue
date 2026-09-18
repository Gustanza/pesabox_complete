<script setup>
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { updateProfile } from '@/api/auth'
import Svgs from '../components/Svgs.vue'

const router = useRouter()
const fullName = ref('')
const email = ref('')
const loading = ref(false)
const error = ref('')

async function save() {
  const name = fullName.value.trim()
  if (!name) {
    error.value = 'Enter your full name'
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
    error.value = e.message || 'Unable to save your profile right now'
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="login-split">
    <div class="login-left">
      <div class="login-shield"><Svgs name="shield" width="60" height="60" /></div>
      <h1>PesaBox <span>Admin</span></h1>
      <p class="sub">One account to manage every savings group on the platform.</p>
      <div class="login-feature"><div class="fi"><Svgs name="chart" /></div><span>Real-time group oversight</span></div>
      <div class="login-feature"><div class="fi"><Svgs name="doc" /></div><span>Deep financial reports</span></div>
      <div class="login-feature"><div class="fi"><Svgs name="shield" /></div><span>Enterprise-grade security</span></div>
    </div>
    <div class="login-right">
      <div class="login-form">
        <h1>Complete your profile</h1>
        <div class="sub">Just a couple of details before you go in</div>

        <label class="field-label">Full name</label>
        <input v-model="fullName" type="text" placeholder="Jane Doe" style="margin-bottom: 18px" @keyup.enter="save" />

        <label class="field-label">Email <span style="color: var(--ink-400); font-weight: 400">(optional)</span></label>
        <input v-model="email" type="email" placeholder="jane@example.com" style="margin-bottom: 18px" @keyup.enter="save" />

        <div v-if="error" style="color: var(--danger); font-size: 13.5px; margin-bottom: 12px">
          {{ error }}
        </div>

        <button class="btn btn-primary btn-block" :disabled="loading" @click="save">
          {{ loading ? 'Saving…' : 'Continue' }}
        </button>
      </div>
    </div>
  </div>
</template>
