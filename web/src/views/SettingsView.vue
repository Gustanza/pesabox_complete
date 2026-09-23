<script setup>
import { onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import Svgs from '../components/Svgs.vue'
import { getNotificationSettings, saveNotificationSettings } from '@/api/settings'

const { t, te } = useI18n()

// Static platform info (display only).
const general = [
  ['set.platformName', 'PesaBox'],
  ['set.currency', 'TZS'],
  ['set.timezone', 'Africa/Dar_es_Salaam']
]
const sms = [
  ['set.provider', 'NextSMS'],
  ['set.senderId', 'PESABOX']
]
const health = [
  ['dash2.api', 'Operational'],
  ['dash2.database', 'Operational'],
  ['dash2.authentication', 'Operational'],
  ['dash.smsProvider', 'Operational'],
  ['dash2.jobs', 'Operational']
]
const admins = [
  { name: 'Raymond', email: 'admin@pesabox.co.tz', role: 'super_admin', status: 'Active' },
  { name: 'Support Admin', email: 'support@pesabox.co.tz', role: 'support_admin', status: 'Active' }
]

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
})

async function setNotif(key, value) {
  const before = notif.value[key]
  notif.value[key] = value
  error.value = ''
  try {
    notif.value = await saveNotificationSettings({ [key]: value })
  } catch (e) {
    notif.value[key] = before
    error.value = e.message || t('settings.saveFailed')
  }
}

// These two have no backend mechanism (no real 2FA flow, no masking layer).
const twoFA = ref(true)
const maskSensitive = ref(true)

const roleText = (r) => (te('roles.' + r) ? t('roles.' + r) : r)

function save() {
  alert(t('setx.saved'))
}
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

      <div class="settings-card">
        <div class="head"><div class="ic"><Svgs name="shield" /></div><h3>{{ t('set.security') }}</h3></div>
        <div class="toggle-row">
          <span class="toggle-label">{{ t('set.twoFactor') }}</span>
          <div class="toggle" :class="twoFA ? 'on' : 'off'" @click="twoFA = !twoFA"><div class="knob"></div></div>
        </div>
        <div class="toggle-row">
          <span class="toggle-label">{{ t('set.maskSensitive') }}</span>
          <div class="toggle" :class="maskSensitive ? 'on' : 'off'" @click="maskSensitive = !maskSensitive"><div class="knob"></div></div>
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
                <th>{{ t('set.email') }}</th>
                <th>{{ t('set.role') }}</th>
                <th>{{ t('common.status') }}</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="a in admins" :key="a.email">
                <td class="cell-strong">{{ a.name }}</td>
                <td class="cell-muted">{{ a.email }}</td>
                <td class="cell-muted">{{ roleText(a.role) }}</td>
                <td><span class="badge green">{{ t('users.active') }}</span></td>
              </tr>
            </tbody>
          </table>
        </div>
        <div style="height: 16px"></div>
      </div>

      <div class="chart-card">
        <h3>{{ t('set.health') }}</h3>
        <div v-for="[name, state2] in health" :key="name" class="status-row">
          <span>{{ t(name) }}</span>
          <span class="status-dot" :class="state2 === 'Operational' ? '' : 'red'"></span>
        </div>
      </div>
    </div>

    <div style="text-align: center">
      <button class="btn btn-primary" style="padding: 15px 40px" @click="save">{{ t('setx.save') }}</button>
      <div style="display: flex; align-items: center; justify-content: center; gap: 8px; margin-top: 14px; color: var(--gold-500); font-size: 13.5px; font-weight: 600">
        <Svgs name="warn" width="16" height="16" /> {{ t('setx.immediate') }}
      </div>
    </div>
  </div>
</template>
