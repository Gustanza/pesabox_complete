// Fails when en.js and sw.js do not have exactly the same keys, or when a
// view uses a t('literal.key') that is missing from en.js.
//   node scripts/check-i18n.mjs
import { readFileSync, readdirSync } from 'node:fs'
import { pathToFileURL } from 'node:url'
import { resolve, join } from 'node:path'

const root = resolve(import.meta.dirname, '..', 'src')
const en = (await import(pathToFileURL(join(root, 'i18n/en.js')))).default
const sw = (await import(pathToFileURL(join(root, 'i18n/sw.js')))).default

const flat = (o, p = '') => Object.entries(o).flatMap(([k, v]) => (typeof v === 'object' ? flat(v, p + k + '.') : [p + k]))
const a = new Set(flat(en))
const b = new Set(flat(sw))
let bad = 0
for (const k of a) if (!b.has(k)) (console.error('missing in sw.js:', k), bad++)
for (const k of b) if (!a.has(k)) (console.error('missing in en.js:', k), bad++)

const files = []
const walk = (d) => readdirSync(d, { withFileTypes: true }).forEach((e) => (e.isDirectory() ? walk(join(d, e.name)) : /\.(vue|js)$/.test(e.name) && files.push(join(d, e.name))))
walk(root)
for (const f of files) {
  for (const m of readFileSync(f, 'utf8').matchAll(/\bt\('([a-zA-Z0-9_.]+)'\s*[,)]/g)) {
    if (!a.has(m[1])) (console.error(`${f.replace(root, 'src')}: unknown key ${m[1]}`), bad++)
  }
}
console.log(bad ? `${bad} problem(s)` : `i18n ok — ${a.size} keys in both languages`)
process.exit(bad ? 1 : 0)
