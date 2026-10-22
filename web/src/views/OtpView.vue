<script setup>
import { ref, onMounted, onUnmounted, computed } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { setLocale } from '@/i18n'
import { requestOtp, verifyOtp } from '@/api/auth'
import logoIcon from '@/assets/logo-icon.svg'
import logoReverse from '@/assets/logo-icon-reverse.svg'

const router = useRouter()
const { t, locale } = useI18n()
const route = useRoute()

const username = route.query.username || ''
const mode = route.query.mode || 'login'
// 255712345678 → +255 712 345 678
const shownPhone = /^255\d{9}$/.test(username)
  ? '+255 ' + username.slice(3).replace(/(\d{3})(\d{3})(\d{3})/, '$1 $2 $3')
  : username

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
    error.value = t('auth.enterDigits', { n: CODE_LENGTH })
    return
  }

  loading.value = true
  error.value = ''

  try {
    await verifyOtp(username, code.value)
    router.push('/')
  } catch (e) {
    error.value = e.message || t('auth.invalidCode')
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="login-wrap">
    <div class="auth-brand">
      <img :src="logoReverse" alt="" />
      <span>{{ t('app.name') }}</span>
    </div>
    <div class="lang-switch login-lang">
      <button v-for="l in ['sw', 'en']" :key="l" :class="{ active: locale === l }" @click="setLocale(l)">{{ l.toUpperCase() }}</button>
    </div>
    <div class="login-card" style="text-align: center">
      <img class="login-logo" :src="logoIcon" :alt="t('app.name')" />
      <h2>{{ t('auth.verifyTitle') }}</h2>
      <div class="sub">
        {{ t('auth.enterCode', { n: CODE_LENGTH }) }} <strong>{{ shownPhone }}</strong
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
          {{ t('auth.didntReceive') }} {{ t('auth.resendIn', { s: String(seconds).padStart(2, '0') }) }}
        </template>
        <template v-else>
          {{ t('auth.didntReceive') }} <a href="#" @click.prevent="resend">{{ t('auth.resend') }}</a>
        </template>
      </div>
      <div v-if="error" style="color: var(--danger); font-size: 13px; margin-bottom: 18px">
        {{ error }}
      </div>
      <button class="btn btn-primary btn-block" :disabled="loading" @click="verify">
        {{ loading ? t('auth.verifying') : t('auth.verify') }}
      </button>
    </div>
  </div>
</template>
