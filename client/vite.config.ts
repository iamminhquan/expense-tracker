import tailwindcss from '@tailwindcss/vite'
import react from '@vitejs/plugin-react'
import { defineConfig } from 'vite'

// https://vite.dev/config/
export default defineConfig({
  plugins: [react(), tailwindcss()],
  server: {
    // Proxy /api to the Gin server during local dev so the browser sees one
    // origin and cookies (the refresh-token cookie) behave the same way they
    // will against a same-site deployment. In production the SPA (Vercel)
    // and the API (Render) are genuinely cross-origin -- see CORS setup in
    // Phase 1. Target is overridable (VITE_API_PROXY_TARGET) for a server
    // run on a non-default port, e.g. alongside another instance already
    // on 8080.
    proxy: {
      '/api': {
        target: process.env.VITE_API_PROXY_TARGET ?? 'http://localhost:8080',
        changeOrigin: true,
      },
    },
  },
})
