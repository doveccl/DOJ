<script setup lang="ts">
import { darkTheme, dateEnUS, dateZhCN, enUS, zhCN } from 'naive-ui'
import type { GlobalThemeOverrides } from 'naive-ui'
import {
  CodeSlashOutline,
  DesktopOutline,
  LanguageOutline,
  LogInOutline,
  MoonOutline,
  PersonAddOutline,
  SunnyOutline
} from '@vicons/ionicons5'
import { useI18n } from 'vue-i18n'
import { apiFetch, errorMessage, type PublicConfig } from './api'
import { setLocale, supportedLocales } from './i18n'
import { useAuthStore } from './stores/auth'
import { useUiStore } from './stores/ui'

type ColorMode = 'system' | 'light' | 'dark'

const route = useRoute()
const router = useRouter()
const { t, locale } = useI18n()
const auth = useAuthStore()
const ui = useUiStore()

const authMode = computed(() => ui.authMode)
const authError = ref('')
const authLoading = ref(false)
const codeLoading = ref(false)
const codeSent = ref(false)
const appConfig = ref<PublicConfig>({
  signup: false,
  guestAccess: true,
  publicCode: false,
  aiEnabled: false,
  smtpConfigured: false,
  notice: ''
})
const savedColorMode = localStorage.getItem('doj.colorMode') as ColorMode | null
const colorMode = ref<ColorMode>(
  savedColorMode === 'light' || savedColorMode === 'dark' || savedColorMode === 'system'
    ? savedColorMode
    : 'system'
)
const systemDark = ref(false)
const loginForm = reactive({ user: '', password: '' })
const registerForm = reactive({ name: '', email: '', password: '', code: '' })
const resolvedColorMode = computed(() =>
  colorMode.value === 'system' ? (systemDark.value ? 'dark' : 'light') : colorMode.value
)
const naiveTheme = computed(() => (resolvedColorMode.value === 'dark' ? darkTheme : null))
const themeOverrides = computed<GlobalThemeOverrides>(() => {
  const dark = resolvedColorMode.value === 'dark'
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
const themeIcon = computed(() => {
  if (colorMode.value === 'system') return DesktopOutline
  return colorMode.value === 'dark' ? MoonOutline : SunnyOutline
})
const themeOptions = computed(() => [
  { label: t('app.system'), key: 'system' },
  { label: t('app.light'), key: 'light' },
  { label: t('app.dark'), key: 'dark' }
])
const userInitial = computed(() => auth.user?.name?.slice(0, 1).toUpperCase() ?? '')
const localeOptions = computed(() =>
  supportedLocales.map((item) => ({
    label: item.label,
    key: item.value
  }))
)
const navItems = computed(() => {
  const options = [{ label: t('nav.problems'), key: '/problems' }]
  if (auth.signedIn) {
    options.push({ label: t('nav.assignments'), key: '/assignments' })
  }
  options.push(
    { label: t('nav.contests'), key: '/contests' },
    { label: t('nav.discussion'), key: '/discussion' },
    { label: t('nav.rank'), key: '/rank' },
    { label: t('nav.submissions'), key: '/submissions' }
  )
  if (auth.user?.admin) {
    options.push({
      label: t('nav.admin'),
      key: '/admin/settings'
    })
  }
  return options
})

const userMenuOptions = computed(() => [
  { label: auth.user?.email ?? '', key: 'email', disabled: true },
  { label: t('app.profile'), key: 'profile' },
  { label: t('app.signOut'), key: 'logout' }
])

let mediaQuery: MediaQueryList | null = null
function handleSystemThemeChange(event: MediaQueryListEvent) {
  systemDark.value = event.matches
}

onMounted(async () => {
  mediaQuery = window.matchMedia('(prefers-color-scheme: dark)')
  systemDark.value = mediaQuery.matches
  mediaQuery.addEventListener('change', handleSystemThemeChange)
  auth.restore()
  try {
    const config = await apiFetch<PublicConfig>('/api/config')
    appConfig.value = { ...appConfig.value, ...config }
  } catch {
    appConfig.value = {
      signup: false,
      guestAccess: true,
      publicCode: false,
      aiEnabled: false,
      smtpConfigured: false,
      notice: ''
    }
  }
})

watch(
  colorMode,
  (value) => {
    localStorage.setItem('doj.colorMode', value)
  },
  { immediate: true }
)

watch(
  resolvedColorMode,
  (value) => {
    document.documentElement.dataset.theme = value
  },
  { immediate: true }
)

onUnmounted(() => {
  mediaQuery?.removeEventListener('change', handleSystemThemeChange)
})

function openAuth(mode: 'login' | 'register') {
  if (mode === 'register' && !appConfig.value.signup) return
  authError.value = ''
  codeSent.value = false
  if (mode === 'register') ui.openRegister()
  else ui.openLogin()
}

async function sendRegisterCode() {
  authError.value = ''
  codeLoading.value = true
  try {
    await auth.requestEmailCode({ purpose: 'register', email: registerForm.email })
    codeSent.value = true
  } catch (error) {
    authError.value = errorMessage(error)
  } finally {
    codeLoading.value = false
  }
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
        code: registerForm.code
      })
    }
    ui.closeAuth()
  } catch (error) {
    authError.value = errorMessage(error)
  } finally {
    authLoading.value = false
  }
}

