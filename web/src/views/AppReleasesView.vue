<script setup>
// Mobile app releases (server/app_releases.go): upload the Android APK here;
// the newest one is offered for download on the landing page.
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import Svgs from '../components/Svgs.vue'
import { listAppReleases, uploadAppRelease, deleteAppRelease } from '@/api/admin'
import { apiUrl } from '@/api/http'

const { t, locale } = useI18n()

const releases = ref([])
const loading = ref(true)
const error = ref('')
const notice = ref('')

const file = ref(null)
const version = ref('')
const notes = ref('')
const uploading = ref(false)
const progress = ref(0)
const fileInput = ref(null)
const dragging = ref(false)

const latest = computed(() => releases.value[0] || null)
const homeUrl = computed(() => window.location.origin + '/#app')
const downloadUrl = computed(() => new URL(apiUrl('/api/public/app/download'), window.location.origin).href)

function fmtSize(n) {
  if (!n) return '—'
  if (n >= 1 << 20) return (n / (1 << 20)).toFixed(1) + ' MB'
  return Math.round(n / 1024) + ' KB'
}
function fmtDate(d) {
  if (!d) return '—'
  return new Date(d).toLocaleString(locale.value === 'sw' ? 'sw-TZ' : 'en-GB', {
    timeZone: 'Africa/Dar_es_Salaam', day: 'numeric', month: 'short', year: 'numeric', hour: '2-digit', minute: '2-digit'
  })
}

async function load() {
  loading.value = true
  error.value = ''
  try {
    const res = await listAppReleases()
    releases.value = res.releases || []
  } catch (e) {
    error.value = e.message
  } finally {
    loading.value = false
  }
}

function pick(f) {
  error.value = ''
  notice.value = ''
  if (!f) return
  if (!/\.apk$/i.test(f.name)) {
    error.value = t('apprel.errApk')
    file.value = null
    return
  }
  file.value = f
  // Suggest a version from names like "helabox-1.2.0.apk" / "app-release-1.2.0.apk".
  if (!version.value) {
    const m = f.name.match(/(\d+\.\d+(?:\.\d+)?(?:[+-][0-9A-Za-z.]+)?)/)
    if (m) version.value = m[1]
  }
}
function onDrop(e) {
  dragging.value = false
  pick(e.dataTransfer?.files?.[0])
}

async function publish() {
  error.value = ''
  notice.value = ''
  if (!file.value) return (error.value = t('apprel.errApk'))
  if (!version.value.trim()) return (error.value = t('apprel.errVersion'))
  uploading.value = true
  progress.value = 0
  try {
    const res = await uploadAppRelease(
      { file: file.value, version: version.value.trim(), notes: notes.value.trim() },
      (p) => (progress.value = p)
    )
    notice.value = t('apprel.published', { v: res?.release?.version || version.value.trim() })
    file.value = null
    version.value = ''
    notes.value = ''
    if (fileInput.value) fileInput.value.value = ''
    await load()
  } catch (e) {
    const map = { too_large: 'apprel.errTooLarge', session: 'apprel.errSession', network: 'apprel.errNetwork' }
    error.value = map[e.message] ? t(map[e.message]) : e.message
  } finally {
    uploading.value = false
  }
}

async function remove(r) {
  if (!window.confirm(t('apprel.confirmRemove', { v: r.version }))) return
  error.value = ''
  notice.value = ''
  try {
    await deleteAppRelease(r.id)
    await load()
  } catch (e) {
    error.value = e.message
  }
}

const copied = ref(false)
async function copyLink() {
  try {
    await navigator.clipboard.writeText(downloadUrl.value)
    copied.value = true
    setTimeout(() => (copied.value = false), 2000)
  } catch {
    window.prompt(t('apprel.copyLink'), downloadUrl.value)
  }
}

onMounted(load)
</script>

