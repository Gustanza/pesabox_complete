// Shared fetch wrapper for every api/*.js module. The access token cookie
// only lives 15 minutes (server default — see config.json / yekonga's
// AccessTokenExpireTime), so any session left open longer than that starts
// getting 401s on everything. The refresh token cookie lives 7 days, so on a
// 401 we try /refresh once (it reads the refresh_token cookie itself) and
// replay the original request — the user only actually gets bounced to
// /login once the refresh token itself is dead.
//
// /refresh rotates (revokes) the refresh token on every use, so concurrent
// 401s must share one in-flight refresh instead of each calling /refresh
// with the same soon-to-be-revoked token.
//
// API_ORIGIN: empty when the dashboard is served next to the API (dev: Vite
// proxy). Set VITE_API_ORIGIN at build time (e.g. https://server.helabox.co.tz)
// when the dashboard is hosted on another domain; the session cookies still
// work because the server scopes them to the page's domain.
export const API_ORIGIN = (import.meta.env.VITE_API_ORIGIN || '').replace(/\/+$/, '')
export const apiUrl = (path) => (path.startsWith('/') ? API_ORIGIN + path : path)

let refreshing = null

function doRefresh() {
  if (!refreshing) {
    refreshing = fetch(apiUrl('/refresh'), {
      credentials: 'include',
      headers: { Accept: 'application/json' }
    }).finally(() => {
      refreshing = null
    })
  }
  return refreshing
}

export async function apiFetch(url, options = {}) {
  // Accept JSON so an expired token comes back as 401 (refreshable) rather
  // than a redirect, which a cross-origin fetch can't follow.
  const opts = { credentials: 'include', ...options, headers: { Accept: 'application/json', ...options.headers } }
  url = apiUrl(url)
  let res = await fetch(url, opts)

  if (res.status === 401) {
    const refreshRes = await doRefresh()
    if (refreshRes.ok) {
      res = await fetch(url, opts)
    }
  }

  return res
}
