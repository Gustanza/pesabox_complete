import { createRouter, createWebHistory } from 'vue-router'
import { currentUser } from '@/api/auth'
import { loadAccess, can, clearAccess } from '@/api/access'

const routes = [
  {
    path: '/login',
    name: 'login',
    component: () => import('@/views/LoginView.vue')
  },
  {
    path: '/register',
    name: 'register',
    component: () => import('@/views/RegisterView.vue')
  },
  {
    path: '/otp',
    name: 'otp',
    component: () => import('@/views/OtpView.vue')
  },
  {
    path: '/complete-profile',
    name: 'complete-profile',
    component: () => import('@/views/CompleteProfileView.vue')
  },
  {
    // Signed in, but the role has no web-dashboard access (group officers,
    // members, accounts not given a role yet).
    path: '/no-access',
    name: 'no-access',
    component: () => import('@/views/NoAccessView.vue')
  },
  {
    path: '/',
    component: () => import('@/components/AppLayout.vue'),
    children: [
      {
        path: '',
        alias: 'dashboard',
        name: 'dashboard',
        meta: { perm: 'dashboard.view' },
        component: () => import('@/views/DashboardView.vue')
      },
      {
        path: 'groups',
        name: 'groups',
        meta: { perm: 'dashboard.view' },
        component: () => import('@/views/GroupsListView.vue')
      },
      {
        path: 'groups/create',
        name: 'create-group',
        meta: { perm: 'groups.create' },
        component: () => import('@/views/CreateGroupView.vue')
      },
      {
        path: 'groups/:id',
        name: 'group-details',
        meta: { perm: 'dashboard.view' },
        component: () => import('@/views/GroupDetailsView.vue')
      },
      {
        path: 'groups/:id/edit',
        name: 'edit-group',
        meta: { perm: 'group.settings' },
        component: () => import('@/views/CreateGroupView.vue')
      },
      {
        path: 'structure',
        name: 'structure',
        meta: { perm: 'structure.view' },
        component: () => import('@/views/StructureView.vue')
      },
      {
        path: 'users',
        name: 'users',
        meta: { perm: 'platform.manage' },
        component: () => import('@/views/UserManagementView.vue')
      },
      {
        path: 'finance',
        name: 'finance',
        meta: { perm: 'reports.view' },
        component: () => import('@/views/FinancialOverviewView.vue')
      },
      {
        path: 'sms',
        name: 'sms',
        meta: { perm: 'sms.view' },
        component: () => import('@/views/SmsDashboardView.vue')
      },
      {
        path: 'reports',
        name: 'reports',
        meta: { perm: 'reports.view' },
        component: () => import('@/views/ReportsView.vue')
      },
      {
        path: 'audit',
        name: 'audit',
        meta: { perm: 'audit.view' },
        component: () => import('@/views/AuditLogsView.vue')
      },
      {
        path: 'settings',
        name: 'settings',
        meta: { perm: 'platform.manage' },
        component: () => import('@/views/SettingsView.vue')
      },
      {
        path: 'profile',
        name: 'profile',
        component: () => import('@/views/ProfileView.vue')
      }
    ]
  },
  // Any unmatched path (typos, stale links) falls back to the dashboard
  // instead of silently rendering a blank page.
  { path: '/:pathMatch(.*)*', redirect: '/' }
]

const router = createRouter({
  history: createWebHistory(),
  routes
})

const publicPaths = ['/login', '/register', '/otp']

// First page a role may open, in sidebar order.
const LANDING = [
  ['dashboard.view', '/'],
  ['reports.view', '/reports'],
  ['platform.manage', '/users']
]

router.beforeEach(async (to) => {
  const isPublic = publicPaths.includes(to.path)

  // Belt-and-braces: currentUser() already swallows its own errors, but a
  // guard that throws or rejects makes Vue Router silently cancel the
  // navigation, leaving the user stuck on the page they were trying to
  // leave with no error shown. Never let that happen here.
  let user = null
  try {
    user = await currentUser()
  } catch (e) {
    user = null
  }
  const isAuthenticated = !!user

  if (!isAuthenticated) {
    clearAccess()
  }

  if (!isPublic && !isAuthenticated) {
    return '/login'
  }

  if ((to.path === '/login' || to.path === '/register') && isAuthenticated) {
    return '/'
  }

  // The OTP login auto-creates a User with no name (see server/main.go's
  // /api/me comment) — herd anyone in that state to complete-profile before
  // they can reach the rest of the app, and away from it once they're done.
  if (isAuthenticated) {
    const profileIncomplete = !user.firstName
    if (profileIncomplete && to.path !== '/complete-profile') {
      return '/complete-profile'
    }
    if (!profileIncomplete && to.path === '/complete-profile') {
      return '/'
    }
    if (profileIncomplete) return

    // Role check (server/access.go). A page the role can't use sends the
    // user to the first page it can — or /no-access when there is none.
    await loadAccess() // cached; cleared on logout / when the session ends
    const landing = LANDING.find(([perm]) => can(perm))?.[1] || '/no-access'
    if (to.path === '/no-access') {
      return landing === '/no-access' ? undefined : landing
    }
    if (to.meta?.perm && !can(to.meta.perm)) {
      return landing === to.path ? '/no-access' : landing
    }
  }
})

export default router