function handleUserCommand(key: string) {
  if (key === 'logout') void auth.logout()
  else if (key === 'profile') void router.push('/profile')
}

function handleThemeCommand(key: string) {
  if (key === 'system' || key === 'light' || key === 'dark') colorMode.value = key
}

function handleLocaleCommand(key: string) {
  if (key === 'zh-CN' || key === 'en') setLocale(key)
}

function navActive(path: string) {
  if (path === '/') return route.path === '/'
  if (path === '/admin/settings') return route.path.startsWith('/admin')
  return route.path === path || route.path.startsWith(`${path}/`)
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
        <router-link to="/" class="brand">
          <span class="brand-mark" aria-hidden="true">
            <n-icon :component="CodeSlashOutline" />
          </span>
          <span class="brand-word">{{ t('app.brand') }}</span>
        </router-link>
        <nav class="topbar-nav" :aria-label="t('app.brand')">
          <router-link
            v-for="item in navItems"
            :key="item.key"
            :to="item.key"
            class="topbar-link"
            :class="{ active: navActive(item.key) }"
          >
            {{ item.label }}
          </router-link>
        </nav>
        <n-space class="topbar-actions">
          <n-dropdown :options="localeOptions" @select="handleLocaleCommand">
            <n-button quaternary circle :aria-label="t('app.locale')">
              <template #icon>
                <n-icon :component="LanguageOutline" />
              </template>
            </n-button>
          </n-dropdown>
          <n-dropdown :options="themeOptions" @select="handleThemeCommand">
            <n-button quaternary circle :aria-label="t('app.colorMode')">
              <template #icon>
                <n-icon :component="themeIcon" />
              </template>
            </n-button>
          </n-dropdown>
          <template v-if="auth.signedIn">
            <n-dropdown :options="userMenuOptions" @select="handleUserCommand">
              <n-button quaternary class="user-trigger">
                <template #icon>
                  <n-avatar v-if="auth.user?.avatarUrl" :size="24" round :src="auth.user.avatarUrl">
                    <template #fallback>{{ userInitial }}</template>
                  </n-avatar>
                  <n-avatar v-else :size="24" round>{{ userInitial }}</n-avatar>
                </template>
                <span class="user-name">{{ auth.user?.name }}</span>
              </n-button>
            </n-dropdown>
          </template>
          <n-space v-else align="center" :size="8" class="auth-actions">
            <n-button secondary size="small" @click="openAuth('login')">
              <template #icon>
                <n-icon :component="LogInOutline" />
              </template>
              {{ t('app.signIn') }}
            </n-button>
            <n-button
              v-if="appConfig.signup"
              type="primary"
              size="small"
              @click="openAuth('register')"
            >
              <template #icon>
                <n-icon :component="PersonAddOutline" />
              </template>
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
      v-if="authMode"
      :show="!!authMode"
      preset="dialog"
      :title="authMode === 'register' ? t('app.signUp') : t('app.signIn')"
      @update:show="(show: boolean) => !show && ui.closeAuth()"
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
      <n-form v-else-if="authMode === 'register'" :model="registerForm" label-placement="top">
        <n-form-item :label="t('app.userName')">
          <n-input v-model:value="registerForm.name" autocomplete="username" />
        </n-form-item>
        <n-form-item :label="t('app.email')">
          <n-input v-model:value="registerForm.email" autocomplete="email" />
        </n-form-item>
        <n-form-item :label="t('app.emailCode')">
          <n-input
            v-model:value="registerForm.code"
            autocomplete="one-time-code"
            @keyup.enter="submitAuth"
          >
            <template #suffix>
              <n-button
                text
                type="primary"
                :loading="codeLoading"
                :disabled="!appConfig.smtpConfigured || !registerForm.email"
                @click="sendRegisterCode"
              >
                {{ t('app.sendCode') }}
              </n-button>
            </template>
          </n-input>
        </n-form-item>
        <n-alert v-if="codeSent" type="success" :show-icon="false" class="card-alert">
          {{ t('app.codeSent') }}
        </n-alert>
        <n-alert
          v-else-if="!appConfig.smtpConfigured"
          type="warning"
          :show-icon="false"
          class="card-alert"
        >
          {{ t('app.smtpRequired') }}
        </n-alert>
        <n-form-item :label="t('app.password')">
          <n-input
            v-model:value="registerForm.password"
            type="password"
            autocomplete="new-password"
            @keyup.enter="submitAuth"
          />
        </n-form-item>
      </n-form>
      <p v-if="authError" class="form-error">{{ authError }}</p>
      <template #action>
        <n-space justify="end">
          <n-button @click="ui.closeAuth()">{{ t('app.cancel') }}</n-button>
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
  gap: 18px;
  min-height: 60px;
  padding: 0 24px;
  background: var(--topbar-bg);
  backdrop-filter: saturate(180%) blur(12px);
  -webkit-backdrop-filter: saturate(180%) blur(12px);
}

