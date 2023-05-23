<script setup lang="ts">
import { useI18nStore } from '@stores/i18n'
const { elocale } = storeToRefs(useI18nStore())
</script>

<template lang="pug">
el-config-provider(:locale="elocale")
  el-backtop
  el-container
    el-header
      nav-menu
    el-main
      router-view
</template>

<style lang="sass">
@import element-plus/theme-chalk/dark/css-vars.css
body
  margin: 0
</style>
