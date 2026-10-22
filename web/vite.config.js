import { fileURLToPath, URL } from 'node:url'

import vue from '@vitejs/plugin-vue'
import { defineConfig } from 'vite'

// https://vite.dev/config/
export default defineConfig({
  // Public path the dashboard is served under, e.g. VITE_BASE=/admin/ for
  // helabox.co.tz/admin. Default: site root.
  base: process.env.VITE_BASE || '/',
  plugins: [vue()],
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url)),
    },
  },
  server: {
    proxy: {
      '/api': {
        target: 'http://localhost:8090',
        changeOrigin: true,
      },
      '/graphql': {
        target: 'http://localhost:8090',
        changeOrigin: true,
      },
      '/me': {
        target: 'http://localhost:8090',
        changeOrigin: true,
      },
      '/logout': {
        target: 'http://localhost:8090',
        changeOrigin: true,
      },
      '/refresh': {
        target: 'http://localhost:8090',
        changeOrigin: true,
      },
    },
  },
})
