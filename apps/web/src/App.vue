<script setup lang="ts">
import {
  NButton,
  NConfigProvider,
  NDropdown,
  NForm,
  NFormItem,
  NInput,
  NLayout,
  NLayoutContent,
  NLayoutHeader,
  NMenu,
  NModal,
  NSelect,
  NSpace,
  NSwitch,
  darkTheme,
  dateEnUS,
  dateZhCN,
  enUS,
  zhCN
} from 'naive-ui'
import type { GlobalThemeOverrides, MenuOption } from 'naive-ui'
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { RouterLink, useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { apiFetch } from './api'
import { setLocale, supportedLocales, type SupportedLocale } from './i18n'
import { useAuthStore } from './stores/auth'

const route = useRoute()
const router = useRouter()
const { t, locale } = useI18n()
const auth = useAuthStore()

const authMode = ref<'login' | 'register' | null>(null)
const authError = ref('')
const authLoading = ref(false)
const appConfig = ref({
  registration: true,
  registrationInviteRequired: false,
  aiCoachingEnabled: true,
  guestProblemsetVisible: true,
  sourceOpenDefault: false
})
const colorMode = ref<'light' | 'dark'>(
  localStorage.getItem('doj.colorMode') === 'dark' ? 'dark' : 'light'
)
const loginForm = reactive({ user: '', password: '' })
const registerForm = reactive({ name: '', email: '', password: '', inviteCode: '' })
const localeValue = computed({
  get: () => locale.value as SupportedLocale,
  set: (value: SupportedLocale) => setLocale(value)
})
const naiveTheme = computed(() => (colorMode.value === 'dark' ? darkTheme : null))
const themeOverrides = computed<GlobalThemeOverrides>(() => {
  const dark = colorMode.value === 'dark'
  const primary = dark ? '#2dd4bf' : '#0d9488'
  const primaryHover = dark ? '#5eead4' : '#0f766e'
  const primaryPressed = dark ? '#14b8a6' : '#115e59'
  return {
    common: {
      primaryColor: primary,
      primaryColorHover: primaryHover,
      primaryColorPressed: primaryPressed,
      primaryColorSuppl: primaryHover
    }
  }
})
const naiveLocale = computed(() => (locale.value === 'en' ? enUS : zhCN))
const naiveDateLocale = computed(() => (locale.value === 'en' ? dateEnUS : dateZhCN))
const topMenuValue = computed(() =>
  route.path.startsWith('/admin') ? '/admin/settings' : route.path
)

const menuOptions = computed(() => {
  const options: MenuOption[] = []
  if (appConfig.value.guestProblemsetVisible || auth.signedIn) {
    options.push({ label: t('nav.problems'), key: '/problems' })
  }
  if (auth.signedIn) {
    options.push({ label: t('nav.assignments'), key: '/assignments' })
  }
  options.push(
    { label: t('nav.contests'), key: '/contests' },
    { label: t('nav.discussion'), key: '/discussion' },
    { label: t('nav.rank'), key: '/rank' },
    { label: t('nav.submissions'), key: '/submissions' }
  )
  if (auth.user?.groups.includes('admin')) {
    options.push({
      label: t('nav.admin'),
      key: '/admin/settings'
    })
  }
  return options
})

const userMenuOptions = computed(() => [
  { label: auth.user?.email ?? '', key: 'email', disabled: true },
  { label: t('app.signOut'), key: 'logout' }
])

onMounted(async () => {
  auth.restore()
  try {
    appConfig.value = await apiFetch<typeof appConfig.value>('/api/config')
  } catch {
    appConfig.value = {
      registration: true,
      registrationInviteRequired: false,
      aiCoachingEnabled: true,
      guestProblemsetVisible: true,
      sourceOpenDefault: false
    }
  }
})

watch(
  colorMode,
  (value) => {
    localStorage.setItem('doj.colorMode', value)
    document.documentElement.dataset.theme = value
  },
  { immediate: true }
)

function openAuth(mode: 'login' | 'register') {
  if (mode === 'register' && !appConfig.value.registration) return
  authError.value = ''
  authMode.value = mode
}

async function submitAuth() {
  authError.value = ''
  authLoading.value = true
  try {
    if (authMode.value === 'login') {
      await auth.login(loginForm)
    } else if (authMode.value === 'register') {
      await auth.register({
        name: registerForm.name,
        email: registerForm.email,
        password: registerForm.password,
        inviteCode: registerForm.inviteCode || undefined
      })
    }
    authMode.value = null
  } catch (error) {
    authError.value = error instanceof Error ? error.message : String(error)
  } finally {
    authLoading.value = false
  }
}

function handleUserCommand(key: string) {
  if (key === 'logout') void auth.logout()
  else if (key === 'profile') void router.push('/profile')
}
</script>

<template>
  <n-config-provider
    :theme="naiveTheme"
    :theme-overrides="themeOverrides"
    :locale="naiveLocale"
    :date-locale="naiveDateLocale"
  >
    <n-layout class="app-shell" position="absolute">
      <n-layout-header class="topbar" bordered>
        <router-link to="/" class="brand">{{ t('app.brand') }}</router-link>
        <n-menu
          class="topbar-menu"
          mode="horizontal"
          :value="topMenuValue"
          :options="menuOptions"
          @update:value="(path: string) => router.push(path)"
        />
        <n-space class="topbar-actions">
          <n-select
            v-model:value="localeValue"
            :options="[...supportedLocales]"
            size="small"
            class="locale-select"
            :aria-label="t('app.locale')"
          />
          <n-switch
            v-model:value="colorMode"
            checked-value="dark"
            unchecked-value="light"
            :aria-label="t('app.colorMode')"
          >
            <template #checked>{{ t('app.dark') }}</template>
            <template #unchecked>{{ t('app.light') }}</template>
          </n-switch>
          <template v-if="auth.signedIn">
            <n-dropdown :options="userMenuOptions" @select="handleUserCommand">
              <n-button text>{{ auth.user?.name }}</n-button>
            </n-dropdown>
          </template>
          <n-space v-else align="center" :size="8" class="auth-actions">
            <n-button secondary size="small" @click="openAuth('login')">
              {{ t('app.signIn') }}
            </n-button>
            <n-button
              v-if="appConfig.registration"
              type="primary"
              size="small"
              @click="openAuth('register')"
            >
              {{ t('app.signUp') }}
            </n-button>
          </n-space>
        </n-space>
      </n-layout-header>
      <n-layout-content class="content" :native-scrollbar="false">
        <router-view />
      </n-layout-content>
    </n-layout>

    <n-modal
      :show="!!authMode"
      preset="dialog"
      :title="authMode === 'register' ? t('app.signUp') : t('app.signIn')"
      @update:show="(show: boolean) => !show && (authMode = null)"
    >
      <n-form v-if="authMode === 'login'" :model="loginForm" label-placement="top">
        <n-form-item :label="t('app.user')">
          <n-input v-model:value="loginForm.user" autocomplete="username" />
        </n-form-item>
        <n-form-item :label="t('app.password')">
          <n-input
            v-model:value="loginForm.password"
            type="password"
            autocomplete="current-password"
            @keyup.enter="submitAuth"
          />
        </n-form-item>
      </n-form>
      <n-form v-else :model="registerForm" label-placement="top">
        <n-form-item :label="t('app.userName')">
          <n-input v-model:value="registerForm.name" autocomplete="username" />
        </n-form-item>
        <n-form-item :label="t('app.email')">
          <n-input v-model:value="registerForm.email" autocomplete="email" />
        </n-form-item>
        <n-form-item :label="t('app.password')">
          <n-input
            v-model:value="registerForm.password"
            type="password"
            autocomplete="new-password"
            @keyup.enter="submitAuth"
          />
        </n-form-item>
        <n-form-item v-if="appConfig.registrationInviteRequired" :label="t('app.inviteCode')">
          <n-input
            v-model:value="registerForm.inviteCode"
            type="password"
            show-password-on="click"
            autocomplete="one-time-code"
            @keyup.enter="submitAuth"
          />
        </n-form-item>
      </n-form>
      <p v-if="authError" class="form-error">{{ authError }}</p>
      <template #action>
        <n-space justify="end">
          <n-button @click="authMode = null">{{ t('app.cancel') }}</n-button>
          <n-button type="primary" :loading="authLoading" @click="submitAuth">
            {{ authMode === 'register' ? t('app.signUp') : t('app.signIn') }}
          </n-button>
        </n-space>
      </template>
    </n-modal>
  </n-config-provider>
</template>

<style scoped lang="scss">
.app-shell {
  background: var(--page-bg);
}

.topbar {
  position: sticky;
  top: 0;
  z-index: 20;
  display: grid;
  grid-template-columns: auto 1fr auto;
  align-items: center;
  gap: 24px;
  height: 56px;
  padding: 0 24px;
  background: var(--topbar-bg);
  backdrop-filter: saturate(180%) blur(12px);
  -webkit-backdrop-filter: saturate(180%) blur(12px);

  :deep(.n-menu) {
    min-width: 0;
    background: transparent;
  }
}

@media (max-width: 640px) {
  .topbar {
    grid-template-columns: 1fr;
    grid-template-rows: auto auto auto;
    height: auto;
    gap: 8px 16px;
    padding: 12px 16px;

    .topbar-actions {
      grid-column: 1 / -1;
      justify-content: flex-start;
      flex-wrap: wrap;
      order: 2;
    }

    .topbar-menu {
      grid-column: 1 / -1;
      order: 3;
    }
  }
}
</style>
