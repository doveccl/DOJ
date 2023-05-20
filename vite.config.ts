import { defineConfig } from 'vite'

export default defineConfig({
  server: {
    proxy: {
      '/api': 'http://127.0.0.1:7974',
      '/ws': 'http://127.0.0.1:7974'
    }
  }
})
