<script setup lang="ts">
import type { MenuOption } from 'naive-ui'
import { useI18n } from 'vue-i18n'
import { useAuthStore } from '../../stores/auth'

const route = useRoute()
const router = useRouter()
const { t } = useI18n()
const auth = useAuthStore()
const canManage = computed(() => auth.user?.admin ?? false)
const wideLayout = ref(false)
let mediaQuery: MediaQueryList | null = null

function handleLayoutChange(event: MediaQueryListEvent) {
  wideLayout.value = event.matches
}

const adminOptions = computed<MenuOption[]>(() => [
  { label: t('nav.settings'), key: '/admin/settings' },
  { label: t('nav.members'), key: '/admin/members' },
  { label: t('nav.languages'), key: '/admin/languages' },
  { label: t('nav.agents'), key: '/admin/agents' }
])

onMounted(() => {
  mediaQuery = window.matchMedia('(min-width: 980px)')
  wideLayout.value = mediaQuery.matches
  mediaQuery.addEventListener('change', handleLayoutChange)
})

onUnmounted(() => {
  mediaQuery?.removeEventListener('change', handleLayoutChange)
})
</script>

<template>
  <section class="admin-layout" :class="{ 'admin-layout-plain': !canManage }">
    <aside v-if="canManage" class="admin-sidebar">
      <n-menu
        :mode="wideLayout ? 'vertical' : 'horizontal'"
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
