<script setup lang="ts">
import { routes } from '@/route'
const dark = useDark()
const switcher = { activeIcon: IEpMoon, inactiveIcon: IEpSunny }
</script>

<template lang="pug">
el-menu#nav(mode="horizontal" :default-active="$route.path" @select="$router.push")
  el-menu-item.monospace(index="/")
    b(style="font-size: 2em") DOJ
  template(v-for="item in routes")
    el-menu-item(v-if="item.name" :index="item.path" :key="item.path") {{ $t(item.name) }}
  el-space#subnav(size="large")
    user-menu
    el-switch(v-bind="switcher" v-model="dark" active-color="var(--el-border-color)" inline-prompt)
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
