<script setup lang="ts">
import { useI18nStore } from '@stores/i18n'
const { elocale } = storeToRefs(useI18nStore())
</script>

<template lang="pug">
el-config-provider(:locale="elocale")
  #container
    el-affix
      nav-menu
    el-main
      router-view
    el-backtop
</template>

<style lang="sass">
@import element-plus/theme-chalk/dark/css-vars.css
#app
  top: 0
  left: 0
  width: 100vw
  height: 100vh
  position: fixed
  overflow-y: auto
  overflow-x: hidden
#container
  width: 100vw
.monospace
  font-family: Cascadia Code, Menlo, Consolas, monospace
</style>
