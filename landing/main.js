// HelaBox landing page: language switch (Swahili default, English),
// header state, mobile drawer, scroll reveal.
(function () {
  'use strict'

  // ---- English copy. Swahili lives in the HTML (read at load). ----------
  var EN = {
    'skip': 'Skip to content',
    'nav.features': 'Features', 'nav.how': 'How It Works', 'nav.reports': 'Reports', 'nav.trust': 'Trust', 'nav.faq': 'FAQ',
    'cta.login': 'Log in', 'cta.dashboard': 'Open the Dashboard', 'cta.how': 'How It Works', 'cta.start': 'Get Started',
    'cta.faq': 'Frequently Asked Questions',
    'hero.eyebrow': 'Savings group (VSLA) platform',
    'hero.title': 'Group Accounts, <span class="accent">Clear and Secure</span>',
    'hero.lead': 'Record savings, loans and meetings on a phone — every member gets an SMS confirmation.',
    'hero.f1': 'An SMS for every transaction',
    'hero.f2': 'No transaction is ever deleted',
    'hero.f3': 'Swahili and English',
    'ph.total': 'Total group savings', 'ph.shares': 'Shares', 'ph.members': 'Members', 'ph.loans': 'Loans',
    'ph.today': "Today's Meeting", 'ph.t1': 'Shares × 3', 'ph.t2': 'Loan repayment', 'ph.t3': 'Social fund',
    'chip.att': 'Attendance 22 / 24',
    'chip.sms': 'Hello Asha, your purchase of 3 shares worth TSh 15,000 is confirmed. Thank you.',
    'chip.rep': 'Savings per month',
    'chip.rev': 'Transaction reversed · balance restored',
    'chip.audit': 'Recorded in the audit log',
    'feat.eyebrow': 'Everything a group needs',
    'feat.title': 'The Whole Meeting, <span class="light">On One Phone</span>',
    'feat.lead': 'The Secretary or Treasurer records everything during the meeting. The maths does itself, and the paper ledger becomes history.',
    'feat.c1.t': 'Savings, Shares & Social Fund',
    'feat.c1.p': "Record mandatory savings, share purchases and social fund contributions for every member — their balance updates instantly.",
    'feat.c2.t': 'Loans With Clear Interest',
    'feat.c2.p': "Interest is fixed the day a loan is issued, following the group's constitution. Every repayment reduces the balance, and late loans show up early.",
    'feat.c3.t': 'Meetings & Attendance',
    'feat.c3.p': 'Open a meeting, mark attendance (present, late, absent, excused) and close it. A closed meeting is locked — nothing changes afterwards.',
    'dash.k1': 'Active groups', 'dash.k2': 'Members', 'dash.k3': 'Savings (TSh)', 'dash.chart': 'Savings vs loans',
    'why.eyebrow': 'Why HelaBox',
    'why.title': 'Group Money Deserves Full <span class="mark">Transparency</span>',
    'why.lead': 'We built HelaBox on one simple rule: money is never deleted or quietly edited.',
    'why.l1': 'No transaction is ever deleted<span>If a mistake happens, the transaction is reversed and the group, loan or fine balance is put back — the history stays.</span>',
    'why.l2': 'A record of every step<span>Every change is logged: who, when and what — in an audit log that cannot be erased.</span>',
    'why.l3': 'Everyone sees only their part<span>Roles and assigned areas decide who sees which groups and what they can do.</span>',
    'why.l4': 'Invite-only sign-in<span>A one-time code (OTP) is sent by SMS to registered users only.</span>',
    'svc.eyebrow': 'Beyond savings',
    'svc.title': 'Every Service a Group Needs',
    'svc.c1.t': 'Group Constitution',
    'svc.c1.p': 'Share value, share limits, interest, loan period and maximum, fine reasons — every group with its own rules.',
    'svc.c2.t': 'Fines',
    'svc.c2.p': "Charge fines for agreed reasons and track what each member has paid and still owes.",
    'svc.c3.t': 'Government Loans',
    'svc.c3.p': "Loans to the group (such as council loans) and their repayments are tracked separately, never mixed with members' savings.",
    'svc.c4.t': 'SMS Confirmations',
    'svc.c4.p': 'Every saving, loan, repayment and fine — the member gets an SMS. Plus reminders for meetings and loans falling due.',
    'org.eyebrow': 'For every level',
    'org.title': 'From One Group to a Whole Programme',
    'org.lead': 'Partners oversee clusters, clusters gather groups, and groups have members. Everyone signs in and sees only their part.',
    'org.r1': 'Super Admin', 'org.r2': 'Staff', 'org.r3': 'Partner', 'org.r4': 'Cluster Manager',
    'org.r5': 'Chairperson · Secretary · Treasurer', 'org.r6': 'Committee', 'org.r7': 'Member',
    'org.n1': 'Partner<small>The organisation running the programme</small>',
    'org.n2': 'Cluster<small>An area or set of groups</small>',
    'org.n3': 'Group<small>Leaders, constitution and meetings</small>',
    'org.n4': 'Member<small>Their savings, shares, loans and SMS</small>',
    'rep.eyebrow': 'Reports & insight',
    'rep.title': 'Accurate Numbers, Ready to Download',
    'rep.lead': 'Balances are always calculated from the real transactions — never from stored totals. Filter by partner, cluster, group and date.',
    'rep.s1': 'Report types — finance, groups, meetings and SMS',
    'rep.s2': 'Download formats: Excel, PDF and CSV',
    'rep.s3': 'SMS message types in Swahili and English',
    'rep.s4u': 'days', 'rep.s4': 'PAR 30 — loans at risk show up early',
    'how.eyebrow': 'How it works', 'how.title': 'Four Steps to Start',
    'how.s1.t': 'Register the Group', 'how.s1.p': "An administrator adds the group, its cluster and the Chairperson's phone number.",
    'how.s2.t': 'Set the Constitution', 'how.s2.p': 'Share value, interest, fines and the services the group uses.',
    'how.s3.t': 'Run Meetings', 'how.s3.p': 'Leaders sign in to the app and record attendance and contributions — members get an SMS.',
    'how.s4.t': 'Follow the Reports', 'how.s4.p': 'Partners and supervisors see the dashboard and download reports for their area.',
    'sms.eyebrow': 'Communication',
    'sms.title': 'Every Member Knows Where They Stand',
    'sms.lead': "Even without a smartphone, a member receives an SMS confirmation for every transaction — in a language they understand. Administrators can edit the messages.",
    'sms.l1': 'Confirmations for savings, shares, loans and fines',
    'sms.l2': 'Reminders for meetings and loans falling due',
    'sms.l3': 'A report of sent and failed messages',
    'sms.b1': 'Hello Asha, your purchase of 3 shares worth TSh 15,000 in Kikundi cha Umoja is confirmed. Thank you.',
    'sms.b2': 'Hello Juma, your loan repayment of TSh 50,000 in Kikundi cha Umoja is confirmed. Thank you.',
    'sms.b3': 'Hello Neema, a reminder that the Kikundi cha Umoja meeting is tomorrow. See you there.',
    'faq.eyebrow': 'Questions', 'faq.title': 'Frequently Asked Questions',
    'faq.q1': 'If a transaction is wrong, is it deleted?',
    'faq.a1': "No. Money is never deleted or edited. A wrong transaction is reversed: it stays in the history, marked as reversed, and the group, loan or fine balance is put back as it was.",
    'faq.q2': "Who can see our group's information?",
    'faq.a2': "Your group's leaders, and supervisors assigned to your group, cluster or partner. Every user sees only the groups inside their own area.",
    'faq.q3': 'Does every member need a smartphone?',
    'faq.a3': 'No. The leaders use the app during the meeting. Members receive SMS confirmations on any phone.',
    'faq.q4': 'How do we sign in?',
    'faq.a4': "With your phone number and a one-time code (OTP) sent by SMS. Only registered users — or the Chairperson of a registered group — can sign in.",
    'faq.q5': 'Once a meeting is closed, can we add contributions?',
    'faq.a5': "No. A closed meeting is locked so its records can't change. New contributions go into the next meeting.",
    'cta.eyebrow': 'Start today',
    'cta.title': 'Your Group Deserves Clear Accounts',
    'cta.lead': 'Already registered? Open the dashboard. Want your group connected? Contact your programme partner or administrator.',
    'foot.about': 'HelaBox is a digital platform for savings and lending groups — for leaders, members, partners and programme supervisors.',
    'foot.product': 'Product', 'foot.p1': 'Savings & Shares', 'foot.p2': 'Loans', 'foot.p3': 'Meetings', 'foot.p4': 'Reports',
    'foot.learn': 'Learn', 'foot.access': 'Sign in', 'foot.a1': 'Admin Dashboard',
    'foot.rights': 'All rights reserved.', 'foot.made': 'Made in Tanzania',
    'nav.app': 'Get the App',
    'app.eyebrow': 'Android app', 'app.title': 'Get HelaBox on Your Phone',
    'app.lead': 'For group leaders: record attendance, savings and loans during the meeting — no computer needed.',
    'app.android': 'Android', 'app.version': 'Version', 'app.download': 'Download APK',
    'app.hint': 'If your phone asks, allow installing apps from your browser. Then sign in with your phone number.',
    'app.tag': 'Savings Groups',
    'foot.privacy': 'Privacy Policy'
  }
  var META = {
    sw: { title: document.title, menu: 'Fungua menyu', close: 'Funga', top: 'Rudi juu' },
    en: { title: 'HelaBox — Savings Groups, Digital', menu: 'Open menu', close: 'Close', top: 'Back to top' }
  }

  var nodes = Array.prototype.slice.call(document.querySelectorAll('[data-i18n]'))
  var SW = {}
  nodes.forEach(function (el) { SW[el.getAttribute('data-i18n')] = el.innerHTML })

  function store(k, v) { try { localStorage.setItem(k, v) } catch (e) {} }
  function load(k) { try { return localStorage.getItem(k) } catch (e) { return null } }

  function setLang(lang) {
    var dict = lang === 'en' ? EN : SW
    nodes.forEach(function (el) {
      var k = el.getAttribute('data-i18n')
      if (dict[k] != null) el.innerHTML = dict[k]
    })
    document.documentElement.lang = lang
    document.title = META[lang].title
    document.querySelectorAll('.lang button').forEach(function (b) {
      b.setAttribute('aria-pressed', String(b.getAttribute('data-lang') === lang))
    })
    var m = document.querySelector('.menu-btn'); if (m) m.setAttribute('aria-label', META[lang].menu)
    var c = document.querySelector('.drawer .close'); if (c) c.setAttribute('aria-label', META[lang].close)
    var t = document.querySelector('.to-top'); if (t) t.setAttribute('aria-label', META[lang].top)
    store('hb-lang', lang)
    renderApp()
  }
  document.querySelectorAll('.lang button').forEach(function (b) {
    b.addEventListener('click', function () { setLang(b.getAttribute('data-lang')) })
  })

  // ---- header: transparent over hero, white when scrolled, hides on scroll down
  var header = document.querySelector('.site-header')
  var toTop = document.querySelector('.to-top')
  var lastY = window.scrollY
  function onScroll() {
    var y = window.scrollY
    header.classList.toggle('scrolled', y > 40)
    header.classList.toggle('hidden', y > 400 && y > lastY && !drawer.classList.contains('open'))
    toTop.classList.toggle('show', y > 700)
    lastY = y
  }
  window.addEventListener('scroll', onScroll, { passive: true })
  toTop.addEventListener('click', function () { window.scrollTo({ top: 0, behavior: 'smooth' }) })

  // ---- mobile drawer
  var drawer = document.getElementById('drawer')
  var menuBtn = document.querySelector('.menu-btn')
  function setDrawer(open) {
    drawer.classList.toggle('open', open)
    drawer.setAttribute('aria-hidden', String(!open))
    menuBtn.setAttribute('aria-expanded', String(open))
    document.body.style.overflow = open ? 'hidden' : ''
    if (open) drawer.querySelector('.close').focus()
    else menuBtn.focus({ preventScroll: true })
  }
  menuBtn.addEventListener('click', function () { setDrawer(true) })
  drawer.querySelectorAll('[data-close]').forEach(function (el) {
    el.addEventListener('click', function () { setDrawer(false) })
  })
  document.addEventListener('keydown', function (e) {
    if (e.key === 'Escape' && drawer.classList.contains('open')) setDrawer(false)
  })

  // ---- scroll reveal
  var reveals = document.querySelectorAll('.reveal')
  if ('IntersectionObserver' in window) {
    var io = new IntersectionObserver(function (entries) {
      entries.forEach(function (en) {
        if (en.isIntersecting) { en.target.classList.add('in'); io.unobserve(en.target) }
      })
    }, { threshold: 0.12, rootMargin: '0px 0px -40px 0px' })
    reveals.forEach(function (el) { io.observe(el) })
  } else {
    reveals.forEach(function (el) { el.classList.add('in') })
  }

  // ---- app download: shown only once an APK has been published (dashboard → Mobile App)
  var appInfo = null
  function fmtSize(n) { return n >= 1048576 ? (n / 1048576).toFixed(1) + ' MB' : Math.round(n / 1024) + ' KB' }
  function renderApp() {
    if (!appInfo) return
    var lang = document.documentElement.lang === 'en' ? 'en-GB' : 'sw-TZ'
    var date = ''
    try { date = new Date(appInfo.releasedAt).toLocaleDateString(lang, { day: 'numeric', month: 'short', year: 'numeric', timeZone: 'Africa/Dar_es_Salaam' }) } catch (e) {}
    var set = function (k, v) { document.querySelectorAll('[data-app="' + k + '"]').forEach(function (el) { el.textContent = v }) }
    set('version', appInfo.version)
    set('size', fmtSize(appInfo.size || 0))
    set('date', date)
    set('chip', 'v' + appInfo.version)
    var notes = document.querySelector('[data-app="notes"]')
    if (notes) { notes.textContent = appInfo.notes || ''; notes.hidden = !appInfo.notes }
    document.querySelectorAll('[data-app="link"]').forEach(function (a) { a.href = appInfo.downloadUrl || '/api/public/app/download' })
  }
  if (window.fetch) {
    fetch('/api/public/app', { headers: { Accept: 'application/json' } })
      .then(function (r) { return r.ok ? r.json() : null })
      .then(function (info) {
        if (!info || !info.available) return
        appInfo = info
        renderApp()
        document.querySelectorAll('[data-app-only]').forEach(function (el) { el.hidden = false })
        var sec = document.getElementById('app')
        if (sec && io) sec.querySelectorAll('.reveal').forEach(function (el) { io.observe(el) })
        if (location.hash === '#app' && sec) sec.scrollIntoView()
      })
      .catch(function () {})
  }

  // ---- init
  var y = document.getElementById('year'); if (y) y.textContent = new Date().getFullYear()
  var saved = load('hb-lang')
  if (saved === 'en') setLang('en')
  onScroll()
})()
