import react from '@vitejs/plugin-react'
import { defineConfig } from 'vite'

export default defineConfig({
  plugins: [react()],
  build: {
    assetsInlineLimit: 0
  },
  server: {
    port: 28080,
    proxy: {
      '/api': 'http://localhost:7974'
    }
  }
})
