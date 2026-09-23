// Users & roles — the sanitized admin routes in server/access_routes.go (the
// built-in User model is never exposed over GraphQL). The 7 roles follow
// "Pesa Box User Levels.pdf" (level 1 → 7).
export { listUsers, addUser, updateUser, deactivateUser, addAssignment, removeAssignment } from './admin.js'

export const ROLES = [
  'super_admin',
  'staff',
  'partner_user',
  'cluster_manager',
  'group_admin',
  'group_officer',
  'group_member'
]

export const PRESETS = ['viewer', 'support', 'operations']

export const POSITIONS = ['mwenyekiti', 'katibu', 'mweka_hazina', 'committee']
