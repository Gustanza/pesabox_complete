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
let refreshing = null

function doRefresh() {
  if (!refreshing) {
    refreshing = fetch('/refresh', {
      credentials: 'include',
      headers: { Accept: 'application/json' }
    }).finally(() => {
      refreshing = null
    })
  }
  return refreshing
}

export async function apiFetch(url, options = {}) {
  const opts = { credentials: 'include', ...options }
  let res = await fetch(url, opts)

  if (res.status === 401) {
    const refreshRes = await doRefresh()
    if (refreshRes.ok) {
      res = await fetch(url, opts)
    }
  }

  return res
}
