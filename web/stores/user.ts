import { useLocalStorage } from '@vueuse/core'

export const useUserStore = defineStore('user', () => {
  const token = useLocalStorage('token', '')

  watchEffect(() => {
    token
  })

  return { token }
})
