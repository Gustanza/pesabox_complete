import { createRouter, createWebHistory } from 'vue-router'
import { currentUser } from '@/api/auth'

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
    path: '/',
    component: () => import('@/components/AppLayout.vue'),
    children: [
      {
        path: '',
        alias: 'dashboard',
        name: 'dashboard',
        component: () => import('@/views/DashboardView.vue')
      },
      {
        path: 'groups',
        name: 'groups',
        component: () => import('@/views/GroupsListView.vue')
      },
      {
        path: 'groups/create',
        name: 'create-group',
        component: () => import('@/views/CreateGroupView.vue')
      },
      {
        path: 'groups/:id',
        name: 'group-details',
        component: () => import('@/views/GroupDetailsView.vue')
      },
      {
        path: 'groups/:id/edit',
        name: 'edit-group',
        component: () => import('@/views/CreateGroupView.vue')
      },
      {
        path: 'users',
        name: 'users',
        component: () => import('@/views/UserManagementView.vue')
      },
      {
        path: 'finance',
        name: 'finance',
        component: () => import('@/views/FinancialOverviewView.vue')
      },
      {
        path: 'sms',
        name: 'sms',
        component: () => import('@/views/SmsDashboardView.vue')
      },
      {
        path: 'reports',
        name: 'reports',
        component: () => import('@/views/ReportsView.vue')
      },
      {
        path: 'audit',
        name: 'audit',
        component: () => import('@/views/AuditLogsView.vue')
      },
      {
        path: 'settings',
        name: 'settings',
        component: () => import('@/views/SettingsView.vue')
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

  if (!isPublic && !isAuthenticated) {
    return '/login'
  }

  if ((to.path === '/login' || to.path === '/register') && isAuthenticated) {
    return '/'
  }
})

export default router