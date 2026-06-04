<script setup lang="ts">
import { NConfigProvider, NLayout, NLayoutContent, NMenu, NSpace } from 'naive-ui'
import { computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'

const route = useRoute()
const router = useRouter()
const { t } = useI18n()

const menuOptions = computed(() => [
  { label: t('home'), key: '/' },
  { label: t('problems'), key: '/problems' },
  { label: t('submissions'), key: '/submissions' }
])
</script>

<template>
  <n-config-provider>
    <n-layout class="app-shell">
      <header class="topbar">
        <div class="brand">{{ t('app') }}</div>
        <n-menu
          mode="horizontal"
          :value="route.path"
          :options="menuOptions"
          @update:value="(path: string) => router.push(path)"
        />
        <n-space class="topbar-actions" />
      </header>
      <n-layout-content class="content">
        <router-view />
      </n-layout-content>
    </n-layout>
  </n-config-provider>
</template>
