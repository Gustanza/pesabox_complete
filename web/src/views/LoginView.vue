<script setup>
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { setLocale } from '@/i18n'
import { requestOtp } from '@/api/auth'
import Svgs from '../components/Svgs.vue'
import logoIcon from '@/assets/logo-icon-reverse.svg'

const router = useRouter()
const { t, locale } = useI18n()
const phone = ref('')
const remember = ref(false)
const loading = ref(false)
const error = ref('')

// The field sits after a fixed +255 prefix: accept 712345678, 0712345678 or a
// pasted +255712345678, and send the full 255XXXXXXXXX number.
function fullPhone(value) {
  let digits = value.replace(/\D/g, '')
  if (digits.startsWith('255') && digits.length === 12) digits = digits.slice(3)
  else if (digits.startsWith('0') && digits.length === 10) digits = digits.slice(1)
  return /^[67]\d{8}$/.test(digits) ? '255' + digits : ''
}

async function signIn() {
  const username = fullPhone(phone.value)
  if (!username) {
    error.value = t('auth.phoneInvalid')
    return
  }

  loading.value = true
  error.value = ''

  try {
    const result = await requestOtp(username)
    if (!result?.status) {
      error.value = result?.message || t('login.unable')
      return
    }
    router.push({
      path: '/otp',
      query: { mode: 'login', username, remember: remember.value ? '1' : '' }
    })
  } catch (e) {
    // The server refuses numbers that haven't been registered (invite-only).
    error.value = /not registered/i.test(e.message) ? t('auth.notRegistered')
      : /deactivated/i.test(e.message) ? t('auth.deactivated')
      : e.message
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="login-split">
    <div class="lang-switch login-lang">
      <button v-for="l in ['sw', 'en']" :key="l" :class="{ active: locale === l }" @click="setLocale(l)">{{ l.toUpperCase() }}</button>
    </div>
    <div class="login-left">
      <div class="login-shield"><img :src="logoIcon" :alt="t('app.name')" /></div>
      <h1>{{ t('app.name') }} <span>Admin</span></h1>
      <p class="sub">{{ t('login.tagline') }}</p>
      <div class="login-feature"><div class="fi"><Svgs name="chart" /></div><span>{{ t('login.f1') }}</span></div>
      <div class="login-feature"><div class="fi"><Svgs name="doc" /></div><span>{{ t('login.f2') }}</span></div>
      <div class="login-feature"><div class="fi"><Svgs name="shield" /></div><span>{{ t('login.f3') }}</span></div>
    </div>
    <div class="login-right">
      <div class="login-form">
        <h1>{{ t('login.welcome') }}</h1>
        <div class="sub">{{ t('login.sub') }}</div>

        <label class="field-label">{{ t('auth.phone') }}</label>
        <div class="phone-field">
          <span class="phone-prefix">
            <svg class="tz-flag" viewBox="0 0 30 20" aria-hidden="true">
              <path d="M0 0h30v20H0z" fill="#1EB53A" />
              <path d="M0 20L30 0v20z" fill="#00A3DD" />
              <path d="M0 20L30 0" stroke="#FCD116" stroke-width="7.5" />
              <path d="M0 20L30 0" stroke="#000" stroke-width="4.5" />
            </svg>
            +255
          </span>
          <input
            v-model="phone"
            type="tel"
            inputmode="tel"
            autocomplete="tel-national"
            placeholder="712 345 678"
            @keyup.enter="signIn"
          />
        </div>

        <div v-if="error" style="color: var(--danger); font-size: 13.5px; margin-bottom: 12px">
          {{ error }}
        </div>

        <div class="chk-row">
          <input v-model="remember" type="checkbox" />
          {{ t('auth.remember') }}
        </div>

        <button class="btn btn-primary btn-block" :disabled="loading" @click="signIn">
          {{ loading ? t('auth.sendingCode') : t('login.continue') }}
        </button>

        <div style="display: flex; align-items: center; justify-content: center; gap: 8px; margin-top: 26px; color: var(--ink-400); font-size: 12.5px">
          <Svgs name="shield" width="14" height="14" /> {{ t('login.logged') }}
        </div>
      </div>
    </div>
  </div>
</template>
