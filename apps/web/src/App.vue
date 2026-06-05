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
  NMenu,
  NModal,
  NSelect,
  NSpace,
  NSwitch,
  darkTheme
} from 'naive-ui'
import type { MenuOption } from 'naive-ui'
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { RouterLink, useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { setLocale, supportedLocales, type SupportedLocale } from './i18n'
import { useAuthStore } from './stores/auth'

const route = useRoute()
const router = useRouter()
const { t, locale } = useI18n()
const auth = useAuthStore()

const authMode = ref<'login' | 'register' | null>(null)
const authError = ref('')
const authLoading = ref(false)
const colorMode = ref<'light' | 'dark'>(
  localStorage.getItem('doj.colorMode') === 'dark' ? 'dark' : 'light'
)
const loginForm = reactive({ user: '', password: '' })
const registerForm = reactive({ name: '', email: '', password: '' })
const localeValue = computed({
  get: () => locale.value as SupportedLocale,
  set: (value: SupportedLocale) => setLocale(value)
})
const naiveTheme = computed(() => (colorMode.value === 'dark' ? darkTheme : null))

const menuOptions = computed(() => {
  const options: MenuOption[] = [
    { label: t('nav.problems'), key: '/problems' },
    { label: t('nav.assignments'), key: '/assignments' },
    { label: t('nav.contests'), key: '/contests' },
    { label: t('nav.discussion'), key: '/bbs' },
    { label: t('nav.rank'), key: '/rank' },
    { label: t('nav.submissions'), key: '/submissions' }
  ]
  if (auth.user?.groups.includes('admin')) {
    options.push({
      label: t('nav.admin'),
      key: 'admin',
      children: [
        { label: t('nav.groups'), key: '/admin/groups' },
        { label: t('nav.users'), key: '/admin/users' },
        { label: t('nav.manageProblems'), key: '/admin/problems' },
        { label: t('nav.assignments'), key: '/admin/assignments' },
        { label: t('nav.contests'), key: '/admin/contests' },
        { label: t('nav.languages'), key: '/admin/languages' },
        { label: t('nav.runners'), key: '/admin/runners' }
      ]
    })
  }
  return options
})

const userMenuOptions = computed(() => [
  { label: auth.user?.email ?? '', key: 'email', disabled: true },
  { label: t('app.signOut'), key: 'logout' }
])

onMounted(() => {
  auth.restore()
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
      await auth.register(registerForm)
    }
    authMode.value = null
  } catch (error) {
    authError.value = error instanceof Error ? error.message : String(error)
  } finally {
    authLoading.value = false
  }
}

function handleUserCommand(key: string) {
  if (key === 'logout') auth.logout()
}
</script>

<template>
  <n-config-provider :theme="naiveTheme">
    <n-layout class="app-shell">
      <header class="topbar">
        <router-link to="/" class="brand">{{ t('app.brand') }}</router-link>
        <n-menu
          mode="horizontal"
          :value="route.path"
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
          <template v-else>
            <n-button text @click="openAuth('login')">{{ t('app.signIn') }}</n-button>
            <n-button type="primary" size="small" @click="openAuth('register')">
              {{ t('app.signUp') }}
            </n-button>
          </template>
        </n-space>
      </header>
      <n-layout-content class="content">
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
