<script setup>
const general = [
  ['Platform Name', 'PesaBox'],
  ['Currency', 'TZS'],
  ['Timezone', 'Africa/Dar_es_Salaam']
]
const sms = [
  ['Provider', 'NextSMS'],
  ['Sender ID', 'PESABOX'],
  ['OTP', 'Enabled'],
  ['Transactional SMS', 'Enabled']
]
const admins = [
  { name: 'Raymond', email: 'admin@pesabox.co.tz', role: 'Super Admin', status: 'Active' },
  { name: 'Support Admin', email: 'support@pesabox.co.tz', role: 'Support', status: 'Active' }
]
const health = [
  ['API', '&#9679; Operational'],
  ['Database', '&#9679; Operational'],
  ['Authentication', '&#9679; Operational'],
  ['SMS Provider', '&#9679; Operational'],
  ['Background Jobs', '&#9679; Operational'],
  ['Failed SMS (24h)', '170']
]
</script>

<template>
  <div>
    <div class="page-head">
      <div>
        <h1>Settings</h1>
        <p>Platform-wide configuration and administrators.</p>
      </div>
    </div>

    <div class="grid2">
      <div class="card">
        <div class="card-head"><h3>General</h3></div>
        <div v-for="[k, v] in general" :key="k" class="field-view">
          <label>{{ k }}</label>
          <div class="box">{{ v }}</div>
        </div>
      </div>
      <div class="card">
        <div class="card-head"><h3>SMS</h3></div>
        <div v-for="[k, v] in sms" :key="k" class="field-view">
          <label>{{ k }}</label>
          <div class="box">{{ v }}</div>
        </div>
      </div>
    </div>

    <div class="card">
      <div class="card-head">
        <h3>System Administrators</h3>
        <a>+ Add Admin</a>
      </div>
      <table class="dtable">
        <thead>
          <tr><th>Name</th><th>Email</th><th>Role</th><th>Status</th></tr>
        </thead>
        <tbody>
          <tr v-for="a in admins" :key="a.email">
            <td class="cell-main">{{ a.name }}</td>
            <td>{{ a.email }}</td>
            <td>{{ a.role }}</td>
            <td><span class="badge green">{{ a.status }}</span></td>
          </tr>
        </tbody>
      </table>
    </div>

    <div class="card">
      <div class="card-head"><h3>System Health</h3></div>
      <div class="grid3">
        <div v-for="[k, v] in health" :key="k" class="field-view">
          <label>{{ k }}</label>
          <div class="box" v-html="v"></div>
        </div>
      </div>
    </div>
  </div>
</template>