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
    // Phase 1.
    proxy: {
      '/api': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      },
    },
  },
})
