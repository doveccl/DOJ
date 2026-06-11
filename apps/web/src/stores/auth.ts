import { apiFetch } from '../api'

export interface AuthUser {
  id: number
  name: string
  email: string
  introduction: string
  admin: boolean
  disabled: boolean
  mustChangePassword: boolean
  avatarUrl: string
}

export const useAuthStore = defineStore('auth', () => {
  const token = ref(localStorage.getItem('doj.token') ?? '')
  const user = ref<AuthUser | null>(null)
  const ready = ref(false)
  let restorePromise: Promise<void> | null = null

  const signedIn = computed(() => !!token.value && !!user.value)
  const isAdmin = computed(() => user.value?.admin === true)

  function setSession(nextToken: string, nextUser: AuthUser) {
    token.value = nextToken
    user.value = nextUser
    localStorage.setItem('doj.token', nextToken)
  }

  async function register(input: {
    name: string
    email: string
    password: string
    code: string
  }) {
    const result = await apiFetch<{ token: string; user: AuthUser }>('/api/auth/register', {
      method: 'POST',
      body: JSON.stringify(input)
    })
    setSession(result.token, result.user)
  }

  async function requestEmailCode(input: { purpose: 'register' | 'change-email'; email: string }) {
    await apiFetch('/api/auth/email-code', {
      method: 'POST',
      body: JSON.stringify(input)
    })
  }

  async function login(input: { user: string; password: string }) {
    const result = await apiFetch<{ token: string; user: AuthUser }>('/api/auth/login', {
      method: 'POST',
      body: JSON.stringify(input)
    })
    setSession(result.token, result.user)
  }

  async function restore() {
    if (restorePromise) return restorePromise
    restorePromise = doRestore().finally(() => {
      restorePromise = null
    })
    return restorePromise
  }

  async function doRestore() {
    if (!token.value) {
      ready.value = true
      return
    }

    try {
      user.value = await apiFetch<AuthUser>('/api/auth/self')
    } catch {
      logout()
    } finally {
      ready.value = true
    }
  }

  async function updateProfile(input: { introduction?: string; currentPassword?: string; password?: string }) {
    user.value = await apiFetch<AuthUser>('/api/auth/self', {
      method: 'PATCH',
      body: JSON.stringify(input)
    })
  }

  async function updateEmail(input: { email: string; code: string }) {
    user.value = await apiFetch<AuthUser>('/api/auth/email', {
      method: 'PATCH',
      body: JSON.stringify(input)
    })
  }

  async function logout() {
    if (token.value) {
      await apiFetch('/api/auth/logout', { method: 'POST' }).catch(() => {})
    }
    token.value = ''
    user.value = null
    localStorage.removeItem('doj.token')
  }

  return {
    ready,
    signedIn,
    isAdmin,
    token,
    user,
    login,
    logout,
    register,
    requestEmailCode,
    restore,
    updateEmail,
    updateProfile
  }
})