<template>
  <div>
    <div class="page-head">
      <div>
        <h1>{{ t('apprel.title') }}</h1>
        <p>{{ t('apprel.subtitle') }}</p>
      </div>
      <div class="page-actions">
        <a class="btn btn-outline" :href="homeUrl" target="_blank" rel="noopener">{{ t('apprel.openHome') }}</a>
      </div>
    </div>

    <div v-if="error" class="ar-msg err">{{ error }}</div>
    <div v-if="notice" class="ar-msg ok">{{ notice }}</div>

    <div class="grid2">
      <!-- upload -->
      <div class="card">
        <div class="card-head"><h3>{{ t('apprel.upload') }}</h3></div>

        <label
          class="drop"
          :class="{ over: dragging, has: file }"
          @dragover.prevent="dragging = true"
          @dragleave.prevent="dragging = false"
          @drop.prevent="onDrop"
        >
          <input ref="fileInput" type="file" accept=".apk,application/vnd.android.package-archive" :disabled="uploading" @change="pick($event.target.files[0])" />
          <span class="drop-ic"><Svgs name="phone" /></span>
          <span v-if="file" class="drop-main">{{ file.name }}</span>
          <span v-else class="drop-main">{{ t('apprel.choose') }}</span>
          <span class="drop-sub">{{ file ? fmtSize(file.size) + ' · ' + t('apprel.change') : t('apprel.dropHint') }}</span>
        </label>

        <div class="field">
          <label for="ar-version">{{ t('apprel.version') }}</label>
          <input id="ar-version" v-model="version" type="text" maxlength="40" :placeholder="t('apprel.versionPh')" :disabled="uploading" />
        </div>
        <div class="field">
          <label for="ar-notes">{{ t('apprel.notes') }}</label>
          <textarea id="ar-notes" v-model="notes" rows="3" maxlength="2000" :placeholder="t('apprel.notesPh')" :disabled="uploading"></textarea>
        </div>

        <div v-if="uploading" class="ar-progress" role="progressbar" :aria-valuenow="Math.round(progress * 100)" aria-valuemin="0" aria-valuemax="100">
          <div class="progress"><div :style="{ width: Math.round(progress * 100) + '%' }"></div></div>
          <small>{{ t('apprel.uploading', { p: Math.round(progress * 100) }) }}</small>
        </div>

        <button class="btn btn-primary btn-block" :disabled="uploading || !file" @click="publish">
          <Svgs name="upload" /> {{ t('apprel.publish') }}
        </button>
      </div>

      <!-- current -->
      <div class="card ar-current">
        <div class="card-head"><h3>{{ t('apprel.current') }}</h3></div>
        <template v-if="latest">
          <div class="ar-live">
            <span class="ar-phone"><Svgs name="phone" /></span>
            <div>
              <div class="ar-ver">v{{ latest.version }} <span class="badge green">{{ t('apprel.live') }}</span></div>
              <div class="cell-muted">{{ fmtDate(latest.uploadedAt) }} · {{ fmtSize(latest.size) }}</div>
            </div>
          </div>
          <p v-if="latest.notes" class="ar-notes">{{ latest.notes }}</p>
          <div class="kv"><span class="k">{{ t('apprel.colDownloads') }}</span><span class="v">{{ latest.downloads || 0 }}</span></div>
          <div class="kv"><span class="k">SHA-256</span><span class="v ar-hash" :title="latest.sha256">{{ latest.sha256.slice(0, 16) }}…</span></div>
          <div class="ar-actions">
            <a class="btn btn-primary btn-sm" :href="downloadUrl"><Svgs name="download" /> {{ t('apprel.download') }}</a>
            <button class="btn btn-outline btn-sm" @click="copyLink">{{ copied ? t('apprel.copied') : t('apprel.copyLink') }}</button>
          </div>
        </template>
        <div v-else-if="!loading" class="empty">
          <div class="ic"><Svgs name="phone" /></div>
          <p>{{ t('apprel.none') }}</p>
        </div>
      </div>
    </div>

    <!-- history -->
    <div v-if="releases.length" class="panel">
      <div class="panel-inner card-head" style="margin-bottom: 0"><h3>{{ t('apprel.history') }}</h3></div>
      <div style="overflow-x: auto">
        <table class="dtable">
          <thead>
            <tr>
              <th>{{ t('apprel.colVersion') }}</th>
              <th>{{ t('apprel.colDate') }}</th>
              <th>{{ t('apprel.colSize') }}</th>
              <th>{{ t('apprel.colBy') }}</th>
              <th>{{ t('apprel.colDownloads') }}</th>
              <th></th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="(r, i) in releases" :key="r.id">
              <td>
                <div class="cell-main">v{{ r.version }} <span v-if="i === 0" class="badge green" style="margin-left: 6px">{{ t('apprel.live') }}</span></div>
                <div v-if="r.notes" class="cell-sub ar-cellnote">{{ r.notes }}</div>
              </td>
              <td>{{ fmtDate(r.uploadedAt) }}</td>
              <td>{{ fmtSize(r.size) }}</td>
              <td>{{ r.uploadedBy || '—' }}</td>
              <td>{{ r.downloads || 0 }}</td>
              <td style="text-align: right">
                <button class="btn btn-ghost btn-sm" @click="remove(r)">{{ t('apprel.remove') }}</button>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
      <p class="cell-muted" style="padding: 0 24px 18px">{{ t('apprel.removeNote') }}</p>
    </div>
  </div>
