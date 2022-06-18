<script setup lang="ts">
import { useI18nStore } from '@stores/i18n'
const { elocale } = storeToRefs(useI18nStore())
</script>

<template>
  <el-config-provider :locale="elocale">
    <el-container>
      <el-header>
        <nav-menu />
      </el-header>
      <el-main>
        <router-view />
      </el-main>
    </el-container>
    <el-backtop />
  </el-config-provider>
</template>

<style lang="stylus">
@import 'element-plus/theme-chalk/dark/css-vars.css'
body
  margin 0
  background-color var(--el-bg-color)
  transition background-color var(--el-transition-duration-fast)
</style>
