import axios from 'axios'

// In dev, Vite proxies /api to the Go backend (see vite.config.js).
// Set VITE_API_BASE to an absolute URL in production (e.g. https://api.example.com).
const baseURL = import.meta.env.VITE_API_BASE || '/api'

const client = axios.create({
  baseURL,
  headers: { 'Content-Type': 'application/json' }
})

// Attach the auth token to every request when present.
client.interceptors.request.use((config) => {
  const token = localStorage.getItem('pb_token')
  if (token) config.headers.Authorization = `Bearer ${token}`
  return config
})

client.interceptors.response.use(
  (res) => res,
  (error) => {
    if (error.response?.status === 401) {
      localStorage.removeItem('pb_token')
    }
    return Promise.reject(error)
  }
)

// Generic helpers: the API wraps payloads in { data: ... }.
export function unwrap(promise) {
  return promise.then((res) => res.data?.data ?? res.data)
}

export const api = {
  get: (path, config) => unwrap(client.get(path, config)),
  post: (path, body, config) => unwrap(client.post(path, body, config)),
  put: (path, body, config) => unwrap(client.put(path, body, config)),
  patch: (path, body, config) => unwrap(client.patch(path, body, config)),
  del: (path, config) => unwrap(client.delete(path, config))
}

export default client