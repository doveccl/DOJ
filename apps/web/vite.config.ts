import vue from '@vitejs/plugin-vue'
import { defineConfig } from 'vite'

export default defineConfig({
  plugins: [vue()],
  server: {
    port: Number(process.env.DOJ_WEB_PORT ?? 28080),
    proxy: {
      '/api': 'http://localhost:7974'
    }
  }
})
