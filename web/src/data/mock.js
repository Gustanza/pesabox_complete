export const AVA = ['#18A672', '#D9A441', '#3E7BFA', '#E15454', '#136B54', '#9B6BD9']

export function initials(name) {
  return name.split(' ').map(w => w[0]).slice(0, 2).join('').toUpperCase()
}

export function avaColor(name) {
  const idx = [...name].reduce((a, c) => a + c.charCodeAt(0), 0) % AVA.length
  return AVA[idx]
}

export const GROUPS = [
  {
    id: 'PB-GRP-000124', name: 'Upendo Vikoba', admin: 'Neema Joseph', members: 24,
    cycle: '3/30', status: 'Active', region: 'Dar es Salaam', freq: 'Monthly',
    savings: 4250000, shares: 3100000, social: 480000, loans: 1850000
  },
  {
    id: 'PB-GRP-000131', name: 'Mshikamano', admin: 'John Michael', members: 18,
    cycle: '18/24', status: 'Active', region: 'Arusha', freq: 'Weekly',
    savings: 2100000, shares: 1450000, social: 210000, loans: 980000
  },
  {
    id: 'PB-GRP-000098', name: 'Tumaini Group', admin: 'Asha Maria', members: 31,
    cycle: '12/30', status: 'Active', region: 'Mwanza', freq: 'Monthly',
    savings: 3600000, shares: 2050000, social: 390000, loans: 640000
  },
  {
    id: 'PB-GRP-000062', name: 'Umoja Women', admin: 'Grace Peter', members: 27,
    cycle: '30/30', status: 'Closed', region: 'Dodoma', freq: 'Monthly',
    savings: 5200000, shares: 3800000, social: 650000, loans: 0
  }
]

export const PENDING_ADMINS = [
  { id: 'ADM-3391', name: 'Fatuma Ramadhani', phone: '+255 754 221 908', registered: '10 Sep 2026' },
  { id: 'ADM-3402', name: 'Peter Kessy', phone: '+255 713 664 220', registered: '11 Sep 2026' },
  { id: 'ADM-3407', name: 'Halima Suleiman', phone: '+255 788 902 371', registered: '12 Sep 2026' }
]

export const USERS = [
  { name: 'Raymond', phone: '+255 767 400 812', role: 'Super Admin', group: '\u2014', groupId: null, status: 'Active' },
  { name: 'Support Admin', phone: '+255 767 400 900', role: 'Support Admin', group: '\u2014', groupId: null, status: 'Active' },
  ...GROUPS.map(g => ({
    name: g.admin,
    phone: '+255 7' + Math.abs(g.id.split('-')[2]).toString().padStart(8, '2'),
    role: 'Group Admin', group: g.name, groupId: g.id, status: 'Active'
  }))
]

export const MEMBERS = [
  { no: 'M-001', name: 'Neema Joseph', shares: 12, savings: 60000, loan: 120000, status: 'Active' },
  { no: 'M-002', name: 'Asha Mwangi', shares: 8, savings: 40000, loan: 0, status: 'Active' },
  { no: 'M-003', name: 'John Mfinanga', shares: 15, savings: 75000, loan: 220000, status: 'Active' },
  { no: 'M-004', name: 'Rehema Abdallah', shares: 6, savings: 30000, loan: 0, status: 'Active' }
]

export const MEETINGS = [
  { id: 'MTG-024', group: 'Upendo', no: '#24', date: '12 Sep 2026', att: '23/24', status: 'Closed', shares: 75000, savings: 120000, social: 48000, repay: 180000, fines: 3000, tx: 42 },
  { id: 'MTG-023', group: 'Upendo', no: '#23', date: '05 Sep 2026', att: '24/24', status: 'Closed', shares: 60000, savings: 110000, social: 44000, repay: 150000, fines: 1000, tx: 38 },
  { id: 'MTG-018', group: 'Mshikamano', no: '#18', date: '12 Sep 2026', att: '17/18', status: 'Closed', shares: 40000, savings: 90000, social: 36000, repay: 60000, fines: 2000, tx: 29 },
  { id: 'MTG-012', group: 'Tumaini', no: '#12', date: '12 Sep 2026', att: '21/30', status: 'Open', shares: 0, savings: 0, social: 0, repay: 0, fines: 0, tx: 0 }
]

export const CYCLES = [
  { id: 'CYC-UP3', group: 'Upendo', cycle: 'Cycle 3', held: 24, total: 30, status: 'Active', started: '01 Jan 2026' },
  { id: 'CYC-MS2', group: 'Mshikamano', cycle: 'Cycle 2', held: 18, total: 24, status: 'Active', started: '15 Mar 2026' },
  { id: 'CYC-TU1', group: 'Tumaini', cycle: 'Cycle 1', held: 30, total: 30, status: 'Closed', started: '02 Jan 2025' },
  { id: 'CYC-UW4', group: 'Umoja', cycle: 'Cycle 4', held: 30, total: 30, status: 'Closed', started: '10 Feb 2025' }
]

export const TX = [
  { id: 'TX-001', group: 'Upendo', member: 'Neema', type: 'Share', amount: '15,000', date: '12 Sep' },
  { id: 'TX-002', group: 'Upendo', member: 'Asha', type: 'Saving', amount: '5,000', date: '12 Sep' },
  { id: 'TX-003', group: 'Tumaini', member: 'John', type: 'Repayment', amount: '20,000', date: '12 Sep' },
  { id: 'TX-004', group: 'Umoja', member: 'Grace', type: 'Fine', amount: '1,000', date: '11 Sep' }
]

export const LOANS = [
  { group: 'Upendo', borrower: 'Neema', principal: '300,000', balance: '120,000', status: 'Active' },
  { group: 'Mshikamano', borrower: 'John', principal: '500,000', balance: '250,000', status: 'Overdue' }
]

export const AUDIT = [
  { time: '09:42', user: 'Admin', action: 'Suspended Group', resource: 'Tumaini' },
  { time: '09:38', user: 'Admin', action: 'Updated Template', resource: 'Loan Reminder' },
  { time: '09:34', user: 'Neema', action: 'Recorded Transaction', resource: 'TX-001' },
  { time: '09:31', user: 'Neema', action: 'Closed Meeting', resource: 'Meeting #24' }
]

export const NAV_ITEMS = [
  { key: 'dashboard', label: 'Dashboard', icon: 'dash', route: '/dashboard' },
  { key: 'groups', label: 'Groups', icon: 'groups', route: '/groups' },
  { key: 'users', label: 'Users', icon: 'user', route: '/users' },
  { key: 'finance', label: 'Finance', icon: 'fin', route: '/finance' },
  { key: 'sms', label: 'SMS', icon: 'sms', route: '/sms' },
  { key: 'reports', label: 'Reports', icon: 'rep', route: '/reports' },
  { key: 'audit', label: 'Audit Logs', icon: 'audit', route: '/audit' },
  { key: 'settings', label: 'Settings', icon: 'settings', route: '/settings' }
]

export function findGroupById(id) {
  return GROUPS.find(g => g.id === id) || GROUPS[0]
}

export function findGroupByKey(key) {
  return GROUPS.find(g => g.name.split(' ')[0] === key) || GROUPS[0]
}
