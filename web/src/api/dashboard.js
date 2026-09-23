// Live programme snapshot (server/report_kpis.go at /api/admin/dashboard),
// scoped to what the signed-in role may see and optionally narrowed with
// { partnerId, clusterId, groupId }.
export { getDashboard, getRollup } from './admin.js'
