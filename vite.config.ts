import { resolve } from 'path'
import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import vueI18n from '@intlify/vite-plugin-vue-i18n'
import Icons from 'unplugin-icons/vite'
import IconsResolver from 'unplugin-icons/resolver'
import AutoImport from 'unplugin-auto-import/vite'
import Components from 'unplugin-vue-components/vite'
import { ElementPlusResolver } from 'unplugin-vue-components/resolvers'

export default defineConfig({
  resolve: {
    alias: {
      '@': resolve(__dirname, 'web'),
      '@pages': resolve(__dirname, 'web/pages'),
      '@stores': resolve(__dirname, 'web/stores'),
      '@components': resolve(__dirname, 'web/components')
    }
  },
  plugins: [
    vue(),
    vueI18n({
      include: resolve(__dirname, 'web/locales/**')
    }),
    Icons(),
    AutoImport({
      dts: 'web/imports.d.ts',
      imports: ['vue', 'vue-router', 'vue-i18n', 'pinia']
    }),
    Components({
      dirs: ['web/components'],
      dts: 'web/components.d.ts',
      resolvers: [ElementPlusResolver(), IconsResolver()]
    })
  ],
  server: {
    proxy: {
      '/ws': 'http://localhost:7974',
      '/api': 'http://localhost:7974'
    }
  }
})
