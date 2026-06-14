export type AuthMode = 'login' | 'register'

export const useUiStore = defineStore('ui', () => {
  const authMode = ref<AuthMode | null>(null)

  function openLogin() {
    authMode.value = 'login'
  }

  function openRegister() {
    authMode.value = 'register'
  }

  function closeAuth() {
    authMode.value = null
  }

  return { authMode, openLogin, openRegister, closeAuth }
})
