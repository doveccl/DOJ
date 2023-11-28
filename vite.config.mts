import vue from '@vitejs/plugin-vue'
import vueI18n from '@intlify/unplugin-vue-i18n/vite'
import Icons from 'unplugin-icons/vite'
import IconsResolver from 'unplugin-icons/resolver'
import AutoImport from 'unplugin-auto-import/vite'
import Components from 'unplugin-vue-components/vite'

import { resolve } from 'path'
import { defineConfig } from 'vite'
import { ElementPlusResolver } from 'unplugin-vue-components/resolvers'
import type { ImportsMap } from 'unplugin-auto-import/types'

const auto: ImportsMap = {
  axios: [['default', 'axios']]
}

export default defineConfig({
  resolve: {
    alias: {
      '@': resolve('web'),
      '@pages': resolve('web/pages'),
      '@stores': resolve('web/stores'),
      '@components': resolve('web/components')
    }
  },
  plugins: [
    vue(),
    vueI18n({
      include: resolve('web/locales/**')
    }),
    Icons(),
    AutoImport({
      dts: 'web/imports.d.ts',
      resolvers: [IconsResolver(), ElementPlusResolver()],
      imports: ['vue', 'vue-router', 'vue-i18n', 'pinia', '@vueuse/core', auto]
    }),
    Components({
      dirs: ['web/components'],
      dts: 'web/components.d.ts',
      resolvers: [IconsResolver(), ElementPlusResolver()]
    })
  ],
  server: {
    port: 28080,
    proxy: {
      '/ws': 'http://localhost:7974',
      '/api': 'http://localhost:7974'
    }
  }
})
