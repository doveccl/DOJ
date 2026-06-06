<script setup lang="ts">
import { NMenu } from 'naive-ui'
import type { MenuOption } from 'naive-ui'
import { computed } from 'vue'
import { RouterView, useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useAuthStore } from '../stores/auth'

const route = useRoute()
const router = useRouter()
const { t } = useI18n()
const auth = useAuthStore()
const canManage = computed(() => auth.user?.groups.includes('admin') ?? false)

const adminOptions = computed<MenuOption[]>(() => [
  { label: t('nav.settings'), key: '/admin/settings' },
  { label: t('nav.members'), key: '/admin/members' },
  { label: t('nav.manageProblems'), key: '/admin/problems' },
  { label: t('nav.assignments'), key: '/admin/assignments' },
  { label: t('nav.contests'), key: '/admin/contests' },
  { label: t('nav.languages'), key: '/admin/languages' },
  { label: t('nav.agents'), key: '/admin/agents' }
])
</script>

<template>
  <section class="admin-layout" :class="{ 'admin-layout-plain': !canManage }">
    <aside v-if="canManage" class="admin-sidebar">
      <n-menu
        :value="route.path"
        :options="adminOptions"
        :collapsed-width="64"
        @update:value="(path: string) => router.push(path)"
      />
    </aside>
    <div class="admin-workspace">
      <router-view />
    </div>
  </section>
</template>
