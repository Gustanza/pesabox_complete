<script setup>
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { requestOtp } from '@/api/auth'

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
  <div class="login-wrap">
    <div class="login-card">
      <div class="login-mark">P</div>
      <h2>Create your account</h2>
      <div class="sub">Get started with PesaBox</div>

      <div class="field">
        <label>Full name</label>
        <div class="inp">
          <input v-model="name" type="text" placeholder="Jane Doe" />
        </div>
      </div>

      <div class="field">
        <label>Phone number</label>
        <div class="inp">
          <input
            v-model="phone"
            type="tel"
            inputmode="tel"
            placeholder="0712 345 678"
            @keyup.enter="register"
          />
        </div>
      </div>

      <div v-if="error" style="color: var(--danger); font-size: 13px; margin-bottom: 12px">
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
</template>
