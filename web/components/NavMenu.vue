<script setup lang="ts">
import { routes } from '@/route'
import { Moon, Sunny } from '@element-plus/icons-vue'
import { useI18nStore } from '@/stores/i18n'

const dark = useDark()
const route = useRoute()
const router = useRouter()
const i18n = useI18nStore()
</script>

<template lang="pug">
el-affix
  el-menu(mode="horizontal" :default-active="route.path" @select="router.push")
    el-menu-item.logo(index="/") DOJ
    template(v-for="item in routes")
      el-menu-item(v-if="item.name" :key="item.path" :index="item.path") {{ $t(item.name) }}
    el-space.right(size="large")
      user-menu
      el-switch.theme(v-model="dark" inline-prompt :active-icon="Moon" :inactive-icon="Sunny")
      el-dropdown(@command="i18n.setLocale")
        el-icon(:size="24")
          i-ion-language-outline
        template(#dropdown)
          el-dropdown-menu
            template(v-for="l in i18n.locales" :key="l")
              el-dropdown-item(:command="l" :disabled="l === i18n.locale") {{ $t('language', l, { locale: l }) }}
</template>

<style scoped lang="sass">
.logo
  font-size: 30px
  font-weight: bold
  font-family: Menlo, Consolas, monospace
.right
  margin-left: auto
  padding-left: 1em
.theme
  --el-switch-on-color: var(--el-border-color)
</style>
