// Talks to the Yekonga auth GraphQL endpoint (/graphql/auth). Registration and
// login both go through a single OTP flow: request a code with requestOtp(),
// then complete registration/login with the same code via verifyOtp(). The
// server sets httpOnly session cookies on success, so there is no token to
// store here.
import { apiFetch } from './http.js'

const AUTH_ENDPOINT = '/graphql/auth'

async function gql(query, variables) {
  const res = await fetch(AUTH_ENDPOINT, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    credentials: 'include',
    body: JSON.stringify({ query, variables })
  })

  const json = await res.json()

  if (json.errors?.length) {
    throw new Error(json.errors[0].message || 'Request failed')
  }

  return json.data
}

export function requestOtp(phone) {
  return gql(
    `mutation($input: OtpInput!){
      otp(input: $input, type: phone) { status message }
    }`,
    { input: { username: phone, usernameType: 'phone' } }
  ).then((data) => data.otp)
}

export function verifyOtp(phone, code) {
  return gql(
    `mutation($input: LoginInput!){
      login(input: $input, type: phone) {
        id username email phone firstName lastName role status isActive
      }
    }`,
    {
      input: { username: phone, usernameType: 'phone', password: code, type: 'OTP' }
    }
  ).then((data) => data.login)
}

export function currentUser() {
  // Never let this reject — it runs on every route change (see the router's
  // navigation guard), and an unhandled rejection there silently cancels the
  // navigation, leaving the user stuck on whatever page they were trying to
  // leave with no error shown. apiFetch already retries once through
  // /refresh on a 401 (the access token only lives 15 minutes), so this only
  // comes back null once the refresh token itself is dead too.
  return apiFetch('/me')
    .then((res) => (res.ok ? res.json() : null))
    .catch(() => null)
}

// Fills in firstName/lastName/email for a session whose account was just
// auto-created by the OTP login (see server/main.go's /api/me) — the phone
// number used to sign in stays the login identifier and is never touched here.
export function updateProfile(changes) {
  return apiFetch('/api/me', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(changes)
  }).then(async (res) => {
    const json = await res.json().catch(() => null)
    if (!res.ok) {
      throw new Error(json?.error || 'Request failed')
    }
    return json
  })
}

export function logout() {
  // Ask for JSON explicitly: without an Accept header the server treats this
  // as a plain browser navigation and replies with a redirect instead.
  return fetch('/logout', {
    credentials: 'include',
    headers: { Accept: 'application/json' }
  })
}
