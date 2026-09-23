<script setup>
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import { logout as logoutRequest } from '@/api/auth'
import { access, clearAccess } from '@/api/access'
import { setLocale } from '@/i18n'

const { t, te, locale } = useI18n()
const router = useRouter()

const role = computed(() => access.data?.role || '')
const runsGroup = computed(() => !!access.data?.homeGroup)

async function logout() {
  try {
    await logoutRequest()
  } catch (e) {
    // still leave
  }
  clearAccess()
  router.push('/login')
}
</script>

<template>
  <div class="na-wrap">
    <div class="na-card">
      <div class="lang-switch" style="justify-content: flex-end; margin-bottom: 18px">
        <button v-for="l in ['sw', 'en']" :key="l" :class="{ active: locale === l }" @click="setLocale(l)">
          {{ l.toUpperCase() }}
        </button>
      </div>
      <div class="na-mark">{{ t('app.name').charAt(0) }}</div>
      <h1>{{ t('access.noAccessTitle') }}</h1>
      <p v-if="runsGroup">
        {{ t('access.useApp', { group: access.data.homeGroup.name }) }}
      </p>
      <p v-else>{{ t('access.noRole') }}</p>
      <p v-if="role" class="na-role">
        {{ t('access.yourRole') }}: <b>{{ te('roles.' + role) ? t('roles.' + role) : role }}</b>
      </p>
      <button class="btn btn-outline" @click="logout">{{ t('nav.logout') }}</button>
    </div>
  </div>
</template>

<style scoped>
.na-wrap {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 24px;
  background: var(--cream, #faf8f4);
}
.na-card {
  max-width: 440px;
  width: 100%;
  background: #fff;
  border: 1px solid var(--line);
  border-radius: var(--radius-lg, 16px);
  box-shadow: var(--shadow);
  padding: 28px 30px 30px;
  text-align: center;
}
.na-mark {
  width: 52px;
  height: 52px;
  margin: 0 auto 16px;
  border-radius: 14px;
  display: flex;
  align-items: center;
  justify-content: center;
  font: 800 24px 'Plus Jakarta Sans', sans-serif;
  color: #fff;
  background: var(--green-600, #18a672);
}
h1 {
  font-size: 20px;
  margin-bottom: 10px;
}
p {
  font-size: 14px;
  color: var(--ink-600);
  line-height: 1.55;
  margin-bottom: 14px;
}
.na-role {
  font-size: 13px;
}
</style>