</template>

<style scoped>
.ar-msg { border-radius: 12px; padding: 12px 16px; font-size: 14px; font-weight: 600; margin-bottom: 16px; }
.ar-msg.err { background: var(--danger-100); color: var(--danger); }
.ar-msg.ok { background: var(--green-100); color: var(--teal-900); }

.drop {
  display: flex; flex-direction: column; align-items: center; justify-content: center; gap: 6px;
  border: 2px dashed var(--line); border-radius: 16px; background: var(--cream);
  padding: 26px 16px; margin-bottom: 18px; text-align: center; cursor: pointer;
  transition: border-color .25s ease, background .25s ease;
}
.drop:hover, .drop.over { border-color: var(--green-600); background: var(--green-100); }
.drop.has { border-style: solid; border-color: var(--green-600); background: #fff; }
.drop input { display: none; }
.drop-ic {
  width: 52px; height: 52px; border-radius: 14px; background: #fff; color: var(--green-600);
  display: inline-flex; align-items: center; justify-content: center; border: 1px solid var(--line); margin-bottom: 4px;
}
.drop-main { font-weight: 700; color: var(--ink-900); word-break: break-all; }
.drop-sub { font-size: 13px; color: var(--ink-400); }

.field label { display: block; font-size: 13px; font-weight: 700; color: var(--ink-700); margin-bottom: 6px; }
.field textarea { resize: vertical; }

.ar-progress { margin-bottom: 16px; }
.ar-progress small { display: block; margin-top: 6px; color: var(--ink-600); font-weight: 600; }

.ar-live { display: flex; align-items: center; gap: 14px; margin-bottom: 14px; }
.ar-phone {
  width: 52px; height: 52px; border-radius: 14px; flex-shrink: 0; color: #fff;
  background: var(--teal-900); display: inline-flex; align-items: center; justify-content: center;
}
.ar-ver { font-size: 22px; font-weight: 800; color: var(--ink-900); display: flex; align-items: center; gap: 8px; }
.ar-notes { background: var(--cream); border-radius: 12px; padding: 12px 14px; font-size: 14px; color: var(--ink-700); white-space: pre-line; margin-bottom: 10px; }
.ar-hash { font-family: ui-monospace, Menlo, Consolas, monospace; font-size: 12.5px; }
.ar-actions { display: flex; flex-wrap: wrap; gap: 10px; margin-top: 18px; }
.ar-cellnote { max-width: 340px; white-space: normal; }
</style>
