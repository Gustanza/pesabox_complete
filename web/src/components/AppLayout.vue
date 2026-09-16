<script setup>
import { computed, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import Svgs from './Svgs.vue'
import { NAV_ITEMS, initials, avaColor } from '../data/mock.js'
import { logout as logoutRequest, currentUser } from '@/api/auth'
import { roleLabel } from '@/api/users'

const route = useRoute()
const router = useRouter()

const me = ref(null)
const meName = computed(() => {
  if (!me.value) return ''
  return [me.value.firstName, me.value.lastName].filter(Boolean).join(' ') || me.value.username
})

onMounted(async () => {
  me.value = await currentUser()
})

const activeKey = computed(() => {
  const path = route.path
  if (path.startsWith('/groups')) return 'groups'
  if (path.startsWith('/users')) return 'users'
  if (path.startsWith('/finance')) return 'finance'
  if (path.startsWith('/sms')) return 'sms'
  if (path.startsWith('/reports')) return 'reports'
  if (path.startsWith('/audit')) return 'audit'
  if (path.startsWith('/settings')) return 'settings'
  return 'dashboard'
})

async function logout() {
  try {
    await logoutRequest()
  } catch (e) {
    // Even if the request fails, still send the user back to login.
  }
  router.push('/login')
}
</script>

<template>
  <div class="app">
    <div class="sidebar">
      <div class="sb-brand">
        <div class="mark">P</div>
        <div>
          <span>PesaBox <span class="accent">Admin</span></span>
          <small>SUPER ADMIN</small>
        </div>
      </div>
      <router-link
        v-for="item in NAV_ITEMS"
        :key="item.key"
        :to="item.route"
        class="sb-item"
        :class="{ active: activeKey === item.key }"
      >
        <Svgs :name="item.icon" />
        <span>{{ item.label }}</span>
      </router-link>
      <div class="sb-foot">
        <router-link to="/settings" class="sb-item">
          <Svgs name="settings" />
          <span>Help</span>
        </router-link>
        <button class="sb-item" @click="logout">
          <Svgs name="logout" />
          <span>Logout</span>
        </button>
      </div>
    </div>
    <div class="main">
      <div class="topbar">
        <div class="top-right">
          <div class="icon-circle">
            <Svgs name="bell" />
            <span class="dot"></span>
          </div>
          <div class="admin-chip" v-if="me">
            <div class="av" :style="{ background: avaColor(meName) }">{{ initials(meName) }}</div>
            <div>
              <div class="nm">{{ meName }}</div>
              <div class="rl">{{ roleLabel(me.role) }}</div>
            </div>
          </div>
        </div>
      </div>
      <div class="content">
        <router-view />
      </div>
    </div>
  </div>
</template>