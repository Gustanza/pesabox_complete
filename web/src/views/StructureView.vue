<script setup>
import { computed, onMounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import Svgs from '../components/Svgs.vue'
import { can } from '@/api/access'
import { listPartners, createPartner, updatePartner, listClusters, createCluster, updateCluster } from '@/api/admin'

// Organisation structure (TODO.md §4 / D7): Partner → Cluster → Group. A group
// sits in exactly one cluster and a cluster in exactly one partner, so the
// roll-up reports never count a group twice. Nothing here is ever deleted —
// partners and clusters are deactivated instead.

const { t } = useI18n()
const router = useRouter()
const canManage = computed(() => can('platform.manage'))

const tab = ref('partners')
const partners = ref([])
const clusters = ref([])
const partnerFilter = ref('')
const loading = ref(true)
const error = ref('')
const saving = ref(false)

const PARTNER_TYPES = ['NGO', 'Government', 'Bank', 'Other']
const emptyPartner = () => ({ id: '', name: '', type: 'NGO', contactPerson: '', phone: '', email: '' })
const emptyCluster = () => ({ id: '', name: '', partnerId: '', region: '', district: '', ward: '' })
const partnerForm = reactive(emptyPartner())
const clusterForm = reactive(emptyCluster())
const showPartnerForm = ref(false)
const showClusterForm = ref(false)

async function load() {
  loading.value = true
  error.value = ''
  try {
    ;[partners.value, clusters.value] = await Promise.all([listPartners(), listClusters()])
  } catch (e) {
    error.value = e.message || t('struct.loadFailed')
  } finally {
    loading.value = false
  }
}
onMounted(load)

const shownClusters = computed(() =>
  clusters.value.filter((c) => !partnerFilter.value || c.partnerId === partnerFilter.value)
)
const activePartners = computed(() => partners.value.filter((p) => p.status !== 'inactive'))

function editPartner(p) {
  Object.assign(partnerForm, emptyPartner(), p)
  showPartnerForm.value = true
}
function newPartner() {
  Object.assign(partnerForm, emptyPartner())
  showPartnerForm.value = true
}
function editCluster(c) {
  Object.assign(clusterForm, emptyCluster(), c)
  showClusterForm.value = true
}
function newCluster() {
  Object.assign(clusterForm, emptyCluster(), { partnerId: partnerFilter.value || activePartners.value[0]?.id || '' })
  showClusterForm.value = true
}

async function run(fn) {
  saving.value = true
  error.value = ''
  try {
    await fn()
    await load()
    return true
  } catch (e) {
    error.value = e.message || t('common.requestFailed')
    return false
  } finally {
    saving.value = false
  }
}

async function savePartner() {
  if (!partnerForm.name.trim()) {
    error.value = t('struct.nameRequired')
    return
  }
  const { id, name, type, contactPerson, phone, email } = partnerForm
  const ok = await run(() => (id ? updatePartner(id, { name, type, contactPerson, phone, email }) : createPartner({ name, type, contactPerson, phone, email })))
  if (ok) showPartnerForm.value = false
}

async function saveCluster() {
  if (!clusterForm.name.trim()) {
    error.value = t('struct.nameRequired')
    return
  }
  if (!clusterForm.partnerId) {
    error.value = t('struct.partnerRequired')
    return
  }
  const { id, name, partnerId, region, district, ward } = clusterForm
  const ok = await run(() => (id ? updateCluster(id, { name, partnerId, region, district, ward }) : createCluster({ name, partnerId, region, district, ward })))
  if (ok) showClusterForm.value = false
}

function toggle(kind, rec) {
  const status = rec.status === 'inactive' ? 'active' : 'inactive'
  const msg = status === 'inactive' ? t('struct.confirmDeactivate', { name: rec.name }) : t('struct.confirmReactivate', { name: rec.name })
  if (!confirm(msg)) return
  run(() => (kind === 'partner' ? updatePartner(rec.id, { status }) : updateCluster(rec.id, { status })))
}

function openClusters(p) {
  partnerFilter.value = p.id
  tab.value = 'clusters'
}
</script>

<template>
  <div>
    <div class="page-head">
      <div>
        <h1>{{ t('struct.title') }}</h1>
        <p>{{ t('struct.subtitle') }}</p>
      </div>
      <div v-if="canManage" class="page-actions">
        <button v-if="tab === 'partners'" class="btn btn-primary" @click="newPartner"><Svgs name="plus" /> {{ t('struct.addPartner') }}</button>
        <button v-else class="btn btn-primary" :disabled="!activePartners.length" @click="newCluster"><Svgs name="plus" /> {{ t('struct.addCluster') }}</button>
      </div>
    </div>

    <div class="dtabs">
      <button class="dtab" :class="{ active: tab === 'partners' }" @click="tab = 'partners'">{{ t('struct.partners') }} ({{ partners.length }})</button>
      <button class="dtab" :class="{ active: tab === 'clusters' }" @click="tab = 'clusters'">{{ t('struct.clusters') }} ({{ clusters.length }})</button>
    </div>

    <div v-if="error" style="color: var(--danger); font-size: 13px; margin-bottom: 12px">{{ error }}</div>

    <!-- partner form -->
    <div v-if="showPartnerForm && tab === 'partners'" class="card" style="max-width: 720px">
      <div class="card-head"><h3>{{ partnerForm.id ? t('struct.editPartner') : t('struct.addPartner') }}</h3></div>
      <div class="field-row">
        <div class="field">
          <label>{{ t('struct.name') }}</label>
          <div class="inp filled"><input v-model="partnerForm.name" :placeholder="t('struct.partnerNamePh')" /></div>
        </div>
        <div class="field">
          <label>{{ t('struct.type') }}</label>
          <div class="inp filled">
            <select v-model="partnerForm.type">
              <option v-for="ty in PARTNER_TYPES" :key="ty" :value="ty">{{ t('struct.types.' + ty) }}</option>
            </select>
          </div>
        </div>
      </div>
      <div class="field-row">
        <div class="field">
          <label>{{ t('struct.contactPerson') }}</label>
          <div class="inp filled"><input v-model="partnerForm.contactPerson" /></div>
        </div>
        <div class="field">
          <label>{{ t('struct.phone') }}</label>
          <div class="inp filled"><input v-model="partnerForm.phone" type="tel" /></div>
        </div>
      </div>
      <div class="field">
        <label>{{ t('struct.email') }}</label>
        <div class="inp filled"><input v-model="partnerForm.email" type="email" /></div>
      </div>
      <div class="cg-actions">
        <button class="btn btn-primary" :disabled="saving" @click="savePartner">{{ saving ? t('struct.saving') : t('common.save') }}</button>
        <button class="btn btn-ghost" @click="showPartnerForm = false">{{ t('common.cancel') }}</button>
      </div>
    </div>

    <!-- cluster form -->
    <div v-if="showClusterForm && tab === 'clusters'" class="card" style="max-width: 720px">
      <div class="card-head"><h3>{{ clusterForm.id ? t('struct.editCluster') : t('struct.addCluster') }}</h3></div>
      <div class="field-row">
        <div class="field">
          <label>{{ t('struct.name') }}</label>
          <div class="inp filled"><input v-model="clusterForm.name" :placeholder="t('struct.clusterNamePh')" /></div>
        </div>
        <div class="field">
          <label>{{ t('struct.partner') }}</label>
          <div class="inp filled">
            <select v-model="clusterForm.partnerId">
              <option v-for="p in activePartners" :key="p.id" :value="p.id">{{ p.name }}</option>
            </select>
          </div>
        </div>
      </div>
      <div class="field-row">
        <div class="field">
          <label>{{ t('cg.region') }}</label>
          <div class="inp filled"><input v-model="clusterForm.region" /></div>
        </div>
        <div class="field">
          <label>{{ t('cg.district') }}</label>
          <div class="inp filled"><input v-model="clusterForm.district" /></div>
        </div>
      </div>
      <div class="field">
        <label>{{ t('cg.ward') }} <span class="opt">{{ t('cg.optional') }}</span></label>
        <div class="inp filled"><input v-model="clusterForm.ward" /></div>
      </div>
      <div class="cg-actions">
        <button class="btn btn-primary" :disabled="saving" @click="saveCluster">{{ saving ? t('struct.saving') : t('common.save') }}</button>
        <button class="btn btn-ghost" @click="showClusterForm = false">{{ t('common.cancel') }}</button>
      </div>
    </div>

    <div class="panel">
      <div v-if="loading" class="empty"><p>{{ t('common.loading') }}</p></div>

      <template v-else-if="tab === 'partners'">
        <div v-if="!partners.length" class="empty"><div class="ic">&#127970;</div><p>{{ t('struct.noPartners') }}</p></div>
        <div v-else style="overflow-x: auto">
          <table class="dtable">
            <thead>
              <tr>
                <th>{{ t('struct.name') }}</th>
                <th>{{ t('struct.type') }}</th>
                <th>{{ t('struct.contactPerson') }}</th>
                <th>{{ t('struct.clusters') }}</th>
                <th>{{ t('struct.groups') }}</th>
                <th>{{ t('common.status') }}</th>
                <th v-if="canManage"></th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="p in partners" :key="p.id">
                <td class="cell-strong clickable" @click="openClusters(p)">{{ p.name }}</td>
                <td class="cell-muted">{{ t('struct.types.' + (p.type || 'Other')) }}</td>
                <td class="cell-muted">{{ [p.contactPerson, p.phone].filter(Boolean).join(' · ') || '—' }}</td>
                <td>{{ p.clusterCount }}</td>
                <td>{{ p.groupCount }}</td>
                <td><span class="badge" :class="p.status === 'inactive' ? 'grey' : 'green'">{{ p.status === 'inactive' ? t('users.inactive') : t('users.active') }}</span></td>
                <td v-if="canManage" style="white-space: nowrap">
                  <button class="btn btn-ghost btn-sm" @click="editPartner(p)">{{ t('gd.edit') }}</button>
                  <button class="btn btn-ghost btn-sm" @click="toggle('partner', p)">{{ p.status === 'inactive' ? t('struct.reactivate') : t('struct.deactivate') }}</button>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </template>

      <template v-else>
        <div class="toolbar">
          <select v-model="partnerFilter" class="filter-select">
            <option value="">{{ t('struct.allPartners') }}</option>
            <option v-for="p in partners" :key="p.id" :value="p.id">{{ p.name }}</option>
          </select>
        </div>
        <div v-if="!shownClusters.length" class="empty"><div class="ic">&#128205;</div><p>{{ t('struct.noClusters') }}</p></div>
        <div v-else style="overflow-x: auto">
          <table class="dtable">
            <thead>
              <tr>
                <th>{{ t('struct.name') }}</th>
                <th>{{ t('struct.partner') }}</th>
                <th>{{ t('gd.location') }}</th>
                <th>{{ t('struct.groups') }}</th>
                <th>{{ t('common.status') }}</th>
                <th v-if="canManage"></th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="c in shownClusters" :key="c.id">
                <td class="cell-strong clickable" @click="router.push({ path: '/groups', query: { clusterId: c.id } })">{{ c.name }}</td>
                <td class="cell-muted">{{ c.partnerName || '—' }}</td>
                <td class="cell-muted">{{ [c.ward, c.district, c.region].filter(Boolean).join(', ') || '—' }}</td>
                <td>{{ c.groupCount }}</td>
                <td><span class="badge" :class="c.status === 'inactive' ? 'grey' : 'green'">{{ c.status === 'inactive' ? t('users.inactive') : t('users.active') }}</span></td>
                <td v-if="canManage" style="white-space: nowrap">
                  <button class="btn btn-ghost btn-sm" @click="editCluster(c)">{{ t('gd.edit') }}</button>
                  <button class="btn btn-ghost btn-sm" @click="toggle('cluster', c)">{{ c.status === 'inactive' ? t('struct.reactivate') : t('struct.deactivate') }}</button>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </template>
      <div style="height: 16px"></div>
    </div>
  </div>
</template>
