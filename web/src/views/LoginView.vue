<script setup>
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { setLocale } from '@/i18n'
import { requestOtp } from '@/api/auth'
import Svgs from '../components/Svgs.vue'

const router = useRouter()
const { t, locale } = useI18n()
const phone = ref('')
const remember = ref(false)
const loading = ref(false)
const error = ref('')

function isPhoneLike(value) {
  return /^\+?\d{9,13}$/.test(value.replace(/[\s-]/g, ''))
}

async function signIn() {
  if (!isPhoneLike(phone.value)) {
    error.value = t('auth.phoneInvalid')
    return
  }

  loading.value = true
  error.value = ''

  try {
    const result = await requestOtp(phone.value)
    if (!result?.status) {
      error.value = result?.message || t('login.unable')
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
      <h1>{{ t('app.name') }} <span>Admin</span></h1>
      <p class="sub">{{ t('login.tagline') }}</p>
      <div class="login-feature"><div class="fi"><Svgs name="chart" /></div><span>{{ t('login.f1') }}</span></div>
      <div class="login-feature"><div class="fi"><Svgs name="doc" /></div><span>{{ t('login.f2') }}</span></div>
      <div class="login-feature"><div class="fi"><Svgs name="shield" /></div><span>{{ t('login.f3') }}</span></div>
    </div>
    <div class="login-right">
      <div class="login-form">
        <div class="lang-switch" style="float: right; margin: 0">
          <button v-for="l in ['sw', 'en']" :key="l" :class="{ active: locale === l }" @click="setLocale(l)">{{ l.toUpperCase() }}</button>
        </div>
        <h1>{{ t('login.welcome') }}</h1>
        <div class="sub">{{ t('login.sub') }}</div>

        <label class="field-label">{{ t('auth.phone') }}</label>
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