.user-trigger {
  padding: 0 8px;

  :deep(.n-button__icon) {
    width: 24px;
    min-width: 24px;
    height: 24px;
    flex-basis: 24px;
  }

  :deep(.n-icon-slot),
  :deep(.n-avatar),
  :deep(.n-avatar img) {
    width: 24px;
    min-width: 24px;
    height: 24px;
  }
}

.user-name {
  display: inline-block;
  max-width: 112px;
  overflow: hidden;
  text-overflow: ellipsis;
  vertical-align: middle;
  white-space: nowrap;
}

.topbar-nav {
  display: flex;
  align-items: center;
  gap: 2px;
  min-width: 0;
  overflow-x: auto;
  scrollbar-width: none;

  &::-webkit-scrollbar {
    display: none;
  }
}

.topbar-link {
  flex: 0 0 auto;
  padding: 7px 11px;
  border-radius: var(--radius-md);
  color: var(--muted-color);
  font-size: 14px;
  font-weight: 500;
  line-height: 1;
  text-decoration: none;
  white-space: nowrap;

  &:hover {
    color: var(--text-color);
    background: color-mix(in srgb, var(--text-color) 7%, transparent);
  }

  &.active {
    color: var(--brand-strong);
    background: var(--brand-soft);
  }
}

@media (max-width: 640px) {
  .topbar {
    grid-template-columns: minmax(0, 1fr);
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

    .topbar-nav {
      grid-column: 1 / -1;
      order: 3;
      width: 100%;
      padding-bottom: 2px;
    }
  }
}
</style>
