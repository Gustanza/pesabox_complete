// Language support. English is the default (Swahili one click away); the choice is remembered in
// localStorage under 'pb_lang' (ReportsView and the API layer read the same
// key so exports and SMS follow the language the admin sees).
import { createI18n } from 'vue-i18n'
import sw from './sw'
import en from './en'

export const LANG_KEY = 'pb_lang'
export const SUPPORTED = ['sw', 'en']

export function savedLocale() {
  try {
    const v = localStorage.getItem(LANG_KEY)
    if (SUPPORTED.includes(v)) return v
  } catch {
    // storage blocked — fall through to the default
  }
  return 'en'
}

const i18n = createI18n({
  legacy: false,
  locale: savedLocale(),
  fallbackLocale: 'en', // a missing key shows English rather than a raw key
  messages: { sw, en }
})

export function setLocale(l) {
  if (!SUPPORTED.includes(l)) return
  i18n.global.locale.value = l
  document.documentElement.lang = l
  try {
    localStorage.setItem(LANG_KEY, l)
  } catch {
    // not persisted; still applies for this session
  }
}

// BCP-47 tag for Intl formatting (dates, numbers).
export function intlLocale() {
  return i18n.global.locale.value === 'sw' ? 'sw-TZ' : 'en-GB'
}

document.documentElement.lang = i18n.global.locale.value

export default i18n
