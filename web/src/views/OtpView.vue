<script setup>
import { ref, onMounted, onUnmounted, computed } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { requestOtp, verifyOtp } from '@/api/auth'

const router = useRouter()
const route = useRoute()

const username = route.query.username || ''
const mode = route.query.mode || 'login'

const CODE_LENGTH = 4
const digits = ref(Array(CODE_LENGTH).fill(''))
const inputs = ref([])
const seconds = ref(42)
const loading = ref(false)
const error = ref('')
let timer = null

const code = computed(() => digits.value.join(''))

function onInput(index) {
  const val = digits.value[index]
  if (val.length > 1) digits.value[index] = val.slice(-1)
  if (val && index < CODE_LENGTH - 1) {
    inputs.value[index + 1]?.focus()
  }
}

function onKeydown(index, e) {
  if (e.key === 'Backspace' && !digits.value[index] && index > 0) {
    inputs.value[index - 1]?.focus()
  }
}

function onFocus(e) {
  e.target.select()
}

function startTimer() {
  clearInterval(timer)
  seconds.value = 42
  timer = setInterval(() => {
    if (seconds.value > 0) seconds.value--
  }, 1000)
}

onMounted(() => {
  if (!username) {
    router.replace(mode === 'register' ? '/register' : '/login')
    return
  }
  startTimer()
})

onUnmounted(() => {
  clearInterval(timer)
})

async function resend() {
  if (seconds.value > 0) return
  error.value = ''
  try {
    await requestOtp(username)
    startTimer()
  } catch (e) {
    error.value = e.message
  }
}

async function verify() {
  if (code.value.length < CODE_LENGTH) {
    error.value = `Enter the ${CODE_LENGTH}-digit code`
    return
  }

  loading.value = true
  error.value = ''

  try {
    await verifyOtp(username, code.value)
    router.push('/')
  } catch (e) {
    error.value = e.message || 'Invalid or expired code'
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="login-wrap">
    <div class="login-card" style="text-align: center">
      <div class="login-mark">P</div>
      <h2>Verify your identity</h2>
      <div class="sub">
        Enter the {{ CODE_LENGTH }}-digit code texted to <strong>{{ username }}</strong
        >.
      </div>
      <div class="otp-row">
        <input
          v-for="(_, i) in CODE_LENGTH"
          :key="i"
          ref="inputs"
          v-model="digits[i]"
          maxlength="1"
          inputmode="numeric"
          @input="onInput(i)"
          @keydown="onKeydown(i, $event)"
          @focus="onFocus"
        />
      </div>
      <div style="font-size: 12px; color: var(--ink-400); margin-bottom: 18px;">
        <template v-if="seconds > 0">
          Didn't receive it? Resend code in {{ String(seconds).padStart(2, '0') }}
        </template>
        <template v-else>
          Didn't receive it? <a href="#" @click.prevent="resend">Resend code</a>
        </template>
      </div>
      <div v-if="error" style="color: var(--danger); font-size: 13px; margin-bottom: 18px">
        {{ error }}
      </div>
      <button class="btn btn-primary btn-block" :disabled="loading" @click="verify">
        {{ loading ? 'Verifying…' : 'Verify' }}
      </button>
    </div>
  </div>
</template>
