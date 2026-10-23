<script setup>
import { computed, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute, useRouter } from 'vue-router'
import Svgs from './Svgs.vue'
import logoDark from '@/assets/logo-dark.svg'
import { initials, avaColor } from '../data/mock.js'
import { logout as logoutRequest, currentUser } from '@/api/auth'
import { access, can, clearAccess, loadAccess } from '@/api/access'
import { setLocale } from '@/i18n'

const { t, te, locale } = useI18n()
function roleText(role) {
  return te('roles.' + role) ? t('roles.' + role) : t('roles.user')
}

const route = useRoute()
const router = useRouter()

// Sidebar entries, each shown only when the role has the permission
// (server/access.go). The server enforces the same rules on every request.
const NAV = [
  { key: 'dashboard', icon: 'dash', route: '/dashboard', perm: 'dashboard.view' },
  { key: 'monitoring', icon: 'eye', route: '/monitoring', perm: 'reports.view' },
  { key: 'groups', icon: 'groups', route: '/groups', perm: 'dashboard.view' },
  { key: 'structure', icon: 'chart', route: '/structure', perm: 'structure.view' },
  { key: 'users', icon: 'user', route: '/users', perm: 'platform.manage' },
  { key: 'reports', icon: 'rep', route: '/reports', perm: 'reports.view' },
  { key: 'sms', icon: 'sms', route: '/sms', perm: 'sms.view' },
  { key: 'audit', icon: 'audit', route: '/audit', perm: 'audit.view' },
  { key: 'app', icon: 'phone', route: '/app', perm: 'platform.manage' },
  { key: 'settings', icon: 'settings', route: '/settings', perm: 'platform.manage' }
]
const navItems = computed(() => (access.loaded ? NAV.filter((n) => can(n.perm)) : []))

const me = ref(null)
const meName = computed(() => {
  if (!me.value) return ''
  return [me.value.firstName, me.value.lastName].filter(Boolean).join(' ') || me.value.username
})
const role = computed(() => access.data?.role || '')

async function loadMe() {
  me.value = await currentUser()
  loadAccess()
}

onMounted(loadMe)
// AppLayout stays mounted across child-route navigation, so the topbar chip
// would otherwise keep showing stale data after e.g. saving /profile and
// navigating back — refetch on every route change instead.
watch(() => route.path, loadMe)

// Phones/tablets (≤900px): the sidebar becomes a drawer opened from the
// topbar menu button and closed on navigation or a tap on the backdrop.
const navOpen = ref(false)
watch(() => route.fullPath, () => { navOpen.value = false })

const activeKey = computed(() => {
  const path = route.path
  const hit = NAV.find((n) => n.key !== 'dashboard' && path.startsWith(n.route))
  return hit ? hit.key : 'dashboard'
})

async function logout() {
  try {
    await logoutRequest()
  } catch (e) {
    // Even if the request fails, still send the user back to login.
  }
  clearAccess()
  router.push('/login')
}
</script>

<template>
  <div class="app">
    <div v-if="navOpen" class="sb-backdrop" @click="navOpen = false"></div>
    <div class="sidebar" :class="{ open: navOpen }">
      <div class="sb-brand">
        <img class="mark" :src="logoDark" :alt="t('app.name')" />
        <small>Admin</small>
      </div>
      <router-link
        v-for="item in navItems"
        :key="item.key"
        :to="item.route"
        class="sb-item"
        :class="{ active: activeKey === item.key }"
      >
        <Svgs :name="item.icon" />
        <span>{{ t('nav.' + item.key) }}</span>
      </router-link>
      <div class="sb-foot">
        <button class="sb-item" @click="logout">
          <Svgs name="logout" />
          <span>{{ t('nav.logout') }}</span>
        </button>
      </div>
    </div>
    <div class="main">
      <div class="topbar">
        <button class="nav-toggle" type="button" :aria-label="t('layout.menu')" :aria-expanded="navOpen" @click="navOpen = !navOpen">
          <svg viewBox="0 0 24 24" width="22" height="22" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><path d="M3 6h18M3 12h18M3 18h18" /></svg>
        </button>
        <div class="top-right">
          <div class="lang-switch" role="group" :aria-label="t('layout.language')">
            <button
              v-for="l in ['sw', 'en']"
              :key="l"
              :class="{ active: locale === l }"
              @click="setLocale(l)"
            >{{ l.toUpperCase() }}</button>
          </div>
          <router-link to="/profile" class="admin-chip" v-if="me">
            <div class="av" :style="{ background: avaColor(meName) }">{{ initials(meName) }}</div>
            <div>
              <div class="nm">{{ meName }}</div>
              <div class="rl">{{ roleText(role) }}</div>
            </div>
          </router-link>
        </div>
      </div>
      <div class="content">
        <router-view />
      </div>
    </div>
  </div>
</template>
