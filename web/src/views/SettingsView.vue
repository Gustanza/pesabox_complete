<script setup>
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import Svgs from '../components/Svgs.vue'
import { getNotificationSettings, saveNotificationSettings } from '@/api/settings'
import { getDashboard, listUsers } from '@/api/admin'

const { t, te } = useI18n()

// Platform info (display only).
const general = computed(() => [
  ['set.platformName', t('app.name')],
  ['set.currency', 'TZS'],
  ['set.timezone', 'Africa/Dar_es_Salaam']
])
// Provider and sender come from the server (server/sms.go) — the sender ID is
// BEEM_SENDER_ID or the approved default, never hard-coded here.
const sms = computed(() => [
  ['set.provider', notif.value.smsProvider || '—'],
  ['set.senderId', notif.value.senderId || '—'],
  ['setx.delivery', notif.value.smsLive ? t('dash.live') : t('dash.devMode')]
])

// Live service status (same source as the dashboard).
const status = ref(null)
const HEALTH_KEYS = [
  ['api', 'dash2.api'],
  ['database', 'dash2.database'],
  ['authentication', 'dash2.authentication'],
  ['smsProvider', 'dash.smsProvider'],
  ['backgroundJobs', 'dash2.jobs']
]
const health = computed(() => HEALTH_KEYS.map(([key, label]) => [label, !!status.value?.[key]]))

// People who administer the platform: super admins and HelaBox staff.
const admins = ref([])

// SMS switches and language are stored on the server and enforced where
// messages are sent (server/sms.go). Optimistic update, rolled back on failure.
const notif = ref({ otpSms: true, transactionalSms: true, remindersSms: true, smsLanguage: 'sw' })
const error = ref('')

onMounted(async () => {
  try {
    notif.value = { ...notif.value, ...(await getNotificationSettings()) }
  } catch (e) {
    error.value = e.message || t('set.loadFailed')
  }
  getDashboard()
    .then((d) => (status.value = d.status))
    .catch(() => {})
  listUsers()
    .then((us) => (admins.value = us.filter((u) => u.role === 'super_admin' || u.role === 'staff')))
    .catch(() => {})
})

async function setNotif(key, value) {
  const before = notif.value[key]
  notif.value[key] = value
  error.value = ''
  try {
    notif.value = { ...notif.value, ...(await saveNotificationSettings({ [key]: value })) }
  } catch (e) {
    notif.value[key] = before
    error.value = e.message || t('settings.saveFailed')
  }
}

const roleText = (r) => (te('roles.' + r) ? t('roles.' + r) : r)
const adminName = (u) => [u.firstName, u.lastName].filter(Boolean).join(' ') || u.username
</script>

<template>
  <div>
    <div class="page-head">
      <div>
        <h1>{{ t('set.title') }}</h1>
        <p>{{ t('set.subtitle') }}</p>
      </div>
    </div>

    <div v-if="error" style="color: var(--danger); font-size: 13.5px; margin-bottom: 12px">
      {{ error }}
    </div>

    <div class="settings-grid">
      <div class="settings-card">
        <div class="head"><div class="ic"><Svgs name="tag" /></div><h3>{{ t('set.general') }}</h3></div>
        <div v-for="[k, v] in general" :key="k" class="field-view">
          <label>{{ t(k) }}</label>
          <div class="box">{{ v }}</div>
        </div>
      </div>

      <div class="settings-card">
        <div class="head"><div class="ic"><Svgs name="sms" /></div><h3>{{ t('set.sms') }}</h3></div>
        <div v-for="[k, v] in sms" :key="k" class="field-view">
          <label>{{ t(k) }}</label>
          <div class="box">{{ v }}</div>
        </div>
        <div class="toggle-row">
          <span class="toggle-label">{{ t('settings.otpTexts') }}</span>
          <div class="toggle" :class="notif.otpSms ? 'on' : 'off'" @click="setNotif('otpSms', !notif.otpSms)"><div class="knob"></div></div>
        </div>
        <div v-if="!notif.otpSms" style="font-size: 12px; color: var(--danger); margin: -4px 0 10px">
          {{ t('settings.otpWarning') }}
        </div>
        <div class="toggle-row">
          <span class="toggle-label">{{ t('settings.transactionalSms') }}</span>
          <div class="toggle" :class="notif.transactionalSms ? 'on' : 'off'" @click="setNotif('transactionalSms', !notif.transactionalSms)"><div class="knob"></div></div>
        </div>
        <div class="toggle-row">
          <span class="toggle-label">{{ t('settings.remindersSms') }}</span>
          <div class="toggle" :class="notif.remindersSms ? 'on' : 'off'" @click="setNotif('remindersSms', !notif.remindersSms)"><div class="knob"></div></div>
        </div>
        <div class="field-view">
          <label>{{ t('settings.smsLanguage') }}</label>
          <select class="box" style="width: 100%" :value="notif.smsLanguage" @change="setNotif('smsLanguage', $event.target.value)">
            <option value="sw">{{ t('set.langSw') }}</option>
            <option value="en">{{ t('set.langEn') }}</option>
          </select>
        </div>
      </div>

    </div>

    <div class="settings-grid" style="grid-template-columns: repeat(auto-fit, minmax(340px, 1fr));">
      <div class="panel">
        <div class="panel-inner" style="padding-bottom: 0">
          <h3 style="font-size: 17px">{{ t('set.admins') }}</h3>
        </div>
        <div style="overflow-x: auto; margin-top: 14px">
          <table class="dtable">
            <thead>
              <tr>
                <th>{{ t('set.name') }}</th>
                <th>{{ t('struct.phone') }}</th>
                <th>{{ t('set.role') }}</th>
                <th>{{ t('common.status') }}</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="a in admins" :key="a.id">
                <td class="cell-strong">{{ adminName(a) }}</td>
                <td class="cell-muted">{{ a.username }}</td>
                <td class="cell-muted">{{ roleText(a.role) }}<template v-if="a.preset"> · {{ t('ur.presets.' + a.preset) }}</template></td>
                <td><span class="badge" :class="a.isActive ? 'green' : 'grey'">{{ a.isActive ? t('users.active') : t('users.inactive') }}</span></td>
              </tr>
            </tbody>
          </table>
        </div>
        <div style="height: 16px"></div>
      </div>

      <div class="chart-card">
        <h3>{{ t('set.health') }}</h3>
        <div v-for="[name, ok] in health" :key="name" class="status-row">
          <span>{{ t(name) }}</span>
          <span class="status-dot" :class="ok ? '' : 'red'"></span>
        </div>
      </div>
    </div>

    <p style="text-align: center; color: var(--ink-400); font-size: 13px">{{ t('setx.immediate') }}</p>
  </div>
</template>
