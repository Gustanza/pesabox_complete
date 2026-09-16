<script setup>
import { ref } from 'vue'
import Svgs from '../components/Svgs.vue'

const general = [
  ['Platform Name', 'PesaBox'],
  ['Currency', 'TZS'],
  ['Timezone', 'Africa/Dar_es_Salaam']
]
const sms = [
  ['Provider', 'NextSMS'],
  ['Sender ID', 'PESABOX']
]
const health = [
  ['API', 'Operational'],
  ['Database', 'Operational'],
  ['Authentication', 'Operational'],
  ['SMS Provider', 'Operational'],
  ['Background Jobs', 'Operational']
]
const admins = [
  { name: 'Raymond', email: 'admin@pesabox.co.tz', role: 'Super Admin', status: 'Active' },
  { name: 'Support Admin', email: 'support@pesabox.co.tz', role: 'Support', status: 'Active' }
]

const otp = ref(true)
const transactionalSms = ref(true)
const twoFA = ref(true)
const maskSensitive = ref(true)

function save() {
  alert('Settings saved')
}
</script>

<template>
  <div>
    <div class="page-head">
      <div>
        <h1>Settings</h1>
        <p>Platform-wide configuration and administrators.</p>
      </div>
    </div>

    <div class="settings-grid">
      <div class="settings-card">
        <div class="head"><div class="ic"><Svgs name="tag" /></div><h3>General</h3></div>
        <div v-for="[k, v] in general" :key="k" class="field-view">
          <label>{{ k }}</label>
          <div class="box">{{ v }}</div>
        </div>
      </div>

      <div class="settings-card">
        <div class="head"><div class="ic"><Svgs name="sms" /></div><h3>SMS</h3></div>
        <div v-for="[k, v] in sms" :key="k" class="field-view">
          <label>{{ k }}</label>
          <div class="box">{{ v }}</div>
        </div>
        <div class="toggle-row">
          <span class="toggle-label">OTP Texts</span>
          <div class="toggle" :class="otp ? 'on' : 'off'" @click="otp = !otp"><div class="knob"></div></div>
        </div>
        <div class="toggle-row">
          <span class="toggle-label">Transactional SMS</span>
          <div class="toggle" :class="transactionalSms ? 'on' : 'off'" @click="transactionalSms = !transactionalSms"><div class="knob"></div></div>
        </div>
      </div>

      <div class="settings-card">
        <div class="head"><div class="ic"><Svgs name="shield" /></div><h3>Security</h3></div>
        <div class="toggle-row">
          <span class="toggle-label">Two-factor authentication</span>
          <div class="toggle" :class="twoFA ? 'on' : 'off'" @click="twoFA = !twoFA"><div class="knob"></div></div>
        </div>
        <div class="toggle-row">
          <span class="toggle-label">Mask sensitive data</span>
          <div class="toggle" :class="maskSensitive ? 'on' : 'off'" @click="maskSensitive = !maskSensitive"><div class="knob"></div></div>
        </div>
      </div>
    </div>

    <div class="settings-grid" style="grid-template-columns: repeat(auto-fit, minmax(340px, 1fr));">
      <div class="panel">
        <div class="panel-inner" style="padding-bottom: 0">
          <h3 style="font-size: 17px">System Administrators</h3>
        </div>
        <div style="overflow-x: auto; margin-top: 14px">
          <table class="dtable">
            <thead>
              <tr><th>Name</th><th>Email</th><th>Role</th><th>Status</th></tr>
            </thead>
            <tbody>
              <tr v-for="a in admins" :key="a.email">
                <td class="cell-strong">{{ a.name }}</td>
                <td class="cell-muted">{{ a.email }}</td>
                <td class="cell-muted">{{ a.role }}</td>
                <td><span class="badge green">{{ a.status }}</span></td>
              </tr>
            </tbody>
          </table>
        </div>
        <div style="height: 16px"></div>
      </div>

      <div class="chart-card">
        <h3>System Health</h3>
        <div v-for="[name, state2] in health" :key="name" class="status-row">
          <span>{{ name }}</span>
          <span class="status-dot" :class="state2 === 'Operational' ? '' : 'red'"></span>
        </div>
      </div>
    </div>

    <div style="text-align: center">
      <button class="btn btn-primary" style="padding: 15px 40px" @click="save">Save Changes</button>
      <div style="display: flex; align-items: center; justify-content: center; gap: 8px; margin-top: 14px; color: var(--gold-500); font-size: 13.5px; font-weight: 600">
        <Svgs name="warn" width="16" height="16" /> Changes take effect immediately.
      </div>
    </div>
  </div>
</template>