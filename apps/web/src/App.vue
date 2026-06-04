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
  NSpace
} from 'naive-ui'
import type { MenuOption } from 'naive-ui'
import { computed, onMounted, reactive, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useAuthStore } from './stores/auth'

const route = useRoute()
const router = useRouter()
const { t } = useI18n()
const auth = useAuthStore()

const authMode = ref<'login' | 'register' | null>(null)
const authError = ref('')
const authLoading = ref(false)
const loginForm = reactive({ user: '', password: '' })
const registerForm = reactive({ name: '', email: '', password: '' })

const menuOptions = computed(() => {
  const options: MenuOption[] = [
    { label: t('home'), key: '/' },
    { label: t('problems'), key: '/problems' },
    { label: t('assignments'), key: '/assignments' },
    { label: t('contests'), key: '/contests' },
    { label: t('submissions'), key: '/submissions' }
  ]
  if (auth.user?.groups.includes('admin')) {
    options.push({
      label: t('admin'),
      key: 'admin',
      children: [
        { label: t('groups'), key: '/admin/groups' },
        { label: t('assignments'), key: '/admin/assignments' },
        { label: t('contests'), key: '/admin/contests' },
        { label: t('languages'), key: '/admin/languages' },
        { label: t('runners'), key: '/admin/runners' }
      ]
    })
  }
  return options
})

const userMenuOptions = computed(() => [
  { label: auth.user?.email ?? '', key: 'email', disabled: true },
  { label: 'Sign out', key: 'logout' }
])

onMounted(() => {
  auth.restore()
})

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
        <n-space class="topbar-actions">
          <template v-if="auth.signedIn">
            <n-dropdown :options="userMenuOptions" @select="handleUserCommand">
              <n-button text>{{ auth.user?.name }}</n-button>
            </n-dropdown>
          </template>
          <template v-else>
            <n-button text @click="openAuth('login')">Sign in</n-button>
            <n-button type="primary" size="small" @click="openAuth('register')">Sign up</n-button>
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
      :title="authMode === 'register' ? 'Sign up' : 'Sign in'"
      @update:show="(show: boolean) => !show && (authMode = null)"
    >
      <n-form v-if="authMode === 'login'" :model="loginForm" label-placement="top">
        <n-form-item label="User">
          <n-input v-model:value="loginForm.user" autocomplete="username" />
        </n-form-item>
        <n-form-item label="Password">
          <n-input
            v-model:value="loginForm.password"
            type="password"
            autocomplete="current-password"
            @keyup.enter="submitAuth"
          />
        </n-form-item>
      </n-form>
      <n-form v-else :model="registerForm" label-placement="top">
        <n-form-item label="Name">
          <n-input v-model:value="registerForm.name" autocomplete="username" />
        </n-form-item>
        <n-form-item label="Email">
          <n-input v-model:value="registerForm.email" autocomplete="email" />
        </n-form-item>
        <n-form-item label="Password">
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
          <n-button @click="authMode = null">Cancel</n-button>
          <n-button type="primary" :loading="authLoading" @click="submitAuth">
            {{ authMode === 'register' ? 'Sign up' : 'Sign in' }}
          </n-button>
        </n-space>
      </template>
    </n-modal>
  </n-config-provider>
</template>
