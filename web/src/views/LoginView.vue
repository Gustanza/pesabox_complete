<script setup>
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { requestOtp } from '@/api/auth'
import Svgs from '../components/Svgs.vue'

const router = useRouter()
const phone = ref('')
const remember = ref(false)
const loading = ref(false)
const error = ref('')

function isPhoneLike(value) {
  return /^\+?\d{9,13}$/.test(value.replace(/[\s-]/g, ''))
}

async function signIn() {
  if (!isPhoneLike(phone.value)) {
    error.value = 'Enter a valid phone number, e.g. 0712 345 678'
    return
  }

  loading.value = true
  error.value = ''

  try {
    const result = await requestOtp(phone.value)
    if (!result?.status) {
      error.value = result?.message || 'Unable to send a code right now'
      return
    }
    router.push({
      path: '/otp',
      query: { mode: 'login', username: phone.value, remember: remember.value ? '1' : '' }
    })
  } catch (e) {
    error.value = e.message
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
      <p class="sub">Oversee every savings group, member and transaction on one platform.</p>
      <div class="login-feature"><div class="fi"><Svgs name="chart" /></div><span>Real-time group oversight</span></div>
      <div class="login-feature"><div class="fi"><Svgs name="doc" /></div><span>Deep financial reports</span></div>
      <div class="login-feature"><div class="fi"><Svgs name="shield" /></div><span>Enterprise-grade security</span></div>
    </div>
    <div class="login-right">
      <div class="login-form">
        <h1>Welcome back</h1>
        <div class="sub">Sign in to your admin account</div>

        <label class="field-label">Phone number</label>
        <input
          v-model="phone"
          type="tel"
          inputmode="tel"
          placeholder="0712 345 678"
          style="margin-bottom: 18px"
          @keyup.enter="signIn"
        />

        <div v-if="error" style="color: var(--danger); font-size: 13.5px; margin-bottom: 12px">
          {{ error }}
        </div>

        <div class="chk-row">
          <input v-model="remember" type="checkbox" />
          Remember this device
        </div>

        <button class="btn btn-primary btn-block" :disabled="loading" @click="signIn">
          {{ loading ? 'Sending code…' : 'Continue' }}
        </button>

        <div class="link">
          No account? <router-link to="/register">Register</router-link>
        </div>

        <div style="display: flex; align-items: center; justify-content: center; gap: 8px; margin-top: 26px; color: var(--ink-400); font-size: 12.5px">
          <Svgs name="shield" width="14" height="14" /> Access is logged for security purposes
        </div>
      </div>
    </div>
  </div>
</template>