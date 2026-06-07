import { defineStore } from 'pinia'
import { computed, ref } from 'vue'
import { apiFetch } from '../api'

export interface AuthUser {
  id: number
  name: string
  email: string
  introduction: string
  groups: string[]
}

export const useAuthStore = defineStore('auth', () => {
  const token = ref(localStorage.getItem('doj.token') ?? '')
  const user = ref<AuthUser | null>(null)
  const ready = ref(false)

  const signedIn = computed(() => !!token.value && !!user.value)

  function setSession(nextToken: string, nextUser: AuthUser) {
    token.value = nextToken
    user.value = nextUser
    localStorage.setItem('doj.token', nextToken)
  }

  async function register(input: {
    name: string
    email: string
    password: string
    inviteCode?: string
  }) {
    const result = await apiFetch<{ token: string; user: AuthUser }>('/api/auth/register', {
      method: 'POST',
      body: JSON.stringify(input)
    })
    setSession(result.token, result.user)
  }

  async function login(input: { user: string; password: string }) {
    const result = await apiFetch<{ token: string; user: AuthUser }>('/api/auth/login', {
      method: 'POST',
      body: JSON.stringify(input)
    })
    setSession(result.token, result.user)
  }

  async function restore() {
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

  async function updateProfile(input: { introduction?: string; password?: string }) {
    user.value = await apiFetch<AuthUser>('/api/auth/self', {
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

  return { ready, signedIn, token, user, login, logout, register, restore, updateProfile }
})
