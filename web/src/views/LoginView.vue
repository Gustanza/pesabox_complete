<script setup>
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { requestOtp } from '@/api/auth'

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
  <div class="login-wrap">
    <div class="login-card">
      <div class="login-mark">P</div>
      <h2>PesaBox</h2>
      <div class="sub">Super Admin Portal</div>

      <div class="field">
        <label>Phone number</label>
        <div class="inp">
          <input
            v-model="phone"
            type="tel"
            inputmode="tel"
            placeholder="0712 345 678"
            @keyup.enter="signIn"
          />
        </div>
      </div>

      <div class="chk-row">
        <input v-model="remember" type="checkbox" />
        Remember this device
      </div>

      <div v-if="error" style="color: var(--danger); font-size: 13px; margin-bottom: 12px">
        {{ error }}
      </div>

      <button class="btn btn-primary btn-block" :disabled="loading" @click="signIn">
        {{ loading ? 'Sending code…' : 'Continue' }}
      </button>
      <div class="link">
        No account? <router-link to="/register">Register</router-link>
      </div>
    </div>
  </div>
</template>
