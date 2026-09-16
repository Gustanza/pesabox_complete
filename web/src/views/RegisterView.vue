<script setup>
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { requestOtp } from '@/api/auth'
import Svgs from '../components/Svgs.vue'

const router = useRouter()
const name = ref('')
const phone = ref('')
const loading = ref(false)
const error = ref('')

function isPhoneLike(value) {
  return /^\+?\d{9,13}$/.test(value.replace(/[\s-]/g, ''))
}

async function register() {
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
      query: { mode: 'register', username: phone.value, name: name.value }
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
      <p class="sub">One account to manage every savings group on the platform.</p>
      <div class="login-feature"><div class="fi"><Svgs name="chart" /></div><span>Real-time group oversight</span></div>
      <div class="login-feature"><div class="fi"><Svgs name="doc" /></div><span>Deep financial reports</span></div>
      <div class="login-feature"><div class="fi"><Svgs name="shield" /></div><span>Enterprise-grade security</span></div>
    </div>
    <div class="login-right">
      <div class="login-form">
        <h1>Create your account</h1>
        <div class="sub">Get started with PesaBox</div>

        <label class="field-label">Full name</label>
        <input v-model="name" type="text" placeholder="Jane Doe" style="margin-bottom: 18px" />

        <label class="field-label">Phone number</label>
        <input
          v-model="phone"
          type="tel"
          inputmode="tel"
          placeholder="0712 345 678"
          style="margin-bottom: 18px"
          @keyup.enter="register"
        />

        <div v-if="error" style="color: var(--danger); font-size: 13.5px; margin-bottom: 12px">
          {{ error }}
        </div>

        <button class="btn btn-primary btn-block" :disabled="loading" @click="register">
          {{ loading ? 'Sending code…' : 'Send me a code' }}
        </button>

        <div class="link">
          Already have an account? <router-link to="/login">Sign in</router-link>
        </div>
      </div>
    </div>
  </div>
</template>