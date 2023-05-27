<script setup lang="ts">
import { routes } from '@/route'
import { Moon, Sunny } from '@element-plus/icons-vue'

const dark = useDark()
const route = useRoute()
const router = useRouter()
</script>

<template lang="pug">
el-menu#nav(mode="horizontal" :default-active="route.path" @select="router.push")
  el-menu-item.monospace(index="/")
    b(style="font-size: 2em") DOJ
  template(v-for="item in routes")
    el-menu-item(v-if="item.name" :index="item.path" :key="item.path") {{ $t(item.name) }}
  el-space#subnav(size="large")
    user-menu
    el-switch(
      v-model="dark"
      active-color="var(--el-border-color)"
      inline-prompt
      :active-icon="Moon"
      :inactive-icon="Sunny"
    )
    el-dropdown(@command="l => ($i18n.locale = l)")
      el-icon(:size="24")
        i-ion-language-outline
      template(#dropdown)
        el-dropdown-menu
          el-dropdown-item(
            v-for="locale in $i18n.availableLocales"
            :command="locale"
            :disabled="locale === $i18n.locale"
            :key="locale"
          ) {{ $t('language', locale, { locale }) }}
</template>

<style lang="sass">
#nav
  background: transparent
  backdrop-filter: blur(5px)
#subnav
  padding: 0 1em
  margin-left: auto
</style>
