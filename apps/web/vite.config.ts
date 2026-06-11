import vue from '@vitejs/plugin-vue'
import AutoImport from 'unplugin-auto-import/vite'
import Components from 'unplugin-vue-components/vite'
import { NaiveUiResolver } from 'unplugin-vue-components/resolvers'
import { defineConfig } from 'vite'

export default defineConfig({
  plugins: [
    vue(),
    AutoImport({
      imports: ['vue', 'pinia', { 'vue-router': ['createRouter', 'createWebHistory', 'RouterLink', 'useRoute', 'useRouter'] }],
      dirs: ['src/composables', 'src/stores'],
      dts: 'src/autoImports.d.ts'
    }),
    Components({
      dirs: ['src/components'],
      dts: 'src/components.d.ts',
      resolvers: [NaiveUiResolver()]
    })
  ],
  server: {
    port: Number(process.env.WEB_PORT ?? 28080),
    proxy: {
      '/api': {
        target: 'http://localhost:7974',
        ws: true
      }
    }
  }
})
